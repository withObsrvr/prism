package handlers

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
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

func TestBuildLedgerDetailDataClassifiesTxsFromOperations(t *testing.T) {
	const (
		classicHash = "4f4fd9bbb34ba918d6c89320a767083b99d40f7044f5296caa91d7bdec43befe"
		sorobanHash = "2c35288cc669ba507a3aeacce3b3db75dee34b5b9645ef36a869f8335f037f80"
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/lake/v1/testnet/api/v1/silver/ledger/3630463/full" {
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

	if got := data.Transactions[0]; got.Hash != sorobanHash || got.Kind != "soroban" || got.OpType != "invoke" || got.OpColor != "violet" {
		t.Fatalf("soroban tx mapped unexpectedly: %+v", got)
	}
	if got := data.Transactions[1]; got.Hash != classicHash || got.Kind != "classic" || got.OpType != "multi" || got.OpColor != "gray" {
		t.Fatalf("classic multi-op tx mapped unexpectedly: %+v", got)
	}
}
