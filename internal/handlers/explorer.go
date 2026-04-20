package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/withObsrvr/prism/internal/gateway"
	"github.com/withObsrvr/prism/internal/humanize"
	"github.com/withObsrvr/prism/internal/templates/pages"
)

// Home renders the search-first landing page shell.
// Data sections are loaded via htmx fragment endpoints (/fragments/home/*).
// The shell only needs minimal data for the search hint (latest ledger number).
func (h *Handlers) Home(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	network := networkFromRequest(r)
	data := mockHomeData(network)

	// Overlay live latest ledger for the search hint display.
	if h.Gateway != nil {
		ctx := r.Context()
		if bronze, err := h.Gateway.GetBronzeNetworkStats(ctx, network); err == nil {
			data.LatestLedger = gateway.FormatNumber(bronze.Ledger.LatestSequence)
			if t, err := time.Parse(time.RFC3339, bronze.Ledger.ClosedAt); err == nil {
				data.LedgerAge = gateway.FormatAge(t)
			}
		} else if stats, err := h.Gateway.GetNetworkStats(ctx, network); err == nil {
			data.LatestLedger = gateway.FormatNumber(stats.Ledger.CurrentSequence)
			if t, err := time.Parse(time.RFC3339, stats.GeneratedAt); err == nil {
				data.LedgerAge = gateway.FormatAge(t)
			}
		}
	}

	if err := pages.Home(data).Render(r.Context(), w); err != nil {
		h.Logger.Error("render home", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// buildHomeData fetches real data from the gateway and transforms it into HomeData.
func (h *Handlers) buildHomeData(r *http.Request, network string) (pages.HomeData, error) {
	ctx := r.Context()

	// Fetch silver network stats (accounts, fees, soroban details).
	stats, err := h.Gateway.GetNetworkStats(ctx, network)
	if err != nil {
		return pages.HomeData{}, fmt.Errorf("fetching network stats: %w", err)
	}

	// Try bronze stats for accurate latest ledger and 24h tx counts.
	bronze, bronzeErr := h.Gateway.GetBronzeNetworkStats(ctx, network)
	if bronzeErr != nil {
		h.Logger.Debug("bronze stats unavailable, using silver", "error", bronzeErr)
	}

	// Prefer bronze for latest sequence; fall back to silver.
	latestSeq := stats.Ledger.CurrentSequence
	if bronze != nil {
		latestSeq = bronze.Ledger.LatestSequence
	}

	// Compute ledger range for recent ledgers (latest 8).
	startSeq := latestSeq - 7
	if startSeq < 1 {
		startSeq = 1
	}

	// Fetch recent ledgers and top contracts.
	ledgers, ledgerErr := h.Gateway.GetLedgers(ctx, network, startSeq, latestSeq, 8, "desc")
	contracts, contractErr := h.Gateway.GetTopContracts(ctx, network, 5)

	// Use bronze tx/soroban counts when available; prefer silver tx counts over ops as fallback.
	txCount24H := stats.Operations24H.Total
	if stats.Transactions24H.Total > 0 {
		txCount24H = stats.Transactions24H.Total
	}
	sorobanCalls := stats.Operations24H.ContractInvoke
	if bronze != nil {
		if bronze.Transactions24H.Total > 0 {
			txCount24H = bronze.Transactions24H.Total
		}
		sorobanCalls = bronze.Transactions24H.SorobanCount
	}

	// Build the HomeData from real data.
	data := pages.HomeData{
		Network:      network,
		LatestLedger: gateway.FormatNumber(latestSeq),
		TxCount24H:   gateway.FormatNumber(txCount24H),
		TPSAvg:       fmt.Sprintf("%.1f", float64(txCount24H)/86400),
		SorobanCalls: gateway.FormatNumber(sorobanCalls),
		FeeEconomy:   "100",
		FeeStandard:  "1,200",
		FeePriority:  "34,000",
		SurgeActive:  false,
		SurgeContext: "Network is uncongested",
		Validators:   35, // Not available from gateway — static for now
	}

	// Ledger age: prefer bronze closed_at (actual ledger close), fall back to silver generated_at.
	if bronze != nil {
		if closeTime, err := time.Parse(time.RFC3339, bronze.Ledger.ClosedAt); err == nil {
			data.LedgerAge = gateway.FormatAge(closeTime)
		} else {
			data.LedgerAge = "just now"
		}
	} else if genTime, err := time.Parse(time.RFC3339, stats.GeneratedAt); err == nil {
		data.LedgerAge = gateway.FormatAge(genTime)
	} else {
		data.LedgerAge = "just now"
	}

	// TPS peak — estimate from avg close time.
	avgClose := stats.Ledger.AvgCloseTimeSeconds
	if bronze != nil && bronze.Ledger.AvgCloseTimeSeconds > 0 {
		avgClose = bronze.Ledger.AvgCloseTimeSeconds
	}
	if avgClose > 0 {
		data.TPSPeak = fmt.Sprintf("%.0f", float64(txCount24H)/86400*3)
	} else {
		data.TPSPeak = "—"
	}

	// 24h change — not available, leave blank.
	data.TxChange = ""
	data.SorobanChange = ""

	// Transform ledgers.
	if ledgerErr == nil && len(ledgers) > 0 {
		homeLedgers := make([]pages.HomeLedger, 0, len(ledgers))
		for i, l := range ledgers {
			age := "—"
			if t, err := time.Parse(time.RFC3339, l.ClosedAt); err == nil {
				d := time.Since(t)
				age = fmt.Sprintf("%.1fs", d.Seconds())
			}
			homeLedgers = append(homeLedgers, pages.HomeLedger{
				Sequence:    gateway.FormatNumber(l.Sequence),
				SequenceRaw: fmt.Sprintf("%d", l.Sequence),
				Age:         age,
				TxCount:     fmt.Sprintf("%d txs", l.SuccessfulTxCount),
				OpCount:     fmt.Sprintf("%d ops", l.OperationCount),
				IsLatest:    i == 0,
			})
		}
		data.Ledgers = homeLedgers
	} else {
		// Use mock ledgers on error.
		mock := mockHomeData(network)
		data.Ledgers = mock.Ledgers
	}

	// Transform contracts.
	if contractErr == nil && len(contracts) > 0 {
		homeContracts := make([]pages.HomeContract, 0, len(contracts))
		for i, c := range contracts {
			homeContracts = append(homeContracts, pages.HomeContract{
				Rank:        i + 1,
				Name:        gateway.ShortAddress(c.ContractID),
				Tag:         "Contract",
				TagColor:    "violet",
				Address:     gateway.ShortAddress(c.ContractID),
				Href:        "/contracts/" + c.ContractID,
				Invocations: gateway.FormatNumber(c.TotalCalls),
				Change:      "",
				IsPositive:  true,
			})
		}
		data.Contracts = homeContracts
	} else {
		mock := mockHomeData(network)
		data.Contracts = mock.Contracts
	}

	// Transactions — not easily available from bronze without a lot of enrichment.
	// Use mock for now; Phase 2 will wire /silver/payments or /silver/transfers.
	mockData := mockHomeData(network)
	data.Transactions = mockData.Transactions
	data.Assets = mockData.Assets

	return data, nil
}

// LatestLedgerPartial returns an HTML fragment with the current latest ledger + age.
// Used by htmx polling on the home page. Falls back to mock values when
// the gateway is unavailable (e.g. mainnet, which isn't live yet).
func (h *Handlers) LatestLedgerPartial(w http.ResponseWriter, r *http.Request) {
	network := networkFromRequest(r)

	// Default to mock values so we never show "—".
	mock := mockHomeData(network)
	ledgerNum := mock.LatestLedger
	age := mock.LedgerAge

	if h.Gateway != nil {
		ctx := r.Context()
		// Prefer bronze for accurate sequence + close time.
		if bronze, err := h.Gateway.GetBronzeNetworkStats(ctx, network); err == nil {
			ledgerNum = gateway.FormatNumber(bronze.Ledger.LatestSequence)
			if t, err := time.Parse(time.RFC3339, bronze.Ledger.ClosedAt); err == nil {
				age = gateway.FormatAge(t)
			}
		} else if stats, err := h.Gateway.GetNetworkStats(ctx, network); err == nil {
			ledgerNum = gateway.FormatNumber(stats.Ledger.CurrentSequence)
			if t, err := time.Parse(time.RFC3339, stats.GeneratedAt); err == nil {
				age = gateway.FormatAge(t)
			}
		}
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `<div class="text-[10px] font-semibold uppercase tracking-wider text-gray-400">Latest Ledger</div>`+
		`<div class="text-lg font-bold text-gray-900 tabular font-mono mt-0.5">%s</div>`+
		`<div class="flex items-center justify-center gap-1 mt-0.5">`+
		`<span class="h-1.5 w-1.5 rounded-full bg-emerald-500"></span>`+
		`<span class="text-[10px] text-emerald-600 font-medium">%s</span>`+
		`</div>`, html.EscapeString(ledgerNum), html.EscapeString(age))
}

// Search renders the full search results page.
func (h *Handlers) Search(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	network := networkFromRequest(r)

	// Smart redirect: if query is unambiguously a single entity, go directly there.
	if redirect := h.detectSmartRedirect(r.Context(), network, query); redirect != "" {
		http.Redirect(w, r, redirect, http.StatusSeeOther)
		return
	}

	var data pages.SearchData
	if h.useLiveData(r) {
		if live, err := h.buildSearchData(r, network, query); err == nil {
			data = live
		} else {
			h.Logger.Warn("live search failed, falling back to mock", "error", err)
		}
	}
	if data.Query == "" {
		data = mockSearchData(query)
	}

	if err := pages.Search(data).Render(r.Context(), w); err != nil {
		h.Logger.Error("render search", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// detectSmartRedirect returns a redirect URL if the query unambiguously matches
// a single entity type. Returns "" if ambiguous or unknown.
func (h *Handlers) detectSmartRedirect(ctx context.Context, network, query string) string {
	q := query
	// G + 55 chars = Stellar account address
	if len(q) == 56 && q[0] == 'G' {
		return "/account/" + q
	}
	// C + 55 chars = Stellar contract or smart wallet address
	if len(q) == 56 && q[0] == 'C' {
		if h.Gateway != nil {
			if info, err := h.Gateway.GetSmartWalletInfo(ctx, network, q); err == nil && info != nil && info.IsSmartWallet {
				return "/account/" + q + "/smart"
			}
		}
		return "/contracts/" + q
	}
	// 64 hex chars = transaction hash
	if len(q) == 64 && isHex(q) {
		return "/tx/" + q
	}
	// Pure number = ledger sequence
	if isNumeric(q) {
		return "/ledger/" + q
	}
	return ""
}

func isHex(s string) bool {
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}

func isNumeric(s string) bool {
	if len(s) == 0 {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

func (h *Handlers) buildSearchData(r *http.Request, network, query string) (pages.SearchData, error) {
	ctx := r.Context()

	results, err := h.Gateway.Search(ctx, network, query)
	if err != nil {
		return pages.SearchData{}, fmt.Errorf("searching: %w", err)
	}

	// Group results by type.
	groups := make(map[string][]pages.SearchResultItem)
	for _, sr := range results.Results {
		iconBg := "bg-gray-100"
		iconHTML := ""
		badge := sr.Type
		badgeColor := "gray"
		href := "/"

		switch sr.Type {
		case "account":
			iconBg = "bg-gray-100"
			badge = "Account"
			href = "/account/" + sr.ID
		case "contract":
			iconBg = "bg-violet-100"
			badge = "Contract"
			badgeColor = "violet"
			href = "/contracts/" + sr.ID
		case "transaction":
			iconBg = "bg-cyan-100"
			badge = "Transaction"
			badgeColor = "cyan"
			href = "/tx/" + sr.ID
		case "ledger":
			iconBg = "bg-emerald-100"
			badge = "Ledger"
			badgeColor = "emerald"
			href = "/ledger/" + sr.ID
		case "asset":
			iconBg = "bg-blue-100"
			badge = "Asset"
			badgeColor = "blue"
			href = "/assets/" + sr.ID
		case "token":
			iconBg = "bg-cyan-100"
			badge = "Token"
			badgeColor = "cyan"
			href = "/contracts/" + sr.ID
		case "smart_wallet":
			iconBg = "bg-violet-100"
			badge = "Smart Wallet"
			badgeColor = "violet"
			href = "/account/" + sr.ID + "/smart"
		}

		groups[sr.Type] = append(groups[sr.Type], pages.SearchResultItem{
			Name:       sr.Label,
			Badge:      badge,
			BadgeColor: badgeColor,
			Subtitle:   sr.ID,
			IconBg:     iconBg,
			IconHTML:   iconHTML,
			Href:       href,
		})
	}

	var resultGroups []pages.SearchResultGroup
	for category, items := range groups {
		resultGroups = append(resultGroups, pages.SearchResultGroup{
			Category: category,
			Items:    items,
		})
	}
	sort.Slice(resultGroups, func(i, j int) bool {
		order := map[string]int{"ledger": 0, "transaction": 1, "account": 2, "contract": 3, "token": 4, "asset": 5}
		oi, ok := order[resultGroups[i].Category]
		if !ok {
			oi = 99
		}
		oj, ok := order[resultGroups[j].Category]
		if !ok {
			oj = 99
		}
		return oi < oj
	})

	// Detect type for the top-of-page hint.
	detectedType := ""
	detectedLabel := ""
	detectedDesc := ""
	detectedHref := ""
	if len(results.Results) > 0 {
		top := results.Results[0]
		detectedType = top.Type
		detectedLabel = fmt.Sprintf("Detected as %s: %s", top.Type, top.Label)
		switch top.Type {
		case "account":
			detectedHref = "/account/" + top.ID
			detectedDesc = "Starts with G + alphanumeric characters. Redirecting to account view."
		case "contract":
			detectedHref = "/contracts/" + top.ID
			detectedDesc = "Starts with C + alphanumeric characters. Redirecting to contract view."
		case "transaction":
			detectedHref = "/tx/" + top.ID
			detectedDesc = "64-character hex string detected. Redirecting to transaction receipt."
		case "ledger":
			detectedHref = "/ledger/" + top.ID
			detectedDesc = "Numeric sequence detected. Redirecting to ledger detail."
		}
	}

	data := pages.SearchData{
		Query:         query,
		DetectedType:  detectedType,
		DetectedLabel: detectedLabel,
		DetectedDesc:  detectedDesc,
		DetectedHref:  detectedHref,
		Results:       resultGroups,
	}

	return data, nil
}

// SearchResults returns an HTML fragment for the live search dropdown on the home page.
// Triggered by hx-get="/partials/search-results" with debounced keyup.
func (h *Handlers) SearchResults(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		w.WriteHeader(http.StatusOK)
		return
	}

	network := networkFromRequest(r)

	// Smart redirect hint — if unambiguous, show a direct link.
	if redirect := h.detectSmartRedirect(r.Context(), network, query); redirect != "" {
		label := "View result"
		switch {
		case len(query) == 56 && query[0] == 'G':
			label = "Account " + gateway.ShortAddress(query)
		case len(query) == 56 && query[0] == 'C' && strings.Contains(redirect, "/smart"):
			label = "Smart Wallet " + gateway.ShortAddress(query)
		case len(query) == 56 && query[0] == 'C':
			label = "Contract " + gateway.ShortAddress(query)
		case len(query) == 64 && isHex(query):
			label = "Transaction " + gateway.ShortHash(query)
		case isNumeric(query):
			label = "Ledger #" + query
		}
		fmt.Fprintf(w, `<div class="rounded-xl border border-border-default bg-surface-card shadow-lg overflow-hidden">
			<a href="%s" class="flex items-center gap-3 px-4 py-3 hover:bg-surface-subtle transition-colors">
				<div class="flex h-8 w-8 items-center justify-center rounded-lg bg-emerald-50 dark:bg-emerald-950/30 flex-shrink-0">
					<svg class="h-4 w-4 text-emerald-600" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M14 5l7 7m0 0l-7 7m7-7H3"></path></svg>
				</div>
				<div class="flex-1">
					<span class="text-sm font-medium text-text-primary">%s</span>
					<span class="text-2xs text-emerald-600 ml-2">Direct match — press Enter</span>
				</div>
			</a>
		</div>`, redirect, html.EscapeString(label))
		return
	}

	// Full search via gateway.
	if h.Gateway == nil {
		fmt.Fprintf(w, `<div class="rounded-xl border border-border-default bg-surface-card shadow-lg px-4 py-3 text-sm text-text-muted">Search unavailable</div>`)
		return
	}

	results, err := h.Gateway.Search(r.Context(), network, query)
	if err != nil || len(results.Results) == 0 {
		fmt.Fprintf(w, `<div class="rounded-xl border border-border-default bg-surface-card shadow-lg px-4 py-3 text-sm text-text-muted">No results for "%s"</div>`, html.EscapeString(query))
		return
	}

	// Render compact result list (max 6 items).
	w.Write([]byte(`<div class="rounded-xl border border-border-default bg-surface-card shadow-lg overflow-hidden">`))
	limit := 6
	if len(results.Results) < limit {
		limit = len(results.Results)
	}
	for i := 0; i < limit; i++ {
		sr := results.Results[i]
		href := "/"
		badge := sr.Type
		badgeColor := "gray"
		switch sr.Type {
		case "account":
			href = "/account/" + sr.ID
			badge = "Account"
		case "contract":
			href = "/contracts/" + sr.ID
			badge = "Contract"
			badgeColor = "violet"
		case "transaction":
			href = "/tx/" + sr.ID
			badge = "Tx"
			badgeColor = "cyan"
		case "ledger":
			href = "/ledger/" + sr.ID
			badge = "Ledger"
			badgeColor = "emerald"
		case "asset":
			href = "/assets/" + sr.ID
			badge = "Asset"
			badgeColor = "blue"
		case "token":
			href = "/contracts/" + sr.ID
			badge = "Token"
			badgeColor = "cyan"
		case "smart_wallet":
			href = "/account/" + sr.ID + "/smart"
			badge = "Wallet"
			badgeColor = "violet"
		}
		safeHref := html.EscapeString(href)
		fmt.Fprintf(w, `<a href="%s" class="flex items-center gap-3 px-4 py-2.5 hover:bg-surface-subtle transition-colors border-b border-border-subtle last:border-b-0">
			<div class="flex-1 min-w-0">
				<span class="text-sm text-text-primary truncate block">%s</span>
				<span class="font-mono text-2xs text-text-muted truncate block">%s</span>
			</div>
			<span class="rounded-full px-2 py-0.5 text-2xs font-semibold ring-1 text-%s-700 bg-%s-50 ring-%s-200 dark:text-%s-400 dark:bg-%s-950/30 dark:ring-%s-800 flex-shrink-0">%s</span>
		</a>`, safeHref,
			html.EscapeString(sr.Label),
			html.EscapeString(gateway.ShortAddress(sr.ID)),
			badgeColor, badgeColor, badgeColor, badgeColor, badgeColor, badgeColor,
			html.EscapeString(badge))
	}
	if len(results.Results) > limit {
		fmt.Fprintf(w, `<a href="/search?q=%s" class="block px-4 py-2 text-center text-xs font-medium text-text-body hover:bg-surface-subtle transition-colors">View all %d results →</a>`,
			html.EscapeString(url.QueryEscape(query)), len(results.Results))
	}
	w.Write([]byte(`</div>`))
}

// LedgerDetail renders a single ledger page.
func (h *Handlers) LedgerDetail(w http.ResponseWriter, r *http.Request) {
	sequence := r.PathValue("sequence")
	network := networkFromRequest(r)

	var data pages.LedgerDetailData
	if h.useLiveData(r) {
		if live, err := h.buildLedgerDetailData(r, network, sequence); err == nil {
			data = live
		} else {
			h.Logger.Warn("live ledger shell data failed, falling back to mock", "error", err)
		}
	}
	if data.Sequence == "" {
		data = mockLedgerDetailData(sequence)
	}

	if err := pages.LedgerDetail(data).Render(r.Context(), w); err != nil {
		h.Logger.Error("render ledger detail", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// buildLedgerDetailData fetches a single ledger and its transactions from the gateway.
// Prefers the single-call /silver/ledger/{seq}/full endpoint, falling back to the legacy
// 6-call fan-out if it fails.
func (h *Handlers) buildLedgerDetailData(r *http.Request, network, sequence string) (pages.LedgerDetailData, error) {
	ctx := r.Context()

	// Parse the sequence number.
	var seq int64
	if _, err := fmt.Sscanf(sequence, "%d", &seq); err != nil {
		return pages.LedgerDetailData{}, fmt.Errorf("invalid sequence: %s", sequence)
	}

	var (
		l             gateway.Ledger
		txs           []gateway.Transaction
		ops           []gateway.Operation
		ledgerFees    *gateway.LedgerFees
		ledgerSoroban *gateway.LedgerSoroban
		txErr         error
		opsErr        error
	)

	// Try the composite single-call endpoint first.
	if full, err := h.Gateway.GetSilverLedgerFull(ctx, network, seq); err == nil {
		l = full.Ledger
		txs = full.Transactions
		ops = full.Operations
		ledgerFees = full.Fees
		ledgerSoroban = full.Soroban
	} else {
		h.Logger.Debug("silver ledger full unavailable, falling back to legacy multi-call", "error", err, "seq", seq)

		// Fallback: legacy 4-call fan-out.
		ledgers, err := h.Gateway.GetLedgers(ctx, network, seq, seq, 1, "asc")
		if err != nil {
			return pages.LedgerDetailData{}, fmt.Errorf("fetching ledger %d: %w", seq, err)
		}
		if len(ledgers) == 0 {
			return pages.LedgerDetailData{}, fmt.Errorf("ledger %d not found", seq)
		}
		l = ledgers[0]

		txs, txErr = h.Gateway.GetTransactions(ctx, network, seq, seq, 50, "asc")
		ops, opsErr = h.Gateway.GetOperations(ctx, network, seq, seq, 200)
		ledgerFees, _ = h.Gateway.GetLedgerFees(ctx, network, seq)
		ledgerSoroban, _ = h.Gateway.GetLedgerSoroban(ctx, network, seq)
	}

	// Best-effort enrichment: batch-decoded summaries for the transactions list.
	// Not critical — fall back to source-account summary if unavailable.
	decodedMap := make(map[string]*gateway.DecodedTransaction)
	if decodedResp, err := h.Gateway.GetBatchDecodedTransactions(ctx, network, nil, seq, 50); err == nil && decodedResp != nil {
		for i := range decodedResp.Transactions {
			dt := &decodedResp.Transactions[i]
			decodedMap[dt.TxHash] = dt
		}
	}

	// Parse close time.
	closedAt := l.ClosedAt
	closeTime := "—"
	if t, err := time.Parse(time.RFC3339, l.ClosedAt); err == nil {
		closedAt = t.Format("Jan 2, 2006 · 15:04:05 UTC")
		closeTime = fmt.Sprintf("%.1fs", 5.0) // default close time estimate
	}

	totalCoins := fmt.Sprintf("%.7f XLM", float64(l.TotalCoins)/10_000_000)
	opsPerTx := float64(0)
	if l.SuccessfulTxCount > 0 {
		opsPerTx = float64(l.OperationCount) / float64(l.SuccessfulTxCount)
	}

	data := pages.LedgerDetailData{
		Sequence:        gateway.FormatNumber(l.Sequence),
		SequenceRaw:     fmt.Sprintf("%d", l.Sequence),
		PrevSequence:    gateway.FormatNumber(l.Sequence - 1),
		PrevSequenceRaw: fmt.Sprintf("%d", l.Sequence-1),
		NextSequence:    gateway.FormatNumber(l.Sequence + 1),
		NextSequenceRaw: fmt.Sprintf("%d", l.Sequence+1),
		ClosedAt:        closedAt,
		CloseTime:       closeTime,
		Hash:            gateway.ShortHash(l.LedgerHash),
		PrevHash:        gateway.ShortHash(l.PreviousLedgerHash),
		Protocol:        fmt.Sprintf("%d", l.ProtocolVersion),
		BaseFee:         gateway.FormatNumber(l.BaseFee),
		MaxTxSetSize:    fmt.Sprintf("%d", l.MaxTxSetSize),
		TotalCoins:      totalCoins,
		TxCount:         fmt.Sprintf("%d", l.TransactionCount),
		TxSuccess:       fmt.Sprintf("%d", l.SuccessfulTxCount),
		TxFailed:        fmt.Sprintf("%d", l.FailedTxCount),
		OpCount:         fmt.Sprintf("%d", l.OperationCount),
		OpsPerTx:        fmt.Sprintf("%.1f", opsPerTx),
		TotalFees: func() string {
			if l.TotalFeeCharged != nil {
				return gateway.FormatNumber(*l.TotalFeeCharged)
			}
			return gateway.FormatNumber(l.BaseFee * int64(l.TransactionCount))
		}(),
		SorobanCalls: func() string {
			if l.SorobanOpCount != nil {
				return fmt.Sprintf("%d", *l.SorobanOpCount)
			}
			return "—"
		}(),
		SorobanPct: "—",
		FeesUSD:    "—",
		EventsEmitted: func() string {
			if l.ContractEventsCount != nil {
				return fmt.Sprintf("%d", *l.ContractEventsCount)
			}
			return "—"
		}(),
		// Soroban runtime — from per-ledger endpoint.
		TotalCPU: func() string {
			if ledgerSoroban != nil {
				return gateway.FormatAbbrev(ledgerSoroban.TotalCPUInsns) + " insn"
			}
			return "—"
		}(),
		StateReads: func() string {
			if ledgerSoroban != nil {
				return fmt.Sprintf("%d", ledgerSoroban.TotalReadBytes)
			}
			return "—"
		}(),
		StateReadKB: func() string {
			if ledgerSoroban != nil {
				return fmt.Sprintf("%.1f KB", float64(ledgerSoroban.TotalReadBytes)/1024)
			}
			return "—"
		}(),
		StateWrites: func() string {
			if ledgerSoroban != nil {
				return fmt.Sprintf("%d", ledgerSoroban.TotalWriteBytes)
			}
			return "—"
		}(),
		StateWriteKB: func() string {
			if ledgerSoroban != nil {
				return fmt.Sprintf("%.1f KB", float64(ledgerSoroban.TotalWriteBytes)/1024)
			}
			return "—"
		}(),
		RentBurned: func() string {
			if ledgerSoroban != nil {
				return fmt.Sprintf("%.4f", float64(ledgerSoroban.TotalRentCharged)/10_000_000)
			}
			return "—"
		}(),
		// Fee distribution — from per-ledger endpoint.
		FeeBase: gateway.FormatNumber(l.BaseFee),
		FeeMedian: func() string {
			if ledgerFees != nil {
				return gateway.FormatNumber(ledgerFees.MedianFee)
			}
			return "—"
		}(),
		FeeP99: func() string {
			if ledgerFees != nil {
				return gateway.FormatNumber(ledgerFees.P90Fee)
			}
			return "—"
		}(), // API provides P90; P99 not available per-ledger
		SurgePct: func() string {
			if ledgerFees != nil && l.MaxTxSetSize > 0 {
				return fmt.Sprintf("%d%%", ledgerFees.TxCount*100/l.MaxTxSetSize)
			}
			return "—"
		}(),
	}

	// Op breakdown from operations data.
	sorobanCount := 0
	if opsErr == nil && len(ops) > 0 {
		typeCounts := make(map[string]int)
		for _, op := range ops {
			typeCounts[op.TypeName]++
			if op.IsSorobanOp {
				sorobanCount++
			}
		}
		totalOps := len(ops)
		if totalOps > 0 {
			data.SorobanCalls = fmt.Sprintf("%d", sorobanCount)
			data.SorobanPct = fmt.Sprintf("%d%%", sorobanCount*100/totalOps)
		}

		colors := []string{"bg-violet-500", "bg-cyan-500", "bg-emerald-500", "bg-amber-500", "bg-gray-400", "bg-gray-300"}
		i := 0
		for name, count := range typeCounts {
			pct := count * 100 / totalOps
			color := "bg-gray-300"
			if i < len(colors) {
				color = colors[i]
			}
			data.OpBreakdown = append(data.OpBreakdown, pages.OpBreakdownItem{
				Name:  name,
				Count: fmt.Sprintf("%d", count),
				Pct:   fmt.Sprintf("%d%%", pct),
				Width: fmt.Sprintf("%d%%", pct),
				Color: color,
			})
			i++
		}
	} else {
		data.OpBreakdown = []pages.OpBreakdownItem{}
	}

	// Transform transactions — use decoded data when available for rich summaries.
	if txErr == nil && len(txs) > 0 {
		ledgerTxs := make([]pages.LedgerTx, 0, len(txs))
		for i, tx := range txs {
			status := "ok"
			if !tx.Successful {
				status = "failed"
			}
			opType := "tx"
			opColor := "gray"
			summary := fmt.Sprintf(`<span class="font-medium text-gray-900">%s</span> · %d op(s)`,
				html.EscapeString(gateway.ShortAddress(tx.SourceAccount)), tx.OperationCount)

			// Enrich from decoded data if available.
			if dt, ok := decodedMap[tx.TransactionHash]; ok {
				if dt.Summary != nil {
					summary = html.EscapeString(dt.Summary.Description)
					switch dt.Summary.Type {
					case "transfer":
						opType = "transfer"
						opColor = "cyan"
					case "swap":
						opType = "swap"
						opColor = "emerald"
					case "mint":
						opType = "mint"
						opColor = "emerald"
					case "burn":
						opType = "burn"
						opColor = "red"
					case "contract_call":
						opType = "invoke"
						opColor = "violet"
					case "multi_op":
						opType = "multi"
						opColor = "violet"
					default:
						opType = dt.Summary.Type
					}
				}
				if walletInfo := h.firstSmartWalletForDecodedTx(ctx, network, dt); walletInfo != nil {
					label := walletTypeLabel(firstNonEmpty(walletInfo.WalletType, walletInfo.Implementation))
					opType = "wallet"
					opColor = "violet"
					summary = html.EscapeString(fmt.Sprintf("%s wallet interaction from %s", label, gateway.ShortAddress(tx.SourceAccount)))
				}
			} else if tx.OperationCount > 1 {
				opType = "multi"
				opColor = "violet"
			}

			ledgerTxs = append(ledgerTxs, pages.LedgerTx{
				Index:     fmt.Sprintf("%d", i+1),
				Status:    status,
				Hash:      tx.TransactionHash,
				ShortHash: gateway.ShortHash(tx.TransactionHash),
				OpType:    opType,
				OpColor:   opColor,
				Summary:   summary,
				Ops:       fmt.Sprintf("%d", tx.OperationCount),
				Fee:       gateway.FormatNumber(tx.MaxFee),
				IsFailed:  !tx.Successful,
			})
		}
		data.Transactions = ledgerTxs
	} else {
		data.Transactions = []pages.LedgerTx{}
	}

	return data, nil
}

// TransactionReceipt renders a single transaction page.
func (h *Handlers) TransactionReceipt(w http.ResponseWriter, r *http.Request) {
	hash := r.PathValue("hash")
	shortHash := hash
	if len(hash) > 12 {
		shortHash = hash[:6] + "..." + hash[len(hash)-4:]
	}
	network := networkFromRequest(r)

	var data pages.TxReceiptData
	if h.useLiveData(r) {
		if live, err := h.buildTxReceiptData(r, network, hash, shortHash); err == nil {
			data = live
		} else {
			h.Logger.Warn("live tx shell data failed, falling back to mock", "error", err)
		}
	}
	if data.Hash == "" {
		data = mockTxReceiptData(hash, shortHash)
	}

	if err := pages.TransactionReceiptV2(data).Render(r.Context(), w); err != nil {
		h.Logger.Error("render transaction receipt", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func (h *Handlers) TransactionReceiptV2(w http.ResponseWriter, r *http.Request) {
	hash := r.PathValue("hash")
	shortHash := hash
	if len(hash) > 12 {
		shortHash = hash[:6] + "..." + hash[len(hash)-4:]
	}
	data := mockTxReceiptData(hash, shortHash)
	pages.TransactionReceiptV2(data).Render(r.Context(), w)
}

func (h *Handlers) TransactionReceiptV3(w http.ResponseWriter, r *http.Request) {
	hash := r.PathValue("hash")
	shortHash := hash
	if len(hash) > 12 {
		shortHash = hash[:6] + "..." + hash[len(hash)-4:]
	}
	data := mockTxReceiptData(hash, shortHash)
	pages.TransactionReceiptV3(data).Render(r.Context(), w)
}

func (h *Handlers) TransactionReceiptV1(w http.ResponseWriter, r *http.Request) {
	hash := r.PathValue("hash")
	shortHash := hash
	if len(hash) > 12 {
		shortHash = hash[:6] + "..." + hash[len(hash)-4:]
	}
	data := mockTxReceiptData(hash, shortHash)
	pages.TransactionReceipt(data).Render(r.Context(), w)
}

// resolveEventAsset returns the (assetLabel, decimals) for an event, applying the precedence
// chain laid out by the silver enricher contract:
//
//  1. token_symbol (inlined by the silver enricher) → use it
//  2. asset_code (classic asset on the event) → use it
//  3. classic native PAYMENT (operation_type=1, source_type=classic, no contract_id, no asset_code) → "XLM"
//  4. unknown contract reference fallback → "token (CBIE...DAMA)"
//  5. completely unknown → "" (render amount alone)
//
// We deliberately do NOT default to "XLM" for non-PAYMENT classic ops (e.g. PATH_PAYMENT)
// because path payments can move arbitrary assets — guessing "XLM" risks displaying wrong data.
func resolveEventAsset(evt gateway.UnifiedEvent) (label string, decimals int) {
	// Decimals default to 7 (native XLM and most Stellar SACs); honor explicit token_decimals when set.
	decimals = 7
	if evt.TokenDecimals > 0 {
		decimals = evt.TokenDecimals
	}

	// 1. Inlined token metadata (preferred).
	if evt.TokenSymbol != "" {
		return evt.TokenSymbol, decimals
	}

	// 2. Explicit classic asset code.
	if evt.AssetCode != "" {
		return evt.AssetCode, decimals
	}

	// 3. Strict classic native: only operation_type 1 (PAYMENT) qualifies as unambiguous native.
	const opTypePayment = 1
	if evt.SourceType == "classic" && evt.ContractID == "" && evt.OperationType == opTypePayment {
		return "XLM", decimals
	}

	// 4. Unknown contract — at least surface the contract reference so the user isn't lied to.
	if evt.ContractID != "" {
		return "token (" + gateway.ShortAddress(evt.ContractID) + ")", decimals
	}

	// 5. No usable info — render amount with no asset label.
	return "", decimals
}

// buildTxReceiptData fetches real transaction data from the gateway.
func (h *Handlers) buildTxReceiptData(r *http.Request, network, hash, shortHash string) (pages.TxReceiptData, error) {
	ctx := r.Context()

	// Fetch the consolidated receipt as the single source of truth for the tx page.
	receipt, err := h.Gateway.GetTransactionReceipt(ctx, network, hash)
	if err != nil {
		return pages.TxReceiptData{}, fmt.Errorf("fetching tx receipt: %w", err)
	}

	opsDecoded := make([]gateway.DecodedOperation, 0, len(receipt.Full.Operations))
	for _, op := range receipt.Full.Operations {
		idx := op.OperationIndex
		if idx == 0 && op.Index > 0 {
			idx = op.Index
		}
		opsDecoded = append(opsDecoded, gateway.DecodedOperation{
			Index:         idx,
			Type:          op.Type,
			TypeName:      op.TypeName,
			SourceAccount: op.SourceAccount,
			ContractID:    op.ContractID,
			FunctionName:  op.FunctionName,
			IsSorobanOp:   op.IsSorobanOp,
			Destination:   op.Destination,
			Amount:        op.Amount,
			AssetCode:     op.AssetCode,
		})
	}
	if len(opsDecoded) == 0 && len(receipt.Semantic.Operations) > 0 {
		opsDecoded = append(opsDecoded, receipt.Semantic.Operations...)
	}

	txLedger := receipt.LedgerSequence
	if receipt.Full.LedgerSequence > 0 {
		txLedger = receipt.Full.LedgerSequence
	}
	txClosedAt := firstNonEmpty(receipt.Full.CreatedAt, receipt.CreatedAt)
	txSource := firstNonEmpty(receipt.Full.SourceAccount, receipt.SourceAccount)
	tx := gateway.TxInfo{
		TxHash:          receipt.TxHash,
		Fee:             receipt.Full.Fee,
		MaxFee:          receipt.Full.MaxFee,
		LedgerSequence:  txLedger,
		OperationCount:  receipt.OperationCount,
		Successful:      receipt.Successful,
		ClosedAt:        txClosedAt,
		SourceAccount:   txSource,
		AccountSequence: receipt.Full.AccountSequence,
	}
	summary := receipt.Full.Summary
	if legacy := receipt.Semantic.LegacySummary; legacy.Description != "" {
		if summary.Description == "" || strings.Contains(summary.Description, " asset ") || (summary.Transfer != nil && summary.Transfer.Asset == "asset") {
			summary = legacy
		}
	}
	if summary.Description == "" {
		summary.Description = receipt.TxType
	}
	if summary.Type == "" {
		summary.Type = firstNonEmpty(receipt.Semantic.Classification.TxType, receipt.TxType)
	}
	events := receipt.Full.Events
	if len(events) == 0 {
		events = receipt.Semantic.Events
	}
	if len(events) == 0 {
		events = receipt.Events
	}
	txFull := &gateway.TxFull{
		Transaction: tx,
		Summary:     summary,
		Operations:  opsDecoded,
		Events:      events,
	}
	if receipt.Full.SorobanResourcesInstructions != nil || receipt.Full.SorobanResourcesReadBytes != nil || receipt.Full.SorobanResourcesWriteBytes != nil {
		txFull.SorobanResources = &gateway.SorobanResources{}
		if receipt.Full.SorobanResourcesInstructions != nil {
			txFull.SorobanResources.Instructions = *receipt.Full.SorobanResourcesInstructions
		}
		if receipt.Full.SorobanResourcesReadBytes != nil {
			txFull.SorobanResources.ReadBytes = *receipt.Full.SorobanResourcesReadBytes
		}
		if receipt.Full.SorobanResourcesWriteBytes != nil {
			txFull.SorobanResources.WriteBytes = *receipt.Full.SorobanResourcesWriteBytes
		}
	}
	var semanticTx *gateway.SemanticTransactionResponse
	if receipt.Semantic.Transaction.TxHash != "" || receipt.Semantic.Classification.TxType != "" || len(receipt.Semantic.Actors) > 0 {
		semanticCopy := receipt.Semantic
		if semanticCopy.Transaction.TxHash == "" {
			sourceAccount := tx.SourceAccount
			semanticCopy.Transaction = gateway.SemanticTransactionInfo{
				TxHash:          tx.TxHash,
				LedgerSequence:  tx.LedgerSequence,
				ClosedAt:        tx.ClosedAt,
				Successful:      tx.Successful,
				Fee:             tx.Fee,
				OperationCount:  tx.OperationCount,
				SourceAccount:   &sourceAccount,
				AccountSequence: &tx.AccountSequence,
				MaxFee:          &tx.MaxFee,
			}
		}
		if semanticCopy.Classification.TxType == "" {
			semanticCopy.Classification.TxType = firstNonEmpty(summary.Type, receipt.TxType)
		}
		if len(semanticCopy.Operations) == 0 {
			semanticCopy.Operations = opsDecoded
		}
		if len(semanticCopy.Events) == 0 {
			semanticCopy.Events = events
		}
		if semanticCopy.LegacySummary.Description == "" {
			semanticCopy.LegacySummary = summary
		}
		semanticTx = &semanticCopy
	}

	normalizedCtx, normErr := h.buildNormalizedTxContext(ctx, network, txFull, semanticTx)
	if normErr != nil {
		h.Logger.Debug("smart-wallet tx normalization unavailable", "hash", hash, "error", normErr)
	}

	status := "success"
	if !tx.Successful {
		status = "failed"
	}

	// Determine if Soroban.
	isSoroban := false
	hasClassic := false
	for _, op := range txFull.Operations {
		if op.IsSorobanOp {
			isSoroban = true
		} else {
			hasClassic = true
		}
	}

	// Parse timestamp.
	timestamp := tx.ClosedAt
	timestampRelative := ""
	timestampISO := tx.ClosedAt
	if t, err := time.Parse(time.RFC3339, tx.ClosedAt); err == nil {
		timestamp = t.Format("Jan 2, 2006 at 15:04:05 UTC")
		timestampRelative = gateway.FormatAge(t)
		timestampISO = t.Format(time.RFC3339)
	}

	// Build summary HTML from the summary description.
	summaryHTML := html.EscapeString(summary.Description)
	if summaryHTML == "" && len(txFull.Operations) == 1 {
		summaryHTML = buildOperationSummary(txFull.Operations[0])
	}

	// Extract flow diagram info from summary.
	sourceAddr := "—"
	sourceAmount := "—"
	contractName := "—"
	contractAddr := "—"
	contractAddrFull := ""
	contractFn := "—"
	destAddr := "—"
	destAmount := "—"
	effectiveRate := "—"
	slippage := "—"
	route := "—"

	if summary.Transfer != nil {
		sourceAddr = gateway.ShortAddress(summary.Transfer.From)
		asset := summary.Transfer.Asset
		amt := ""
		if asset == "" || asset == "tokens" || asset == "XLM" {
			asset = "XLM"
			amt = gateway.FormatStroopsToXLM(summary.Transfer.Amount)
		} else {
			amt = gateway.FormatDecimalAmount(summary.Transfer.Amount)
		}
		sourceAmount = amt + " " + asset
		destAddr = gateway.ShortAddress(summary.Transfer.To)
		destAmount = "+" + amt + " " + asset
	}
	if summary.Swap != nil {
		inAmount := gateway.FormatDecimalAmount(summary.Swap.AmountIn)
		if summary.Swap.AssetIn == "XLM" {
			inAmount = gateway.FormatStroopsToXLM(summary.Swap.AmountIn)
		}
		outAmount := gateway.FormatDecimalAmount(summary.Swap.AmountOut)
		if summary.Swap.AssetOut == "XLM" {
			outAmount = gateway.FormatStroopsToXLM(summary.Swap.AmountOut)
		}
		sourceAmount = inAmount + " " + summary.Swap.AssetIn
		destAmount = "+" + outAmount + " " + summary.Swap.AssetOut
		route = summary.Swap.AssetIn + " → " + summary.Swap.AssetOut
	}
	if len(txFull.Operations) > 0 {
		op := txFull.Operations[0]
		if op.ContractID != "" {
			contractAddr = gateway.ShortAddress(op.ContractID)
			contractAddrFull = op.ContractID
			contractName = contractAddr
		}
		if op.FunctionName != "" {
			contractFn = op.FunctionName + "()"
		}
		if op.SourceAccount != "" {
			sourceAddr = gateway.ShortAddress(op.SourceAccount)
		}
		if op.Destination != "" && destAddr == "—" {
			destAddr = gateway.ShortAddress(op.Destination)
		}
		if op.Amount != "" && sourceAmount == "—" {
			asset := op.AssetCode
			if asset == "" {
				asset = "XLM"
				sourceAmount = gateway.FormatStroopsToXLM(op.Amount) + " " + asset
				destAmount = "+" + gateway.FormatStroopsToXLM(op.Amount) + " " + asset
			} else {
				sourceAmount = gateway.FormatDecimalAmount(op.Amount) + " " + asset
				destAmount = "+" + gateway.FormatDecimalAmount(op.Amount) + " " + asset
			}
		}
	}

	// Build operations list.
	ops := make([]pages.TxOperation, 0, len(txFull.Operations))
	for _, op := range txFull.Operations {
		opStatus := "Success"
		if !tx.Successful {
			opStatus = "Failed"
		}

		opSummary := buildOperationSummary(op)

		ops = append(ops, pages.TxOperation{
			Index:       fmt.Sprintf("%d", op.Index+1),
			Type:        gateway.OperationDisplayName(op.TypeName),
			IsSoroban:   op.IsSorobanOp,
			IsPrimary:   op.Index == 0,
			Status:      opStatus,
			SummaryHTML: opSummary,
			Contract:    gateway.ShortAddress(op.ContractID),
			Function:    op.FunctionName,
		})
	}

	// Build events list from unified events.
	// Token metadata (symbol, decimals) is now inlined on each event by the API enricher.
	evts := make([]pages.TxEvent, 0)
	for i, evt := range txFull.Events {
		typeColor := "gray"
		switch evt.EventType {
		case "transfer":
			typeColor = "cyan"
		case "mint":
			typeColor = "emerald"
		case "burn":
			typeColor = "red"
		}

		dataHTML := ""
		if evt.Amount != "" {
			assetLabel, decimals := resolveEventAsset(evt)
			amount := gateway.FormatTokenAmount(evt.Amount, decimals)
			display := amount
			if assetLabel != "" {
				display = amount + " " + assetLabel
			}
			dataHTML = fmt.Sprintf(`<span class="font-medium">%s</span>`, html.EscapeString(display))
			if evt.From != "" {
				dataHTML += fmt.Sprintf(` from <span class="font-mono text-xs">%s</span>`, html.EscapeString(gateway.ShortAddress(evt.From)))
			}
			if evt.To != "" {
				dataHTML += fmt.Sprintf(` to <span class="font-mono text-xs">%s</span>`, html.EscapeString(gateway.ShortAddress(evt.To)))
			}
		}
		evts = append(evts, pages.TxEvent{
			Index:     fmt.Sprintf("%d", i),
			Type:      evt.EventType,
			TypeColor: typeColor,
			Contract:  gateway.ShortAddress(evt.ContractID),
			DataHTML:  dataHTML,
		})
	}

	// Build balance changes from receipt diffs.
	var balChanges []pages.TxBalanceChange
	for _, bc := range receipt.Diffs {
		isPositive := true
		if strings.HasPrefix(bc.Delta, "-") {
			isPositive = false
		}
		asset := bc.Asset
		assetType := "Classic"
		if asset == "" || asset == "native" {
			asset = "XLM"
			assetType = "Native"
		}
		balChanges = append(balChanges, pages.TxBalanceChange{
			Account:    gateway.ShortAddress(bc.Account),
			Asset:      asset,
			AssetType:  assetType,
			TypeColor:  "gray",
			Change:     bc.Delta,
			IsPositive: isPositive,
		})
	}

	// Receipt currently exposes only flattened balance diffs; state changes are omitted.
	var stateChanges []pages.TxStateChange

	// Build effects list from receipt effects.
	var effects []pages.TxEffect
	for i, eff := range receipt.Effects {
		effectType := eff.EffectTypeString
		if effectType == "" {
			effectType = fmt.Sprint(eff.EffectType)
		}
		amount := ""
		if eff.Amount != "" {
			amount = gateway.FormatDecimalAmount(eff.Amount)
		}
		isPositive := strings.Contains(effectType, "credited") || strings.Contains(effectType, "created")
		asset := ""
		if eff.Asset != nil {
			asset = eff.Asset.Code
		}
		summary := html.EscapeString(gateway.EffectDisplayName(effectType))
		if amount != "" {
			display := amount
			if asset != "" {
				display += " " + asset
			}
			summary = fmt.Sprintf(`%s <span class="font-semibold">%s</span>`, summary, html.EscapeString(display))
		}
		effects = append(effects, pages.TxEffect{
			Index:       fmt.Sprintf("%d", i),
			Type:        effectType,
			TypeDisplay: gateway.EffectDisplayName(effectType),
			TypeColor:   gateway.EffectTypeColor(effectType),
			Account:     gateway.ShortAddress(eff.AccountID),
			AccountFull: eff.AccountID,
			Asset:       asset,
			Amount:      amount,
			IsPositive:  isPositive,
			SummaryHTML: summary,
		})
	}

	data := pages.TxReceiptData{
		Hash:              hash,
		ShortHash:         shortHash,
		Status:            status,
		IsSoroban:         isSoroban,
		HasClassicOps:     hasClassic,
		SummaryHTML:       summaryHTML,
		Timestamp:         timestamp,
		TimestampRelative: timestampRelative,
		TimestampISO:      timestampISO,
		Ledger:            gateway.FormatNumber(tx.LedgerSequence),
		LedgerRaw:         fmt.Sprintf("%d", tx.LedgerSequence),
		OpsCount:          fmt.Sprintf("%d", tx.OperationCount),
		EventsCount:       fmt.Sprintf("%d", len(txFull.Events)),
		SourceAddr:        sourceAddr,
		SourceAddrFull:    tx.SourceAccount,
		SourceAmount:      sourceAmount,
		ContractName:      contractName,
		ContractAddr:      contractAddr,
		ContractAddrFull:  contractAddrFull,
		ContractFn:        contractFn,
		DestAddr:          destAddr,
		DestAmount:        destAmount,
		EffectiveRate:     effectiveRate,
		Slippage:          slippage,
		Route:             route,
		FeePaid:           gateway.FormatNumber(tx.Fee),
		FeePaidXLM:        gateway.FormatFeeXLM(tx.Fee),
		MaxFee: func() string {
			if tx.MaxFee > 0 {
				return gateway.FormatNumber(tx.MaxFee)
			}
			return ""
		}(),
		FeeUSD: "—",
		SorobanCPU: func() string {
			if txFull.SorobanResources != nil && txFull.SorobanResources.Instructions > 0 {
				return gateway.FormatAbbrev(txFull.SorobanResources.Instructions) + " insn"
			}
			return "—"
		}(),
		SorobanMem: "—",
		SorobanReads: func() string {
			if txFull.SorobanResources != nil && txFull.SorobanResources.ReadBytes > 0 {
				return fmt.Sprintf("%.1f KB", float64(txFull.SorobanResources.ReadBytes)/1024)
			}
			return "—"
		}(),
		SorobanWrites: func() string {
			if txFull.SorobanResources != nil && txFull.SorobanResources.WriteBytes > 0 {
				return fmt.Sprintf("%.1f KB", float64(txFull.SorobanResources.WriteBytes)/1024)
			}
			return "—"
		}(),
		SeqNumber: func() string {
			if tx.AccountSequence > 0 {
				return fmt.Sprintf("%d", tx.AccountSequence)
			}
			return ""
		}(),
		Operations:     ops,
		Events:         evts,
		BalanceChanges: balChanges,
		StateChanges:   stateChanges,
		Effects:        effects,
	}

	if semanticTx != nil {
		human := humanize.BuildTxNarrative(semanticTx)
		data.HumanTitle = human.Title
		data.HumanNarrative = human.Narrative
		data.ConfidenceLabel = human.ConfidenceLabel
		data.ConfidenceValue = human.ConfidenceValue
		data.SemanticTxType = semanticTx.Classification.TxType
		data.SemanticSubtype = semanticTx.Classification.Subtype
		for _, actor := range human.Actors {
			data.HumanActors = append(data.HumanActors, pages.TxHumanActor{
				Label:     actor.Label,
				Role:      actor.Role,
				ActorType: actor.ActorType,
				Href:      actor.Href,
			})
		}
		for _, ev := range human.Evidence {
			data.HumanEvidence = append(data.HumanEvidence, pages.TxHumanEvidence{
				Label: ev.Label,
				Value: ev.Value,
			})
		}
		for _, sig := range human.Signals {
			data.HumanSignals = append(data.HumanSignals, pages.TxHumanSignal{
				Title:    sig.Title,
				Severity: sig.Severity,
				Summary:  sig.Summary,
			})
		}
	}
	if data.HumanTitle == "" {
		data.HumanTitle = gateway.OperationDisplayName(firstNonEmpty(summary.Type, receipt.TxType))
	}
	if data.HumanNarrative == "" {
		data.HumanNarrative = summary.Description
	}

	if normalizedCtx != nil {
		data.ActorModelSource = normalizedCtx.SourceOfTruth
		if normalizedCtx.Submitter != nil {
			data.SubmitterAddr = normalizedCtx.Submitter.ID
			data.SubmitterShort = normalizedCtx.Submitter.Short
			data.SourceAddr = normalizedCtx.Submitter.Short
			data.SourceAddrFull = normalizedCtx.Submitter.ID
		}
		if normalizedCtx.FeePayer != nil {
			data.FeePayerAddr = normalizedCtx.FeePayer.ID
			data.FeePayerShort = normalizedCtx.FeePayer.Short
		}
		if normalizedCtx.EffectiveActor != nil {
			data.EffectiveActorAddr = normalizedCtx.EffectiveActor.ID
			data.EffectiveActorShort = normalizedCtx.EffectiveActor.Short
			data.EffectiveActorType = normalizedCtx.EffectiveActor.ActorType
			data.EffectiveActorHref = normalizedCtx.EffectiveActor.Href
		}
		if normalizedCtx.DownstreamContract != nil {
			data.DownstreamContractAddr = normalizedCtx.DownstreamContract.ID
			data.DownstreamContractShort = normalizedCtx.DownstreamContract.Short
			data.DownstreamContractHref = normalizedCtx.DownstreamContract.Href
		}
		if normalizedCtx.DownstreamFunction != "" {
			data.DownstreamFunctionName = normalizedCtx.DownstreamFunction
		}
		if normalizedCtx.Wallet != nil && normalizedCtx.Wallet.Detected {
			data.SmartWalletDetected = true
			data.SmartWalletContract = normalizedCtx.Wallet.ContractID
			data.SmartWalletContractShort = normalizedCtx.Wallet.ContractShort
			data.SmartWalletType = walletTypeLabel(normalizedCtx.Wallet.WalletType)
			data.SmartWalletImplementation = normalizedCtx.Wallet.Implementation
			data.SmartWalletConfidence = walletConfidenceLabel(normalizedCtx.Wallet.Confidence)
			if data.EffectiveActorAddr == "" {
				data.EffectiveActorAddr = normalizedCtx.Wallet.ContractID
				data.EffectiveActorShort = normalizedCtx.Wallet.ContractShort
				data.EffectiveActorType = "smart_wallet"
				data.EffectiveActorHref = "/account/" + normalizedCtx.Wallet.ContractID + "/smart"
			}
			if data.HumanTitle == "" {
				data.HumanTitle = "Smart wallet transaction"
			}
			if data.HumanNarrative == "" {
				data.HumanNarrative = fmt.Sprintf("This transaction was submitted by %s but executed through the %s smart wallet %s.", gateway.ShortAddress(data.SubmitterAddr), data.SmartWalletType, data.SmartWalletContractShort)
			}
			data.HumanEvidence = append(data.HumanEvidence,
				pages.TxHumanEvidence{Label: "Wallet involved", Value: "Yes"},
				pages.TxHumanEvidence{Label: "Wallet type", Value: data.SmartWalletType},
				pages.TxHumanEvidence{Label: "Effective actor", Value: "smart_wallet"},
				pages.TxHumanEvidence{Label: "Actor model source", Value: normalizedCtx.SourceOfTruth},
			)
			data.HumanSignals = append(data.HumanSignals,
				pages.TxHumanSignal{Title: "Smart wallet involved", Severity: "info", Summary: "Prism detected smart-wallet-mediated execution in this transaction."},
				pages.TxHumanSignal{Title: "Delegated execution", Severity: "info", Summary: "The submitting classic account is distinct from the effective wallet actor."},
			)
		}
		if len(data.HumanActors) == 0 {
			if data.SubmitterAddr != "" {
				data.HumanActors = append(data.HumanActors, pages.TxHumanActor{Label: data.SubmitterShort, Role: "Submitter", ActorType: "classic_account", Href: "/account/" + data.SubmitterAddr})
			}
			if data.EffectiveActorAddr != "" {
				data.HumanActors = append(data.HumanActors, pages.TxHumanActor{Label: data.EffectiveActorShort, Role: "Effective Actor", ActorType: data.EffectiveActorType, Href: data.EffectiveActorHref})
			}
			if data.DownstreamContractAddr != "" {
				data.HumanActors = append(data.HumanActors, pages.TxHumanActor{Label: data.DownstreamContractShort, Role: "Protocol", ActorType: "contract", Href: data.DownstreamContractHref})
			}
		}
		data.HumanEvidence = dedupeHumanEvidence(data.HumanEvidence)
		data.HumanSignals = dedupeHumanSignals(data.HumanSignals)
	}

	return data, nil
}

func dedupeHumanEvidence(items []pages.TxHumanEvidence) []pages.TxHumanEvidence {
	seen := map[string]bool{}
	out := make([]pages.TxHumanEvidence, 0, len(items))
	for _, item := range items {
		key := item.Label + ":" + item.Value
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, item)
	}
	return out
}

func dedupeHumanSignals(items []pages.TxHumanSignal) []pages.TxHumanSignal {
	seen := map[string]bool{}
	out := make([]pages.TxHumanSignal, 0, len(items))
	for _, item := range items {
		key := item.Title + ":" + item.Summary
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, item)
	}
	return out
}

// formatSorobanCall formats a Soroban invoke_host_function operation summary
// with decoded arguments, similar to stellar.expert's display:
//
//	mint(GCFB…TEKT, 10000000i128)
//
// buildOperationSummary produces a human-readable HTML summary for a single operation.
func buildOperationSummary(op gateway.DecodedOperation) string {
	source := html.EscapeString(gateway.ShortAddress(op.SourceAccount))

	// Soroban contract calls — use decoded args when available.
	if op.FunctionName != "" && op.ArgumentsJSON != "" {
		return formatSorobanCall(op.SourceAccount, op.FunctionName, op.ArgumentsJSON)
	}
	if op.FunctionName != "" {
		return fmt.Sprintf(`<span class="font-medium text-text-primary">%s</span> called <span class="font-mono text-violet-600 dark:text-violet-400">%s()</span>`,
			source, html.EscapeString(op.FunctionName))
	}

	displayName := strings.ToLower(gateway.OperationDisplayName(op.TypeName))
	dest := html.EscapeString(gateway.ShortAddress(op.Destination))
	asset := op.AssetCode
	if asset == "" {
		asset = "XLM"
	}

	// Classic operations — use available fields for richer summaries.
	// Note: op.Amount from the API is in stroops (7 decimal places).
	fmtAmount := gateway.FormatStroopsToXLM(op.Amount)

	switch op.TypeName {
	case "create_account", "OperationTypeCreateAccount":
		return fmt.Sprintf(`Created account <span class="font-mono text-text-primary">%s</span> with <span class="font-semibold">%s XLM</span>`, dest, fmtAmount)
	case "payment", "OperationTypePayment":
		if op.Amount != "" && op.Destination != "" {
			return fmt.Sprintf(`Sent <span class="font-semibold">%s %s</span> to <span class="font-mono text-text-primary">%s</span>`,
				fmtAmount, html.EscapeString(asset), dest)
		}
	case "account_merge", "OperationTypeAccountMerge":
		if op.Destination != "" {
			return fmt.Sprintf(`<span class="font-mono text-text-primary">%s</span> merged account into <span class="font-mono text-text-primary">%s</span>`, source, dest)
		}
	case "change_trust", "OperationTypeChangeTrust":
		if op.AssetCode != "" {
			return fmt.Sprintf(`<span class="font-mono text-text-primary">%s</span> added trustline for <span class="font-semibold">%s</span>`, source, html.EscapeString(op.AssetCode))
		}
	case "manage_sell_offer", "OperationTypeManageSellOffer":
		if op.Amount != "" {
			return fmt.Sprintf(`<span class="font-mono text-text-primary">%s</span> sell offer: <span class="font-semibold">%s %s</span>`,
				source, fmtAmount, html.EscapeString(asset))
		}
	case "manage_buy_offer", "OperationTypeManageBuyOffer":
		if op.Amount != "" {
			return fmt.Sprintf(`<span class="font-mono text-text-primary">%s</span> buy offer: <span class="font-semibold">%s %s</span>`,
				source, fmtAmount, html.EscapeString(asset))
		}
	case "set_options", "OperationTypeSetOptions":
		return fmt.Sprintf(`<span class="font-mono text-text-primary">%s</span> updated account options`, source)
	case "manage_data", "OperationTypeManageData":
		return fmt.Sprintf(`<span class="font-mono text-text-primary">%s</span> updated data entry`, source)
	case "set_trust_line_flags", "OperationTypeSetTrustLineFlags":
		if op.Destination != "" {
			return fmt.Sprintf(`<span class="font-mono text-text-primary">%s</span> set trust flags for <span class="font-mono text-text-primary">%s</span>`, source, dest)
		}
	}

	// Fallback: source + display name + optional destination.
	if op.Destination != "" {
		return fmt.Sprintf(`<span class="font-medium text-text-primary">%s</span> %s <span class="font-mono text-text-primary">%s</span>`,
			source, html.EscapeString(displayName), dest)
	}
	return fmt.Sprintf(`<span class="font-medium text-text-primary">%s</span> %s`, source, html.EscapeString(displayName))
}

// buildEffectSummary produces a human-readable HTML summary for a single effect.
func buildEffectSummary(eff gateway.TransactionEffect) string {
	acct := html.EscapeString(gateway.ShortAddress(eff.AccountID))
	asset := "XLM"
	if eff.Asset != nil && eff.Asset.Code != "" {
		asset = eff.Asset.Code
	}
	// Amount is already decimal from the API (e.g. "10000.0000000").
	amount := gateway.FormatDecimalAmount(eff.Amount)

	// Helper to read string values from details map.
	detailStr := func(key string) string {
		if eff.Details == nil {
			return ""
		}
		if v, ok := eff.Details[key]; ok {
			return fmt.Sprintf("%v", v)
		}
		return ""
	}

	switch eff.EffectTypeString {
	// Account
	case "account_created":
		bal := gateway.FormatDecimalAmount(detailStr("starting_balance"))
		if bal != "" {
			return fmt.Sprintf(`<span class="font-mono text-text-primary">%s</span> was created with <span class="font-semibold">%s XLM</span>`, acct, bal)
		}
		return fmt.Sprintf(`<span class="font-mono text-text-primary">%s</span> was created`, acct)
	case "account_removed":
		return fmt.Sprintf(`<span class="font-mono text-text-primary">%s</span> was removed`, acct)
	case "account_credited", "contract_credited":
		return fmt.Sprintf(`<span class="font-mono text-text-primary">%s</span> received <span class="font-semibold text-emerald-600 dark:text-emerald-400">+%s %s</span>`, acct, amount, html.EscapeString(asset))
	case "account_debited", "contract_debited":
		return fmt.Sprintf(`<span class="font-mono text-text-primary">%s</span> sent <span class="font-semibold text-red-600 dark:text-red-400">-%s %s</span>`, acct, amount, html.EscapeString(asset))
	case "account_thresholds_updated":
		return fmt.Sprintf(`<span class="font-mono text-text-primary">%s</span> updated thresholds`, acct)
	case "account_home_domain_updated":
		domain := detailStr("home_domain")
		if domain != "" {
			return fmt.Sprintf(`<span class="font-mono text-text-primary">%s</span> set home domain to <span class="font-semibold">%s</span>`, acct, html.EscapeString(domain))
		}
		return fmt.Sprintf(`<span class="font-mono text-text-primary">%s</span> updated home domain`, acct)
	case "account_flags_updated":
		return fmt.Sprintf(`<span class="font-mono text-text-primary">%s</span> updated account flags`, acct)

	// Signer
	case "signer_created":
		key := gateway.ShortAddress(detailStr("public_key"))
		return fmt.Sprintf(`Signer <span class="font-mono text-text-primary">%s</span> added to <span class="font-mono text-text-primary">%s</span>`, html.EscapeString(key), acct)
	case "signer_removed":
		key := gateway.ShortAddress(detailStr("public_key"))
		return fmt.Sprintf(`Signer <span class="font-mono text-text-primary">%s</span> removed from <span class="font-mono text-text-primary">%s</span>`, html.EscapeString(key), acct)
	case "signer_updated":
		key := gateway.ShortAddress(detailStr("public_key"))
		return fmt.Sprintf(`Signer <span class="font-mono text-text-primary">%s</span> updated on <span class="font-mono text-text-primary">%s</span>`, html.EscapeString(key), acct)

	// Trustline
	case "trustline_created":
		return fmt.Sprintf(`<span class="font-mono text-text-primary">%s</span> added trustline for <span class="font-semibold">%s</span>`, acct, html.EscapeString(asset))
	case "trustline_removed":
		return fmt.Sprintf(`<span class="font-mono text-text-primary">%s</span> removed trustline for <span class="font-semibold">%s</span>`, acct, html.EscapeString(asset))
	case "trustline_updated":
		return fmt.Sprintf(`<span class="font-mono text-text-primary">%s</span> updated trustline for <span class="font-semibold">%s</span>`, acct, html.EscapeString(asset))
	case "trustline_flags_updated":
		return fmt.Sprintf(`<span class="font-mono text-text-primary">%s</span> updated trust flags`, acct)

	// DEX
	case "trade":
		return fmt.Sprintf(`<span class="font-mono text-text-primary">%s</span> traded <span class="font-semibold">%s %s</span>`, acct, amount, html.EscapeString(asset))
	case "offer_created":
		return fmt.Sprintf(`<span class="font-mono text-text-primary">%s</span> created offer`, acct)
	case "offer_removed":
		return fmt.Sprintf(`<span class="font-mono text-text-primary">%s</span> removed offer`, acct)
	case "offer_updated":
		return fmt.Sprintf(`<span class="font-mono text-text-primary">%s</span> updated offer`, acct)

	// Data
	case "data_created":
		name := detailStr("name")
		if name != "" {
			return fmt.Sprintf(`<span class="font-mono text-text-primary">%s</span> created data entry <span class="font-semibold">%s</span>`, acct, html.EscapeString(name))
		}
		return fmt.Sprintf(`<span class="font-mono text-text-primary">%s</span> created data entry`, acct)
	case "data_removed":
		name := detailStr("name")
		if name != "" {
			return fmt.Sprintf(`<span class="font-mono text-text-primary">%s</span> removed data entry <span class="font-semibold">%s</span>`, acct, html.EscapeString(name))
		}
		return fmt.Sprintf(`<span class="font-mono text-text-primary">%s</span> removed data entry`, acct)
	case "data_updated":
		name := detailStr("name")
		if name != "" {
			return fmt.Sprintf(`<span class="font-mono text-text-primary">%s</span> updated data entry <span class="font-semibold">%s</span>`, acct, html.EscapeString(name))
		}
		return fmt.Sprintf(`<span class="font-mono text-text-primary">%s</span> updated data entry`, acct)
	case "sequence_bumped":
		return fmt.Sprintf(`<span class="font-mono text-text-primary">%s</span> bumped sequence`, acct)

	// Claimable Balance
	case "claimable_balance_created", "claimable_balance_claimant_created":
		return fmt.Sprintf(`<span class="font-mono text-text-primary">%s</span> created claimable balance of <span class="font-semibold">%s %s</span>`, acct, amount, html.EscapeString(asset))
	case "claimable_balance_claimed":
		return fmt.Sprintf(`<span class="font-mono text-text-primary">%s</span> claimed <span class="font-semibold">%s %s</span>`, acct, amount, html.EscapeString(asset))
	case "claimable_balance_clawed_back":
		return fmt.Sprintf(`Claimable balance clawed back from <span class="font-mono text-text-primary">%s</span>`, acct)

	// Liquidity Pool
	case "liquidity_pool_deposited":
		return fmt.Sprintf(`<span class="font-mono text-text-primary">%s</span> deposited into liquidity pool`, acct)
	case "liquidity_pool_withdrew":
		return fmt.Sprintf(`<span class="font-mono text-text-primary">%s</span> withdrew from liquidity pool`, acct)
	case "liquidity_pool_trade":
		return fmt.Sprintf(`<span class="font-mono text-text-primary">%s</span> pool trade`, acct)
	case "liquidity_pool_created":
		return fmt.Sprintf(`Liquidity pool created by <span class="font-mono text-text-primary">%s</span>`, acct)
	case "liquidity_pool_removed":
		return fmt.Sprintf(`Liquidity pool removed`)

	// Soroban
	case "extend_footprint_ttl":
		return fmt.Sprintf(`<span class="font-mono text-text-primary">%s</span> extended TTL`, acct)
	case "restore_footprint":
		return fmt.Sprintf(`<span class="font-mono text-text-primary">%s</span> restored footprint`, acct)

	default:
		return fmt.Sprintf(`<span class="font-mono text-text-primary">%s</span> %s`, acct, html.EscapeString(gateway.EffectDisplayName(eff.EffectTypeString)))
	}
}

func formatSorobanCall(source, functionName, argsJSON string) string {
	// Parse the arguments JSON array.
	var args []map[string]any
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return fmt.Sprintf(`<span class="font-medium text-gray-900">%s</span> called <span class="font-mono text-violet-600">%s</span>`,
			html.EscapeString(gateway.ShortAddress(source)), html.EscapeString(functionName))
	}

	// Format each argument concisely.
	formattedArgs := make([]string, 0, len(args))
	for _, arg := range args {
		formattedArgs = append(formattedArgs, html.EscapeString(formatScVal(arg)))
	}

	argsStr := ""
	for i, a := range formattedArgs {
		if i > 0 {
			argsStr += ", "
		}
		argsStr += `<span class="font-mono text-xs">` + a + `</span>`
	}

	return fmt.Sprintf(`<span class="font-medium text-gray-900">%s</span> <span class="font-mono text-violet-600">%s</span>(%s)`,
		html.EscapeString(gateway.ShortAddress(source)),
		html.EscapeString(functionName),
		argsStr)
}

// formatScVal formats a decoded Soroban ScVal for display.
func formatScVal(val map[string]any) string {
	valType, _ := val["type"].(string)
	switch valType {
	case "account", "contract":
		if addr, ok := val["address"].(string); ok {
			return gateway.ShortAddress(addr)
		}
	case "i128", "u128":
		if v, ok := val["value"].(string); ok {
			return v + valType
		}
	case "i64", "u64", "i32", "u32":
		if v, ok := val["value"].(string); ok {
			return v
		}
		if v, ok := val["value"].(float64); ok {
			return fmt.Sprintf("%.0f", v)
		}
	case "bool":
		if v, ok := val["value"].(bool); ok {
			if v {
				return "true"
			}
			return "false"
		}
	case "symbol":
		if v, ok := val["value"].(string); ok {
			return v
		}
	}
	if v, ok := val["value"]; ok {
		return fmt.Sprintf("%v", v)
	}
	return fmt.Sprintf("%v", val)
}
