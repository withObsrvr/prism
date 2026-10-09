package summary

import (
	"strings"
	"testing"
)

func TestLedgerBaselinesUseNearestPriorWindowAndMedian(t *testing.T) {
	samples := make([]LedgerBaselineSample, 0, 40)
	for sequence := int64(60); sequence < 100; sequence++ {
		changes := sequence - 50
		samples = append(samples, LedgerBaselineSample{Sequence: sequence, Transactions: sequence - 50, Operations: (sequence - 50) * 2, StateChanges: &changes})
	}
	// Neither a duplicate nor the target ledger may influence the baseline.
	samples = append(samples, LedgerBaselineSample{Sequence: 99, Transactions: 9999, Operations: 9999}, LedgerBaselineSample{Sequence: 100, Transactions: 9999, Operations: 9999})

	got := BuildLedgerBaselines(100, samples)
	if got.Version != LedgerBaselineVersion || got.FirstLedger != 68 || got.LastLedger != 99 {
		t.Fatalf("baseline window = %+v", got)
	}
	if got.Transactions.Status != Available || got.Transactions.Value != 34 {
		t.Fatalf("transaction median = %+v", got.Transactions)
	}
	if got.Operations.Value != 67 || got.StateChanges.Value != 34 {
		t.Fatalf("medians = operations %+v, changes %+v", got.Operations, got.StateChanges)
	}
	if !strings.Contains(got.Transactions.Source, "n=32") {
		t.Fatalf("source = %q", got.Transactions.Source)
	}
}

func TestLedgerBaselinesRequireMinimumIndependentSamples(t *testing.T) {
	samples := make([]LedgerBaselineSample, 15)
	for i := range samples {
		samples[i] = LedgerBaselineSample{Sequence: int64(i + 1), Transactions: 10, Operations: 20}
	}
	got := BuildLedgerBaselines(20, samples)
	if got.Transactions.Status != Unavailable || got.Operations.Status != Unavailable || got.StateChanges.Status != Unavailable {
		t.Fatalf("insufficient baseline became available: %+v", got)
	}
}

func TestLedgerBaselinesDoNotTurnZeroMedianIntoRelativeClaim(t *testing.T) {
	samples := make([]LedgerBaselineSample, 16)
	for i := range samples {
		samples[i] = LedgerBaselineSample{Sequence: int64(i + 1)}
	}
	got := BuildLedgerBaselines(20, samples)
	if got.Transactions.Status != Unavailable || !strings.Contains(got.Transactions.Caveat, "median was zero") {
		t.Fatalf("zero median = %+v", got.Transactions)
	}
}
