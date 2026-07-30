package gateway

import (
	"encoding/json"
	"testing"
)

func TestHomeSummaryDecodesVersionedInsightFacts(t *testing.T) {
	const payload = `{
  "network":"testnet",
  "components":{"insights":{"status":"ready"}},
  "insights":[{
    "insight_id":"hiev1_wwnNOzs1woMnG3W4aRHIShks_8_e0A70A62b6lJm7IE",
    "network":"testnet","type":"failure_spike","evidence_version":"home_insight_evidence_v1",
    "definition":{"rule_id":"contract_failure_spike","rule_version":"1","comparison_method":"rolling_7d_median_prior_complete_hour","minimum_observed":3,"minimum_ratio":3},
    "subject":{"kind":"contract","id":"CAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAD2KM","identity":{"display_name":"Example Router","kind":"protocol_contract","verification_status":"inferred","source":"semantic_contract_registry"}},
    "observed":{"value":42,"window_start":"2026-07-25T14:00:00Z","window_end":"2026-07-25T15:00:00Z","first_ledger":3794701,"last_ledger":3795424,"source_ledger":3795424},
    "baseline":{"value":7,"window_start":"2026-07-18T14:00:00Z","window_end":"2026-07-25T14:00:00Z","complete_hour_count":168,"zero_baseline_policy":"omit_ratio_insight"},
    "ratio":6,
    "facts":{"kind":"failure_spike","attempt_count":110,"success_count":68,"failure_count":42,"distinct_transaction_count":42,"distinct_caller_count":19,"network_failure_count":77,"subject_failure_share":0.5454545454545454},
    "primary_contributor":{"dimension":"function","kind":"function","key":"swap","count":38,"denominator_name":"subject_failure_count","denominator_value":42,"share":0.9047619047619048,"first_ledger":3794701,"last_ledger":3795410},
    "evidence_locator":{"kind":"contract_invocations","contract_id":"CAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAD2KM","ledger_start":3794701,"ledger_end":3795424,"status":"failed"},
    "evidence_count":42,"status":"ready","caveats":[],
    "provenance":{"sources":["serving.sv_home_insight_history"],"complete_through_ledger":3795424,"updated_at":"2026-07-25T15:00:05Z"}
  }]
}`
	var summary HomeSummaryResponse
	if err := json.Unmarshal([]byte(payload), &summary); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if len(summary.Insights) != 1 {
		t.Fatalf("insights = %d", len(summary.Insights))
	}
	item := summary.Insights[0]
	if item.EvidenceVersion != "home_insight_evidence_v1" || item.Facts == nil || item.Facts.Failure == nil || item.Facts.Failure.FailureCount != 42 {
		t.Fatalf("versioned failure facts were not typed: %+v", item)
	}
	if item.Subject.Identity == nil || item.Subject.Identity.DisplayName != "Example Router" || item.EvidenceLocator == nil || item.EvidenceLocator.Status != "failed" {
		t.Fatalf("identity or locator was not decoded: %+v", item)
	}
}
