package handlers

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/withObsrvr/prism/internal/events"
	"github.com/withObsrvr/prism/internal/templates/pages"
)

// mockHomeData returns hardcoded home page data for when the gateway is unavailable.
func mockHomeData(network string) pages.HomeData {
	return pages.HomeData{
		Network:       network,
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
			{Sequence: "61,504,113", SequenceRaw: "61504113", Age: "3.8s", TxCount: "42 txs", OpCount: "14 ops", IsLatest: true},
			{Sequence: "61,504,112", SequenceRaw: "61504112", Age: "8.9s", TxCount: "38 txs", OpCount: "12 ops"},
			{Sequence: "61,504,111", SequenceRaw: "61504111", Age: "14.2s", TxCount: "51 txs", OpCount: "18 ops"},
			{Sequence: "61,504,110", SequenceRaw: "61504110", Age: "19.6s", TxCount: "29 txs", OpCount: "9 ops"},
			{Sequence: "61,504,109", SequenceRaw: "61504109", Age: "24.8s", TxCount: "44 txs", OpCount: "15 ops"},
			{Sequence: "61,504,108", SequenceRaw: "61504108", Age: "30.1s", TxCount: "36 txs", OpCount: "11 ops"},
			{Sequence: "61,504,107", SequenceRaw: "61504107", Age: "35.4s", TxCount: "47 txs", OpCount: "16 ops"},
			{Sequence: "61,504,106", SequenceRaw: "61504106", Age: "40.7s", TxCount: "33 txs", OpCount: "10 ops"},
		},
		Contracts: []pages.HomeContract{
			{Rank: 1, Name: "Soroswap Router", Tag: "DEX", TagColor: "violet", Address: "CDLZ9...WK42", Href: "/contracts/CDLZ9WK42ABCDEFGHIJKLMNOPQRSTUV1234567890ABCDEFGHIJKLMN", Invocations: "284,102", Change: "+18%", IsPositive: true},
			{Rank: 2, Name: "Blend Protocol", Tag: "Lending", TagColor: "cyan", Address: "CBTRE...P8N1", Href: "/contracts/CBTREP8N1ABCDEFGHIJKLMNOPQRSTUV1234567890ABCDEFGHIJKLMN", Invocations: "198,433", Change: "+42%", IsPositive: true},
			{Rank: 3, Name: "Phoenix DEX", Tag: "DEX", TagColor: "violet", Address: "CDNS4...A7Q3", Href: "/contracts/CDNS4A7Q3ABCDEFGHIJKLMNOPQRSTUV1234567890ABCDEFGHIJKLMN", Invocations: "156,891", Change: "+7%", IsPositive: true},
			{Rank: 4, Name: "Aquarius AMM", Tag: "DEX", TagColor: "violet", Address: "CAUQR...L2B5", Href: "/contracts/CAUQRL2B5ABCDEFGHIJKLMNOPQRSTUV1234567890ABCDEFGHIJKLMN", Invocations: "134,220", Change: "-3%", IsPositive: false},
			{Rank: 5, Name: "FxDAO Vault", Tag: "Stablecoin", TagColor: "emerald", Address: "CCWN7...QR41", Href: "/contracts/CCWN7QR41ABCDEFGHIJKLMNOPQRSTUV1234567890ABCDEFGHIJKLMN", Invocations: "89,744", Change: "+31%", IsPositive: true},
		},
		Assets: []pages.HomeAsset{
			{Code: "XLM", Issuer: "Native", Initial: "X", BgColor: "bg-gray-900", Volume: "$12.4M", Change: "+5.2%", IsPositive: true},
			{Code: "USDC", Issuer: "Centre", Initial: "U", BgColor: "bg-blue-600", Volume: "$8.7M", Change: "+1.8%", IsPositive: true},
			{Code: "yXLM", Issuer: "Ultra Capital", Initial: "y", BgColor: "bg-emerald-600", Volume: "$3.2M", Change: "+14.1%", IsPositive: true},
			{Code: "AQUA", Issuer: "Aquarius", Initial: "A", BgColor: "bg-cyan-500", Volume: "$1.8M", Change: "-2.4%", IsPositive: false},
			{Code: "SHX", Issuer: "Stronghold", Initial: "S", BgColor: "bg-orange-500", Volume: "$924K", Change: "+8.7%", IsPositive: true},
		},
	}
}

// mockLedgerDetailData returns hardcoded ledger detail data.
func mockLedgerDetailData(sequence string) pages.LedgerDetailData {
	prevSequence, prevRaw := mockAdjacentSequence(sequence, -1)
	nextSequence, nextRaw := mockAdjacentSequence(sequence, 1)
	return pages.LedgerDetailData{
		Sequence:        formatMockSequence(sequence),
		SequenceRaw:     strings.ReplaceAll(sequence, ",", ""),
		PrevSequence:    prevSequence,
		PrevSequenceRaw: prevRaw,
		NextSequence:    nextSequence,
		NextSequenceRaw: nextRaw,
		ClosedAt:        "Mar 2, 2026 · 14:32:18 UTC",
		CloseTime:       "5.1s",
		Hash:            "a1b2c3d4e5f6...7890abcdef12",
		PrevHash:        "9f8e7d6c5b4a...3210fedcba98",
		Protocol:        "22",
		BaseFee:         "100",
		MaxTxSetSize:    "200",
		TotalCoins:      "105,443,902,087.3472865 XLM",
		TxCount:         "47",
		TxSuccess:       "46",
		TxFailed:        "1",
		OpCount:         "108",
		OpsPerTx:        "2.3",
		SorobanCalls:    "36",
		SorobanPct:      "33%",
		TotalFees:       "1,842",
		FeesUSD:         "$0.18",
		EventsEmitted:   "62",
		TotalCPU:        "412M",
		StateReads:      "84",
		StateReadKB:     "168 KB",
		StateWrites:     "36",
		StateWriteKB:    "42 KB",
		RentBurned:      "24.8",
		// Built through the real helper so the mock cannot drift from the
		// family/tier assignment the live path produces. The mix deliberately
		// exercises every family, including three transfer types so the
		// within-family shade ramp is visible.
		OpBreakdown: buildOpBreakdown(map[string]int{
			"Invoke Contract": 30,
			"Payment":         24,
			"Path Payment":    12,
			"Manage Offer":    11,
			"Create Account":  9,
			"Clawback":        8,
			"Manage Data":     8,
			"Extend TTL":      6,
		}, 108),
		Transactions: []pages.LedgerTx{
			{Index: "1", Status: "ok", Hash: "8f2a1b3c", ShortHash: "8f2a...1b3c", Kind: "soroban", OpType: "swap", Family: "market", Summary: `<span class="font-medium text-gray-900">GABC...7X</span> swapped <span class="font-semibold text-red-600">5,000 XLM</span> for <span class="font-semibold text-emerald-600">485 USDC</span> via Soroswap`, Ops: "3", Fee: "10,200"},
			{Index: "2", Status: "ok", Hash: "c4e93d0f", ShortHash: "c4e9...3d0f", Kind: "classic", OpType: "transfer", Family: "transfer", Summary: `<span class="font-medium text-gray-900">GDEF...2P</span> sent <span class="font-semibold text-gray-900">2,500 USDC</span> to <span class="font-medium text-gray-900">GKLM...1V</span>`, Ops: "1", Fee: "100"},
			{Index: "3", Status: "ok", Hash: "a1b28e4f", ShortHash: "a1b2...8e4f", Kind: "soroban", OpType: "swap", Family: "market", Summary: `<span class="font-medium text-gray-900">GNOP...3W</span> swapped <span class="font-semibold text-red-600">12,400 XLM</span> for <span class="font-semibold text-emerald-600">1,202 USDC</span> via Soroswap`, Ops: "2", Fee: "8,400"},
			{Index: "4", Status: "ok", Hash: "f7d24c1a", ShortHash: "f7d2...4c1a", Kind: "soroban", OpType: "mint", Family: "transfer", Summary: `<span class="font-medium text-gray-900">Blend Protocol</span> minted <span class="font-semibold text-emerald-600">45,000 BLND</span> emission to <span class="font-medium text-gray-900">GDEF...9R</span>`, Ops: "1", Fee: "5,200"},
			{Index: "5", Status: "ok", Hash: "9e8f2a5b", ShortHash: "9e8f...2a5b", Kind: "soroban", OpType: "invoke", Family: "contract", Summary: `<span class="font-medium text-gray-900">GHIJ...2M</span> called <span class="font-mono text-violet-600">deposit()</span> on <span class="font-medium text-gray-900">Blend Pool</span> with <span class="font-semibold text-gray-900">10,000 USDC</span>`, Ops: "2", Fee: "12,800"},
			{Index: "6", Status: "ok", Hash: "d4c37f8e", ShortHash: "d4c3...7f8e", Kind: "classic", OpType: "payment", Family: "transfer", Summary: `<span class="font-medium text-gray-900">GQRS...5X</span> sent <span class="font-semibold text-gray-900">8,400 EURC</span> to <span class="font-medium text-gray-900">GTUV...7Y</span>`, Ops: "1", Fee: "100"},
			{Index: "7", Status: "ok", Hash: "b8a71d2c", ShortHash: "b8a7...1d2c", Kind: "classic", OpType: "path pay", Family: "transfer", Summary: `<span class="font-medium text-gray-900">GWXY...9Z</span> path payment: <span class="font-semibold text-red-600">1,000 XLM</span> to deliver <span class="font-semibold text-emerald-600">92.40 EURC</span>`, Ops: "1", Fee: "200"},
			{Index: "8", Status: "failed", Hash: "e5d49g0h", ShortHash: "e5d4...9g0h", Kind: "failed", OpType: "invoke", Family: "contract", Summary: `<span class="font-medium text-gray-900">GFAIL...4K</span> called <span class="font-mono text-violet-600">swap()</span> on Soroswap — <span class="text-red-600 font-medium">tx_failed: insufficient balance</span>`, Ops: "1", Fee: "6,100", IsFailed: true},
		},
		Narrative:    "Busier than usual — Soroswap and Blend dominated this ledger, with 47 transactions producing 108 operations.",
		SubNarrative: "Closed in 5.1 seconds. 18 Soroban calls emitted 62 events; one transaction failed and the rest settled cleanly.",
		ActivityTag:  "Soroban-heavy",
		SpotlightCards: []pages.LedgerSpotlightCard{
			{Kicker: "New deployments", Value: "2", Body: "Fresh contracts appeared in this ledger, including first-time deployer activity.", Tone: "deploy"},
			{Kicker: "TTL extensions", Value: "14", Body: "Persistent contract state was renewed before archival risk.", Tone: "ttl"},
			{Kicker: "Failed invocations", Value: "1", Body: "One Soroban transaction reverted after validation or execution checks.", Tone: "failed"},
			{Kicker: "Events emitted", Value: "62", Body: "CAP-67 events were emitted for swaps, mints, deposits, and payments.", Tone: "events"},
			{Kicker: "State changes", Value: "36", Body: "Contract storage entries were updated during execution.", Tone: "state"},
			{Kicker: "Agent payments", Value: "3", Body: "x402-style machine payments detected in the ledger activity mix.", Tone: "agent"},
		},
		ChangedAccounts: []pages.LedgerChangedAccount{
			{Name: "Soroswap Router", Address: "CAXY…Z10P", Touched: "14 txs", Deltas: []string{"+250.1k USDC reserves", "−1.03M XLM reserves", "+0.30 USDC LP fees"}},
			{Name: "Blend Pool · USDC", Address: "CBLD…AA21", Touched: "8 txs", Deltas: []string{"+50,000 USDC supplied", "−12,000 USDC borrowed", "interest accrued"}},
			{Name: "Aquabot #7", Address: "GABC…7X92", Touched: "1 tx", Deltas: []string{"−5,000 XLM", "+485 USDC", "−0.0241 XLM fee"}},
		},
		FeeBase:   "100",
		FeeMedian: "200",
		FeeP99:    "12,800",
		SurgePct:  "17%",
	}
}

// mockTxReceiptData returns hardcoded transaction receipt data.
func mockTxReceiptData(hash, shortHash string) pages.TxReceiptData {
	return pages.TxReceiptData{
		Hash:            hash,
		ShortHash:       shortHash,
		Status:          "success",
		IsSoroban:       true,
		HasClassicOps:   true,
		HumanTitle:      "Smart wallet swap",
		HumanNarrative:  "Aquabot #7 swapped 5,000 XLM for 485.00 USDC through Soroswap.",
		ConfidenceLabel: "High confidence",
		ConfidenceValue: "0.94",
		SemanticTxType:  "smart_wallet_swap",
		SemanticSubtype: "dex_swap",
		HumanActors: []pages.TxHumanActor{
			{Label: "Aquabot #7", Role: "Effective Actor", ActorType: "Smart Wallet", Href: "/account/GABC7DEF8GHI9JKL0MNO1PQR2STU3VWX4YZ567890ABCDEFGHIJKLMNOP"},
			{Label: "Soroswap Router", Role: "Protocol", ActorType: "Contract", Href: "/contracts/CAXY7K2M8P4Q9R1S2T3U4V5W6X7Y8Z10PABCDEFGHIJKLMNOPQRSTUVWX"},
			{Label: "USDC", Role: "Receiver", ActorType: "Token Contract", Href: "/contracts/CCW67K2M8P4Q9R1S2T3U4V5W6X7Y87YMKABCDEFGHIJKLMNOPQRSTUV"},
		},
		HumanEvidence: []pages.TxHumanEvidence{
			{Label: "Type", Value: "smart_wallet_swap"},
			{Label: "Subtype", Value: "dex_swap"},
			{Label: "Function", Value: "swap"},
			{Label: "Contract", Value: "Soroswap Router"},
			{Label: "Primary operation", Value: "Contract Call"},
			{Label: "Asset movement", Value: "5,000 XLM → 485 USDC"},
			{Label: "Confidence", Value: "0.94"},
			{Label: "Wallet involved", Value: "Yes"},
		},
		HumanSignals: []pages.TxHumanSignal{
			{Title: "Smart wallet involved", Severity: "info", Summary: "Prism detected smart-wallet participation in this transaction."},
			{Title: "DEX interaction", Severity: "info", Summary: "This transaction swapped one asset for another through a protocol route."},
		},
		SummaryHTML:       `<span class="font-semibold">GABC...7X92</span> swapped <span class="font-semibold text-red-600">5,000 XLM</span> for <span class="font-semibold text-emerald-600">485.00 USDC</span> via <span class="font-semibold">Soroswap Router</span>`,
		Timestamp:         "Oct 24, 2026 at 14:14:22 UTC",
		TimestampRelative: "2 min ago",
		TimestampISO:      "2026-10-24T14:14:22Z",
		Ledger:            "4,819,284",
		LedgerRaw:         "4819284",
		OpsCount:          "3",
		EventsCount:       "4",
		SourceAddr:        "GABC...7X92",
		SourceAmount:      "−5,000 XLM",
		ContractName:      "Soroswap Router",
		ContractAddr:      "CAXY...Z10P",
		ContractAddrFull:  "CAXY7K2M8P4Q9R1S2T3U4V5W6X7Y8Z10PABCDEFGHIJKLMNOPQRSTUVWX",
		ContractFn:        "swap()",
		DestAddr:          "GABC...7X92",
		DestAmount:        "+485.00 USDC",
		EffectiveRate:     "1 XLM = 0.097 USDC",
		Slippage:          "0.12%",
		Route:             "XLM → USDC (direct)",
		FeePaid:           "10,200",
		FeePaidXLM:        "0.0010 XLM",
		MaxFee:            "20,325",
		FeeUSD:            "$0.001",
		SorobanCPU:        "24.2M",
		SorobanMem:        "1.2 MB",
		SorobanReads:      "4",
		SorobanWrites:     "3",
		SeqNumber:         "1042891748409",
		Operations: []pages.TxOperation{
			{Index: "1", Type: "Invoke Contract", IsSoroban: true, Status: "Success", SummaryHTML: `<span class="font-medium text-gray-900">GABC...7X92</span> approved the <span class="font-medium text-gray-900">Soroswap Router</span> to spend up to <span class="font-semibold text-gray-900">5,000 XLM</span>`, Contract: "CCW6...7YMK", Function: "approve()"},
			{Index: "2", Type: "Invoke Contract", IsSoroban: true, IsPrimary: true, Status: "Success", SummaryHTML: `<span class="font-medium text-gray-900">GABC...7X92</span> swapped <span class="font-semibold text-red-600">5,000 XLM</span> for <span class="font-semibold text-emerald-600">485.00 USDC</span> at a rate of <span class="font-medium text-gray-900">0.097 USDC/XLM</span>`, Contract: "CAXY...Z10P", Function: "swap()"},
			{Index: "3", Type: "Manage Data", Status: "Success", SummaryHTML: `Set data entry <span class="font-mono font-medium text-gray-900">"last_swap_rate"</span> to <span class="font-mono font-medium text-gray-900">"0.097"</span> on account <span class="font-medium text-gray-900">GABC...7X92</span>`},
		},
		Events: mockTxEvents(),
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
		// V3 humanized fields
		HeroTitle:            "Swapped 5,000 XLM for 485 USDC via Soroswap",
		AISummaryHTML:        `GABC...7X92 executed a token swap through the Soroswap decentralized exchange, converting 5,000 XLM into 485.00 USDC at a rate of 0.097 USDC per XLM. The swap routed through the XLM/USDC liquidity pool with only 0.12% slippage — well within normal parameters. This account has been actively trading for several months and this is a routine swap operation.`,
		SourceAccountContext: "Active 8 months · 1,204 transactions",
		DestAccountLabel:     "Destination",
		DestAccountContext:   "Same account (self-swap)",
		DestAccountIsNew:     false,
		FeeType:              "Soroban",
		FeeStroops:           "10,200",
		EnvelopeXDR:          "AAAAAQAAAAB...",
		LedgerCloseTime:      "1729779262",
		Timeline: []pages.TxTimelineItem{
			{
				DotColor:    "gray",
				BadgeLabel:  "Approve",
				BadgeColor:  "gray",
				Contract:    "CCW6...7YMK",
				SummaryHTML: `<span class="font-mono text-text-muted text-xs">GABC...7X92</span> approved <span class="font-semibold">Soroswap Router</span> to spend up to <span class="font-semibold">5,000 XLM</span>`,
				ContextLine: "Token allowance set for the swap contract",
			},
			{
				DotColor:    "blue",
				BadgeLabel:  "Transfer",
				BadgeColor:  "cyan",
				Contract:    "XLM SAC",
				SummaryHTML: `<span class="font-semibold">5,000 XLM</span> moved from <span class="font-mono text-text-muted text-xs">GABC...7X92</span> to <span class="font-mono text-text-muted text-xs">Pool:CXLM...LP</span>`,
				ContextLine: "XLM deposited into liquidity pool · native XLM",
			},
			{
				DotColor:    "green",
				BadgeLabel:  "Transfer",
				BadgeColor:  "cyan",
				Contract:    "USDC",
				SummaryHTML: `<span class="font-semibold">485.00 USDC</span> moved from <span class="font-mono text-text-muted text-xs">Pool:CUSDC...LP</span> to <span class="font-mono text-text-muted text-xs">GABC...7X92</span>`,
				ContextLine: "USDC withdrawn from liquidity pool to user",
			},
			{
				DotColor:    "blue",
				BadgeLabel:  "Swap",
				BadgeColor:  "violet",
				Contract:    "CAXY...Z10P",
				SummaryHTML: `<span class="font-mono text-text-muted text-xs">GABC...7X92</span> swapped <span class="font-semibold text-red-600 dark:text-red-400">5,000 XLM</span> for <span class="font-semibold text-emerald-600 dark:text-emerald-400">485.00 USDC</span>`,
				ContextLine: "Soroswap Router · effective rate 0.097 USDC/XLM · 0.12% slippage",
			},
			{
				DotColor:    "red",
				BadgeLabel:  "Debit",
				BadgeColor:  "red",
				SummaryHTML: `<span class="font-mono text-text-muted text-xs">GABC...7X92</span> sent <span class="font-semibold text-red-600 dark:text-red-400">−5,000 XLM</span>`,
				ContextLine: "Balance reduced by the transferred amount",
			},
			{
				DotColor:    "green",
				BadgeLabel:  "Credit",
				BadgeColor:  "emerald",
				SummaryHTML: `<span class="font-mono text-text-muted text-xs">GABC...7X92</span> received <span class="font-semibold text-emerald-600 dark:text-emerald-400">+485.00 USDC</span>`,
				ContextLine: "USDC balance increased from swap proceeds",
				IsLast:      true,
			},
		},
	}
}

func mockTxEvents() []pages.TxEvent {
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

// mockNetworkHealthData returns hardcoded network health data.
func mockNetworkHealthData() pages.NetworkHealthData {
	return pages.NetworkHealthData{
		Status:             "Operational",
		StatusColor:        "emerald",
		LatestLedger:       "61,504,113",
		LedgerAge:          "4s ago",
		AvgCloseTime:       "5.2s",
		CurrentTPS:         "142",
		PeakTPS:            "312",
		Tx24h:              "1.2M",
		TxChange:           "+8.4%",
		Ops24h:             "3.8M",
		OpsPerTx:           "3.2",
		FailureRate:        "0.12%",
		LedgerCapacity:     "42%",
		CapacityStatus:     "Plenty of room",
		FeeBase:            "100",
		FeeMedian:          "200",
		FeeP99:             "50,000",
		DailyFees:          "42,100 XLM",
		SurgePricing:       "Inactive",
		SorobanInvocations: "284,102",
		ActiveContracts:    "1,847",
		TotalState:         "2.4 GB",
		RentBurned:         "12,400 XLM",
		AvgCPU:             "18.2M insn",
		ProtocolVer:        "22",
		ProtocolSubtitle:   "Soroban Smart Contracts enabled",
		CoreVersion:        "stellar-core 25.2.0",
		NextUpgrade:        "No upgrade scheduled",
		HorizonVer:         "2.30.0",
		SorobanRPCVer:      "21.4.0",
		Agreement:          "97.1%",
		ConsensusHalted:    "No",
		ValidatorCount:     "35",
		QuorumSets:         "7",
		AvgLatency:         "1.2s",
		Validators: []pages.ValidatorRow{
			{Name: "SDF 1", Org: "SDF", Address: "GCGB2S...ZSTYH", Uptime: "99.95%", LastVote: "4s ago", Status: "Validating", StatusColor: "emerald", Version: "25.2.0", Quorum: "5/7", Latency: "0.8s"},
			{Name: "Blockdaemon 1", Org: "Blockdaemon", Address: "GA7UJ...K29XH", Uptime: "99.98%", LastVote: "4s ago", Status: "Validating", StatusColor: "emerald", Version: "25.2.0", Quorum: "5/7", Latency: "1.1s"},
			{Name: "SatoshiPay DE", Org: "SatoshiPay", Address: "GBFZF...Q72PT", Uptime: "99.92%", LastVote: "6s ago", Status: "Validating", StatusColor: "emerald", Version: "25.1.3", Quorum: "5/7", Latency: "1.4s"},
			{Name: "Lobstr Pool 1", Org: "Lobstr", Address: "GCWJK...H8R4P", Uptime: "99.87%", LastVote: "5s ago", Status: "Validating", StatusColor: "emerald", Version: "25.2.0", Quorum: "5/7", Latency: "1.0s"},
			{Name: "Franklin Temp.", Org: "Franklin", Address: "GCZJM...W2K18", Uptime: "99.99%", LastVote: "4s ago", Status: "Validating", StatusColor: "emerald", Version: "25.2.0", Quorum: "5/7", Latency: "0.9s"},
		},
		RecentLedgers: []pages.NetworkLedger{
			{Sequence: "61,504,113", SequenceRaw: "61504113", Age: "4s ago", TxCount: "42", OpsCount: "134", SorobanCalls: "18", SorobanPct: "43%", Fees: "8,200", CloseTime: "5.1s", IsLatest: true},
			{Sequence: "61,504,112", SequenceRaw: "61504112", Age: "9s ago", TxCount: "38", OpsCount: "98", SorobanCalls: "12", SorobanPct: "32%", Fees: "6,400", CloseTime: "5.3s"},
			{Sequence: "61,504,111", SequenceRaw: "61504111", Age: "14s ago", TxCount: "51", OpsCount: "187", SorobanCalls: "24", SorobanPct: "47%", Fees: "12,100", CloseTime: "4.8s"},
			{Sequence: "61,504,110", SequenceRaw: "61504110", Age: "20s ago", TxCount: "29", OpsCount: "72", SorobanCalls: "8", SorobanPct: "28%", Fees: "4,200", CloseTime: "6.8s", IsSlow: true},
			{Sequence: "61,504,109", SequenceRaw: "61504109", Age: "25s ago", TxCount: "44", OpsCount: "156", SorobanCalls: "19", SorobanPct: "43%", Fees: "9,800", CloseTime: "5.0s"},
		},
	}
}

// mockContractListData returns hardcoded contract list data.
func mockContractListData() pages.ContractListData {
	return pages.ContractListData{
		Contracts: []pages.ContractListItem{
			{Rank: 1, Name: "Soroswap Router", Address: "CAXY...Z10P", Href: "/contracts/CAXY7K2M8P4Q9R1S2T3U4V5W6X7Y8Z10PABCDEFGHIJKLMNOPQRSTUVWX", Tag: "DEX", TagColor: "violet", Invocations: "284,102", Change: "+12.4%", IsPositive: true},
			{Rank: 2, Name: "Blend Protocol", Address: "CBLND...P2R8", Href: "/contracts/CBLND7K2M8P4Q9R1S2T3U4V5W6X7Y8P2R8ABCDEFGHIJKLMNOPQRSTUV", Tag: "Lending", TagColor: "emerald", Invocations: "142,847", Change: "+8.2%", IsPositive: true},
			{Rank: 3, Name: "USDC Token", Address: "CCW6...7YMK", Href: "/contracts/CCW67K2M8P4Q9R1S2T3U4V5W6X7Y87YMKABCDEFGHIJKLMNOPQRSTUV", Tag: "Token", TagColor: "blue", Invocations: "98,204", Change: "+3.1%", IsPositive: true},
			{Rank: 4, Name: "Aquarius AMM", Address: "CAQUA...8K32", Href: "/contracts/CAQUA7K2M8P4Q9R1S2T3U4V5W6X7Y88K32ABCDEFGHIJKLMNOPQRSTUV", Tag: "AMM", TagColor: "cyan", Invocations: "67,421", Change: "-2.4%", IsPositive: false},
			{Rank: 5, Name: "Phoenix DEX", Address: "CPHNX...R4M2", Href: "/contracts/CPHNX7K2M8P4Q9R1S2T3U4V5W6X7Y8R4M2ABCDEFGHIJKLMNOPQRSTUV", Tag: "DEX", TagColor: "violet", Invocations: "45,892", Change: "+22.1%", IsPositive: true},
			{Rank: 6, Name: "FxDAO Stablecoin", Address: "CFXD...Q7K1", Href: "/contracts/CFXD7K2M8P4Q9R1S2T3U4V5W6X7Y8Q7K1ABCDEFGHIJKLMNOPQRSTUV", Tag: "Stablecoin", TagColor: "amber", Invocations: "28,104", Change: "+5.7%", IsPositive: true},
			{Rank: 7, Name: "Stellar Geometries", Address: "CNFT...G7K2", Href: "/contracts/CNFT7K2M8P4Q9R1S2T3U4V5W6X7Y8G7K2ABCDEFGHIJKLMNOPQRSTUV", Tag: "NFT", TagColor: "rose", Invocations: "12,847", Change: "+142%", IsPositive: true},
			{Rank: 8, Name: "Reflector Oracle", Address: "CREFL...M8P4", Href: "/contracts/CREFL7K2M8P4Q9R1S2T3U4V5W6X7Y8M8P4ABCDEFGHIJKLMNOPQRSTUV", Tag: "Oracle", TagColor: "gray", Invocations: "8,421", Change: "-0.8%", IsPositive: false},
		},
	}
}

// mockContractDetailData returns hardcoded contract detail data.
func mockContractDetailData() pages.ContractDetailData {
	return pages.ContractDetailData{
		Name:              "Soroswap Router",
		Address:           "CAXY7K2M8P4Q9R1S2T3U4V5W6X7Y8Z10PABCDEFGHIJKLMNOPQRSTUVWX",
		IsVerified:        true,
		Tags:              []string{"DEX", "AMM"},
		DeployedAt:        "Jan 15, 2024",
		DeployLedger:      "4,201,847",
		TotalInvocations:  "284,102",
		StorageEntries:    "1,847",
		StateSize:         "2.4 MB",
		MonthlyRent:       "42,100 XLM",
		LastInvoked:       "2 min ago",
		Creator:           "GABC...7X92",
		CreatorHref:       "/account/GABC7DEF8GHI9JKL0MNO1PQR2STU3VWX4YZ567890",
		WASMSize:          "142 KB",
		FunctionsCount:    "12",
		IsSourceVerified:  true,
		ActivityPoints:    []float64{4200, 4800, 5100, 4900, 5400, 6200, 5800, 6500, 7200, 6800, 7500, 8200, 7800, 8500, 7200, 6800, 7500, 8200, 8800, 9200, 8500, 7800, 7200, 6800, 7500, 8200, 8800, 9200, 9500, 8800},
		AvgInvocations:    "9,470",
		PeakInvocations:   "14,200",
		SuccessRate:       "99.2%",
		ExportedFunctions: []string{"swap", "add_liquidity", "remove_liquidity", "get_reserves", "get_pair", "initialize", "set_admin", "get_admin", "deposit", "withdraw", "get_amounts_out", "get_amounts_in"},
		Narrative:         "Soroswap Router appears to be a DEX-oriented contract with swap and liquidity-management functions.",
		Context:           "Created by GABC...7X92 · Recent activity observed · using about 2.4 MB of state",
		FunctionSummary:   "Function surface includes swapped assets, add liquidity, remove liquidity, get reserves.",
		StorageSummary:    "This contract currently has 1,847 storage entries using about 2.4 MB of state.",
		Signals: []pages.ContractHumanSignal{
			{Title: "Known project", Severity: "info", Summary: "This contract name suggests it is associated with the Soroswap project."},
			{Title: "Large state footprint", Severity: "warn", Summary: "Soroswap Router currently uses about 2.4 MB of state, which is large enough to merit review."},
		},
		Evidence: []pages.ContractHumanEvidence{
			{Label: "Name", Value: "Soroswap Router"},
			{Label: "Project inference", Value: "soroswap"},
			{Label: "Observed functions", Value: "swap, add_liquidity, remove_liquidity, get_reserves"},
			{Label: "Creator", Value: "GABC...7X92"},
			{Label: "Storage entries", Value: "1,847"},
			{Label: "Observed invocations", Value: "284,102"},
		},
		Functions: []pages.ContractFunction{
			{Name: "swap", Calls24h: "4,847", Calls7d: "32,104", Calls30d: "142,847", SuccessRate: "99.4%", AvgCPU: "24.2M insn", LastCalled: "2 min ago"},
			{Name: "add_liquidity", Calls24h: "1,201", Calls7d: "8,402", Calls30d: "48,201", SuccessRate: "98.8%", AvgCPU: "18.4M insn", LastCalled: "8 min ago"},
			{Name: "remove_liquidity", Calls24h: "847", Calls7d: "5,921", Calls30d: "32,104", SuccessRate: "99.1%", AvgCPU: "16.8M insn", LastCalled: "22 min ago"},
			{Name: "get_reserves", Calls24h: "2,421", Calls7d: "12,847", Calls30d: "28,421", SuccessRate: "100%", AvgCPU: "2.1M insn", LastCalled: "30s ago"},
			{Name: "initialize", Calls24h: "0", Calls7d: "0", Calls30d: "1", SuccessRate: "100%", AvgCPU: "42.8M insn", LastCalled: "Jan 15, 2024"},
		},
		Invocations: []pages.ContractInvocation{
			{TxHash: "8f2a1b4c5d6e7f8a", ShortHash: "8f2a...1b4c", Function: "swap", Caller: "GABC...7X92", Status: "Success", StatusColor: "emerald", Summary: "Swapped 5,000 XLM -> 485 USDC", CPUUsed: "24.2M", Age: "2 min ago"},
			{TxHash: "c4e93d7f2a1b8e5a", ShortHash: "c4e9...3d7f", Function: "add_liquidity", Caller: "GDEF...9R23", Status: "Success", StatusColor: "emerald", Summary: "Added 10,000 XLM + 970 USDC", CPUUsed: "18.4M", Age: "8 min ago"},
		},
		StorageItems: []pages.ContractStorageItem{
			{Key: "Admin", Type: "Instance", TypeColor: "blue", Size: "64 B", TTL: "42 days", Sub: "Contract admin config", ValueType: "Address", Value: "GABC…7F9A", KeyHash: "a1b2c3d4e5", SizeBytes: 64, TTLDays: 42, TTLLedgers: 604000, HealthPct: 58},
			{Key: "fee_recipient", Type: "Instance", TypeColor: "blue", Size: "96 B", TTL: "40 days", Sub: "Protocol fee recipient", ValueType: "Address", Value: "GNOP…3K8M", KeyHash: "e1f2a3b4c5", SizeBytes: 96, TTLDays: 40, TTLLedgers: 580000, HealthPct: 55},
			{Key: "total_swaps", Type: "Instance", TypeColor: "blue", Size: "64 B", TTL: "39 days", Sub: "Lifetime swap counter", ValueType: "U64", Value: "1284771", KeyHash: "a7b8c9d0e1", SizeBytes: 64, TTLDays: 39, TTLLedgers: 562000, HealthPct: 53},
			{Key: "Balance:GDKX…8R42", Type: "Persistent", TypeColor: "cyan", Size: "128 B", TTL: "3 days", Sub: "User LP share balance", ValueType: "i128", Value: "48211900000", KeyHash: "b0c1d2e3f4", SizeBytes: 128, TTLDays: 3, TTLLedgers: 43000, HealthPct: 4},
			{Key: "Balance:GHIJ…2M56", Type: "Persistent", TypeColor: "cyan", Size: "128 B", TTL: "4 days", Sub: "User LP share balance", ValueType: "i128", Value: "12905000000", KeyHash: "c1d2e3f4a5", SizeBytes: 128, TTLDays: 4, TTLLedgers: 57000, HealthPct: 5},
			{Key: "Allowance:GABC…7X92", Type: "Persistent", TypeColor: "cyan", Size: "96 B", TTL: "6 days", Sub: "Token allowance", ValueType: "i128", Value: "5000000000", KeyHash: "d2e3f4a5b6", SizeBytes: 96, TTLDays: 6, TTLLedgers: 86000, HealthPct: 8},
			{Key: "SwapRoute:0x8f2a", Type: "Temporary", TypeColor: "amber", Size: "256 B", TTL: "12 days", Sub: "Cached swap route", ValueType: "Bytes", Value: "0x8f2a…c1e0", KeyHash: "e3f4a5b6c7", SizeBytes: 256, TTLDays: 12, TTLLedgers: 172000, HealthPct: 16},
			{Key: "SwapRoute:0x1a2b", Type: "Temporary", TypeColor: "amber", Size: "256 B", TTL: "9 days", Sub: "Cached swap route", ValueType: "Bytes", Value: "0x1a2b…9d4f", KeyHash: "f4a5b6c7d8", SizeBytes: 256, TTLDays: 9, TTLLedgers: 129000, HealthPct: 12},
			{Key: "Balance:GXYZ…9B3C", Type: "Persistent", TypeColor: "cyan", Size: "128 B", TTL: "22 days", Sub: "User LP share balance", ValueType: "i128", Value: "9820500000", KeyHash: "a5b6c7d8e9", SizeBytes: 128, TTLDays: 22, TTLLedgers: 316000, HealthPct: 31},
			{Key: "Allowance:GPQR…4K2L", Type: "Persistent", TypeColor: "cyan", Size: "96 B", TTL: "18 days", Sub: "Token allowance", ValueType: "i128", Value: "2500000000", KeyHash: "b6c7d8e9f0", SizeBytes: 96, TTLDays: 18, TTLLedgers: 259000, HealthPct: 24},
			{Key: "nonce::0x9f4e", Type: "Temporary", TypeColor: "amber", Size: "64 B", TTL: "2 days", Sub: "Replay nonce", ValueType: "U64", Value: "9182736455", KeyHash: "c7d8e9f0a1", SizeBytes: 64, TTLDays: 2, TTLLedgers: 28000, HealthPct: 3},
			{Key: "ReserveA:XLM/USDC", Type: "Persistent", TypeColor: "cyan", Size: "128 B", TTL: "38 days", Sub: "Pool reserve balance", ValueType: "Map", Value: `{"reserve":"84920500000","shares":"1.2e9"}`, KeyHash: "d8e9f0a1b2", SizeBytes: 128, TTLDays: 38, TTLLedgers: 547000, HealthPct: 52},
			{Key: "ReserveB:XLM/EURC", Type: "Persistent", TypeColor: "cyan", Size: "128 B", TTL: "34 days", Sub: "Pool reserve balance", ValueType: "Map", Value: `{"reserve":"31770200000","shares":"6.4e8"}`, KeyHash: "e9f0a1b2c3", SizeBytes: 128, TTLDays: 34, TTLLedgers: 512000, HealthPct: 47},
			{Key: "TotalShares:XLM/USDC", Type: "Persistent", TypeColor: "cyan", Size: "96 B", TTL: "41 days", Sub: "Pool total shares", ValueType: "i128", Value: "1204885000000", KeyHash: "f0a1b2c3d4", SizeBytes: 96, TTLDays: 41, TTLLedgers: 590000, HealthPct: 55},
		},
		StorageStats: pages.ContractStorageStats{
			TotalEntries:   "1,847",
			MonthlyRentXLM: "42,100",
			MonthlyRentUSD: "~$4,083",
			Healthy:        "1,692",
			AtRisk:         "143",
			Critical:       "12",
		},
		StorageTypes: []pages.ContractStorageTypeAgg{
			{Name: "Instance", Entries: "4", Size: "2.4 KB", MinTTL: "39d"},
			{Name: "Persistent", Entries: "1,251", Size: "847 KB", MinTTL: "3d"},
			{Name: "Temporary", Entries: "592", Size: "124 KB", MinTTL: "2d"},
		},
		WASMHash: "a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2e3f4a5b6c7d8e9f0",
	}
}

// mockAssetDirectoryData returns hardcoded asset directory data.
func mockAssetDirectoryData() pages.AssetDirectoryData {
	return pages.AssetDirectoryData{
		TotalAssets:   "12,847",
		ClassicCount:  "11,203",
		SEP41Count:    "1,644",
		NetworkVolume: "$24.8M",
		VolumeChange:  "+12.4%",
		Trustlines:    "8.2M",
		DEXLiquidity:  "$142M",
		ActiveFilter:  "all",
		CurrentPage:   1,
		TotalPages:    128,
		Assets: []pages.AssetRow{
			{Rank: 1, Code: "XLM", Name: "Stellar Lumens", BgColor: "bg-gray-900", Initial: "X", Price: "$0.097", Change: "+2.4%", IsUp: true, IsVerified: true, MarketCap: "$3.2B", Volume: "$89.2M", Supply: "50.0B", Holders: "8.2M", TypeBadge: "Native", TypeColor: "gray"},
			{Rank: 2, Code: "USDC", Name: "USD Coin", BgColor: "bg-blue-600", Initial: "U", Price: "$1.00", Change: "+0.01%", IsUp: true, IsVerified: true, MarketCap: "$142M", Volume: "$18.4M", Supply: "142M", Holders: "284K", TypeBadge: "Classic", TypeColor: "gray"},
			{Rank: 3, Code: "yUSDC", Name: "Blend USDC", BgColor: "bg-emerald-600", Initial: "y", Price: "$1.03", Change: "+3.2%", IsUp: true, IsVerified: true, MarketCap: "$84M", Volume: "$4.2M", Supply: "81.6M", Holders: "12.4K", TypeBadge: "SEP-41", TypeColor: "violet"},
			{Rank: 4, Code: "AQUA", Name: "Aquarius", BgColor: "bg-cyan-600", Initial: "A", Price: "$0.0049", Change: "-1.8%", IsUp: false, IsVerified: true, MarketCap: "$49M", Volume: "$2.1M", Supply: "10B", Holders: "142K", TypeBadge: "Classic", TypeColor: "gray"},
			{Rank: 5, Code: "BLND", Name: "Blend Token", BgColor: "bg-violet-600", Initial: "B", Price: "$0.039", Change: "+12.4%", IsUp: true, IsVerified: true, MarketCap: "$39M", Volume: "$1.8M", Supply: "1B", Holders: "8.4K", TypeBadge: "SEP-41", TypeColor: "violet"},
			{Rank: 6, Code: "SHX", Name: "Stronghold", BgColor: "bg-amber-600", Initial: "S", Price: "$0.0028", Change: "-0.5%", IsUp: false, MarketCap: "$28M", Volume: "$890K", Supply: "10B", Holders: "92K", TypeBadge: "Classic", TypeColor: "gray"},
			{Rank: 7, Code: "EURC", Name: "Euro Coin", BgColor: "bg-blue-500", Initial: "E", Price: "$1.08", Change: "+0.3%", IsUp: true, IsVerified: true, MarketCap: "$21M", Volume: "$1.2M", Supply: "19.4M", Holders: "18K", TypeBadge: "Classic", TypeColor: "gray"},
			{Rank: 8, Code: "RWA", Name: "Real World Asset", BgColor: "bg-rose-600", Initial: "R", Price: "$12.40", Change: "+5.1%", IsUp: true, MarketCap: "$12.4M", Volume: "$420K", Supply: "1M", Holders: "2.1K", TypeBadge: "SEP-41", TypeColor: "violet"},
			{Rank: 9, Code: "FIDR", Name: "Fidelity Fund", BgColor: "bg-green-700", Initial: "F", Price: "$100.00", Change: "+0.8%", IsUp: true, IsVerified: true, MarketCap: "$10M", Volume: "$200K", Supply: "100K", Holders: "847", TypeBadge: "Classic", TypeColor: "gray"},
			{Rank: 10, Code: "BTC", Name: "Wrapped Bitcoin", BgColor: "bg-orange-500", Initial: "B", Price: "$97,240", Change: "+1.2%", IsUp: true, MarketCap: "$9.7M", Volume: "$340K", Supply: "100", Holders: "1.2K", TypeBadge: "Classic", TypeColor: "gray"},
		},
	}
}

// mockAccountData returns hardcoded account portfolio data.
func mockAccountData() pages.AccountData {
	return pages.AccountData{
		Address:        "GABC7DEF8GHI9JKL0MNO1PQR2STU3VWX4YZ567890ABCDEFGHIJKLMNOP",
		ShortAddress:   "GABC...MNOP",
		TotalValue:     "$58,247",
		TotalCents:     ".82",
		XLMBalance:     "124,500.00 XLM",
		Trustlines:     "12",
		ActiveOffers:   "1",
		Subentries:     "14",
		CreatedAt:      "Jan 15, 2024",
		HomeDomain:     "stellar.org",
		SequenceNumber: "214,041,928,194,121",
		IsFunded:       true,
		SignerCount:    "1",
		Balances: []pages.AccountBalance{
			{Code: "XLM", Name: "Stellar Lumens", BgColor: "bg-gray-900", Type: "Native", TypeColor: "gray", Balance: "124,500.00", ValueUSD: "$12,078.50"},
			{Code: "USDC", Name: "USD Coin", Issuer: "Centre", BgColor: "bg-blue-600", Type: "Classic", TypeColor: "gray", Balance: "25,000.00", ValueUSD: "$25,000.00"},
			{Code: "yUSDC", Name: "Blend USDC", Issuer: "Blend", BgColor: "bg-emerald-600", Type: "SEP-41", TypeColor: "violet", Balance: "18,200.00", ValueUSD: "$18,247.32"},
			{Code: "AQUA", Name: "Aquarius", Issuer: "Aquarius", BgColor: "bg-cyan-600", Type: "Classic", TypeColor: "gray", Balance: "500,000.00", ValueUSD: "$2,450.00"},
			{Code: "BLND", Name: "Blend Token", Issuer: "Blend", BgColor: "bg-violet-600", Type: "SEP-41", TypeColor: "violet", Balance: "12,000.00", ValueUSD: "$471.00"},
		},
		Activities: []pages.AccountActivity{
			{IconBg: "bg-emerald-50", IconColor: "text-emerald-600", Summary: "Swap: 12,400 XLM → 1,202.80 USDC via Soroswap", Badge: "Swap", BadgeColor: "violet", TxHash: "8f2a1b4c5d6e7f8a", ShortHash: "8f2a...1b4c", Time: "2 min ago", DateGroup: "Today"},
			{IconBg: "bg-violet-50", IconColor: "text-violet-600", Summary: "Blend Protocol: supply(USDC, 5,000)", Badge: "Contract", BadgeColor: "violet", TxHash: "c4e93d7f2a1b8e5a", ShortHash: "c4e9...3d7f", Time: "1 hour ago", DateGroup: "Today"},
			{IconBg: "bg-blue-50", IconColor: "text-blue-600", Summary: "Sent 500 USDC to GDEF...9R23", Badge: "Payment", BadgeColor: "blue", TxHash: "a1b28e5a9f7d2c4e", ShortHash: "a1b2...8e5a", Time: "18 hours ago", DateGroup: "Yesterday"},
			{IconBg: "bg-gray-50", IconColor: "text-gray-600", Summary: "Added trustline: BLND (Blend Token)", Badge: "Trustline", BadgeColor: "gray", TxHash: "f7d24c9b3a8e1f2d", ShortHash: "f7d2...4c9b", Time: "22 hours ago", DateGroup: "Yesterday"},
		},
		Contracts: []pages.AccountContract{
			{Name: "Soroswap Router", Badge: "DEX", BadgeColor: "violet", Address: "CAXY...Z10P", Href: "/contracts/CAXY7K2M8P4Q9R1S2T3U4V5W6X7Y8Z10PABCDEFGHIJKLMNOPQRSTUVWX", TopFn: "swap", Calls: "142", Fees: "2,847 XLM", LastCall: "2 min ago"},
			{Name: "Blend Protocol", Badge: "Lending", BadgeColor: "emerald", Address: "CBLND...P2R8", Href: "/contracts/CBLND7K2M8P4Q9R1S2T3U4V5W6X7Y8P2R8ABCDEFGHIJKLMNOPQRSTUV", TopFn: "supply", Calls: "38", Fees: "412 XLM", LastCall: "1 hour ago"},
		},
		Offers: []pages.AccountOffer{
			{Side: "Buy", SideColor: "emerald", Pair: "USDC/XLM", Price: "0.097", PriceUnit: "XLM", Amount: "10,000 USDC", OfferID: "892341"},
		},
		Signers: []pages.AccountSigner{
			{Address: "GABC...MNOP", Type: "ed25519", IsSelf: true, Weight: "1"},
		},
		Thresholds: []pages.AccountThreshold{
			{Label: "Low", Value: "1", Pct: "100%", Color: "emerald"},
			{Label: "Medium", Value: "1", Pct: "100%", Color: "emerald"},
			{Label: "High", Value: "1", Pct: "100%", Color: "emerald"},
		},
	}
}

// mockSearchData returns hardcoded search data.
func mockSearchData(query string) pages.SearchData {
	return pages.SearchData{
		Query:          query,
		DetectedType:   "account",
		DetectedLabel:  "Looks like a Stellar account address",
		DetectedDesc:   "Starts with G + alphanumeric characters. Redirecting to account view.",
		DetectedHref:   "/account/" + query,
		RecentSearches: []string{"Soroswap Router", "GABC...7X92", "8f2a...1b3c", "USDC", "Ledger 5,104,938"},
		Results: []pages.SearchResultGroup{
			{Category: "Contracts", Items: []pages.SearchResultItem{
				{Name: "Soroswap Router", Badge: "DEX", BadgeColor: "violet", Subtitle: "CAXY...Z10P", IconBg: "bg-violet-100", Href: "/contracts/CAXY"},
			}},
			{Category: "Accounts", Items: []pages.SearchResultItem{
				{Name: "GABC...7X92", Badge: "Funded", BadgeColor: "emerald", Subtitle: "$58,247 total value", IconBg: "bg-gray-100", Href: "/account/GABC"},
			}},
		},
	}
}

// mockEventsFirehoseData returns hardcoded firehose events data.
func mockEventsFirehoseData() pages.EventsFirehoseData {
	return pages.EventsFirehoseData{
		MatchedEvents: "2,847",
		LedgerStart:   "5,104,892",
		LedgerEnd:     "5,104,938",
		EventsPerSec:  "48",
		Filters: []pages.EventFilter{
			{Label: "contract:Soroswap", Color: "violet"},
			{Label: "type:transfer,swap", Color: "cyan"},
		},
		Events: mockFirehoseEvents(),
	}
}

func mockFirehoseEvents() []pages.FirehoseEvent {
	type mockEvent struct {
		pages.FirehoseEvent
		raw events.RawEvent
	}

	mocks := []mockEvent{
		{
			FirehoseEvent: pages.FirehoseEvent{Time: "0.4s", Type: "transfer", TypeColor: "transfer", ContractName: "USDC Token", ContractAddr: "CCW6...7YMK", Ledger: "5,104,938", TxShort: "8f2a...1b", TxHash: "8f2a1b4c", IsNew: true, AlertBadge: "WHALE", AlertColor: "amber", DetailJSON: "{\n  \"type\": \"transfer\",\n  \"from\": \"GABC...7X92\",\n  \"to\": \"GDEF...9R23\",\n  \"amount\": \"2500.0000000\",\n  \"asset\": \"USDC:GA5Z...CCW6\"\n}", DetailMeta: "Large USDC transfer (>$1,000). Source account is a known DeFi power user with 142 contract interactions this month."},
			raw:           events.RawEvent{Type: "transfer", From: "GABC...7X", To: "GDEF...9R", Amount: "2,500.00", Asset: "USDC"},
		},
		{
			FirehoseEvent: pages.FirehoseEvent{Time: "0.4s", Type: "swap", TypeColor: "contract", ContractName: "Soroswap Router", ContractAddr: "CAXY...Z10P", Ledger: "5,104,938", TxShort: "c4e9...3d", TxHash: "c4e93d7f", IsNew: true, DetailJSON: "{\n  \"type\": \"swap\",\n  \"pair\": \"XLM/USDC\",\n  \"amount_in\": \"12400.0000000\",\n  \"amount_out\": \"1202.8000000\",\n  \"rate\": \"0.097\"\n}", DetailMeta: "Soroswap Router swap via XLM/USDC direct pool. Effective rate: 0.097 USDC/XLM."},
			raw:           events.RawEvent{Type: "swap", From: "GNOP...3W", PairIn: "12,400 XLM", PairOut: "1,202.80 USDC", Router: "Soroswap"},
		},
		{
			FirehoseEvent: pages.FirehoseEvent{Time: "2.1s", Type: "transfer", TypeColor: "transfer", ContractName: "USDC Token", ContractAddr: "CCW6...7YMK", Ledger: "5,104,937", TxShort: "a1b2...8e", TxHash: "a1b28e5a"},
			raw:           events.RawEvent{Type: "transfer", From: "GKLM...1V", To: "GHIJ...2M", Amount: "500.00", Asset: "USDC"},
		},
		{
			FirehoseEvent: pages.FirehoseEvent{Time: "2.1s", Type: "mint", TypeColor: "mint", ContractName: "BLND Token", ContractAddr: "CBLND...E84K", Ledger: "5,104,937", TxShort: "f7d2...4c", TxHash: "f7d24c9b"},
			raw:           events.RawEvent{Type: "mint", To: "GDEF...9R", Amount: "45,000", Asset: "BLND"},
		},
		{
			FirehoseEvent: pages.FirehoseEvent{Time: "7.3s", Type: "swap", TypeColor: "contract", ContractName: "Soroswap Router", ContractAddr: "CAXY...Z10P", Ledger: "5,104,936", TxShort: "9e8f...2a", TxHash: "9e8f2a1b"},
			raw:           events.RawEvent{Type: "swap", From: "GQRS...5X", PairIn: "500,000 AQUA", PairOut: "6,200 XLM", Router: "Soroswap"},
		},
		{
			FirehoseEvent: pages.FirehoseEvent{Time: "7.3s", Type: "approve", TypeColor: "approve", ContractName: "USDC Token", ContractAddr: "CCW6...7YMK", Ledger: "5,104,936", TxShort: "d3c2...7f", TxHash: "d3c27f1a"},
			raw:           events.RawEvent{Type: "approve", From: "GABC...7X", Spender: "CAXY...Z10P", Amount: "∞", Asset: "USDC"},
		},
	}

	result := make([]pages.FirehoseEvent, len(mocks))
	for i, m := range mocks {
		fe := m.FirehoseEvent
		if decoded := events.Decode(m.raw); decoded != nil {
			fe.TopicsHTML = decoded.TopicsHTML()
		}
		result[i] = fe
	}
	return result
}

func mockAdjacentSequence(sequence string, delta int64) (string, string) {
	raw := strings.ReplaceAll(sequence, ",", "")
	n, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return sequence, raw
	}
	n += delta
	if n < 0 {
		n = 0
	}
	return formatMockSequence(fmt.Sprintf("%d", n)), fmt.Sprintf("%d", n)
}

func formatMockSequence(sequence string) string {
	raw := strings.ReplaceAll(sequence, ",", "")
	n := len(raw)
	if n <= 3 {
		return raw
	}
	pre := n % 3
	if pre == 0 {
		pre = 3
	}
	var b strings.Builder
	b.WriteString(raw[:pre])
	for i := pre; i < n; i += 3 {
		b.WriteString(",")
		b.WriteString(raw[i : i+3])
	}
	return b.String()
}
