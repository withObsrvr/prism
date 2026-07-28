package txoutcome

import (
	"strings"
	"testing"

	"github.com/withObsrvr/prism/internal/gateway"
)

func TestInterpretFailedPayment(t *testing.T) {
	operationIndex := 0
	packet := &gateway.TransactionOutcome{
		EvidenceVersion: "transaction_outcome_v1",
		Status:          "ready",
		Outcome:         "failed",
		Failure: &gateway.TransactionFailureEvidence{
			Status:         "ready",
			Phase:          "operation_execution",
			Scope:          "operation",
			NormalizedCode: "payment_no_trust",
			OperationIndex: &operationIndex,
			OperationType:  "PAYMENT",
		},
		Operations: []gateway.TransactionOutcomeOperation{{OperationIndex: 0, OperationType: "PAYMENT", ExecutionOutcome: "failed"}},
	}

	got := Interpret(packet)
	if got.Heading != "Payment failed" || got.ReasonLabel != "Destination does not trust the asset" || got.OperationNumber != 1 {
		t.Fatalf("interpretation = %+v", got)
	}
	if !strings.Contains(got.Summary, "no trustline") || !got.ReasonAvailable {
		t.Fatalf("summary = %q, available=%t", got.Summary, got.ReasonAvailable)
	}
}

func TestInterpretMultiOperationRollback(t *testing.T) {
	operationIndex := 1
	packet := &gateway.TransactionOutcome{
		EvidenceVersion: "transaction_outcome_v1",
		Status:          "ready",
		Outcome:         "failed",
		Failure: &gateway.TransactionFailureEvidence{
			Status:         "ready",
			Phase:          "operation_execution",
			Scope:          "operation",
			NormalizedCode: "payment_no_destination",
			OperationIndex: &operationIndex,
			OperationType:  "PAYMENT",
		},
		Operations: []gateway.TransactionOutcomeOperation{
			{OperationIndex: 0, OperationType: "MANAGE_DATA", ExecutionOutcome: "succeeded", AppliedToLedger: false},
			{OperationIndex: 1, OperationType: "PAYMENT", ExecutionOutcome: "failed", AppliedToLedger: false},
		},
	}

	got := Interpret(packet)
	if got.RolledBackOperations != 1 || !strings.Contains(got.Summary, "One earlier operation executed successfully") || !strings.Contains(got.Summary, "was not applied") {
		t.Fatalf("rollback interpretation = %+v", got)
	}
}

func TestInterpretSorobanInvocation(t *testing.T) {
	operationIndex := 0
	packet := &gateway.TransactionOutcome{
		EvidenceVersion: "transaction_outcome_v1",
		Status:          "partial",
		Outcome:         "failed",
		Failure: &gateway.TransactionFailureEvidence{
			Status:         "ready",
			Phase:          "soroban_host",
			Scope:          "host_function",
			NormalizedCode: "invoke_host_function_trapped",
			OperationIndex: &operationIndex,
			OperationType:  "INVOKE_HOST_FUNCTION",
		},
		PrimaryInvocation: &gateway.TransactionPrimaryInvocation{
			FunctionName: "update_state",
			ContractID:   "CAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABSC4",
			Arguments: []gateway.DecodedScVal{
				{Type: "address", Value: "GACCOUNT", Display: "GACC...OUNT"},
				{Type: "u32", Value: float64(7), Display: "7"},
			},
		},
	}

	got := Interpret(packet)
	if got.Heading != "update_state() failed" || got.PhaseLabel != "Soroban host execution" || got.ReasonLabel != "Contract execution trapped" {
		t.Fatalf("Soroban interpretation = %+v", got)
	}
	if len(got.ArgumentLabels) != 2 || got.ArgumentLabels[0] != "GACC...OUNT" {
		t.Fatalf("arguments = %#v", got.ArgumentLabels)
	}
}

func TestInterpretPartialReasonDoesNotGuess(t *testing.T) {
	packet := &gateway.TransactionOutcome{
		EvidenceVersion: "transaction_outcome_v1",
		Status:          "partial",
		Outcome:         "failed",
		Failure: &gateway.TransactionFailureEvidence{
			Status:         "partial",
			Phase:          "operation_execution",
			Scope:          "transaction",
			NormalizedCode: "transaction_failed",
		},
	}

	got := Interpret(packet)
	if got.ReasonAvailable || got.ReasonLabel != "Reason unresolved" || !strings.Contains(got.Summary, "does not identify a more specific reason") {
		t.Fatalf("partial interpretation = %+v", got)
	}
}

func TestInterpretTransactionValidationFailure(t *testing.T) {
	packet := &gateway.TransactionOutcome{
		EvidenceVersion: "transaction_outcome_v1",
		Status:          "ready",
		Outcome:         "failed",
		Failure: &gateway.TransactionFailureEvidence{
			Status:         "ready",
			Phase:          "transaction_validation",
			Scope:          "transaction",
			NormalizedCode: "insufficient_fee",
		},
		Operations: []gateway.TransactionOutcomeOperation{{ExecutionOutcome: "not_executed"}},
	}

	got := Interpret(packet)
	if got.Heading != "Transaction rejected" || got.NotExecutedOperations != 1 || got.ReasonLabel != "Insufficient fee" || !strings.Contains(got.Summary, "One operation did not execute") {
		t.Fatalf("validation interpretation = %+v", got)
	}
}
