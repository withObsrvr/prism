package summary

import (
	"encoding/json"
	"os"
	"testing"
)

type ledgerBandFixture struct {
	Name                string                     `json:"name"`
	Transactions        int64                      `json:"transactions"`
	Operations          int64                      `json:"operations"`
	Failed              int64                      `json:"failed"`
	LargestFailureGroup int64                      `json:"largest_failure_group"`
	SorobanTransactions int64                      `json:"soroban_transactions"`
	Capacity            map[string]resourceFixture `json:"capacity"`
	BaseFee             int64                      `json:"base_fee"`
	InclusionFee        int64                      `json:"inclusion_fee"`
	TransactionBaseline int64                      `json:"transaction_baseline"`
	OperationBaseline   int64                      `json:"operation_baseline"`
	Changes             int64                      `json:"changes"`
	ChangeBaseline      int64                      `json:"change_baseline"`
	Expected            expectedBands              `json:"expected"`
}

type resourceFixture struct {
	Used   int64        `json:"used"`
	Limit  int64        `json:"limit"`
	Status Availability `json:"status"`
}

type expectedBands struct {
	Capacity    PressureBand    `json:"capacity"`
	Fees        FeeBand         `json:"fees"`
	Outcomes    OutcomeBand     `json:"outcomes"`
	Activity    ActivityBand    `json:"activity"`
	Composition CompositionBand `json:"composition"`
	StateChange ChangeBand      `json:"state_change"`
}

func TestLedgerBandFixturesV2(t *testing.T) {
	body, err := os.ReadFile("testdata/ledger-bands-v2.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixtures []ledgerBandFixture
	if err := json.Unmarshal(body, &fixtures); err != nil {
		t.Fatal(err)
	}
	for _, fixture := range fixtures {
		t.Run(fixture.Name, func(t *testing.T) {
			state, err := BuildLedgerSummaryState(fixtureFacts(fixture))
			if err != nil {
				t.Fatal(err)
			}
			got := expectedBands{state.Bands.Capacity.Overall, state.Bands.Fees.Overall, state.Bands.Outcomes, state.Bands.Activity, state.Bands.Composition, state.Bands.StateChange.Volume}
			if got != fixture.Expected {
				t.Fatalf("bands = %+v, want %+v", got, fixture.Expected)
			}
		})
	}
}

func TestLedgerBandsFailClosedForPartialOutcomes(t *testing.T) {
	facts := fixtureFacts(ledgerBandFixture{Transactions: 100, Operations: 100, Failed: 0})
	facts.Transactions.Status = Partial
	state, err := BuildLedgerSummaryState(facts)
	if err != nil {
		t.Fatal(err)
	}
	if state.Bands.Outcomes != OutcomePartial {
		t.Fatalf("outcomes = %q", state.Bands.Outcomes)
	}
}

func TestValidateLedgerFactsRejectsContradictoryCounts(t *testing.T) {
	facts := fixtureFacts(ledgerBandFixture{Transactions: 10, Operations: 10, Failed: 2})
	facts.Successful.Value = 9
	if _, err := BuildLedgerSummaryState(facts); err == nil {
		t.Fatal("expected contradictory transaction counts to fail")
	}
}

func TestMissingFeeEvidenceNeverBecomesBaseFeeBand(t *testing.T) {
	facts := fixtureFacts(ledgerBandFixture{Transactions: 1, Operations: 1})
	facts.Fees.BaseFee = Int64Evidence{Status: Unavailable}
	facts.Fees.InclusionFee = Int64Evidence{Status: Unavailable}
	state, err := BuildLedgerSummaryState(facts)
	if err != nil {
		t.Fatal(err)
	}
	if state.Bands.Fees.Inclusion != FeeUnavailable {
		t.Fatalf("fee band = %q", state.Bands.Fees.Inclusion)
	}
}

func TestNotApplicableCapacityDoesNotMakeKnownCapacityPartial(t *testing.T) {
	facts := fixtureFacts(ledgerBandFixture{Transactions: 1, Operations: 1, Capacity: map[string]resourceFixture{"cpu": {Used: 20, Limit: 100, Status: Available}}})
	state, err := BuildLedgerSummaryState(facts)
	if err != nil {
		t.Fatal(err)
	}
	if state.Bands.Capacity.Overall != PressureHealthy {
		t.Fatalf("capacity = %q", state.Bands.Capacity.Overall)
	}
}

func TestEntirelyNotApplicableCapacityHasDistinctBand(t *testing.T) {
	facts := fixtureFacts(ledgerBandFixture{Transactions: 1, Operations: 1})
	state, err := BuildLedgerSummaryState(facts)
	if err != nil {
		t.Fatal(err)
	}
	if state.Bands.Capacity.Overall != PressureNotApplicable {
		t.Fatalf("capacity = %q", state.Bands.Capacity.Overall)
	}
}

func TestZeroTransactionLedgerHasEmptyOutcome(t *testing.T) {
	facts := fixtureFacts(ledgerBandFixture{})
	state, err := BuildLedgerSummaryState(facts)
	if err != nil {
		t.Fatal(err)
	}
	if state.Bands.Outcomes != OutcomeEmpty {
		t.Fatalf("outcomes = %q", state.Bands.Outcomes)
	}
	candidates, err := EligibleLedgerCandidates(state.Bands)
	if err != nil {
		t.Fatal(err)
	}
	selection := SelectLedgerDeterministically(candidates)
	if selection.Lead != InterpretationEmptyLedger {
		t.Fatalf("lead = %q, candidates = %+v", selection.Lead, candidates)
	}
}

func fixtureFacts(f ledgerBandFixture) LedgerFacts {
	na := ResourceFacts{Used: Int64Evidence{Status: NotApplicable}, Limit: Int64Evidence{Status: NotApplicable}}
	capacity := LedgerCapacityFacts{CPU: na, ReadEntries: na, WriteEntries: na, ReadBytes: na, WriteBytes: na, TransactionBytes: na, EventReturnBytes: na}
	setResource := func(name string, target *ResourceFacts) {
		value, ok := f.Capacity[name]
		if !ok {
			return
		}
		target.Used = Int64Evidence{Value: value.Used, Status: value.Status}
		target.Limit = Int64Evidence{Value: value.Limit, Status: value.Status}
	}
	setResource("cpu", &capacity.CPU)
	setResource("read_entries", &capacity.ReadEntries)
	setResource("write_entries", &capacity.WriteEntries)

	baseline := func(value int64) Int64Evidence {
		if value == 0 {
			return Int64Evidence{Status: Unavailable}
		}
		return Known(value)
	}
	changes := Int64Evidence{Status: Unavailable}
	if f.Changes > 0 {
		changes = Known(f.Changes)
	}
	return LedgerFacts{
		Sequence: 1, Evidence: Evidence{Status: Available},
		Transactions: Known(f.Transactions), Operations: Known(f.Operations), Successful: Known(f.Transactions - f.Failed), Failed: Known(f.Failed),
		FailureCause: FailureCauseFacts{LargestGroup: Known(f.LargestFailureGroup), Distinct: Int64Evidence{Status: Unavailable}},
		Capacity:     capacity,
		Fees:         LedgerFeeFacts{BaseFee: Known(f.BaseFee), InclusionFee: Known(f.InclusionFee), ResourceFee: Int64Evidence{Status: NotApplicable}, RentFee: Int64Evidence{Status: NotApplicable}, StorageWriteFee: Int64Evidence{Status: NotApplicable}, TotalCharged: Int64Evidence{Status: Unavailable}},
		Activity:     LedgerActivityFacts{TransactionBaseline: baseline(f.TransactionBaseline), OperationBaseline: baseline(f.OperationBaseline)},
		Mix:          LedgerCompositionFacts{SorobanTransactions: Known(f.SorobanTransactions)},
		Changes:      LedgerChangeFacts{Created: changes, Updated: Known(0), Deleted: Known(0), Restored: Known(0), Archived: Int64Evidence{Status: Unavailable}, TTLExtended: Int64Evidence{Status: Unavailable}, Baseline: baseline(f.ChangeBaseline)},
	}
}
