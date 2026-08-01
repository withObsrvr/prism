package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/withObsrvr/prism/internal/gateway"
)

const insightDetailTestID = "hiev1_wwnNOzs1woMnG3W4aRHIShks_8_e0A70A62b6lJm7IE"

func TestInsightDetailV2RendersValidatedEvidenceNarrative(t *testing.T) {
	packet := insightDetailTestPacket()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/lake/v1/testnet/api/v1/home/insights/"+insightDetailTestID {
			t.Fatalf("path = %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(packet); err != nil {
			t.Fatalf("encode packet: %v", err)
		}
	}))
	defer server.Close()

	logger := testHomeLogger()
	client := gateway.New(gateway.Config{BaseURL: server.URL, APIKey: "test", Timeout: time.Second}, logger, context.Background())
	defer client.Stop()
	h := &Handlers{Logger: logger, Gateway: client, DataSource: "auto"}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v2/insight/"+insightDetailTestID+"?network=testnet", nil)
	request.SetPathValue("id", insightDetailTestID)
	h.InsightDetailV2(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	body := recorder.Body.String()
	for _, want := range []string{
		"Why Prism flagged this",
		"42 failures",
		"90%",
		"Contract stopped unexpectedly",
		"Transactions to inspect",
		"Samples do not prove aggregate completeness",
		"aaaaaaaa…aaaaaa",
		"serving.sv_home_insight_history",
		"/v2/tx/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa?network=testnet",
		"/v2/explore?contract=CAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAD2KM&amp;from_ledger=3794701&amp;network=testnet&amp;status=failed&amp;time=coverage&amp;to_ledger=3795424",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("response missing %q", want)
		}
	}
	if strings.Contains(body, "Demo fixture") || strings.Contains(body, "raw_row") {
		t.Fatalf("response included fallback or unvalidated evidence: %s", body)
	}
}

func TestInsightDetailV2RejectsInvalidIDBeforeGateway(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { calls++ }))
	defer server.Close()
	logger := testHomeLogger()
	client := gateway.New(gateway.Config{BaseURL: server.URL, APIKey: "test", Timeout: time.Second}, logger, context.Background())
	defer client.Stop()
	h := &Handlers{Logger: logger, Gateway: client, DataSource: "auto"}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v2/insight/not-an-insight?network=testnet", nil)
	request.SetPathValue("id", "not-an-insight")
	h.InsightDetailV2(recorder, request)

	if recorder.Code != http.StatusBadRequest || !strings.Contains(recorder.Body.String(), "This insight link is invalid") || strings.Contains(recorder.Body.String(), "Try again") {
		t.Fatalf("invalid response = %d %s", recorder.Code, recorder.Body.String())
	}
	if calls != 0 {
		t.Fatalf("invalid ID made %d upstream calls", calls)
	}
}

func TestInsightDetailV2PreservesMissingAndUnavailableStates(t *testing.T) {
	tests := []struct {
		name       string
		upstream   int
		wantStatus int
		wantCopy   string
		wantRetry  bool
	}{
		{name: "missing", upstream: http.StatusNotFound, wantStatus: http.StatusNotFound, wantCopy: "Insight evidence was not found"},
		{name: "unavailable", upstream: http.StatusServiceUnavailable, wantStatus: http.StatusServiceUnavailable, wantCopy: "temporarily unavailable", wantRetry: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(test.upstream)
				_, _ = w.Write([]byte(`{"error":{"code":"test"}}`))
			}))
			defer server.Close()
			logger := testHomeLogger()
			client := gateway.New(gateway.Config{BaseURL: server.URL, APIKey: "test", Timeout: time.Second}, logger, context.Background())
			defer client.Stop()
			h := &Handlers{Logger: logger, Gateway: client, DataSource: "auto"}

			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, "/v2/insight/"+insightDetailTestID+"?network=testnet", nil)
			request.SetPathValue("id", insightDetailTestID)
			h.InsightDetailV2(recorder, request)

			body := recorder.Body.String()
			if recorder.Code != test.wantStatus || !strings.Contains(body, test.wantCopy) || strings.Contains(body, "42 failures") {
				t.Fatalf("response = %d %s", recorder.Code, body)
			}
			if strings.Contains(body, "Try again") != test.wantRetry {
				t.Fatalf("retry presence = %t, want %t", strings.Contains(body, "Try again"), test.wantRetry)
			}
		})
	}
}

func TestInsightDetailV2FailsClosedOnCorruptedPacket(t *testing.T) {
	packet := insightDetailTestPacket()
	packet.Subject.ID = "CBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBVQG"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(packet)
	}))
	defer server.Close()
	logger := testHomeLogger()
	client := gateway.New(gateway.Config{BaseURL: server.URL, APIKey: "test", Timeout: time.Second}, logger, context.Background())
	defer client.Stop()
	h := &Handlers{Logger: logger, Gateway: client, DataSource: "auto"}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v2/insight/"+insightDetailTestID+"?network=testnet", nil)
	request.SetPathValue("id", insightDetailTestID)
	h.InsightDetailV2(recorder, request)

	if recorder.Code != http.StatusServiceUnavailable || !strings.Contains(recorder.Body.String(), "could not verify this insight") || strings.Contains(recorder.Body.String(), "42 failures") {
		t.Fatalf("corrupted response = %d %s", recorder.Code, recorder.Body.String())
	}
}

func insightDetailTestPacket() gateway.HomeInsightDetailResponse {
	minimumObserved := 3.0
	caveats := []gateway.HomeInsightCaveat{{Code: "contributor_distribution_truncated", Field: "contributors", Retryable: false}}
	operation := 0
	return gateway.HomeInsightDetailResponse{
		HomeSummaryInsight: gateway.HomeSummaryInsight{
			InsightID:       insightDetailTestID,
			Network:         "testnet",
			Type:            "failure_spike",
			EvidenceVersion: "home_insight_evidence_v1",
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
		},
		Contributors: []gateway.HomeInsightContribution{
			{Dimension: "function", Rank: 1, Kind: "function", Key: "swap", Count: 38, DenominatorName: "subject_failure_count", DenominatorValue: 42, Share: 38.0 / 42.0, FirstLedger: 3794701, LastLedger: 3795410},
			{Dimension: "result_code", Rank: 1, Kind: "result_code", Key: "HOST_FUNCTION_TRAPPED", Count: 31, DenominatorName: "subject_failure_count", DenominatorValue: 42, Share: 31.0 / 42.0, FirstLedger: 3794701, LastLedger: 3795402},
		},
		Samples: []gateway.HomeInsightSample{{SampleKind: "failed_transaction", Rank: 1, LedgerSequence: 3794701, TransactionHash: strings.Repeat("a", 64), OperationIndex: &operation, ContractID: "CAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAD2KM", FunctionName: "swap", ResultCode: "HOST_FUNCTION_TRAPPED", SelectionMethod: "highest_impact_then_earliest_latest"}},
	}
}
