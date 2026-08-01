package insight

import (
	"testing"

	"github.com/withObsrvr/prism/internal/gateway"
)

func TestEvaluationExplainsAuthoritativeQuietHour(t *testing.T) {
	value := evaluationFixture()
	if err := ValidateEvaluation(value); err != nil {
		t.Fatalf("ValidateEvaluation() error = %v", err)
	}
	checks := EvaluationChecks(value)
	if len(checks) != 3 || checks[0].Label != "Contract failures" || checks[0].Value != "0.73× typical" || checks[1].Detail != "1 now, 2 typical" {
		t.Fatalf("checks = %+v", checks)
	}
	delivery := &gateway.HomeInsightDelivery{Mode: "current", EvaluatedWindowEnd: value.WindowEnd, RetainedAt: "2026-07-31T22:00:19Z", MaxAgeSeconds: 21600, ProjectionLagSecond: 41}
	if err := ValidateInsightDelivery(delivery, value); err != nil {
		t.Fatalf("ValidateInsightDelivery() error = %v", err)
	}
}

func TestEvaluationFailsClosedOnRegistryAndThresholdContradictions(t *testing.T) {
	tests := map[string]func(*gateway.HomeInsightEvaluationEnvelope){
		"missing registered rule": func(value *gateway.HomeInsightEvaluationEnvelope) { value.Rules = value.Rules[:2] },
		"duplicate rule":          func(value *gateway.HomeInsightEvaluationEnvelope) { value.Rules[2] = value.Rules[1] },
		"ratio mismatch":          func(value *gateway.HomeInsightEvaluationEnvelope) { ratio := 9.0; value.Rules[0].Ratio = &ratio },
		"false crossing":          func(value *gateway.HomeInsightEvaluationEnvelope) { value.Rules[0].ThresholdCrossed = true },
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			value := evaluationFixture()
			mutate(value)
			if err := ValidateEvaluation(value); err == nil {
				t.Fatal("invalid evaluation was accepted")
			}
		})
	}
}

func TestEvaluationAndCurrentRowsMustReconcile(t *testing.T) {
	value := evaluationFixture()
	if err := ValidateEvaluationInsights(value, nil, "empty"); err != nil {
		t.Fatalf("quiet evaluation did not reconcile: %v", err)
	}
	item := failureFixture()
	item.Observed.WindowStart = value.WindowStart
	item.Observed.WindowEnd = value.WindowEnd
	if err := ValidateEvaluationInsights(value, []gateway.HomeSummaryInsight{item}, "ready"); err == nil {
		t.Fatal("current insight without a threshold crossing was accepted")
	}
	observed, ratio := 44.0, 4.0
	value.Rules[0].ObservedValue = &observed
	value.Rules[0].Ratio = &ratio
	value.Rules[0].ThresholdCrossed = true
	value.Rules[0].QualifyingSubjectCount = 1
	if err := ValidateEvaluationInsights(value, []gateway.HomeSummaryInsight{item}, "ready"); err != nil {
		t.Fatalf("matching current insight was rejected: %v", err)
	}
}

func evaluationFixture() *gateway.HomeInsightEvaluationEnvelope {
	minimumFailure, minimumDeployments := 3.0, 2.0
	minimumRatioFailure, minimumRatioDeployments, minimumRatioActivity := 3.0, 3.0, 2.0
	observedFailure, baselineFailure, ratioFailure := 8.0, 11.0, 8.0/11.0
	observedDeployments, baselineDeployments, ratioDeployments := 1.0, 2.0, .5
	observedActivity, baselineActivity, ratioActivity := 6500.0, 6300.0, 6500.0/6300.0
	first, last := int64(3903100), int64(3903157)
	return &gateway.HomeInsightEvaluationEnvelope{
		EvidenceVersion: evaluationVersionV1, RegistryVersion: "home_insight_detector_registry_v1", Status: "ready", WindowStart: "2026-07-31T21:00:00Z", WindowEnd: "2026-07-31T22:00:00Z", ComparisonMethod: comparisonMethodV1, CompleteThroughLedger: last, Caveats: []gateway.HomeInsightCaveat{},
		Rules: []gateway.HomeInsightEvaluationRule{
			{Type: "failure_spike", Family: "risk", Direction: "negative", RuleID: "contract_failure_spike", RuleVersion: "1", ComparisonMethod: comparisonMethodV1, Status: "ready", EvaluationOutcome: "evaluated", Subject: &gateway.HomeSummaryInsightSubject{Kind: "contract", ID: "CAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAD2KM"}, EvaluatedSubjectCount: 28, ObservedValue: &observedFailure, BaselineValue: &baselineFailure, Ratio: &ratioFailure, MinimumObserved: &minimumFailure, MinimumRatio: &minimumRatioFailure, RatioComparison: "at_least", ObservedFirstLedger: &first, ObservedLastLedger: &last, Caveats: []gateway.HomeInsightCaveat{}},
			{Type: "contract_deployments_spike", Family: "activity", Direction: "neutral", RuleID: "network_contract_deployments_spike", RuleVersion: "1", ComparisonMethod: comparisonMethodV1, Status: "ready", EvaluationOutcome: "evaluated", Subject: &gateway.HomeSummaryInsightSubject{Kind: "network", ID: "testnet"}, EvaluatedSubjectCount: 1, ObservedValue: &observedDeployments, BaselineValue: &baselineDeployments, Ratio: &ratioDeployments, MinimumObserved: &minimumDeployments, MinimumRatio: &minimumRatioDeployments, RatioComparison: "at_least", ObservedFirstLedger: &first, ObservedLastLedger: &last, Caveats: []gateway.HomeInsightCaveat{}},
			{Type: "transaction_activity_spike", Family: "activity", Direction: "neutral", RuleID: "network_transaction_activity_spike", RuleVersion: "1", ComparisonMethod: comparisonMethodV1, Status: "ready", EvaluationOutcome: "evaluated", Subject: &gateway.HomeSummaryInsightSubject{Kind: "network", ID: "testnet"}, EvaluatedSubjectCount: 1, ObservedValue: &observedActivity, BaselineValue: &baselineActivity, Ratio: &ratioActivity, MinimumRatio: &minimumRatioActivity, RatioComparison: "at_least", ObservedFirstLedger: &first, ObservedLastLedger: &last, Caveats: []gateway.HomeInsightCaveat{}},
		},
		Provenance: gateway.HomeInsightEvidenceProvenance{Sources: []string{"serving.sv_home_insight_evaluations_current"}, CompleteThroughLedger: last, UpdatedAt: "2026-07-31T22:00:19Z"},
	}
}
