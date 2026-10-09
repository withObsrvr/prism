package summary

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"
)

type interpretationFixture struct {
	Name  string `json:"name"`
	Bands struct {
		Evidence    EvidenceBand    `json:"evidence"`
		Capacity    PressureBand    `json:"capacity"`
		Fees        FeeBand         `json:"fees"`
		Outcomes    OutcomeBand     `json:"outcomes"`
		Activity    ActivityBand    `json:"activity"`
		Composition CompositionBand `json:"composition"`
		StateChange ChangeBand      `json:"state_change"`
	} `json:"bands"`
	Eligible  []InterpretationID `json:"eligible"`
	Lead      InterpretationID   `json:"lead"`
	Forbidden []InterpretationID `json:"forbidden"`
}

func TestLedgerInterpretationFixturesV2(t *testing.T) {
	body, err := os.ReadFile("testdata/ledger-interpretations-v2.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixtures []interpretationFixture
	if err := json.Unmarshal(body, &fixtures); err != nil {
		t.Fatal(err)
	}
	for _, fixture := range fixtures {
		t.Run(fixture.Name, func(t *testing.T) {
			bands := LedgerBands{Evidence: fixture.Bands.Evidence, Capacity: LedgerCapacityBands{Overall: fixture.Bands.Capacity}, Fees: LedgerFeeBands{Overall: fixture.Bands.Fees}, Outcomes: fixture.Bands.Outcomes, Activity: fixture.Bands.Activity, Composition: fixture.Bands.Composition, StateChange: LedgerStateChangeBands{Volume: fixture.Bands.StateChange}}
			candidates, err := EligibleLedgerCandidates(bands)
			if err != nil {
				t.Fatal(err)
			}
			ids := candidateIDs(candidates)
			if !reflect.DeepEqual(ids, fixture.Eligible) {
				t.Fatalf("eligible = %v, want %v", ids, fixture.Eligible)
			}
			selection := SelectLedgerDeterministically(candidates)
			if selection.Lead != fixture.Lead {
				t.Fatalf("lead = %q, want %q", selection.Lead, fixture.Lead)
			}
			for _, forbidden := range fixture.Forbidden {
				if containsInterpretation(ids, forbidden) {
					t.Errorf("unsafe interpretation %q was eligible", forbidden)
				}
			}
		})
	}
}

func TestLedgerInterpretationRegistryIsValid(t *testing.T) {
	if err := ValidateInterpretationRegistry(LedgerInterpretationsV2); err != nil {
		t.Fatal(err)
	}
}

func TestRegistryRejectsUnknownClaimAndBandValue(t *testing.T) {
	tests := []InterpretationSpec{
		spec("bad_claim", "bad", 1, []ClaimID{"unknown"}, group(complete())),
		spec("bad_value", "bad", 1, []ClaimID{ClaimNoFailures}, group(req("outcomes", "perfect"))),
	}
	for _, item := range tests {
		if err := ValidateInterpretationRegistry([]InterpretationSpec{item}); err == nil {
			t.Errorf("registry accepted invalid spec %q", item.ID)
		}
	}
}

func TestDeterministicSelectionUsesIDForPriorityTie(t *testing.T) {
	selection := SelectLedgerDeterministically([]Candidate{{ID: "z", Priority: 10}, {ID: "a", Priority: 10}})
	if selection.Lead != "a" {
		t.Fatalf("lead = %q", selection.Lead)
	}
}

func candidateIDs(values []Candidate) []InterpretationID {
	out := make([]InterpretationID, 0, len(values))
	for _, value := range values {
		out = append(out, value.ID)
	}
	return out
}
func containsInterpretation(values []InterpretationID, want InterpretationID) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
