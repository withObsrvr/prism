package gateway

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

const testHomeInsightID = "hiev1_wwnNOzs1woMnG3W4aRHIShks_8_e0A70A62b6lJm7IE"

func TestGetHomeInsightDecodesAndCachesDetailPacket(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.URL.Path != "/lake/v1/testnet/api/v1/home/insights/"+testHomeInsightID {
			t.Errorf("path = %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, testHomeInsightDetailJSON)
	}))
	defer server.Close()

	client := New(Config{BaseURL: server.URL, APIKey: "test", Timeout: time.Second}, slog.New(slog.NewTextHandler(io.Discard, nil)), context.Background())
	defer client.Stop()

	for range 2 {
		packet, err := client.GetHomeInsight(context.Background(), "testnet", testHomeInsightID)
		if err != nil {
			t.Fatalf("GetHomeInsight() error = %v", err)
		}
		if packet.Facts == nil || packet.Facts.Failure == nil || len(packet.Contributors) != 1 || len(packet.Samples) != 1 {
			t.Fatalf("detail packet = %+v", packet)
		}
	}
	if calls.Load() != 1 {
		t.Fatalf("upstream calls = %d, want 1", calls.Load())
	}
}

func TestGetHomeInsightRejectsRouteIdentityMismatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, testHomeInsightDetailJSON)
	}))
	defer server.Close()

	client := New(Config{BaseURL: server.URL, APIKey: "test", Timeout: time.Second}, slog.New(slog.NewTextHandler(io.Discard, nil)), context.Background())
	defer client.Stop()

	if _, err := client.GetHomeInsight(context.Background(), "mainnet", testHomeInsightID); err == nil {
		t.Fatal("GetHomeInsight() accepted a packet for the wrong network")
	}
}

func TestGetHomeInsightRejectsMalformedIDWithoutCallingGateway(t *testing.T) {
	if ValidHomeInsightID("not-an-insight") {
		t.Fatal("malformed insight ID was accepted")
	}
	client := New(Config{BaseURL: "http://127.0.0.1:1", APIKey: "test", Timeout: time.Second}, slog.New(slog.NewTextHandler(io.Discard, nil)), context.Background())
	defer client.Stop()
	if _, err := client.GetHomeInsight(context.Background(), "testnet", "not-an-insight"); err == nil {
		t.Fatal("GetHomeInsight() accepted a malformed insight ID")
	}
}

func TestGatewayDecodesEvidenceV2AndEvaluationFields(t *testing.T) {
	if !ValidHomeInsightID("hiev2_NykqrnkhWFaXhSkKMWK1_nk6MHj6Smy1n7ovYgM3Htc") {
		t.Fatal("frozen evidence-v2 ID was rejected")
	}
	var response HomeSummaryResponse
	raw := `{
		"network":"testnet",
		"components":{"insights":{"status":"empty"}},
		"insights":[],
		"recent_insights":[{"insight_id":"hiev2_NykqrnkhWFaXhSkKMWK1_nk6MHj6Smy1n7ovYgM3Htc","network":"testnet","type":"successful_activity_growth","family":"activity","direction":"positive","severity":"high","facts":{"kind":"successful_activity_growth","included_transaction_count":6500,"successful_transaction_count":6400,"failed_transaction_count":100}}],
		"insight_evaluation":{"evidence_version":"home_insight_evaluation_v1","registry_version":"home_insight_detector_registry_v1","status":"ready","rules":[],"caveats":[],"provenance":{"sources":[]}},
		"insight_delivery":{"mode":"current","max_age_seconds":21600,"projection_lag_seconds":41,"projection_ledger_lag":0}
	}`
	if err := json.Unmarshal([]byte(raw), &response); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if response.InsightEvaluation == nil || response.InsightDelivery == nil || response.RecentInsights == nil || len(*response.RecentInsights) != 1 {
		t.Fatalf("new summary fields were dropped: %+v", response)
	}
	recent := (*response.RecentInsights)[0]
	if recent.Facts == nil || recent.Facts.Growth == nil || recent.Family != "activity" || recent.Direction != "positive" || recent.Severity != "high" {
		t.Fatalf("evidence-v2 semantics were not decoded: %+v", recent)
	}
}

const testHomeInsightDetailJSON = `{
  "insight_id":"hiev1_wwnNOzs1woMnG3W4aRHIShks_8_e0A70A62b6lJm7IE",
  "network":"testnet",
  "type":"failure_spike",
  "evidence_version":"home_insight_evidence_v1",
  "definition":{"rule_id":"contract_failure_spike","rule_version":"1","comparison_method":"rolling_7d_median_prior_complete_hour","minimum_observed":3,"minimum_ratio":3},
  "subject":{"kind":"contract","id":"CAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAD2KM","identity":{"display_name":"Example Router","kind":"protocol_contract","verification_status":"inferred","source":"semantic_contract_registry"}},
  "observed":{"value":42,"window_start":"2026-07-25T14:00:00Z","window_end":"2026-07-25T15:00:00Z","first_ledger":3794701,"last_ledger":3795424,"source_ledger":3795424},
  "baseline":{"value":7,"window_start":"2026-07-18T14:00:00Z","window_end":"2026-07-25T14:00:00Z","complete_hour_count":168,"zero_baseline_policy":"omit_ratio_insight"},
  "ratio":6,
  "facts":{"kind":"failure_spike","attempt_count":110,"success_count":68,"failure_count":42,"distinct_transaction_count":42,"distinct_caller_count":19,"network_failure_count":77,"subject_failure_share":0.5454545454545454},
  "primary_contributor":{"dimension":"function","kind":"function","key":"swap","count":38,"denominator_name":"subject_failure_count","denominator_value":42,"share":0.9047619047619048,"first_ledger":3794701,"last_ledger":3795410},
  "evidence_locator":{"kind":"contract_invocations","contract_id":"CAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAD2KM","ledger_start":3794701,"ledger_end":3795424,"status":"failed"},
  "evidence_count":42,"status":"ready","caveats":[],
  "contributors":[{"dimension":"function","rank":1,"kind":"function","key":"swap","count":38,"denominator_name":"subject_failure_count","denominator_value":42,"share":0.9047619047619048,"first_ledger":3794701,"last_ledger":3795410}],
  "samples":[{"sample_kind":"failed_transaction","rank":1,"ledger_sequence":3794701,"transaction_hash":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","operation_index":0,"contract_id":"CAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAD2KM","function_name":"swap","result_code":"HOST_FUNCTION_TRAPPED","selection_method":"highest_impact_then_earliest_latest"}],
  "provenance":{"sources":["serving.sv_home_insight_history"],"complete_through_ledger":3795424,"updated_at":"2026-07-25T15:00:05Z"}
}`
