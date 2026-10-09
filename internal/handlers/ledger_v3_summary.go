package handlers

import (
	"context"
	"errors"
	"math"
	"strings"
	"time"

	"github.com/withObsrvr/prism/internal/gateway"
	"github.com/withObsrvr/prism/internal/jev"
	"github.com/withObsrvr/prism/internal/summary"
)

type ledgerV3SummaryEvidence struct {
	Full         *gateway.LedgerFullResponse
	Transactions []gateway.Transaction
	Operations   []gateway.Operation
	Baselines    summary.LedgerBaselines
	Usage        *gateway.LedgerSoroban
	Config       *gateway.SorobanConfig
	Fees         *gateway.LedgerFees
	Changes      *gateway.LedgerChanges
}

func (h *Handlers) buildLedgerV3SummaryEnvelope(ctx context.Context, network string, sequence int64, full *gateway.LedgerFullResponse, txs []gateway.Transaction, ops []gateway.Operation, changes *gateway.LedgerChanges) (summary.LedgerSummaryEnvelope, error) {
	shadowIdentity := "disabled"
	if h.LedgerSelector != nil {
		shadowIdentity = h.LedgerSelector.LedgerSelectorIdentity()
	}
	cacheKey := summary.NewLedgerSummaryCacheKey(network, sequence, shadowIdentity)
	if cached, ok := h.LedgerSummaries.Get(cacheKey); ok {
		cached.SelectionTrace.CacheStatus = "hit"
		h.logLedgerSummarySelection(network, sequence, cached)
		return cached, nil
	}
	usage, _ := h.Gateway.GetLedgerSoroban(ctx, network, sequence)
	config, _ := h.Gateway.GetSorobanConfigAtLedger(ctx, network, sequence)
	fees, _ := h.Gateway.GetLedgerFees(ctx, network, sequence)
	baselines := h.loadLedgerV3Baselines(ctx, network, sequence)
	envelope, err := summary.BuildLedgerSummaryEnvelope(ledgerFactsFromGateway(ledgerV3SummaryEvidence{Full: full, Transactions: txs, Operations: ops, Baselines: baselines, Usage: usage, Config: config, Fees: fees, Changes: changes}))
	if err != nil {
		return summary.LedgerSummaryEnvelope{}, err
	}
	shadowStarted := time.Now()
	h.applyLedgerShadowSelection(ctx, &envelope)
	h.recordLedgerShadow(network, sequence, envelope, time.Since(shadowStarted))
	if h.LedgerSummaries != nil {
		envelope.SelectionTrace.CacheStatus = "miss"
		h.LedgerSummaries.Set(cacheKey, envelope)
	}
	h.logLedgerSummarySelection(network, sequence, envelope)
	return envelope, nil
}

func (h *Handlers) recordLedgerShadow(network string, sequence int64, envelope summary.LedgerSummaryEnvelope, latency time.Duration) {
	if h.LedgerShadowRecorder == nil || envelope.SelectionTrace.Shadow == nil || envelope.Selection.Lead == "" {
		return
	}
	shadow := envelope.SelectionTrace.Shadow
	shadowIdentity := "disabled"
	if h.LedgerSelector != nil {
		shadowIdentity = h.LedgerSelector.LedgerSelectorIdentity()
	}
	eligible := make([]string, 0, len(envelope.SelectionTrace.Eligible))
	for _, id := range envelope.SelectionTrace.Eligible {
		eligible = append(eligible, string(id))
	}
	record := jev.LedgerShadowRecord{
		RecordVersion: jev.LedgerShadowRecordVersion,
		Network:       network, LedgerSequence: sequence,
		Bands: summary.LedgerBandValues(envelope.State.Bands), Eligible: eligible,
		Deterministic: string(envelope.Selection.Lead), ShadowStatus: shadow.Status,
		ShadowLead: string(shadow.Lead), Probabilities: shadow.Probabilities,
		Confidence: shadow.Confidence, Agrees: shadow.Agrees, Reason: shadow.Reason,
		Model: shadow.Model, RegistryVersion: shadow.RegistryVersion,
		BandVersion: envelope.BandVersion, BaselineVersion: envelope.BaselineVersion,
		ThresholdVersion: envelope.ThresholdVersion, TemplateVersion: envelope.TemplateVersion,
		SelectorVersion: envelope.SelectionTrace.SelectorVersion,
		ShadowIdentity:  shadowIdentity,
		LatencyMS:       latency.Milliseconds(), InputTokens: shadow.InputTokens, OutputTokens: shadow.OutputTokens,
	}
	if err := h.LedgerShadowRecorder.RecordLedgerShadow(record); err != nil && h.Logger != nil {
		h.Logger.Warn("record ledger shadow evaluation", "network", network, "sequence", sequence, "error", err)
	}
}

func (h *Handlers) applyLedgerShadowSelection(ctx context.Context, envelope *summary.LedgerSummaryEnvelope) {
	if envelope == nil {
		return
	}
	shadow := &summary.ShadowSelectionTrace{Status: "disabled", Reason: "not_configured"}
	envelope.SelectionTrace.Shadow = shadow
	if h.LedgerSelector == nil {
		return
	}
	if len(envelope.Candidates) < 2 {
		shadow.Status, shadow.Reason = "skipped", "fewer_than_two_eligible_choices"
		return
	}
	request := jev.LedgerSelectionRequest{Bands: summary.LedgerBandValues(envelope.State.Bands), Candidates: make([]jev.LedgerSelectionCandidate, 0, len(envelope.Candidates))}
	for _, candidate := range envelope.Candidates {
		request.Candidates = append(request.Candidates, jev.LedgerSelectionCandidate{ID: string(candidate.ID), Description: candidate.Description})
	}
	selection, err := h.LedgerSelector.SelectLedgerSummary(ctx, request)
	if err != nil {
		shadow.Status, shadow.Reason = "error", shadowErrorReason(err)
		return
	}
	if !validLedgerShadowSelection(selection, envelope.Candidates) {
		shadow.Status, shadow.Reason = "rejected", "invalid_or_ineligible_response"
		return
	}
	shadow.Status = "succeeded"
	shadow.RegistryVersion = selection.RegistryVersion
	shadow.Model = selection.Model
	shadow.Lead = summary.InterpretationID(selection.Lead)
	shadow.Probabilities = selection.Probabilities
	shadow.Confidence = selection.Confidence
	shadow.Agrees = shadow.Lead == envelope.Selection.Lead
	shadow.InputTokens = selection.Usage.InputTokens
	shadow.OutputTokens = selection.Usage.OutputTokens
}

func validLedgerShadowSelection(selection jev.LedgerSelection, candidates []summary.Candidate) bool {
	if selection.RegistryVersion != jev.LedgerSelectorRegistryVersion || strings.TrimSpace(selection.Model) == "" || selection.Confidence < 0 || selection.Confidence > 1 || math.IsNaN(selection.Confidence) || math.IsInf(selection.Confidence, 0) || selection.Usage.InputTokens < 0 || selection.Usage.OutputTokens < 0 || len(selection.Probabilities) != len(candidates) {
		return false
	}
	eligible, total, leadFound := make(map[string]struct{}, len(candidates)), 0.0, false
	for _, candidate := range candidates {
		eligible[string(candidate.ID)] = struct{}{}
	}
	for id, probability := range selection.Probabilities {
		if _, ok := eligible[id]; !ok || probability < 0 || probability > 1 || math.IsNaN(probability) || math.IsInf(probability, 0) {
			return false
		}
		total += probability
	}
	if _, leadFound = eligible[selection.Lead]; !leadFound {
		return false
	}
	return math.Abs(total-1) <= .02
}

func shadowErrorReason(err error) string {
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return "timeout_or_cancellation"
	}
	return "request_or_validation_error"
}

func (h *Handlers) logLedgerSummarySelection(network string, sequence int64, envelope summary.LedgerSummaryEnvelope) {
	if h.Logger == nil {
		return
	}
	trace := envelope.SelectionTrace
	h.Logger.Info("ledger summary selected",
		"network", network,
		"sequence", sequence,
		"mode", trace.Mode,
		"selector_version", trace.SelectorVersion,
		"eligible_count", len(trace.Eligible),
		"deterministic_lead", trace.DeterministicLead,
		"applied_lead", trace.AppliedLead,
		"applied_source", trace.AppliedSource,
		"cache_status", trace.CacheStatus,
		"registry_version", envelope.RegistryVersion,
		"band_version", envelope.BandVersion,
		"baseline_version", envelope.BaselineVersion,
		"template_version", envelope.TemplateVersion,
		"shadow_status", shadowTraceValue(trace.Shadow, func(value *summary.ShadowSelectionTrace) any { return value.Status }),
		"shadow_lead", shadowTraceValue(trace.Shadow, func(value *summary.ShadowSelectionTrace) any { return value.Lead }),
		"shadow_confidence", shadowTraceValue(trace.Shadow, func(value *summary.ShadowSelectionTrace) any { return value.Confidence }),
		"shadow_agrees", shadowTraceValue(trace.Shadow, func(value *summary.ShadowSelectionTrace) any { return value.Agrees }),
		"shadow_reason", shadowTraceValue(trace.Shadow, func(value *summary.ShadowSelectionTrace) any { return value.Reason }),
	)
}

func shadowTraceValue(trace *summary.ShadowSelectionTrace, value func(*summary.ShadowSelectionTrace) any) any {
	if trace == nil {
		return nil
	}
	return value(trace)
}

func (h *Handlers) loadLedgerV3Baselines(ctx context.Context, network string, sequence int64) summary.LedgerBaselines {
	start := sequence - summary.LedgerBaselineWindow
	if start < 1 {
		start = 1
	}
	if sequence <= 1 {
		return summary.BuildLedgerBaselines(sequence, nil)
	}
	ledgers, err := h.Gateway.GetLedgers(ctx, network, start, sequence-1, summary.LedgerBaselineWindow, "asc")
	if err != nil {
		return summary.BuildLedgerBaselines(sequence, nil)
	}
	samples := make([]summary.LedgerBaselineSample, 0, len(ledgers))
	for _, ledger := range ledgers {
		total := ledger.TransactionCount
		if total == 0 {
			total = ledger.SuccessfulTxCount + ledger.FailedTxCount
		}
		samples = append(samples, summary.LedgerBaselineSample{Sequence: ledger.Sequence, Transactions: int64(total), Operations: int64(ledger.OperationCount)})
	}
	return summary.BuildLedgerBaselines(sequence, samples)
}

func ledgerFactsFromGateway(e ledgerV3SummaryEvidence) summary.LedgerFacts {
	if e.Full == nil {
		return summary.LedgerFacts{}
	}
	l := e.Full.Ledger
	txTotal := l.TransactionCount
	if txTotal == 0 {
		txTotal = l.SuccessfulTxCount + l.FailedTxCount
	}
	txStatus := summary.Available
	if len(e.Transactions) < txTotal {
		txStatus = summary.Partial
	}
	opStatus := summary.Available
	if len(e.Operations) < l.OperationCount {
		opStatus = summary.Partial
	}
	evidenceStatus := summary.Available
	if e.Full.Partial || txStatus != summary.Available || opStatus != summary.Available {
		evidenceStatus = summary.Partial
	}

	facts := summary.LedgerFacts{
		Sequence: l.Sequence, ProtocolVersion: l.ProtocolVersion,
		Evidence:     summary.Evidence{Status: evidenceStatus, Sources: []string{"GET /silver/ledgers/{seq}/full"}, CompleteThroughLedger: l.Sequence},
		Transactions: measured(int64(txTotal), txStatus), Operations: measured(int64(l.OperationCount), opStatus),
		Successful: measured(int64(l.SuccessfulTxCount), txStatus), Failed: measured(int64(l.FailedTxCount), txStatus),
		FailureCause: failureCauseFacts(e.Transactions, txStatus),
		Activity:     activityFacts(e.Baselines),
		Mix:          summary.LedgerCompositionFacts{SorobanTransactions: missing("Soroban transaction count was not served.")},
		Capacity:     unavailableCapacity(), Fees: unavailableFees(), Changes: unavailableChanges(),
	}

	sorobanTxCount := int64(0)
	if e.Usage != nil {
		sorobanTxCount = e.Usage.SorobanTxCount
	} else if e.Full.Soroban != nil {
		sorobanTxCount = e.Full.Soroban.SorobanTxCount
	}
	if e.Usage != nil || e.Full.Soroban != nil {
		facts.Mix.SorobanTransactions = measured(sorobanTxCount, txStatus)
	}
	facts.Capacity = capacityFacts(e.Usage, e.Config, sorobanTxCount)
	facts.Fees = feeFacts(l, e.Fees, e.Usage, sorobanTxCount)
	facts.Changes = changeFacts(e.Changes)
	if e.Baselines.Version == summary.LedgerBaselineVersion {
		facts.Changes.Baseline = e.Baselines.StateChanges
	}
	if e.Baselines.Transactions.Status == summary.Available || e.Baselines.Operations.Status == summary.Available {
		facts.Evidence.Sources = append(facts.Evidence.Sources, "GET /bronze/ledgers; "+summary.LedgerBaselineVersion)
	}
	return facts
}

func activityFacts(baselines summary.LedgerBaselines) summary.LedgerActivityFacts {
	if baselines.Version != summary.LedgerBaselineVersion {
		return summary.LedgerActivityFacts{TransactionBaseline: missing("No versioned ledger activity baseline was supplied."), OperationBaseline: missing("No versioned ledger activity baseline was supplied.")}
	}
	return summary.LedgerActivityFacts{TransactionBaseline: baselines.Transactions, OperationBaseline: baselines.Operations}
}

func measured(value int64, status summary.Availability) summary.Int64Evidence {
	return summary.Int64Evidence{Value: value, Status: status}
}
func missing(caveat string) summary.Int64Evidence {
	return summary.Int64Evidence{Status: summary.Unavailable, Caveat: caveat}
}
func notApplicable(caveat string) summary.Int64Evidence {
	return summary.Int64Evidence{Status: summary.NotApplicable, Caveat: caveat}
}

func failureCauseFacts(txs []gateway.Transaction, status summary.Availability) summary.FailureCauseFacts {
	counts := map[string]int64{}
	for _, tx := range txs {
		if tx.Successful {
			continue
		}
		cause := tx.ResultCode
		if tx.ContractErrorType != "" {
			cause = tx.ContractErrorType
		}
		if cause == "" {
			cause = "unrecorded"
		}
		counts[cause]++
	}
	largest := int64(0)
	for _, count := range counts {
		if count > largest {
			largest = count
		}
	}
	return summary.FailureCauseFacts{LargestGroup: measured(largest, status), Distinct: measured(int64(len(counts)), status)}
}

func unavailableCapacity() summary.LedgerCapacityFacts {
	r := summary.ResourceFacts{Used: missing("Resource usage unavailable."), Limit: missing("Effective ledger limit unavailable.")}
	return summary.LedgerCapacityFacts{CPU: r, ReadEntries: r, WriteEntries: r, ReadBytes: r, WriteBytes: r, TransactionBytes: r, EventReturnBytes: r}
}

func capacityFacts(usage *gateway.LedgerSoroban, config *gateway.SorobanConfig, sorobanTxCount int64) summary.LedgerCapacityFacts {
	if sorobanTxCount == 0 && usage != nil {
		r := summary.ResourceFacts{Used: notApplicable("No Soroban transactions were included."), Limit: notApplicable("No Soroban transactions were included.")}
		return summary.LedgerCapacityFacts{CPU: r, ReadEntries: r, WriteEntries: r, ReadBytes: r, WriteBytes: r, TransactionBytes: r, EventReturnBytes: r}
	}
	result := unavailableCapacity()
	if usage == nil {
		return result
	}
	result.CPU.Used = summary.Known(usage.TotalCPUInsns)
	result.ReadBytes.Used = summary.Known(usage.TotalReadBytes)
	result.WriteBytes.Used = summary.Known(usage.TotalWriteBytes)
	if usage.FootprintEntriesAvailable {
		result.ReadEntries.Used = summary.Known(usage.TotalReadEntries)
		result.WriteEntries.Used = summary.Known(usage.TotalWriteEntries)
	} else {
		result.ReadEntries.Used = missing("Declared read footprints were not recorded for this ledger.")
		result.WriteEntries.Used = missing("Declared write footprints were not recorded for this ledger.")
	}
	if config != nil {
		result.CPU.Limit = positiveLimit(config.Instructions.LedgerMax)
		result.ReadEntries.Limit = positiveLimit(config.LedgerLimits.MaxReadEntries)
		result.WriteEntries.Limit = positiveLimit(config.LedgerLimits.MaxWriteEntries)
		result.ReadBytes.Limit = positiveLimit(config.LedgerLimits.MaxReadBytes)
		result.WriteBytes.Limit = positiveLimit(config.LedgerLimits.MaxWriteBytes)
	}
	// Current Gateway evidence has no comparable ledger denominator for these.
	result.TransactionBytes = summary.ResourceFacts{Used: missing("Soroban envelope bytes are served without a comparable ledger cap."), Limit: missing("Comparable ledger transaction-byte cap unavailable.")}
	result.EventReturnBytes = summary.ResourceFacts{Used: missing("Event and return-value bytes are not served."), Limit: missing("Effective event and return-value byte cap unavailable.")}
	return result
}

func positiveLimit(value int64) summary.Int64Evidence {
	if value <= 0 {
		return missing("Effective ledger limit was not recorded.")
	}
	return summary.Known(value)
}

func unavailableFees() summary.LedgerFeeFacts {
	return summary.LedgerFeeFacts{BaseFee: missing("Base fee unavailable."), InclusionFee: missing("Inclusion fee unavailable."), ResourceFee: missing("Resource-fee component unavailable."), RentFee: missing("Rent unavailable."), StorageWriteFee: missing("Storage write fee unavailable."), TotalCharged: missing("Total fees unavailable.")}
}

func feeFacts(l gateway.Ledger, fees *gateway.LedgerFees, usage *gateway.LedgerSoroban, sorobanTxCount int64) summary.LedgerFeeFacts {
	result := unavailableFees()
	if l.BaseFee > 0 {
		result.BaseFee = summary.Known(l.BaseFee)
	}
	if fees != nil && fees.TxCount > 0 && sorobanTxCount == 0 {
		result.InclusionFee = summary.Known(fees.MedianFee)
		result.InclusionFee.Source = "GET /silver/ledgers/{seq}/fees .median_fee"
		result.TotalCharged = summary.Known(fees.TotalFees)
	} else if fees != nil && fees.TxCount > 0 {
		result.InclusionFee = missing("Median charged fee includes Soroban resource fees; the inclusion component is not served separately.")
		result.TotalCharged = summary.Known(fees.TotalFees)
	}
	if sorobanTxCount == 0 && usage != nil {
		result.ResourceFee = notApplicable("No Soroban transactions were included.")
		result.RentFee = notApplicable("No Soroban transactions were included.")
		result.StorageWriteFee = notApplicable("No Soroban transactions were included.")
	} else if usage != nil {
		result.RentFee = summary.Known(usage.TotalRentCharged)
	}
	return result
}

func unavailableChanges() summary.LedgerChangeFacts {
	return summary.LedgerChangeFacts{Created: missing("Change statistics unavailable."), Updated: missing("Change statistics unavailable."), Deleted: missing("Change statistics unavailable."), Archived: missing("Archival statistics unavailable."), Restored: missing("Change statistics unavailable."), TTLExtended: missing("TTL extension statistics unavailable."), Baseline: missing("No versioned state-change baseline is wired yet.")}
}

func changeFacts(changes *gateway.LedgerChanges) summary.LedgerChangeFacts {
	result := unavailableChanges()
	if changes == nil || !changes.Available {
		return result
	}
	result.Created, result.Updated, result.Deleted = summary.Known(changes.Created), summary.Known(changes.Updated), summary.Known(changes.Deleted)
	result.Archived, result.Restored = summary.Known(changes.Evicted), summary.Known(changes.Restored)
	return result
}
