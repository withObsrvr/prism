package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/withObsrvr/prism/internal/gateway"
	vmv2 "github.com/withObsrvr/prism/internal/templates/v2/viewmodel"
)

// Live provenance for the capacity meters, distinct from the mock constants:
// these describe values that were actually served for the ledger on screen.
var (
	provLiveFootprintEntries = vmv2.Provenance{
		Kind:   vmv2.ProvenanceServed,
		Origin: "GET /silver/ledgers/{seq}/soroban ÷ GET /silver/soroban/config?ledger={seq}",
		Note:   "Declared footprint entries, which is what the protocol charges capacity against. Counting observed changes instead undercounts substantially.",
	}
	provLiveCPU = vmv2.Provenance{
		Kind:   vmv2.ProvenanceServed,
		Origin: "GET /silver/ledgers/{seq}/soroban .total_cpu_insns ÷ .instructions.ledger_max",
		Note:   "Both halves served; the cap is resolved as of this ledger, not today.",
	}
	// A meter with no denominator is a gap, not a zero. Rendering a percentage
	// against a missing cap would invent a number.
	provMissingCap = vmv2.Provenance{
		Kind:   vmv2.ProvenanceGap,
		Origin: "numerator served, denominator absent",
		Note:   "The config setting carrying this limit has not been recorded for the network, so no honest percentage exists.",
	}
)

var provLiveLedgerStats = vmv2.Provenance{
	Kind:   vmv2.ProvenanceServed,
	Origin: "GET /silver/ledgers/{seq}/full .ledger",
	Note:   "Ledger header and counts as recorded in the close meta.",
}

// overlayLedgerV3Header replaces the header, standing strip and rail facts
// with the ledger's own record.
//
// It runs before the other overlays because they refine what it establishes:
// the capacity overlay rewrites the headline once it knows which limit bound,
// and would otherwise be overwritten by the counts set here.
func (h *Handlers) overlayLedgerV3Header(ctx context.Context, network string, sequence int64, data *vmv2.LedgerDetailV3Data) {
	if h.Gateway == nil {
		return
	}

	full, err := h.Gateway.GetSilverLedgerFull(ctx, network, sequence)
	if err != nil || full == nil {
		if h.Logger != nil && err != nil {
			h.Logger.Debug("ledger v3: ledger full unavailable", "sequence", sequence, "error", err)
		}
		return
	}
	l := full.Ledger

	txCount := l.TransactionCount
	if txCount == 0 {
		txCount = l.SuccessfulTxCount + l.FailedTxCount
	}

	data.Header.Hash = l.LedgerHash
	if l.ProtocolVersion > 0 {
		data.Header.ProtocolVersion = strconv.Itoa(l.ProtocolVersion)
	}
	if closed, err := time.Parse(time.RFC3339, l.ClosedAt); err == nil {
		data.Header.ClosedAt = closed.Format("02 Jan 2006, 15:04:05")
		data.Header.ClosedRelative = humanizeSince(closed)
	}
	data.Header.TxTabCount = formatThousands(int64(txCount))
	data.Header.Kicker = fmt.Sprintf("%s transactions · closed in %s",
		formatThousands(int64(txCount)), data.Header.CloseTime)

	if len(data.Header.Badges) >= 3 {
		data.Header.Badges[2] = vmv2.LedgerV3Badge{
			Label: fmt.Sprintf("%d failed", l.FailedTxCount),
		}
	}

	for i, s := range data.Standing {
		if s.Key != "Failures" {
			continue
		}
		data.Standing[i].Value = fmt.Sprintf("%d of %d", l.FailedTxCount, txCount)
		data.Standing[i].Source = provLiveLedgerStats
		if txCount > 0 {
			data.Standing[i].Detail = fmt.Sprintf("%.1f%% of this ledger's transactions",
				float64(l.FailedTxCount)*100/float64(txCount))
		}
		data.Standing[i].Dot = "g"
		if l.FailedTxCount > 0 {
			data.Standing[i].Dot = "a"
		}
	}

	applyLedgerV3RailContents(data, l, txCount, full.Soroban)
	applyLedgerV3LedeContents(data, l, txCount, full.Soroban)
	applyLedgerV3Ticks(data, network, full.Transactions)
}

// applyLedgerV3Ticks rebuilds the apply-order strip from the ledger's own
// transactions.
//
// Bar height is operation count, scaled to the tallest bar in this ledger
// rather than to a fixed ceiling: the strip is there to show the shape of one
// ledger, and a fixed scale flattens every quiet ledger into a uniform row.
//
// The strip is left as it was when the response carried no transactions,
// because an empty strip and an unfetched one look identical.
func applyLedgerV3Ticks(data *vmv2.LedgerDetailV3Data, network string, txs []gateway.Transaction) {
	if len(txs) == 0 {
		return
	}

	maxOps := 1
	for _, tx := range txs {
		if tx.OperationCount > maxOps {
			maxOps = tx.OperationCount
		}
	}

	ticks := make([]vmv2.LedgerV3Tick, 0, len(txs))
	for i, tx := range txs {
		height := 20 + (tx.OperationCount * 80 / maxOps)

		kind := "calls"
		status := "Succeeded"
		if !tx.Successful {
			kind = "fail"
			status = "Failed"
		}

		opLabel := "operation"
		if tx.OperationCount != 1 {
			opLabel = "operations"
		}

		ticks = append(ticks, vmv2.LedgerV3Tick{
			Position:  i + 1,
			Kind:      kind,
			HeightPct: height,
			Failed:    !tx.Successful,
			OpCount:   tx.OperationCount,
			TipTitle:  fmt.Sprintf("Position %d of %d", i+1, len(txs)),
			TipDetail: fmt.Sprintf("%d %s · %s", tx.OperationCount, opLabel, shortHash(tx.TransactionHash)),
			TipStatus: status,
			Href:      ledgerV3TxHref(tx.TransactionHash, network),
			AriaLabel: fmt.Sprintf("Transaction at position %d, %d %s, %s",
				i+1, tx.OperationCount, opLabel, strings.ToLower(status)),
		})
	}
	data.Strip.Ticks = ticks

	// The legend counts a breakdown by operation family that this response
	// cannot supply, so it is cleared rather than left describing a different
	// ledger's mix.
	data.Strip.Legend = nil

	sources := make(map[string]struct{}, len(txs))
	for _, tx := range txs {
		if tx.SourceAccount != "" {
			sources[tx.SourceAccount] = struct{}{}
		}
	}
	for i, stat := range data.Strip.Foot {
		if stat.Label != "Distinct source accounts" {
			continue
		}
		data.Strip.Foot[i].Value = formatThousands(int64(len(sources)))
		data.Strip.Foot[i].Source = vmv2.Provenance{
			Kind:   vmv2.ProvenanceDerived,
			Origin: "count of distinct source_account across GET /silver/ledgers/{seq}/full .transactions",
		}
	}
}

func shortHash(h string) string {
	if len(h) <= 12 {
		return h
	}
	return h[:8] + "…" + h[len(h)-4:]
}

// applyLedgerV3LedeContents rewrites the second opening paragraph, which
// otherwise reports another ledger's Soroban share and failure count.
//
// The mock's version drew a conclusion from the failures — that they clustered
// on two protocols. That reading needs the per-transaction result codes, so it
// is dropped rather than restated over numbers that cannot support it.
func applyLedgerV3LedeContents(data *vmv2.LedgerDetailV3Data, l gateway.Ledger, txCount int, soroban *gateway.LedgerSoroban) {
	if len(data.Lede) < 2 || txCount == 0 {
		return
	}

	sorobanClause := ""
	if soroban != nil {
		sorobanClause = fmt.Sprintf("Soroban accounted for %d%% of the transactions.<a class=\"pc-cite\" href=\"#n3\">3</a> ",
			int(soroban.SorobanTxCount*100/int64(txCount)))
	}

	switch l.FailedTxCount {
	case 0:
		data.Lede[1] = sorobanClause + `Every transaction in this ledger succeeded, so there is no failure pattern to read into.<a class="pc-cite" href="#n4">4</a>`
	case 1:
		data.Lede[1] = sorobanClause + `One transaction failed. A single failure is an event, not a pattern — the transactions tab carries its result code.<a class="pc-cite" href="#n4">4</a>`
	default:
		data.Lede[1] = fmt.Sprintf(
			`%s<b>%d transactions failed</b> out of %d. Whether that points at one protocol having a bad five seconds or at the network is a question the result codes answer, and they sit in the transactions tab.<a class="pc-cite" href="#n4">4</a>`,
			sorobanClause, l.FailedTxCount, txCount)
	}
}

// applyLedgerV3RailContents restates the rail's header and contents blocks.
func applyLedgerV3RailContents(data *vmv2.LedgerDetailV3Data, l gateway.Ledger, txCount int, soroban *gateway.LedgerSoroban) {
	for i, group := range data.Rail.Groups {
		for j, row := range group.Rows {
			switch {
			case group.Heading == "Header" && row.Label == "Protocol" && l.ProtocolVersion > 0:
				data.Rail.Groups[i].Rows[j].Value = strconv.Itoa(l.ProtocolVersion)
			case group.Heading == "Contents" && row.Label == "Transactions":
				data.Rail.Groups[i].Rows[j].Value = formatThousands(int64(txCount))
			case group.Heading == "Contents" && row.Label == "Operations":
				data.Rail.Groups[i].Rows[j].Value = formatThousands(int64(l.OperationCount))
			case group.Heading == "Contents" && row.Label == "Failed":
				data.Rail.Groups[i].Rows[j].Value = formatThousands(int64(l.FailedTxCount))
			case group.Heading == "Contents" && row.Label == "Soroban share":
				if soroban != nil && txCount > 0 {
					data.Rail.Groups[i].Rows[j].Value = fmt.Sprintf("%d%%",
						int(soroban.SorobanTxCount*100/int64(txCount)))
				}
			}
		}
	}

	for i, stat := range data.Strip.Foot {
		switch stat.Label {
		case "Transactions":
			data.Strip.Foot[i].Value = formatThousands(int64(txCount))
			data.Strip.Foot[i].Source = provLiveLedgerStats
		case "Operations":
			data.Strip.Foot[i].Value = formatThousands(int64(l.OperationCount))
			data.Strip.Foot[i].Source = provLiveLedgerStats
		case "Failed":
			data.Strip.Foot[i].Value = formatThousands(int64(l.FailedTxCount))
			data.Strip.Foot[i].Source = provLiveLedgerStats
		case "Contracts invoked":
			if soroban != nil {
				data.Strip.Foot[i].Value = formatThousands(soroban.UniqueContracts)
				data.Strip.Foot[i].Source = provLiveLedgerStats
			}
		}
	}
}

// humanizeSince renders an age the way the rest of the page does.
func humanizeSince(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%d sec ago", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%d min ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%d hr ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%d days ago", int(d.Hours()/24))
	}
}

// overlayLedgerV3Capacity replaces the mock capacity meters with values served
// for this ledger.
//
// The caps are fetched as of the ledger rather than as of now: capacity limits
// change on protocol upgrade, and dividing a historical ledger by today's caps
// produces a percentage that looks entirely reasonable and is wrong.
//
// Anything that cannot be served is left as it was rather than being filled
// with a plausible number — the page distinguishes served from derived from
// absent, and that distinction is the point of it.
func (h *Handlers) overlayLedgerV3Capacity(ctx context.Context, network string, sequence int64, data *vmv2.LedgerDetailV3Data) {
	if h.Gateway == nil {
		return
	}

	usage, err := h.Gateway.GetLedgerSoroban(ctx, network, sequence)
	if err != nil {
		var apiErr *gateway.APIError
		if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusNotFound {
			if h.Logger != nil {
				h.Logger.Debug("ledger v3: soroban usage unavailable", "sequence", sequence, "error", err)
			}
			return
		}

		// This endpoint reads bronze, which is pruned once silver has consumed
		// it, so a 404 has two very different meanings: a ledger that carried
		// no transactions used none of its capacity, while a ledger whose
		// transactions have simply aged out of bronze used an amount nobody
		// can now state. Only the ledger's own transaction count tells them
		// apart, and rendering the second as zero would be inventing a
		// measurement.
		if h.ledgerHadTransactions(ctx, network, sequence) {
			h.markLedgerV3CapacityUnavailable(data)
			return
		}
		usage = &gateway.LedgerSoroban{LedgerSequence: sequence}
	}
	if usage == nil {
		return
	}

	cfg, err := h.Gateway.GetSorobanConfigAtLedger(ctx, network, sequence)
	if err != nil || cfg == nil {
		if h.Logger != nil && err != nil {
			h.Logger.Debug("ledger v3: soroban config unavailable", "sequence", sequence, "error", err)
		}
		return
	}

	// Footprint entries and CPU come from different columns with different
	// histories: bronze cold carries the instruction counts but not the
	// footprint ones, so an older ledger can measure its compute while its
	// writes and reads remain unknown. Reporting the unknown ones as zero
	// would put them at the bottom of the meter and imply the ledger declared
	// nothing.
	writesMeter := capacityMeter("Ledger writes", usage.TotalWriteEntries, cfg.LedgerLimits.MaxWriteEntries,
		"entries",
		"Every contract that stores something competes for these.",
		provLiveFootprintEntries)
	readsMeter := capacityMeter("Ledger reads", usage.TotalReadEntries, cfg.LedgerLimits.MaxReadEntries,
		"entries",
		"Reads are cheap and plentiful, and rarely the binding constraint.",
		provLiveFootprintEntries)
	if !usage.FootprintEntriesAvailable {
		writesMeter = unmeasuredMeter("Ledger writes")
		readsMeter = unmeasuredMeter("Ledger reads")
	}

	meters := []vmv2.LedgerV3Meter{
		writesMeter,
		capacityMeter("CPU instructions", usage.TotalCPUInsns, cfg.Instructions.LedgerMax,
			"instructions",
			"Soroban compute across every invocation in this ledger.",
			provLiveCPU),
		readsMeter,
	}

	// The binding constraint is whichever meter is closest to its cap, and only
	// when one was actually computable.
	binding, bindingPct := -1, -1
	for i, m := range meters {
		if m.Source.Kind != vmv2.ProvenanceGap && m.Pct > bindingPct {
			binding, bindingPct = i, m.Pct
		}
	}
	if binding >= 0 {
		meters[binding].Binding = true
	}

	// The transaction-size meter is carried over untouched: its numerator is
	// served but the only published cap is Soroban-scoped while the meter sums
	// every envelope. That is a scoping decision, not missing data.
	for _, m := range data.Capacity.Meters {
		if m.Name == "Transaction size" {
			m.Used = fmt.Sprintf("%s KB", formatKB(usage.TotalEnvelopeBytes))
			m.Binding = false
			meters = append(meters, m)
			break
		}
	}

	data.Capacity.Meters = meters
	data.Capacity.Note = fmt.Sprintf(
		`<b>What this means for anyone deploying here.</b> A contract that writes many entries competes for the scarce resource; one that computes heavily does not. In this ledger the largest single transaction declared <b>%d write entries</b> of the %s available to the whole ledger.`,
		usage.MaxWriteEntriesSingleTx, formatCap(cfg.LedgerLimits.MaxWriteEntries))

	if binding >= 0 {
		applyLedgerV3BindingNarrative(data, meters[binding], usage, cfg)
	}
}

// applyLedgerV3BindingNarrative brings the headline, badge, standing strip and
// rail into line with the meters.
//
// Without this the page states a conclusion its own numbers contradict — a
// headline reading "ran out of writes" above a writes meter at 3%. A page whose
// whole argument is that it shows you where each number came from cannot
// disagree with itself about the number.
func applyLedgerV3BindingNarrative(
	data *vmv2.LedgerDetailV3Data,
	binding vmv2.LedgerV3Meter,
	usage *gateway.LedgerSoroban,
	cfg *gateway.SorobanConfig,
) {
	resource := bindingResourceWord(binding.Name)

	// Under roughly half capacity nothing is meaningfully scarce, and calling
	// something the binding constraint would overstate what happened.
	if binding.Pct >= 50 {
		data.Header.HeadlineLead = "This ledger ran out of"
		data.Header.HeadlineEmphasis = resource
		data.Header.HeadlineTrail = "before anything else"
	} else {
		data.Header.HeadlineLead = "This ledger closed with room to spare on"
		data.Header.HeadlineEmphasis = "every limit"
		data.Header.HeadlineTrail = "it has"
	}
	data.Header.HeadlineSource = binding.Source

	tone := ""
	if binding.Pct >= 80 {
		tone = "warn"
	}
	if len(data.Header.Badges) > 0 {
		data.Header.Badges[0] = vmv2.LedgerV3Badge{
			Label: fmt.Sprintf("%s %d%% full", capitalise(resource), binding.Pct),
			Tone:  tone,
		}
	}

	dot := "g"
	if binding.Pct >= 80 {
		dot = "a"
	}
	for i, s := range data.Standing {
		if s.Key != "Binding limit" {
			continue
		}
		data.Standing[i].Value = fmt.Sprintf("%s, %d%%", capitalise(resource), binding.Pct)
		data.Standing[i].Detail = fmt.Sprintf("%s of %s",
			binding.Used, strings.TrimSuffix(binding.Cap, " cap"))
		data.Standing[i].Dot = dot
		data.Standing[i].Source = binding.Source
	}

	applyLedgerV3RailCapacity(data, usage, cfg)
	applyLedgerV3LedeCapacity(data, binding, usage, cfg)

	if binding.Pct >= 50 {
		data.Rail.Subtitle = fmt.Sprintf("%s-limited", capitalise(resource))
	} else {
		data.Rail.Subtitle = "Within every limit"
	}
}

// applyLedgerV3FeeTOC keeps the contents entry in step with the fee multiple it
// names, so the sidebar does not advertise a figure the section never shows.
func applyLedgerV3FeeTOC(data *vmv2.LedgerDetailV3Data, multiple string) {
	for i, entry := range data.Rail.TOC {
		if strings.HasPrefix(entry.Label, "Why fees were") {
			data.Rail.TOC[i].Label = fmt.Sprintf("Why fees were %s base", multiple)
		}
	}
}

// applyLedgerV3LedeCapacity rewrites the opening paragraph's capacity clause.
// It is the most prominent claim on the page, so leaving it describing a
// different ledger is worse than having no lede at all.
func applyLedgerV3LedeCapacity(
	data *vmv2.LedgerDetailV3Data,
	binding vmv2.LedgerV3Meter,
	usage *gateway.LedgerSoroban,
	cfg *gateway.SorobanConfig,
) {
	if len(data.Lede) == 0 {
		return
	}

	writePct := pctOf(usage.TotalWriteEntries, cfg.LedgerLimits.MaxWriteEntries)
	cpuPct := pctOf(usage.TotalCPUInsns, cfg.Instructions.LedgerMax)
	readPct := pctOf(usage.TotalReadEntries, cfg.LedgerLimits.MaxReadEntries)

	if binding.Pct >= 50 {
		data.Lede[0] = fmt.Sprintf(
			`<span class="pc-drop">A</span> ledger is not a container of equal parts. Each limit is counted separately, and by the time this one closed it had used <b>%d%% of its write capacity</b> while sitting at %d%% of CPU and %d%% of reads.<a class="pc-cite" href="#n1">1</a> %s was the binding constraint, so the fee market priced contention for a resource most transactions here barely used.<a class="pc-cite" href="#n2">2</a>`,
			writePct, cpuPct, readPct, capitalise(bindingResourceWord(binding.Name)))
		return
	}

	data.Lede[0] = fmt.Sprintf(
		`<span class="pc-drop">A</span> ledger is not a container of equal parts. Each limit is counted separately, and this one closed well inside all of them — <b>%d%% of write capacity</b>, %d%% of CPU, %d%% of reads.<a class="pc-cite" href="#n1">1</a> Nothing was scarce, so nothing here was priced by contention. The interesting question for a ledger like this is not what filled up but what it cost anyway.<a class="pc-cite" href="#n2">2</a>`,
		writePct, cpuPct, readPct)
}

func pctOf(used, limit int64) int {
	if limit <= 0 {
		return 0
	}
	return int((used * 100) / limit)
}

var provLiveEntryChanges = vmv2.Provenance{
	Kind:   vmv2.ProvenanceServed,
	Origin: "GET /silver/ledgers/{seq}/changes",
	Note:   "Read from the ledger change stream. Created, updated, deleted and restored sum to the total; eviction is counted apart from it because a sweep is not a transaction's doing.",
}

// overlayLedgerV3Changes replaces the state-change panel with served counts.
//
// A ledger whose statistics were never captured is left alone rather than
// shown as zeros: the endpoint reports that case explicitly, and a row of
// zeros would otherwise read as "nothing changed here".
func (h *Handlers) overlayLedgerV3Changes(ctx context.Context, network string, sequence int64, data *vmv2.LedgerDetailV3Data) *gateway.LedgerChanges {
	if h.Gateway == nil {
		return nil
	}

	changes, err := h.Gateway.GetLedgerChanges(ctx, network, sequence)
	if err != nil || changes == nil || !changes.Available {
		if h.Logger != nil && err != nil {
			h.Logger.Debug("ledger v3: ledger changes unavailable", "sequence", sequence, "error", err)
		}
		return changes
	}

	data.Changes.Total = formatThousands(changes.Total)
	data.Changes.Source = provLiveEntryChanges

	cells := []vmv2.LedgerV3ChangeCell{
		{Label: "Created", Count: formatThousands(changes.Created), Note: "Entries that did not exist before this ledger.", Source: provLiveEntryChanges},
		{Label: "Updated", Count: formatThousands(changes.Updated), Note: "Existing entries whose contents changed.", Source: provLiveEntryChanges},
		{Label: "Deleted", Count: formatThousands(changes.Deleted), Note: "Entries removed by a transaction.", Source: provLiveEntryChanges},
		{Label: "Restored", Count: formatThousands(changes.Restored), Note: "Archived entries brought back. Under CAP-0062 this happens inside a contract call, with no restore operation.", Source: provLiveEntryChanges},
	}
	tone := ""
	if changes.Evicted > 0 {
		tone = "warn"
	}
	cells = append(cells, vmv2.LedgerV3ChangeCell{
		Label: "Archived",
		Count: formatThousands(changes.Evicted),
		Note:  "Swept by the protocol for expiry. Not part of the total above.",
		Tone:  tone, Source: provLiveEntryChanges,
	})
	data.Changes.Cells = cells

	if types := entryTypeRows(changes.ByType); len(types) > 0 {
		data.Changes.EntryTypes = types
	}

	data.Changes.Note = fmt.Sprintf(
		`<b>%s archived against %s restored.</b> The first four cells are the change stream and sum to %s. Archival is not in that total — eviction is a ledger-level sweep, not a transaction's doing, which is why it can dwarf everything else. On most single ledgers both run at zero.`,
		formatThousands(changes.Evicted), formatThousands(changes.Restored), formatThousands(changes.Total))

	data.Header.StateTabCount = formatThousands(changes.Total)

	for i, group := range data.Rail.Groups {
		if group.Heading != "State" {
			continue
		}
		for j, row := range group.Rows {
			switch row.Label {
			case "Entry changes":
				data.Rail.Groups[i].Rows[j].Value = formatThousands(changes.Total)
			case "Archived":
				data.Rail.Groups[i].Rows[j].Value = formatThousands(changes.Evicted)
			case "Restored":
				data.Rail.Groups[i].Rows[j].Value = formatThousands(changes.Restored)
			}
		}
	}

	return changes
}

// entryTypeRows orders the per-type breakdown by size, because which entry
// type dominates is the point of the panel and it varies by ledger.
func entryTypeRows(byType map[string]gateway.LedgerChangeTypeCounts) []vmv2.LedgerV3EntryType {
	if len(byType) == 0 {
		return nil
	}

	type row struct {
		label string
		count int64
	}
	rows := make([]row, 0, len(byType))
	for name, counts := range byType {
		if counts.Total == 0 {
			continue
		}
		rows = append(rows, row{label: entryTypeLabel(name), count: counts.Total})
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].count != rows[j].count {
			return rows[i].count > rows[j].count
		}
		return rows[i].label < rows[j].label
	})

	out := make([]vmv2.LedgerV3EntryType, 0, len(rows))
	for _, r := range rows {
		out = append(out, vmv2.LedgerV3EntryType{Label: r.label, Count: formatThousands(r.count)})
	}
	return out
}

func entryTypeLabel(name string) string {
	switch name {
	case "account":
		return "Account"
	case "trustline":
		return "Trustline"
	case "contract_data":
		return "Contract data"
	case "contract_code":
		return "Contract code"
	case "offer":
		return "Offer"
	case "ttl":
		return "TTL"
	case "liquidity_pool":
		return "Liquidity pool"
	case "data":
		return "Account data"
	case "claimable_balance":
		return "Claimable balance"
	default:
		return capitalise(strings.ReplaceAll(name, "_", " "))
	}
}

var provLiveFees = vmv2.Provenance{
	Kind:   vmv2.ProvenanceServed,
	Origin: "GET /silver/ledgers/{seq}/fees",
	Note:   "Charged fees and the highest bid, which are different numbers: every included transaction clears at the same price regardless of what it offered.",
}

// overlayLedgerV3Fees replaces the fee panel with served figures.
//
// The clearing price is the median charged fee rather than the mean: the mean
// is dragged upward by a few large bids and would overstate what a typical
// transaction actually paid.
func (h *Handlers) overlayLedgerV3Fees(ctx context.Context, network string, sequence int64, data *vmv2.LedgerDetailV3Data) {
	if h.Gateway == nil {
		return
	}

	fees, err := h.Gateway.GetLedgerFees(ctx, network, sequence)
	if err != nil || fees == nil || fees.TxCount == 0 {
		if h.Logger != nil && err != nil {
			h.Logger.Debug("ledger v3: ledger fees unavailable", "sequence", sequence, "error", err)
		}
		return
	}

	const baseFeeStroops = 100
	clearing := fees.MedianFee

	data.Fees.ClearingValue = formatStroopsXLM(clearing)
	data.Fees.ClearingUnit = "XLM"
	data.Fees.BaseFee = fmt.Sprintf("%s XLM", formatStroopsXLM(baseFeeStroops))
	data.Fees.TotalCollected = fmt.Sprintf("%s XLM", formatStroopsXLM(fees.TotalFees))
	data.Fees.Source = provLiveFees

	multiple := float64(clearing) / float64(baseFeeStroops)
	data.Fees.Multiple = formatMultiple(multiple)

	if multiple <= 1.0 {
		data.Fees.ClearingLabel = "Clearing fee — at base, nothing contested"
	} else {
		data.Fees.ClearingLabel = "Clearing fee, charged to all"
	}

	// The gap between the highest bid and the clearing price is the whole
	// point: it is money the winner offered and never had to pay.
	if fees.MaxBid > clearing && clearing > 0 {
		data.Fees.HighestBidNote = fmt.Sprintf(
			`Against a base fee of <b>%s XLM</b>. Everyone paid the same %s, whether they bid it or bid far more — <b>the highest bid in this ledger was %s the clearing price and was charged the clearing price</b>.`,
			formatStroopsXLM(baseFeeStroops), data.Fees.Multiple,
			formatMultiple(float64(fees.MaxBid)/float64(clearing)))
	} else {
		data.Fees.HighestBidNote = fmt.Sprintf(
			`Against a base fee of <b>%s XLM</b>. Nothing in this ledger bid meaningfully above the clearing price, which is what an uncontested ledger looks like.`,
			formatStroopsXLM(baseFeeStroops))
	}

	if len(data.Header.Badges) >= 2 {
		badgeTone := ""
		if multiple >= 2 {
			badgeTone = "warn"
		}
		data.Header.Badges[1] = vmv2.LedgerV3Badge{
			Label: fmt.Sprintf("Fees %s base", data.Fees.Multiple),
			Tone:  badgeTone,
		}
	}

	for i, s := range data.Standing {
		if s.Key != "Fee level" {
			continue
		}
		data.Standing[i].Value = fmt.Sprintf("%s base", data.Fees.Multiple)
		data.Standing[i].Source = provLiveFees
		if multiple > 1.5 {
			data.Standing[i].Detail = "Surge pricing was active"
			data.Standing[i].Dot = "a"
		} else {
			data.Standing[i].Detail = "At or near base — nothing contested"
			data.Standing[i].Dot = "g"
		}
	}

	for i, group := range data.Rail.Groups {
		if group.Heading != "Fees" {
			continue
		}
		for j, row := range group.Rows {
			switch row.Label {
			case "Clearing":
				data.Rail.Groups[i].Rows[j].Value = fmt.Sprintf("%s XLM", formatStroopsXLM(clearing))
			case "Base":
				data.Rail.Groups[i].Rows[j].Value = fmt.Sprintf("%s XLM", formatStroopsXLM(baseFeeStroops))
			case "Multiple":
				data.Rail.Groups[i].Rows[j].Value = data.Fees.Multiple
			case "Total collected":
				data.Rail.Groups[i].Rows[j].Value = fmt.Sprintf("%s XLM", formatStroopsXLM(fees.TotalFees))
			}
		}
	}

	applyLedgerV3FeeTOC(data, data.Fees.Multiple)
}

func formatStroopsXLM(stroops int64) string {
	return strconv.FormatFloat(float64(stroops)/1e7, 'f', 5, 64)
}

func formatMultiple(m float64) string {
	if m >= 10 {
		return fmt.Sprintf("%.0f×", m)
	}
	return fmt.Sprintf("%.1f×", m)
}

// applyLedgerV3RailCapacity restates the rail's capacity block from the same
// figures as the meters, so the summary and the detail cannot disagree.
func applyLedgerV3RailCapacity(data *vmv2.LedgerDetailV3Data, usage *gateway.LedgerSoroban, cfg *gateway.SorobanConfig) {
	for i, group := range data.Rail.Groups {
		if group.Heading != "Capacity" {
			continue
		}
		for j, row := range group.Rows {
			switch row.Label {
			case "Writes":
				// Writes and reads come from the footprint columns, which an
				// older schema does not carry. Showing "0 / 1000" there would
				// contradict the meter beside it, which says the figure was
				// never recorded.
				if !usage.FootprintEntriesAvailable {
					data.Rail.Groups[i].Rows[j].Value = "—"
					data.Rail.Groups[i].Rows[j].IsGap = true
					continue
				}
				data.Rail.Groups[i].Rows[j].Value = railRatio(usage.TotalWriteEntries, cfg.LedgerLimits.MaxWriteEntries)
			case "CPU":
				data.Rail.Groups[i].Rows[j].Value = railPct(usage.TotalCPUInsns, cfg.Instructions.LedgerMax)
			case "Reads":
				if !usage.FootprintEntriesAvailable {
					data.Rail.Groups[i].Rows[j].Value = "—"
					data.Rail.Groups[i].Rows[j].IsGap = true
					continue
				}
				data.Rail.Groups[i].Rows[j].Value = railPct(usage.TotalReadEntries, cfg.LedgerLimits.MaxReadEntries)
			}
		}
	}
}

func railRatio(used, limit int64) string {
	if limit <= 0 {
		return "—"
	}
	return fmt.Sprintf("%d / %d", used, limit)
}

func railPct(used, limit int64) string {
	if limit <= 0 {
		return "—"
	}
	return fmt.Sprintf("%d%%", (used*100)/limit)
}

// bindingResourceWord reduces a meter name to the noun the headline uses.
func bindingResourceWord(meterName string) string {
	switch meterName {
	case "Ledger writes":
		return "writes"
	case "Ledger reads":
		return "reads"
	case "CPU instructions":
		return "CPU"
	case "Transaction size":
		return "size"
	default:
		return strings.ToLower(meterName)
	}
}

func capitalise(s string) string {
	if s == "" || s == "CPU" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// ledgerHadTransactions reports whether the ledger carried any transactions at
// all, read from the durable serving record rather than from bronze.
func (h *Handlers) ledgerHadTransactions(ctx context.Context, network string, sequence int64) bool {
	full, err := h.Gateway.GetSilverLedgerFull(ctx, network, sequence)
	if err != nil || full == nil {
		// Unable to tell, so assume the ledger was not empty: reporting
		// "unavailable" is recoverable, reporting a false zero is not.
		return true
	}
	l := full.Ledger
	if l.TransactionCount > 0 {
		return true
	}
	return l.SuccessfulTxCount+l.FailedTxCount > 0
}

// unmeasuredMeter renders a limit whose usage the serving layer could not
// measure. It reports the gap rather than a percentage, because a bar sitting
// at zero says the ledger declared nothing, which is a different claim.
func unmeasuredMeter(name string) vmv2.LedgerV3Meter {
	return vmv2.LedgerV3Meter{
		Name:   name,
		Pct:    0,
		Used:   "not recorded",
		Cap:    "—",
		Note:   "Declared footprint entries are not held for this ledger, so its share of this limit cannot be stated.",
		Source: provFootprintNotInCold,
	}
}

var provFootprintNotInCold = vmv2.Provenance{
	Kind:   vmv2.ProvenanceGap,
	Origin: "GET /silver/ledgers/{seq}/soroban .footprint_entries_available = false",
	Note:   "This ledger is answered from bronze cold, whose schema predates the declared-footprint columns. The usage is unknown, not zero.",
}

var provCapacityPruned = vmv2.Provenance{
	Kind:   vmv2.ProvenanceGap,
	Origin: "GET /silver/ledgers/{seq}/soroban — no bronze rows for this ledger",
	Note:   "This ledger carried transactions, but the per-transaction resource rows have been pruned from bronze, so its capacity usage can no longer be measured. It is not zero; it is unknown.",
}

// markLedgerV3CapacityUnavailable states that the usage cannot be measured,
// rather than leaving the mock's numbers standing or printing zeros.
func (h *Handlers) markLedgerV3CapacityUnavailable(data *vmv2.LedgerDetailV3Data) {
	meters := make([]vmv2.LedgerV3Meter, 0, len(data.Capacity.Meters))
	for _, name := range []string{"Ledger writes", "CPU instructions", "Ledger reads"} {
		meters = append(meters, vmv2.LedgerV3Meter{
			Name:   name,
			Pct:    0,
			Used:   "not recorded",
			Cap:    "—",
			Note:   "Per-transaction resource rows for this ledger have aged out of bronze.",
			Source: provCapacityPruned,
		})
	}
	data.Capacity.Meters = meters
	data.Capacity.Note = `<b>Capacity cannot be stated for this ledger.</b> The per-transaction footprint rows it would be computed from have been pruned, and a ledger whose usage is unknown is not a ledger that used nothing. The limits themselves are unchanged; only the numerators are gone.`
	data.Rail.Subtitle = "Capacity not recorded"

	// Everything downstream that names a binding limit has to go quiet too.
	// Leaving the mock's "Writes 97% full" beside meters reading "not
	// recorded" states a fact the page has just said it cannot know.
	data.Header.HeadlineLead = "This ledger's"
	data.Header.HeadlineEmphasis = "capacity"
	data.Header.HeadlineTrail = "was not recorded"
	data.Header.HeadlineSource = provCapacityPruned

	if len(data.Header.Badges) > 0 {
		data.Header.Badges[0] = vmv2.LedgerV3Badge{Label: "Capacity not recorded"}
	}

	for i, s := range data.Standing {
		if s.Key != "Binding limit" {
			continue
		}
		data.Standing[i].Value = "Not recorded"
		data.Standing[i].Detail = "Footprint rows pruned from bronze"
		data.Standing[i].Dot = "none"
		data.Standing[i].Source = provCapacityPruned
	}

	for i, group := range data.Rail.Groups {
		if group.Heading != "Capacity" {
			continue
		}
		for j := range group.Rows {
			data.Rail.Groups[i].Rows[j].Value = "—"
			data.Rail.Groups[i].Rows[j].IsGap = true
		}
	}

	// The lede's capacity sentence has to go with it.
	if len(data.Lede) > 0 {
		data.Lede[0] = `<span class="pc-drop">A</span> ledger is not a container of equal parts — each limit is counted separately, and a ledger is full when any one of them is exhausted. <b>Which limit bound this one cannot be said.</b> The per-transaction footprint rows the calculation needs have been pruned, so the honest answer is that its capacity usage is unknown rather than low.<a class="pc-cite" href="#n1">1</a>`
	}
}

// capacityMeter renders one limit. A cap of zero means the limit was never
// recorded, which is indistinguishable from a real cap of zero — so it is
// reported as a gap rather than as a division.
func capacityMeter(name string, used, limit int64, unit, note string, served vmv2.Provenance) vmv2.LedgerV3Meter {
	if limit <= 0 {
		return vmv2.LedgerV3Meter{
			Name:   name,
			Pct:    0,
			Used:   fmt.Sprintf("%s %s", formatCount(used), unit),
			Cap:    "no cap recorded",
			Note:   note,
			Source: provMissingCap,
		}
	}

	pct := int((used * 100) / limit)
	if pct > 100 {
		pct = 100
	}
	return vmv2.LedgerV3Meter{
		Name:   name,
		Pct:    pct,
		Used:   fmt.Sprintf("%s %s", formatCount(used), unit),
		Cap:    fmt.Sprintf("%s cap", formatCap(limit)),
		Note:   note,
		Source: served,
	}
}

func formatCount(v int64) string {
	switch {
	case v >= 1_000_000_000:
		return fmt.Sprintf("%.1f B", float64(v)/1_000_000_000)
	case v >= 1_000_000:
		return fmt.Sprintf("%.1f M", float64(v)/1_000_000)
	case v >= 10_000:
		return fmt.Sprintf("%.1f K", float64(v)/1_000)
	default:
		return fmt.Sprintf("%d", v)
	}
}

func formatCap(v int64) string { return formatCount(v) }

func formatKB(v int64) string { return fmt.Sprintf("%.1f", float64(v)/1024) }
