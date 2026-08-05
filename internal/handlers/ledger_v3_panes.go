package handlers

import (
	"fmt"
	"net/url"
	"sort"
	"strings"

	"github.com/withObsrvr/prism/internal/gateway"
	vmv2 "github.com/withObsrvr/prism/internal/templates/v2/viewmodel"
)

// ledgerV3EntryTypeRow pairs an entry type with its change counts so the rows
// can be ordered by volume before rendering.
type ledgerV3EntryTypeRow struct {
	name   string
	counts gateway.LedgerChangeTypeCounts
}

var provLiveTransactions = vmv2.Provenance{
	Kind:   vmv2.ProvenanceServed,
	Origin: "GET /silver/ledgers/{seq}/full .transactions and .operations",
	Note:   "One row per transaction, in apply order, described from its operations.",
}

// provNoSemanticReading marks what the operation rows cannot say. The mock
// wrote lines like \"swapped 1,240 USDC for 4,118 XLM\", which needs the
// transaction's semantic reading — the events and balance deltas — not the
// operation list. Those are per-transaction endpoints, and calling one per row
// would be a request per transaction.
var provNoSemanticReading = vmv2.Provenance{
	Kind:   vmv2.ProvenanceGap,
	Origin: "no ledger-scoped semantic transaction projection",
	Note:   "Rows describe the operations a transaction carried. What it meant — amounts swapped, positions changed — needs the per-transaction semantic reading, which has no ledger-wide equivalent.",
}

// overlayLedgerV3Panes fills the transactions and state-changes tabs.
//
// Both are built only from what the ledger endpoints return. Where the design
// asked for something no projection serves, the pane says so rather than
// keeping the mock's illustrative rows, which describe a different ledger.
func (h *Handlers) overlayLedgerV3Panes(
	data *vmv2.LedgerDetailV3Data,
	network string,
	txs []gateway.Transaction,
	ops []gateway.Operation,
	changes *gateway.LedgerChanges,
) {
	if len(txs) > 0 {
		applyLedgerV3TxPane(data, network, txs, ops)
		applyLedgerV3Failures(data, txs, ops)
	}
	if changes != nil && changes.Available {
		applyLedgerV3StatePane(data, changes)
	}
}

// applyLedgerV3TxPane rebuilds the transactions tab from the ledger's own
// transactions and operations.
func applyLedgerV3TxPane(data *vmv2.LedgerDetailV3Data, network string, txs []gateway.Transaction, ops []gateway.Operation) {
	opsByTx := make(map[string][]gateway.Operation, len(txs))
	for _, op := range ops {
		opsByTx[op.TransactionHash] = append(opsByTx[op.TransactionHash], op)
	}

	rows := make([]vmv2.LedgerV3Row, 0, len(txs))
	kindCounts := map[string]int{}
	failed := 0

	for i, tx := range txs {
		txOps := opsByTx[tx.TransactionHash]
		kind := ledgerV3TxKind(txOps)
		kindCounts[kind]++

		result, status := "ok", "Applied"
		if !tx.Successful {
			result, status = "fail", "Failed"
			failed++
		}

		rows = append(rows, vmv2.LedgerV3Row{
			Kind:   kind,
			Result: result,
			// Apply order is the position, not a timestamp: every transaction
			// in a ledger shares the ledger's close time, so a per-row clock
			// would be the same value repeated.
			Stamp:  fmt.Sprintf("#%d", i+1),
			Who:    shortAccount(tx.SourceAccount),
			Status: status,
			Failed: !tx.Successful,
			Href:   ledgerV3TxHref(tx.TransactionHash, network),
			Say:    ledgerV3TxSay(txOps, tx.OperationCount),
			Meta:   ledgerV3TxMeta(tx),
			Search: ledgerV3TxSearchTerms(tx, txOps, kind),
		})
	}

	total := len(txs)
	pane := vmv2.LedgerV3TxPane{
		Title: "Every transaction in this ledger",
		Intro: fmt.Sprintf("All %d, in apply order. Filter by kind or outcome — the distribution above the table redraws to match.", total),
		SaidLead: fmt.Sprintf("Showing <b>all %d transaction%s</b> in apply order.",
			total, plural(total)),
		ShownLabel:  fmt.Sprintf("%d shown of %d", total, total),
		TotalLabel:  fmt.Sprintf("%d of %d shown", total, total),
		SortOptions: []string{"Apply order"},
		Rows:        rows,
		Facets: []vmv2.LedgerV3Facet{
			{
				Key: "kind", Label: "Kind", PopHead: "What the transaction did",
				Source:  provLiveTransactions,
				Options: ledgerV3KindOptions(kindCounts, total),
			},
			{
				Key: "res", Label: "Outcome", PopHead: "Applied or failed",
				Source: provLiveTransactions,
				Options: []vmv2.LedgerV3FacetOption{
					{Value: "", Label: "Any", Count: fmt.Sprintf("%d", total)},
					{Value: "ok", Label: "Applied", Count: fmt.Sprintf("%d", total-failed)},
					{Value: "fail", Label: "Failed", Count: fmt.Sprintf("%d", failed)},
				},
			},
		},
		Distribution: ledgerV3ApplyDistribution(txs),
	}

	data.TxPane = pane
}

var provLiveFailures = vmv2.Provenance{
	Kind:   vmv2.ProvenanceServed,
	Origin: "GET /silver/ledgers/{seq}/full .transactions .result_code and .contract_error_type",
	Note:   "Failures grouped by the canonical result code, with the contract error where the transaction emitted one.",
}

// applyLedgerV3Failures groups the ledger's failed transactions by cause.
//
// Grouping is the point of the section. Seven failures listed separately read
// as network trouble; the same seven grouped show that four shared one code
// and belong to one protocol having a bad five seconds. That distinction needs
// the result code, which the ledger response now carries.
//
// A contract error, where present, is the more specific cause and takes
// precedence over the transaction-level code: tx_FAILED says a Soroban
// transaction failed, while the ScError says what the contract objected to.
func applyLedgerV3Failures(data *vmv2.LedgerDetailV3Data, txs []gateway.Transaction, ops []gateway.Operation) {
	opsByTx := make(map[string][]gateway.Operation, len(txs))
	for _, op := range ops {
		opsByTx[op.TransactionHash] = append(opsByTx[op.TransactionHash], op)
	}

	type failGroup struct {
		code     string
		contract string
		count    int
		ops      map[string]int
		charged  int64
		free     bool
	}
	byCause := map[string]*failGroup{}
	order := make([]string, 0, 4)
	failed := 0

	for _, tx := range txs {
		if tx.Successful {
			continue
		}
		failed++

		cause := tx.ResultCode
		if cause == "" {
			cause = "cause not recorded"
		}
		if tx.ContractErrorType != "" {
			cause = ledgerV3ContractErrorLabel(tx)
		}

		g, ok := byCause[cause]
		if !ok {
			g = &failGroup{code: cause, contract: tx.ContractErrorType, ops: map[string]int{}}
			byCause[cause] = g
			order = append(order, cause)
		}
		g.count++
		g.charged += tx.FeeCharged
		if tx.FeeCharged == 0 {
			g.free = true
		}
		for _, op := range opsByTx[tx.TransactionHash] {
			g.ops[humanOperationName(op.TypeName)]++
		}
	}

	// Largest cause first: the group that explains the most failures is the
	// one worth reading.
	sort.SliceStable(order, func(i, j int) bool {
		return byCause[order[i]].count > byCause[order[j]].count
	})

	groups := make([]vmv2.LedgerV3FailGroup, 0, len(order))
	for _, key := range order {
		g := byCause[key]

		// A group whose transactions were charged nothing never reached
		// execution — a fee is what the network takes for doing the work of
		// rejecting something, so its absence means it did not get that far.
		feeNote := "fees charged in full"
		if g.free && g.charged == 0 {
			feeNote = "no fee charged"
		}

		groups = append(groups, vmv2.LedgerV3FailGroup{
			Count:  g.count,
			Title:  ledgerV3FailTitle(g.count, g.ops),
			Detail: ledgerV3FailDetail(g.count, g.charged, g.contract),
			Code:   g.code,
			FeeNote: feeNote,
		})
	}

	data.Failures = vmv2.LedgerV3Failures{
		Source: provLiveFailures,
		Groups: groups,
	}

	switch {
	case failed == 0:
		data.Failures.Aside = "no failures"
		data.Failures.Intro = "Every transaction this ledger included was applied."
		data.Failures.Note = "<b>Nothing failed in this ledger.</b> Every transaction it included was applied."
	default:
		causes := len(order)
		largest := 0
		for _, g := range byCause {
			if g.count > largest {
				largest = g.count
			}
		}
		data.Failures.Aside = fmt.Sprintf("%d failure%s · %d cause%s", failed, plural(failed), causes, plural(causes))
		data.Failures.Intro = "Grouped by result code rather than listed. Separate failures reading as one list look like network trouble; the same failures grouped show whether one cause explains most of them."
		data.Failures.Note = ledgerV3FailuresNote(failed, causes, largest)
	}
}

// ledgerV3ContractErrorLabel names a contract failure by its error type and,
// where the contract defined one, its code. Only contract-defined errors carry
// a number, so a host error is reported by type alone rather than as code 0.
func ledgerV3ContractErrorLabel(tx gateway.Transaction) string {
	if tx.ContractErrorType == "contract" && tx.ContractErrorCode != 0 {
		return fmt.Sprintf("contract error #%d", tx.ContractErrorCode)
	}
	return fmt.Sprintf("%s error", strings.ReplaceAll(tx.ContractErrorType, "_", " "))
}

func ledgerV3FailTitle(count int, opCounts map[string]int) string {
	verb := "transactions failed"
	if count == 1 {
		verb = "transaction failed"
	}

	if len(opCounts) == 0 {
		return fmt.Sprintf("%d %s", count, verb)
	}

	// Name the operation the group's transactions were most often attempting,
	// which is usually what the reader wants to know about the failure.
	top, topN := "", 0
	for name, n := range opCounts {
		if n > topN || (n == topN && name < top) {
			top, topN = name, n
		}
	}
	return fmt.Sprintf("%d %s <em>while attempting %s</em>", count, verb, top)
}

func ledgerV3FailDetail(count int, charged int64, contractError string) string {
	var b strings.Builder
	if count > 1 {
		b.WriteString(fmt.Sprintf("All %d returned the same code. ", count))
	}
	if contractError != "" {
		b.WriteString("The contract itself rejected the call rather than the network refusing the transaction. ")
	}
	if charged > 0 {
		b.WriteString(fmt.Sprintf("Charged %s XLM in total, because the network did the work of rejecting %s.",
			formatStroopsXLM(charged), pluralIt(count)))
	} else {
		b.WriteString("Nothing was charged, so execution was never reached.")
	}
	return b.String()
}

// ledgerV3FailuresNote states whether the failures concentrate on one cause,
// which is the reading the grouping exists to support. A raw failure count
// invites the wrong conclusion in both directions: several failures sharing
// one code are one problem, while the same number spread across distinct codes
// are unrelated events that happened to land together.
func ledgerV3FailuresNote(failed, causes, largest int) string {
	switch {
	case causes == 1 && failed > 1:
		return fmt.Sprintf(
			`<b>All %d failures share one code.</b> That is one problem, not %d — a single cause reaching several transactions in the same five seconds. A raw count of %d invites the opposite reading.`,
			failed, failed, failed)
	case largest > 1 && largest*2 >= failed:
		return fmt.Sprintf(
			`<b>%d of the %d failures share one code.</b> The rest are unrelated, so the count worth quoting is %d causes rather than %d failures.`,
			largest, failed, causes, failed)
	case failed == 1:
		return `<b>One transaction failed.</b> A single failure is an event rather than a pattern; there is nothing here to generalise from.`
	default:
		return fmt.Sprintf(
			`<b>%d failures across %d distinct codes.</b> Nothing concentrates, so these are unrelated events that happened to land in the same ledger rather than one fault reaching several transactions.`,
			failed, causes)
	}
}

func pluralIt(n int) string {
	if n == 1 {
		return "it"
	}
	return "them"
}

// ledgerV3TxSearchTerms collects what a reader might type that the row's
// rendered text does not already contain: the full hash and account rather
// than their abbreviations, every operation type rather than the first three,
// and the kind label shown in the facet beside the search box.
func ledgerV3TxSearchTerms(tx gateway.Transaction, ops []gateway.Operation, kind string) string {
	terms := []string{
		tx.TransactionHash,
		tx.SourceAccount,
		kind,
		ledgerV3KindLabel(kind),
		tx.ResultCode,
		tx.ContractErrorType,
	}
	if tx.Successful {
		terms = append(terms, "applied", "success")
	} else {
		terms = append(terms, "failed", "failure")
	}

	seen := map[string]bool{}
	for _, op := range ops {
		name := humanOperationName(op.TypeName)
		if seen[name] {
			continue
		}
		seen[name] = true
		terms = append(terms, name, op.TypeName)
		if op.Destination != "" {
			terms = append(terms, op.Destination)
		}
		if op.SorobanContract != "" {
			terms = append(terms, op.SorobanContract)
		}
		if op.SorobanFunction != "" {
			terms = append(terms, op.SorobanFunction)
		}
	}

	out := terms[:0]
	for _, t := range terms {
		if t != "" {
			out = append(out, t)
		}
	}
	return strings.Join(out, " ")
}

// ledgerV3TxHref links a row to the v2 transaction page, carrying the network.
//
// v2 is the current surface; /tx/{hash} is the legacy one, and sending readers
// there from a v3 page moves them backwards a generation. The network has to
// travel with the link because the transaction page loads its contents through
// fragment requests that inherit nothing from the page — without it they
// resolve to the default network, find nothing, and leave the page on its
// loading skeleton.
func ledgerV3TxHref(hash, network string) string {
	href := "/v2/tx/" + url.PathEscape(hash)
	if network == "" {
		return href
	}
	return href + "?network=" + url.QueryEscape(network)
}

// ledgerV3KindLabel names a kind in the singular, for search terms and prose.
func ledgerV3KindLabel(kind string) string {
	switch kind {
	case "calls":
		return "contract call"
	case "payments":
		return "payment"
	case "markets":
		return "market order"
	case "deployments":
		return "deployment"
	default:
		return "other"
	}
}

// ledgerV3KindLabelPlural names a kind for the facet list. "Other" has no
// plural that reads well, so it is left alone rather than forced into one.
func ledgerV3KindLabelPlural(kind string) string {
	if kind == "other" {
		return "Other"
	}
	return capitalise(ledgerV3KindLabel(kind)) + "s"
}

// ledgerV3TxKind classifies a transaction by the operations it carried.
// Soroban wins over the classic families because a contract call is the more
// specific fact about a transaction that contains one.
func ledgerV3TxKind(ops []gateway.Operation) string {
	if len(ops) == 0 {
		return "other"
	}
	for _, op := range ops {
		if op.IsSorobanOp {
			if strings.Contains(strings.ToUpper(op.TypeName), "CREATE") {
				return "deployments"
			}
			return "calls"
		}
	}
	for _, op := range ops {
		switch {
		case op.IsPaymentOp:
			return "payments"
		case strings.Contains(op.TypeName, "OFFER"),
			strings.Contains(op.TypeName, "LIQUIDITY_POOL"):
			return "markets"
		}
	}
	return "other"
}

// ledgerV3TxSay describes a transaction from its operations. It states what
// the operations were rather than what they meant, which is the most the
// ledger endpoint supports.
func ledgerV3TxSay(ops []gateway.Operation, declaredOps int) string {
	if len(ops) == 0 {
		if declaredOps > 0 {
			return fmt.Sprintf("Carried %d operation%s", declaredOps, plural(declaredOps))
		}
		return "No operations recorded"
	}

	// Name the distinct operation types in the order they appear, so a
	// multi-operation transaction reads as what it actually did rather than
	// as its first step alone.
	seen := map[string]bool{}
	names := make([]string, 0, 3)
	for _, op := range ops {
		label := humanOperationName(op.TypeName)
		if seen[label] {
			continue
		}
		seen[label] = true
		names = append(names, label)
		if len(names) == 3 {
			break
		}
	}

	desc := strings.Join(names, ", ")
	if len(seen) > 3 {
		desc += fmt.Sprintf(" and %d more", len(seen)-3)
	}

	first := ops[0]
	if first.IsSorobanOp && first.SorobanFunction != "" {
		return fmt.Sprintf(`<em>Invoked</em> <span class="amt">%s</span> <em>on</em> <span class="amt">%s</span>`,
			first.SorobanFunction, shortContract(first.SorobanContract))
	}
	if first.IsPaymentOp && first.Amount != "" && first.Destination != "" {
		return fmt.Sprintf(`<em>Sent</em> <span class="amt">%s</span> <em>to</em> <span class="amt">%s</span>`,
			formatStroopAmount(first.Amount), shortAccount(first.Destination))
	}

	// Not wrapped in <em>. In this row, em marks the connective words around a
	// value — the "Sent" and "to" in "Sent 500 XLM to GBEF…" — and is styled
	// back accordingly. The operation list is the row's actual content, so
	// emphasising all of it renders the primary line as though it were
	// subordinate to something, which is how it came to look washed out.
	return desc
}

// ledgerV3TxMeta labels the fee as a bid, because that is what the ledger
// response carries. MaxFee is what the transaction was willing to pay; what it
// was actually charged is the ledger's clearing price, which is usually far
// lower. Calling the bid "fee" would overstate what this transaction cost by
// whatever margin its sender left.
func ledgerV3TxMeta(tx gateway.Transaction) string {
	if !tx.Successful && tx.ResultCode != "" {
		reason := tx.ResultCode
		if tx.ContractErrorType != "" {
			reason = ledgerV3ContractErrorLabel(tx)
		}
		return fmt.Sprintf(`%d operation%s · <span class="err">%s</span> · paid %s XLM`,
			tx.OperationCount, plural(tx.OperationCount), reason, formatStroopsXLM(tx.FeeCharged))
	}
	return fmt.Sprintf("%d operation%s · paid %s XLM of %s bid",
		tx.OperationCount, plural(tx.OperationCount),
		formatStroopsXLM(tx.FeeCharged), formatStroopsXLM(tx.MaxFee))
}

func ledgerV3KindOptions(counts map[string]int, total int) []vmv2.LedgerV3FacetOption {
	opts := []vmv2.LedgerV3FacetOption{{Value: "", Label: "Any", Count: fmt.Sprintf("%d", total)}}
	keys := make([]string, 0, len(counts))
	for k := range counts {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if counts[keys[i]] != counts[keys[j]] {
			return counts[keys[i]] > counts[keys[j]]
		}
		return keys[i] < keys[j]
	})
	for _, k := range keys {
		opts = append(opts, vmv2.LedgerV3FacetOption{
			Value: k, Label: ledgerV3KindLabelPlural(k), Count: fmt.Sprintf("%d", counts[k]),
		})
	}
	return opts
}

// ledgerV3ApplyDistribution buckets transactions by position in apply order.
//
// Position, not time: the protocol randomises apply order and every
// transaction shares the ledger's close time, so this shows the shape of the
// shuffle. A flat distribution is the expected result.
func ledgerV3ApplyDistribution(txs []gateway.Transaction) vmv2.LedgerV3Distribution {
	const buckets = 12
	total := len(txs)

	counts := make([]int, buckets)
	for i := range txs {
		b := i * buckets / total
		if b >= buckets {
			b = buckets - 1
		}
		counts[b]++
	}

	maxCount := 1
	for _, c := range counts {
		if c > maxCount {
			maxCount = c
		}
	}

	bars := make([]vmv2.LedgerV3DistBar, 0, buckets)
	for _, c := range counts {
		bars = append(bars, vmv2.LedgerV3DistBar{
			HeightPct: c * 100 / maxCount,
			Title:     fmt.Sprintf("%d transaction%s", c, plural(c)),
		})
	}

	return vmv2.LedgerV3Distribution{
		Heading: "Position within apply order",
		Aside:   fmt.Sprintf("%d buckets · %d transaction%s", buckets, total, plural(total)),
		AxisMin: "first",
		AxisMid: "middle",
		AxisMax: "last",
		Bars:    bars,
		Legend:  "Apply order is protocol-randomised, so this distribution is flat by design — a cluster would mean something is wrong with the shuffle, not with the traffic.",
	}
}

var provLiveEntryTypes = vmv2.Provenance{
	Kind:   vmv2.ProvenanceServed,
	Origin: "GET /silver/ledgers/{seq}/changes .by_type",
	Note:   "Counts per ledger entry type, read from the ledger change stream.",
}

// provNoPerEntryChanges marks the one thing this pane cannot show. Counts per
// entry type are served; the individual entries behind them are not projected
// at ledger scope, and reconstructing them would mean a request per
// transaction.
var provNoPerEntryChanges = vmv2.Provenance{
	Kind:   vmv2.ProvenanceGap,
	Origin: "no ledger-scoped per-entry change projection",
	Note:   "Which specific ledger entries changed is available per transaction, not per ledger. Listing them here needs a projection that records each changed entry against its ledger.",
}

// applyLedgerV3StatePane rebuilds the state-changes tab.
//
// Rows are one per entry type rather than one per changed entry, because the
// per-entry detail is not projected at ledger scope. That is stated in the
// pane instead of being filled with the mock's example entries.
func applyLedgerV3StatePane(data *vmv2.LedgerDetailV3Data, changes *gateway.LedgerChanges) {
	rows := make([]vmv2.LedgerV3Row, 0, len(changes.ByType))

	ordered := make([]ledgerV3EntryTypeRow, 0, len(changes.ByType))
	for name, counts := range changes.ByType {
		if counts.Total == 0 {
			continue
		}
		ordered = append(ordered, ledgerV3EntryTypeRow{name: name, counts: counts})
	}
	sort.Slice(ordered, func(i, j int) bool {
		if ordered[i].counts.Total != ordered[j].counts.Total {
			return ordered[i].counts.Total > ordered[j].counts.Total
		}
		return ordered[i].name < ordered[j].name
	})

	for _, t := range ordered {
		rows = append(rows, vmv2.LedgerV3Row{
			Kind:   t.name,
			Result: "ok",
			Stamp:  formatThousands(t.counts.Total),
			Who:    entryTypeLabel(t.name),
			Status: "Changed",
			Say: fmt.Sprintf(`<em>%s entries changed</em> <span class="amt">%s</span>`,
				entryTypeLabel(t.name), formatThousands(t.counts.Total)),
			Meta: changeBreakdown(t.counts),
		})
	}

	// The gap row sits with the data rather than in a footnote, so the reader
	// meets it at the point they would otherwise wonder where the entries are.
	rows = append(rows, vmv2.LedgerV3Row{
		IsGap:   true,
		GapNote: "Which individual entries changed is not projected at ledger scope. The counts above are complete; the entries behind them can only be read one transaction at a time.",
	})

	data.StatePane = vmv2.LedgerV3StatePane{
		Title: "What this ledger changed",
		Intro: fmt.Sprintf("%s entr%s changed across %d type%s. Eviction is listed apart from the total, because a sweep is the protocol's doing rather than a transaction's.",
			formatThousands(changes.Total), pluralY(changes.Total), len(ordered), plural(len(ordered))),
		SaidLead: fmt.Sprintf("Showing <b>%d entry type%s</b> by volume.", len(ordered), plural(len(ordered))),
		Cells:    data.Changes.Cells,
		Rows:     rows,
		ShownLabel: fmt.Sprintf("%d type%s shown", len(ordered), plural(len(ordered))),
		TotalLabel: fmt.Sprintf("%s change%s total", formatThousands(changes.Total), plural(int(changes.Total))),
		SortOption: []string{"Volume"},
		Facets: []vmv2.LedgerV3Facet{
			{
				Key: "ct", Label: "Entry type", PopHead: "Which kind of ledger entry",
				Source:  provLiveEntryTypes,
				Options: entryTypeFacetOptions(ordered, changes.Total),
			},
		},
	}
}

func entryTypeFacetOptions(ordered []ledgerV3EntryTypeRow, total int64) []vmv2.LedgerV3FacetOption {
	opts := []vmv2.LedgerV3FacetOption{{Value: "", Label: "Any", Count: formatThousands(total)}}
	for _, t := range ordered {
		opts = append(opts, vmv2.LedgerV3FacetOption{
			Value: t.name,
			Label: entryTypeLabel(t.name),
			Count: formatThousands(t.counts.Total),
		})
	}
	return opts
}

// changeBreakdown lists only the non-zero movements, so a type that was purely
// updated does not read as one that was also created and deleted zero times.
func changeBreakdown(c gateway.LedgerChangeTypeCounts) string {
	parts := make([]string, 0, 4)
	for _, p := range []struct {
		label string
		n     int64
	}{
		{"created", c.Created},
		{"updated", c.Updated},
		{"deleted", c.Deleted},
		{"restored", c.Restored},
	} {
		if p.n > 0 {
			parts = append(parts, fmt.Sprintf("%s %s", formatThousands(p.n), p.label))
		}
	}
	if len(parts) == 0 {
		return "no movement recorded"
	}
	return strings.Join(parts, " · ")
}

func shortAccount(a string) string {
	if len(a) <= 12 {
		return a
	}
	return a[:4] + "…" + a[len(a)-4:]
}

func shortContract(c string) string {
	if c == "" {
		return "unknown contract"
	}
	if len(c) <= 14 {
		return c
	}
	return c[:6] + "…" + c[len(c)-4:]
}

// formatStroopAmount renders a raw stroop amount. Assets carry seven decimal
// places on Stellar regardless of issuer.
func formatStroopAmount(raw string) string {
	var v int64
	if _, err := fmt.Sscanf(raw, "%d", &v); err != nil {
		return raw
	}
	return fmt.Sprintf("%.7f", float64(v)/1e7)
}

func humanOperationName(typeName string) string {
	if typeName == "" {
		return "operation"
	}
	return strings.ToLower(strings.ReplaceAll(typeName, "_", " "))
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

func pluralY(n int64) string {
	if n == 1 {
		return "y"
	}
	return "ies"
}
