package handlers

import (
	"fmt"
	"net/http"

	"github.com/withObsrvr/prism/internal/events"
	"github.com/withObsrvr/prism/internal/templates/pages"
)

// Home renders the search-first landing page.
func (h *Handlers) Home(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	data := pages.HomeData{
		Network:       h.Network,
		LatestLedger:  "61,504,113",
		LedgerAge:     "3.8s ago",
		TxCount24H:    "1,284,392",
		TxChange:      "↑ 12.3%",
		TPSAvg:        "14.8",
		TPSPeak:       "142",
		SorobanCalls:  "847,201",
		SorobanChange: "↑ 28.7%",
		Validators:    42,
		FeeEconomy:    "100",
		FeeStandard:   "1,200",
		FeePriority:   "34,000",
		SurgeActive:   false,
		SurgeContext:  "Network is uncongested",
		Transactions: []pages.HomeTx{
			{Hash: "8f2a7c1b3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c2d3e4f5a6b7c8d1b3c", ShortHash: "8f2a7c...1b3c", Type: "contract_call", TypeLabel: "Contract Call", Summary: "Soroswap Router • swap()", From: "GBXK4...R2M7", To: "CDLZ9...WK42", Ops: "3 ops", Fee: "0.012 XLM", Age: "4s ago"},
			{Hash: "a91e03fc824d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c2d3e4f5a6b7c8dfc82", ShortHash: "a91e03...fc82", Type: "payment", TypeLabel: "Payment", Summary: "500.00 USDC → Circle hot wallet", From: "GCKW2...NP4X", To: "GDQP7...H93A", Ops: "1 op", Fee: "0.001 XLM", Age: "8s ago"},
			{Hash: "3d72b1e4a74d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c2d3e4f5a6b7c8de4a7", ShortHash: "3d72b1...e4a7", Type: "contract_call", TypeLabel: "Contract Call", Summary: "Blend Protocol • supply()", From: "GA7TN...QE5L", To: "CBTRE...P8N1", Ops: "5 ops", Fee: "0.034 XLM", Age: "12s ago"},
			{Hash: "c504a829d14d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c2d3e4f5a6b7c8d29d1", ShortHash: "c504a8...29d1", Type: "account_merge", TypeLabel: "Account Merge", Summary: "Merge into GDQP7...H93A", From: "GCVN8...WT3R", To: "GDQP7...H93A", Ops: "1 op", Fee: "0.001 XLM", Age: "15s ago"},
			{Hash: "71f3e28bc04d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c2d3e4f5a6b7c8d8bc0", ShortHash: "71f3e2...8bc0", Type: "contract_call", TypeLabel: "Contract Call", Summary: "Phoenix DEX • provide_liquidity()", From: "GBXR5...KM92", To: "CDNS4...A7Q3", Ops: "4 ops", Fee: "0.028 XLM", Age: "19s ago"},
			{Hash: "e28f914d564d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c2d3e4f5a6b7c8d4d56", ShortHash: "e28f91...4d56", Type: "payment", TypeLabel: "Payment", Summary: "12,450.00 yXLM → Yield distributor", From: "GAUTH...B4KQ", To: "GCYLD...M7R2", Ops: "2 ops", Fee: "0.002 XLM", Age: "23s ago"},
		},
		Ledgers: []pages.HomeLedger{
			{Sequence: "61,504,113", Age: "3.8s", TxCount: "42 txs", OpCount: "14 ops", IsLatest: true},
			{Sequence: "61,504,112", Age: "8.9s", TxCount: "38 txs", OpCount: "12 ops"},
			{Sequence: "61,504,111", Age: "14.2s", TxCount: "51 txs", OpCount: "18 ops"},
			{Sequence: "61,504,110", Age: "19.6s", TxCount: "29 txs", OpCount: "9 ops"},
			{Sequence: "61,504,109", Age: "24.8s", TxCount: "44 txs", OpCount: "15 ops"},
			{Sequence: "61,504,108", Age: "30.1s", TxCount: "36 txs", OpCount: "11 ops"},
			{Sequence: "61,504,107", Age: "35.4s", TxCount: "47 txs", OpCount: "16 ops"},
			{Sequence: "61,504,106", Age: "40.7s", TxCount: "33 txs", OpCount: "10 ops"},
		},
		Contracts: []pages.HomeContract{
			{Rank: 1, Name: "Soroswap Router", Tag: "DEX", TagColor: "violet", Address: "CDLZ9...WK42", Invocations: "284,102", Change: "+18%", IsPositive: true},
			{Rank: 2, Name: "Blend Protocol", Tag: "Lending", TagColor: "cyan", Address: "CBTRE...P8N1", Invocations: "198,433", Change: "+42%", IsPositive: true},
			{Rank: 3, Name: "Phoenix DEX", Tag: "DEX", TagColor: "violet", Address: "CDNS4...A7Q3", Invocations: "156,891", Change: "+7%", IsPositive: true},
			{Rank: 4, Name: "Aquarius AMM", Tag: "DEX", TagColor: "violet", Address: "CAUQR...L2B5", Invocations: "134,220", Change: "-3%", IsPositive: false},
			{Rank: 5, Name: "FxDAO Vault", Tag: "Stablecoin", TagColor: "emerald", Address: "CCWN7...QR41", Invocations: "89,744", Change: "+31%", IsPositive: true},
		},
		Assets: []pages.HomeAsset{
			{Code: "XLM", Issuer: "Native", Initial: "X", BgColor: "bg-gray-900", Volume: "$12.4M", Change: "+5.2%", IsPositive: true},
			{Code: "USDC", Issuer: "Centre", Initial: "U", BgColor: "bg-blue-600", Volume: "$8.7M", Change: "+1.8%", IsPositive: true},
			{Code: "yXLM", Issuer: "Ultra Capital", Initial: "y", BgColor: "bg-emerald-600", Volume: "$3.2M", Change: "+14.1%", IsPositive: true},
			{Code: "AQUA", Issuer: "Aquarius", Initial: "A", BgColor: "bg-cyan-500", Volume: "$1.8M", Change: "-2.4%", IsPositive: false},
			{Code: "SHX", Issuer: "Stronghold", Initial: "S", BgColor: "bg-orange-500", Volume: "$924K", Change: "+8.7%", IsPositive: true},
		},
	}

	if err := pages.Home(data).Render(r.Context(), w); err != nil {
		h.Logger.Error("render home", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// Search renders the full search results page.
func (h *Handlers) Search(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	data := pages.SearchData{
		Query:          query,
		DetectedType:   "account",
		DetectedLabel:  "Looks like a Stellar account address",
		DetectedDesc:   "Starts with G + alphanumeric characters. Redirecting to account view.",
		DetectedHref:   "/account/" + query,
		RecentSearches: []string{"Soroswap Router", "GABC...7X92", "8f2a...1b3c", "USDC", "Ledger 5,104,938"},
		Results: []pages.SearchResultGroup{
			{Category: "Contracts", Items: []pages.SearchResultItem{
				{Name: "Soroswap Router", Badge: "DEX", BadgeColor: "violet", Subtitle: "CAXY...Z10P · 284K invocations", IconBg: "bg-violet-100", IconHTML: `<svg class="h-4 w-4 text-violet-600" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 20l4-16m4 4l4 4-4 4M6 16l-4-4 4-4"></path></svg>`, Href: "/contracts/CAXY"},
				{Name: "Soroswap Factory", Badge: "DEX", BadgeColor: "violet", Subtitle: "CFACT...R2K8 · 12K invocations", IconBg: "bg-violet-100", IconHTML: `<svg class="h-4 w-4 text-violet-600" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 20l4-16m4 4l4 4-4 4M6 16l-4-4 4-4"></path></svg>`, Href: "/contracts/CFACT"},
			}},
			{Category: "Accounts", Items: []pages.SearchResultItem{
				{Name: "GABC...7X92", Badge: "Funded", BadgeColor: "emerald", Subtitle: "$58,247 total value · 12 trustlines", IconBg: "bg-gray-100", IconHTML: `<svg class="h-4 w-4 text-gray-500" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z"></path></svg>`, Href: "/account/GABC"},
			}},
			{Category: "Assets", Items: []pages.SearchResultItem{
				{Name: "USDC", Badge: "Classic", BadgeColor: "gray", Subtitle: "USD Coin · Centre · $142M market cap", IconBg: "bg-blue-100", IconHTML: `<svg class="h-4 w-4 text-blue-600" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8c-1.657 0-3 .895-3 2s1.343 2 3 2 3 .895 3 2-1.343 2-3 2m0-8c1.11 0 2.08.402 2.599 1M12 8V7m0 1v8m0 0v1m0-1c-1.11 0-2.08-.402-2.599-1M21 12a9 9 0 11-18 0 9 9 0 0118 0z"></path></svg>`, Href: "/assets/USDC-Centre"},
				{Name: "yUSDC", Badge: "SEP-41", BadgeColor: "violet", Subtitle: "Blend USDC · $84M market cap", IconBg: "bg-emerald-100", IconHTML: `<svg class="h-4 w-4 text-emerald-600" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8c-1.657 0-3 .895-3 2s1.343 2 3 2 3 .895 3 2-1.343 2-3 2m0-8c1.11 0 2.08.402 2.599 1M12 8V7m0 1v8m0 0v1m0-1c-1.11 0-2.08-.402-2.599-1M21 12a9 9 0 11-18 0 9 9 0 0118 0z"></path></svg>`, Href: "/assets/yUSDC-Blend"},
			}},
		},
	}

	if err := pages.Search(data).Render(r.Context(), w); err != nil {
		h.Logger.Error("render search", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
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

	data := pages.LedgerDetailData{
		Sequence:      sequence,
		PrevSequence:  "5,104,937",
		NextSequence:  "5,104,939",
		ClosedAt:      "Mar 2, 2026 · 14:32:18 UTC",
		CloseTime:     "5.1s",
		Hash:          "a1b2c3d4e5f6...7890abcdef12",
		PrevHash:      "9f8e7d6c5b4a...3210fedcba98",
		Protocol:      "22",
		BaseFee:       "100",
		MaxTxSetSize:  "200",
		TotalCoins:    "105,443,902,087.3472865 XLM",
		TxCount:       "47",
		TxSuccess:     "46",
		TxFailed:      "1",
		OpCount:       "108",
		OpsPerTx:      "2.3",
		SorobanCalls:  "18",
		SorobanPct:    "38%",
		TotalFees:     "1,842",
		FeesUSD:       "$0.18",
		EventsEmitted: "62",
		TotalCPU:      "412M",
		StateReads:    "84",
		StateReadKB:   "168 KB",
		StateWrites:   "36",
		StateWriteKB:  "42 KB",
		RentBurned:    "24.8",
		OpBreakdown: []pages.OpBreakdownItem{
			{Name: "Invoke Contract", Count: "37", Pct: "34%", Width: "34%", Color: "bg-violet-500"},
			{Name: "Payment", Count: "30", Pct: "28%", Width: "28%", Color: "bg-cyan-500"},
			{Name: "Path Payment", Count: "15", Pct: "14%", Width: "14%", Color: "bg-emerald-500"},
			{Name: "Manage Offer", Count: "11", Pct: "10%", Width: "10%", Color: "bg-amber-500"},
			{Name: "Create Account", Count: "9", Pct: "8%", Width: "8%", Color: "bg-gray-400"},
			{Name: "Other", Count: "6", Pct: "6%", Width: "6%", Color: "bg-gray-300"},
		},
		Transactions: []pages.LedgerTx{
			{Index: "1", Status: "ok", Hash: "8f2a1b3c", ShortHash: "8f2a...1b3c", OpType: "swap", OpColor: "violet", Summary: `<span class="font-medium text-gray-900">GABC...7X</span> swapped <span class="font-semibold text-red-600">5,000 XLM</span> for <span class="font-semibold text-emerald-600">485 USDC</span> via Soroswap`, Ops: "3", Fee: "10,200"},
			{Index: "2", Status: "ok", Hash: "c4e93d0f", ShortHash: "c4e9...3d0f", OpType: "transfer", OpColor: "cyan", Summary: `<span class="font-medium text-gray-900">GDEF...2P</span> sent <span class="font-semibold text-gray-900">2,500 USDC</span> to <span class="font-medium text-gray-900">GKLM...1V</span>`, Ops: "1", Fee: "100"},
			{Index: "3", Status: "ok", Hash: "a1b28e4f", ShortHash: "a1b2...8e4f", OpType: "swap", OpColor: "violet", Summary: `<span class="font-medium text-gray-900">GNOP...3W</span> swapped <span class="font-semibold text-red-600">12,400 XLM</span> for <span class="font-semibold text-emerald-600">1,202 USDC</span> via Soroswap`, Ops: "2", Fee: "8,400"},
			{Index: "4", Status: "ok", Hash: "f7d24c1a", ShortHash: "f7d2...4c1a", OpType: "mint", OpColor: "emerald", Summary: `<span class="font-medium text-gray-900">Blend Protocol</span> minted <span class="font-semibold text-emerald-600">45,000 BLND</span> emission to <span class="font-medium text-gray-900">GDEF...9R</span>`, Ops: "1", Fee: "5,200"},
			{Index: "5", Status: "ok", Hash: "9e8f2a5b", ShortHash: "9e8f...2a5b", OpType: "invoke", OpColor: "violet", Summary: `<span class="font-medium text-gray-900">GHIJ...2M</span> called <span class="font-mono text-violet-600">deposit()</span> on <span class="font-medium text-gray-900">Blend Pool</span> with <span class="font-semibold text-gray-900">10,000 USDC</span>`, Ops: "2", Fee: "12,800"},
			{Index: "6", Status: "ok", Hash: "d4c37f8e", ShortHash: "d4c3...7f8e", OpType: "payment", OpColor: "cyan", Summary: `<span class="font-medium text-gray-900">GQRS...5X</span> sent <span class="font-semibold text-gray-900">8,400 EURC</span> to <span class="font-medium text-gray-900">GTUV...7Y</span>`, Ops: "1", Fee: "100"},
			{Index: "7", Status: "ok", Hash: "b8a71d2c", ShortHash: "b8a7...1d2c", OpType: "path pay", OpColor: "emerald", Summary: `<span class="font-medium text-gray-900">GWXY...9Z</span> path payment: <span class="font-semibold text-red-600">1,000 XLM</span> to deliver <span class="font-semibold text-emerald-600">92.40 EURC</span>`, Ops: "1", Fee: "200"},
			{Index: "8", Status: "failed", Hash: "e5d49g0h", ShortHash: "e5d4...9g0h", OpType: "invoke", OpColor: "violet", Summary: `<span class="font-medium text-gray-900">GFAIL...4K</span> called <span class="font-mono text-violet-600">swap()</span> on Soroswap — <span class="text-red-600 font-medium">tx_failed: insufficient balance</span>`, Ops: "1", Fee: "6,100", IsFailed: true},
		},
		FeeBase:   "100",
		FeeMedian: "200",
		FeeP99:    "12,800",
		SurgePct:  "17%",
	}

	if err := pages.LedgerDetail(data).Render(r.Context(), w); err != nil {
		h.Logger.Error("render ledger detail", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// TransactionReceipt renders a single transaction page.
func (h *Handlers) TransactionReceipt(w http.ResponseWriter, r *http.Request) {
	hash := r.PathValue("hash")
	shortHash := hash
	if len(hash) > 12 {
		shortHash = hash[:6] + "..." + hash[len(hash)-4:]
	}

	data := pages.TxReceiptData{
		Hash:          hash,
		ShortHash:     shortHash,
		Status:        "success",
		IsSoroban:     true,
		HasClassicOps: true,
		SummaryHTML:   `<span class="font-semibold">GABC...7X92</span> swapped <span class="font-semibold text-red-600">5,000 XLM</span> for <span class="font-semibold text-emerald-600">485.00 USDC</span> via <span class="font-semibold">Soroswap Router</span>`,
		Timestamp:     "Oct 24, 2026 at 14:14:22 UTC",
		Ledger:        "4,819,284",
		OpsCount:      "3",
		EventsCount:   "4",
		SourceAddr:    "GABC...7X92",
		SourceAmount:  "−5,000 XLM",
		ContractName:  "Soroswap Router",
		ContractAddr:  "CAXY...Z10P",
		ContractFn:    "swap()",
		DestAddr:      "GABC...7X92",
		DestAmount:    "+485.00 USDC",
		EffectiveRate: "1 XLM = 0.097 USDC",
		Slippage:      "0.12%",
		Route:         "XLM → USDC (direct)",
		FeePaid:       "10,200",
		FeeUSD:        "$0.001",
		SorobanCPU:    "24.2M",
		SorobanMem:    "1.2 MB",
		SorobanReads:  "4",
		SorobanWrites: "3",
		SeqNumber:     "1042891748409",
		Operations: []pages.TxOperation{
			{Index: "1", Type: "Invoke Contract", IsSoroban: true, Status: "Success", SummaryHTML: `<span class="font-medium text-gray-900">GABC...7X92</span> approved the <span class="font-medium text-gray-900">Soroswap Router</span> to spend up to <span class="font-semibold text-gray-900">5,000 XLM</span>`, Contract: "CCW6...7YMK", Function: "approve()"},
			{Index: "2", Type: "Invoke Contract", IsSoroban: true, IsPrimary: true, Status: "Success", SummaryHTML: `<span class="font-medium text-gray-900">GABC...7X92</span> swapped <span class="font-semibold text-red-600">5,000 XLM</span> for <span class="font-semibold text-emerald-600">485.00 USDC</span> at a rate of <span class="font-medium text-gray-900">0.097 USDC/XLM</span>`, Contract: "CAXY...Z10P", Function: "swap()"},
			{Index: "3", Type: "Manage Data", Status: "Success", SummaryHTML: `Set data entry <span class="font-mono font-medium text-gray-900">"last_swap_rate"</span> to <span class="font-mono font-medium text-gray-900">"0.097"</span> on account <span class="font-medium text-gray-900">GABC...7X92</span>`},
		},
		Events: buildTxEvents(),
		BalanceChanges: []pages.TxBalanceChange{
			{Account: "GABC...7X92", Asset: "XLM", AssetType: "Native", TypeColor: "gray", Change: "−5,000.0000000", IsPositive: false},
			{Account: "GABC...7X92", Asset: "USDC", AssetType: "SEP-41", TypeColor: "violet", Change: "+485.0000000", IsPositive: true},
			{Account: "GABC...7X92", Asset: "XLM", AssetType: "Fee", TypeColor: "gray", Change: "−0.0010200", IsPositive: false, IsFee: true},
			{Account: "Pool:CXLM...LP", Asset: "XLM", AssetType: "Native", TypeColor: "gray", Change: "+5,000.0000000", IsPositive: true, IsPool: true},
			{Account: "Pool:CUSDC...LP", Asset: "USDC", AssetType: "SEP-41", TypeColor: "violet", Change: "−485.0000000", IsPositive: false, IsPool: true},
		},
		StateChanges: []pages.TxStateChange{
			{Action: "Modified", Key: "Balance:GABC...7X92", Contract: "XLM SAC", DetailHTML: `<span class="text-gray-400 line-through">55,000.0000000</span> → <span class="text-gray-900 font-medium">50,000.0000000</span>`},
			{Action: "Modified", Key: "Balance:GABC...7X92", Contract: "USDC (SEP-41)", DetailHTML: `<span class="text-gray-400 line-through">1,200.0000000</span> → <span class="text-gray-900 font-medium">1,685.0000000</span>`},
			{Action: "Modified", Key: "Pool:reserves", Contract: "Soroswap XLM/USDC Pool", DetailHTML: `Reserve A: 2,450,000 → 2,455,000 XLM · Reserve B: 237,650 → 237,165 USDC`},
		},
	}

	if err := pages.TransactionReceipt(data).Render(r.Context(), w); err != nil {
		h.Logger.Error("render transaction receipt", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// buildTxEvents creates mock transaction receipt events using the event decoder
// to generate human-readable DataHTML from structured event data.
func buildTxEvents() []pages.TxEvent {
	type mockTxEvent struct {
		pages.TxEvent
		raw events.RawEvent
	}

	mocks := []mockTxEvent{
		{
			TxEvent: pages.TxEvent{Index: "0", Type: "approve", TypeColor: "gray", Contract: "XLM (Native SAC)"},
			raw:     events.RawEvent{Type: "approve", From: "GABC...7X", Spender: "Soroswap Router", Amount: "5,000", Asset: "XLM"},
		},
		{
			TxEvent: pages.TxEvent{Index: "1", Type: "transfer", TypeColor: "cyan", Contract: "XLM (Native SAC)"},
			raw:     events.RawEvent{Type: "transfer", From: "GABC...7X", To: "Pool:CXLM...LP", Amount: "5,000", Asset: "XLM"},
		},
		{
			TxEvent: pages.TxEvent{Index: "2", Type: "transfer", TypeColor: "cyan", Contract: "USDC"},
			raw:     events.RawEvent{Type: "transfer", From: "Pool:CUSDC...LP", To: "GABC...7X", Amount: "485.00", Asset: "USDC"},
		},
		{
			TxEvent: pages.TxEvent{Index: "3", Type: "swap", TypeColor: "violet", Contract: "Soroswap Router"},
			raw:     events.RawEvent{Type: "swap", From: "GABC...7X", PairIn: "5,000 XLM", PairOut: "485.00 USDC", Router: "Soroswap"},
		},
	}

	result := make([]pages.TxEvent, len(mocks))
	for i, m := range mocks {
		te := m.TxEvent
		if decoded := events.Decode(m.raw); decoded != nil {
			te.DataHTML = decoded.TopicsHTML()
		}
		result[i] = te
	}
	return result
}
