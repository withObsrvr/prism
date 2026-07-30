package gateway

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestClientDecodesTransactionOutcomeV1(t *testing.T) {
	hash := strings.Repeat("a", 64)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got, want := r.URL.Path, "/lake/v1/testnet/api/v1/silver/tx/"+hash+"/failure-evidence"; got != want {
			t.Fatalf("path = %q, want %q", got, want)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{
  "evidence_version":"transaction_outcome_v1",
  "status":"partial",
  "network":"testnet",
  "transaction_hash":"`+hash+`",
  "ledger_sequence":3848081,
  "outcome":"failed",
  "applied_to_ledger":false,
  "transaction_result":{"normalized_code":"transaction_failed","raw_code":"TransactionResultCodeTxFailed","source":"transaction_result_xdr"},
  "failure":{"status":"ready","phase":"operation_execution","scope":"operation","normalized_code":"payment_no_trust","raw_code":"PaymentResultCodePaymentNoTrust","source":"operation_result_xdr","operation_index":1,"operation_type":"PAYMENT","transaction_raw_code":"TransactionResultCodeTxFailed"},
  "operations":[
    {"operation_index":0,"operation_type":"MANAGE_DATA","execution_outcome":"succeeded","applied_to_ledger":false,"result":{"normalized_code":"manage_data_success","raw_code":"ManageDataResultCodeManageDataSuccess","source":"operation_result_xdr"}},
    {"operation_index":1,"operation_type":"PAYMENT","execution_outcome":"failed","applied_to_ledger":false,"result":{"normalized_code":"payment_no_trust","raw_code":"PaymentResultCodePaymentNoTrust","source":"operation_result_xdr"}}
  ],
  "primary_invocation":{"operation_index":1,"contract_id":"CAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABSC4","function_name":"swap","arguments":[{"type":"address","value":"GAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAWHF","display":"GAAA...AWHF"}],"decode_status":"ready","execution_outcome":"failed","applied_to_ledger":false,"identity":{"kind":"contract","id":"CAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABSC4","verification_status":"unverified","source":"transaction_envelope_xdr"}},
  "components":{"transaction_result":{"status":"ready","source":"transaction_result_xdr"},"diagnostics":{"status":"partial","limitation":"coverage unavailable"}},
  "caveats":[{"code":"diagnostic_evidence_incomplete","message":"Diagnostic coverage is incomplete.","affects":["diagnostics"]}],
  "locators":[{"kind":"decoded_operation","href":"/api/v1/silver/tx/`+hash+`/decoded","operation_index":1}],
  "provenance":{"transaction_source_ledger":3848081,"complete_through_ledger":3848081,"sources":["transaction_result_xdr"],"resolved_at":"2026-07-28T12:00:00Z"}
}`)
	}))
	defer server.Close()

	client := New(Config{BaseURL: server.URL, APIKey: "test", Timeout: time.Second}, slog.New(slog.NewTextHandler(io.Discard, nil)), context.Background())
	defer client.Stop()

	outcome, err := client.GetTransactionOutcome(context.Background(), "testnet", hash)
	if err != nil {
		t.Fatalf("GetTransactionOutcome: %v", err)
	}
	if outcome.EvidenceVersion != "transaction_outcome_v1" || outcome.Outcome != "failed" || outcome.AppliedToLedger {
		t.Fatalf("outcome envelope = %+v", outcome)
	}
	if outcome.Failure == nil || outcome.Failure.NormalizedCode != "payment_no_trust" || outcome.Failure.OperationIndex == nil || *outcome.Failure.OperationIndex != 1 {
		t.Fatalf("failure = %+v", outcome.Failure)
	}
	if len(outcome.Operations) != 2 || outcome.Operations[0].ExecutionOutcome != "succeeded" || outcome.Operations[0].AppliedToLedger {
		t.Fatalf("operation rollback evidence = %+v", outcome.Operations)
	}
	if outcome.PrimaryInvocation == nil || outcome.PrimaryInvocation.FunctionName != "swap" || len(outcome.PrimaryInvocation.Arguments) != 1 {
		t.Fatalf("primary invocation = %+v", outcome.PrimaryInvocation)
	}
	if got := outcome.PrimaryInvocation.Arguments[0].Value; got != "GAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAWHF" {
		t.Fatalf("decoded argument value = %#v", got)
	}
	if !transactionOutcomeMayImprove(*outcome) {
		t.Fatal("partial diagnostics should keep a short cache TTL")
	}
}

func TestClientRejectsUnknownTransactionOutcomeVersion(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"evidence_version":"transaction_outcome_v2"}`)
	}))
	defer server.Close()

	client := New(Config{BaseURL: server.URL, APIKey: "test", Timeout: time.Second}, slog.New(slog.NewTextHandler(io.Discard, nil)), context.Background())
	defer client.Stop()

	_, err := client.GetTransactionOutcome(context.Background(), "testnet", strings.Repeat("b", 64))
	if err == nil || !strings.Contains(err.Error(), "unsupported transaction outcome evidence version") {
		t.Fatalf("error = %v", err)
	}
}

func TestTransactionOutcomeReadyPacketUsesImmutableCache(t *testing.T) {
	outcome := TransactionOutcome{
		EvidenceVersion: "transaction_outcome_v1",
		Status:          "ready",
		Components: map[string]TransactionComponent{
			"transaction_result": {Status: "ready"},
			"operations":         {Status: "ready"},
			"diagnostics":        {Status: "empty"},
			"call_graph":         {Status: "empty"},
		},
	}
	if transactionOutcomeMayImprove(outcome) {
		t.Fatal("complete packet should be treated as immutable")
	}
}

func TestClientNegativeCachesMissingTransactionOutcome(t *testing.T) {
	var calls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		http.Error(w, `{"error":"transaction not found"}`, http.StatusNotFound)
	}))
	defer server.Close()

	client := New(Config{BaseURL: server.URL, APIKey: "test", Timeout: time.Second}, slog.New(slog.NewTextHandler(io.Discard, nil)), context.Background())
	defer client.Stop()
	hash := strings.Repeat("c", 64)
	for i := 0; i < 2; i++ {
		if _, err := client.GetTransactionOutcome(context.Background(), "testnet", hash); err == nil {
			t.Fatalf("request %d unexpectedly succeeded", i+1)
		}
	}
	if calls != 1 {
		t.Fatalf("upstream calls = %d, want 1", calls)
	}
}
