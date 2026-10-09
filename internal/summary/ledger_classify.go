package summary

import (
	"fmt"
	"math"
)

type LedgerThresholds struct {
	CapacityElevated  float64
	CapacityNear      float64
	FeeElevated       float64
	FeeExtreme        float64
	FailureWidespread float64
	ActivityQuiet     float64
	ActivityBusy      float64
	SorobanHeavy      float64
	ClassicHeavy      float64
	ChangeQuiet       float64
	ChangeHeavy       float64
}

var LedgerThresholdsV1 = LedgerThresholds{
	CapacityElevated:  0.50,
	CapacityNear:      0.80,
	FeeElevated:       1.10,
	FeeExtreme:        10,
	FailureWidespread: 0.10,
	ActivityQuiet:     0.50,
	ActivityBusy:      1.50,
	SorobanHeavy:      0.60,
	ClassicHeavy:      0.25,
	ChangeQuiet:       0.50,
	ChangeHeavy:       1.50,
}

func BuildLedgerSummaryState(facts LedgerFacts) (LedgerSummaryState, error) {
	if err := ValidateLedgerFacts(facts); err != nil {
		return LedgerSummaryState{}, err
	}
	metrics := BuildLedgerMetrics(facts)
	return LedgerSummaryState{Facts: facts, Metrics: metrics, Bands: ClassifyLedgerBands(facts, metrics, LedgerThresholdsV1)}, nil
}

func ValidateLedgerFacts(f LedgerFacts) error {
	if f.Sequence <= 0 {
		return fmt.Errorf("ledger sequence must be positive")
	}
	for name, value := range map[string]Int64Evidence{
		"transactions": f.Transactions, "operations": f.Operations, "successful": f.Successful, "failed": f.Failed,
	} {
		if isObserved(value.Status) && value.Value < 0 {
			return fmt.Errorf("%s cannot be negative", name)
		}
	}
	if f.Transactions.Status == Available && f.Successful.Status == Available && f.Failed.Status == Available && f.Successful.Value+f.Failed.Value != f.Transactions.Value {
		return fmt.Errorf("successful plus failed transactions must equal total transactions")
	}
	resources := map[string]ResourceFacts{
		"cpu": f.Capacity.CPU, "read_entries": f.Capacity.ReadEntries, "write_entries": f.Capacity.WriteEntries,
		"read_bytes": f.Capacity.ReadBytes, "write_bytes": f.Capacity.WriteBytes, "transaction_bytes": f.Capacity.TransactionBytes, "event_return_bytes": f.Capacity.EventReturnBytes,
	}
	for name, resource := range resources {
		if isObserved(resource.Used.Status) && resource.Used.Value < 0 {
			return fmt.Errorf("%s usage cannot be negative", name)
		}
		if isObserved(resource.Limit.Status) && resource.Limit.Value <= 0 {
			return fmt.Errorf("%s limit must be positive", name)
		}
	}
	return nil
}

func BuildLedgerMetrics(f LedgerFacts) LedgerMetrics {
	changes := sumEvidence(f.Changes.Created, f.Changes.Updated, f.Changes.Deleted, f.Changes.Restored)
	return LedgerMetrics{
		FailureRate:       ratioEvidence(f.Failed, f.Transactions),
		LargestCauseShare: ratioEvidence(f.FailureCause.LargestGroup, f.Failed),
		Capacity: LedgerCapacityMetrics{
			CPU: resourceRatio(f.Capacity.CPU), ReadEntries: resourceRatio(f.Capacity.ReadEntries), WriteEntries: resourceRatio(f.Capacity.WriteEntries),
			ReadBytes: resourceRatio(f.Capacity.ReadBytes), WriteBytes: resourceRatio(f.Capacity.WriteBytes), TransactionBytes: resourceRatio(f.Capacity.TransactionBytes), EventReturnBytes: resourceRatio(f.Capacity.EventReturnBytes),
		},
		Fees:             LedgerFeeMetrics{InclusionMultiple: ratioEvidence(f.Fees.InclusionFee, f.Fees.BaseFee)},
		Activity:         LedgerActivityMetrics{TransactionRatio: ratioEvidence(f.Transactions, f.Activity.TransactionBaseline), OperationRatio: ratioEvidence(f.Operations, f.Activity.OperationBaseline)},
		SorobanShare:     ratioEvidence(f.Mix.SorobanTransactions, f.Transactions),
		StateChangeTotal: changes,
		StateChangeRatio: ratioEvidence(changes, f.Changes.Baseline),
	}
}

func ClassifyLedgerBands(f LedgerFacts, m LedgerMetrics, t LedgerThresholds) LedgerBands {
	capacity, binding := classifyCapacity(m.Capacity, t)
	return LedgerBands{
		Version: LedgerBandVersion, ThresholdVersion: LedgerThresholdVersion,
		Evidence: classifyEvidence(f.Evidence.Status),
		Capacity: LedgerCapacityBands{
			Overall: capacity, BindingResource: binding,
			CPU: pressure(m.Capacity.CPU, t), ReadEntries: pressure(m.Capacity.ReadEntries, t), WriteEntries: pressure(m.Capacity.WriteEntries, t),
			ReadBytes: pressure(m.Capacity.ReadBytes, t), WriteBytes: pressure(m.Capacity.WriteBytes, t), TransactionBytes: pressure(m.Capacity.TransactionBytes, t), EventReturnBytes: pressure(m.Capacity.EventReturnBytes, t),
		},
		Fees: LedgerFeeBands{
			Overall: feePressure(m.Fees.InclusionMultiple, t), Inclusion: feePressure(m.Fees.InclusionMultiple, t),
			Resource: normalizedAvailability(f.Fees.ResourceFee.Status), Rent: normalizedAvailability(f.Fees.RentFee.Status), StorageWrite: normalizedAvailability(f.Fees.StorageWriteFee.Status),
		},
		Outcomes:    classifyOutcomes(f, m, t),
		Activity:    classifyActivity(m.Activity, t),
		Composition: classifyComposition(m.SorobanShare, t),
		StateChange: LedgerStateChangeBands{
			Volume: classifyChanges(m.StateChangeRatio, t), Created: normalizedAvailability(f.Changes.Created.Status), Updated: normalizedAvailability(f.Changes.Updated.Status),
			Deleted: normalizedAvailability(f.Changes.Deleted.Status), Archived: normalizedAvailability(f.Changes.Archived.Status), Restored: normalizedAvailability(f.Changes.Restored.Status), TTL: normalizedAvailability(f.Changes.TTLExtended.Status),
		},
	}
}

func ratioEvidence(n, d Int64Evidence) RatioMetric {
	status := combineAvailability(n.Status, d.Status)
	metric := RatioMetric{Numerator: n.Value, Denominator: d.Value, Status: status}
	if status != Available || d.Value <= 0 {
		if status == Available {
			metric.Status, metric.Caveat = Unavailable, "denominator is not positive"
		}
		return metric
	}
	metric.Ratio = float64(n.Value) / float64(d.Value)
	return metric
}

func resourceRatio(r ResourceFacts) RatioMetric { return ratioEvidence(r.Used, r.Limit) }

func sumEvidence(values ...Int64Evidence) Int64Evidence {
	result := Int64Evidence{Status: Available}
	seen := false
	for _, value := range values {
		if value.Status == NotApplicable {
			continue
		}
		seen = true
		result.Value += value.Value
		result.Status = combineAvailability(result.Status, value.Status)
	}
	if !seen {
		result.Status = Unavailable
	}
	return result
}

func pressure(m RatioMetric, t LedgerThresholds) PressureBand {
	switch m.Status {
	case Partial, Stale:
		return PressurePartial
	case Available:
	default:
		return PressureUnavailable
	}
	switch {
	case m.Ratio > 1:
		return PressureExceeded
	case m.Ratio >= t.CapacityNear:
		return PressureNearLimit
	case m.Ratio >= t.CapacityElevated:
		return PressureElevated
	default:
		return PressureHealthy
	}
}

func classifyCapacity(m LedgerCapacityMetrics, t LedgerThresholds) (PressureBand, string) {
	metrics := []struct {
		name   string
		metric RatioMetric
	}{
		{"cpu", m.CPU}, {"read_entries", m.ReadEntries}, {"write_entries", m.WriteEntries}, {"read_bytes", m.ReadBytes}, {"write_bytes", m.WriteBytes}, {"transaction_bytes", m.TransactionBytes}, {"event_return_bytes", m.EventReturnBytes},
	}
	known, applicable, incomplete, maxRatio, binding := 0, 0, false, -1.0, ""
	for _, item := range metrics {
		if item.metric.Status == Available {
			applicable++
			known++
			if item.metric.Ratio > maxRatio {
				maxRatio, binding = item.metric.Ratio, item.name
			}
		} else if item.metric.Status != NotApplicable {
			applicable++
			incomplete = true
		}
	}
	if applicable == 0 {
		return PressureNotApplicable, ""
	}
	if known == 0 {
		return PressureUnavailable, ""
	}
	if incomplete {
		return PressurePartial, binding
	}
	return pressure(RatioMetric{Ratio: maxRatio, Status: Available}, t), binding
}

func feePressure(m RatioMetric, t LedgerThresholds) FeeBand {
	switch m.Status {
	case Partial, Stale:
		return FeePartial
	case Available:
	default:
		return FeeUnavailable
	}
	if m.Denominator <= 0 {
		return FeeNotEvaluated
	}
	switch {
	case m.Ratio > t.FeeExtreme:
		return FeeExtreme
	case m.Ratio > t.FeeElevated:
		return FeeElevated
	default:
		return FeeBase
	}
}

func classifyOutcomes(f LedgerFacts, m LedgerMetrics, t LedgerThresholds) OutcomeBand {
	if f.Transactions.Status == Partial || f.Failed.Status == Partial || f.Transactions.Status == Stale || f.Failed.Status == Stale {
		return OutcomePartial
	}
	if f.Transactions.Status != Available || f.Failed.Status != Available {
		return OutcomeUnavailable
	}
	if f.Transactions.Value == 0 {
		return OutcomeEmpty
	}
	if f.Failed.Value == 0 {
		return OutcomeNone
	}
	if m.FailureRate.Status == Available && m.FailureRate.Ratio >= t.FailureWidespread {
		return OutcomeWidespread
	}
	if f.FailureCause.LargestGroup.Status == Available && f.FailureCause.LargestGroup.Value >= 2 && m.LargestCauseShare.Status == Available && m.LargestCauseShare.Ratio >= .5 {
		return OutcomeClustered
	}
	return OutcomeIsolated
}

func classifyActivity(m LedgerActivityMetrics, t LedgerThresholds) ActivityBand {
	values := []RatioMetric{m.TransactionRatio, m.OperationRatio}
	maxRatio, known, partial := 0.0, false, false
	for _, value := range values {
		if value.Status == Available {
			known = true
			maxRatio = math.Max(maxRatio, value.Ratio)
		} else if value.Status == Partial || value.Status == Stale {
			partial = true
		}
	}
	if partial {
		return ActivityPartial
	}
	if !known {
		return ActivityNotEvaluated
	}
	if maxRatio < t.ActivityQuiet {
		return ActivityQuiet
	}
	if maxRatio > t.ActivityBusy {
		return ActivityBusy
	}
	return ActivityTypical
}

func classifyComposition(m RatioMetric, t LedgerThresholds) CompositionBand {
	if m.Status == Partial || m.Status == Stale {
		return CompositionPartial
	}
	if m.Status != Available {
		return CompositionNotEvaluated
	}
	if m.Ratio >= t.SorobanHeavy {
		return CompositionSorobanHeavy
	}
	if m.Ratio <= t.ClassicHeavy {
		return CompositionClassicHeavy
	}
	return CompositionMixed
}

func classifyChanges(m RatioMetric, t LedgerThresholds) ChangeBand {
	if m.Status == Partial || m.Status == Stale {
		return ChangePartial
	}
	if m.Status != Available {
		return ChangeNotEvaluated
	}
	if m.Ratio < t.ChangeQuiet {
		return ChangeQuiet
	}
	if m.Ratio > t.ChangeHeavy {
		return ChangeHeavy
	}
	return ChangeTypical
}

func classifyEvidence(status Availability) EvidenceBand {
	switch status {
	case Available:
		return EvidenceComplete
	case Partial:
		return EvidencePartial
	case Stale:
		return EvidenceStale
	default:
		return EvidenceUnavailable
	}
}

func normalizedAvailability(status Availability) Availability {
	if status == "" {
		return Unavailable
	}
	return status
}

func combineAvailability(a, b Availability) Availability {
	a, b = normalizedAvailability(a), normalizedAvailability(b)
	if a == NotApplicable || b == NotApplicable {
		return NotApplicable
	}
	if a == Unavailable || b == Unavailable {
		return Unavailable
	}
	if a == Partial || b == Partial {
		return Partial
	}
	if a == Stale || b == Stale {
		return Stale
	}
	return Available
}

func isObserved(status Availability) bool {
	return status == Available || status == Partial || status == Stale
}
