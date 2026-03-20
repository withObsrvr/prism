package handlers

import (
	"fmt"
	"html"
	"net/http"
	"sort"
	"time"

	"github.com/withObsrvr/prism/internal/gateway"
	"github.com/withObsrvr/prism/internal/templates/pages"
)

// Home renders the search-first landing page.
// Uses mock data for everything except the latest ledger number, which is
// fetched live from the gateway when available (currently testnet only).
func (h *Handlers) Home(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	network := networkFromRequest(r)
	data := mockHomeData(network)

	// Overlay live latest ledger if gateway is available.
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
				Sequence: gateway.FormatNumber(l.Sequence),
				Age:      age,
				TxCount:  fmt.Sprintf("%d txs", l.SuccessfulTxCount),
				OpCount:  fmt.Sprintf("%d ops", l.OperationCount),
				IsLatest: i == 0,
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

	data := mockSearchData(query)

	if err := pages.Search(data).Render(r.Context(), w); err != nil {
		h.Logger.Error("render search", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
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
		order := map[string]int{"ledger": 0, "transaction": 1, "account": 2, "contract": 3, "asset": 4}
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

// SearchResults returns an HTML fragment for live search.
func (h *Handlers) SearchResults(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		w.WriteHeader(http.StatusOK)
		return
	}
	fmt.Fprintf(w, `<div class="search-results">Results for: %s</div>`, query)
}

// LedgerDetail renders a single ledger page.
func (h *Handlers) LedgerDetail(w http.ResponseWriter, r *http.Request) {
	sequence := r.PathValue("sequence")
	data := mockLedgerDetailData(sequence)

	if err := pages.LedgerDetail(data).Render(r.Context(), w); err != nil {
		h.Logger.Error("render ledger detail", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// buildLedgerDetailData fetches a single ledger and its transactions from the gateway.
func (h *Handlers) buildLedgerDetailData(r *http.Request, network, sequence string) (pages.LedgerDetailData, error) {
	ctx := r.Context()

	// Parse the sequence number.
	var seq int64
	if _, err := fmt.Sscanf(sequence, "%d", &seq); err != nil {
		return pages.LedgerDetailData{}, fmt.Errorf("invalid sequence: %s", sequence)
	}

	// Fetch the single ledger.
	ledgers, err := h.Gateway.GetLedgers(ctx, network, seq, seq, 1, "asc")
	if err != nil {
		return pages.LedgerDetailData{}, fmt.Errorf("fetching ledger %d: %w", seq, err)
	}
	if len(ledgers) == 0 {
		return pages.LedgerDetailData{}, fmt.Errorf("ledger %d not found", seq)
	}

	l := ledgers[0]

	// Fetch transactions and operations for this ledger.
	txs, txErr := h.Gateway.GetTransactions(ctx, network, seq, seq, 50, "asc")
	ops, opsErr := h.Gateway.GetOperations(ctx, network, seq, seq, 200)

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
		Sequence:     gateway.FormatNumber(l.Sequence),
		PrevSequence: gateway.FormatNumber(l.Sequence - 1),
		NextSequence: gateway.FormatNumber(l.Sequence + 1),
		ClosedAt:     closedAt,
		CloseTime:    closeTime,
		Hash:         gateway.ShortHash(l.LedgerHash),
		PrevHash:     gateway.ShortHash(l.PreviousLedgerHash),
		Protocol:     fmt.Sprintf("%d", l.ProtocolVersion),
		BaseFee:      gateway.FormatNumber(l.BaseFee),
		MaxTxSetSize: fmt.Sprintf("%d", l.MaxTxSetSize),
		TotalCoins:   totalCoins,
		TxCount:      fmt.Sprintf("%d", l.TransactionCount),
		TxSuccess:    fmt.Sprintf("%d", l.SuccessfulTxCount),
		TxFailed:     fmt.Sprintf("%d", l.FailedTxCount),
		OpCount:      fmt.Sprintf("%d", l.OperationCount),
		OpsPerTx:     fmt.Sprintf("%.1f", opsPerTx),
		TotalFees:    func() string { if l.TotalFeeCharged != nil { return gateway.FormatNumber(*l.TotalFeeCharged) }; return gateway.FormatNumber(l.BaseFee * int64(l.TransactionCount)) }(),
		// Fields not available from bronze — use defaults
		SorobanCalls:  "—",
		SorobanPct:    "—",
		FeesUSD:       "—",
		EventsEmitted: "—",
		TotalCPU:      "—",
		StateReads:    "—",
		StateReadKB:   "—",
		StateWrites:   "—",
		StateWriteKB:  "—",
		RentBurned:    "—",
		FeeBase:       gateway.FormatNumber(l.BaseFee),
		FeeMedian:     "—",
		FeeP99:        "—",
		SurgePct:      "—",
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

	// Transform transactions.
	if txErr == nil && len(txs) > 0 {
		ledgerTxs := make([]pages.LedgerTx, 0, len(txs))
		for i, tx := range txs {
			status := "ok"
			if !tx.Successful {
				status = "failed"
			}
			opType := "tx"
			opColor := "gray"
			if tx.OperationCount > 1 {
				opType = "multi"
				opColor = "violet"
			}

			summary := fmt.Sprintf(`<span class="font-medium text-gray-900">%s</span> · %d op(s)`,
				html.EscapeString(gateway.ShortAddress(tx.SourceAccount)), tx.OperationCount)

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

	data := mockTxReceiptData(hash, shortHash)

	if err := pages.TransactionReceiptV2(data).Render(r.Context(), w); err != nil {
		h.Logger.Error("render transaction receipt", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
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

// buildTxReceiptData fetches real transaction data from the gateway.
func (h *Handlers) buildTxReceiptData(r *http.Request, network, hash, shortHash string) (pages.TxReceiptData, error) {
	ctx := r.Context()

	// Fetch full decoded transaction and diffs concurrently.
	txFull, fullErr := h.Gateway.GetTransactionFull(ctx, network, hash)
	txDiffs, diffsErr := h.Gateway.GetTransactionDiffs(ctx, network, hash)

	if fullErr != nil {
		return pages.TxReceiptData{}, fmt.Errorf("fetching tx full: %w", fullErr)
	}

	tx := txFull.Transaction
	summary := txFull.Summary

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
	if t, err := time.Parse(time.RFC3339, tx.ClosedAt); err == nil {
		timestamp = t.Format("Jan 2, 2006 at 15:04:05 UTC")
	}

	// Build summary HTML from the summary description.
	summaryHTML := html.EscapeString(summary.Description)

	// Extract flow diagram info from summary.
	sourceAddr := "—"
	sourceAmount := "—"
	contractName := "—"
	contractAddr := "—"
	contractFn := "—"
	destAddr := "—"
	destAmount := "—"
	effectiveRate := "—"
	slippage := "—"
	route := "—"

	if summary.Transfer != nil {
		sourceAddr = gateway.ShortAddress(summary.Transfer.From)
		sourceAmount = summary.Transfer.Amount + " " + summary.Transfer.Asset
		destAddr = gateway.ShortAddress(summary.Transfer.To)
		destAmount = "+" + summary.Transfer.Amount + " " + summary.Transfer.Asset
	}
	if summary.Swap != nil {
		sourceAmount = summary.Swap.AmountIn + " " + summary.Swap.AssetIn
		destAmount = "+" + summary.Swap.AmountOut + " " + summary.Swap.AssetOut
		route = summary.Swap.AssetIn + " -> " + summary.Swap.AssetOut
	}
	if len(txFull.Operations) > 0 {
		op := txFull.Operations[0]
		if op.ContractID != "" {
			contractAddr = gateway.ShortAddress(op.ContractID)
			contractName = contractAddr
		}
		if op.FunctionName != "" {
			contractFn = op.FunctionName + "()"
		}
		if op.SourceAccount != "" {
			sourceAddr = gateway.ShortAddress(op.SourceAccount)
		}
	}

	// Build operations list.
	ops := make([]pages.TxOperation, 0, len(txFull.Operations))
	for _, op := range txFull.Operations {
		opStatus := "Success"
		if !tx.Successful {
			opStatus = "Failed"
		}
		ops = append(ops, pages.TxOperation{
			Index:       fmt.Sprintf("%d", op.Index+1),
			Type:        op.TypeName,
			IsSoroban:   op.IsSorobanOp,
			IsPrimary:   op.Index == 0,
			Status:      opStatus,
			SummaryHTML: fmt.Sprintf(`<span class="font-medium text-gray-900">%s</span> %s`, html.EscapeString(gateway.ShortAddress(op.SourceAccount)), html.EscapeString(op.TypeName)),
			Contract:    gateway.ShortAddress(op.ContractID),
			Function:    op.FunctionName,
		})
	}

	// Build events list.
	evts := make([]pages.TxEvent, 0, len(txFull.Events))
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
			dataHTML = fmt.Sprintf(`<span class="font-medium">%s %s</span>`, html.EscapeString(evt.Amount), html.EscapeString(evt.AssetCode))
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

	// Build balance changes from diffs.
	var balChanges []pages.TxBalanceChange
	if diffsErr == nil && txDiffs != nil {
		for _, bc := range txDiffs.BalanceChanges {
			isPositive := true
			if len(bc.Delta) > 0 && bc.Delta[0] == '-' {
				isPositive = false
			}
			assetType := bc.AssetType
			typeColor := "gray"
			if bc.AssetType == "credit_alphanum4" || bc.AssetType == "credit_alphanum12" {
				assetType = "Classic"
			}
			if bc.AssetCode == "" {
				bc.AssetCode = "XLM"
				assetType = "Native"
			}
			balChanges = append(balChanges, pages.TxBalanceChange{
				Account:    gateway.ShortAddress(bc.Address),
				Asset:      bc.AssetCode,
				AssetType:  assetType,
				TypeColor:  typeColor,
				Change:     bc.Delta,
				IsPositive: isPositive,
			})
		}
	}

	// Build state changes from diffs.
	var stateChanges []pages.TxStateChange
	if diffsErr == nil && txDiffs != nil {
		for _, sc := range txDiffs.StateChanges {
			stateChanges = append(stateChanges, pages.TxStateChange{
				Action:     sc.Type,
				Key:        sc.Key,
				Contract:   sc.EntryType,
				DetailHTML: fmt.Sprintf(`<span class="text-gray-400">%s</span> → <span class="text-gray-900 font-medium">%s</span>`, html.EscapeString(sc.Before), html.EscapeString(sc.After)),
			})
		}
	}

	data := pages.TxReceiptData{
		Hash:          hash,
		ShortHash:     shortHash,
		Status:        status,
		IsSoroban:     isSoroban,
		HasClassicOps: hasClassic,
		SummaryHTML:   summaryHTML,
		Timestamp:     timestamp,
		Ledger:        gateway.FormatNumber(tx.LedgerSequence),
		OpsCount:      fmt.Sprintf("%d", tx.OperationCount),
		EventsCount:   fmt.Sprintf("%d", len(txFull.Events)),
		SourceAddr:    sourceAddr,
		SourceAmount:  sourceAmount,
		ContractName:  contractName,
		ContractAddr:  contractAddr,
		ContractFn:    contractFn,
		DestAddr:      destAddr,
		DestAmount:    destAmount,
		EffectiveRate: effectiveRate,
		Slippage:      slippage,
		Route:         route,
		FeePaid:       gateway.FormatNumber(tx.Fee),
		FeeUSD:        "—",
		SorobanCPU:    "—",
		SorobanMem:    "—",
		SorobanReads:  "—",
		SorobanWrites: "—",
		SeqNumber:     "—",
		Operations:    ops,
		Events:        evts,
		BalanceChanges: balChanges,
		StateChanges:  stateChanges,
	}

	return data, nil
}
