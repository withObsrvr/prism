package humanize

import (
	"fmt"
	"strings"

	"github.com/withObsrvr/prism/internal/gateway"
)

type HumanContractSummary struct {
	Narrative       string
	Context         string
	FunctionSummary string
	StorageSummary  string
	Signals         []ContractSignal
	Evidence        []ContractEvidence
}

type ContractSignal struct {
	Title    string
	Severity string
	Summary  string
}

type ContractEvidence struct {
	Label string
	Value string
}

func BuildContractSummary(meta *gateway.ContractMetadata, analytics *gateway.ContractAnalytics) HumanContractSummary {
	summary := HumanContractSummary{}
	name := contractDisplayName(meta)
	orderedFunctions := observedFunctionNames(meta, analytics)
	project := InferProject(name)
	behavior := inferContractBehavior(orderedFunctions, project)

	summary.Narrative = contractNarrative(name, project, behavior, orderedFunctions)
	summary.Context = contractContext(meta, analytics)
	summary.FunctionSummary = buildContractFunctionSummary(orderedFunctions)
	summary.StorageSummary = buildContractStorageSummary(meta)
	summary.Signals = buildContractSignals(name, project, behavior, orderedFunctions, meta, analytics)
	summary.Evidence = buildContractEvidence(name, project, behavior, orderedFunctions, meta, analytics)
	return summary
}

func contractDisplayName(meta *gateway.ContractMetadata) string {
	if meta != nil && meta.DisplayName != "" {
		return meta.DisplayName
	}
	if meta != nil && meta.ContractID != "" {
		return gateway.ShortAddress(meta.ContractID)
	}
	return "This contract"
}

func observedFunctionNames(meta *gateway.ContractMetadata, analytics *gateway.ContractAnalytics) []string {
	seen := map[string]bool{}
	out := make([]string, 0)
	if analytics != nil {
		for _, fn := range analytics.TopFunctions {
			if fn.Name == "" || seen[fn.Name] {
				continue
			}
			seen[fn.Name] = true
			out = append(out, fn.Name)
		}
	}
	if meta != nil {
		for _, fn := range meta.ExportedFunctions {
			if fn.Name == "" || seen[fn.Name] {
				continue
			}
			seen[fn.Name] = true
			out = append(out, fn.Name)
		}
	}
	return out
}

func inferContractBehavior(functionNames []string, project string) string {
	switch {
	case looksManagedToken(functionNames):
		return "managed_token"
	case looksTokenLike(functionNames):
		return "token"
	case looksOracleLike(functionNames):
		return "oracle"
	case looksDexLike(functionNames) || project == "soroswap":
		return "dex"
	case looksFactoryLike(functionNames):
		return "factory"
	case looksAdminHeavy(functionNames):
		return "admin"
	default:
		return "generic"
	}
}

func contractNarrative(name, project, behavior string, functionNames []string) string {
	switch behavior {
	case "managed_token":
		return fmt.Sprintf("%s appears to be a managed token contract with mint, burn, transfer, and compliance-style controls.", name)
	case "token":
		return fmt.Sprintf("%s appears to be a token-style contract with familiar transfer and approval functions.", name)
	case "oracle":
		return fmt.Sprintf("%s appears to be an oracle-style contract focused on publishing or updating price data.", name)
	case "dex":
		if project != "" {
			return fmt.Sprintf("%s appears to be a %s contract focused on swaps and liquidity management.", name, strings.ToUpper(project[:1])+project[1:])
		}
		return fmt.Sprintf("%s appears to be a DEX-style contract focused on swaps and liquidity management.", name)
	case "factory":
		return fmt.Sprintf("%s appears to be a factory-style contract that creates pools, vaults, or other protocol components.", name)
	case "admin":
		return fmt.Sprintf("%s appears to expose mostly administrative or control-oriented contract functions.", name)
	default:
		if project != "" {
			return fmt.Sprintf("%s appears to belong to the %s project and exposes %d observed functions.", name, strings.ToUpper(project[:1])+project[1:], len(functionNames))
		}
		if len(functionNames) > 0 {
			return fmt.Sprintf("%s exposes %d functions and looks like an active Soroban contract.", name, len(functionNames))
		}
		return fmt.Sprintf("%s is a Soroban contract with limited semantic detail available so far.", name)
	}
}

func contractContext(meta *gateway.ContractMetadata, analytics *gateway.ContractAnalytics) string {
	parts := []string{}
	if meta != nil && meta.CreatorAddress != "" {
		parts = append(parts, "Created by "+gateway.ShortAddress(meta.CreatorAddress))
	}
	if analytics != nil {
		if analytics.Timeline.LastActivity != "" {
			parts = append(parts, "Recent activity observed")
		} else if contractObservedInvocations(analytics) > 0 {
			parts = append(parts, "Observed function activity without a recent timestamp")
		}
	}
	if meta != nil && meta.TotalStateSizeBytes > 0 {
		parts = append(parts, fmt.Sprintf("using about %s of state", humanBytes(meta.TotalStateSizeBytes)))
	}
	return strings.Join(parts, " · ")
}

func buildContractFunctionSummary(functionNames []string) string {
	if len(functionNames) == 0 {
		return "No exported function information is available yet."
	}
	categoryCounts := map[string]int{}
	readable := make([]string, 0, min(4, len(functionNames)))
	for _, fn := range functionNames {
		if rule, ok := LookupFunctionNarration(fn); ok {
			if rule.Phrase != "" {
				readable = append(readable, summaryPhrase(rule.Phrase, fn))
			} else {
				readable = append(readable, HumanizeFunctionName(fn))
			}
			if rule.Category != "" {
				categoryCounts[rule.Category]++
			}
		} else {
			readable = append(readable, HumanizeFunctionName(fn))
		}
		if len(readable) == 4 {
			break
		}
	}
	prefix := "Most observed behavior involves " + naturalJoin(readable) + "."
	if topCategory := dominantCategory(categoryCounts); topCategory != "" {
		prefix += " Observed usage is concentrated in " + strings.ReplaceAll(topCategory, "_", " ") + " functions."
	}
	return prefix
}

func summaryPhrase(phrase, functionName string) string {
	phrase = strings.TrimSpace(phrase)
	if phrase == "" {
		return HumanizeFunctionName(functionName)
	}
	switch phrase {
	case "updated a price":
		return "price updates"
	case "created a pool":
		return "pool creation"
	case "created a vault":
		return "vault creation"
	case "created a trading pair":
		return "trading pair creation"
	case "created a market":
		return "market creation"
	case "initialized a pool":
		return "pool initialization"
	case "queried total fees":
		return "fee queries"
	case "queried pool reserves":
		return "reserve queries"
	case "looked up a pool":
		return "pool lookups"
	case "burned tokens":
		return "token burns"
	case "minted tokens":
		return "token minting"
	case "transferred assets":
		return "asset transfers"
	case "approved spending":
		return "spending approvals"
	case "added liquidity":
		return "liquidity adds"
	case "removed liquidity":
		return "liquidity removals"
	}
	if strings.HasPrefix(phrase, "created ") {
		return strings.TrimPrefix(phrase, "created ") + " creation"
	}
	if strings.HasPrefix(phrase, "updated ") {
		return strings.TrimPrefix(phrase, "updated ") + " updates"
	}
	if strings.HasPrefix(phrase, "queried ") {
		return strings.TrimPrefix(phrase, "queried ") + " queries"
	}
	return phrase
}

func naturalJoin(items []string) string {
	switch len(items) {
	case 0:
		return ""
	case 1:
		return items[0]
	case 2:
		return items[0] + " and " + items[1]
	default:
		return strings.Join(items[:len(items)-1], ", ") + ", and " + items[len(items)-1]
	}
}

func dominantCategory(counts map[string]int) string {
	best := ""
	bestCount := 0
	for k, v := range counts {
		if v > bestCount {
			best = k
			bestCount = v
		}
	}
	return best
}

func buildContractStorageSummary(meta *gateway.ContractMetadata) string {
	if meta == nil {
		return "No storage summary is available yet."
	}
	entries := meta.TotalEntries
	size := humanBytes(meta.TotalStateSizeBytes)
	if entries == 0 && meta.TotalStateSizeBytes == 0 {
		return "This contract currently shows little or no indexed storage."
	}
	return fmt.Sprintf("This contract currently has %s storage entries using about %s of state.", gateway.FormatNumber(entries), size)
}

func buildContractSignals(name, project, behavior string, functionNames []string, meta *gateway.ContractMetadata, analytics *gateway.ContractAnalytics) []ContractSignal {
	var signals []ContractSignal
	switch behavior {
	case "managed_token":
		signals = append(signals,
			ContractSignal{Title: "Managed token controls", Severity: "warn", Summary: "This contract exposes mint, burn, or compliance-style controls in addition to transfer behavior."},
			ContractSignal{Title: "Token-like interface", Severity: "info", Summary: "Exported functions suggest token-style behavior with transfer and supply-management flows."},
		)
	case "token":
		signals = append(signals, ContractSignal{Title: "Token-like interface", Severity: "info", Summary: "Exported functions suggest token-style behavior with transfer and approval flows."})
	case "oracle":
		signals = append(signals, ContractSignal{Title: "Oracle-style behavior", Severity: "info", Summary: "Observed functions suggest this contract publishes or updates prices or rates."})
	case "dex":
		signals = append(signals, ContractSignal{Title: "DEX activity", Severity: "info", Summary: "Observed functions suggest swap routing or liquidity management behavior."})
	case "factory":
		signals = append(signals, ContractSignal{Title: "Factory-style behavior", Severity: "info", Summary: "Observed functions suggest this contract creates pools, vaults, markets, or other protocol components."})
	}
	if project != "" {
		signals = append(signals, ContractSignal{Title: "Known project", Severity: "info", Summary: fmt.Sprintf("This contract name suggests it is associated with the %s project.", strings.Title(project))})
	}
	if meta != nil && meta.TotalStateSizeBytes > 128*1024 {
		signals = append(signals, ContractSignal{Title: "Large state footprint", Severity: "warn", Summary: fmt.Sprintf("%s currently uses about %s of state, which is large enough to merit review.", name, humanBytes(meta.TotalStateSizeBytes))})
	}
	if analytics != nil {
		observed := contractObservedInvocations(analytics)
		if observed == 0 && analytics.Timeline.LastActivity == "" {
			signals = append(signals, ContractSignal{Title: "Limited activity", Severity: "info", Summary: "Prism has not observed recent invocation activity for this contract."})
		}
	}
	return dedupeContractSignals(signals)
}

func buildContractEvidence(name, project, behavior string, functionNames []string, meta *gateway.ContractMetadata, analytics *gateway.ContractAnalytics) []ContractEvidence {
	evidence := []ContractEvidence{}
	if name != "" {
		evidence = append(evidence, ContractEvidence{Label: "Name", Value: name})
	}
	if project != "" {
		evidence = append(evidence, ContractEvidence{Label: "Project inference", Value: project})
	}
	if behavior != "generic" {
		evidence = append(evidence, ContractEvidence{Label: "Behavior inference", Value: strings.ReplaceAll(behavior, "_", "-")})
	}
	if len(functionNames) > 0 {
		shown := functionNames
		if len(shown) > 5 {
			shown = shown[:5]
		}
		evidence = append(evidence, ContractEvidence{Label: "Observed functions", Value: strings.Join(shown, ", ")})
	}
	if meta != nil {
		if meta.CreatorAddress != "" {
			evidence = append(evidence, ContractEvidence{Label: "Creator", Value: gateway.ShortAddress(meta.CreatorAddress)})
		}
		if meta.TotalEntries > 0 {
			evidence = append(evidence, ContractEvidence{Label: "Storage entries", Value: gateway.FormatNumber(meta.TotalEntries)})
		}
	}
	if analytics != nil {
		if totalCalls := contractObservedInvocations(analytics); totalCalls > 0 {
			evidence = append(evidence, ContractEvidence{Label: "Observed invocations", Value: gateway.FormatNumber(totalCalls)})
		}
	}
	return evidence
}

func dedupeContractSignals(in []ContractSignal) []ContractSignal {
	seen := map[string]bool{}
	out := make([]ContractSignal, 0, len(in))
	for _, s := range in {
		k := s.Title + ":" + s.Summary
		if seen[k] {
			continue
		}
		seen[k] = true
		out = append(out, s)
	}
	return out
}

func looksTokenLike(functionNames []string) bool {
	set := functionSet(functionNames)
	return set["transfer"] && (set["approve"] || set["transfer_from"])
}

func looksManagedToken(functionNames []string) bool {
	set := functionSet(functionNames)
	return (set["mint"] || set["mint_to"] || set["burn"]) && (set["transfer"] || set["transfer_from"]) && (set["add_to_blocked_list"] || set["remove_from_blocked_list"] || set["set_revoker"] || set["change_mint_budget"])
}

func looksOracleLike(functionNames []string) bool {
	set := functionSet(functionNames)
	return set["set_price"] || set["update_price"] || set["publish_price"] || set["set_prices"]
}

func looksDexLike(functionNames []string) bool {
	set := functionSet(functionNames)
	return set["swap"] || set["add_liquidity"] || set["remove_liquidity"] || set["get_reserves"]
}

func looksFactoryLike(functionNames []string) bool {
	set := functionSet(functionNames)
	return set["create_pool"] || set["create_vault"] || set["create_pair"] || set["initialize_pool"] || set["create_market"]
}

func looksAdminHeavy(functionNames []string) bool {
	count := 0
	for _, fn := range functionNames {
		if rule, ok := LookupFunctionNarration(fn); ok && (strings.Contains(rule.Category, "admin") || strings.Contains(rule.Category, "compliance")) {
			count++
		}
	}
	return count >= 2
}

func functionSet(functionNames []string) map[string]bool {
	set := map[string]bool{}
	for _, fn := range functionNames {
		set[fn] = true
	}
	return set
}

func contractObservedInvocations(analytics *gateway.ContractAnalytics) int64 {
	if analytics == nil {
		return 0
	}
	totalCalls := analytics.Stats.TotalCallsAsCaller + analytics.Stats.TotalCallsAsCallee
	if totalCalls > 0 {
		return totalCalls
	}
	var topTotal int64
	for _, fn := range analytics.TopFunctions {
		topTotal += fn.Count
	}
	return topTotal
}

func humanBytes(n int64) string {
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

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
