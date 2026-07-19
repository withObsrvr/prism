package handlers

import (
	"context"
	"encoding/json"
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

	// Redirect smart wallets and rules-based smart accounts to the dedicated
	// smart account view. Each detection call is bounded so a slow gateway
	// cannot stall the contract detail page.
	if h.useLiveData(r) {
		redirectSmartAccount := false
		walletType := ""
		walletCtx, walletCancel := context.WithTimeout(r.Context(), 750*time.Millisecond)
		walletInfo, walletErr := h.Gateway.GetSmartWalletInfo(walletCtx, network, id)
		walletCancel()
		if walletErr == nil && walletInfo != nil && walletInfo.IsSmartWallet {
			redirectSmartAccount = true
			walletType = walletInfo.WalletType
		} else {
			rulesCtx, rulesCancel := context.WithTimeout(r.Context(), 750*time.Millisecond)
			state, rulesErr := h.Gateway.GetSmartAccountRules(rulesCtx, network, id, nil)
			rulesCancel()
			if rulesErr == nil && smartAccountStateHasData(state) {
				redirectSmartAccount = true
				walletType = state.Summary.WalletType
			}
		}
		if redirectSmartAccount {
			h.Logger.Info("smart account detected", "contract", id, "wallet_type", walletType)
			prefix := ""
			if strings.HasPrefix(r.URL.Path, "/v2/") {
				prefix = "/v2"
			}
			http.Redirect(w, r, prefix+"/account/"+id+"/smart", http.StatusSeeOther)
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
		if r.URL.Query().Get("mock") == "true" {
			data = mockContractDetailData()
		} else {
			data = unavailableContractDetailData(id, network)
		}
	}

	return data, true
}

func (h *Handlers) buildContractDetailData(r *http.Request, network, contractID string) (pages.ContractDetailData, error) {
	ctx := r.Context()
	balanceLookup := h.startAddressBalanceLookup(ctx, network, contractID)

	storageResp, storageErr := h.Gateway.GetContractStorage(ctx, network, contractID, 100)
	if storageErr != nil {
		h.Logger.Warn("contract storage failed", "error", storageErr, "contract", contractID, "network", network)
	}
	metadata, metaErr := h.Gateway.GetContractMetadata(ctx, network, contractID)
	analytics, analyticsErr := h.Gateway.GetContractAnalytics(ctx, network, contractID)
	if metaErr != nil && analyticsErr != nil && (storageResp == nil || len(storageResp.Entries) == 0) {
		return pages.ContractDetailData{}, fmt.Errorf("fetching contract detail: metadata=%v analytics=%v storage=%v", metaErr, analyticsErr, storageErr)
	}

	recentCalls, recentCallsErr := h.Gateway.GetContractRecentCalls(ctx, network, contractID, 10)
	if recentCallsErr != nil {
		h.Logger.Warn("contract recent calls failed", "error", recentCallsErr, "contract", contractID, "network", network)
	}

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
		if metadata.TotalEntries > 0 {
			data.StorageEntries = gateway.FormatNumber(metadata.TotalEntries)
		}
		data.StateSize = formatBytes(metadata.TotalStateSizeBytes)
		if metadata.EstimatedMonthlyRentStroops > 0 {
			data.MonthlyRent = formatRentXLM(metadata.EstimatedMonthlyRentStroops) + " XLM"
		}
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
		data.StorageItems, data.StorageStats, data.StorageTypes = buildStorageExplorer(storageResp.Entries)
	}
	if metadata != nil {
		if metadata.TotalEntries > 0 {
			data.StorageStats.TotalEntries = gateway.FormatNumber(metadata.TotalEntries)
		}
		if metadata.EstimatedMonthlyRentStroops > 0 {
			data.StorageStats.MonthlyRentXLM = formatRentXLM(metadata.EstimatedMonthlyRentStroops)
		}
	}

	if data.StorageEntries == "" || data.StorageEntries == "0" {
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
	balanceResult := <-balanceLookup
	data.Portfolio = balanceResult.portfolio
	if balanceResult.err != nil && h.Logger != nil {
		h.Logger.Warn("contract balances unavailable", "contract", contractID, "network", network, "error", balanceResult.err)
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
		Portfolio:        unavailableBalancePortfolio(contractID),
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

const (
	storageLedgersPerDay = 17280 // ~5s ledger close time
	// Full health bar represents ~60 days of TTL runway; entries above
	// ~24 days read healthy, 9-24 days at risk, below 9 days critical.
	storageHealthWindow = 60 * storageLedgersPerDay
)

// buildStorageExplorer maps serving-endpoint storage entries into the
// enriched shape the v2 State & rent panel renders, plus sample-level
// health counts and a per-durability breakdown.
func buildStorageExplorer(entries []gateway.ContractStorageEntry) ([]pages.ContractStorageItem, pages.ContractStorageStats, []pages.ContractStorageTypeAgg) {
	items := make([]pages.ContractStorageItem, 0, len(entries))
	healthy, atRisk, critical := 0, 0, 0
	type agg struct {
		entries int
		bytes   int64
		minTTL  int64
	}
	typeAggs := map[string]*agg{}

	for _, entry := range entries {
		// The fetch uses live_only=false (see GetContractStorage); keep
		// live-only semantics by dropping expired entries here.
		if entry.Expired {
			continue
		}
		durability := storageDurability(entry)

		healthPct := 0
		ttlDays := 0
		if entry.TTLRemaining > 0 {
			ttlDays = int(entry.TTLRemaining / storageLedgersPerDay)
			healthPct = int(entry.TTLRemaining * 100 / storageHealthWindow)
			if healthPct > 100 {
				healthPct = 100
			}
			if healthPct < 1 {
				healthPct = 1
			}
			switch {
			case healthPct >= 40:
				healthy++
			case healthPct >= 15:
				atRisk++
			default:
				critical++
			}
		}

		value, valueType := decodeStorageValue(entry)
		items = append(items, pages.ContractStorageItem{
			Key:        decodeStorageKey(entry),
			Type:       durability,
			TypeColor:  storageTypeColor(durability),
			Size:       formatBytes(entry.SizeBytes),
			TTL:        formatStorageTTL(entry.TTLRemaining),
			ValueType:  valueType,
			Value:      value,
			KeyHash:    entry.KeyHash,
			SizeBytes:  int(entry.SizeBytes),
			TTLDays:    ttlDays,
			TTLLedgers: int(entry.TTLRemaining),
			HealthPct:  healthPct,
		})

		a := typeAggs[durability]
		if a == nil {
			a = &agg{minTTL: -1}
			typeAggs[durability] = a
		}
		a.entries++
		a.bytes += entry.SizeBytes
		if entry.TTLRemaining > 0 && (a.minTTL < 0 || entry.TTLRemaining < a.minTTL) {
			a.minTTL = entry.TTLRemaining
		}
	}

	stats := pages.ContractStorageStats{TotalEntries: gateway.FormatNumber(int64(len(items)))}
	if healthy+atRisk+critical > 0 {
		stats.Healthy = gateway.FormatNumber(int64(healthy))
		stats.AtRisk = gateway.FormatNumber(int64(atRisk))
		stats.Critical = gateway.FormatNumber(int64(critical))
	}

	var typeBreakdown []pages.ContractStorageTypeAgg
	for _, name := range []string{"Instance", "Persistent", "Temporary"} {
		a := typeAggs[name]
		if a == nil {
			continue
		}
		minTTL := "—"
		if a.minTTL >= 0 {
			minTTL = fmt.Sprintf("%dd", a.minTTL/storageLedgersPerDay)
		}
		typeBreakdown = append(typeBreakdown, pages.ContractStorageTypeAgg{
			Name:    name,
			Entries: gateway.FormatNumber(int64(a.entries)),
			Size:    formatBytes(a.bytes),
			MinTTL:  minTTL,
		})
	}

	return items, stats, typeBreakdown
}

// formatRentXLM renders a stroop amount as compact XLM (e.g. "54.3").
func formatRentXLM(stroops int64) string {
	switch {
	case stroops >= 10_000_000_000:
		return gateway.FormatNumber((stroops + 5_000_000) / 10_000_000)
	case stroops >= 100_000_000:
		tenths := (stroops + 500_000) / 1_000_000
		return fmt.Sprintf("%d.%d", tenths/10, tenths%10)
	default:
		hundredths := (stroops + 50_000) / 100_000
		return fmt.Sprintf("%d.%02d", hundredths/100, hundredths%100)
	}
}

func formatStorageTTL(ttlLedgers int64) string {
	if ttlLedgers <= 0 {
		return "—"
	}
	days := ttlLedgers / storageLedgersPerDay
	if days == 1 {
		return "1 day"
	}
	if days > 0 {
		return fmt.Sprintf("%d days", days)
	}
	return "< 1 day"
}

// decodedScVal matches the gateway's decoded key/value wrapper:
// {"type": "vec", "value": [...], "display": "[Randomness, 30199779]"}
type decodedScVal struct {
	Type    string          `json:"type"`
	Value   json.RawMessage `json:"value"`
	Display string          `json:"display"`
}

// decodeStorageKey renders key_decoded into a compact display key,
// falling back to the truncated raw key hash.
func decodeStorageKey(entry gateway.ContractStorageEntry) string {
	var kd decodedScVal
	if len(entry.KeyDecoded) > 0 && json.Unmarshal(entry.KeyDecoded, &kd) == nil {
		if kd.Type == "ledger_key_contract_instance" {
			return "Contract instance"
		}
		// Vec keys like [Balance, GDKX...] read best in Key:Part form.
		if kd.Type == "vec" && len(kd.Value) > 0 {
			var parts []decodedScVal
			if json.Unmarshal(kd.Value, &parts) == nil && len(parts) > 0 {
				joined := make([]string, 0, len(parts))
				for _, p := range parts {
					joined = append(joined, truncateMiddle(p.Display, 24))
				}
				return truncateMiddle(strings.Join(joined, ":"), 64)
			}
		}
		if kd.Display != "" {
			return truncateMiddle(kd.Display, 64)
		}
	}
	return truncateMiddle(entry.Key, 24)
}

// decodeStorageValue renders value_decoded into a display string and a
// value-type label, falling back to the raw XDR data_value.
func decodeStorageValue(entry gateway.ContractStorageEntry) (string, string) {
	var vd decodedScVal
	if len(entry.ValueDecoded) > 0 && json.Unmarshal(entry.ValueDecoded, &vd) == nil && vd.Type != "" {
		return truncateMiddle(renderScVal(entry.ValueDecoded, 0), 400), prettyScType(vd.Type)
	}
	if entry.DataValue != "" {
		return truncateMiddle(entry.DataValue, 200), "XDR"
	}
	return "", ""
}

const scValMaxDepth = 3

// renderScVal renders a decoded ScVal wrapper into a readable string,
// recursing into maps and vecs whose top-level display is only a summary
// like "map{6}". Falls back to the gateway's display string.
func renderScVal(raw json.RawMessage, depth int) string {
	var v decodedScVal
	if json.Unmarshal(raw, &v) != nil || v.Type == "" {
		return ""
	}
	switch v.Type {
	case "bytes":
		// The display is just "bytes[N]"; the hex payload is more useful.
		var b struct {
			Hex string `json:"hex"`
		}
		if json.Unmarshal(v.Value, &b) == nil && b.Hex != "" {
			return "0x" + b.Hex
		}
	case "map":
		if depth < scValMaxDepth {
			var m struct {
				Entries []struct {
					Key   json.RawMessage `json:"key"`
					Value json.RawMessage `json:"value"`
				} `json:"entries"`
			}
			if json.Unmarshal(v.Value, &m) == nil && len(m.Entries) > 0 {
				parts := make([]string, 0, len(m.Entries))
				for _, e := range m.Entries {
					parts = append(parts, renderScVal(e.Key, depth+1)+": "+renderScVal(e.Value, depth+1))
				}
				// Top-level maps with several entries read best one per line.
				if depth == 0 && len(parts) > 3 {
					return "{\n  " + strings.Join(parts, ",\n  ") + "\n}"
				}
				return "{" + strings.Join(parts, ", ") + "}"
			}
		}
	case "vec":
		if depth < scValMaxDepth {
			var items []json.RawMessage
			if json.Unmarshal(v.Value, &items) == nil && len(items) > 0 {
				parts := make([]string, 0, len(items))
				for _, it := range items {
					parts = append(parts, renderScVal(it, depth+1))
				}
				return "[" + strings.Join(parts, ", ") + "]"
			}
		}
	case "contract_instance":
		var inst struct {
			ExecutableType string `json:"executable_type"`
			StorageEntries int64  `json:"storage_entries"`
		}
		if json.Unmarshal(v.Value, &inst) == nil {
			exec := strings.TrimPrefix(inst.ExecutableType, "ContractExecutableTypeContractExecutable")
			if exec == "" {
				exec = "Contract"
			}
			return fmt.Sprintf("%s executable · %d instance storage entries", exec, inst.StorageEntries)
		}
	}
	return v.Display
}

func prettyScType(t string) string {
	switch strings.ToLower(t) {
	case "u32", "u64", "u128", "u256":
		return "U" + t[1:]
	case "i32", "i64", "i128", "i256":
		return t
	case "bool":
		return "Bool"
	case "contract_instance":
		return "Instance"
	default:
		return titleCase(t)
	}
}

// storageDurability extracts the clean durability label. The serving
// endpoint puts it in `type` ("persistent"); `durability` carries the XDR
// enum name ("ContractDataDurabilityPersistent").
func storageDurability(entry gateway.ContractStorageEntry) string {
	if entry.Type != "" {
		return titleCase(entry.Type)
	}
	return titleCase(strings.TrimPrefix(entry.Durability, "ContractDataDurability"))
}

func (h *Handlers) ContractEvents(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if isHTMX(r) {
		fmt.Fprintf(w, `<div id="tab-content">Events for %s</div>`, id)
		return
	}
	fmt.Fprintf(w, "Contract Events: %s", id)
}
