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
	data := mockExploreV2Data(network, filters)
	if h.useLiveData(r) {
		data = exploreV2ShellData(network, filters)
	}
	data.LiveHref = exploreLiveURL(r)
	if err := pagesv2.Explore(data).Render(r.Context(), w); err != nil {
		h.Logger.Error("render explore v2", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func (h *Handlers) ExploreV2Header(w http.ResponseWriter, r *http.Request) {
	network := networkFromRequest(r)
	header := exploreHeader(network)
	if h.useLiveData(r) {
		ctx, cancel := context.WithTimeout(r.Context(), 1500*time.Millisecond)
		defer cancel()
		if recent, err := h.Gateway.GetSilverRecentLedgers(ctx, network, 1); err == nil && recent != nil && len(recent.Ledgers) > 0 {
			ledger := recent.Ledgers[0]
			header.LedgerNumber = gateway.FormatNumber(ledger.LedgerSequence)
			if t, err := time.Parse(time.RFC3339, ledger.ClosedAt); err == nil {
				header.AgeLabel = gateway.FormatAge(t)
			}
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
	data := mockExploreV2Data(network, exploreFiltersFromRequest(r))
	if h.useLiveData(r) {
		if live, err := h.buildExploreV2Data(r, network); err == nil {
			data = live
		} else {
			h.Logger.Warn("live explore v2 failed, falling back to mock", "network", network, "error", err)
			data.ErrorMessage = "Showing a design fallback while the gateway connection recovers."
		}
	}
	if err := pagesv2.ExploreMain(data).Render(r.Context(), w); err != nil {
		h.Logger.Error("render explore v2 live", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func (h *Handlers) buildExploreV2Data(r *http.Request, network string) (vmv2.ExploreData, error) {
	filters := exploreFiltersFromRequest(r)
	params := gateway.ExplorerEventsParams{
		Tab:          exploreTabForFilters(filters),
		TopicMatch:   exploreFirstNonEmpty(filters.Fn, filters.Asset, filters.Q),
		TxHash:       txHashIfLikely(filters.Q),
		ContractID:   contractIDIfLikely(filters.Q),
		ContractName: contractNameIfLikely(filters.Q),
		Cursor:       r.URL.Query().Get("cursor"),
		Limit:        48,
		Order:        "desc",
	}
	if params.TxHash != "" || params.ContractID != "" {
		params.TopicMatch = ""
		params.ContractName = ""
	}
	resp, err := h.Gateway.GetExplorerEvents(r.Context(), network, params)
	if err != nil {
		return vmv2.ExploreData{}, fmt.Errorf("fetching explorer events: %w", err)
	}

	rows := make([]vmv2.ExploreRow, 0, len(resp.Events))
	for _, event := range resp.Events {
		row := exploreRowFromGateway(event)
		if !exploreRowMatches(row, filters) {
			continue
		}
		rows = append(rows, row)
	}
	if params.ContractID != "" && len(rows) == 0 {
		rows = h.exploreRowsForTokenContract(r, network, params.ContractID, filters)
	}

	header := exploreHeader(network)
	if resp.Meta.LedgerRange.Max > 0 {
		header.LedgerNumber = gateway.FormatNumber(resp.Meta.LedgerRange.Max)
	}
	matched := int64(resp.Meta.MatchedCount)
	if params.ContractID != "" && matched == 0 && len(rows) > 0 {
		matched = int64(len(rows))
	}
	data := vmv2.ExploreData{
		Header:     header,
		Filters:    filters,
		Summary:    exploreSummary(filters, matched, resp.Meta.LedgerRange.Min, resp.Meta.LedgerRange.Max, resp.Meta.EventsPerSecond, true),
		Presets:    explorePresets(),
		Rows:       rows,
		SourceLive: true,
		HasMore:    resp.HasMore,
	}
	if resp.NextCursor != nil {
		data.NextCursor = *resp.NextCursor
		data.NextHref = exploreMoreURL(r, *resp.NextCursor)
	}
	return data, nil
}

func exploreFiltersFromRequest(r *http.Request) vmv2.ExploreFilters {
	q := r.URL.Query()
	filters := vmv2.ExploreFilters{
		Scope:  cleanOneOf(q.Get("scope"), "all", "soroban", "classic"),
		Topic:  cleanOneOf(q.Get("topic"), "", "transfer", "swap", "mint", "burn", "approve", "deposit", "withdraw", "custom"),
		Fn:     strings.TrimSpace(q.Get("fn")),
		Asset:  strings.ToUpper(strings.TrimSpace(q.Get("asset"))),
		Time:   cleanOneOf(q.Get("time"), "1h", "24h", "7d", "30d"),
		Status: cleanOneOf(q.Get("status"), "", "success", "failed"),
		Q:      strings.TrimSpace(q.Get("q")),
	}
	if filters.Scope == "" {
		filters.Scope = "all"
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

func exploreTabForFilters(f vmv2.ExploreFilters) string {
	switch f.Topic {
	case "transfer":
		return "transfers"
	case "swap":
		return "swaps"
	case "mint", "burn":
		return "mints_burns"
	case "approve", "deposit", "withdraw", "custom":
		return "contract_calls"
	default:
		if f.Scope == "classic" {
			return "transfers"
		}
		return ""
	}
}

func (h *Handlers) exploreRowsForTokenContract(r *http.Request, network string, contractID string, filters vmv2.ExploreFilters) []vmv2.ExploreRow {
	transfers, err := h.Gateway.GetTokenTransfers(r.Context(), network, contractID, 48)
	if err == nil && transfers != nil && len(transfers.Transfers) > 0 {
		rows := make([]vmv2.ExploreRow, 0, len(transfers.Transfers))
		for _, transfer := range transfers.Transfers {
			row := exploreRowFromTokenTransfer(transfer, contractID)
			if !exploreRowMatches(row, filters) {
				continue
			}
			rows = append(rows, row)
		}
		return rows
	}
	if err != nil {
		h.Logger.Warn("token transfer fallback failed for explore contract search", "contract", contractID, "network", network, "error", err)
	}
	generic, err := h.Gateway.GetGenericEvents(r.Context(), network, contractID, "", 48)
	if err != nil || generic == nil {
		if err != nil {
			h.Logger.Warn("generic event fallback failed for explore contract search", "contract", contractID, "network", network, "error", err)
		}
		return nil
	}
	rows := make([]vmv2.ExploreRow, 0, len(generic.Events))
	for _, event := range generic.Events {
		row := exploreRowFromGenericEvent(event)
		if !exploreRowMatches(row, filters) {
			continue
		}
		rows = append(rows, row)
	}
	return rows
}

func exploreRowFromTokenTransfer(t gateway.TokenTransfer, contractID string) vmv2.ExploreRow {
	closed, _ := time.Parse(time.RFC3339, t.ClosedAt)
	when := "recent"
	age := "just now"
	if !closed.IsZero() {
		when = closed.Format("15:04:05")
		age = gateway.FormatAge(closed)
	}
	topic := normalizeExploreTopic(t.EventType)
	if topic == "custom" && t.EventType == "" {
		topic = "transfer"
	}
	amount := strings.TrimSpace(t.Amount)
	headline := gateway.ShortAddress(t.From) + " sent value to " + gateway.ShortAddress(t.To)
	if amount != "" {
		headline = gateway.ShortAddress(t.From) + " sent " + amount + " token units to " + gateway.ShortAddress(t.To)
	}
	if strings.EqualFold(t.EventType, "mint") {
		headline = "Token contract minted " + exploreFirstNonEmpty(amount, "value") + " to " + gateway.ShortAddress(t.To)
	}
	return vmv2.ExploreRow{
		When: when, Age: age, Scope: "soroban", Protocol: "Token contract", Topic: topic, Function: exploreFirstNonEmpty(t.EventType, "transfer"), Headline: headline,
		From: t.From, Contract: contractID, TxHash: t.TxHash, Ledger: gateway.FormatNumber(t.LedgerSequence), Events: "1", Status: "Success", StatusTone: "success", EvidenceHref: "/v2/tx/" + t.TxHash, ContractHref: "/v2/contract/" + contractID, LedgerHref: "/v2/ledger/" + strconv.FormatInt(t.LedgerSequence, 10),
	}
}

func exploreRowFromGenericEvent(e gateway.GenericEvent) vmv2.ExploreRow {
	closed, _ := time.Parse(time.RFC3339, e.ClosedAt)
	when := "recent"
	age := "just now"
	if !closed.IsZero() {
		when = closed.Format("15:04:05")
		age = gateway.FormatAge(closed)
	}
	topic := normalizeExploreTopic(e.EventType + " " + e.TopicsDecoded)
	return vmv2.ExploreRow{When: when, Age: age, Scope: "soroban", Protocol: "Contract event", Topic: topic, Function: exploreFirstNonEmpty(e.EventType, "event"), Headline: "Contract emitted " + exploreFirstNonEmpty(e.EventType, "an event"), Contract: e.ContractID, TxHash: e.TxHash, Ledger: gateway.FormatNumber(e.LedgerSequence), Events: "1", Status: "Success", StatusTone: "success", EvidenceHref: "/v2/tx/" + e.TxHash, ContractHref: "/v2/contract/" + e.ContractID, LedgerHref: "/v2/ledger/" + strconv.FormatInt(e.LedgerSequence, 10)}
}

func exploreRowFromGateway(e gateway.ExplorerEvent) vmv2.ExploreRow {
	closed, _ := time.Parse(time.RFC3339, e.ClosedAt)
	when := "recent"
	age := "just now"
	if !closed.IsZero() {
		when = closed.Format("15:04:05")
		age = gateway.FormatAge(closed)
	}
	topic := normalizeExploreTopic(exploreFirstNonEmpty(derefString(e.Topic0), e.Type))
	protocol := exploreFirstNonEmpty(derefString(e.Protocol), derefString(e.ContractName), derefString(e.ContractSymbol), "Soroban")
	asset := strings.ToUpper(exploreFirstNonEmpty(derefString(e.ContractSymbol), extractAssetFromExplorerEvent(e)))
	fn := exploreFirstNonEmpty(derefString(e.Topic0), e.Type)
	status := "Success"
	statusTone := "success"
	if !e.PublicSuccessful() {
		status = "Failed"
		statusTone = "failed"
	}
	headline := buildExploreHeadline(topic, protocol, asset, e)
	return vmv2.ExploreRow{
		When:         when,
		Age:          age,
		Scope:        "soroban",
		Protocol:     protocol,
		Topic:        topic,
		Function:     fn,
		Headline:     headline,
		From:         derefString(e.Topic1),
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
	from := gateway.ShortAddress(derefString(e.Topic1))
	to := gateway.ShortAddress(derefString(e.Topic2))
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
	data := mockExploreV2Data(network, filters)
	data.Rows = nil
	data.SourceLive = false
	data.Loading = true
	data.HasMore = false
	data.NextCursor = ""
	data.NextHref = ""
	data.Header.LedgerNumber = "syncing"
	data.Header.AgeLabel = "latest ledger"
	data.Summary.MatchedLabel = "…"
	data.Summary.LedgerRange = "Fetching live ledger range"
	data.Summary.EvidenceLabel = "Loading live gateway data"
	data.Summary.EventsPerSec = ""
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
	return vmv2.ExploreData{Header: exploreHeader(network), Filters: filters, Summary: exploreSummary(filters, int64(len(filtered)), 52844190, 52844201, nil, false), Presets: explorePresets(), Rows: filtered, SourceLive: false}
}

func exploreHeader(network string) vmv2.ExploreHeaderData {
	return vmv2.ExploreHeaderData{Network: network, LedgerNumber: "52,844,201", AgeLabel: "2 seconds ago"}
}

func exploreSummary(f vmv2.ExploreFilters, count int64, minLedger int64, maxLedger int64, eps *float64, live bool) vmv2.ExploreSummary {
	scope := map[string]string{"all": "all Stellar activity", "soroban": "Soroban activity", "classic": "classic activity"}[f.Scope]
	if scope == "" {
		scope = "all Stellar activity"
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
		summary.EvidenceLabel = "Fallback sample"
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
		{Name: "USDC movement", Body: "All visible USDC transfers, mints, burns, and approvals.", Href: "/v2/explore?scope=all&asset=USDC&time=1h"},
		{Name: "Blend lending", Body: "Supply, borrow, repay, and withdraw activity.", Href: "/v2/explore?scope=soroban&q=Blend&time=24h"},
		{Name: "Failed calls", Body: "Recent reverted Soroban activity with evidence links.", Href: "/v2/explore?scope=soroban&status=failed&time=1h"},
	}
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
