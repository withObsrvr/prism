package intent

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/withObsrvr/prism/internal/gateway"
)

func TestDeterministicSearchIntentsUseOnlyGatewayEvidence(t *testing.T) {
	hash := strings.Repeat("a", 64)
	contractID := "C" + strings.Repeat("A", 55)
	recentClosedAt := time.Now().Add(-5 * time.Minute).UTC().Format(time.RFC3339)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/lake/v1/testnet/api/v1/silver/tx/" + hash + "/failure-evidence":
			_, _ = io.WriteString(w, `{"evidence_version":"transaction_outcome_v1","status":"partial","network":"testnet","transaction_hash":"`+hash+`","ledger_sequence":42,"outcome":"failed","applied_to_ledger":false,"transaction_result":{"normalized_code":"transaction_failed","raw_code":"TransactionResultCodeTxFailed","source":"transaction_result_xdr"},"failure":{"status":"ready","phase":"soroban_host","scope":"host_function","normalized_code":"invoke_host_function_trapped","raw_code":"InvokeHostFunctionResultCodeInvokeHostFunctionTrapped","source":"soroban_host_result_xdr","operation_index":1,"operation_type":"INVOKE_HOST_FUNCTION","transaction_raw_code":"TransactionResultCodeTxFailed"},"operations":[{"operation_index":0,"operation_type":"MANAGE_DATA","execution_outcome":"succeeded","applied_to_ledger":false},{"operation_index":1,"operation_type":"INVOKE_HOST_FUNCTION","execution_outcome":"failed","applied_to_ledger":false}],"primary_invocation":{"operation_index":1,"contract_id":"`+contractID+`","function_name":"swap","arguments":[{"type":"address","value":"GABC","display":"GABC"},{"type":"u32","value":7,"display":"7"}],"decode_status":"ready","execution_outcome":"failed","applied_to_ledger":false,"identity":{"kind":"contract","id":"`+contractID+`","verification_status":"unverified","source":"transaction_envelope_xdr"}},"components":{"transaction_result":{"status":"ready"},"diagnostics":{"status":"partial"}},"locators":[],"provenance":{"transaction_source_ledger":42,"complete_through_ledger":42,"sources":["transaction_result_xdr"],"resolved_at":"2026-07-28T12:00:00Z"}}`)
		case "/lake/v1/testnet/api/v1/silver/tx/" + hash + "/receipt":
			_, _ = io.WriteString(w, `{"tx_hash":"`+hash+`","ledger_sequence":42,"successful":false,"operation_count":2,"semantic":{"transaction":{"tx_hash":"`+hash+`","ledger_sequence":42,"successful":false},"classification":{"tx_type":"contract_call","confidence":"high"},"call_graph":[{"from":"GABC","to":"`+contractID+`","function":"swap","successful":false}]}}`)
		case "/lake/v1/testnet/api/v1/silver/contracts/" + contractID + "/analytics":
			_, _ = io.WriteString(w, `{"contract_id":"`+contractID+`","stats":{"total_calls_as_caller":20,"total_calls_as_callee":80,"unique_callers":12},"top_functions":[{"name":"swap","count":60}],"daily_calls_7d":[{"date":"2026-07-27","count":9},{"date":"2026-07-28","count":11}]}`)
		case "/lake/v1/testnet/api/v1/silver/assets/XLM":
			_, _ = io.WriteString(w, `{"canonical_slug":"XLM","asset_code":"XLM","symbol":"XLM","holder_count":900,"transfers_24h":125,"unique_accounts_24h":44,"volume_24h":"12500.00"}`)
		case "/lake/v1/testnet/api/v1/silver/transactions/recent":
			_, _ = fmt.Fprintf(w, `{"latest_sequence":42,"count":2,"transactions":[{"tx_hash":"%s","ledger_sequence":42,"closed_at":"%s","successful":false,"summary":{"type":"transfer"}},{"tx_hash":"%s","ledger_sequence":41,"closed_at":"%s","successful":true,"summary":{"type":"transfer"}}]}`, hash, recentClosedAt, strings.Repeat("b", 64), recentClosedAt)
		case "/lake/v1/testnet/api/v1/home/summary":
			_, _ = io.WriteString(w, `{"components":{"ttl_attention":{"status":"ready"}},"hero":{"ttl":{"expiring_contract_count":1,"worst_remaining_hours":12}},"contracts_needing_attention":[{"contract_id":"`+contractID+`","remaining_hours":12,"remaining_human":"12 hours"}]}`)
		default:
			t.Errorf("unexpected intent endpoint %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	client := gateway.New(gateway.Config{BaseURL: server.URL, APIKey: "test", Timeout: time.Second}, slog.New(slog.NewTextHandler(io.Discard, nil)), context.Background())
	defer client.Stop()

	tests := []struct {
		query       string
		id          ID
		answerParts []string
	}{
		{"why did " + hash + " fail?", TransactionFailure, []string{"Soroban host stopped the contract invocation", "earlier operation executed successfully", "primary invocation was swap(GABC, 7)"}},
		{"How active is contract " + contractID + " today?", ContractActivity, []string{"20 calls across 2 returned daily buckets", "most observed function is swap"}},
		{"How active is XLM today?", AssetActivity, []string{"125 transfers", "44 unique accounts", "12500.00"}},
		{"Are any recent transfers failing?", RecentFailures, []string{"Among 2 transfer transactions", "1 failed"}},
		{"Which contracts are expiring this week?", ExpiringContracts, []string{"current TTL attention snapshot", "12 hours"}},
	}
	registry := DefaultRegistry()
	for _, test := range tests {
		t.Run(string(test.id), func(t *testing.T) {
			match, ok := registry.Match(test.query)
			if !ok || match.ID != test.id {
				t.Fatalf("Match(%q) = %+v, ok=%t", test.query, match, ok)
			}
			result, err := registry.Execute(context.Background(), Env{Gateway: client, Network: "testnet"}, match)
			if err != nil {
				t.Fatalf("Execute(%s): %v", test.id, err)
			}
			if !result.EvidenceAvailable {
				t.Errorf("%s did not mark fetched Gateway evidence available", test.id)
			}
			for _, part := range test.answerParts {
				if !strings.Contains(result.Answer, part) {
					t.Errorf("%s answer missing %q: %s", test.id, part, result.Answer)
				}
			}
			if len(result.Actions) == 0 {
				t.Errorf("%s result has no proof path", test.id)
			}
		})
	}
}

func TestTransactionFailureIntentFallsBackToReceiptBeforeE1AReachesNetwork(t *testing.T) {
	hash := strings.Repeat("c", 64)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/lake/v1/mainnet/api/v1/silver/tx/" + hash + "/failure-evidence":
			http.NotFound(w, r)
		case "/lake/v1/mainnet/api/v1/silver/tx/" + hash + "/receipt":
			_, _ = io.WriteString(w, `{"tx_hash":"`+hash+`","ledger_sequence":99,"successful":false,"operation_count":1}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()
	client := gateway.New(gateway.Config{BaseURL: server.URL, APIKey: "test", Timeout: time.Second}, slog.New(slog.NewTextHandler(io.Discard, nil)), context.Background())
	defer client.Stop()

	match, ok := DefaultRegistry().Match("why did " + hash + " fail?")
	if !ok {
		t.Fatal("transaction failure question did not match")
	}
	result, err := DefaultRegistry().Execute(context.Background(), Env{Gateway: client, Network: "mainnet"}, match)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(result.Answer, "receipt marks this transaction failed") || len(result.Warnings) < 2 {
		t.Fatalf("fallback result = %+v", result)
	}
}

func TestTTLQuestionDoesNotClaimCalendarWindow(t *testing.T) {
	match, ok := DefaultRegistry().Match("Which contracts are expiring this week?")
	if !ok {
		t.Fatal("TTL question did not match")
	}
	result, err := DefaultRegistry().Execute(context.Background(), Env{}, match)
	if err != nil {
		t.Fatalf("execute unavailable TTL: %v", err)
	}
	if strings.Contains(result.Answer, "this week") || strings.Contains(result.Answer, "next week") {
		t.Fatalf("TTL answer claimed a calendar window: %s", result.Answer)
	}
	if result.EvidenceAvailable {
		t.Fatal("TTL result marked evidence available without a Gateway")
	}
}

func TestContractActivityOmitsInconsistentAggregateStats(t *testing.T) {
	contractID := "C" + strings.Repeat("A", 55)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"contract_id":"`+contractID+`","stats":{"total_calls_as_caller":0,"total_calls_as_callee":0,"unique_callers":0},"top_functions":[{"name":"push","count":1038}],"daily_calls_7d":[]}`)
	}))
	defer server.Close()
	client := gateway.New(gateway.Config{BaseURL: server.URL, APIKey: "test", Timeout: time.Second}, slog.New(slog.NewTextHandler(io.Discard, nil)), context.Background())
	defer client.Stop()
	match, ok := DefaultRegistry().Match("How active is contract " + contractID + " today?")
	if !ok {
		t.Fatal("contract activity did not match")
	}
	result, err := DefaultRegistry().Execute(context.Background(), Env{Gateway: client, Network: "testnet"}, match)
	if err != nil {
		t.Fatalf("execute inconsistent contract analytics: %v", err)
	}
	if strings.Contains(result.Answer, "0 all-time observed calls") || !strings.Contains(result.Answer, "at least 1038 calls") || len(result.Warnings) == 0 {
		t.Fatalf("inconsistent analytics answer was not bounded: %+v", result)
	}
}
