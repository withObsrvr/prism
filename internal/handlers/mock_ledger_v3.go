package handlers

import (
	"fmt"
	"strconv"
	"strings"

	vmv2 "github.com/withObsrvr/prism/internal/templates/v2/viewmodel"
)

// Mock data for the Ledger Detail v3 prototype.
//
// Numbers reproduce the design exactly so the layout can be judged at real
// density. Every block carries a Provenance naming the obsrvr-lake source that
// would feed it. The nebu origins soroban-tx-resources and ledger-change-stats
// closed most of the original gaps; what remains is listed in ledgerV3Gaps.

// Provenance constants, one per distinct source. Declared together so the full
// data dependency of the page is readable in one place.
var (
	provLedgerStats = vmv2.Provenance{
		Kind:   vmv2.ProvenanceServed,
		Origin: "serving.sv_ledger_stats_recent",
	}
	provTxRecent = vmv2.Provenance{
		Kind:   vmv2.ProvenanceServed,
		Origin: "serving.sv_transactions_recent",
	}
	provApplyOrder = vmv2.Provenance{
		Kind:   vmv2.ProvenanceDerived,
		Origin: "serving.sv_transactions_recent.transaction_id",
		Note:   "TOID encodes ledger sequence and application order; sort ascending to recover apply order.",
	}
	provFeesEndpoint = vmv2.Provenance{
		Kind:   vmv2.ProvenanceServed,
		Origin: "GET /api/v1/silver/ledgers/{seq}/fees",
	}
	provSorobanEndpoint = vmv2.Provenance{
		Kind:   vmv2.ProvenanceServed,
		Origin: "GET /api/v1/silver/ledgers/{seq}/soroban",
	}
	provContractLabels = vmv2.Provenance{
		Kind:   vmv2.ProvenanceServed,
		Origin: "serving.sv_contract_labels",
		Note:   "Joined on sv_transactions_recent.primary_contract_id.",
	}
	provCPUUtilization = vmv2.Provenance{
		Kind:   vmv2.ProvenanceDerived,
		Origin: "GET /api/v1/silver/ledgers/{seq}/soroban .total_cpu_insns ÷ silver.config_settings_current.ledger_max_instructions",
		Note:   "Numerator is served; the effective-at-ledger cap comes from the same lookup getServingLedgerLimits already performs.",
	}
	// --- closed by the nebu origins added for this page ---

	provFootprintEntries = vmv2.Provenance{
		Kind:   vmv2.ProvenanceDerived,
		Origin: "SUM(soroban-tx-resources.readWriteEntries | .readOnlyEntries) ÷ silver.config_settings_current.ledger_max_{write,read}_ledger_entries",
		Note:   "Declared footprint entries, which is what the protocol charges capacity against. Counting observed changes instead undercounts by 20-70%.",
	}
	provTxWriteEntries = vmv2.Provenance{
		Kind:   vmv2.ProvenanceServed,
		Origin: "soroban-tx-resources.readWriteEntries",
		Note:   "Per transaction, so a single contract's share of the ledger write budget is attributable.",
	}
	provResultCodes = vmv2.Provenance{
		Kind:   vmv2.ProvenanceServed,
		Origin: "soroban-tx-resources.resultCode, .operationResultCodes, .contractErrorType, .contractErrorCode",
		Note:   "Decoded per transaction. contractErrorCode carries the contract's own number — the #4 in #4 HealthFactorTooLow.",
	}
	provEntryChanges = vmv2.Provenance{
		Kind:   vmv2.ProvenanceServed,
		Origin: "ledger-change-stats.ledgerEntries{Created,Updated,Deleted,Restored} + .entryTypes[]",
		Note:   "Walks the full change stream, so classic entries are included alongside contract data.",
	}
	provArchivalKeys = vmv2.Provenance{
		Kind:   vmv2.ProvenanceServed,
		Origin: "ledger-change-stats.evictedKeys, .evictedByType[], .ledgerEntriesRestored",
		Note:   "Check evictionAvailable before reading evictedKeys: on meta versions predating eviction a zero would otherwise read as 'nothing evicted'.",
	}

	// --- gaps ---

	provTxSizeBytes = vmv2.Provenance{
		Kind:   vmv2.ProvenanceGap,
		Origin: "numerator only",
		Note:   "soroban-tx-resources.envelopeSizeBytes supplies the numerator, but the only published cap (txMaxSizeBytes) is Soroban-only and this meter sums every envelope. Needs a scoping decision, not more data.",
	}
	provContractAttribution = vmv2.Provenance{
		Kind:   vmv2.ProvenanceGap,
		Origin: "none",
		Note:   "ledger-change-stats aggregates evictions and restorations to {entry_type, count}; the contract identity is dropped. EvictedLedgerKeys() carries it, so closing this means emitting contract IDs alongside the counts.",
	}
	provArchivalCost = vmv2.Provenance{
		Kind:   vmv2.ProvenanceGap,
		Origin: "none",
		Note:   "Per-entry rent attribution. soroban-tx-resources.resourceFeeStroops is charged per transaction, not per restored or archived entry.",
	}
	provExcluded = vmv2.Provenance{
		Kind:   vmv2.ProvenanceNone,
		Origin: "not observable",
		Note:   "A closed ledger records inclusions only. Transactions that bid below the clearing price are written nowhere.",
	}
)

// ledgerV3TickSpec encodes the 118 transactions in apply order, one token per
// transaction: a kind letter followed by the operation count. An uppercase
// letter marks a failed transaction. Totals 359 operations and 7 failures,
// matching the design.
var ledgerV3TickSpec = []string{
	"c4 m2 c2 p1 c3 p2 p2 c5 o1 C3 c5 c3 m3 p1 c5 m4 p2 c4 c7 c4 c2 p1 p2 C7",
	"p2 c3 p1 c2 p1 m4 o2 p1 c7 c2 c5 p1 p2 c2 c4 c5 c6 C1 c6 p2 c2 c3 c5 c5",
	"d2 c6 o1 c4 p2 c3 c3 m4 c5 c4 C2 c7 m4 c3 m2 m1 d1 c6 C2 c4 c6 d2 p1 c5",
	"c7 o2 c3 c2 c2 p1 p1 c3 c5 p2 m4 p1 p1 p1 c5 m1 p1 C2 c4 c2 c2 c2 m1 p2",
	"p1 c7 c3 c3 c3 m4 m4 c2 O1 c5 p1 c4 p2 m4 p1 c7 o2 c7 m4 c5 c2 p1",
}

// ledgerV3FailureDetail describes each failed position, keyed by apply order.
var ledgerV3FailureDetail = map[int]struct{ Protocol, Code string }{
	10:  {"Blend", "#4 HealthFactorTooLow"},
	24:  {"Blend", "#4 HealthFactorTooLow"},
	42:  {"Soroswap", "#8 InsufficientOutputAmount"},
	59:  {"Blend", "#4 HealthFactorTooLow"},
	67:  {"Blend", "#4 HealthFactorTooLow"},
	90:  {"Soroswap", "#8 InsufficientOutputAmount"},
	105: {"classic", "tx_BAD_SEQ"},
}

var ledgerV3KindNames = map[byte]string{
	'c': "calls",
	'p': "payments",
	'm': "markets",
	'd': "deployments",
	'o': "other",
}

var ledgerV3KindLabels = map[string]string{
	"calls":       "Contract call",
	"payments":    "Payment",
	"markets":     "Market order",
	"deployments": "Deployment",
	"other":       "Other",
}

// ledgerV3TickHeightByOps are the bar heights the design uses, indexed by
// operation count. Kept as a table rather than a formula because the design's
// steps are not perfectly linear.
var ledgerV3TickHeightByOps = [8]int{0, 43, 52, 61, 69, 78, 87, 96}

// ledgerV3TickHeight maps an operation count to the bar height the design uses.
func ledgerV3TickHeight(ops int) int {
	if ops < 1 {
		ops = 1
	}
	if ops >= len(ledgerV3TickHeightByOps) {
		ops = len(ledgerV3TickHeightByOps) - 1
	}
	return ledgerV3TickHeightByOps[ops]
}

func buildLedgerV3Ticks() []vmv2.LedgerV3Tick {
	tokens := strings.Fields(strings.Join(ledgerV3TickSpec, " "))
	ticks := make([]vmv2.LedgerV3Tick, 0, len(tokens))
	for i, tok := range tokens {
		position := i + 1
		letter := tok[0]
		failed := letter >= 'A' && letter <= 'Z'
		if failed {
			letter += 'a' - 'A'
		}
		ops, _ := strconv.Atoi(tok[1:])
		kind := ledgerV3KindNames[letter]

		tick := vmv2.LedgerV3Tick{
			Position:  position,
			Kind:      kind,
			HeightPct: ledgerV3TickHeight(ops),
			Failed:    failed,
			OpCount:   ops,
			TipTitle:  fmt.Sprintf("Position %d", position),
			TipDetail: fmt.Sprintf("%s · %s", ledgerV3KindLabels[kind], pluralOps(ops)),
			TipStatus: "applied",
			Href:      "#",
			AriaLabel: fmt.Sprintf("Transaction at position %d", position),
		}
		if failed {
			detail := ledgerV3FailureDetail[position]
			tick.TipTitle = fmt.Sprintf("Position %d · failed", position)
			tick.TipDetail = fmt.Sprintf("%s · %s", detail.Protocol, detail.Code)
			tick.TipStatus = fmt.Sprintf("%s · reverted", pluralOps(ops))
			tick.ErrorCode = detail.Code
			tick.ProtocolID = detail.Protocol
		}
		ticks = append(ticks, tick)
	}
	return ticks
}

func pluralOps(n int) string {
	if n == 1 {
		return "1 operation"
	}
	return fmt.Sprintf("%d operations", n)
}

// mockLedgerDetailV3Data returns the full prototype page.
func mockLedgerDetailV3Data(sequence, network string) vmv2.LedgerDetailV3Data {
	seq := strings.TrimSpace(sequence)
	if seq == "" {
		seq = "55812406"
	}
	seqNum, err := strconv.ParseInt(seq, 10, 64)
	if err != nil {
		seqNum = 55812406
		seq = "55812406"
	}

	data := vmv2.LedgerDetailV3Data{Network: network}

	data.Header = vmv2.LedgerV3Header{
		Sequence:         formatThousands(seqNum),
		SequenceRaw:      seq,
		Hash:             "9f4c2b7e18a05d63c94f1e27ab8d3056f2e7c419b6d80a3f5e2c17948bd6f302",
		ClosedAt:         "04 Aug 2026, 09:41:17",
		ClosedRelative:   "4 min ago",
		CloseTime:        "5.2s",
		ProtocolVersion:  "22",
		PrevSequence:     formatThousands(seqNum - 1),
		PrevSequenceRaw:  strconv.FormatInt(seqNum-1, 10),
		Kicker:           "118 transactions · closed in 5.2s",
		HeadlineLead:     "This ledger ran out of",
		HeadlineEmphasis: "writes",
		HeadlineTrail:    "before anything else",
		TxTabCount:       "118",
		StateTabCount:    "3,220",
		HeadlineSource:   provFootprintEntries,
		Badges: []vmv2.LedgerV3Badge{
			{Label: "Writes 97% full", Tone: "warn"},
			{Label: "Fees 42× base", Tone: "warn"},
			{Label: "7 failed"},
		},
	}

	data.Lede = []string{
		`<span class="pc-drop">A</span> ledger is not a container of equal parts. This one held <b>118 transactions</b>, and by the time it closed it had used <b>97% of its write capacity</b> while sitting at 71% of CPU and 45% of reads.<a class="pc-cite" href="#n1">1</a> Writes were the binding constraint, so the fee market cleared at <b>42× the base fee</b> — every transaction here paid a premium set by contention for a resource most of them barely used.<a class="pc-cite" href="#n2">2</a>`,
		`Soroban accounted for <b>52%</b> of the transactions and nearly all of the writes.<a class="pc-cite" href="#n3">3</a> Seven transactions failed, and <span class="term">six of the seven share two error codes</span> — which points at two protocols having a bad five seconds rather than at the network.<a class="pc-cite" href="#n4">4</a>`,
	}

	data.Standing = []vmv2.LedgerV3Standing{
		{Key: "Binding limit", Value: "Writes, 97%", Detail: "124 of 128 entries", Dot: "a", Source: provFootprintEntries},
		{Key: "Fee level", Value: "42× base", Detail: "Surge pricing was active", Dot: "a", Source: provFeesEndpoint},
		{Key: "Close time", Value: "5.2s", Detail: "Target 5.0s · no consensus delay", Dot: "g", Source: provLedgerStats},
		{Key: "Failures", Value: "7 of 118", Detail: "5.9% against a 1.1% median", Dot: "a", Source: provLedgerStats},
		{Key: "Excluded", Value: "Not observable", Detail: "Ledgers record inclusions only", Dot: "none", Source: provExcluded},
	}

	data.Strip = vmv2.LedgerV3Strip{
		Ticks: buildLedgerV3Ticks(),
		Foot: []vmv2.LedgerV3Stat{
			{Label: "Transactions", Value: "118", Source: provLedgerStats},
			{Label: "Operations", Value: "359", Source: provLedgerStats},
			{Label: "Failed", Value: "7", Source: provLedgerStats},
			{Label: "Distinct source accounts", Value: "96", Source: vmv2.Provenance{
				Kind:   vmv2.ProvenanceDerived,
				Origin: "count(distinct sv_transactions_recent.source_account) where ledger_sequence = ?",
			}},
			{Label: "Contracts invoked", Value: "23", Source: provSorobanEndpoint},
		},
		Legend: []vmv2.LedgerV3LegendEntry{
			{Label: "Contract calls", Count: "61", Color: "--ph-violet"},
			{Label: "Payments", Count: "34", Color: "--ph-teal"},
			{Label: "Market orders", Count: "15", Color: "--ph-green"},
			{Label: "Deployments", Count: "3", Color: "--ph-amber"},
			{Label: "Other", Count: "5", Color: "--ph-border-strong"},
			{Label: "Failed", Count: "7", Color: "--ph-red"},
		},
		Note:   `<b>Why the bars are uneven.</b> A transaction is one unit of inclusion but any number of units of work — the tallest bars here carry seven operations, the shortest one. Counting transactions tells you how busy a ledger looked; counting operations tells you how busy it was.`,
		Source: provApplyOrder,
	}

	data.Capacity = vmv2.LedgerV3Capacity{
		Meters: []vmv2.LedgerV3Meter{
			{
				Name: "Ledger writes", Pct: 97, Used: "124 entries", Cap: "128 cap",
				Note:    "Every contract that stores something competes for these. This is what ran out.",
				Binding: true, Source: provFootprintEntries,
			},
			{
				Name: "CPU instructions", Pct: 71, Used: "71.2 M", Cap: "100 cap",
				Note:   "Soroban work fits comfortably; no call came close to its own budget.",
				Source: provCPUUtilization,
			},
			{
				Name: "Ledger reads", Pct: 45, Used: "89 entries", Cap: "200 cap",
				Note:   "Reads are cheap and plentiful. Never the binding constraint in practice.",
				Source: provFootprintEntries,
			},
			{
				Name: "Transaction size", Pct: 41, Used: "54.6 KB", Cap: "133 cap",
				Note:   "Total envelope bytes across all 118 transactions.",
				Source: provTxSizeBytes,
			},
		},
		Note: `<b>What this means for anyone deploying here.</b> A contract that writes many entries competes for the scarce resource; one that computes heavily does not. In this ledger a single oracle update wrote 14 entries — <b>11% of the entire ledger's write budget in one transaction</b>.<a class="pc-cite" href="#n6">6</a>`,
	}

	data.Fees = vmv2.LedgerV3Fees{
		ClearingLabel:   "Clearing fee, charged to all",
		ClearingValue:   "0.00042",
		ClearingUnit:    "XLM",
		BaseFee:         "0.00001 XLM",
		Multiple:        "42×",
		TotalCollected:  "0.0497 XLM",
		HighestBidNote:  `Against a base fee of <b>0.00001 XLM</b>. Everyone paid the same 42×, whether they bid it or bid far more — <b>the highest bid in this ledger was 17.8× the clearing price and was charged the clearing price</b>.`,
		HistoryFromNote: "12 ledgers ago · at base",
		HistoryToNote:   "this ledger · 42×",
		CannotTitle:     "What a fee buys, and what it does not",
		CannotBody:      `<b>Prism cannot tell you what was left out.</b> A closed ledger records what was included, not what competed. Transactions that bid below the clearing price were never written anywhere Prism can read — so no honest number exists for "how many missed this ledger."<a class="pc-cite" href="#n2">2</a>`,
		Source:          provFeesEndpoint,
		ExcludedSource:  provExcluded,
		History:         buildLedgerV3FeeHistory(),
	}

	data.Failures = vmv2.LedgerV3Failures{
		Source: provResultCodes,
		Groups: []vmv2.LedgerV3FailGroup{
			{
				Count:      4,
				Title:      `Blend refused four borrows <em>because the positions were already underwater</em>`,
				Detail:     "All four returned the same code within 1.4 seconds of each other. Consistent with a collateral price moving and positions not yet rebalanced — Prism can see the timing and the identical codes, not the oracle price that caused it.",
				Code:       "#4 HealthFactorTooLow",
				FeeNote:    "fees charged in full",
				ProtocolID: "Blend",
			},
			{
				Count:      2,
				Title:      `Soroswap refused two swaps <em>because the output fell below the floor the caller set</em>`,
				Detail:     "Ordinary slippage protection doing its job. Both callers set a 1.2% floor and the pool moved further than that between quote and execution. Not a fault.",
				Code:       "#8 InsufficientOutputAmount",
				FeeNote:    "fees charged in full",
				ProtocolID: "Soroswap",
			},
			{
				Count:      1,
				Title:      `One classic transaction was rejected <em>for using a sequence number that had already been consumed</em>`,
				Detail:     "A resubmission of something already applied, usually a client retrying after a timeout. No value moved and no fee was charged, because it never reached execution.",
				Code:       "tx_BAD_SEQ",
				FeeNote:    "no fee charged",
				ProtocolID: "classic",
			},
		},
		Note: `Six of seven failures were <b>contract logic declining to proceed</b>, which is the system working. Only the seventh is a client error. A raw failure count of 7 against a median of 1.3 invites the wrong conclusion.<a class="pc-cite" href="#n4">4</a>`,
	}

	// Every figure in this panel is the measured profile of mainnet ledger
	// 61,500,126, taken verbatim rather than invented. That ledger was chosen
	// because it is one of the few that exercises both eviction and
	// restoration; on most ledgers both are zero.
	//
	// The design's original numbers were wrong in three ways, and the shape of
	// the panel changes as a result:
	//
	//   - It labelled deletion "Archived, restorable for a fee". Temporary
	//     entries expiring are deleted and are NOT restorable. Only persistent
	//     entries evicted to the hot archive are. Those are now separate cells.
	//   - It put contract data at 85% of changes. Real ledgers are
	//     account-dominated; contract data is 14% here.
	//   - Its headline read "more data came back than expired". The real ratio
	//     is inverted, by about 90x.
	changeCells := []vmv2.LedgerV3ChangeCell{
		{Label: "Created", Count: "35", Note: "New accounts, trustlines, offers, and contract entries", Source: provEntryChanges},
		{Label: "Modified", Count: "3,131", Note: "Mostly balance updates on existing entries", Source: provEntryChanges},
		{Label: "Deleted", Count: "32", Note: "Entries removed outright. Temporary storage expiring lands here, and is gone rather than archived", Source: provEntryChanges},
		{Label: "Restored", Count: "22", Note: "Contract data pulled back out of the archive", Source: provArchivalKeys},
		{Label: "Archived", Count: "2,000", Note: "Persistent entries evicted to the hot archive. These are the restorable ones", Tone: "warn", Source: provArchivalKeys},
	}

	data.Changes = vmv2.LedgerV3Changes{
		Total: "3,220",
		Cells: changeCells,
		EntryTypes: []vmv2.LedgerV3EntryType{
			{Label: "Account", Count: "1,639"},
			{Label: "Trustline", Count: "743"},
			{Label: "Contract data", Count: "463"},
			{Label: "Offer", Count: "307"},
			{Label: "TTL", Count: "36"},
			{Label: "Liquidity pool", Count: "32"},
		},
		Note:   `<b>2,000 archived against 22 restored.</b> The first four cells are the change stream and sum to 3,220. Archival is not in that total — eviction is a ledger-level sweep, not a transaction's doing, which is why it can dwarf everything else. It runs at zero for long stretches and then hits its 1,000-entry-per-ledger cap, as here. Watching the archived-to-restored ratio is only meaningful across a window; on any single ledger it is usually 0 against 0.`,
		Source: provEntryChanges,
	}

	data.Chain = vmv2.LedgerV3Chain{
		Source: provLedgerStats,
		Note:   `Fees rose one ledger <b>before</b> this one and fell one ledger after. Anyone who submitted during the episode paid the premium; anyone who waited twenty seconds did not.`,
		Neighbors: []vmv2.LedgerV3Neighbor{
			{Sequence: formatThousands(seqNum - 3), Note: `<b>Ordinary</b>. Fees at base, nothing contested.`, TxCount: "86 tx", WritePct: 52, Href: ledgerV3Href(seqNum - 3)},
			{Sequence: formatThousands(seqNum - 2), Note: `Volume rising. Fees still at base.`, TxCount: "98 tx", WritePct: 61, Href: ledgerV3Href(seqNum - 2)},
			{Sequence: formatThousands(seqNum - 1), Note: `First ledger where fees cleared above base — surge pricing began here.`, TxCount: "104 tx", WritePct: 78, Href: ledgerV3Href(seqNum - 1)},
			{Sequence: formatThousands(seqNum), Note: `<b>This ledger</b>. Write capacity 97% full, mean fee 42× base.`, TxCount: "118 tx", WritePct: 97, Self: true, Href: "#"},
			{Sequence: formatThousands(seqNum + 1), Note: `Pressure easing. Fees fell to 11× base.`, TxCount: "91 tx", WritePct: 74, Href: ledgerV3Href(seqNum + 1)},
			{Sequence: formatThousands(seqNum + 2), Note: `Back to base fees. The episode lasted three ledgers.`, TxCount: "84 tx", WritePct: 58, Href: ledgerV3Href(seqNum + 2)},
		},
	}

	data.Notes = []vmv2.LedgerV3Note{
		{ID: "n1", Index: 1, Text: `<b>Read.</b> Resource usage summed from the <span class="mono">SorobanTransactionMeta</span> of every Soroban transaction in the set, against the network limits in the current config entry.`, Source: "soroban-tx-resources"},
		{ID: "n2", Index: 2, Text: `<b>Read, with a stated gap.</b> The clearing fee is the inclusion fee charged in <span class="mono">feeCharged</span>, identical across all included transactions. Excluded transactions are not recorded in any ledger — that number is unavailable, not withheld.`, Source: "feeCharged · all 118"},
		{ID: "n3", Index: 3, Text: `<b>Read.</b> 61 of 118 transactions contain at least one <span class="mono">InvokeHostFunction</span> operation. Classification is by operation type, not by heuristic.`, Source: "61 invocations"},
		{ID: "n4", Index: 4, Text: `<b>Read, grouped.</b> Error codes taken verbatim from each failed transaction's result. The <em>cause</em> attributed to the Blend group is an inference from shared code and timing — Prism cannot read the oracle price behind it.`, Source: "soroban-tx-resources"},
		{ID: "n5", Index: 5, Text: `<b>Protocol behaviour.</b> Stellar shuffles the apply order of a ledger's transaction set deterministically from the previous ledger's hash. Fee affects inclusion, never intra-ledger position.`, Source: "protocol 22"},
		{ID: "n6", Index: 6, Text: `<b>Read.</b> The Reflector oracle transaction wrote 14 ledger entries of the 124 written in this ledger.`, Source: "soroban-tx-resources"},
	}

	data.Rail = vmv2.LedgerV3Rail{
		Title:    "Ledger " + formatThousands(seqNum),
		Subtitle: "Write-limited · surge pricing active",
		Groups: []vmv2.LedgerV3RailGroup{
			{Heading: "Header", Rows: []vmv2.LedgerV3RailRow{
				{Label: "Sequence", Value: formatThousands(seqNum), Mono: true},
				{Label: "Closed", Value: "04 Aug 2026, 09:41:17"},
				{Label: "Close time", Value: "5.2s", Mono: true},
				{Label: "Protocol", Value: "22", Mono: true},
				{Label: "Previous", Value: formatThousands(seqNum - 1), Mono: true, Href: ledgerV3Href(seqNum - 1)},
			}},
			{Heading: "Contents", Rows: []vmv2.LedgerV3RailRow{
				{Label: "Transactions", Value: "118", Mono: true},
				{Label: "Operations", Value: "359", Mono: true},
				{Label: "Failed", Value: "7", Mono: true},
				{Label: "Soroban share", Value: "52%", Mono: true},
			}},
			{Heading: "Capacity", Rows: []vmv2.LedgerV3RailRow{
				{Label: "Writes", Value: "124 / 128", Mono: true},
				{Label: "CPU", Value: "71%", Mono: true},
				{Label: "Reads", Value: "45%", Mono: true},
				{Label: "Size", Value: "41%", Mono: true, IsGap: true},
			}},
			{Heading: "Fees", Rows: []vmv2.LedgerV3RailRow{
				{Label: "Clearing", Value: "0.00042 XLM", Mono: true},
				{Label: "Base", Value: "0.00001 XLM", Mono: true},
				{Label: "Multiple", Value: "42×", Mono: true},
				{Label: "Total collected", Value: "0.0497 XLM", Mono: true},
			}},
			{Heading: "State", Rows: []vmv2.LedgerV3RailRow{
				{Label: "Entry changes", Value: "3,220", Mono: true},
				{Label: "Archived", Value: "2,000", Mono: true},
				{Label: "Restored", Value: "22", Mono: true},
			}},
		},
		TOC: []vmv2.LedgerV3TOCEntry{
			{Label: "The transactions, in order", Href: "#s1"},
			{Label: "What actually filled up", Href: "#s2"},
			{Label: "Why fees were 42× base", Href: "#s3"},
			{Label: "What did not work", Href: "#s4"},
			{Label: "What it changed", Href: "#s5"},
			{Label: "Where it sits", Href: "#s6"},
			{Label: "Provenance", Href: "#notes"},
		},
		Footnote: `<b>How to read this page.</b> Claims come first; the evidence for each sits in Provenance, linked by number. Where a question has no on-chain answer, the page says so rather than filling the gap.`,
	}

	data.TxPane = buildLedgerV3TxPane()
	data.StatePane = buildLedgerV3StatePane(changeCells)
	data.Gaps = ledgerV3Gaps()

	return data
}

func ledgerV3Href(seq int64) string {
	return "/v2/ledger/" + strconv.FormatInt(seq, 10) + "/v3"
}

func buildLedgerV3FeeHistory() []vmv2.LedgerV3FeeBar {
	bars := make([]vmv2.LedgerV3FeeBar, 0, 12)
	for range 10 {
		bars = append(bars, vmv2.LedgerV3FeeBar{HeightPct: 4, Title: "0.00001 XLM · 1× base"})
	}
	bars = append(bars,
		vmv2.LedgerV3FeeBar{HeightPct: 34, Active: true, Title: "0.00014 XLM · 14× base"},
		vmv2.LedgerV3FeeBar{HeightPct: 100, Active: true, Title: "0.00042 XLM · 42× base"},
	)
	return bars
}

func buildLedgerV3TxPane() vmv2.LedgerV3TxPane {
	pane := vmv2.LedgerV3TxPane{
		Title:       "Every transaction in this ledger",
		Intro:       "All 118, in apply order. Filter by kind, outcome, or protocol — the distribution above the table redraws to match, so you can see where in the five seconds your filter lands.",
		SaidLead:    `Showing <b>all 118 transactions</b> in apply order.`,
		ShownLabel:  "10 shown of 118",
		TotalLabel:  "10 of 118 shown",
		SortOptions: []string{"Apply order", "Fee", "Operations"},
		Facets: []vmv2.LedgerV3Facet{
			{Key: "kind", Label: "Kind", PopHead: "Transaction kind", Source: provTxRecent, Options: []vmv2.LedgerV3FacetOption{
				{Value: "calls", Label: "Contract call", Count: "61"},
				{Value: "payments", Label: "Payment", Count: "34"},
				{Value: "markets", Label: "Market order", Count: "15"},
				{Value: "deployments", Label: "Deployment", Count: "3"},
				{Value: "other", Label: "Other", Count: "5"},
			}},
			{Key: "res", Label: "Outcome", PopHead: "Result", Source: provTxRecent, Options: []vmv2.LedgerV3FacetOption{
				{Value: "ok", Label: "Applied", Count: "111"},
				{Value: "fail", Label: "Failed", Count: "7"},
			}},
			{Key: "proto", Label: "Protocol", PopHead: "Known protocol", Source: provContractLabels, Options: []vmv2.LedgerV3FacetOption{
				{Value: "soroswap", Label: "Soroswap", Count: "28"},
				{Value: "blend", Label: "Blend", Count: "17"},
				{Value: "reflector", Label: "Reflector", Count: "4"},
				{Value: "unlabelled", Label: "Unlabelled", Count: "12"},
			}},
			{Key: "writes", Label: "Writes", PopHead: "Ledger writes consumed", Source: provTxWriteEntries, Options: []vmv2.LedgerV3FacetOption{
				{Value: "none", Label: "None", Count: "51"},
				{Value: "low", Label: "1 – 3", Count: "54"},
				{Value: "high", Label: "4 or more", Count: "13"},
			}},
		},
		Distribution: vmv2.LedgerV3Distribution{
			Heading: "Position within the five seconds",
			Aside:   "12 buckets · 118 transactions",
			AxisMin: "0.0s", AxisMid: "2.6s", AxisMax: "5.2s",
			Legend: "Apply order is protocol-randomised, so this distribution is flat by design — a cluster would mean something is wrong with the shuffle, not with the traffic.",
		},
	}

	counts := []int{10, 10, 10, 10, 10, 9, 10, 10, 10, 10, 10, 9}
	for _, c := range counts {
		pane.Distribution.Bars = append(pane.Distribution.Bars, vmv2.LedgerV3DistBar{
			HeightPct: c * 10,
			Title:     fmt.Sprintf("%d transactions", c),
		})
	}

	pane.Rows = []vmv2.LedgerV3Row{
		{Kind: "calls", Result: "ok", Stamp: "+0.4s", Who: "GBQ4…M2XP", Status: "Applied", Href: "#",
			Say:  `<em>Swapped</em> <span class="amt">1,240.00 USDC</span> <em>for</em> <span class="amt">4,118.42 XLM</span>`,
			Meta: `8 frames · 0.0042 XLM · 42× base`},
		{Kind: "payments", Result: "ok", Stamp: "+0.7s", Who: "GANC…P4KD", Status: "Applied", Href: "#",
			Say:  `<em>Sent</em> <span class="amt">5,000.00 USDC</span> <em>to</em> <span class="amt">GNEW…2Q7X</span>`,
			Meta: `sponsored onboarding · 7 operations`},
		{Kind: "calls", Result: "fail", Stamp: "+1.1s", Who: "GDQ7…4M8P", Status: "Failed", Failed: true, Href: "#",
			Say:  `<em>Tried to borrow</em> <span class="amt">2,400.00 USDC</span> — <em>reverted</em>`,
			Meta: `<span class="err">#4 HealthFactorTooLow</span> · fee still charged`,
		},
		{Kind: "markets", Result: "ok", Stamp: "+1.4s", Who: "GMKT…8H2C", Status: "Applied", Href: "#",
			Say:  `<em>Filled a</em> <span class="amt">12,000 XLM</span> <em>sell order at 0.2951</em>`,
			Meta: `partial fill 74% · classic DEX`},
		{Kind: "calls", Result: "ok", Stamp: "+1.9s", Who: "GLEN…5K1D", Status: "Applied", Href: "#",
			Say:  `<em>Repaid</em> <span class="amt">900.00 USDC</span> <em>and withdrew collateral</em>`,
			Meta: `health factor 1.41 → 2.08`},
		{Kind: "deployments", Result: "ok", Stamp: "+2.3s", Who: "GDEP…7T3F", Status: "Applied", Href: "#",
			Say:  `<em>Deployed a contract</em> <span class="amt">CFORK92A…8XM4</span>`,
			Meta: `bytecode matches a known router · unverified`},
		{Kind: "calls", Result: "fail", Stamp: "+2.8s", Who: "GSLP…9W4B", Status: "Failed", Failed: true, Href: "#",
			Say:  `<em>Tried to swap</em> <span class="amt">40,000 XLM</span> — <em>output below floor</em>`,
			Meta: `<span class="err">#8 InsufficientOutputAmount</span> · slippage 1.2%`,
		},
		{Kind: "payments", Result: "ok", Stamp: "+3.4s", Who: "GB3Q…KM2W", Status: "Applied", Href: "#",
			Say:  `<em>Received</em> <span class="amt">240.00 EURC</span> <em>from an anchor</em>`,
			Meta: `memo: settlement 08-01`},
		{Kind: "calls", Result: "ok", Stamp: "+4.1s", Who: "GORC…2WN8", Status: "Applied", Href: "#",
			Say:  `<em>Wrote</em> <span class="amt">14 price feeds</span> <em>to storage</em>`,
			Meta: `14 writes · the largest single writer here`},
		{Kind: "other", Result: "ok", Stamp: "+4.9s", Who: "GTRU…6J8N", Status: "Applied", Href: "#",
			Say:  `<em>Opened a trustline to</em> <span class="amt">USDC</span>`,
			Meta: `locked 0.5 XLM of reserve`},
	}

	return pane
}

func buildLedgerV3StatePane(cells []vmv2.LedgerV3ChangeCell) vmv2.LedgerV3StatePane {
	return vmv2.LedgerV3StatePane{
		Title:      "Every entry this ledger changed",
		Intro:      "3,220 changes, grouped by the entry they touched. Two thousand more entries were evicted to the archive by the ledger itself, which is not a change any transaction made and so is counted apart from these.",
		SaidLead:   `Showing <b>all 3,220 changes</b>, grouped by entry.`,
		Cells:      cells,
		ShownLabel: "5 shown of 3,220",
		TotalLabel: "Notable changes",
		SortOption: []string{"By impact", "By entry"},
		Facets: []vmv2.LedgerV3Facet{
			{Key: "ct", Label: "Change", PopHead: "Change type", Source: provEntryChanges, Options: []vmv2.LedgerV3FacetOption{
				{Value: "new", Label: "Created", Count: "35"},
				{Value: "mod", Label: "Modified", Count: "3,131"},
				{Value: "del", Label: "Deleted", Count: "32"},
				{Value: "res", Label: "Restored", Count: "22"},
				{Value: "arc", Label: "Archived", Count: "2,000"},
			}},
			{Key: "et", Label: "Entry type", PopHead: "Entry type", Source: provEntryChanges, Options: []vmv2.LedgerV3FacetOption{
				{Value: "ac", Label: "Account", Count: "1,639"},
				{Value: "tl", Label: "Trustline", Count: "743"},
				{Value: "cd", Label: "Contract data", Count: "463"},
				{Value: "of", Label: "Offer", Count: "307"},
				{Value: "ttl", Label: "TTL", Count: "36"},
				{Value: "lp", Label: "Liquidity pool", Count: "32"},
			}},
		},
		Rows: []vmv2.LedgerV3Row{
			// The ledger-level totals above are served. These rows are not: they
			// name the contract behind each change, and ledger-change-stats
			// aggregates evictions and restorations to {entry_type, count}.
			{Kind: "arc", Stamp: "arc", Who: "CNFT8823…QP41", Status: "Archived", Href: "#",
				Say:   `<em>A thousand storage entries evicted, and the thousand TTL entries paired with them</em>`,
				Meta:  `the sweep's per-ledger cap · restorable for a fee · spread across many contracts`,
				IsGap: true, GapNote: "which contracts these belonged to is unattributed, as is the restore cost"},
			{Kind: "res", Stamp: "res", Who: "CBLEND7X…4KQ2", Status: "Restored", Href: "#",
				Say:   `<em>Eleven entries pulled back out of the archive, with their eleven TTLs</em>`,
				Meta:  `restored inside an ordinary contract call, with no restore operation of its own`,
				IsGap: true, GapNote: "which contract was restored, and at what rent cost, are both unattributed"},
			{Kind: "new", Stamp: "new", Who: "CFORK92A…8XM4", Status: "Created", Href: "#",
				Say:   `<em>A contract instance was created at</em> <span class="amt">CFORK92A…8XM4</span>`,
				Meta:  `bytecode matches a known router · unverified deployer`,
				IsGap: true, GapNote: "creation count is served; the created entry's identity is not"},
			// Served: readWriteEntries is per transaction, so joining on tx_hash
			// to sv_transactions_recent.primary_contract_id attributes the share.
			{Kind: "mod", Stamp: "mod", Who: "CORCL55B…2WN8", Status: "Modified", Href: "#",
				Say:  `<em>Fourteen price feeds updated on</em> <span class="amt">CORCL55B…2WN8</span>`,
				Meta: `14 writes · 11% of this ledger's entire write budget`},
			{Kind: "new", Stamp: "new", Who: "GNEW…2Q7X", Status: "Created", Href: "#",
				Say:   `<em>An account was created at</em> <span class="amt">GNEW…2Q7X</span> <em>with four sponsored entries</em>`,
				Meta:  `2.5 XLM of reserve billed to the anchor`,
				IsGap: true, GapNote: "creation count is served; the created account's identity is not"},
		},
	}
}

func ledgerV3Gaps() []vmv2.LedgerV3Gap {
	return []vmv2.LedgerV3Gap{
		{
			Section:  "What it changed · notable changes",
			Field:    "Which contract was archived or restored",
			Missing:  provContractAttribution.Note,
			Proposal: "Emit contract IDs alongside the counts. EvictedLedgerKeys() returns LedgerKeys that already carry the contract address — the processor discards it during aggregation.",
		},
		{
			Section:  "What it changed · notable changes",
			Field:    "Restore and archival cost in XLM (0.4, 1.82)",
			Missing:  provArchivalCost.Note,
			Proposal: "Either compute rent from entry size and the state-archival config settings, or drop the cost from these rows and keep the counts.",
		},
		{
			Section:  "What actually filled up",
			Field:    "Transaction size meter (54.6 KB of 133 cap)",
			Missing:  provTxSizeBytes.Note,
			Proposal: "Scope the meter to Soroban transactions so the cap applies, or show the byte total with no cap. This is a design decision, not missing data.",
		},
	}
}

// formatThousands renders an integer with comma separators.
func formatThousands(n int64) string {
	neg := n < 0
	if neg {
		n = -n
	}
	s := strconv.FormatInt(n, 10)
	if len(s) <= 3 {
		if neg {
			return "-" + s
		}
		return s
	}
	var b strings.Builder
	lead := len(s) % 3
	if lead > 0 {
		b.WriteString(s[:lead])
	}
	for i := lead; i < len(s); i += 3 {
		if b.Len() > 0 {
			b.WriteByte(',')
		}
		b.WriteString(s[i : i+3])
	}
	if neg {
		return "-" + b.String()
	}
	return b.String()
}
