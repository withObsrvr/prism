package summary

import (
	"fmt"
	"sort"
)

const (
	LedgerBaselineVersion        = "ledger_baselines_v1"
	LedgerBaselineMethod         = "median_prior_32_complete_ledgers"
	LedgerBaselineWindow         = 32
	LedgerBaselineMinimumSamples = 16
)

// LedgerBaselineSample is deliberately presentation-neutral. A nil change
// count means that the ledger has no served state-change evidence; it does not
// mean that the ledger changed zero entries.
type LedgerBaselineSample struct {
	Sequence     int64
	Transactions int64
	Operations   int64
	StateChanges *int64
}

type LedgerBaselines struct {
	Version            string        `json:"version"`
	Method             string        `json:"method"`
	Window             int           `json:"window"`
	MinimumSamples     int           `json:"minimum_samples"`
	FirstLedger        int64         `json:"first_ledger,omitempty"`
	LastLedger         int64         `json:"last_ledger,omitempty"`
	ActivitySamples    int           `json:"activity_samples"`
	StateChangeSamples int           `json:"state_change_samples"`
	Transactions       Int64Evidence `json:"transactions"`
	Operations         Int64Evidence `json:"operations"`
	StateChanges       Int64Evidence `json:"state_changes"`
}

// BuildLedgerBaselines compares a ledger only with preceding ledgers. Samples
// at or after targetSequence, duplicate sequences, invalid counts, and samples
// outside the nearest versioned window are excluded.
func BuildLedgerBaselines(targetSequence int64, samples []LedgerBaselineSample) LedgerBaselines {
	result := LedgerBaselines{
		Version: LedgerBaselineVersion, Method: LedgerBaselineMethod,
		Window: LedgerBaselineWindow, MinimumSamples: LedgerBaselineMinimumSamples,
		Transactions: unavailableBaseline("Insufficient preceding ledger activity evidence."),
		Operations:   unavailableBaseline("Insufficient preceding ledger activity evidence."),
		StateChanges: unavailableBaseline("Insufficient preceding ledger state-change evidence."),
	}

	valid := make([]LedgerBaselineSample, 0, len(samples))
	seen := make(map[int64]struct{}, len(samples))
	for _, sample := range samples {
		if sample.Sequence <= 0 || sample.Sequence >= targetSequence || sample.Transactions < 0 || sample.Operations < 0 {
			continue
		}
		if _, exists := seen[sample.Sequence]; exists {
			continue
		}
		seen[sample.Sequence] = struct{}{}
		valid = append(valid, sample)
	}
	sort.Slice(valid, func(i, j int) bool { return valid[i].Sequence < valid[j].Sequence })
	if len(valid) > LedgerBaselineWindow {
		valid = valid[len(valid)-LedgerBaselineWindow:]
	}
	if len(valid) > 0 {
		result.FirstLedger = valid[0].Sequence
		result.LastLedger = valid[len(valid)-1].Sequence
	}

	transactions := make([]int64, 0, len(valid))
	operations := make([]int64, 0, len(valid))
	changes := make([]int64, 0, len(valid))
	for _, sample := range valid {
		transactions = append(transactions, sample.Transactions)
		operations = append(operations, sample.Operations)
		if sample.StateChanges != nil && *sample.StateChanges >= 0 {
			changes = append(changes, *sample.StateChanges)
		}
	}
	result.ActivitySamples = len(transactions)
	result.StateChangeSamples = len(changes)
	result.Transactions = baselineEvidence(transactions, "transactions", result.FirstLedger, result.LastLedger)
	result.Operations = baselineEvidence(operations, "operations", result.FirstLedger, result.LastLedger)
	result.StateChanges = baselineEvidence(changes, "state changes", result.FirstLedger, result.LastLedger)
	return result
}

func baselineEvidence(values []int64, label string, first, last int64) Int64Evidence {
	if len(values) < LedgerBaselineMinimumSamples {
		return unavailableBaseline(fmt.Sprintf("Only %d of %d required preceding ledger samples were available for %s.", len(values), LedgerBaselineMinimumSamples, label))
	}
	value := integerMedian(values)
	if value <= 0 {
		return unavailableBaseline(fmt.Sprintf("The %s median was zero, so a relative band was not evaluated.", label))
	}
	return Int64Evidence{
		Value: value, Status: Available,
		Source: fmt.Sprintf("%s; ledgers %d-%d; n=%d", LedgerBaselineMethod, first, last, len(values)),
	}
}

func unavailableBaseline(caveat string) Int64Evidence {
	return Int64Evidence{Status: Unavailable, Caveat: caveat}
}

func integerMedian(values []int64) int64 {
	ordered := append([]int64(nil), values...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i] < ordered[j] })
	middle := len(ordered) / 2
	if len(ordered)%2 == 1 {
		return ordered[middle]
	}
	// Round an even-sized median to the nearest integer without summing two
	// potentially large values.
	low, high := ordered[middle-1], ordered[middle]
	return low + (high-low+1)/2
}
