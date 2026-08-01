package insight

import (
	"strings"
	"testing"

	"github.com/withObsrvr/prism/internal/gateway"
)

func TestInterpretPositiveEvidenceV2(t *testing.T) {
	tests := []struct {
		name string
		item gateway.HomeSummaryInsight
		want []string
	}{
		{"successful activity growth", growthFixture(), []string{"Successful network activity grew", "6,400 successful transactions", "2%"}},
		{"failure recovery", recoveryFixture(), []string{"returned to their normal range", "46", "240 calls", "not explained by inactivity"}},
		{"new contract adoption", adoptionFixture(), []string{"new contract is gaining use", "75 calls", "9 distinct callers", "96%"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := Interpret(test.item)
			if err != nil {
				t.Fatalf("Interpret() error = %v", err)
			}
			if result.Generic || result.Severity != "positive" {
				t.Fatalf("positive evidence was not recognized: %+v", result)
			}
			for _, want := range test.want {
				if !strings.Contains(result.Title+result.Summary+result.Detail, want) {
					t.Errorf("interpretation missing %q: %+v", want, result)
				}
			}
			if _, err := InterpretDetail(gateway.HomeInsightDetailResponse{HomeSummaryInsight: test.item}); err != nil {
				t.Errorf("InterpretDetail() rejected the frozen v2 identity: %v", err)
			}
		})
	}
}

func TestInterpretPositiveEvidenceRejectsFalseGoodNews(t *testing.T) {
	tests := map[string]gateway.HomeSummaryInsight{
		"failure-heavy growth":            growthFixture(),
		"recovery by inactivity":          recoveryFixture(),
		"adoption without enough callers": adoptionFixture(),
	}
	tests["failure-heavy growth"].Facts.Growth.CurrentFailureRate = .25
	tests["recovery by inactivity"].Facts.Recovery.CurrentAttemptCount = 5
	tests["adoption without enough callers"].Facts.Adoption.DistinctCallerCount = 1
	for name, item := range tests {
		t.Run(name, func(t *testing.T) {
			if _, err := Interpret(item); err == nil {
				t.Fatal("invalid positive evidence was narrated")
			}
		})
	}
}

func growthFixture() gateway.HomeSummaryInsight {
	minimum := 500.0
	caveats := []gateway.HomeInsightCaveat{}
	return gateway.HomeSummaryInsight{
		InsightID: "hiev2_NykqrnkhWFaXhSkKMWK1_nk6MHj6Smy1n7ovYgM3Htc", Network: "testnet", Type: "successful_activity_growth", Family: "activity", Direction: "positive", Severity: "high", EvidenceVersion: EvidenceVersionV2,
		Definition: &gateway.HomeInsightDefinition{RuleID: "network_successful_activity_growth", RuleVersion: "1", ComparisonMethod: comparisonMethodV1, MinimumObserved: &minimum, MinimumRatio: 2, RatioComparison: "at_least"},
		Subject:    gateway.HomeSummaryInsightSubject{Kind: "network", ID: "testnet"},
		Observed:   &gateway.HomeInsightObserved{Value: 6400, WindowStart: "2026-07-31T21:00:00Z", WindowEnd: "2026-07-31T22:00:00Z", FirstLedger: 3903100, LastLedger: 3903157, SourceLedger: 3903157},
		Baseline:   &gateway.HomeInsightBaseline{Value: 3200, WindowStart: "2026-07-24T21:00:00Z", WindowEnd: "2026-07-31T21:00:00Z", CompleteHourCount: 168, ZeroBaselinePolicy: "omit_ratio_insight"}, Ratio: 2,
		Facts:           &gateway.HomeInsightFacts{Growth: &gateway.HomeInsightGrowthFacts{Kind: "successful_activity_growth", IncludedTransactionCount: 6500, SuccessfulTransactionCount: 6400, FailedTransactionCount: 100, IncludedOperationCount: 7200, SorobanTransactionCount: 1800, ClassicOnlyTransactionCount: 4700, BaselineSuccessfulTransactionCount: 3200, CurrentFailureRate: 100.0 / 6500.0, BaselineFailureRate: .03, MaximumFailureRate: .1, FailureRateTolerance: .02}},
		EvidenceLocator: &gateway.HomeInsightEvidenceLocator{Kind: "ledger_activity", LedgerStart: 3903100, LedgerEnd: 3903157, Status: "successful"}, EvidenceCount: 6400, Status: "ready", Caveats: &caveats,
		EvidenceProvenance: &gateway.HomeInsightEvidenceProvenance{Sources: []string{"serving.sv_home_insight_history", "serving.sv_home_insight_growth_facts"}, CompleteThroughLedger: 3903157, UpdatedAt: "2026-07-31T22:00:19Z"},
	}
}

func recoveryFixture() gateway.HomeSummaryInsight {
	minimum := 20.0
	caveats := []gateway.HomeInsightCaveat{}
	return gateway.HomeSummaryInsight{
		InsightID: "hiev2_vz802Mvt2WIyjyJT12HbDUOZYEnJwDhMUZhKvttqfyw", Network: "testnet", Type: "failure_recovery", Family: "recovery", Direction: "positive", Severity: "high", EvidenceVersion: EvidenceVersionV2,
		Definition: &gateway.HomeInsightDefinition{RuleID: "contract_failure_recovery", RuleVersion: "1", ComparisonMethod: comparisonMethodV1, MinimumObserved: &minimum, MinimumRatio: 1, RatioComparison: "at_most"},
		Subject:    gateway.HomeSummaryInsightSubject{Kind: "contract", ID: "CBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBVQG"},
		Observed:   &gateway.HomeInsightObserved{Value: 1, WindowStart: "2026-07-31T21:00:00Z", WindowEnd: "2026-07-31T22:00:00Z", FirstLedger: 3903100, LastLedger: 3903157, SourceLedger: 3903157},
		Baseline:   &gateway.HomeInsightBaseline{Value: 3, WindowStart: "2026-07-24T21:00:00Z", WindowEnd: "2026-07-31T21:00:00Z", CompleteHourCount: 168, ZeroBaselinePolicy: "omit_ratio_insight"}, Ratio: 1.0 / 3.0,
		Facts:           &gateway.HomeInsightFacts{Recovery: &gateway.HomeInsightRecoveryFacts{Kind: "failure_recovery", PriorInsightID: "hiev1_6hRt7AxFnVB9QhXsFAwfM-snYHEt9fC0BXaM7ElaVnU", PriorWindowStart: "2026-07-31T17:00:00Z", PriorWindowEnd: "2026-07-31T18:00:00Z", PriorFailureCount: 46, CurrentFailureCount: 1, CurrentAttemptCount: 240, CurrentSuccessCount: 239, BaselineFailureCount: 2, BaselineAttemptCount: 210, NormalRangeFailureCount: 3, MinimumAttemptCount: 20, ActivityFloorRatio: .25}},
		EvidenceLocator: &gateway.HomeInsightEvidenceLocator{Kind: "contract_invocations", ContractID: "CBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBVQG", LedgerStart: 3903100, LedgerEnd: 3903157, Status: "successful"}, EvidenceCount: 240, Status: "ready", Caveats: &caveats,
		EvidenceProvenance: &gateway.HomeInsightEvidenceProvenance{Sources: []string{"serving.sv_home_insight_history", "serving.sv_home_insight_recovery_facts"}, CompleteThroughLedger: 3903157, UpdatedAt: "2026-07-31T22:00:19Z"},
	}
}

func adoptionFixture() gateway.HomeSummaryInsight {
	minimum := 25.0
	caveats := []gateway.HomeInsightCaveat{}
	return gateway.HomeSummaryInsight{
		InsightID: "hiev2_rOahI9XCKX9A8xPnjnjqKUtolq2sFBmL1f4NDMBEDIU", Network: "testnet", Type: "new_contract_adoption", Family: "adoption", Direction: "positive", Severity: "high", EvidenceVersion: EvidenceVersionV2,
		Definition: &gateway.HomeInsightDefinition{RuleID: "contract_new_adoption", RuleVersion: "1", ComparisonMethod: comparisonMethodAdoption, MinimumObserved: &minimum, MinimumRatio: 1, RatioComparison: "at_least"},
		Subject:    gateway.HomeSummaryInsightSubject{Kind: "contract", ID: "CBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBVQG"},
		Observed:   &gateway.HomeInsightObserved{Value: 75, WindowStart: "2026-07-31T20:12:04Z", WindowEnd: "2026-07-31T22:00:00Z", FirstLedger: 3903100, LastLedger: 3903157, SourceLedger: 3903157},
		Baseline:   &gateway.HomeInsightBaseline{Value: 25, WindowStart: "2026-07-31T20:12:04Z", WindowEnd: "2026-07-31T20:12:04Z", CompleteHourCount: 0, ZeroBaselinePolicy: "omit_ratio_insight"}, Ratio: 3,
		Facts:           &gateway.HomeInsightFacts{Adoption: &gateway.HomeInsightAdoptionFacts{Kind: "new_contract_adoption", ContractID: "CBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBVQG", DeploymentLedger: 3903100, DeployedAt: "2026-07-31T20:12:04Z", DeploymentTransactionHash: strings.Repeat("d", 64), DeploymentOperationIndex: 0, DeployerAccount: "GAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAWHF", CallsSinceDeployment: 75, DistinctCallerCount: 9, SuccessCount: 72, FailureCount: 3, SuccessRate: .96, TopFunction: "swap", ObservationWindowEnd: "2026-07-31T22:00:00Z", AdoptionAgeSeconds: 6476, MinimumCalls: 25, MinimumDistinctCallers: 3, MinimumSuccessRate: .8, MaximumAdoptionAgeSeconds: 259200}},
		EvidenceLocator: &gateway.HomeInsightEvidenceLocator{Kind: "contract_deployments", ContractID: "CBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBVQG", LedgerStart: 3903100, LedgerEnd: 3903157}, EvidenceCount: 75, Status: "ready", Caveats: &caveats,
		EvidenceProvenance: &gateway.HomeInsightEvidenceProvenance{Sources: []string{"serving.sv_home_insight_history", "serving.sv_home_insight_adoption_facts"}, CompleteThroughLedger: 3903157, UpdatedAt: "2026-07-31T22:00:19Z"},
	}
}
