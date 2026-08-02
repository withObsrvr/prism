package insight

import (
	"strings"
	"testing"

	"github.com/withObsrvr/prism/internal/gateway"
)

func TestInterpretDetailValidatesAndExplainsBoundedEvidence(t *testing.T) {
	item := failureFixture()
	packet := gateway.HomeInsightDetailResponse{
		HomeSummaryInsight: item,
		Contributors: []gateway.HomeInsightContribution{
			{Dimension: "function", Rank: 1, Kind: "function", Key: "swap", Count: 38, DenominatorName: "subject_failure_count", DenominatorValue: 42, Share: 38.0 / 42.0, FirstLedger: 3794701, LastLedger: 3795410},
			{Dimension: "result_code", Rank: 1, Kind: "result_code", Key: "HOST_FUNCTION_TRAPPED", Count: 31, DenominatorName: "subject_failure_count", DenominatorValue: 42, Share: 31.0 / 42.0, FirstLedger: 3794701, LastLedger: 3795402},
		},
		Samples: []gateway.HomeInsightSample{{SampleKind: "failed_transaction", Rank: 1, LedgerSequence: 3794701, TransactionHash: strings.Repeat("a", 64), OperationIndex: intPointer(0), ContractID: item.Subject.ID, FunctionName: "swap", ResultCode: "HOST_FUNCTION_TRAPPED", SelectionMethod: detailSampleSelectionV1}},
	}

	result, err := InterpretDetail(packet)
	if err != nil {
		t.Fatalf("InterpretDetail() error = %v", err)
	}
	if len(result.Contributors) != 2 || result.Contributors[0].Dimension != "Function" || result.Contributors[0].ShareLabel != "90%" || result.Contributors[1].Label != "Contract stopped unexpectedly" {
		t.Fatalf("contributors = %+v", result.Contributors)
	}
	if len(result.Samples) != 1 || result.Samples[0].KindLabel != "Failed transaction" || !strings.Contains(result.Samples[0].Context, "Function swap") || !strings.Contains(result.Samples[0].Context, "Contract stopped unexpectedly") {
		t.Fatalf("samples = %+v", result.Samples)
	}
	if !strings.Contains(result.MatchSummary, "at least 3×") || !strings.Contains(result.RuleSummary, "168") {
		t.Fatalf("rule explanation = %q / %q", result.MatchSummary, result.RuleSummary)
	}
}

func TestInterpretDetailRejectsIdentityAndBoundViolations(t *testing.T) {
	tests := map[string]func(*gateway.HomeInsightDetailResponse){
		"stable ID mismatch": func(packet *gateway.HomeInsightDetailResponse) { packet.Network = "mainnet" },
		"contributor beyond source ledger": func(packet *gateway.HomeInsightDetailResponse) {
			packet.Contributors[0].LastLedger = packet.Observed.SourceLedger + 1
		},
		"sample outside window": func(packet *gateway.HomeInsightDetailResponse) {
			packet.Samples[0].LedgerSequence = packet.Observed.LastLedger + 1
		},
		"unsupported sample kind": func(packet *gateway.HomeInsightDetailResponse) { packet.Samples[0].SampleKind = "raw_row" },
		"incomplete primary identity": func(packet *gateway.HomeInsightDetailResponse) {
			packet.PrimaryContributor.Identity = &gateway.HomeInsightIdentity{DisplayName: "Unproven label"}
		},
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			item := failureFixture()
			packet := gateway.HomeInsightDetailResponse{
				HomeSummaryInsight: item,
				Contributors:       []gateway.HomeInsightContribution{{Dimension: "function", Rank: 1, Kind: "function", Key: "swap", Count: 38, DenominatorName: "subject_failure_count", DenominatorValue: 42, Share: 38.0 / 42.0, FirstLedger: 3794701, LastLedger: 3795410}},
				Samples:            []gateway.HomeInsightSample{{SampleKind: "failed_transaction", Rank: 1, LedgerSequence: 3794701, TransactionHash: strings.Repeat("a", 64), SelectionMethod: detailSampleSelectionV1}},
			}
			mutate(&packet)
			if _, err := InterpretDetail(packet); err == nil {
				t.Fatal("InterpretDetail() accepted invalid detail evidence")
			}
		})
	}
}

func TestInterpretDetailCoversEveryDeployedInsightType(t *testing.T) {
	tests := []struct {
		name             string
		item             gateway.HomeSummaryInsight
		contributor      gateway.HomeInsightContribution
		sample           gateway.HomeInsightSample
		wantTitle        string
		wantContribution string
		wantLabel        string
		wantSample       string
	}{
		{
			name:             "deployment adoption",
			item:             deploymentFixture(),
			contributor:      gateway.HomeInsightContribution{Dimension: "deployed_contract_activity", Rank: 1, Kind: "contract", Key: "CBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBVQG", Count: 91, DenominatorName: "cohort_calls_since_deployment", DenominatorValue: 130, Share: 0.7, FirstLedger: 3794800, LastLedger: 3795420},
			sample:           gateway.HomeInsightSample{SampleKind: "deployment_transaction", Rank: 1, LedgerSequence: 3794800, TransactionHash: strings.Repeat("b", 64), ContractID: "CBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBVQG", SelectionMethod: detailSampleSelectionV1},
			wantTitle:        "Contract deployments increased",
			wantContribution: "New contract",
			wantLabel:        "New Example Contract",
			wantSample:       "Deployment transaction",
		},
		{
			name:             "network activity",
			item:             activityFixture(),
			contributor:      gateway.HomeInsightContribution{Dimension: "operation_category", Rank: 1, Kind: "category", Key: "soroban", Count: 6000, DenominatorName: "included_operation_count", DenominatorValue: 9800, Share: 6000.0 / 9800.0, FirstLedger: 3794701, LastLedger: 3795424},
			sample:           gateway.HomeInsightSample{SampleKind: "activity_transaction", Rank: 1, LedgerSequence: 3795000, TransactionHash: strings.Repeat("c", 64), SelectionMethod: detailSampleSelectionV1},
			wantTitle:        "Transaction activity increased",
			wantContribution: "Operation category",
			wantLabel:        "soroban",
			wantSample:       "Activity transaction",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			packet := gateway.HomeInsightDetailResponse{HomeSummaryInsight: test.item, Contributors: []gateway.HomeInsightContribution{test.contributor}, Samples: []gateway.HomeInsightSample{test.sample}}
			if test.name == "deployment adoption" {
				packet.PrimaryContributor.Identity = &gateway.HomeInsightIdentity{DisplayName: "New Example Contract", Kind: "contract", VerificationStatus: "unknown", Source: "contract_id"}
			}
			result, err := InterpretDetail(packet)
			if err != nil {
				t.Fatalf("InterpretDetail() error = %v", err)
			}
			if !strings.Contains(result.Title, test.wantTitle) || len(result.Contributors) != 1 || result.Contributors[0].Dimension != test.wantContribution || result.Contributors[0].Label != test.wantLabel || len(result.Samples) != 1 || result.Samples[0].KindLabel != test.wantSample {
				t.Fatalf("detail interpretation = %+v", result)
			}
		})
	}
}

func intPointer(value int) *int { return &value }
