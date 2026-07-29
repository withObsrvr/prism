package insight

import (
	"strings"
	"testing"

	"github.com/withObsrvr/prism/internal/gateway"
)

func TestInterpretFailureUsesOnlyReconciledEvidence(t *testing.T) {
	item := failureFixture()
	interpretation, err := Interpret(item)
	if err != nil {
		t.Fatalf("Interpret() error = %v", err)
	}
	for _, want := range []string{"42 failures", "median of 7", "swap accounted for 38 of 42 failures", "90%"} {
		if !strings.Contains(interpretation.Summary+interpretation.Detail, want) {
			t.Errorf("interpretation missing %q: %+v", want, interpretation)
		}
	}
	if interpretation.RuleID != "contract_failure_spike" || interpretation.Subject.Label != "Example Router" || len(interpretation.Evidence) != 1 {
		t.Fatalf("typed rule metadata was not preserved: %+v", interpretation)
	}
	if !strings.Contains(interpretation.Evidence[0].Href, "contract=CAAAAA") || !strings.Contains(interpretation.Evidence[0].Href, "status=failed") || !strings.Contains(interpretation.Evidence[0].Href, "time=coverage") {
		t.Fatalf("evidence locator was not mapped exactly: %+v", interpretation.Evidence)
	}
}

func TestInterpretRejectsInconsistentFacts(t *testing.T) {
	item := failureFixture()
	item.Facts.Failure.SuccessCount--
	if _, err := Interpret(item); err == nil {
		t.Fatal("Interpret() accepted inconsistent failure totals")
	}
}

func TestInterpretRejectsInconsistentContributorClaim(t *testing.T) {
	item := failureFixture()
	item.PrimaryContributor.Share = 0.25
	if _, err := Interpret(item); err == nil {
		t.Fatal("Interpret() accepted a contributor share that did not reconcile")
	}
}

func TestInterpretRejectsMissingRequiredCaveatArray(t *testing.T) {
	item := failureFixture()
	item.Caveats = nil
	if _, err := Interpret(item); err == nil {
		t.Fatal("Interpret() accepted a packet without the required caveat array")
	}
}

func TestInterpretDeploymentSpikeUsesTypedActivityFacts(t *testing.T) {
	interpretation, err := Interpret(deploymentFixture())
	if err != nil {
		t.Fatalf("Interpret() error = %v", err)
	}
	for _, want := range []string{"6 contracts", "3 times", "91 calls", "87 succeeded", "4 failed"} {
		if !strings.Contains(interpretation.Summary+interpretation.Detail, want) {
			t.Errorf("deployment interpretation missing %q: %+v", want, interpretation)
		}
	}
	if interpretation.RuleID != "network_contract_deployments_spike" || !strings.Contains(interpretation.Evidence[0].Href, "from_ledger=3794701") {
		t.Fatalf("deployment rule metadata was not preserved: %+v", interpretation)
	}
}

func TestInterpretActivitySpikeUsesNamedContributorDenominator(t *testing.T) {
	interpretation, err := Interpret(activityFixture())
	if err != nil {
		t.Fatalf("Interpret() error = %v", err)
	}
	for _, want := range []string{"3,200", "2.29 times", "6,000 of 9,800 included operations", "61%"} {
		if !strings.Contains(interpretation.Summary+interpretation.Detail, want) {
			t.Errorf("activity interpretation missing %q: %+v", want, interpretation)
		}
	}
}

func TestInterpretUnknownVersionFallsBackToMeasuredValues(t *testing.T) {
	item := gateway.HomeSummaryInsight{
		Type:             "future_signal",
		EvidenceVersion:  "home_insight_evidence_v2",
		ObservedValue:    12,
		BaselineValue:    4,
		ComparisonMethod: "future_method",
		WindowStart:      "2026-07-25T14:00:00Z",
		WindowEnd:        "2026-07-25T15:00:00Z",
	}
	interpretation, err := Interpret(item)
	if err != nil {
		t.Fatalf("Interpret() error = %v", err)
	}
	if !interpretation.Generic || !strings.Contains(interpretation.Summary, "observation was 12") || !strings.Contains(interpretation.Summary, "baseline of 4") {
		t.Fatalf("unknown version did not degrade to measured facts: %+v", interpretation)
	}
	if len(interpretation.Evidence) != 0 || len(interpretation.Caveats) != 1 {
		t.Fatalf("unknown version claimed unsupported evidence: %+v", interpretation)
	}
}

func failureFixture() gateway.HomeSummaryInsight {
	minimumObserved := 3.0
	caveats := []gateway.HomeInsightCaveat{}
	return gateway.HomeSummaryInsight{
		InsightID:       "hiev1_wwnNOzs1woMnG3W4aRHIShks_8_e0A70A62b6lJm7IE",
		Network:         "testnet",
		Type:            "failure_spike",
		EvidenceVersion: EvidenceVersionV1,
		Definition: &gateway.HomeInsightDefinition{
			RuleID: "contract_failure_spike", RuleVersion: "1", ComparisonMethod: "rolling_7d_median_prior_complete_hour", MinimumObserved: &minimumObserved, MinimumRatio: 3,
		},
		Subject: gateway.HomeSummaryInsightSubject{
			Kind: "contract", ID: "CAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAD2KM",
			Identity: &gateway.HomeInsightIdentity{DisplayName: "Example Router", Kind: "protocol_contract", VerificationStatus: "inferred", Source: "semantic_contract_registry"},
		},
		Observed: &gateway.HomeInsightObserved{Value: 42, WindowStart: "2026-07-25T14:00:00Z", WindowEnd: "2026-07-25T15:00:00Z", FirstLedger: 3794701, LastLedger: 3795424, SourceLedger: 3795424},
		Baseline: &gateway.HomeInsightBaseline{Value: 7, WindowStart: "2026-07-18T14:00:00Z", WindowEnd: "2026-07-25T14:00:00Z", CompleteHourCount: 168, ZeroBaselinePolicy: "omit_ratio_insight"},
		Ratio:    6,
		Facts: &gateway.HomeInsightFacts{Failure: &gateway.HomeInsightFailureFacts{
			Kind: "failure_spike", AttemptCount: 110, SuccessCount: 68, FailureCount: 42, DistinctTransactionCount: 42, DistinctCallerCount: 19, NetworkFailureCount: 77, SubjectFailureShare: 42.0 / 77.0,
		}},
		PrimaryContributor: &gateway.HomeInsightContribution{Dimension: "function", Kind: "function", Key: "swap", Count: 38, DenominatorName: "subject_failure_count", DenominatorValue: 42, Share: 38.0 / 42.0, FirstLedger: 3794701, LastLedger: 3795410},
		EvidenceLocator:    &gateway.HomeInsightEvidenceLocator{Kind: "contract_invocations", ContractID: "CAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAD2KM", LedgerStart: 3794701, LedgerEnd: 3795424, Status: "failed"},
		EvidenceCount:      42,
		Status:             "ready",
		Caveats:            &caveats,
		EvidenceProvenance: &gateway.HomeInsightEvidenceProvenance{Sources: []string{"serving.sv_home_insight_history"}, CompleteThroughLedger: 3795424, UpdatedAt: "2026-07-25T15:00:05Z"},
	}
}

func deploymentFixture() gateway.HomeSummaryInsight {
	minimumObserved := 2.0
	caveats := []gateway.HomeInsightCaveat{}
	return gateway.HomeSummaryInsight{
		InsightID:       "hiev1_4Hswy_IBL14jF1duj8MUuIUyvIYLw_BmOeQTje3BHvM",
		Network:         "testnet",
		Type:            "contract_deployments_spike",
		EvidenceVersion: EvidenceVersionV1,
		Definition: &gateway.HomeInsightDefinition{
			RuleID: "network_contract_deployments_spike", RuleVersion: "1", ComparisonMethod: comparisonMethodV1, MinimumObserved: &minimumObserved, MinimumRatio: 3,
		},
		Subject:  gateway.HomeSummaryInsightSubject{Kind: "network", ID: "testnet"},
		Observed: &gateway.HomeInsightObserved{Value: 6, WindowStart: "2026-07-25T14:00:00Z", WindowEnd: "2026-07-25T15:00:00Z", FirstLedger: 3794701, LastLedger: 3794900, SourceLedger: 3795424},
		Baseline: &gateway.HomeInsightBaseline{Value: 2, WindowStart: "2026-07-18T14:00:00Z", WindowEnd: "2026-07-25T14:00:00Z", CompleteHourCount: 168, ZeroBaselinePolicy: "omit_ratio_insight"},
		Ratio:    3,
		Facts: &gateway.HomeInsightFacts{Deployment: &gateway.HomeInsightDeploymentFacts{
			Kind: "contract_deployments_spike", DeploymentCount: 6,
			PrimaryContract: gateway.HomeInsightPrimaryContract{ContractID: "CBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBVQG", DeploymentLedger: 3794800, DeployedAt: "2026-07-25T14:15:00Z", CallsSinceDeployment: 91, DistinctCallerCount: 24, SuccessCount: 87, FailureCount: 4, ActivityWindowStart: "2026-07-25T14:15:00Z", ActivityWindowEnd: "2026-07-25T15:00:00Z"},
		}},
		PrimaryContributor: &gateway.HomeInsightContribution{Dimension: "deployed_contract_activity", Kind: "contract", Key: "CBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBVQG", Count: 91, DenominatorName: "cohort_calls_since_deployment", DenominatorValue: 130, Share: 0.7, FirstLedger: 3794800, LastLedger: 3795420},
		EvidenceLocator:    &gateway.HomeInsightEvidenceLocator{Kind: "contract_deployments", LedgerStart: 3794701, LedgerEnd: 3794900},
		EvidenceCount:      6,
		Status:             "ready",
		Caveats:            &caveats,
		EvidenceProvenance: &gateway.HomeInsightEvidenceProvenance{Sources: []string{"serving.sv_home_insight_history"}, CompleteThroughLedger: 3795424, UpdatedAt: "2026-07-25T15:00:05Z"},
	}
}

func activityFixture() gateway.HomeSummaryInsight {
	caveats := []gateway.HomeInsightCaveat{}
	return gateway.HomeSummaryInsight{
		InsightID:       "hiev1_J722cD1Pa2hXMKbtQAyrOLvK3YoDphVjYxdUJlhCnrA",
		Network:         "testnet",
		Type:            "transaction_activity_spike",
		EvidenceVersion: EvidenceVersionV1,
		Definition: &gateway.HomeInsightDefinition{
			RuleID: "network_transaction_activity_spike", RuleVersion: "1", ComparisonMethod: comparisonMethodV1, MinimumRatio: 2,
		},
		Subject:  gateway.HomeSummaryInsightSubject{Kind: "network", ID: "testnet"},
		Observed: &gateway.HomeInsightObserved{Value: 3200, WindowStart: "2026-07-25T14:00:00Z", WindowEnd: "2026-07-25T15:00:00Z", FirstLedger: 3794701, LastLedger: 3795424, SourceLedger: 3795424},
		Baseline: &gateway.HomeInsightBaseline{Value: 1400, WindowStart: "2026-07-18T14:00:00Z", WindowEnd: "2026-07-25T14:00:00Z", CompleteHourCount: 168, ZeroBaselinePolicy: "omit_ratio_insight"},
		Ratio:    3200.0 / 1400.0,
		Facts: &gateway.HomeInsightFacts{Activity: &gateway.HomeInsightActivityFacts{
			Kind: "transaction_activity_spike", IncludedTransactionCount: 3200, SuccessfulTransactionCount: 3100, FailedTransactionCount: 100, IncludedOperationCount: 9800, SorobanTransactionCount: 800, ClassicOnlyTransactionCount: 2400,
		}},
		PrimaryContributor: &gateway.HomeInsightContribution{Dimension: "operation_category", Kind: "category", Key: "soroban", Count: 6000, DenominatorName: "included_operation_count", DenominatorValue: 9800, Share: 6000.0 / 9800.0, FirstLedger: 3794701, LastLedger: 3795424},
		EvidenceLocator:    &gateway.HomeInsightEvidenceLocator{Kind: "ledger_activity", LedgerStart: 3794701, LedgerEnd: 3795424, Category: "soroban"},
		EvidenceCount:      3200,
		Status:             "ready",
		Caveats:            &caveats,
		EvidenceProvenance: &gateway.HomeInsightEvidenceProvenance{Sources: []string{"serving.sv_home_insight_history"}, CompleteThroughLedger: 3795424, UpdatedAt: "2026-07-25T15:00:05Z"},
	}
}
