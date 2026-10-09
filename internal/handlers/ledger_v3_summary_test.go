package handlers

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/withObsrvr/prism/internal/gateway"
	"github.com/withObsrvr/prism/internal/jev"
	"github.com/withObsrvr/prism/internal/summary"
)

type fakeLedgerSelector struct {
	selection jev.LedgerSelection
	err       error
	request   jev.LedgerSelectionRequest
}

type fakeShadowRecorder struct{ records []jev.LedgerShadowRecord }

func (f *fakeShadowRecorder) RecordLedgerShadow(record jev.LedgerShadowRecord) error {
	f.records = append(f.records, record)
	return nil
}

func (f *fakeLedgerSelector) SelectLedgerSummary(_ context.Context, request jev.LedgerSelectionRequest) (jev.LedgerSelection, error) {
	f.request = request
	return f.selection, f.err
}

func (f *fakeLedgerSelector) LedgerSelectorIdentity() string { return "test-shadow" }

func TestLedgerFactsAdapterBuildsConservativeSorobanEnvelope(t *testing.T) {
	evidence := completeLedgerSummaryEvidence()
	facts := ledgerFactsFromGateway(evidence)
	envelope, err := summary.BuildLedgerSummaryEnvelope(facts)
	if err != nil {
		t.Fatal(err)
	}

	if envelope.State.Bands.Evidence != summary.EvidenceComplete {
		t.Fatalf("evidence = %q", envelope.State.Bands.Evidence)
	}
	if envelope.State.Bands.Capacity.CPU != summary.PressureElevated {
		t.Fatalf("cpu = %q", envelope.State.Bands.Capacity.CPU)
	}
	if envelope.State.Bands.Capacity.Overall != summary.PressurePartial {
		t.Fatalf("overall capacity = %q", envelope.State.Bands.Capacity.Overall)
	}
	if envelope.State.Bands.Fees.Inclusion != summary.FeeUnavailable {
		t.Fatalf("Soroban median charged fee was treated as inclusion fee: %q", envelope.State.Bands.Fees.Inclusion)
	}
	if envelope.Selection.Lead != summary.InterpretationSorobanHeavy {
		t.Fatalf("lead = %q", envelope.Selection.Lead)
	}
	if envelope.BandVersion != summary.LedgerBandVersion || envelope.RegistryVersion != summary.LedgerInterpretationRegistryVersion {
		t.Fatalf("versions missing: %+v", envelope)
	}
}

func TestLedgerFactsAdapterTreatsSorobanCapacityAsNotApplicable(t *testing.T) {
	evidence := completeLedgerSummaryEvidence()
	evidence.Full.Ledger.TransactionCount = 1
	evidence.Full.Ledger.SuccessfulTxCount = 1
	evidence.Full.Ledger.OperationCount = 1
	evidence.Transactions = evidence.Transactions[:1]
	evidence.Transactions[0].Successful = true
	evidence.Operations = evidence.Operations[:1]
	evidence.Usage = &gateway.LedgerSoroban{LedgerSequence: 123, SorobanTxCount: 0}
	evidence.Full.Soroban = evidence.Usage
	evidence.Fees = &gateway.LedgerFees{LedgerSequence: 123, TxCount: 1, MedianFee: 100, TotalFees: 100}

	envelope, err := summary.BuildLedgerSummaryEnvelope(ledgerFactsFromGateway(evidence))
	if err != nil {
		t.Fatal(err)
	}
	if envelope.State.Bands.Capacity.Overall != summary.PressureNotApplicable {
		t.Fatalf("capacity = %q", envelope.State.Bands.Capacity.Overall)
	}
	if envelope.State.Bands.Fees.Inclusion != summary.FeeBase {
		t.Fatalf("classic inclusion fee = %q", envelope.State.Bands.Fees.Inclusion)
	}
	if envelope.Selection.Lead != summary.InterpretationAllSucceeded {
		t.Fatalf("lead = %q", envelope.Selection.Lead)
	}
	if containsCandidate(envelope.Candidates, summary.InterpretationEvidenceIncomplete) {
		t.Fatal("not-applicable Soroban capacity admitted evidence_incomplete")
	}
}

func TestLedgerFactsAdapterMarksBoundedTransactionRowsPartial(t *testing.T) {
	evidence := completeLedgerSummaryEvidence()
	evidence.Transactions = evidence.Transactions[:1]
	envelope, err := summary.BuildLedgerSummaryEnvelope(ledgerFactsFromGateway(evidence))
	if err != nil {
		t.Fatal(err)
	}
	if envelope.State.Bands.Evidence != summary.EvidencePartial || envelope.State.Bands.Outcomes != summary.OutcomePartial {
		t.Fatalf("bands = %+v", envelope.State.Bands)
	}
	if envelope.Selection.Lead != summary.InterpretationEvidenceIncomplete {
		t.Fatalf("lead = %q", envelope.Selection.Lead)
	}
	if containsCandidate(envelope.Candidates, summary.InterpretationAllSucceeded) {
		t.Fatal("partial rows admitted complete-success interpretation")
	}
}

func TestLedgerFactsAdapterClassifiesActivityAgainstVersionedBaseline(t *testing.T) {
	evidence := completeLedgerSummaryEvidence()
	samples := make([]summary.LedgerBaselineSample, summary.LedgerBaselineMinimumSamples)
	for i := range samples {
		samples[i] = summary.LedgerBaselineSample{Sequence: int64(100 + i), Transactions: 1, Operations: 1}
	}
	evidence.Baselines = summary.BuildLedgerBaselines(123, samples)

	envelope, err := summary.BuildLedgerSummaryEnvelope(ledgerFactsFromGateway(evidence))
	if err != nil {
		t.Fatal(err)
	}
	if envelope.BaselineVersion != summary.LedgerBaselineVersion {
		t.Fatalf("baseline version = %q", envelope.BaselineVersion)
	}
	if envelope.State.Bands.Activity != summary.ActivityBusy {
		t.Fatalf("activity = %q, metrics = %+v", envelope.State.Bands.Activity, envelope.State.Metrics.Activity)
	}
	if envelope.State.Bands.StateChange.Volume != summary.ChangeNotEvaluated {
		t.Fatalf("state changes used absent history: %q", envelope.State.Bands.StateChange.Volume)
	}
}

func TestLedgerSummaryCacheHitNeedsNoGatewayCalls(t *testing.T) {
	evidence := completeLedgerSummaryEvidence()
	want, err := summary.BuildLedgerSummaryEnvelope(ledgerFactsFromGateway(evidence))
	if err != nil {
		t.Fatal(err)
	}
	cache := summary.NewLedgerSummaryCache()
	cache.Set(summary.NewLedgerSummaryCacheKey("testnet", 123), want)
	h := &Handlers{LedgerSummaries: cache}

	got, err := h.buildLedgerV3SummaryEnvelope(context.Background(), "testnet", 123, nil, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got.Selection.Lead != want.Selection.Lead || got.SelectionTrace.CacheStatus != "hit" {
		t.Fatalf("cached envelope = %+v", got)
	}
}

func TestLedgerShadowRecordsValidatedChoiceWithoutApplyingIt(t *testing.T) {
	envelope, err := summary.BuildLedgerSummaryEnvelope(ledgerFactsFromGateway(completeLedgerSummaryEvidence()))
	if err != nil {
		t.Fatal(err)
	}
	if len(envelope.Candidates) < 2 {
		t.Fatalf("fixture needs competing candidates: %+v", envelope.Candidates)
	}
	deterministic := envelope.Selection.Lead
	shadowLead := envelope.Candidates[len(envelope.Candidates)-1].ID
	probabilities := make(map[string]float64, len(envelope.Candidates))
	for _, candidate := range envelope.Candidates {
		probabilities[string(candidate.ID)] = 0
	}
	probabilities[string(shadowLead)] = 1
	selector := &fakeLedgerSelector{selection: jev.LedgerSelection{RegistryVersion: jev.LedgerSelectorRegistryVersion, Model: "jev-test", Lead: string(shadowLead), Probabilities: probabilities, Confidence: .9}}
	h := &Handlers{LedgerSelector: selector}
	h.applyLedgerShadowSelection(context.Background(), &envelope)

	if envelope.Selection.Lead != deterministic {
		t.Fatalf("shadow changed applied lead from %q to %q", deterministic, envelope.Selection.Lead)
	}
	if envelope.SelectionTrace.Shadow == nil || envelope.SelectionTrace.Shadow.Status != "succeeded" || envelope.SelectionTrace.Shadow.Lead != shadowLead {
		t.Fatalf("shadow trace = %+v", envelope.SelectionTrace.Shadow)
	}
	if len(selector.request.Bands) != 7 || len(selector.request.Candidates) != len(envelope.Candidates) {
		t.Fatalf("shadow request = %+v", selector.request)
	}
	recorder := &fakeShadowRecorder{}
	h.LedgerShadowRecorder = recorder
	h.recordLedgerShadow("testnet", 123, envelope, 25*time.Millisecond)
	if len(recorder.records) != 1 || recorder.records[0].RecordVersion != jev.LedgerShadowRecordVersion || len(recorder.records[0].Bands) != 7 || recorder.records[0].LatencyMS != 25 {
		t.Fatalf("evaluation record = %+v", recorder.records)
	}
}

func TestLedgerShadowRejectsIneligibleSelectorOutput(t *testing.T) {
	envelope, err := summary.BuildLedgerSummaryEnvelope(ledgerFactsFromGateway(completeLedgerSummaryEvidence()))
	if err != nil {
		t.Fatal(err)
	}
	selector := &fakeLedgerSelector{selection: jev.LedgerSelection{RegistryVersion: jev.LedgerSelectorRegistryVersion, Model: "jev-test", Lead: "invented", Probabilities: map[string]float64{"invented": 1}, Confidence: 1}}
	h := &Handlers{LedgerSelector: selector}
	h.applyLedgerShadowSelection(context.Background(), &envelope)
	if envelope.SelectionTrace.Shadow == nil || envelope.SelectionTrace.Shadow.Status != "rejected" {
		t.Fatalf("shadow trace = %+v", envelope.SelectionTrace.Shadow)
	}
}

func TestLedgerShadowFailureIsRecordedWithoutChangingSelection(t *testing.T) {
	envelope, err := summary.BuildLedgerSummaryEnvelope(ledgerFactsFromGateway(completeLedgerSummaryEvidence()))
	if err != nil {
		t.Fatal(err)
	}
	want := envelope.Selection
	h := &Handlers{LedgerSelector: &fakeLedgerSelector{err: errors.New("upstream unavailable")}}
	h.applyLedgerShadowSelection(context.Background(), &envelope)
	if envelope.Selection.Lead != want.Lead || envelope.SelectionTrace.Shadow == nil || envelope.SelectionTrace.Shadow.Status != "error" || envelope.SelectionTrace.Shadow.Reason != "request_or_validation_error" {
		t.Fatalf("selection = %+v, shadow = %+v", envelope.Selection, envelope.SelectionTrace.Shadow)
	}
}

func completeLedgerSummaryEvidence() ledgerV3SummaryEvidence {
	txs := []gateway.Transaction{
		{TransactionHash: "a", Successful: true, OperationCount: 1},
		{TransactionHash: "b", Successful: true, OperationCount: 1},
	}
	ops := []gateway.Operation{{TransactionHash: "a", TypeName: "INVOKE_HOST_FUNCTION"}, {TransactionHash: "b", TypeName: "INVOKE_HOST_FUNCTION"}}
	usage := &gateway.LedgerSoroban{LedgerSequence: 123, SorobanTxCount: 2, TotalCPUInsns: 60, TotalReadEntries: 20, TotalWriteEntries: 30, FootprintEntriesAvailable: true, TotalReadBytes: 40, TotalWriteBytes: 50, TotalRentCharged: 7}
	return ledgerV3SummaryEvidence{
		Full:         &gateway.LedgerFullResponse{LedgerSequence: 123, Ledger: gateway.Ledger{Sequence: 123, ProtocolVersion: 23, TransactionCount: 2, SuccessfulTxCount: 2, OperationCount: 2, BaseFee: 100}, Transactions: txs, Operations: ops, Soroban: usage},
		Transactions: txs, Operations: ops, Usage: usage,
		Config:  &gateway.SorobanConfig{Instructions: gateway.SorobanInstructionLimits{LedgerMax: 100}, LedgerLimits: gateway.SorobanIOLimits{MaxReadEntries: 100, MaxWriteEntries: 100, MaxReadBytes: 100, MaxWriteBytes: 100}},
		Fees:    &gateway.LedgerFees{LedgerSequence: 123, TxCount: 2, MedianFee: 200, TotalFees: 400},
		Changes: &gateway.LedgerChanges{LedgerSequence: 123, Available: true, Created: 1, Updated: 2},
	}
}

func containsCandidate(values []summary.Candidate, id summary.InterpretationID) bool {
	for _, value := range values {
		if value.ID == id {
			return true
		}
	}
	return false
}
