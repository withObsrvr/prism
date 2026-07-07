package handlers

import (
	"testing"

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
