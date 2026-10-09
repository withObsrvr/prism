// Package summary turns typed evidence into deterministic semantic state.
// It does not render prose or retrieve data.
package summary

type Availability string

const (
	Available     Availability = "available"
	Partial       Availability = "partial"
	Stale         Availability = "stale"
	Unavailable   Availability = "unavailable"
	NotApplicable Availability = "not_applicable"
)

type Evidence struct {
	Status                Availability `json:"status"`
	Sources               []string     `json:"sources,omitempty"`
	CompleteThroughLedger int64        `json:"complete_through_ledger,omitempty"`
	Caveats               []string     `json:"caveats,omitempty"`
}

type Int64Evidence struct {
	Value  int64        `json:"value"`
	Status Availability `json:"status"`
	Source string       `json:"source,omitempty"`
	Caveat string       `json:"caveat,omitempty"`
}

func Known(value int64) Int64Evidence {
	return Int64Evidence{Value: value, Status: Available}
}

type RatioMetric struct {
	Numerator   int64        `json:"numerator"`
	Denominator int64        `json:"denominator"`
	Ratio       float64      `json:"ratio"`
	Status      Availability `json:"status"`
	Caveat      string       `json:"caveat,omitempty"`
}

type PressureBand string

const (
	PressureUnavailable   PressureBand = "unavailable"
	PressureNotApplicable PressureBand = "not_applicable"
	PressurePartial       PressureBand = "partial"
	PressureHealthy       PressureBand = "healthy"
	PressureElevated      PressureBand = "elevated"
	PressureNearLimit     PressureBand = "near_limit"
	PressureExceeded      PressureBand = "exceeded"
)

type FeeBand string

const (
	FeeUnavailable  FeeBand = "unavailable"
	FeePartial      FeeBand = "partial"
	FeeNotEvaluated FeeBand = "not_evaluated"
	FeeBase         FeeBand = "base"
	FeeElevated     FeeBand = "elevated"
	FeeExtreme      FeeBand = "extreme"
)

type OutcomeBand string

const (
	OutcomeUnavailable OutcomeBand = "unavailable"
	OutcomePartial     OutcomeBand = "partial"
	OutcomeEmpty       OutcomeBand = "empty"
	OutcomeNone        OutcomeBand = "none"
	OutcomeIsolated    OutcomeBand = "isolated"
	OutcomeClustered   OutcomeBand = "clustered"
	OutcomeWidespread  OutcomeBand = "widespread"
)

type ActivityBand string

const (
	ActivityUnavailable  ActivityBand = "unavailable"
	ActivityPartial      ActivityBand = "partial"
	ActivityNotEvaluated ActivityBand = "not_evaluated"
	ActivityQuiet        ActivityBand = "quiet"
	ActivityTypical      ActivityBand = "typical"
	ActivityBusy         ActivityBand = "busy"
)

type CompositionBand string

const (
	CompositionUnavailable  CompositionBand = "unavailable"
	CompositionPartial      CompositionBand = "partial"
	CompositionNotEvaluated CompositionBand = "not_evaluated"
	CompositionClassicHeavy CompositionBand = "classic_heavy"
	CompositionMixed        CompositionBand = "mixed"
	CompositionSorobanHeavy CompositionBand = "soroban_heavy"
)

type ChangeBand string

const (
	ChangeUnavailable  ChangeBand = "unavailable"
	ChangePartial      ChangeBand = "partial"
	ChangeNotEvaluated ChangeBand = "not_evaluated"
	ChangeQuiet        ChangeBand = "quiet"
	ChangeTypical      ChangeBand = "typical"
	ChangeHeavy        ChangeBand = "heavy"
)

type EvidenceBand string

const (
	EvidenceComplete    EvidenceBand = "complete"
	EvidencePartial     EvidenceBand = "partial"
	EvidenceStale       EvidenceBand = "stale"
	EvidenceUnavailable EvidenceBand = "unavailable"
)
