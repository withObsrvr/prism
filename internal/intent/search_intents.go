package intent

import (
	"context"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/withObsrvr/prism/internal/gateway"
	prismsearch "github.com/withObsrvr/prism/internal/search"
	"github.com/withObsrvr/prism/internal/txoutcome"
)

type transactionFailureHandler struct{}

func (transactionFailureHandler) ID() ID { return TransactionFailure }

func (transactionFailureHandler) Match(input string, _ *Registry) (Match, bool) {
	entity := prismsearch.ExtractIdentifier(input)
	if entity.Type != prismsearch.ClassTxHash || prismsearch.Classify(input).Known() {
		return Match{}, false
	}
	normalized := normalize(input)
	if !containsAny(normalized, "fail", "failed", "failure", "error", "revert", "wrong") {
		return Match{}, false
	}
	return Match{ID: TransactionFailure, Confidence: 0.96, Slots: map[string]string{"tx_hash": entity.Value}, Reason: "transaction hash with failure question"}, true
}

func (transactionFailureHandler) Execute(ctx context.Context, env Env, match Match) (Result, error) {
	hash := match.Slots["tx_hash"]
	result := Result{
		Title:      "Transaction failure evidence",
		Confidence: match.Confidence,
		Evidence:   []EvidenceLink{{Label: "Transaction " + short(hash), Href: "/v2/tx/" + url.PathEscape(hash)}},
		Actions:    []ActionLink{{Label: "Inspect transaction receipt", Href: "/v2/tx/" + url.PathEscape(hash)}},
	}
	if env.Gateway == nil {
		result.Answer = "Prism recognized the transaction failure question, but live transaction outcome evidence is unavailable."
		result.Warnings = []string{"Gateway client is not configured."}
		return result, nil
	}
	outcome, outcomeErr := env.Gateway.GetTransactionOutcome(ctx, env.Network, hash)
	if outcomeErr == nil {
		return transactionOutcomeResult(result, outcome), nil
	}

	// E1A is additive and is not yet deployed on every network. Preserve the
	// truthful receipt-only answer until the structured outcome packet reaches
	// mainnet, but make the limitation visible.
	receipt, err := env.Gateway.GetTransactionReceipt(ctx, env.Network, hash)
	if err != nil {
		return result, err
	}
	result.Warnings = append(result.Warnings, "Structured transaction outcome evidence is unavailable; Prism used the consolidated receipt as a fallback.")
	return transactionReceiptFallbackResult(result, receipt), nil
}

func transactionOutcomeResult(result Result, outcome *gateway.TransactionOutcome) Result {
	interpretation := txoutcome.Interpret(outcome)
	result.EvidenceAvailable = true
	status := firstNonEmpty(interpretation.Outcome, "unknown")
	result.Metrics = []Metric{
		{Label: "Status", Value: status},
		{Label: "Operations", Value: fmt.Sprintf("%d", len(outcome.Operations))},
		{Label: "Ledger", Value: fmt.Sprintf("%d", outcome.LedgerSequence)},
	}
	if interpretation.ReasonLabel != "" {
		result.Metrics = append(result.Metrics, Metric{Label: "Reason", Value: interpretation.ReasonLabel})
	}
	if interpretation.OperationNumber > 0 {
		result.Metrics = append(result.Metrics, Metric{Label: "Failed operation", Value: fmt.Sprintf("#%d %s", interpretation.OperationNumber, firstNonEmpty(interpretation.OperationLabel, "operation"))})
	}
	if interpretation.PhaseLabel != "" {
		result.Metrics = append(result.Metrics, Metric{Label: "Failure phase", Value: interpretation.PhaseLabel})
	}
	if outcome.LedgerSequence > 0 {
		result.Evidence = append(result.Evidence, EvidenceLink{Label: fmt.Sprintf("Ledger %d", outcome.LedgerSequence), Href: fmt.Sprintf("/v2/ledger/%d", outcome.LedgerSequence)})
	}

	if interpretation.Outcome == "succeeded" {
		result.Answer = "The authoritative transaction result marks this transaction successful, so there is no on-chain failure to explain."
		return result
	}

	result.Answer = interpretation.Summary
	if invocation := outcome.PrimaryInvocation; invocation != nil {
		result.Answer += " The primary invocation was " + invocationLabel(invocation) + "."
		result.Metrics = append(result.Metrics, Metric{Label: "Invocation", Value: firstNonEmpty(invocation.FunctionName, "contract call")})
	}
	result.Warnings = append(result.Warnings, interpretation.Caveats...)
	if interpretation.EvidenceStatus == "partial" {
		result.Warnings = append(result.Warnings, "Some optional transaction evidence is incomplete; the transaction result remains authoritative.")
	} else if interpretation.EvidenceStatus == "stale" {
		result.Warnings = append(result.Warnings, "Some optional transaction evidence is delayed; the transaction result remains authoritative.")
	}
	return result
}

func transactionReceiptFallbackResult(result Result, receipt *gateway.TxReceipt) Result {
	result.EvidenceAvailable = true
	status := "failed"
	if receipt.Successful {
		status = "successful"
	}
	result.Metrics = []Metric{
		{Label: "Status", Value: status},
		{Label: "Operations", Value: fmt.Sprintf("%d", receipt.OperationCount)},
		{Label: "Ledger", Value: fmt.Sprintf("%d", receipt.LedgerSequence)},
	}
	if receipt.LedgerSequence > 0 {
		result.Evidence = append(result.Evidence, EvidenceLink{Label: fmt.Sprintf("Ledger %d", receipt.LedgerSequence), Href: fmt.Sprintf("/v2/ledger/%d", receipt.LedgerSequence)})
	}
	if receipt.Successful {
		result.Answer = "This transaction is marked successful in the consolidated receipt, so Prism does not have failure evidence to explain."
		return result
	}

	result.Answer = "The consolidated receipt marks this transaction failed."
	if edge, ok := firstFailedCallEdge(receipt.Semantic.CallGraph); ok {
		result.Answer += " The decoded semantic call graph shows an unsuccessful " + firstNonEmpty(edge.Function, "contract") + " call to " + short(edge.To) + "."
		result.Metrics = append(result.Metrics, Metric{Label: "Failed call", Value: firstNonEmpty(edge.Function, "contract call")})
	}
	classification := strings.TrimSpace(receipt.Semantic.Classification.TxType)
	if classification != "" {
		result.Answer += " Prism classified the transaction as " + classification + "."
		result.Metrics = append(result.Metrics, Metric{Label: "Type", Value: classification})
	}
	result.Answer += " The available receipt does not expose a decoded result code, so Prism cannot state the underlying cause."
	result.Warnings = append(result.Warnings, "No decoded transaction result code was supplied by the current receipt contract.")
	return result
}

func invocationLabel(invocation *gateway.TransactionPrimaryInvocation) string {
	function := firstNonEmpty(invocation.FunctionName, "contract call")
	arguments := make([]string, 0, len(invocation.Arguments))
	for _, argument := range invocation.Arguments {
		label := strings.TrimSpace(argument.Display)
		if label == "" && argument.Value != nil {
			label = strings.TrimSpace(fmt.Sprint(argument.Value))
		}
		if label != "" {
			arguments = append(arguments, label)
		}
		if len(arguments) == 3 {
			break
		}
	}
	call := function + "()"
	if len(arguments) > 0 {
		call = function + "(" + strings.Join(arguments, ", ") + ")"
	}
	if invocation.ContractID != "" {
		call += " on " + short(invocation.ContractID)
	}
	return call
}

type contractActivityHandler struct{}

func (contractActivityHandler) ID() ID { return ContractActivity }

func (contractActivityHandler) Match(input string, _ *Registry) (Match, bool) {
	entity := prismsearch.ExtractIdentifier(input)
	if entity.Type != prismsearch.ClassContract || prismsearch.Classify(input).Known() {
		return Match{}, false
	}
	normalized := normalize(input)
	if !containsAny(normalized, "busy", "active", "activity", "calls", "used", "usage", "popular") {
		return Match{}, false
	}
	requested := timeSlot(normalized)
	return Match{ID: ContractActivity, Confidence: 0.9, Slots: map[string]string{"contract_id": entity.Value, "requested_time": requested, "window": "seven_daily_buckets"}, Reason: "contract identifier with activity wording"}, true
}

func (contractActivityHandler) Execute(ctx context.Context, env Env, match Match) (Result, error) {
	contractID := match.Slots["contract_id"]
	result := Result{
		Title:      "Contract activity",
		Confidence: match.Confidence,
		Evidence:   []EvidenceLink{{Label: "Contract " + short(contractID), Href: "/v2/contract/" + url.PathEscape(contractID)}},
		Actions:    []ActionLink{{Label: "Explore contract activity", Href: "/v2/explore?q=" + url.QueryEscape(contractID)}},
	}
	if env.Gateway == nil {
		result.Answer = "Prism recognized the contract activity question, but live contract analytics are unavailable."
		result.Warnings = []string{"Gateway client is not configured."}
		return result, nil
	}
	analytics, err := env.Gateway.GetContractAnalytics(ctx, env.Network, contractID)
	if err != nil {
		return result, err
	}
	result.EvidenceAvailable = true
	allTime := analytics.Stats.TotalCallsAsCaller + analytics.Stats.TotalCallsAsCallee
	sevenDay := int64(0)
	for _, bucket := range analytics.DailyCalls7D {
		sevenDay += bucket.Count
	}
	topFunctionCalls := int64(0)
	for _, function := range analytics.TopFunctions {
		topFunctionCalls += function.Count
	}
	statsConsistent := allTime >= topFunctionCalls
	if len(analytics.DailyCalls7D) > 0 {
		result.Answer = fmt.Sprintf("The contract analytics report %d calls across %d returned daily buckets", sevenDay, len(analytics.DailyCalls7D))
		result.Metrics = append(result.Metrics, Metric{Label: "Daily bucket calls", Value: fmt.Sprintf("%d", sevenDay)})
	} else {
		result.Answer = "The contract analytics returned no daily activity buckets"
	}
	if statsConsistent {
		result.Answer += fmt.Sprintf(" and %d all-time observed calls", allTime)
		result.Metrics = append(result.Metrics, Metric{Label: "All-time calls", Value: fmt.Sprintf("%d", allTime)}, Metric{Label: "Unique callers", Value: fmt.Sprintf("%d", analytics.Stats.UniqueCallers)})
	} else {
		result.Answer += fmt.Sprintf(". Its top-function evidence records at least %d calls, but the aggregate call statistics are inconsistent and have been omitted", topFunctionCalls)
		result.Warnings = append(result.Warnings, "The returned aggregate call totals are lower than the supplied top-function counts.")
	}
	if len(analytics.TopFunctions) > 0 {
		result.Answer += ". Its most observed function is " + analytics.TopFunctions[0].Name + fmt.Sprintf(" with %d calls", analytics.TopFunctions[0].Count)
		result.Metrics = append(result.Metrics, Metric{Label: "Top function", Value: analytics.TopFunctions[0].Name})
	}
	result.Answer += "."
	if requested := match.Slots["requested_time"]; requested != "" && requested != "7d" {
		result.Warnings = append(result.Warnings, "This endpoint supplies up to seven daily buckets, not "+humanTime(requested)+".")
	}
	return result, nil
}

type assetActivityHandler struct{}

func (assetActivityHandler) ID() ID { return AssetActivity }

func (assetActivityHandler) Match(input string, _ *Registry) (Match, bool) {
	normalized := normalize(input)
	if !containsAny(normalized, "busy", "active", "activity", "holders", "transfers", "volume", "used", "usage") || !questionLike(input) {
		return Match{}, false
	}
	asset := unambiguousAssetIdentity(input)
	if asset == "" {
		return Match{}, false
	}
	requested := timeSlot(normalized)
	if requested == "" {
		requested = "24h"
	}
	return Match{ID: AssetActivity, Confidence: 0.86, Slots: map[string]string{"asset": asset, "requested_time": requested, "window": "24h"}, Reason: "unambiguous asset with activity question"}, true
}

func (assetActivityHandler) Execute(ctx context.Context, env Env, match Match) (Result, error) {
	asset := match.Slots["asset"]
	displayAsset := assetDisplayLabel(asset)
	result := Result{Title: "Asset activity: " + displayAsset, Confidence: match.Confidence}
	if env.Gateway == nil {
		result.Answer = "Prism recognized the asset activity question, but live asset evidence is unavailable."
		result.Warnings = []string{"Gateway client is not configured."}
		result.Actions = []ActionLink{{Label: "Open asset", Href: "/v2/assets/" + url.PathEscape(asset)}}
		return result, nil
	}
	detail, err := env.Gateway.GetAssetDetail(ctx, env.Network, asset)
	if err != nil {
		return result, err
	}
	result.EvidenceAvailable = true
	label := firstNonEmpty(detail.Symbol, detail.AssetCode, detail.DisplayName, displayAsset)
	slug := firstNonEmpty(detail.CanonicalSlug, detail.ContractID, asset)
	result.Answer = fmt.Sprintf("The current %s asset record reports %d transfers and %d unique accounts in the last 24 hours", label, detail.Transfers24H, detail.UniqueAccounts24H)
	result.Metrics = []Metric{{Label: "Transfers, 24h", Value: fmt.Sprintf("%d", detail.Transfers24H)}, {Label: "Unique accounts, 24h", Value: fmt.Sprintf("%d", detail.UniqueAccounts24H)}, {Label: "Holders", Value: fmt.Sprintf("%d", detail.HolderCount)}}
	if detail.Volume24H != "" {
		result.Answer += ", with reported volume of " + detail.Volume24H
		result.Metrics = append(result.Metrics, Metric{Label: "Volume, 24h", Value: detail.Volume24H})
	}
	result.Answer += "."
	if requested := match.Slots["requested_time"]; requested != "" && requested != "24h" {
		result.Warnings = append(result.Warnings, "The asset record supplies 24-hour activity, not "+humanTime(requested)+".")
	}
	result.Evidence = []EvidenceLink{{Label: label + " asset record", Href: "/v2/assets/" + url.PathEscape(slug)}}
	result.Actions = []ActionLink{{Label: "Open " + label, Href: "/v2/assets/" + url.PathEscape(slug)}, {Label: "Explore transfers", Href: "/v2/explore?asset=" + url.QueryEscape(asset) + "&topic=transfer&time=24h"}}
	return result, nil
}

type recentFailuresHandler struct{}

func (recentFailuresHandler) ID() ID { return RecentFailures }

func (recentFailuresHandler) Match(input string, _ *Registry) (Match, bool) {
	normalized := normalize(input)
	if !questionLike(input) || !containsAny(normalized, "fail", "failed", "failures", "failing") || !containsAny(normalized, "transaction", "transactions", "transfer", "transfers", "calls", "activity") {
		return Match{}, false
	}
	parsed, _ := prismsearch.Parse(input)
	requested := timeSlot(normalized)
	if requested == "" {
		requested = "1h"
	}
	return Match{ID: RecentFailures, Confidence: 0.84, Slots: map[string]string{"requested_time": requested, "topic": parsed.Topic, "asset": parsed.Asset}, Reason: "recent failure question"}, true
}

func (recentFailuresHandler) Execute(ctx context.Context, env Env, match Match) (Result, error) {
	result := Result{Title: "Recent transaction failures", Confidence: match.Confidence, Actions: []ActionLink{{Label: "Explore failed activity", Href: recentFailureExploreHref(match)}}}
	if env.Gateway == nil {
		result.Answer = "Prism recognized the recent-failure question, but live decoded transactions are unavailable."
		result.Warnings = []string{"Gateway client is not configured."}
		return result, nil
	}
	response, err := env.Gateway.GetSilverRecentTransactions(ctx, env.Network, 100)
	if err != nil {
		return result, err
	}
	result.EvidenceAvailable = true
	transactions := append([]gateway.DecodedTransaction(nil), response.Transactions...)
	sort.SliceStable(transactions, func(i, j int) bool { return transactions[i].ClosedAt < transactions[j].ClosedAt })
	window := match.Slots["requested_time"]
	cutoff := time.Now().Add(-durationForWindow(window))
	considered := 0
	failed := 0
	oldest := time.Time{}
	for _, transaction := range transactions {
		closedAt, parseErr := time.Parse(time.RFC3339, transaction.ClosedAt)
		if parseErr != nil || closedAt.Before(cutoff) || !matchesFailureSubject(transaction, match.Slots["topic"], match.Slots["asset"]) {
			continue
		}
		if oldest.IsZero() || closedAt.Before(oldest) {
			oldest = closedAt
		}
		considered++
		if !transaction.Successful {
			failed++
			if len(result.Evidence) < 5 {
				result.Evidence = append(result.Evidence, EvidenceLink{Label: "Failed transaction " + short(transaction.TxHash), Href: "/v2/tx/" + url.PathEscape(transaction.TxHash)})
			}
		}
	}
	subject := "decoded transactions"
	if topic := match.Slots["topic"]; topic != "" {
		subject = topic + " transactions"
	}
	result.Answer = fmt.Sprintf("Among %d %s returned inside %s, %d failed.", considered, subject, humanTime(window), failed)
	result.Metrics = []Metric{{Label: "Returned in window", Value: fmt.Sprintf("%d", considered)}, {Label: "Failed", Value: fmt.Sprintf("%d", failed)}, {Label: "Collection limit", Value: "100"}}
	result.Warnings = append(result.Warnings, "This answer inspects the latest bounded collection of up to 100 decoded transactions. The endpoint does not prove complete coverage of "+humanTime(window)+".")
	if len(transactions) >= 100 && (oldest.IsZero() || oldest.After(cutoff)) {
		result.Warnings = append(result.Warnings, "The collection limit was reached before Prism could observe the start of the requested window.")
	}
	return result, nil
}

func firstFailedCallEdge(edges []gateway.SemanticCallEdge) (gateway.SemanticCallEdge, bool) {
	for _, edge := range edges {
		if !edge.Successful {
			return edge, true
		}
	}
	return gateway.SemanticCallEdge{}, false
}

func containsAny(value string, candidates ...string) bool {
	for _, candidate := range candidates {
		if strings.Contains(value, candidate) {
			return true
		}
	}
	return false
}

func questionLike(value string) bool {
	if strings.Contains(value, "?") {
		return true
	}
	normalized := normalize(value)
	for _, prefix := range []string{"is ", "are ", "was ", "were ", "how ", "what ", "which ", "did ", "do ", "does ", "any "} {
		if strings.HasPrefix(normalized, prefix) {
			return true
		}
	}
	return false
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func unambiguousAssetIdentity(input string) string {
	for _, raw := range strings.Fields(input) {
		token := strings.Trim(raw, "\"'`()[]{}<>,.!?")
		if strings.EqualFold(token, "XLM") {
			return "XLM"
		}
		for _, separator := range []string{":", "-"} {
			parts := strings.SplitN(token, separator, 2)
			if len(parts) != 2 || !validAssetCode(parts[0]) {
				continue
			}
			issuer := prismsearch.Classify(parts[1])
			if issuer.Type == prismsearch.ClassAccount {
				return strings.ToUpper(parts[0]) + ":" + issuer.Value
			}
		}
	}
	return ""
}

func validAssetCode(value string) bool {
	value = strings.ToUpper(strings.TrimSpace(value))
	if len(value) == 0 || len(value) > 12 {
		return false
	}
	for _, char := range value {
		if (char < 'A' || char > 'Z') && (char < '0' || char > '9') {
			return false
		}
	}
	return true
}

func assetDisplayLabel(asset string) string {
	for _, separator := range []string{":", "-"} {
		if code, _, ok := strings.Cut(asset, separator); ok && validAssetCode(code) {
			return strings.ToUpper(code)
		}
	}
	if classified := prismsearch.Classify(asset); classified.Type == prismsearch.ClassContract {
		return short(classified.Value)
	}
	return asset
}

func durationForWindow(value string) time.Duration {
	switch value {
	case "24h":
		return 24 * time.Hour
	case "7d":
		return 7 * 24 * time.Hour
	case "30d":
		return 30 * 24 * time.Hour
	default:
		return time.Hour
	}
}

func matchesFailureSubject(transaction gateway.DecodedTransaction, topic, asset string) bool {
	if topic != "" {
		if transaction.Summary == nil || !strings.EqualFold(transaction.Summary.Type, topic) {
			return false
		}
	}
	if asset != "" {
		if transaction.Summary == nil {
			return false
		}
		if !summaryContainsAsset(*transaction.Summary, asset) {
			return false
		}
	}
	return true
}

func summaryContainsAsset(summary gateway.TxSummary, asset string) bool {
	asset = strings.TrimSpace(asset)
	if asset == "" {
		return true
	}
	for _, transfer := range []*gateway.TransferDetail{summary.Transfer, summary.Mint, summary.Burn} {
		if transfer != nil && strings.EqualFold(strings.TrimSpace(transfer.Asset), asset) {
			return true
		}
	}
	return summary.Swap != nil && (strings.EqualFold(strings.TrimSpace(summary.Swap.AssetIn), asset) || strings.EqualFold(strings.TrimSpace(summary.Swap.AssetOut), asset))
}

func recentFailureExploreHref(match Match) string {
	values := url.Values{"status": {"failed"}}
	if value := match.Slots["requested_time"]; value != "" && value != "1h" {
		values.Set("time", value)
	}
	if value := match.Slots["topic"]; value != "" {
		values.Set("topic", value)
		values.Set("fn", value)
	}
	if value := match.Slots["asset"]; value != "" {
		values.Set("asset", value)
	}
	return "/v2/explore?" + values.Encode()
}
