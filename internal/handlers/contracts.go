package handlers

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/withObsrvr/prism/internal/gateway"
	"github.com/withObsrvr/prism/internal/humanize"
	"github.com/withObsrvr/prism/internal/templates/pages"
	pagesv2 "github.com/withObsrvr/prism/internal/templates/v2/pages"
)

func (h *Handlers) ContractList(w http.ResponseWriter, r *http.Request) {
	data := mockContractListData()
	pages.ContractList(data).Render(r.Context(), w)
}

func (h *Handlers) ContractDetail(w http.ResponseWriter, r *http.Request) {
	data, ok := h.contractDetailDataForRequest(w, r)
	if !ok {
		return
	}
	pages.ContractDetail(data).Render(r.Context(), w)
}

func (h *Handlers) ContractDetailV2(w http.ResponseWriter, r *http.Request) {
	data, ok := h.contractDetailDataForRequest(w, r)
	if !ok {
		return
	}
	pagesv2.ContractDetail(data, networkFromRequest(r)).Render(r.Context(), w)
}

func (h *Handlers) contractDetailDataForRequest(w http.ResponseWriter, r *http.Request) (pages.ContractDetailData, bool) {
	id := r.PathValue("id")
	network := networkFromRequest(r)

	// Redirect smart-wallet contracts to the dedicated wallet view.
	if h.useLiveData(r) {
		if walletInfo, err := h.Gateway.GetSmartWalletInfo(r.Context(), network, id); err == nil && walletInfo.IsSmartWallet {
			h.Logger.Info("smart wallet detected", "contract", id, "wallet_type", walletInfo.WalletType)
			http.Redirect(w, r, "/account/"+id+"/smart", http.StatusSeeOther)
			return pages.ContractDetailData{}, false
		}
	}

	var data pages.ContractDetailData
	if h.useLiveData(r) {
		if live, err := h.buildContractDetailData(r, network, id); err == nil {
			data = live
		} else {
			h.Logger.Warn("live contract shell data failed", "error", err, "contract", id)
			data = unavailableContractDetailData(id, network)
		}
	}
	if data.Address == "" {
		data = mockContractDetailData()
	}

	return data, true
}

func (h *Handlers) buildContractDetailData(r *http.Request, network, contractID string) (pages.ContractDetailData, error) {
	ctx := r.Context()

	metadata, metaErr := h.Gateway.GetContractMetadata(ctx, network, contractID)
	analytics, analyticsErr := h.Gateway.GetContractAnalytics(ctx, network, contractID)
	if metaErr != nil && analyticsErr != nil {
		return pages.ContractDetailData{}, fmt.Errorf("fetching contract detail: metadata=%v analytics=%v", metaErr, analyticsErr)
	}

	recentCalls, recentCallsErr := h.Gateway.GetContractRecentCalls(ctx, network, contractID, 10)
	if recentCallsErr != nil {
		h.Logger.Warn("contract recent calls failed", "error", recentCallsErr, "contract", contractID, "network", network)
	}
	storageResp, _ := h.Gateway.GetContractStorage(ctx, network, contractID, 3)

	data := pages.ContractDetailData{
		Name:         gateway.ShortAddress(contractID),
		Address:      contractID,
		DeployedAt:   "—",
		DeployLedger: "—",
		LastInvoked:  "—",
		StateSize:    "—",
		WASMHash:     "—",
		WASMSize:     "—",
		MonthlyRent:  "—",
		SuccessRate:  "—",
	}

	if metadata != nil {
		if metadata.DisplayName != "" {
			data.Name = metadata.DisplayName
		}
		if metadata.ContractID != "" {
			data.Address = metadata.ContractID
		}
		if metadata.ContractType != "" && metadata.ContractType != "contract" {
			data.Tags = []string{titleCase(metadata.ContractType)}
		}
		if metadata.CreatorAddress != "" {
			data.Creator = gateway.ShortAddress(metadata.CreatorAddress)
			data.CreatorHref = "/account/" + metadata.CreatorAddress
		}
		if metadata.CreatedAt != "" {
			if t, err := time.Parse(time.RFC3339, metadata.CreatedAt); err == nil {
				data.DeployedAt = t.Format("Jan 2, 2006")
			}
		}
		if metadata.CreatedLedger > 0 {
			data.DeployLedger = gateway.FormatNumber(metadata.CreatedLedger)
		}
		if metadata.WASMHash != "" {
			data.WASMHash = metadata.WASMHash
		}
		data.StorageEntries = gateway.FormatNumber(metadata.TotalEntries)
		data.StateSize = formatBytes(metadata.TotalStateSizeBytes)
		data.FunctionsCount = fmt.Sprintf("%d", len(metadata.ExportedFunctions))
		for _, fn := range metadata.ExportedFunctions {
			if fn.Name == "" {
				continue
			}
			data.ExportedFunctions = append(data.ExportedFunctions, fn.Name)
		}
	}

	if analytics != nil {
		var funcs []pages.ContractFunction
		for _, f := range analytics.TopFunctions {
			desc, category := describeContractFunction(f.Name)
			funcs = append(funcs, pages.ContractFunction{
				Name:        f.Name,
				Description: desc,
				Category:    category,
				Calls24h:    gateway.FormatNumber(f.Count),
				Calls7d:     "—",
				Calls30d:    "—",
			})
		}
		if len(funcs) > 0 {
			data.Functions = funcs
		}

		var activityPoints []float64
		var total7d int64
		var peak int64
		var topFunctionTotal int64
		for _, f := range analytics.TopFunctions {
			topFunctionTotal += f.Count
		}
		for _, d := range analytics.DailyCalls7D {
			activityPoints = append(activityPoints, float64(d.Count))
			total7d += d.Count
			if d.Count > peak {
				peak = d.Count
			}
		}
		data.ActivityPoints = activityPoints
		if len(activityPoints) > 0 {
			data.AvgInvocations = gateway.FormatNumber(total7d / int64(len(activityPoints)))
			data.PeakInvocations = gateway.FormatNumber(peak)
		}

		totalCalls := analytics.Stats.TotalCallsAsCaller + analytics.Stats.TotalCallsAsCallee
		if totalCalls == 0 && topFunctionTotal > 0 {
			totalCalls = topFunctionTotal
		}
		data.TotalInvocations = gateway.FormatNumber(totalCalls)
		data.LastInvoked = formatContractAge(analytics.Timeline.LastActivity)
		if data.LastInvoked == "—" && topFunctionTotal > 0 {
			data.LastInvoked = "Recent activity observed"
		}
		if data.DeployedAt == "—" {
			if analytics.Timeline.FirstSeen != "" {
				if t, err := time.Parse(time.RFC3339, analytics.Timeline.FirstSeen); err == nil {
					data.DeployedAt = t.Format("Jan 2, 2006")
				}
			}
		}
		if data.FunctionsCount == "" || data.FunctionsCount == "0" {
			data.FunctionsCount = fmt.Sprintf("%d", len(analytics.TopFunctions))
		}
	}

	if len(data.Functions) == 0 && metadata != nil {
		for _, fn := range metadata.ExportedFunctions {
			desc, category := describeContractFunction(fn.Name)
			data.Functions = append(data.Functions, pages.ContractFunction{
				Name:        fn.Name,
				Description: desc,
				Category:    category,
				Calls24h:    gateway.FormatNumber(fn.CallCount),
				Calls7d:     "—",
				Calls30d:    "—",
			})
		}
	}

	for _, call := range recentCalls {
		statusColor := "emerald"
		status := "Success"
		if !call.Successful {
			statusColor = "red"
			status = "Failed"
		}
		data.Invocations = append(data.Invocations, pages.ContractInvocation{
			TxHash:      call.TransactionHash,
			ShortHash:   gateway.ShortHash(call.TransactionHash),
			Function:    call.FunctionName,
			Caller:      gateway.ShortAddress(call.SourceAccount),
			Status:      status,
			StatusColor: statusColor,
			Summary:     summarizeContractInvocation(call.FunctionName, call.SourceAccount, data.Name),
			Age:         formatContractAge(call.ClosedAt),
		})
	}

	if storageResp != nil {
		for _, entry := range storageResp.Entries {
			data.StorageItems = append(data.StorageItems, pages.ContractStorageItem{
				Key:       truncateMiddle(entry.Key, 20),
				Type:      titleCase(entry.Type),
				TypeColor: storageTypeColor(entry.Type),
				Size:      formatBytes(entry.SizeBytes),
				TTL:       "—",
			})
		}
	}

	if data.StorageEntries == "" {
		data.StorageEntries = fmt.Sprintf("%d", len(data.StorageItems))
	}
	if data.TotalInvocations == "" {
		data.TotalInvocations = "—"
	}
	if data.FunctionsCount == "" {
		data.FunctionsCount = fmt.Sprintf("%d", len(data.Functions))
	}

	human := humanize.BuildContractSummary(metadata, analytics)
	data.Narrative = human.Narrative
	data.Context = human.Context
	data.FunctionSummary = human.FunctionSummary
	data.StorageSummary = human.StorageSummary
	for _, sig := range human.Signals {
		data.Signals = append(data.Signals, pages.ContractHumanSignal{
			Title:    sig.Title,
			Severity: sig.Severity,
			Summary:  sig.Summary,
		})
	}
	for _, ev := range human.Evidence {
		data.Evidence = append(data.Evidence, pages.ContractHumanEvidence{
			Label: ev.Label,
			Value: ev.Value,
		})
	}

	return data, nil
}

func formatContractAge(ts string) string {
	if ts == "" {
		return "—"
	}
	t, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		return "—"
	}
	d := time.Since(t)
	switch {
	case d >= 24*time.Hour:
		return fmt.Sprintf("%.0fd ago", d.Hours()/24)
	case d >= time.Hour:
		return fmt.Sprintf("%.0fh ago", d.Hours())
	case d >= time.Minute:
		return fmt.Sprintf("%.0fm ago", d.Minutes())
	default:
		return fmt.Sprintf("%.0fs ago", d.Seconds())
	}
}

func formatBytes(n int64) string {
	switch {
	case n >= 1024*1024*1024:
		return fmt.Sprintf("%.1f GB", float64(n)/(1024*1024*1024))
	case n >= 1024*1024:
		return fmt.Sprintf("%.1f MB", float64(n)/(1024*1024))
	case n >= 1024:
		return fmt.Sprintf("%.1f KB", float64(n)/1024)
	case n > 0:
		return fmt.Sprintf("%d B", n)
	default:
		return "0 B"
	}
}

func truncateMiddle(s string, max int) string {
	if len(s) <= max || max < 8 {
		return s
	}
	prefix := (max - 3) / 2
	suffix := max - 3 - prefix
	return s[:prefix] + "..." + s[len(s)-suffix:]
}

func titleCase(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + strings.ToLower(s[1:])
}

func unavailableContractDetailData(contractID string, network string) pages.ContractDetailData {
	short := gateway.ShortAddress(contractID)
	return pages.ContractDetailData{
		Name:             short,
		Address:          contractID,
		DeployedAt:       "—",
		DeployLedger:     "—",
		LastInvoked:      "—",
		StateSize:        "—",
		WASMHash:         "—",
		WASMSize:         "—",
		MonthlyRent:      "—",
		StorageEntries:   "—",
		TotalInvocations: "—",
		FunctionsCount:   "—",
		SuccessRate:      "—",
		Narrative:        short + " could not be loaded from live " + network + " data right now.",
		Context:          "Prism did not fall back to mock data for this page so the live-data issue is visible during QA.",
		Signals: []pages.ContractHumanSignal{{
			Title:    "Live data unavailable",
			Severity: "warn",
			Summary:  "Contract metadata or analytics could not be loaded for this contract on the selected network.",
		}},
		Evidence: []pages.ContractHumanEvidence{{Label: "Requested contract", Value: contractID}},
	}
}

func describeContractFunction(functionName string) (description, category string) {
	if functionName == "" {
		return "", ""
	}
	if rule, ok := humanize.LookupFunctionNarration(functionName); ok {
		description = rule.Description
		if description == "" && rule.Phrase != "" {
			description = strings.ToUpper(rule.Phrase[:1]) + rule.Phrase[1:] + "."
		}
		return description, strings.ReplaceAll(rule.Category, "_", " ")
	}
	return strings.ToUpper(humanize.HumanizeFunctionName(functionName)[:1]) + humanize.HumanizeFunctionName(functionName)[1:] + ".", ""
}

func summarizeContractInvocation(functionName, sourceAccount, contractName string) string {
	caller := gateway.ShortAddress(sourceAccount)
	if functionName == "" {
		if contractName != "" && contractName != gateway.ShortAddress(contractName) {
			return fmt.Sprintf("%s interacted with %s", caller, contractName)
		}
		return fmt.Sprintf("%s called this contract", caller)
	}
	if rule, ok := humanize.LookupFunctionNarration(functionName); ok && rule.Phrase != "" {
		if contractName != "" {
			return fmt.Sprintf("%s %s on %s", caller, rule.Phrase, contractName)
		}
		return fmt.Sprintf("%s %s", caller, rule.Phrase)
	}
	return fmt.Sprintf("%s called %s()", caller, humanize.HumanizeFunctionName(functionName))
}

func storageTypeColor(t string) string {
	switch strings.ToLower(t) {
	case "instance":
		return "blue"
	case "temporary":
		return "amber"
	default:
		return "cyan"
	}
}

func (h *Handlers) ContractEvents(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if isHTMX(r) {
		fmt.Fprintf(w, `<div id="tab-content">Events for %s</div>`, id)
		return
	}
	fmt.Fprintf(w, "Contract Events: %s", id)
}
