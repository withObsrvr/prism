package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"math"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/withObsrvr/prism/internal/gateway"
	componentsv2 "github.com/withObsrvr/prism/internal/templates/v2/components"
	pagesv2 "github.com/withObsrvr/prism/internal/templates/v2/pages"
	vmv2 "github.com/withObsrvr/prism/internal/templates/v2/viewmodel"
)

func (h *Handlers) HomeV2(w http.ResponseWriter, r *http.Request) {
	network := networkFromRequest(r)
	data := mockHomeV2Data(network)
	if h.useLiveData(r) {
		if live, err := h.buildHomeV2Data(r, network); err == nil {
			data = live
		} else {
			h.Logger.Warn("live home v2 feed failed, falling back to mock", "network", network, "error", err)
		}
	}
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
			RoleCopyJSON: mustJSON(buildMockHomeV2RoleCopy(network)),
		},
		Prompt: vmv2.PromptData{Placeholder: cfg.Placeholder},
		Alert: vmv2.AlertData{
			Title: "Three contracts are running out of room",
			Body:  "Blend’s lending pool, Soroswap’s router, and Phoenix’s AMM have under four days of persistent-storage life left. If nobody extends them, they’ll be archived and stop responding.",
			Meta:  "Why this matters: contract operators need to extend TTL before it hits zero, otherwise their users see failures.",
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
					IncludedOperationCount: "424", SuccessfulOperationCount: "420", FailedOperationCount: "4",
					Meta:            "with 424 operations",
					Introducer:      "sdf-validator-1",
					Chips:           []componentsv2.LedgerMetricChip{{Label: "50 swaps", Kind: "swap"}, {Label: "67 calls", Kind: "call"}, {Label: "54 agent payments", Kind: "agent"}, {Label: "33 classic", Kind: "classic"}, {Label: "2 deploys", Kind: "deploy"}, {Label: "2 confidential", Kind: "confidential"}},
					InstructionsPct: 61, ReadWritePct: 35, CloseTime: "4.9s", Age: "just now", SideMeta: "5 ledgers / min",
				},
				{
					LedgerNumber: "52,844,200", TransactionCount: "226",
					IncludedOperationCount: "452", SuccessfulOperationCount: "450", FailedOperationCount: "2",
					Meta:            "with 452 operations",
					Introducer:      "whale-stack",
					Chips:           []componentsv2.LedgerMetricChip{{Label: "42 swaps", Kind: "swap"}, {Label: "75 calls", Kind: "call"}, {Label: "37 agent payments", Kind: "agent"}, {Label: "67 classic", Kind: "classic"}, {Label: "3 deploys", Kind: "deploy"}, {Label: "1 confidential", Kind: "confidential"}},
					InstructionsPct: 76, ReadWritePct: 45, CloseTime: "4.9s", Age: "5 seconds ago", SideMeta: "5 ledgers / min",
				},
				{
					LedgerNumber: "52,844,199", TransactionCount: "187",
					IncludedOperationCount: "561", SuccessfulOperationCount: "558", FailedOperationCount: "3",
					Meta:            "with 561 operations",
					Introducer:      "publicnode",
					Chips:           []componentsv2.LedgerMetricChip{{Label: "35 swaps", Kind: "swap"}, {Label: "70 calls", Kind: "call"}, {Label: "30 agent payments", Kind: "agent"}, {Label: "52 classic", Kind: "classic"}, {Label: "1 deploy", Kind: "deploy"}},
					InstructionsPct: 67, ReadWritePct: 36, CloseTime: "4.9s", Age: "10 seconds ago", SideMeta: "5 ledgers / min",
				},
				{
					LedgerNumber: "52,844,198", TransactionCount: "199",
					IncludedOperationCount: "388", SuccessfulOperationCount: "385", FailedOperationCount: "3",
					Meta:            "with 388 operations",
					Introducer:      "lobstr",
					Chips:           []componentsv2.LedgerMetricChip{{Label: "50 swaps", Kind: "swap"}, {Label: "62 calls", Kind: "call"}, {Label: "55 agent payments", Kind: "agent"}, {Label: "32 classic", Kind: "classic"}, {Label: "1 deploy", Kind: "deploy"}},
					InstructionsPct: 50, ReadWritePct: 50, CloseTime: "4.9s", Age: "15 seconds ago", SideMeta: "5 ledgers / min",
				},
				{
					LedgerNumber: "52,844,197", TransactionCount: "197",
					IncludedOperationCount: "788", SuccessfulOperationCount: "783", FailedOperationCount: "5",
					Meta:            "with 788 operations",
					Introducer:      "coinbase-cloud",
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
				{Kicker: "Phoenix · AMM", Value: "~4 days 12 hours", Tone: "is-warn", Body: "About 78,340 ledgers remaining, within our expiring-soon window.", BarColor: "#d6aa45", BarWidth: "68%", CTA: "Remind me →"},
				{Kicker: "Aquabot · Rewards", Value: "~14 days", Tone: "is-green", Body: "248,110 ledgers of runway. Healthy.", BarColor: "#64a86a", BarWidth: "92%", CTA: ""},
			},
		},
		Leaders: vmv2.LeadersSectionData{
			Title: "Who’s being used the most today",
			Copy:  "Ranked by how many times each contract was called in the last 24 hours, not dollars, not TVL. The actual activity.",
			Cards: []vmv2.LeaderCardData{
				{Label: "Most active", Value: "84,201", Entity: "Soroswap · Router", EntityMark: "S", EntityTone: "is-swap", Body: "412 different people and contracts called it, mostly swaps and pool deposits."},
				{Label: "Runner-up", Value: "52,318", Entity: "Blend · Lending Pool", EntityMark: "B", EntityTone: "is-call", Body: "187 unique callers, deposits and borrows dominated the mix."},
				{Label: "Third place", Value: "31,052", Entity: "Phoenix · AMM", EntityMark: "P", EntityTone: "is-deploy", Body: "143 unique callers. Volume down slightly from yesterday."},
				{Label: "Fastest growing", Value: "28,918", Entity: "x402 · Gateway", EntityMark: "x", EntityTone: "is-agent", Body: "1,204 unique agents, up 3× in a week. Mostly per-API-call micropayments."},
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
			"Open source, <a href=\"#\">github.com/obsrvr/prism</a>",
		},
	}
}

const (
	homeV2SorobanInstructionLimit = int64(100_000_000)
	homeV2SorobanReadWriteLimit   = int64(3_500_000)

	homeV2HomeSummaryTimeout      = 750 * time.Millisecond
	homeV2RecentLedgersTimeout    = 2 * time.Second
	homeV2PerLedgerSummaryTimeout = 1500 * time.Millisecond
)

type homeV2NetworkCfg struct {
	LedgerNumber string
	HeadlineHTML string
	HeroBody     string
	Placeholder  string
	FeedTitle    string
	FeedCopy     string
	FeedNote     string
}

type homeV2HeroRoleCopy struct {
	Headline string `json:"headline"`
	Body     string `json:"body"`
}

func homeV2RoleNetworkLabels(network string) (string, string, string) {
	switch {
	case strings.EqualFold(network, "testnet"):
		return "The Stellar testnet", "Testnet Soroban", "You’re viewing <b>Testnet</b>. "
	case strings.EqualFold(network, "futurenet"):
		return "The Stellar futurenet", "Futurenet Soroban", "You’re viewing <b>Futurenet</b>. "
	default:
		return "The Stellar network", "Soroban", ""
	}
}

func mustJSON(v any) string {
	body, err := json.Marshal(v)
	if err != nil {
		return "{}"
	}
	return string(body)
}

func homeV2NetworkConfig(network string) homeV2NetworkCfg {
	switch network {
	case "testnet":
		return homeV2NetworkCfg{
			LedgerNumber: "14,284,112",
			HeadlineHTML: `The Stellar <span class="is-green">testnet</span> is healthy and lightly loaded right now, a good place to ship changes, test flows, and inspect Soroban behavior safely.`,
			HeroBody:     "You’re looking at Stellar’s Soroban-first explorer on testnet. Use it to validate contract calls, inspect events, and sanity-check app behavior before mainnet.",
			Placeholder:  "Paste a testnet hash, address, contract, or ask a question…",
			FeedTitle:    "What just happened on Testnet",
			FeedCopy:     "Each card is still a ledger, but here the emphasis is development flow: contract calls, deployments, and agent-style testing traffic.",
			FeedNote:     "testnet ledgers streaming every 5 seconds",
		}
	case "futurenet":
		return homeV2NetworkCfg{
			LedgerNumber: "8,104,552",
			HeadlineHTML: `The Stellar <span class="is-green">futurenet</span> is active right now, expect sharper protocol edges, experimental traffic, and faster-moving assumptions.`,
			HeroBody:     "This is the place to inspect upcoming behavior, try features earlier, and watch how contracts behave before those changes settle elsewhere.",
			Placeholder:  "Paste a futurenet hash, address, contract, or ask a question…",
			FeedTitle:    "What just happened on Futurenet",
			FeedCopy:     "Futurenet is where early integrations surface first. Expect a feed that skews toward deployments, experiments, and debugging-oriented traffic.",
			FeedNote:     "futurenet ledgers streaming every 5 seconds",
		}
	default:
		return homeV2NetworkCfg{
			LedgerNumber: "52,844,201",
			HeadlineHTML: `The Stellar network is <span class="is-green">healthy</span> and <em>busier than usual</em> right now, 187 transactions every 5 seconds, with 2,314 smart contracts active today.`,
			HeroBody:     "You’re looking at Stellar’s Soroban-first explorer. Every transaction below is classified and described in one sentence, swaps, contract calls, agent payments, and classic payments, all in plain English.",
			Placeholder:  "Paste a hash, an address, or ask a question…",
			FeedTitle:    "What just happened",
			FeedCopy:     "By default, each card is a ledger, the five-second heartbeat of Stellar. Pick a classification pill to drill into individual transactions.",
			FeedNote:     "a new ledger every 5 seconds",
		}
	}
}

func buildMockHomeV2RoleCopy(network string) map[string]homeV2HeroRoleCopy {
	networkLabel, devLabel, bodyPrefix := homeV2RoleNetworkLabels(network)
	return map[string]homeV2HeroRoleCopy{
		"curious": {
			Headline: fmt.Sprintf(`%s is <span class="is-green">healthy</span> and <em>busier than usual</em> right now, 187 transactions every 5 seconds, with 2,314 smart contracts active today.`, networkLabel),
			Body:     bodyPrefix + `You’re looking at Stellar’s <b>Soroban-first</b> explorer. Every transaction below is classified and described in one sentence, swaps, contract calls, agent payments, and classic payments, all in plain English. <a class="v2-linkish" href="#">How this works →</a>`,
		},
		"developer": {
			Headline: fmt.Sprintf(`%s looks <span class="is-green">healthy</span> right now, instruction budget is <em>64%% used</em>, read / write is <em>60%%</em>, and 2,314 contracts were active in the last day.`, devLabel),
			Body:     bodyPrefix + `Jump to any contract by pasting its C… address. We decode function names, sub-calls, and events, and we surface TTL status that most explorers miss. <a class="v2-linkish" href="#">API docs →</a>`,
		},
		"operator": {
			Headline: `3 contracts need <em>TTL attention</em> right now. The worst case has <span style="color:var(--v2-orange)">under 17 hours</span> remaining.`,
			Body:     bodyPrefix + `Paste your contract address to see runway, renewal state, and the exact restore path before storage expires. <a class="v2-linkish" href="#">TTL monitoring alerts →</a>`,
		},
		"compliance": {
			Headline: `<em>Activity looks normal right now.</em> No anomaly burst is flagged, and agent-driven volume is up <em>22%</em> week over week.`,
			Body:     bodyPrefix + `Every transaction is classified, DEX swap, contract call, agent payment, or classic transfer. Paste an address to review its recent pattern with counterparties labeled. <a class="v2-linkish" href="#">Compliance workflows →</a>`,
		},
		"user": {
			Headline: `The network looks <span class="is-green">healthy</span>. Paste a transaction hash and we’ll explain what happened in one sentence.`,
			Body:     bodyPrefix + `No jargon. No hex. Just a plain-English answer with links to the raw details if you want them. <a class="v2-linkish" href="#">How Prism reads transactions →</a>`,
		},
	}
}

func buildHomeV2RoleCopy(summary *gateway.HomeSummaryResponse) map[string]homeV2HeroRoleCopy {
	networkLabel, devLabel, bodyPrefix := homeV2RoleNetworkLabels(summary.Network)
	statusWord := homeV2FirstNonEmpty(summary.Hero.Health.Status, "healthy")
	activityBand := homeV2FirstNonEmpty(summary.Hero.Health.ActivityBand, "normal")
	activityPhrase := "active"
	switch {
	case summary.Hero.Health.ActivityBand == "busy":
		activityPhrase = "busier than usual"
	case summary.Hero.Health.ActivityBand == "quiet" || summary.Hero.Health.LoadBand == "light":
		activityPhrase = "lightly loaded"
	case summary.Hero.Health.ActivityBand == "normal":
		activityPhrase = "running normally"
	}
	txAvg := gateway.FormatNumber(summary.Hero.Cadence.TxPerLedgerRecentAvg)
	if summary.Hero.Cadence.TxPerLedgerRecentAvg == 0 {
		txAvg = gateway.FormatNumber(summary.Hero.LatestLedger.TransactionCount)
	}
	contracts := gateway.FormatNumber(summary.Hero.Contracts.Active24h)
	if summary.Hero.Contracts.Active24h == 0 {
		contracts = "a few"
	}
	expiringContracts := summary.Alert.AffectedContractCount
	if expiringContracts == 0 {
		expiringContracts = summary.Hero.TTL.ExpiringContractCount
	}
	worstHours := summary.Alert.WorstRemainingHours
	if worstHours == 0 {
		worstHours = summary.Hero.TTL.WorstRemainingHours
	}
	topNames := make([]string, 0, 2)
	for _, c := range summary.ContractsNeedingAttention {
		name := homeV2FirstNonEmpty(c.ProtocolName, c.ContractName)
		if name != "" {
			topNames = append(topNames, name)
		}
		if len(topNames) == 2 {
			break
		}
	}
	operatorHeadline := "No urgent <em>TTL risks</em> are visible right now."
	operatorBody := bodyPrefix + `Paste your contract address to check runway, renewal state, and the restore path if anything changes. <a class="v2-linkish" href="#">TTL monitoring alerts →</a>`
	if expiringContracts > 0 {
		operatorHeadline = fmt.Sprintf(`%s contracts need <em>TTL attention</em> right now.`, gateway.FormatNumber(expiringContracts))
		if worstHours > 0 {
			operatorHeadline = fmt.Sprintf(`%s contracts need <em>TTL attention</em> right now. The worst case has <span style="color:var(--v2-orange)">under %s</span> remaining.`, gateway.FormatNumber(expiringContracts), humanizeHours(worstHours))
		}
		if len(topNames) > 0 {
			operatorBody = bodyPrefix + fmt.Sprintf(`The closest contracts include %s. Paste your contract address to see runway, renewal state, and the exact restore path before storage expires. <a class="v2-linkish" href="#">TTL monitoring alerts →</a>`, html.EscapeString(strings.Join(topNames, " and ")))
		} else {
			operatorBody = bodyPrefix + `Paste your contract address to see runway, renewal state, and the exact restore path before storage expires. <a class="v2-linkish" href="#">TTL monitoring alerts →</a>`
		}
	}
	anomalyText := "No anomaly burst is flagged right now"
	if summary.Hero.Trends.AnomalyDetected {
		anomalyText = "An anomaly burst is flagged and worth a closer look"
	}
	agentTrendText := "agent trend unavailable"
	if summary.Hero.Trends.AgentActivityWoWPct != 0 {
		agentTrendText = fmt.Sprintf("agent-driven volume is up <em>%s</em> week over week", html.EscapeString(fmt.Sprintf("%.0f%%", summary.Hero.Trends.AgentActivityWoWPct)))
	}
	mixParts := []string{}
	if summary.Hero.ActivityMix.SwapTx24h > 0 {
		mixParts = append(mixParts, gateway.FormatNumber(summary.Hero.ActivityMix.SwapTx24h)+" swaps")
	}
	if summary.Hero.ActivityMix.ContractCallTx24h > 0 {
		mixParts = append(mixParts, gateway.FormatNumber(summary.Hero.ActivityMix.ContractCallTx24h)+" contract calls")
	}
	if summary.Hero.ActivityMix.AgentTx24h > 0 {
		mixParts = append(mixParts, gateway.FormatNumber(summary.Hero.ActivityMix.AgentTx24h)+" agent payments")
	}
	complianceBody := bodyPrefix + `Every transaction is classified, DEX swap, contract call, agent payment, or classic transfer. Paste an address to review its recent pattern with counterparties labeled. <a class="v2-linkish" href="#">Compliance workflows →</a>`
	if len(mixParts) > 0 {
		complianceBody = bodyPrefix + fmt.Sprintf(`Recent labeled activity includes %s. Paste an address to review its pattern with counterparties and transaction types already classified. <a class="v2-linkish" href="#">Compliance workflows →</a>`, html.EscapeString(strings.Join(mixParts, " · ")))
	}
	developerHeadline := fmt.Sprintf(`%s looks <span class="is-green">%s</span> right now, instruction budget is <em>%s%% used</em>, read / write is <em>%s%%</em>, and %s contracts were active in the last day.`, devLabel, html.EscapeString(statusWord), formatPercentMain(summary.Utilization.InstructionPct), formatPercentMain(summary.Utilization.ReadWritePct), contracts)
	if summary.Utilization.InstructionPct == 0 && summary.Utilization.ReadWritePct == 0 {
		developerHeadline = fmt.Sprintf(`%s looks <span class="is-green">%s</span> right now, with %s contracts active in the last day.`, devLabel, html.EscapeString(statusWord), contracts)
	}
	return map[string]homeV2HeroRoleCopy{
		"curious": {
			Headline: fmt.Sprintf(`%s is <span class="is-green">%s</span> and <em>%s</em> right now, %s transactions every 5 seconds, with %s smart contracts active today.`, networkLabel, html.EscapeString(statusWord), html.EscapeString(activityPhrase), txAvg, contracts),
			Body:     bodyPrefix + `You’re looking at Stellar’s <b>Soroban-first</b> explorer. Every transaction below is classified and explained in one sentence, swaps, contract calls, agent payments, and classic payments in plain English. <a class="v2-linkish" href="#">How this works →</a>`,
		},
		"developer": {
			Headline: developerHeadline,
			Body:     bodyPrefix + `Jump to any contract by pasting its C… address. We decode function names, sub-calls, and events, and we surface TTL state that most explorers miss. <a class="v2-linkish" href="#">API docs →</a>`,
		},
		"operator": {
			Headline: operatorHeadline,
			Body:     operatorBody,
		},
		"compliance": {
			Headline: fmt.Sprintf(`<em>Activity looks %s right now.</em> %s, and %s.`, html.EscapeString(activityBand), html.EscapeString(anomalyText), agentTrendText),
			Body:     complianceBody,
		},
		"user": {
			Headline: fmt.Sprintf(`The network looks <span class="is-green">%s</span>. Paste a transaction hash and we’ll explain what happened in one sentence.`, html.EscapeString(statusWord)),
			Body:     bodyPrefix + `No jargon. No hex. Just a plain-English answer with links to the raw details if you want them. <a class="v2-linkish" href="#">How Prism reads transactions →</a>`,
		},
	}
}

func (h *Handlers) buildHomeV2FeedData(r *http.Request, network string) (homeV2FeedPayloadHeader, []vmv2.LedgerRowData, homeV2Feed, error) {
	if h.Gateway == nil {
		return homeV2FeedPayloadHeader{}, nil, homeV2Feed{}, fmt.Errorf("gateway unavailable")
	}

	recentCtx, recentCancel := context.WithTimeout(r.Context(), homeV2RecentLedgersTimeout)
	defer recentCancel()
	recent, err := h.Gateway.GetSilverRecentLedgers(recentCtx, network, 6)
	if err != nil {
		return homeV2FeedPayloadHeader{}, nil, homeV2Feed{}, err
	}
	if recent == nil || len(recent.Ledgers) == 0 {
		return homeV2FeedPayloadHeader{}, nil, homeV2Feed{}, fmt.Errorf("no recent ledgers returned")
	}

	summaries := make([]*gateway.LedgerFeedSummary, len(recent.Ledgers))
	if r.URL.Query().Get("enrich_ledgers") == "true" {
		summaries = h.loadHomeV2LedgerSummaries(r.Context(), network, recent.Ledgers)
	}

	rows := make([]vmv2.LedgerRowData, 0, len(recent.Ledgers))
	feedLedgers := make([]homeV2FeedLedger, 0, len(recent.Ledgers))
	feedTxs := make([]homeV2FeedTransaction, 0, len(recent.Ledgers)*3)
	for i, ledger := range recent.Ledgers {
		var next *gateway.RecentLedger
		if i+1 < len(recent.Ledgers) {
			next = &recent.Ledgers[i+1]
		}
		row, feedLedger, ledgerTxs := buildHomeV2FeedLedger(network, ledger, next, summaries[i])
		rows = append(rows, row)
		feedLedgers = append(feedLedgers, feedLedger)
		feedTxs = append(feedTxs, ledgerTxs...)
	}

	header := homeV2FeedPayloadHeader{}
	if first := recent.Ledgers[0]; first.LedgerSequence > 0 {
		header.LedgerNumber = gateway.FormatNumber(first.LedgerSequence)
		if t, ok := parseGatewayTime(first.ClosedAt); ok {
			header.AgeLabel = gateway.FormatAge(t)
		}
	}
	return header, rows, homeV2Feed{Ledgers: feedLedgers, Transactions: feedTxs}, nil
}

func (h *Handlers) loadHomeV2LedgerSummaries(ctx context.Context, network string, ledgers []gateway.RecentLedger) []*gateway.LedgerFeedSummary {
	summaries := make([]*gateway.LedgerFeedSummary, len(ledgers))
	var wg sync.WaitGroup
	for i := range ledgers {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			summaryCtx, cancel := context.WithTimeout(ctx, homeV2PerLedgerSummaryTimeout)
			defer cancel()
			summary, err := h.Gateway.GetSilverLedgerFeedSummary(summaryCtx, network, ledgers[i].LedgerSequence)
			if err != nil {
				h.Logger.Warn("home v2 summary fetch failed", "sequence", ledgers[i].LedgerSequence, "error", err)
				return
			}
			summaries[i] = summary
		}()
	}
	wg.Wait()
	return summaries
}

func (h *Handlers) buildHomeV2Data(r *http.Request, network string) (vmv2.HomeData, error) {
	if h.Gateway == nil {
		return vmv2.HomeData{}, fmt.Errorf("gateway unavailable")
	}

	data := mockHomeV2Data(network)
	summaryCtx, summaryCancel := context.WithTimeout(r.Context(), homeV2HomeSummaryTimeout)
	if summary, err := h.Gateway.GetHomeSummary(summaryCtx, network); err == nil && summary != nil {
		applyHomeSummaryV2(&data, summary)
		if r.URL.Query().Get("validate_ttl") == "true" {
			h.validateHomeV2TTLAttention(r, network, &data, summary)
		}
	} else if err != nil {
		h.Logger.Warn("home v2 home-summary fetch failed", "network", network, "error", err)
	}
	summaryCancel()
	header, rows, feed, err := h.buildHomeV2FeedData(r, network)
	if err != nil {
		return vmv2.HomeData{}, err
	}
	if len(rows) > 0 {
		data.LedgerFeed.Rows = rows
	}
	if header.LedgerNumber != "" {
		data.Header.LedgerNumber = header.LedgerNumber
	}
	if header.AgeLabel != "" {
		data.Header.AgeLabel = header.AgeLabel
	}
	feedJSON, _ := json.Marshal(feed)
	data.FeedJSON = string(feedJSON)
	data.FeedLive = true
	return data, nil
}

func applyHomeSummaryV2(data *vmv2.HomeData, summary *gateway.HomeSummaryResponse) {
	if data == nil || summary == nil {
		return
	}
	if summary.Header.LatestLedgerSequence > 0 {
		data.Header.LedgerNumber = gateway.FormatNumber(summary.Header.LatestLedgerSequence)
	}
	if t, ok := parseGatewayTime(summary.Header.LatestLedgerClosedAt); ok {
		data.Header.AgeLabel = gateway.FormatAge(t)
	}

	data.Hero = buildHomeV2Hero(summary)
	data.Alert = buildHomeV2Alert(summary)
	data.Attention = buildHomeV2Attention(summary)
	data.Leaders = buildHomeV2Leaders(summary)
	data.Utilization = buildHomeV2Utilization(summary)
	applyHomeV2Meta(data, summary)
}

func buildHomeV2Hero(summary *gateway.HomeSummaryResponse) vmv2.HeroData {
	networkLabel := "The Stellar network"
	networkBody := "Stellar"
	placeholder := ""
	if strings.EqualFold(summary.Network, "testnet") {
		networkLabel = `The Stellar <span class="is-green">testnet</span>`
		networkBody = "Testnet"
		placeholder = " on testnet"
	}
	statusWord := "healthy"
	if summary.Hero.Health.Status != "" {
		statusWord = summary.Hero.Health.Status
	}
	activityPhrase := "active"
	if summary.Hero.Health.ActivityBand == "busy" {
		activityPhrase = "busier than usual"
	} else if summary.Hero.Health.ActivityBand == "quiet" || summary.Hero.Health.LoadBand == "light" {
		activityPhrase = "lightly loaded"
	} else if summary.Hero.Health.ActivityBand == "normal" {
		activityPhrase = "running normally"
	}
	txAvg := gateway.FormatNumber(summary.Hero.Cadence.TxPerLedgerRecentAvg)
	if summary.Hero.Cadence.TxPerLedgerRecentAvg == 0 {
		txAvg = gateway.FormatNumber(summary.Hero.LatestLedger.TransactionCount)
	}
	contracts := gateway.FormatNumber(summary.Hero.Contracts.Active24h)
	headline := fmt.Sprintf(`%s is <span class="is-green">%s</span> and <em>%s</em> right now, %s transactions every 5 seconds, with %s smart contracts active today.`, networkLabel, html.EscapeString(statusWord), html.EscapeString(activityPhrase), txAvg, contracts)
	body := fmt.Sprintf(`You’re looking at %s’s Soroban-first explorer%s. Every transaction below is classified and described in one sentence, swaps, contract calls, and classic payments in plain English.`, networkBody, placeholder)
	return vmv2.HeroData{
		Eyebrow:      "What brings you here?",
		HeadlineHTML: headline,
		Body:         body,
		RoleCopyJSON: mustJSON(buildHomeV2RoleCopy(summary)),
	}
}

func buildHomeV2Alert(summary *gateway.HomeSummaryResponse) vmv2.AlertData {
	count := summary.Alert.AffectedContractCount
	if count == 0 {
		count = summary.Hero.TTL.ExpiringContractCount
	}
	hours := summary.Alert.WorstRemainingHours
	if hours == 0 {
		hours = summary.Hero.TTL.WorstRemainingHours
	}
	title := fmt.Sprintf("%s contracts need TTL attention", gateway.FormatNumber(count))
	if count == 1 {
		title = "One contract needs TTL attention"
	}
	names := summary.Alert.TopContracts
	body := fmt.Sprintf("%s contracts on %s are nearing expiration, with the worst case under %s remaining.", gateway.FormatNumber(count), summary.Network, humanizeHours(hours))
	if len(names) > 0 {
		body = fmt.Sprintf("%s contracts on %s are nearing expiration, including %s, with the worst case under %s remaining.", gateway.FormatNumber(count), summary.Network, strings.Join(names, ", "), humanizeHours(hours))
	}
	meta := "Why this matters: contract operators need to extend TTL before it hits zero, otherwise their users see failures."
	cta := "Review →"
	return vmv2.AlertData{Title: title, Body: body, Meta: meta, CTA: cta}
}

func buildHomeV2Attention(summary *gateway.HomeSummaryResponse) vmv2.AttentionSectionData {
	data := vmv2.AttentionSectionData{
		Title: "Contracts that might need attention",
		Copy:  "Smart contracts on Stellar archive their storage after a while unless someone renews it. No other explorer shows you this.",
	}
	for _, c := range summary.ContractsNeedingAttention {
		if len(data.Cards) == 4 {
			break
		}
		name := homeV2FirstNonEmpty(c.ProtocolName, c.ContractName, gateway.ShortAddress(c.ContractID))
		if c.ContractName != "" && c.ProtocolName != "" && c.ContractName != c.ProtocolName {
			name = c.ProtocolName + " · " + c.ContractName
		}
		barWidth := fmt.Sprintf("%.0f%%", clampFloat(c.RunwayPct, 0, 100))
		body := fmt.Sprintf("About %s ledgers of runway remain. Status: %s.", gateway.FormatNumber(c.RemainingLedgers), strings.ReplaceAll(c.Status, "_", " "))
		if c.RemainingHuman != "" {
			body = fmt.Sprintf("About %s ledgers of runway remain, roughly %s.", gateway.FormatNumber(c.RemainingLedgers), c.RemainingHuman)
		}
		data.Cards = append(data.Cards, vmv2.AttentionCardData{
			Kicker:   name,
			Value:    fmt.Sprintf("~%s left", homeV2FirstNonEmpty(c.RemainingHuman, humanizeHours(c.RemainingHours))),
			Tone:     attentionTone(c.Severity),
			Body:     body,
			BarColor: attentionBarColor(c.Severity),
			BarWidth: barWidth,
			CTA:      "How to extend this →",
			Href:     homeV2ContractHref(c.ContractID),
		})
	}
	return data
}

func (h *Handlers) validateHomeV2TTLAttention(r *http.Request, network string, data *vmv2.HomeData, summary *gateway.HomeSummaryResponse) {
	if h == nil || h.Gateway == nil || data == nil || summary == nil || len(summary.ContractsNeedingAttention) == 0 {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 2200*time.Millisecond)
	defer cancel()

	validated := vmv2.AttentionSectionData{
		Title: "Persistent storage that might need attention",
		Copy:  "Prism only shows TTL warnings here after checking gateway storage rows and excluding temporary entries, which often expire by design.",
	}
	validatedCount := int64(0)
	var worstRemaining int64
	validatedNames := []string{}
	for _, c := range summary.ContractsNeedingAttention {
		storage, err := h.Gateway.GetContractStorage(ctx, network, c.ContractID, 100)
		if err != nil || storage == nil {
			h.Logger.Warn("home v2 TTL validation failed", "network", network, "contract", c.ContractID, "error", err)
			continue
		}
		remaining := nearestPersistentStorageRunway(storage.Entries, summary.Header.LatestLedgerSequence)
		if remaining <= 0 {
			continue
		}
		validatedCount++
		if worstRemaining == 0 || remaining < worstRemaining {
			worstRemaining = remaining
		}
		copy := c
		copy.RemainingLedgers = remaining
		copy.RemainingHours = int64(math.Ceil(float64(remaining) * 5 / 3600))
		copy.RemainingHuman = humanizeLedgerRunway(remaining)
		copy.RunwayPct = homeV2RunwayPctFromLedgers(remaining)
		name := homeV2FirstNonEmpty(copy.ProtocolName, copy.ContractName, gateway.ShortAddress(copy.ContractID))
		if copy.ContractName != "" && copy.ProtocolName != "" && copy.ContractName != copy.ProtocolName {
			name = copy.ProtocolName + " · " + copy.ContractName
		}
		if len(validatedNames) < 3 {
			validatedNames = append(validatedNames, name)
		}
		if len(validated.Cards) < 4 {
			validated.Cards = append(validated.Cards, vmv2.AttentionCardData{
				Kicker:   name,
				Value:    fmt.Sprintf("~%s left", copy.RemainingHuman),
				Tone:     attentionTone(copy.Severity),
				Body:     fmt.Sprintf("Nearest persistent storage entry has about %s ledgers of runway, roughly %s.", gateway.FormatNumber(copy.RemainingLedgers), copy.RemainingHuman),
				BarColor: attentionBarColor(copy.Severity),
				BarWidth: fmt.Sprintf("%.0f%%", clampFloat(copy.RunwayPct, 0, 100)),
				CTA:      "Review storage →",
				Href:     homeV2ContractHref(copy.ContractID),
			})
		}
	}
	if validatedCount == 0 {
		validated.Copy = "Gateway currently reports no persistent storage entries in the expiring window after validation. Temporary entries are excluded because they often expire by design."
		data.Alert = vmv2.AlertData{
			Title: "No validated persistent TTL risk right now",
			Body:  "The gateway summary contained near-expiration rows, but validation showed they were temporary storage or stale TTL rows rather than persistent storage requiring operator action.",
			Meta:  "Why this matters: temporary entries can expire normally; persistent entries are the ones operators usually need to extend or restore.",
			CTA:   "Review →",
		}
	} else {
		title := fmt.Sprintf("%s persistent storage entries need TTL attention", gateway.FormatNumber(validatedCount))
		if validatedCount == 1 {
			title = "One persistent storage entry needs TTL attention"
		}
		body := fmt.Sprintf("%s validated persistent storage entries on %s are nearing expiration after excluding temporary storage. The closest runway is about %s.", gateway.FormatNumber(validatedCount), network, humanizeLedgerRunway(worstRemaining))
		if len(validatedNames) > 0 {
			body = fmt.Sprintf("%s validated persistent storage entries on %s are nearing expiration, including %s. The closest runway is about %s.", gateway.FormatNumber(validatedCount), network, strings.Join(validatedNames, ", "), humanizeLedgerRunway(worstRemaining))
		}
		data.Alert = vmv2.AlertData{
			Title: title,
			Body:  body,
			Meta:  "Why this matters: persistent storage should be extended before it expires; temporary storage is excluded from this warning because it often expires by design.",
			CTA:   "Review →",
		}
	}
	data.Attention = validated
}

func nearestPersistentStorageRunway(entries []gateway.ContractStorageEntry, currentLedger int64) int64 {
	var nearest int64
	for _, e := range entries {
		text := strings.ToLower(e.Type + " " + e.Durability)
		if !strings.Contains(text, "persistent") {
			continue
		}
		remaining := e.TTLRemaining
		if e.LiveUntilLedgerSeq > 0 && currentLedger > 0 {
			remaining = e.LiveUntilLedgerSeq - currentLedger
		}
		if remaining <= 0 {
			continue
		}
		if nearest == 0 || remaining < nearest {
			nearest = remaining
		}
	}
	return nearest
}

func humanizeLedgerRunway(ledgers int64) string {
	hours := int64(math.Ceil(float64(ledgers) * 5 / 3600))
	return humanizeHours(hours)
}

func homeV2RunwayPctFromLedgers(ledgers int64) float64 {
	// Home summary runway_pct is a percentage, where 100,000 ledgers of runway is 100%.
	// Keep the same semantics for locally validated TTL rows before formatting as %.0f%%.
	return float64(percentOf(ledgers, 100_000))
}

func buildHomeV2Leaders(summary *gateway.HomeSummaryResponse) vmv2.LeadersSectionData {
	data := vmv2.LeadersSectionData{
		Title: "Who’s being used the most today",
		Copy:  "Ranked by how many times each contract was called in the last 24 hours, not dollars, not TVL. The actual activity.",
	}
	labels := []string{"Most active", "Runner-up", "Third place", "Fastest growing"}
	tones := []string{"is-swap", "is-call", "is-deploy", "is-agent"}
	marks := []string{"S", "B", "P", "x"}
	for i, leader := range summary.Leaders {
		if i == 4 {
			break
		}
		entity := homeV2FirstNonEmpty(leader.ProtocolName, leader.ContractName, gateway.ShortAddress(leader.ContractID))
		body := fmt.Sprintf("%s unique callers in the last 24 hours. Dominant actions: %s.", gateway.FormatNumber(leader.UniqueCallers24h), strings.Join(leader.DominantActions, ", "))
		if len(leader.DominantActions) == 0 {
			body = fmt.Sprintf("%s unique callers in the last 24 hours.", gateway.FormatNumber(leader.UniqueCallers24h))
		}
		data.Cards = append(data.Cards, vmv2.LeaderCardData{
			Label:      labels[boundedIndex(i, len(labels))],
			Value:      gateway.FormatNumber(leader.CallCount24h),
			Entity:     entity,
			EntityMark: marks[boundedIndex(i, len(marks))],
			EntityTone: tones[boundedIndex(i, len(tones))],
			Body:       body,
			Href:       homeV2ContractHref(leader.ContractID),
		})
	}
	return data
}

func homeV2ContractHref(contractID string) string {
	contractID = strings.TrimSpace(contractID)
	if contractID == "" {
		return ""
	}
	return "/v2/contract/" + contractID
}

func buildHomeV2Utilization(summary *gateway.HomeSummaryResponse) vmv2.UtilizationSectionData {
	u := summary.Utilization
	cards := []vmv2.UtilizationCardData{
		{
			Label:     "Instruction budget",
			ValueMain: formatPercentMain(u.InstructionPct),
			ValueUnit: "% used",
			Tone:      utilTone(u.InstructionPct),
			Body:      fmt.Sprintf("%s of %s instructions at ledger %s.", formatBigUnit(u.InstructionUsed), formatBigUnit(u.InstructionLimit), gateway.FormatNumber(u.SourceLedger)),
			BarColor:  utilBarColor(u.InstructionPct),
			BarWidth:  fmt.Sprintf("%.0f%%", clampFloat(u.InstructionPct, 0, 100)),
		},
		{
			Label:     "Read / write",
			ValueMain: formatPercentMain(u.ReadWritePct),
			ValueUnit: "% used",
			Tone:      utilTone(u.ReadWritePct),
			Body:      fmt.Sprintf("%s of %s moved in and out of ledger storage.", formatBytesHuman(u.ReadWriteUsedBytes), formatBytesHuman(u.ReadWriteLimitBytes)),
			BarColor:  utilBarColor(u.ReadWritePct),
			BarWidth:  fmt.Sprintf("%.0f%%", clampFloat(u.ReadWritePct, 0, 100)),
		},
		{
			Label:     "Transaction size",
			ValueMain: formatPercentMain(u.TxSizePct),
			ValueUnit: "% used",
			Tone:      utilTone(u.TxSizePct),
			Body:      txSizeBody(u),
			BarColor:  utilBarColor(u.TxSizePct),
			BarWidth:  fmt.Sprintf("%.0f%%", clampFloat(u.TxSizePct, 0, 100)),
		},
	}
	return vmv2.UtilizationSectionData{Title: "How much of the network is being used", Copy: "Every ledger has limits. When they fill up, fees rise sharply. Here’s where we sit right now.", Cards: cards}
}

func applyHomeV2Meta(data *vmv2.HomeData, summary *gateway.HomeSummaryResponse) {
	items := make([]string, 0, 4)
	if summary.Meta.HistoryStartProtocol > 0 {
		items = append(items, fmt.Sprintf("Complete history from <b>protocol v%d</b> · no gaps", summary.Meta.HistoryStartProtocol))
	}
	if summary.Meta.LookupAvgMS > 0 || summary.Meta.LookupP99MS > 0 {
		items = append(items, fmt.Sprintf("Lookups averaging <b>%d ms</b> · 99th percentile <b>%d ms</b>", summary.Meta.LookupAvgMS, summary.Meta.LookupP99MS))
	}
	if summary.Meta.KnownProtocolCount > 0 {
		items = append(items, fmt.Sprintf("<b>%s known protocols</b> · named automatically", gateway.FormatNumber(summary.Meta.KnownProtocolCount)))
	}
	items = append(items, "Open source, <a href=\"#\">github.com/obsrvr/prism</a>")
	data.FooterItems = items
}

func buildHomeV2FeedLedger(network string, recent gateway.RecentLedger, next *gateway.RecentLedger, summary *gateway.LedgerFeedSummary) (vmv2.LedgerRowData, homeV2FeedLedger, []homeV2FeedTransaction) {
	seq := gateway.FormatNumber(recent.LedgerSequence)
	age := "just now"
	if t, ok := parseGatewayTime(recent.ClosedAt); ok {
		age = gateway.FormatAge(t)
	}
	closeTime := "5.0s"
	if next != nil {
		if curr, ok := parseGatewayTime(recent.ClosedAt); ok {
			if prev, ok := parseGatewayTime(next.ClosedAt); ok {
				delta := curr.Sub(prev).Seconds()
				if delta > 0 {
					closeTime = gateway.FormatCloseTime(delta)
				}
			}
		}
	}

	txCount, includedOperations, successfulOperations, failedOperations := homeV2RecentLedgerCounts(recent)
	meta := fmt.Sprintf("with %s operations", gateway.FormatNumber(includedOperations))
	if failedOperations > 0 {
		meta += fmt.Sprintf(" (%s failed)", gateway.FormatNumber(failedOperations))
	}
	chips, feedChips := homeV2RecentOperationChips(recent)
	instructionsPct := 0
	readWritePct := 0
	samples := []homeV2FeedTransaction{}
	allTxs := []homeV2FeedTransaction{}
	validator := recent.Validator

	if summary != nil {
		if summary.Totals.TransactionCount > 0 {
			txCount = summary.Totals.TransactionCount
		}
		if includedOperations == 0 {
			includedOperations = summary.Totals.TransactionSetOperationCount
			if includedOperations == 0 {
				includedOperations = summary.Totals.OperationCount
			}
			successfulOperations = summary.Totals.SuccessfulOperationCount
			if successfulOperations == 0 {
				successfulOperations = summary.Totals.OperationCount
			}
			failedOperations = summary.Totals.FailedOperationCount
			meta = fmt.Sprintf("with %s operations", gateway.FormatNumber(includedOperations))
			if failedOperations > 0 {
				meta += fmt.Sprintf(" (%s failed)", gateway.FormatNumber(failedOperations))
			}
		}
		if summary.Ledger.Validator != nil && validator.PublicKey == "" {
			validator = *summary.Ledger.Validator
		}
		if len(chips) == 0 {
			chips, feedChips = homeV2FeedChips(summary)
		}
		instructionsPct = clampPct(percentOf(summary.SorobanUtilization.InstructionsUsed, homeV2SorobanInstructionLimit))
		readWritePct = clampPct(percentOf(summary.SorobanUtilization.ReadWriteBytesUsed, homeV2SorobanReadWriteLimit))
		for _, tx := range summary.RepresentativeTransactions {
			mapped := mapHomeV2RepresentativeTx(tx, age)
			samples = append(samples, mapped)
			allTxs = append(allTxs, mapped)
		}
	}
	introducer := homeV2IntroducerLabel(validator, summary)

	if len(chips) == 0 {
		chips = []componentsv2.LedgerMetricChip{{Label: pluralizeChip(int64(recent.SuccessfulTxCount), "successful tx", "successful txs"), Kind: "classic"}}
		feedChips = []homeV2FeedLedgerChip{{Label: pluralizeChip(int64(recent.SuccessfulTxCount), "successful tx", "successful txs"), Kind: "classic"}}
	}

	row := vmv2.LedgerRowData{
		LedgerNumber:             seq,
		TransactionCount:         gateway.FormatNumber(txCount),
		IncludedOperationCount:   gateway.FormatNumber(includedOperations),
		SuccessfulOperationCount: gateway.FormatNumber(successfulOperations),
		FailedOperationCount:     gateway.FormatNumber(failedOperations),
		Meta:                     meta,
		Introducer:               introducer,
		Chips:                    chips,
		InstructionsPct:          instructionsPct,
		ReadWritePct:             readWritePct,
		CloseTime:                closeTime,
		Age:                      age,
		SideMeta:                 homeV2SideMeta(network),
	}
	feedLedger := homeV2FeedLedger{
		LedgerNumber:             seq,
		TransactionCount:         gateway.FormatNumber(txCount),
		IncludedOperationCount:   gateway.FormatNumber(includedOperations),
		SuccessfulOperationCount: gateway.FormatNumber(successfulOperations),
		FailedOperationCount:     gateway.FormatNumber(failedOperations),
		Meta:                     meta,
		Introducer:               introducer,
		Chips:                    feedChips,
		InstructionsPct:          instructionsPct,
		ReadWritePct:             readWritePct,
		CloseTime:                closeTime,
		Age:                      age,
		SideMeta:                 homeV2SideMeta(network),
		Samples:                  samples,
	}
	return row, feedLedger, allTxs
}

func homeV2RecentLedgerCounts(recent gateway.RecentLedger) (transactions, includedOperations, successfulOperations, failedOperations int64) {
	transactions = int64(recent.TransactionCount)
	if transactions == 0 {
		transactions = int64(recent.Transactions.Total)
	}
	if transactions == 0 {
		transactions = int64(recent.SuccessfulTxCount + recent.FailedTxCount)
	}

	includedOperations = int64(recent.Operations.Included)
	if includedOperations == 0 {
		includedOperations = int64(recent.TransactionSetOperationCount)
	}
	successfulOperations = int64(recent.Operations.Successful)
	if successfulOperations == 0 {
		successfulOperations = int64(recent.SuccessfulOperationCount)
	}
	if successfulOperations == 0 {
		successfulOperations = int64(recent.OperationCount)
	}
	if includedOperations == 0 {
		includedOperations = successfulOperations
	}

	failedOperations = int64(recent.Operations.Failed)
	if failedOperations == 0 {
		failedOperations = int64(recent.FailedOperationCount)
	}
	if failedOperations == 0 && includedOperations >= successfulOperations {
		failedOperations = includedOperations - successfulOperations
	}
	return transactions, includedOperations, successfulOperations, failedOperations
}

func homeV2RecentOperationChips(recent gateway.RecentLedger) ([]componentsv2.LedgerMetricChip, []homeV2FeedLedgerChip) {
	if recent.Operations.ClassificationStatus != "materialized" {
		return nil, nil
	}
	items := []struct {
		count    int64
		singular string
		plural   string
		kind     string
	}{
		{int64(recent.Operations.Categories.AccountCreation), "account creation", "account creations", "account"},
		{int64(recent.Operations.Categories.Payments), "payment", "payments", "payment"},
		{int64(recent.Operations.Categories.OffersAndAMMs), "offer / AMM", "offers / AMMs", "market"},
		{int64(recent.Operations.Categories.Trustlines), "trustline", "trustlines", "trustline"},
		{int64(recent.Operations.Categories.ClaimableBalances), "claimable balance", "claimable balances", "claimable"},
		{int64(recent.Operations.Categories.Sponsorship), "sponsorship", "sponsorships", "sponsorship"},
		{int64(recent.Operations.Categories.Soroban), "Soroban op", "Soroban ops", "soroban"},
		{int64(recent.Operations.Categories.Other), "other op", "other ops", "other"},
	}
	chips := make([]componentsv2.LedgerMetricChip, 0, len(items))
	feedChips := make([]homeV2FeedLedgerChip, 0, len(items))
	for _, item := range items {
		if item.count <= 0 {
			continue
		}
		label := pluralizeChip(item.count, item.singular, item.plural)
		chips = append(chips, componentsv2.LedgerMetricChip{Label: label, Kind: item.kind})
		feedChips = append(feedChips, homeV2FeedLedgerChip{Label: label, Kind: item.kind})
	}
	return chips, feedChips
}

func homeV2IntroducerLabel(validator gateway.LedgerValidator, summary *gateway.LedgerFeedSummary) string {
	for _, candidate := range []string{validator.DisplayName, validator.Name, validator.Alias, validator.HomeDomain} {
		if label := strings.TrimSpace(candidate); label != "" {
			return label
		}
	}
	if publicKey := strings.TrimSpace(validator.PublicKey); publicKey != "" {
		return gateway.ShortAddress(publicKey)
	}
	if summary != nil {
		if publicKey := strings.TrimSpace(summary.Ledger.ClosedByValidator); publicKey != "" {
			return gateway.ShortAddress(publicKey)
		}
	}
	return ""
}

func homeV2FeedChips(summary *gateway.LedgerFeedSummary) ([]componentsv2.LedgerMetricChip, []homeV2FeedLedgerChip) {
	if summary == nil {
		return nil, nil
	}
	add := func(items *[]componentsv2.LedgerMetricChip, feedItems *[]homeV2FeedLedgerChip, count int64, singular, plural, kind string) {
		if count <= 0 {
			return
		}
		label := pluralizeChip(count, singular, plural)
		*items = append(*items, componentsv2.LedgerMetricChip{Label: label, Kind: kind})
		*feedItems = append(*feedItems, homeV2FeedLedgerChip{Label: label, Kind: kind})
	}
	chips := []componentsv2.LedgerMetricChip{}
	feedChips := []homeV2FeedLedgerChip{}
	add(&chips, &feedChips, summary.ClassificationCounts.SwapTxCount, "swap", "swaps", "swap")
	add(&chips, &feedChips, summary.ClassificationCounts.ContractCallTxCount, "call", "calls", "call")
	add(&chips, &feedChips, summary.ClassificationCounts.ClassicTxCount, "classic tx", "classic txs", "classic")
	return chips, feedChips
}

func mapHomeV2RepresentativeTx(tx gateway.LedgerFeedRepresentativeTransaction, when string) homeV2FeedTransaction {
	kind := homeV2TxKind(tx)
	headline := strings.TrimSpace(tx.Summary.Description)
	if headline == "" {
		headline = strings.TrimSpace(tx.CategoryLabel)
	}
	contextParts := []string{}
	if tx.Classification.Subtype != "" {
		contextParts = append(contextParts, tx.Classification.Subtype)
	}
	if tx.Summary.FunctionName != "" {
		contextParts = append(contextParts, tx.Summary.FunctionName+"()")
	}
	if tx.Summary.ProtocolLabel != "" {
		contextParts = append(contextParts, tx.Summary.ProtocolLabel)
	}
	if tx.Classification.Confidence != "" {
		contextParts = append(contextParts, tx.Classification.Confidence+" confidence")
	}
	entity := tx.CategoryLabel
	if tx.Actors != nil && tx.Actors.Primary != nil && tx.Actors.Primary.Label != "" {
		entity = tx.Actors.Primary.Label
	}
	return homeV2FeedTransaction{
		Kind:     kind,
		Headline: html.EscapeString(headline),
		Context:  strings.Join(contextParts, " · "),
		Entity:   entity,
		Hash:     gateway.ShortHash(tx.TxHash),
		When:     when,
	}
}

func homeV2TxKind(tx gateway.LedgerFeedRepresentativeTransaction) string {
	switch tx.Category {
	case "swap":
		return "swap"
	case "contract_call":
		return "call"
	case "classic_payment":
		return "classic"
	default:
		switch tx.Classification.TxType {
		case "swap":
			return "swap"
		case "contract_call":
			return "call"
		default:
			return "classic"
		}
	}
}

func homeV2SideMeta(network string) string {
	switch network {
	case "testnet":
		return "testnet cadence"
	case "futurenet":
		return "futurenet cadence"
	default:
		return "5 ledgers / min"
	}
}

func pluralizeChip(count int64, singular, plural string) string {
	if count == 1 {
		return fmt.Sprintf("%s %s", gateway.FormatNumber(count), singular)
	}
	return fmt.Sprintf("%s %s", gateway.FormatNumber(count), plural)
}

func percentOf(used, limit int64) int {
	if used <= 0 || limit <= 0 {
		return 0
	}
	return int(math.Round((float64(used) / float64(limit)) * 100))
}

func clampPct(v int) int {
	if v < 0 {
		return 0
	}
	if v > 100 {
		return 100
	}
	return v
}

func clampFloat(v, low, high float64) float64 {
	if v < low {
		return low
	}
	if v > high {
		return high
	}
	return v
}

func homeV2FirstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func boundedIndex(i, n int) int {
	if n <= 0 {
		return 0
	}
	if i < 0 {
		return 0
	}
	if i >= n {
		return n - 1
	}
	return i
}

func humanizeHours(hours int64) string {
	switch {
	case hours <= 1:
		return "1 hour"
	case hours < 24:
		return fmt.Sprintf("%d hours", hours)
	case hours%24 == 0:
		days := hours / 24
		if days == 1 {
			return "1 day"
		}
		return fmt.Sprintf("%d days", days)
	default:
		days := hours / 24
		rem := hours % 24
		return fmt.Sprintf("%d days %d hours", days, rem)
	}
}

func attentionTone(severity string) string {
	switch severity {
	case "bad":
		return "is-bad"
	case "warn":
		return "is-warn"
	default:
		return "is-green"
	}
}

func attentionBarColor(severity string) string {
	switch severity {
	case "bad":
		return "#ea8c69"
	case "warn":
		return "#d6aa45"
	default:
		return "#64a86a"
	}
}

func utilTone(pct float64) string {
	if pct >= 60 {
		return "is-warn"
	}
	return ""
}

func utilBarColor(pct float64) string {
	if pct >= 60 {
		return "#d6aa45"
	}
	return "#64a86a"
}

func formatPercentMain(v float64) string {
	return fmt.Sprintf("%.0f", v)
}

func formatBigUnit(v int64) string {
	switch {
	case v >= 1_000_000_000:
		return fmt.Sprintf("%.1fB", float64(v)/1_000_000_000)
	case v >= 1_000_000:
		return fmt.Sprintf("%.1fM", float64(v)/1_000_000)
	case v >= 1_000:
		return fmt.Sprintf("%.1fK", float64(v)/1_000)
	default:
		return gateway.FormatNumber(v)
	}
}

func formatBytesHuman(v int64) string {
	switch {
	case v >= 1_000_000:
		return fmt.Sprintf("%.1f MB", float64(v)/1_000_000)
	case v >= 1_000:
		return fmt.Sprintf("%.1f KB", float64(v)/1_000)
	default:
		return fmt.Sprintf("%d B", v)
	}
}

func txSizeBody(u gateway.HomeSummaryUtilization) string {
	if u.AvgTxSizeBytes > 0 && u.TxSizeLimitBytes > 0 {
		return fmt.Sprintf("Average transaction is %s of %s allowed.", formatBytesHuman(u.AvgTxSizeBytes), formatBytesHuman(u.TxSizeLimitBytes))
	}
	if u.TxSizeLimitBytes > 0 {
		return fmt.Sprintf("Transaction size limit is %s on this network.", formatBytesHuman(u.TxSizeLimitBytes))
	}
	return "Transaction size usage is currently unavailable."
}

type homeV2FeedPayload struct {
	Live   bool                    `json:"live"`
	Header homeV2FeedPayloadHeader `json:"header"`
	Feed   homeV2Feed              `json:"feed"`
}

type homeV2FeedPayloadHeader struct {
	LedgerNumber string `json:"ledgerNumber"`
	AgeLabel     string `json:"ageLabel"`
}

func (h *Handlers) HomeV2Feed(w http.ResponseWriter, r *http.Request) {
	network := networkFromRequest(r)
	payload := homeV2FeedPayload{Live: false}
	if h.useLiveData(r) {
		if header, _, feed, err := h.buildHomeV2FeedData(r, network); err == nil {
			payload.Live = true
			payload.Header = header
			payload.Feed = feed
		}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(payload)
}

func parseGatewayTime(value string) (time.Time, bool) {
	if value == "" {
		return time.Time{}, false
	}
	t, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}

type homeV2Feed struct {
	Ledgers      []homeV2FeedLedger      `json:"ledgers"`
	Transactions []homeV2FeedTransaction `json:"transactions"`
}

type homeV2FeedLedger struct {
	LedgerNumber             string                  `json:"ledgerNumber"`
	TransactionCount         string                  `json:"transactionCount"`
	IncludedOperationCount   string                  `json:"includedOperationCount"`
	SuccessfulOperationCount string                  `json:"successfulOperationCount"`
	FailedOperationCount     string                  `json:"failedOperationCount"`
	Meta                     string                  `json:"meta"`
	Introducer               string                  `json:"introducer,omitempty"`
	Chips                    []homeV2FeedLedgerChip  `json:"chips"`
	InstructionsPct          int                     `json:"instructionsPct"`
	ReadWritePct             int                     `json:"readWritePct"`
	CloseTime                string                  `json:"closeTime"`
	Age                      string                  `json:"age"`
	SideMeta                 string                  `json:"sideMeta"`
	Samples                  []homeV2FeedTransaction `json:"samples"`
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
			{LedgerNumber: base, TransactionCount: "212", IncludedOperationCount: "424", SuccessfulOperationCount: "420", FailedOperationCount: "4", Meta: titlePrefix + "with 424 operations", Introducer: "sdf-validator-1", Chips: []homeV2FeedLedgerChip{{Label: "50 swaps", Kind: "swap"}, {Label: "67 calls", Kind: "call"}, {Label: "54 agent payments", Kind: "agent"}, {Label: "33 classic", Kind: "classic"}, {Label: "2 deploys", Kind: "deploy"}, {Label: "2 confidential", Kind: "confidential"}}, InstructionsPct: 61, ReadWritePct: 35, CloseTime: "4.9s", Age: "just now", SideMeta: sideMeta, Samples: []homeV2FeedTransaction{txs[1], txs[0], txs[2]}},
			{LedgerNumber: decLedger(base, 1), TransactionCount: "226", IncludedOperationCount: "452", SuccessfulOperationCount: "450", FailedOperationCount: "2", Meta: titlePrefix + "with 452 operations", Introducer: "whale-stack", Chips: []homeV2FeedLedgerChip{{Label: "42 swaps", Kind: "swap"}, {Label: "75 calls", Kind: "call"}, {Label: "37 agent payments", Kind: "agent"}, {Label: "67 classic", Kind: "classic"}, {Label: "3 deploys", Kind: "deploy"}, {Label: "1 confidential", Kind: "confidential"}}, InstructionsPct: 76, ReadWritePct: 45, CloseTime: "4.9s", Age: "5 seconds ago", SideMeta: sideMeta, Samples: []homeV2FeedTransaction{txs[6], txs[7], txs[5]}},
			{LedgerNumber: decLedger(base, 2), TransactionCount: "187", IncludedOperationCount: "561", SuccessfulOperationCount: "558", FailedOperationCount: "3", Meta: titlePrefix + "with 561 operations", Introducer: "publicnode", Chips: []homeV2FeedLedgerChip{{Label: "35 swaps", Kind: "swap"}, {Label: "70 calls", Kind: "call"}, {Label: "30 agent payments", Kind: "agent"}, {Label: "52 classic", Kind: "classic"}, {Label: "1 deploy", Kind: "deploy"}}, InstructionsPct: 67, ReadWritePct: 36, CloseTime: "4.9s", Age: "10 seconds ago", SideMeta: sideMeta, Samples: []homeV2FeedTransaction{txs[3], txs[4], txs[5]}},
			{LedgerNumber: decLedger(base, 3), TransactionCount: "199", IncludedOperationCount: "388", SuccessfulOperationCount: "385", FailedOperationCount: "3", Meta: titlePrefix + "with 388 operations", Introducer: "lobstr", Chips: []homeV2FeedLedgerChip{{Label: "50 swaps", Kind: "swap"}, {Label: "62 calls", Kind: "call"}, {Label: "55 agent payments", Kind: "agent"}, {Label: "32 classic", Kind: "classic"}, {Label: "1 deploy", Kind: "deploy"}}, InstructionsPct: 50, ReadWritePct: 50, CloseTime: "4.9s", Age: "15 seconds ago", SideMeta: sideMeta, Samples: []homeV2FeedTransaction{txs[2], txs[0], txs[5]}},
			{LedgerNumber: decLedger(base, 4), TransactionCount: "197", IncludedOperationCount: "788", SuccessfulOperationCount: "783", FailedOperationCount: "5", Meta: titlePrefix + "with 788 operations", Introducer: "coinbase-cloud", Chips: []homeV2FeedLedgerChip{{Label: "53 swaps", Kind: "swap"}, {Label: "66 calls", Kind: "call"}, {Label: "33 agent payments", Kind: "agent"}, {Label: "41 classic", Kind: "classic"}, {Label: "2 deploys", Kind: "deploy"}, {Label: "2 confidential", Kind: "confidential"}}, InstructionsPct: 50, ReadWritePct: 38, CloseTime: "4.9s", Age: "20 seconds ago", SideMeta: sideMeta, Samples: []homeV2FeedTransaction{txs[4], txs[6], txs[1]}},
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
