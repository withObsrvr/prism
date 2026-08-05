package handlers

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/withObsrvr/prism/internal/gateway"
)

func TestTxReceiptFromDecodedOmitsUnknownSemanticSequenceAndMaxFee(t *testing.T) {
	receipt := txReceiptFromDecoded(gateway.DecodedTransaction{
		TxHash:         "abc123",
		LedgerSequence: 123,
		ClosedAt:       "2026-01-02T03:04:05Z",
		Successful:     true,
		Fee:            100,
		OperationCount: 1,
	})

	tx := receipt.Semantic.Transaction
	if tx.AccountSequence != nil {
		t.Fatalf("AccountSequence = %v, want nil when decoded transaction does not include it", *tx.AccountSequence)
	}
	if tx.MaxFee != nil {
		t.Fatalf("MaxFee = %v, want nil when decoded transaction does not include it", *tx.MaxFee)
	}
}

func TestTransactionOperationStatusUsesExecutionAndApplicationEvidence(t *testing.T) {
	outcome := &gateway.TransactionOutcome{Operations: []gateway.TransactionOutcomeOperation{
		{OperationIndex: 0, ExecutionOutcome: "succeeded", AppliedToLedger: false},
		{OperationIndex: 1, ExecutionOutcome: "failed", AppliedToLedger: false},
		{OperationIndex: 2, ExecutionOutcome: "not_executed", AppliedToLedger: false},
		{OperationIndex: 3, ExecutionOutcome: "succeeded", AppliedToLedger: true},
	}}
	tests := []struct {
		position int
		want     string
	}{
		{0, "Executed, not applied"},
		{1, "Failed"},
		{2, "Not executed"},
		{3, "Success"},
		{4, "Unknown"},
	}
	for _, test := range tests {
		if got := transactionOperationStatus(outcome, test.position, false); got != test.want {
			t.Errorf("position %d = %q, want %q", test.position, got, test.want)
		}
	}
	if got := transactionOperationStatus(nil, 0, true); got != "Success" {
		t.Errorf("successful receipt fallback = %q, want Success", got)
	}
	if got := transactionOperationStatus(nil, 0, false); got != "Unknown" {
		t.Errorf("failed receipt fallback = %q, want Unknown", got)
	}
}

func TestBuildTxReceiptDataLetsE1AOwnOutcomeAndOperationStates(t *testing.T) {
	hash := strings.Repeat("a", 64)
	var receiptCalls, outcomeCalls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/lake/v1/testnet/api/v1/silver/tx/" + hash + "/receipt":
			receiptCalls++
			_, _ = io.WriteString(w, `{
				"tx_hash":"`+hash+`",
				"ledger_sequence":123,
				"created_at":"2026-07-28T12:00:00Z",
				"source_account":"GAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAWHF",
				"successful":true,
				"operation_count":3,
				"tx_type":"multi_op",
				"full":{
					"tx_hash":"`+hash+`",
					"created_at":"2026-07-28T12:00:00Z",
					"successful":true,
					"source_account":"GAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAWHF",
					"ledger_sequence":123,
					"fee":300,
					"operations":[
						{"index":0,"type_name":"manage_data","source_account":"GAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAWHF"},
						{"index":1,"type_name":"payment","source_account":"GAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAWHF"},
						{"index":2,"type_name":"payment","source_account":"GAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAWHF"}
					]
				}
			}`)
		case "/lake/v1/testnet/api/v1/silver/tx/" + hash + "/failure-evidence":
			outcomeCalls++
			_, _ = io.WriteString(w, `{
				"evidence_version":"transaction_outcome_v1",
				"status":"ready",
				"network":"testnet",
				"transaction_hash":"`+hash+`",
				"ledger_sequence":123,
				"outcome":"failed",
				"applied_to_ledger":false,
				"transaction_result":{"normalized_code":"transaction_failed","raw_code":"TransactionResultCodeTxFailed","source":"transaction_result_xdr"},
				"failure":{"status":"ready","phase":"operation_execution","scope":"operation","normalized_code":"payment_underfunded","raw_code":"PaymentResultCodePaymentUnderfunded","source":"operation_result_xdr","operation_index":1,"operation_type":"PAYMENT","transaction_raw_code":"TransactionResultCodeTxFailed"},
				"operations":[
					{"operation_index":0,"operation_type":"MANAGE_DATA","execution_outcome":"succeeded","applied_to_ledger":false},
					{"operation_index":1,"operation_type":"PAYMENT","execution_outcome":"failed","applied_to_ledger":false},
					{"operation_index":2,"operation_type":"PAYMENT","execution_outcome":"not_executed","applied_to_ledger":false}
				],
				"components":{},
				"locators":[],
				"provenance":{"transaction_source_ledger":123,"complete_through_ledger":123,"sources":["transaction_result_xdr"],"resolved_at":"2026-07-28T12:00:01Z"}
			}`)
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	gw := gateway.New(gateway.Config{BaseURL: server.URL, APIKey: "test", Timeout: time.Second}, slog.New(slog.NewTextHandler(io.Discard, nil)), context.Background())
	defer gw.Stop()
	h := &Handlers{Gateway: gw, Logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
	req := httptest.NewRequest(http.MethodGet, "/v2/tx/"+hash+"?network=testnet", nil)
	data, err := h.buildTxReceiptData(req, "testnet", hash, gateway.ShortHash(hash))
	if err != nil {
		t.Fatalf("buildTxReceiptData: %v", err)
	}
	if receiptCalls != 1 || outcomeCalls != 1 {
		t.Fatalf("calls receipt=%d outcome=%d, want one each", receiptCalls, outcomeCalls)
	}
	if data.Status != "failed" || data.OutcomeEvidence == nil {
		t.Fatalf("E1A did not own the enclosing outcome: status=%q outcome=%v", data.Status, data.OutcomeEvidence)
	}
	wantStatuses := []string{"Executed, not applied", "Failed", "Not executed"}
	if len(data.Operations) != len(wantStatuses) {
		t.Fatalf("operations=%d, want %d", len(data.Operations), len(wantStatuses))
	}
	for i, want := range wantStatuses {
		if got := data.Operations[i].Status; got != want {
			t.Errorf("operation %d status=%q, want %q", i, got, want)
		}
	}
}

func TestBuildLedgerDetailDataClassifiesTxsFromOperations(t *testing.T) {
	const (
		classicHash = "4f4fd9bbb34ba918d6c89320a767083b99d40f7044f5296caa91d7bdec43befe"
		sorobanHash = "2c35288cc669ba507a3aeacce3b3db75dee34b5b9645ef36a869f8335f037f80"
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/lake/v1/testnet/api/v1/silver/ledgers/3630463/full" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{
			"ledger_sequence":3630463,
			"ledger":{
				"sequence":3630463,
				"ledger_hash":"7fd270989c",
				"previous_ledger_hash":"2584488d1c",
				"closed_at":"2026-07-16T02:32:06Z",
				"successful_tx_count":2,
				"failed_tx_count":0,
				"operation_count":4,
				"transaction_count":2,
				"protocol_version":27,
				"total_coins":100000000000,
				"base_fee":100,
				"max_tx_set_size":1000,
				"soroban_op_count":1,
				"total_fee_charged":1058934,
				"contract_events_count":1
			},
			"transactions":[
				{
					"transaction_hash":"`+sorobanHash+`",
					"source_account":"GBQHXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXQLIQ",
					"ledger_sequence":3630463,
					"max_fee":1058634,
					"operation_count":1,
					"successful":true
				},
				{
					"transaction_hash":"`+classicHash+`",
					"source_account":"GBBLXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXDG2K",
					"ledger_sequence":3630463,
					"max_fee":300,
					"operation_count":3,
					"successful":true
				}
			],
			"operations":[
				{"transaction_hash":"`+sorobanHash+`","operation_id":1,"ledger_sequence":3630463,"source_account":"GBQH","type_name":"INVOKE_HOST_FUNCTION","tx_successful":true,"tx_fee_charged":1058634,"is_soroban_op":true},
				{"transaction_hash":"`+classicHash+`","operation_id":2,"ledger_sequence":3630463,"source_account":"GBBL","type_name":"SET_TRUST_LINE_FLAGS","tx_successful":true,"tx_fee_charged":300,"is_soroban_op":false},
				{"transaction_hash":"`+classicHash+`","operation_id":3,"ledger_sequence":3630463,"source_account":"GBBL","type_name":"PAYMENT","tx_successful":true,"tx_fee_charged":300,"is_soroban_op":false},
				{"transaction_hash":"`+classicHash+`","operation_id":4,"ledger_sequence":3630463,"source_account":"GBBL","type_name":"MANAGE_BUY_OFFER","tx_successful":true,"tx_fee_charged":300,"is_soroban_op":false}
			],
			"generated_at":"2026-07-16T02:32:07Z"
		}`)
	}))
	defer server.Close()

	gw := gateway.New(gateway.Config{BaseURL: server.URL, APIKey: "test", Timeout: time.Second}, slog.New(slog.NewTextHandler(io.Discard, nil)), context.Background())
	defer gw.Stop()
	h := &Handlers{Gateway: gw, Logger: slog.New(slog.NewTextHandler(io.Discard, nil))}

	req := httptest.NewRequest(http.MethodGet, "/v2/ledger/3630463?network=testnet", nil)
	data, err := h.buildLedgerDetailData(req, "testnet", "3630463")
	if err != nil {
		t.Fatalf("buildLedgerDetailData error = %v", err)
	}
	if len(data.Transactions) != 2 {
		t.Fatalf("transactions = %d, want 2", len(data.Transactions))
	}

	if got := data.Transactions[0]; got.Hash != sorobanHash || got.Kind != "soroban" || got.OpType != "invoke" || got.Family != gateway.OpFamilyContract {
		t.Fatalf("soroban tx mapped unexpectedly: %+v", got)
	}
	if got := data.Transactions[1]; got.Hash != classicHash || got.Kind != "classic" || got.OpType != "multi" || got.Family != gateway.OpFamilyOther {
		t.Fatalf("classic multi-op tx mapped unexpectedly: %+v", got)
	}
}

func TestBuildLedgerDetailDataRefetchesIncompleteCompositeTransactions(t *testing.T) {
	hashes := []string{"tx-one", "tx-two", "tx-three", "tx-four"}
	var transactionCalls int

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/lake/v1/testnet/api/v1/silver/ledgers/1073104/full":
			_, _ = io.WriteString(w, `{
				"ledger_sequence":1073104,
				"ledger":{
					"sequence":1073104,
					"closed_at":"2026-02-17T21:31:37Z",
					"successful_tx_count":2,
					"failed_tx_count":2,
					"operation_count":2,
					"transaction_count":4,
					"protocol_version":25,
					"base_fee":100
				},
				"partial":true,
				"warnings":["transactions data unavailable"]
			}`)
		case "/lake/v1/testnet/api/v1/bronze/transactions":
			transactionCalls++
			if got := r.URL.Query().Get("start"); got != "1073104" {
				t.Errorf("start = %q, want 1073104", got)
			}
			if got := r.URL.Query().Get("end"); got != "1073104" {
				t.Errorf("end = %q, want 1073104", got)
			}
			if got := r.URL.Query().Get("limit"); got != "50" {
				t.Errorf("limit = %q, want 50", got)
			}
			_, _ = io.WriteString(w, `{
				"count":4,
				"start":1073104,
				"end":1073104,
				"transactions":[
					{"transaction_hash":"tx-one","ledger_sequence":1073104,"operation_count":1,"successful":true},
					{"transaction_hash":"tx-two","ledger_sequence":1073104,"operation_count":1,"successful":false},
					{"transaction_hash":"tx-three","ledger_sequence":1073104,"operation_count":1,"successful":false},
					{"transaction_hash":"tx-four","ledger_sequence":1073104,"operation_count":1,"successful":true}
				]
			}`)
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	gw := gateway.New(gateway.Config{BaseURL: server.URL, APIKey: "test", Timeout: time.Second}, slog.New(slog.NewTextHandler(io.Discard, nil)), context.Background())
	defer gw.Stop()
	h := &Handlers{Gateway: gw, Logger: slog.New(slog.NewTextHandler(io.Discard, nil))}

	req := httptest.NewRequest(http.MethodGet, "/v2/ledger/1073104?network=testnet", nil)
	data, err := h.buildLedgerDetailData(req, "testnet", "1073104")
	if err != nil {
		t.Fatalf("buildLedgerDetailData error = %v", err)
	}
	if transactionCalls != 1 {
		t.Fatalf("transaction endpoint calls = %d, want 1", transactionCalls)
	}
	if len(data.Transactions) != len(hashes) {
		t.Fatalf("transactions = %d, want %d", len(data.Transactions), len(hashes))
	}
	for i, hash := range hashes {
		if data.Transactions[i].Hash != hash {
			t.Fatalf("transaction %d hash = %q, want %q", i, data.Transactions[i].Hash, hash)
		}
	}
}
