package handlers

import (
	"context"
	"fmt"
	"html"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/withObsrvr/prism/internal/gateway"
	pagesv2 "github.com/withObsrvr/prism/internal/templates/v2/pages"
	vmv2 "github.com/withObsrvr/prism/internal/templates/v2/viewmodel"
)

func (h *Handlers) ExploreV2(w http.ResponseWriter, r *http.Request) {
	network := networkFromRequest(r)
	filters := exploreFiltersFromRequest(r)
	data := exploreV2ShellData(network, filters)
	if h.useExplicitMockData(r) {
		data = mockExploreV2Data(network, filters)
		data.DemoData = true
	}
	data.LiveHref = exploreLiveURL(r)
	if err := pagesv2.Explore(data).Render(r.Context(), w); err != nil {
		h.Logger.Error("render explore v2", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func (h *Handlers) ExploreV2Header(w http.ResponseWriter, r *http.Request) {
	network := networkFromRequest(r)
	header := unavailableExploreHeader(network)
	if h.useExplicitMockData(r) {
		header = exploreHeader(network)
	} else if h.Gateway != nil {
		ctx, cancel := context.WithTimeout(r.Context(), 6*time.Second)
		defer cancel()
		if recent, err := h.Gateway.GetSilverRecentLedgers(ctx, network, 1); err == nil && recent != nil && len(recent.Ledgers) > 0 {
			ledger := recent.Ledgers[0]
			header.LedgerNumber = gateway.FormatNumber(ledger.LedgerSequence)
			if t, err := time.Parse(time.RFC3339, ledger.ClosedAt); err == nil {
				header.AgeLabel = gateway.FormatAge(t)
			}
			header.Status = "live"
		} else if err != nil {
			h.Logger.Warn("explore header ledger refresh failed", "network", network, "error", err)
		}
	}
	if err := pagesv2.ExploreHeartbeat(header, false).Render(r.Context(), w); err != nil {
		h.Logger.Error("render explore v2 header", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func (h *Handlers) ExploreV2Live(w http.ResponseWriter, r *http.Request) {
	network := networkFromRequest(r)
	filters := exploreFiltersFromRequest(r)
	data := unavailableExploreV2Data(network, filters, "Matching activity is temporarily unavailable.")
	if h.useExplicitMockData(r) {
		data = mockExploreV2Data(network, filters)
		data.DemoData = true
	} else if h.Gateway != nil {
		if live, err := h.buildExploreV2Data(r, network); err == nil {
			data = live
		} else {
			if h.Logger != nil {
				h.Logger.Warn("live explore v2 unavailable", "network", network, "error", err)
			}
			data = unavailableExploreV2Data(network, filters, "Matching activity is temporarily unavailable.")
		}
	}
	if err := pagesv2.ExploreMain(data).Render(r.Context(), w); err != nil {
		h.Logger.Error("render explore v2 live", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func (h *Handlers) buildExploreV2Data(r *http.Request, network string) (vmv2.ExploreData, error) {
	filters := exploreFiltersFromRequest(r)
	params, validationMessage := explorerParamsForFilters(filters, r.URL.Query().Get("cursor"), time.Now().UTC())
	if validationMessage != "" {
		return invalidExploreV2Data(network, filters, validationMessage), nil
	}
	resp, err := h.Gateway.GetExplorerEvents(r.Context(), network, params)
	if err != nil {
		return vmv2.ExploreData{}, fmt.Errorf("fetching explorer events: %w", err)
	}

	rows := make([]vmv2.ExploreRow, 0, len(resp.Events))
	for _, event := range resp.Events {
		rows = append(rows, exploreRowFromGateway(event))
	}

	header := unavailableExploreHeader(network)
	if resp.Coverage != nil && resp.Coverage.CompleteThru > 0 {
		header.LedgerNumber = gateway.FormatNumber(resp.Coverage.CompleteThru)
		header.Status = resp.Status
		if header.Status == "ready" || header.Status == "empty" {
			header.Status = "live"
		}
		header.AgeLabel = "serving coverage"
		if resp.Coverage.UpdatedAt != nil {
			if updatedAt, parseErr := time.Parse(time.RFC3339, *resp.Coverage.UpdatedAt); parseErr == nil {
				header.AgeLabel = gateway.FormatAge(updatedAt)
			}
		}
	}
	matched := int64(resp.Meta.MatchedCount)
	summary := exploreSummary(filters, matched, resp.Meta.LedgerRange.Min, resp.Meta.LedgerRange.Max, resp.Meta.EventsPerSecond, true)
	summary.CountCapped = resp.Meta.CountCapped
	summary.MatchedLabel = exploreMatchedLabel(matched, resp.Meta.CountCapped)
	summary.LedgerRange = exploreCoverageLabel(resp.Coverage)
	summary.EvidenceLabel = exploreEvidenceLabel(resp)
	data := vmv2.ExploreData{
		Header:     header,
		Filters:    filters,
		Summary:    summary,
		Presets:    explorePresets(),
		Rows:       rows,
		State:      resp.Status,
		Warnings:   append([]string(nil), resp.Warnings...),
		SourceLive: resp.Status != "unavailable",
		HasMore:    resp.HasMore,
	}
	if resp.Status == "unavailable" {
		data.ErrorMessage = exploreFirstNonEmpty(firstExploreWarning(resp.Warnings), "The serving event projection is unavailable.")
		data.HasMore = false
	}
	if resp.NextCursor != nil {
		data.NextCursor = *resp.NextCursor
		data.NextHref = exploreMoreURL(r, *resp.NextCursor)
	}
	return data, nil
}

func exploreFiltersFromRequest(r *http.Request) vmv2.ExploreFilters {
	q := r.URL.Query()
	startLedger := strings.TrimSpace(q.Get("start_ledger"))
	endLedger := strings.TrimSpace(q.Get("end_ledger"))
	timeFilter := cleanOneOf(q.Get("time"), "1h", "24h", "7d", "30d", "coverage")
	if strings.TrimSpace(q.Get("time")) == "" && (startLedger != "" || endLedger != "") {
		timeFilter = "coverage"
	}
	filters := vmv2.ExploreFilters{
		Scope:       cleanOneOf(q.Get("scope"), "soroban", "classic"),
		Topic:       cleanOneOf(q.Get("topic"), "", "transfer", "swap", "mint", "burn", "approve", "contract_call"),
		Fn:          strings.TrimSpace(q.Get("fn")),
		Asset:       strings.TrimSpace(q.Get("asset")),
		Actor:       strings.ToUpper(strings.TrimSpace(q.Get("actor"))),
		Time:        timeFilter,
		Status:      cleanOneOf(q.Get("status"), "", "success", "failed"),
		StartLedger: startLedger,
		EndLedger:   endLedger,
		Q:           strings.TrimSpace(q.Get("q")),
	}
	if filters.Scope == "" {
		filters.Scope = "soroban"
	}
	if filters.Time == "" {
		filters.Time = "1h"
	}
	return filters
}

func cleanOneOf(value string, allowed ...string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	for _, a := range allowed {
		if value == a {
			return value
		}
	}
	if len(allowed) > 0 {
		return allowed[0]
	}
	return ""
}

func explorerParamsForFilters(filters vmv2.ExploreFilters, cursor string, now time.Time) (gateway.ExplorerEventsParams, string) {
	params := gateway.ExplorerEventsParams{Cursor: strings.TrimSpace(cursor), Limit: 48, Order: "desc"}
	if filters.Scope == "classic" {
		return params, "Classic operations are not part of the serving contract-event index yet. Use the Soroban event scope or inspect a classic transaction directly."
	}
	if filters.Topic != "" {
		params.Types = []string{filters.Topic}
	}
	params.Function = filters.Fn
	if filters.Asset != "" {
		normalized, ok := normalizeExploreAssetFilter(filters.Asset)
		if !ok {
			return params, "Choose one exact asset: XLM, CODE:G… with its issuer, or a C… token contract. A bare code can refer to multiple issuers."
		}
		params.Asset = normalized
	}
	if filters.Actor != "" {
		if !looksLikeExploreActor(filters.Actor) {
			return params, "Actor must be a complete G…, C…, or M… address."
		}
		params.Actor = filters.Actor
	}
	if filters.Status != "" {
		successful := filters.Status == "success"
		params.Successful = &successful
	}
	startLedger, message := parseExploreLedger(filters.StartLedger, "From ledger")
	if message != "" {
		return params, message
	}
	endLedger, message := parseExploreLedger(filters.EndLedger, "Through ledger")
	if message != "" {
		return params, message
	}
	if startLedger > 0 && endLedger > 0 && startLedger > endLedger {
		return params, "From ledger must be less than or equal to through ledger."
	}
	params.StartLedger = startLedger
	params.EndLedger = endLedger
	window := map[string]time.Duration{"1h": time.Hour, "24h": 24 * time.Hour, "7d": 7 * 24 * time.Hour, "30d": 30 * 24 * time.Hour}[filters.Time]
	if window > 0 {
		params.StartTime = now.UTC().Truncate(time.Minute).Add(-window)
	}

	query := strings.TrimSpace(filters.Q)
	switch {
	case txHashIfLikely(query) != "":
		params.TxHash = strings.ToLower(query)
	case contractIDIfLikely(query) != "":
		params.ContractID = strings.ToUpper(query)
	case looksLikeExploreActor(query):
		if params.Actor != "" && !strings.EqualFold(params.Actor, query) {
			return params, "Use either the actor field or an address in the search field, not two different actors."
		}
		params.Actor = strings.ToUpper(query)
	case query != "":
		params.ContractName = query
	}
	return params, ""
}

func exploreRowFromGateway(e gateway.ExplorerEvent) vmv2.ExploreRow {
	closed, _ := time.Parse(time.RFC3339, e.ClosedAt)
	when := "recent"
	age := "just now"
	if !closed.IsZero() {
		when = closed.Format("15:04:05")
		age = gateway.FormatAge(closed)
	}
	topic := normalizeExploreTopic(exploreFirstNonEmpty(e.Type, derefString(e.Topic0)))
	protocol := exploreFirstNonEmpty(derefString(e.Protocol), derefString(e.ContractName), derefString(e.ContractSymbol), "Soroban")
	asset := strings.ToUpper(exploreFirstNonEmpty(derefString(e.AssetKey), derefString(e.ContractSymbol), extractAssetFromExplorerEvent(e)))
	fn := exploreFirstNonEmpty(derefString(e.FunctionName), derefString(e.Topic0), e.Type)
	status := "Success"
	statusTone := "success"
	if !e.PublicSuccessful() {
		status = "Failed"
		statusTone = "failed"
	}
	headline := buildExploreHeadline(topic, protocol, asset, e)
	from := exploreFirstNonEmpty(derefString(e.FromAddress), explorerActorForRole(e.Actors, "from", "sender", "source"), derefString(e.Topic1))
	return vmv2.ExploreRow{
		When:         when,
		Age:          age,
		Scope:        "soroban",
		Protocol:     protocol,
		Topic:        topic,
		Function:     fn,
		Headline:     headline,
		From:         from,
		Contract:     derefString(e.ContractID),
		TxHash:       e.TransactionHash,
		Ledger:       gateway.FormatNumber(e.LedgerSequence),
		Events:       "1",
		Status:       status,
		StatusTone:   statusTone,
		Asset:        asset,
		EvidenceHref: "/v2/tx/" + e.TransactionHash,
		ContractHref: "/v2/contract/" + derefString(e.ContractID),
		LedgerHref:   "/v2/ledger/" + strconv.FormatInt(e.LedgerSequence, 10),
	}
}

func buildExploreHeadline(topic string, protocol string, asset string, e gateway.ExplorerEvent) string {
	amount := extractAmountFromText(exploreFirstNonEmpty(derefString(e.DataDecoded), derefString(e.Data)))
	from := gateway.ShortAddress(exploreFirstNonEmpty(derefString(e.FromAddress), explorerActorForRole(e.Actors, "from", "sender", "source"), derefString(e.Topic1)))
	to := gateway.ShortAddress(exploreFirstNonEmpty(derefString(e.ToAddress), explorerActorForRole(e.Actors, "to", "recipient", "destination"), derefString(e.Topic2)))
	subject := "A contract"
	if from != "" {
		subject = from
	}
	switch topic {
	case "swap":
		return subject + " swapped through " + protocol
	case "transfer":
		if amount != "" && asset != "" {
			return subject + " transferred " + amount + " " + asset + " to " + exploreFirstNonEmpty(to, "another account")
		}
		return subject + " transferred value to " + exploreFirstNonEmpty(to, "another account")
	case "mint":
		return protocol + " minted " + exploreFirstNonEmpty(asset, "a token")
	case "burn":
		return subject + " burned " + exploreFirstNonEmpty(asset, "a token")
	case "approve":
		return subject + " approved a spending allowance"
	default:
		return protocol + " emitted a " + exploreFirstNonEmpty(derefString(e.Topic0), e.Type, "contract") + " event"
	}
}

func normalizeExploreTopic(v string) string {
	v = strings.ToLower(v)
	switch {
	case strings.Contains(v, "swap"):
		return "swap"
	case strings.Contains(v, "transfer") || strings.Contains(v, "payment"):
		return "transfer"
	case strings.Contains(v, "mint"):
		return "mint"
	case strings.Contains(v, "burn"):
		return "burn"
	case strings.Contains(v, "approve") || strings.Contains(v, "allowance"):
		return "approve"
	case strings.Contains(v, "deposit") || strings.Contains(v, "supply"):
		return "deposit"
	case strings.Contains(v, "withdraw") || strings.Contains(v, "borrow"):
		return "withdraw"
	default:
		return "custom"
	}
}

func exploreRowMatches(row vmv2.ExploreRow, f vmv2.ExploreFilters) bool {
	if f.Scope == "classic" && row.Scope != "classic" {
		return false
	}
	if f.Scope == "soroban" && row.Scope != "soroban" {
		return false
	}
	if f.Topic != "" && row.Topic != f.Topic {
		return false
	}
	if f.Fn != "" && !strings.Contains(strings.ToLower(row.Function), strings.ToLower(f.Fn)) {
		return false
	}
	if f.Asset != "" && !strings.Contains(strings.ToUpper(row.Asset+" "+row.Headline), strings.ToUpper(f.Asset)) {
		return false
	}
	if f.Status == "success" && row.StatusTone != "success" {
		return false
	}
	if f.Status == "failed" && row.StatusTone != "failed" {
		return false
	}
	if f.Q != "" {
		haystack := strings.ToLower(row.Protocol + " " + row.Topic + " " + row.Function + " " + row.Headline + " " + row.TxHash + " " + row.Contract)
		if !strings.Contains(haystack, strings.ToLower(f.Q)) {
			return false
		}
	}
	return true
}

func exploreV2ShellData(network string, filters vmv2.ExploreFilters) vmv2.ExploreData {
	summary := exploreSummary(filters, 0, 0, 0, nil, true)
	summary.MatchedLabel = ""
	summary.LedgerRange = "Fetching live ledger range"
	summary.EvidenceLabel = "Loading live gateway data"
	return vmv2.ExploreData{
		Header:  vmv2.ExploreHeaderData{Network: network, LedgerNumber: "Unavailable", AgeLabel: "Waiting for evidence", Status: "loading"},
		Filters: filters,
		Summary: summary,
		Presets: explorePresets(),
		State:   "loading",
		Loading: true,
	}
}

func unavailableExploreV2Data(network string, filters vmv2.ExploreFilters, message string) vmv2.ExploreData {
	data := exploreV2ShellData(network, filters)
	data.Loading = false
	data.State = "unavailable"
	data.Rows = nil
	data.ErrorMessage = message
	data.Header = unavailableExploreHeader(network)
	data.Summary.MatchedLabel = ""
	data.Summary.LedgerRange = "Ledger range unavailable"
	data.Summary.EvidenceLabel = "Live evidence unavailable"
	return data
}

func invalidExploreV2Data(network string, filters vmv2.ExploreFilters, message string) vmv2.ExploreData {
	data := exploreV2ShellData(network, filters)
	data.Loading = false
	data.State = "invalid"
	data.ErrorMessage = message
	data.Header = unavailableExploreHeader(network)
	data.Summary.MatchedLabel = ""
	data.Summary.LedgerRange = "Query not sent"
	data.Summary.EvidenceLabel = "No evidence claimed"
	return data
}

func mockExploreV2Data(network string, filters vmv2.ExploreFilters) vmv2.ExploreData {
	rows := []vmv2.ExploreRow{
		{When: "14:32:04", Age: "4s ago", Scope: "soroban", Protocol: "Soroswap", Topic: "swap", Function: "swap", Headline: "Alice swapped 100 USDC for 412.04 XLM on Soroswap", From: "GABC7F9A", Contract: "CCRT9981", TxHash: "abc49f4e2", Ledger: "52,844,201", Events: "4", Status: "Success", StatusTone: "success", Asset: "USDC", EvidenceHref: "/v2/tx/abc49f4e2"},
		{When: "14:31:58", Age: "10s ago", Scope: "soroban", Protocol: "Blend", Topic: "deposit", Function: "supply", Headline: "An institutional address supplied 50,000 USDC as collateral on Blend", From: "GLN177HH", Contract: "CBLDAA21", TxHash: "f2b90ced4", Ledger: "52,844,200", Events: "2", Status: "Success", StatusTone: "success", Asset: "USDC", EvidenceHref: "/v2/tx/f2b90ced4"},
		{When: "14:31:52", Age: "16s ago", Scope: "classic", Protocol: "Classic", Topic: "transfer", Function: "payment", Headline: "Treasury wallet sent 8,200 EURC with memo INV-204", From: "GAK78D2A", TxHash: "44e1aa092", Ledger: "52,844,198", Events: "1", Status: "Success", StatusTone: "success", Asset: "EURC", EvidenceHref: "/v2/tx/44e1aa092"},
		{When: "14:31:21", Age: "47s ago", Scope: "soroban", Protocol: "SAC", Topic: "approve", Function: "approve", Headline: "GAB4 approved Soroswap router to spend 1,000 USDC", From: "GAB44F9A", Contract: "CCQHSWAP", TxHash: "ee207a0b1", Ledger: "52,844,193", Events: "1", Status: "Success", StatusTone: "success", Asset: "USDC", EvidenceHref: "/v2/tx/ee207a0b1"},
		{When: "14:31:08", Age: "60s ago", Scope: "soroban", Protocol: "Soroswap", Topic: "swap", Function: "swap", Headline: "GZK4 swap reverted because slippage exceeded their cap", From: "GZK40099", Contract: "CCRT9981", TxHash: "fail22ee", Ledger: "52,844,190", Events: "1", Status: "Failed", StatusTone: "failed", Asset: "USDC", EvidenceHref: "/v2/tx/fail22ee"},
	}
	filtered := make([]vmv2.ExploreRow, 0, len(rows))
	for _, row := range rows {
		if exploreRowMatches(row, filters) {
			filtered = append(filtered, row)
		}
	}
	return vmv2.ExploreData{Header: exploreHeader(network), Filters: filters, Summary: exploreSummary(filters, int64(len(filtered)), 52844190, 52844201, nil, false), Presets: explorePresets(), Rows: filtered, State: "demo", SourceLive: false}
}

func exploreHeader(network string) vmv2.ExploreHeaderData {
	return vmv2.ExploreHeaderData{Network: network, LedgerNumber: "52,844,201", AgeLabel: "Synthetic fixture", Status: "demo"}
}

func unavailableExploreHeader(network string) vmv2.ExploreHeaderData {
	return vmv2.ExploreHeaderData{Network: network, LedgerNumber: "Unavailable", AgeLabel: "Waiting for evidence", Status: "unavailable"}
}

func exploreSummary(f vmv2.ExploreFilters, count int64, minLedger int64, maxLedger int64, eps *float64, live bool) vmv2.ExploreSummary {
	scope := map[string]string{"soroban": "Soroban contract events", "classic": "classic activity"}[f.Scope]
	if scope == "" {
		scope = "Soroban contract events"
	}
	parts := []string{"Show", "<b class=\"scope\">" + html.EscapeString(scope) + "</b>"}
	if f.Q != "" {
		parts = append(parts, "matching "+exploreQueryHTML(f.Q))
	}
	if f.Topic != "" {
		parts = append(parts, "about <b class=\"topic\">"+html.EscapeString(f.Topic)+"</b>")
	}
	if f.Fn != "" {
		parts = append(parts, "calling <b class=\"fn\">"+html.EscapeString(f.Fn)+"</b>")
	}
	if f.Asset != "" {
		parts = append(parts, "touching <b class=\"asset\">"+html.EscapeString(f.Asset)+"</b>")
	}
	if f.Actor != "" {
		parts = append(parts, "involving <b class=\"actor\"><code>"+html.EscapeString(gateway.ShortAddress(f.Actor))+"</code></b>")
	}
	if f.Status == "failed" {
		parts = append(parts, "that <b class=\"status\">failed</b>")
	} else if f.Status == "success" {
		parts = append(parts, "that <b class=\"status\">succeeded</b>")
	}
	parts = append(parts, "in the <b class=\"time\">"+html.EscapeString(timeWindowLabel(f.Time))+"</b>.")
	summary := vmv2.ExploreSummary{SentenceHTML: strings.Join(parts, " "), MatchedLabel: gateway.FormatNumber(count), WindowLabel: timeWindowLabel(f.Time), LedgerRange: "Ledgers " + gateway.FormatNumber(minLedger) + " to " + gateway.FormatNumber(maxLedger)}
	if live {
		summary.EvidenceLabel = "Live gateway data"
	} else {
		summary.EvidenceLabel = "Demo fixture"
	}
	if eps != nil {
		summary.EventsPerSec = fmt.Sprintf("%.1f", *eps)
	}
	return summary
}

func exploreQueryHTML(q string) string {
	q = strings.TrimSpace(q)
	label := "“" + q + "”"
	kind := "query"
	switch {
	case contractIDIfLikely(q) != "":
		label = "contract <code>" + html.EscapeString(gateway.ShortAddress(q)) + "</code>"
		kind = "contract"
	case txHashIfLikely(q) != "":
		label = "transaction <code>" + html.EscapeString(gateway.ShortHash(q)) + "</code>"
		kind = "tx"
	case strings.HasPrefix(strings.ToUpper(q), "G") && len(q) == 56:
		label = "address <code>" + html.EscapeString(gateway.ShortAddress(q)) + "</code>"
		kind = "address"
	}
	return "<b class=\"" + kind + "\">" + label + "</b>"
}

func timeWindowLabel(v string) string {
	switch v {
	case "coverage":
		return "retained serving coverage"
	case "24h":
		return "last 24 hours"
	case "7d":
		return "last 7 days"
	case "30d":
		return "last 30 days"
	default:
		return "last hour"
	}
}

func explorePresets() []vmv2.ExplorePreset {
	return []vmv2.ExplorePreset{
		{Name: "Soroswap swaps", Body: "Swap events and router calls in the last 24 hours.", Href: "/v2/explore?scope=soroban&topic=swap&q=Soroswap&time=24h"},
		{Name: "XLM movement", Body: "Contract events that reference the native asset.", Href: "/v2/explore?asset=XLM&topic=transfer&time=1h"},
		{Name: "Blend lending", Body: "Supply, borrow, repay, and withdraw activity.", Href: "/v2/explore?scope=soroban&q=Blend&time=24h"},
		{Name: "Failed calls", Body: "Recent reverted Soroban activity with evidence links.", Href: "/v2/explore?scope=soroban&status=failed&time=1h"},
	}
}

func exploreMatchedLabel(count int64, capped bool) string {
	label := gateway.FormatNumber(count)
	if capped {
		return label + "+"
	}
	return label
}

func exploreCoverageLabel(coverage *gateway.ServingCoverageMetadata) string {
	if coverage == nil || coverage.CompleteThru <= 0 {
		return "Serving coverage unavailable"
	}
	return "Coverage ledgers " + gateway.FormatNumber(coverage.CompleteFrom) + " to " + gateway.FormatNumber(coverage.CompleteThru)
}

func exploreEvidenceLabel(response *gateway.ExplorerEventsResponse) string {
	if response == nil {
		return "Evidence unavailable"
	}
	switch response.Status {
	case "partial":
		return "Serving-only, partial coverage"
	case "unavailable":
		return "Serving evidence unavailable"
	default:
		return "Serving-only evidence"
	}
}

func firstExploreWarning(warnings []string) string {
	for _, warning := range warnings {
		if warning = strings.TrimSpace(warning); warning != "" {
			return warning
		}
	}
	return ""
}

func parseExploreLedger(value string, label string) (int64, string) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, ""
	}
	ledger, err := strconv.ParseInt(value, 10, 64)
	if err != nil || ledger < 1 {
		return 0, label + " must be a positive ledger number."
	}
	return ledger, ""
}

func normalizeExploreAssetFilter(value string) (string, bool) {
	value = strings.TrimSpace(value)
	if strings.EqualFold(value, "XLM") || strings.EqualFold(value, "native") {
		return "XLM", true
	}
	upper := strings.ToUpper(value)
	if strings.HasPrefix(upper, "C") && looksLikeExploreActor(upper) {
		return upper, true
	}
	parts := strings.Split(upper, ":")
	if len(parts) != 2 || len(parts[0]) < 1 || len(parts[0]) > 12 || !strings.HasPrefix(parts[1], "G") || !looksLikeExploreActor(parts[1]) {
		return "", false
	}
	for _, char := range parts[0] {
		if !(char >= 'A' && char <= 'Z') && !(char >= '0' && char <= '9') {
			return "", false
		}
	}
	return upper, true
}

func looksLikeExploreActor(value string) bool {
	value = strings.ToUpper(strings.TrimSpace(value))
	validLength := (len(value) == 56 && (strings.HasPrefix(value, "G") || strings.HasPrefix(value, "C"))) || (len(value) == 69 && strings.HasPrefix(value, "M"))
	if !validLength {
		return false
	}
	for _, char := range value {
		if !(char >= 'A' && char <= 'Z') && !(char >= '2' && char <= '7') {
			return false
		}
	}
	return true
}

func explorerActorForRole(actors []gateway.ExplorerEventActor, roles ...string) string {
	for _, role := range roles {
		for _, actor := range actors {
			if strings.EqualFold(strings.TrimSpace(actor.Role), role) && strings.TrimSpace(actor.Address) != "" {
				return strings.TrimSpace(actor.Address)
			}
		}
	}
	return ""
}

func derefString(v *string) string {
	if v == nil {
		return ""
	}
	return strings.TrimSpace(*v)
}

func exploreFirstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func txHashIfLikely(q string) string {
	q = strings.TrimSpace(q)
	if len(q) == 64 && isHex(q) {
		return q
	}
	return ""
}

func contractIDIfLikely(q string) string {
	q = strings.TrimSpace(q)
	if len(q) == 56 && strings.HasPrefix(strings.ToUpper(q), "C") && !strings.Contains(q, " ") {
		return q
	}
	return ""
}

func contractNameIfLikely(q string) string {
	q = strings.TrimSpace(q)
	if q == "" || txHashIfLikely(q) != "" || contractIDIfLikely(q) != "" {
		return ""
	}
	return q
}

func extractAssetFromExplorerEvent(e gateway.ExplorerEvent) string {
	text := strings.ToUpper(exploreFirstNonEmpty(derefString(e.TopicsDecoded), derefString(e.DataDecoded)))
	for _, asset := range []string{"USDC", "XLM", "EURC", "AQUA", "BLND", "YUSDC", "YXLM"} {
		if strings.Contains(text, asset) {
			return asset
		}
	}
	return ""
}

func extractAmountFromText(text string) string {
	fields := strings.Fields(strings.ReplaceAll(strings.ReplaceAll(text, ",", ""), "\"", ""))
	for _, f := range fields {
		if _, err := strconv.ParseFloat(f, 64); err == nil {
			return f
		}
	}
	return ""
}

func exploreLiveURL(r *http.Request) string {
	q := url.Values{}
	for key, values := range r.URL.Query() {
		for _, value := range values {
			q.Add(key, value)
		}
	}
	if q.Encode() == "" {
		return "/v2/explore/live"
	}
	return "/v2/explore/live?" + q.Encode()
}

func exploreMoreURL(r *http.Request, cursor string) string {
	q := url.Values{}
	for key, values := range r.URL.Query() {
		for _, value := range values {
			q.Add(key, value)
		}
	}
	q.Set("cursor", cursor)
	return "/v2/explore?" + q.Encode()
}
