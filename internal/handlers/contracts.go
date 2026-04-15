package handlers

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/withObsrvr/prism/internal/gateway"
	"github.com/withObsrvr/prism/internal/templates/pages"
)

func (h *Handlers) ContractList(w http.ResponseWriter, r *http.Request) {
	data := mockContractListData()
	pages.ContractList(data).Render(r.Context(), w)
}

func (h *Handlers) ContractDetail(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	network := networkFromRequest(r)

	// Detect smart wallets for future use. The redirect is disabled until
	// the smart account page is wired to accept and render the requested ID.
	// TODO: Enable redirect once SmartAccountDashboard uses the path {id}.
	if h.useLiveData(r) {
		if walletInfo, err := h.Gateway.GetSmartWalletInfo(r.Context(), network, id); err == nil && walletInfo.IsSmartWallet {
			h.Logger.Info("smart wallet detected", "contract", id, "wallet_type", walletInfo.WalletType)
			// http.Redirect(w, r, "/account/"+id+"/smart", http.StatusSeeOther)
			// return
		}
	}

	var data pages.ContractDetailData
	if h.useLiveData(r) {
		if live, err := h.buildContractDetailData(r, network, id); err == nil {
			data = live
		} else {
			h.Logger.Warn("live contract shell data failed, falling back to mock", "error", err)
		}
	}
	if data.Address == "" {
		data = mockContractDetailData()
	}

	pages.ContractDetailV2(data).Render(r.Context(), w)
}

func (h *Handlers) buildContractDetailData(r *http.Request, network, contractID string) (pages.ContractDetailData, error) {
	ctx := r.Context()

	metadata, metaErr := h.Gateway.GetContractMetadata(ctx, network, contractID)
	analytics, analyticsErr := h.Gateway.GetContractAnalytics(ctx, network, contractID)
	if metaErr != nil && analyticsErr != nil {
		return pages.ContractDetailData{}, fmt.Errorf("fetching contract detail: metadata=%v analytics=%v", metaErr, analyticsErr)
	}

	recentCalls, _ := h.Gateway.GetContractRecentCalls(ctx, network, contractID, 10)
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
			funcs = append(funcs, pages.ContractFunction{
				Name:     f.Name,
				Calls24h: gateway.FormatNumber(f.Count),
				Calls7d:  "—",
				Calls30d: "—",
			})
		}
		if len(funcs) > 0 {
			data.Functions = funcs
		}

		var activityPoints []float64
		var total7d int64
		var peak int64
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
		data.TotalInvocations = gateway.FormatNumber(totalCalls)
		data.LastInvoked = formatContractAge(analytics.Timeline.LastActivity)
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
			data.Functions = append(data.Functions, pages.ContractFunction{
				Name:     fn.Name,
				Calls24h: gateway.FormatNumber(fn.CallCount),
				Calls7d:  "—",
				Calls30d: "—",
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
