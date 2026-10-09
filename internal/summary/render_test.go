package summary

import (
	"strings"
	"testing"
)

func TestEveryRegisteredInterpretationHasControlledTemplate(t *testing.T) {
	for _, spec := range LedgerInterpretationsV2 {
		t.Run(string(spec.ID), func(t *testing.T) {
			state := renderTestState()
			for _, requirement := range spec.AnyOf[0].All {
				setRenderTestBand(&state.Bands, requirement.Path, requirement.Allowed[0])
			}
			candidates, err := EligibleLedgerCandidates(state.Bands)
			if err != nil {
				t.Fatal(err)
			}
			if !candidatePresent(candidates, spec.ID) {
				t.Fatalf("fixture did not make %q eligible", spec.ID)
			}
			rendered, err := RenderLedgerSummary(renderTestEnvelope(state, candidates, spec.ID))
			if err != nil {
				t.Fatal(err)
			}
			if rendered.Headline.Emphasis == "" || rendered.Detail == "" {
				t.Fatalf("empty controlled template: %+v", rendered)
			}
		})
	}
}

func TestRenderRejectsSelectionThatStateDoesNotAdmit(t *testing.T) {
	state := renderTestState()
	state.Bands.Evidence, state.Bands.Outcomes = EvidencePartial, OutcomePartial
	envelope := renderTestEnvelope(state, []Candidate{{ID: InterpretationAllSucceeded}}, InterpretationAllSucceeded)
	if _, err := RenderLedgerSummary(envelope); err == nil {
		t.Fatal("render accepted ineligible success claim")
	}
}

func TestRoutineTemplateUsesExactFactsAndControlledActions(t *testing.T) {
	state := renderTestState()
	state.Bands.Evidence, state.Bands.Capacity.Overall, state.Bands.Fees.Overall = EvidenceComplete, PressureHealthy, FeeBase
	state.Bands.Outcomes, state.Bands.Activity = OutcomeNone, ActivityTypical
	candidates, err := EligibleLedgerCandidates(state.Bands)
	if err != nil {
		t.Fatal(err)
	}
	rendered, err := RenderLedgerSummary(renderTestEnvelope(state, candidates, InterpretationRoutineLedger))
	if err != nil {
		t.Fatal(err)
	}
	if rendered.Headline.Emphasis != "routine" || !strings.Contains(rendered.Detail, "100 included transactions") {
		t.Fatalf("rendered = %+v", rendered)
	}
	if len(rendered.NextActions) == 0 || rendered.NextActions[0].Href != "#s1" {
		t.Fatalf("actions = %+v", rendered.NextActions)
	}
}

func renderTestState() LedgerSummaryState {
	return LedgerSummaryState{
		Facts:   LedgerFacts{Sequence: 123, Evidence: Evidence{Status: Available}, Transactions: Known(100), Failed: Known(4)},
		Metrics: LedgerMetrics{FailureRate: RatioMetric{Numerator: 4, Denominator: 100, Ratio: .04, Status: Available}, LargestCauseShare: RatioMetric{Numerator: 3, Denominator: 4, Ratio: .75, Status: Available}, SorobanShare: RatioMetric{Numerator: 70, Denominator: 100, Ratio: .7, Status: Available}, StateChangeTotal: Known(250), StateChangeRatio: RatioMetric{Ratio: 2.5, Status: Available}, Fees: LedgerFeeMetrics{InclusionMultiple: RatioMetric{Ratio: 12, Status: Available}}, Activity: LedgerActivityMetrics{TransactionRatio: RatioMetric{Ratio: 2, Status: Available}, OperationRatio: RatioMetric{Ratio: 1.8, Status: Available}}, Capacity: LedgerCapacityMetrics{CPU: RatioMetric{Ratio: .9, Status: Available}}},
		Bands:   LedgerBands{Version: LedgerBandVersion, ThresholdVersion: LedgerThresholdVersion, Evidence: EvidenceComplete, Capacity: LedgerCapacityBands{Overall: PressureElevated, BindingResource: "cpu"}, Fees: LedgerFeeBands{Overall: FeeBase}, Outcomes: OutcomeIsolated, Activity: ActivityNotEvaluated, Composition: CompositionMixed, StateChange: LedgerStateChangeBands{Volume: ChangeNotEvaluated}},
	}
}

func renderTestEnvelope(state LedgerSummaryState, candidates []Candidate, lead InterpretationID) LedgerSummaryEnvelope {
	eligible := make([]InterpretationID, 0, len(candidates))
	for _, candidate := range candidates {
		eligible = append(eligible, candidate.ID)
	}
	return LedgerSummaryEnvelope{State: state, Candidates: candidates, Selection: Selection{Lead: lead, Source: "deterministic", RegistryVersion: LedgerInterpretationRegistryVersion}, SelectionTrace: SelectionTrace{Surface: "ledger_summary", SelectorVersion: LedgerSelectorVersion, Mode: "deterministic", Eligible: eligible, DeterministicLead: lead, AppliedLead: lead, AppliedSource: "deterministic", CacheStatus: "bypass"}, BandVersion: LedgerBandVersion, BaselineVersion: LedgerBaselineVersion, ThresholdVersion: LedgerThresholdVersion, RegistryVersion: LedgerInterpretationRegistryVersion, TemplateVersion: LedgerTemplateVersion}
}

func setRenderTestBand(b *LedgerBands, path, value string) {
	switch path {
	case "evidence":
		b.Evidence = EvidenceBand(value)
	case "capacity.overall":
		b.Capacity.Overall = PressureBand(value)
	case "fees.overall":
		b.Fees.Overall = FeeBand(value)
	case "outcomes":
		b.Outcomes = OutcomeBand(value)
	case "activity":
		b.Activity = ActivityBand(value)
	case "composition":
		b.Composition = CompositionBand(value)
	case "state_change.volume":
		b.StateChange.Volume = ChangeBand(value)
	}
}
func candidatePresent(values []Candidate, id InterpretationID) bool {
	for _, value := range values {
		if value.ID == id {
			return true
		}
	}
	return false
}
