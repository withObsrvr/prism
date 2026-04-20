package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	componentsv2 "github.com/withObsrvr/prism/internal/templates/v2/components"
	pagesv2 "github.com/withObsrvr/prism/internal/templates/v2/pages"
	vmv2 "github.com/withObsrvr/prism/internal/templates/v2/viewmodel"
)

func (h *Handlers) HomeV2(w http.ResponseWriter, r *http.Request) {
	data := mockHomeV2Data(networkFromRequest(r))
	if err := pagesv2.Home(data).Render(r.Context(), w); err != nil {
		h.Logger.Error("render home v2", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func mockHomeV2Data(network string) vmv2.HomeData {
	cfg := homeV2NetworkConfig(network)
	feedJSON, _ := json.Marshal(mockHomeV2Feed(network))
	return vmv2.HomeData{
		Header: componentsv2.HeaderData{LedgerNumber: cfg.LedgerNumber, AgeLabel: "2 seconds ago", Network: network},
		Hero: vmv2.HeroData{
			Eyebrow:      "What brings you here?",
			HeadlineHTML: cfg.HeadlineHTML,
			Body:         cfg.HeroBody,
		},
		Prompt: vmv2.PromptData{Placeholder: cfg.Placeholder},
		Alert: vmv2.AlertData{
			Title: "Three contracts are running out of room",
			Body:  "Blend’s lending pool, Soroswap’s router, and Phoenix’s AMM have under four days of persistent-storage life left. If nobody extends them, they’ll be archived and stop responding.",
			Meta:  "Why this matters: contract operators need to extend TTL before it hits zero — otherwise their users see failures.",
			CTA:   "Review →",
		},
		LedgerFeed: vmv2.LedgerFeedData{
			Title: cfg.FeedTitle,
			Copy:  cfg.FeedCopy,
			Note:  cfg.FeedNote,
			Filters: []vmv2.FilterChipData{
				{Label: "Everything", Active: true},
				{Label: "All transactions"},
				{Label: "Swaps", DotColor: "var(--v2-teal)"},
				{Label: "Contract calls", DotColor: "var(--v2-violet)"},
				{Label: "Agent payments", DotColor: "var(--v2-orange)"},
				{Label: "New deployments", DotColor: "var(--v2-gold)"},
				{Label: "Confidential", DotColor: "var(--v2-blue)"},
				{Label: "Classic"},
			},
			Rows: []vmv2.LedgerRowData{
				{
					LedgerNumber: "52,844,201", TransactionCount: "212",
					Meta:            "across 424 operations · closed by sdf-validator-1",
					Chips:           []componentsv2.LedgerMetricChip{{Label: "50 swaps", Kind: "swap"}, {Label: "67 calls", Kind: "call"}, {Label: "54 agent payments", Kind: "agent"}, {Label: "33 classic", Kind: "classic"}, {Label: "2 deploys", Kind: "deploy"}, {Label: "2 confidential", Kind: "confidential"}},
					InstructionsPct: 61, ReadWritePct: 35, CloseTime: "4.9s", Age: "just now", SideMeta: "5 ledgers / min",
				},
				{
					LedgerNumber: "52,844,200", TransactionCount: "226",
					Meta:            "across 452 operations · closed by whale-stack",
					Chips:           []componentsv2.LedgerMetricChip{{Label: "42 swaps", Kind: "swap"}, {Label: "75 calls", Kind: "call"}, {Label: "37 agent payments", Kind: "agent"}, {Label: "67 classic", Kind: "classic"}, {Label: "3 deploys", Kind: "deploy"}, {Label: "1 confidential", Kind: "confidential"}},
					InstructionsPct: 76, ReadWritePct: 45, CloseTime: "4.9s", Age: "5 seconds ago", SideMeta: "5 ledgers / min",
				},
				{
					LedgerNumber: "52,844,199", TransactionCount: "187",
					Meta:            "across 561 operations · closed by publicnode",
					Chips:           []componentsv2.LedgerMetricChip{{Label: "35 swaps", Kind: "swap"}, {Label: "70 calls", Kind: "call"}, {Label: "30 agent payments", Kind: "agent"}, {Label: "52 classic", Kind: "classic"}, {Label: "1 deploy", Kind: "deploy"}},
					InstructionsPct: 67, ReadWritePct: 36, CloseTime: "4.9s", Age: "10 seconds ago", SideMeta: "5 ledgers / min",
				},
				{
					LedgerNumber: "52,844,198", TransactionCount: "199",
					Meta:            "across 388 operations · closed by lobstr",
					Chips:           []componentsv2.LedgerMetricChip{{Label: "50 swaps", Kind: "swap"}, {Label: "62 calls", Kind: "call"}, {Label: "55 agent payments", Kind: "agent"}, {Label: "32 classic", Kind: "classic"}, {Label: "1 deploy", Kind: "deploy"}},
					InstructionsPct: 50, ReadWritePct: 50, CloseTime: "4.9s", Age: "15 seconds ago", SideMeta: "5 ledgers / min",
				},
				{
					LedgerNumber: "52,844,197", TransactionCount: "197",
					Meta:            "across 788 operations · closed by coinbase-cloud",
					Chips:           []componentsv2.LedgerMetricChip{{Label: "53 swaps", Kind: "swap"}, {Label: "66 calls", Kind: "call"}, {Label: "33 agent payments", Kind: "agent"}, {Label: "41 classic", Kind: "classic"}, {Label: "2 deploys", Kind: "deploy"}, {Label: "2 confidential", Kind: "confidential"}},
					InstructionsPct: 50, ReadWritePct: 38, CloseTime: "4.9s", Age: "20 seconds ago", SideMeta: "5 ledgers / min",
				},
			},
		},
		FeedJSON: string(feedJSON),
		Attention: vmv2.AttentionSectionData{
			Title: "Contracts that might need attention",
			Copy:  "Smart contracts on Stellar archive their storage after a while unless someone renews it. No other explorer shows you this.",
			Cards: []vmv2.AttentionCardData{
				{Kicker: "Blend · Lending Pool", Value: "~17 hours left", Tone: "is-bad", Body: "About 12,418 ledgers before the pool’s persistent state is archived. Users won’t be able to deposit, borrow, or repay until it’s restored.", BarColor: "#ea8c69", BarWidth: "12%", CTA: "How to extend this →"},
				{Kicker: "Soroswap · Router", Value: "~2 days 16 hours", Tone: "is-warn", Body: "Roughly 46,902 ledgers of life remain. Still comfortable, but worth renewing this week.", BarColor: "#d6aa45", BarWidth: "42%", CTA: "Remind me →"},
				{Kicker: "Phoenix · AMM", Value: "~4 days 12 hours", Tone: "is-warn", Body: "About 78,340 ledgers remaining — within our expiring-soon window.", BarColor: "#d6aa45", BarWidth: "68%", CTA: "Remind me →"},
				{Kicker: "Aquabot · Rewards", Value: "~14 days", Tone: "is-green", Body: "248,110 ledgers of runway. Healthy.", BarColor: "#64a86a", BarWidth: "92%", CTA: ""},
			},
		},
		Leaders: vmv2.LeadersSectionData{
			Title: "Who’s being used the most today",
			Copy:  "Ranked by how many times each contract was called in the last 24 hours — not dollars, not TVL. The actual activity.",
			Cards: []vmv2.LeaderCardData{
				{Label: "Most active", Value: "84,201", Entity: "Soroswap · Router", EntityMark: "S", EntityTone: "is-swap", Body: "412 different people and contracts called it — mostly swaps and pool deposits."},
				{Label: "Runner-up", Value: "52,318", Entity: "Blend · Lending Pool", EntityMark: "B", EntityTone: "is-call", Body: "187 unique callers — deposits and borrows dominated the mix."},
				{Label: "Third place", Value: "31,052", Entity: "Phoenix · AMM", EntityMark: "P", EntityTone: "is-deploy", Body: "143 unique callers. Volume down slightly from yesterday."},
				{Label: "Fastest growing", Value: "28,918", Entity: "x402 · Gateway", EntityMark: "x", EntityTone: "is-agent", Body: "1,204 unique agents — up 3× in a week. Mostly per-API-call micropayments."},
			},
		},
		Utilization: vmv2.UtilizationSectionData{
			Title: "How much of the network is being used",
			Copy:  "Every ledger has limits. When they fill up, fees rise sharply. Here’s where we sit right now.",
			Cards: []vmv2.UtilizationCardData{
				{Label: "Instruction budget", ValueMain: "64", ValueUnit: "% used", Tone: "is-warn", Body: "64.2M of 100M instructions this ledger. Above 60%, fees start to rise.", BarColor: "#d6aa45", BarWidth: "64%"},
				{Label: "Read / write", ValueMain: "60", ValueUnit: "% used", Tone: "", Body: "2.1 MB of 3.5 MB moved in and out of ledger storage.", BarColor: "#64a86a", BarWidth: "60%"},
				{Label: "Transaction size", ValueMain: "48", ValueUnit: "% used", Tone: "", Body: "Average transaction is 89 KB of the 128 KB allowed.", BarColor: "#64a86a", BarWidth: "48%"},
			},
		},
		FooterItems: []string{
			"Complete history from <b>protocol v20</b> · no gaps",
			"Lookups averaging <b>94 ms</b> · 99th percentile <b>312 ms</b>",
			"<b>412 known protocols</b> · named automatically",
			"Open source — <a href=\"#\">github.com/obsrvr/prism</a>",
		},
	}
}

type homeV2NetworkCfg struct {
	LedgerNumber string
	HeadlineHTML string
	HeroBody     string
	Placeholder  string
	FeedTitle    string
	FeedCopy     string
	FeedNote     string
}

func homeV2NetworkConfig(network string) homeV2NetworkCfg {
	switch network {
	case "testnet":
		return homeV2NetworkCfg{
			LedgerNumber: "14,284,112",
			HeadlineHTML: `The Stellar <span class="is-green">testnet</span> is healthy and lightly loaded right now — a good place to ship changes, test flows, and inspect Soroban behavior safely.`,
			HeroBody:     "You’re looking at Stellar’s Soroban-first explorer on testnet. Use it to validate contract calls, inspect events, and sanity-check app behavior before mainnet.",
			Placeholder:  "Paste a testnet hash, address, contract, or ask a question…",
			FeedTitle:    "What just happened on Testnet",
			FeedCopy:     "Each card is still a ledger — but here the emphasis is development flow: contract calls, deployments, and agent-style testing traffic.",
			FeedNote:     "testnet ledgers streaming every 5 seconds",
		}
	case "futurenet":
		return homeV2NetworkCfg{
			LedgerNumber: "8,104,552",
			HeadlineHTML: `The Stellar <span class="is-green">futurenet</span> is active right now — expect sharper protocol edges, experimental traffic, and faster-moving assumptions.`,
			HeroBody:     "This is the place to inspect upcoming behavior, try features earlier, and watch how contracts behave before those changes settle elsewhere.",
			Placeholder:  "Paste a futurenet hash, address, contract, or ask a question…",
			FeedTitle:    "What just happened on Futurenet",
			FeedCopy:     "Futurenet is where early integrations surface first. Expect a feed that skews toward deployments, experiments, and debugging-oriented traffic.",
			FeedNote:     "futurenet ledgers streaming every 5 seconds",
		}
	default:
		return homeV2NetworkCfg{
			LedgerNumber: "52,844,201",
			HeadlineHTML: `The Stellar network is <span class="is-green">healthy</span> and <em>busier than usual</em> right now — 187 transactions every 5 seconds, with 2,314 smart contracts active today.`,
			HeroBody:     "You’re looking at Stellar’s Soroban-first explorer. Every transaction below is classified and described in one sentence — swaps, contract calls, agent payments, and classic payments, all in plain English.",
			Placeholder:  "Paste a hash, an address, or ask a question…",
			FeedTitle:    "What just happened",
			FeedCopy:     "By default, each card is a ledger — the five-second heartbeat of Stellar. Pick a classification pill to drill into individual transactions.",
			FeedNote:     "a new ledger every 5 seconds",
		}
	}
}

type homeV2Feed struct {
	Ledgers      []homeV2FeedLedger      `json:"ledgers"`
	Transactions []homeV2FeedTransaction `json:"transactions"`
}

type homeV2FeedLedger struct {
	LedgerNumber     string                  `json:"ledgerNumber"`
	TransactionCount string                  `json:"transactionCount"`
	Meta             string                  `json:"meta"`
	Chips            []homeV2FeedLedgerChip  `json:"chips"`
	InstructionsPct  int                     `json:"instructionsPct"`
	ReadWritePct     int                     `json:"readWritePct"`
	CloseTime        string                  `json:"closeTime"`
	Age              string                  `json:"age"`
	SideMeta         string                  `json:"sideMeta"`
	Samples          []homeV2FeedTransaction `json:"samples"`
}

type homeV2FeedLedgerChip struct {
	Label string `json:"label"`
	Kind  string `json:"kind"`
}

type homeV2FeedTransaction struct {
	Kind     string `json:"kind"`
	Headline string `json:"headline"`
	Context  string `json:"context"`
	Entity   string `json:"entity"`
	Hash     string `json:"hash"`
	When     string `json:"when"`
}

func mockHomeV2Feed(network string) homeV2Feed {
	txs := []homeV2FeedTransaction{
		{Kind: "swap", Headline: "Someone swapped <b>1,240 XLM</b> for <b>287.60 USDC</b>", Context: "via the Soroswap router · rate 1 XLM ≈ 0.23 USDC · low slippage", Entity: "a whale on Soroswap", Hash: "4f82c1ab…9d10", When: "a few seconds ago"},
		{Kind: "call", Headline: "Blend’s lending pool processed a deposit", Context: "three sub-calls, 12.4M instructions · depositor added collateral in USDC", Entity: "submit() on Blend V2", Hash: "89c021dd…117a", When: "11 seconds ago"},
		{Kind: "agent", Headline: "An AI agent paid <b>$0.05 USDC</b> for an API call", Context: "x402 micro-payment · one of 3,041 today from this agent", Entity: "openai-proxy-a12", Hash: "19deab42…ac03", When: "18 seconds ago"},
		{Kind: "deploy", Headline: "A brand new smart contract was deployed", Context: "14 exported functions · WASM size 42 KB · init() in same transaction", Entity: "a developer", Hash: "c41a8b9e…7af2", When: "41 seconds ago"},
		{Kind: "confidential", Headline: "A confidential transfer was verified", Context: "the amount is hidden behind a zero-knowledge proof", Entity: "an anonymous sender", Hash: "8842fe11…00ca", When: "34 seconds ago"},
		{Kind: "classic", Headline: "Someone sent <b>420 XLM</b> to an exchange", Context: "memo says refund · no smart contract involved", Entity: "a classic payment", Hash: "ab0188f1…221d", When: "about a minute ago"},
		{Kind: "swap", Headline: "A stablecoin pair traded <b>500 USDC → 501.23 USDT</b>", Context: "two hops through SDEX · slippage 0.04%", Entity: "a small trader", Hash: "11fd992a…c442", When: "29 seconds ago"},
		{Kind: "call", Headline: "A borrower deposited into Blend’s pool", Context: "two sub-calls · 8.9M instructions", Entity: "deposit() on Blend V2", Hash: "db1ce780…0f44", When: "90 seconds ago"},
	}
	base := "52,844,201"
	sideMeta := "5 ledgers / min"
	titlePrefix := ""
	if network == "testnet" {
		base = "14,284,112"
		sideMeta = "testnet cadence"
		titlePrefix = "testnet · "
	} else if network == "futurenet" {
		base = "8,104,552"
		sideMeta = "futurenet cadence"
		titlePrefix = "futurenet · "
	}
	return homeV2Feed{
		Ledgers: []homeV2FeedLedger{
			{LedgerNumber: base, TransactionCount: "212", Meta: titlePrefix + "across 424 operations · closed by sdf-validator-1", Chips: []homeV2FeedLedgerChip{{Label: "50 swaps", Kind: "swap"}, {Label: "67 calls", Kind: "call"}, {Label: "54 agent payments", Kind: "agent"}, {Label: "33 classic", Kind: "classic"}, {Label: "2 deploys", Kind: "deploy"}, {Label: "2 confidential", Kind: "confidential"}}, InstructionsPct: 61, ReadWritePct: 35, CloseTime: "4.9s", Age: "just now", SideMeta: sideMeta, Samples: []homeV2FeedTransaction{txs[1], txs[0], txs[2]}},
			{LedgerNumber: decLedger(base, 1), TransactionCount: "226", Meta: titlePrefix + "across 452 operations · closed by whale-stack", Chips: []homeV2FeedLedgerChip{{Label: "42 swaps", Kind: "swap"}, {Label: "75 calls", Kind: "call"}, {Label: "37 agent payments", Kind: "agent"}, {Label: "67 classic", Kind: "classic"}, {Label: "3 deploys", Kind: "deploy"}, {Label: "1 confidential", Kind: "confidential"}}, InstructionsPct: 76, ReadWritePct: 45, CloseTime: "4.9s", Age: "5 seconds ago", SideMeta: sideMeta, Samples: []homeV2FeedTransaction{txs[6], txs[7], txs[5]}},
			{LedgerNumber: decLedger(base, 2), TransactionCount: "187", Meta: titlePrefix + "across 561 operations · closed by publicnode", Chips: []homeV2FeedLedgerChip{{Label: "35 swaps", Kind: "swap"}, {Label: "70 calls", Kind: "call"}, {Label: "30 agent payments", Kind: "agent"}, {Label: "52 classic", Kind: "classic"}, {Label: "1 deploy", Kind: "deploy"}}, InstructionsPct: 67, ReadWritePct: 36, CloseTime: "4.9s", Age: "10 seconds ago", SideMeta: sideMeta, Samples: []homeV2FeedTransaction{txs[3], txs[4], txs[5]}},
			{LedgerNumber: decLedger(base, 3), TransactionCount: "199", Meta: titlePrefix + "across 388 operations · closed by lobstr", Chips: []homeV2FeedLedgerChip{{Label: "50 swaps", Kind: "swap"}, {Label: "62 calls", Kind: "call"}, {Label: "55 agent payments", Kind: "agent"}, {Label: "32 classic", Kind: "classic"}, {Label: "1 deploy", Kind: "deploy"}}, InstructionsPct: 50, ReadWritePct: 50, CloseTime: "4.9s", Age: "15 seconds ago", SideMeta: sideMeta, Samples: []homeV2FeedTransaction{txs[2], txs[0], txs[5]}},
			{LedgerNumber: decLedger(base, 4), TransactionCount: "197", Meta: titlePrefix + "across 788 operations · closed by coinbase-cloud", Chips: []homeV2FeedLedgerChip{{Label: "53 swaps", Kind: "swap"}, {Label: "66 calls", Kind: "call"}, {Label: "33 agent payments", Kind: "agent"}, {Label: "41 classic", Kind: "classic"}, {Label: "2 deploys", Kind: "deploy"}, {Label: "2 confidential", Kind: "confidential"}}, InstructionsPct: 50, ReadWritePct: 38, CloseTime: "4.9s", Age: "20 seconds ago", SideMeta: sideMeta, Samples: []homeV2FeedTransaction{txs[4], txs[6], txs[1]}},
		},
		Transactions: txs,
	}
}

func decLedger(base string, n int) string {
	var v int
	for _, ch := range base {
		if ch >= '0' && ch <= '9' {
			v = v*10 + int(ch-'0')
		}
	}
	return fmt.Sprintf("%d", v-n)
}
