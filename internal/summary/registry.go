package summary

import (
	"fmt"
	"sort"
)

type ClaimID string
type InterpretationID string

const (
	ClaimEvidenceIncomplete ClaimID = "ledger_evidence_incomplete"
	ClaimCapacityHealthy    ClaimID = "ledger_capacity_healthy"
	ClaimCapacityPressure   ClaimID = "ledger_capacity_pressure"
	ClaimFeesBase           ClaimID = "ledger_fees_base"
	ClaimFeesElevated       ClaimID = "ledger_fees_elevated"
	ClaimNoFailures         ClaimID = "ledger_no_failures"
	ClaimNoTransactions     ClaimID = "ledger_no_transactions"
	ClaimIsolatedFailures   ClaimID = "ledger_failures_isolated"
	ClaimClusteredFailures  ClaimID = "ledger_failures_clustered"
	ClaimWidespreadFailures ClaimID = "ledger_failures_widespread"
	ClaimActivityBusy       ClaimID = "ledger_activity_busy"
	ClaimSorobanHeavy       ClaimID = "ledger_soroban_heavy"
	ClaimStateChangeHeavy   ClaimID = "ledger_state_change_heavy"
)

const (
	InterpretationEvidenceIncomplete          InterpretationID = "evidence_incomplete"
	InterpretationRoutineLedger               InterpretationID = "routine_ledger"
	InterpretationActiveHealthy               InterpretationID = "active_healthy_ledger"
	InterpretationAllSucceeded                InterpretationID = "all_transactions_succeeded"
	InterpretationEmptyLedger                 InterpretationID = "empty_ledger"
	InterpretationIsolatedFailure             InterpretationID = "isolated_transaction_failure"
	InterpretationClusteredFailures           InterpretationID = "clustered_transaction_failures"
	InterpretationWidespreadFailures          InterpretationID = "widespread_transaction_failures"
	InterpretationCapacityPressure            InterpretationID = "capacity_pressure"
	InterpretationHealthyCapacity             InterpretationID = "healthy_capacity"
	InterpretationElevatedFees                InterpretationID = "elevated_fees"
	InterpretationHealthyCapacityElevatedFees InterpretationID = "healthy_capacity_with_elevated_fees"
	InterpretationSorobanHeavy                InterpretationID = "soroban_heavy"
	InterpretationStateChangeHeavy            InterpretationID = "state_change_heavy"
)

const (
	LedgerInterpretationRegistryVersion = "ledger_interpretations_v2"
	LedgerTemplateVersion               = "ledger_summary_templates_v1"
)

// BandRequirement is deliberately declarative so a registry can be audited,
// serialized, and presented to Jev later without shipping executable policy.
type BandRequirement struct {
	Path    string   `json:"path"`
	Allowed []string `json:"allowed"`
}

type RequirementGroup struct {
	All []BandRequirement `json:"all"`
}

type InterpretationSpec struct {
	ID              InterpretationID   `json:"id"`
	Description     string             `json:"description"`
	Claims          []ClaimID          `json:"claims"`
	AnyOf           []RequirementGroup `json:"any_of"`
	Priority        int                `json:"priority"`
	TemplateVersion string             `json:"template_version"`
}

type Candidate struct {
	ID          InterpretationID `json:"id"`
	Description string           `json:"description"`
	Claims      []ClaimID        `json:"claims"`
	Priority    int              `json:"priority"`
}

type Selection struct {
	Lead            InterpretationID   `json:"lead"`
	Supporting      []InterpretationID `json:"supporting,omitempty"`
	Source          string             `json:"source"`
	RegistryVersion string             `json:"registry_version"`
}

var LedgerInterpretationsV2 = []InterpretationSpec{
	// Evidence incompleteness is a fallback, not the most important thing that
	// happened in a ledger. Independent verified interpretations lead whenever
	// they exist; caveats still disclose every missing evidence family.
	spec(InterpretationEvidenceIncomplete, "Some evidence needed for a complete ledger interpretation is unavailable or partial.", 5, []ClaimID{ClaimEvidenceIncomplete},
		group(req("evidence", "partial", "stale", "unavailable")),
		group(req("capacity.overall", "partial", "unavailable")),
		group(req("outcomes", "partial", "unavailable"))),
	spec(InterpretationWidespreadFailures, "Failures affected a material share of included transactions.", 100, []ClaimID{ClaimWidespreadFailures}, group(complete(), req("outcomes", "widespread"))),
	spec(InterpretationClusteredFailures, "Several failures share one dominant recorded cause.", 90, []ClaimID{ClaimClusteredFailures}, group(complete(), req("outcomes", "clustered"))),
	spec(InterpretationCapacityPressure, "At least one measured ledger resource approached or exceeded its effective limit.", 80, []ClaimID{ClaimCapacityPressure}, group(complete(), req("capacity.overall", "near_limit", "exceeded"))),
	spec(InterpretationHealthyCapacityElevatedFees, "Measured resources retained headroom while inclusion fees were elevated.", 75, []ClaimID{ClaimCapacityHealthy, ClaimFeesElevated}, group(complete(), req("capacity.overall", "healthy"), req("fees.overall", "elevated", "extreme"))),
	spec(InterpretationElevatedFees, "The measured inclusion fee was elevated relative to the ledger base fee.", 70, []ClaimID{ClaimFeesElevated}, group(complete(), req("fees.overall", "elevated", "extreme"))),
	spec(InterpretationStateChangeHeavy, "State-change volume was heavy relative to its versioned baseline.", 60, []ClaimID{ClaimStateChangeHeavy}, group(complete(), req("state_change.volume", "heavy"))),
	spec(InterpretationSorobanHeavy, "Soroban transactions formed a large share of included transactions.", 50, []ClaimID{ClaimSorobanHeavy}, group(complete(), req("composition", "soroban_heavy"))),
	spec(InterpretationEmptyLedger, "No transactions were included in this ledger.", 48, []ClaimID{ClaimNoTransactions}, group(complete(), req("outcomes", "empty"))),
	spec(InterpretationActiveHealthy, "Activity was busy while measured capacity and outcomes remained healthy.", 45, []ClaimID{ClaimActivityBusy, ClaimCapacityHealthy, ClaimNoFailures}, group(complete(), req("activity", "busy"), req("capacity.overall", "healthy"), req("outcomes", "none"))),
	spec(InterpretationIsolatedFailure, "A small number of failures were isolated rather than a ledger-wide pattern.", 40, []ClaimID{ClaimIsolatedFailures}, group(complete(), req("outcomes", "isolated"))),
	spec(InterpretationAllSucceeded, "Every included transaction in the complete evidence set succeeded.", 35, []ClaimID{ClaimNoFailures}, group(complete(), req("outcomes", "none"))),
	spec(InterpretationHealthyCapacity, "Every applicable measured resource retained substantial headroom.", 30, []ClaimID{ClaimCapacityHealthy}, group(complete(), req("capacity.overall", "healthy"))),
	spec(InterpretationRoutineLedger, "Activity, fees, outcomes, and measured capacity were all within their routine bands.", 37, []ClaimID{ClaimCapacityHealthy, ClaimFeesBase, ClaimNoFailures}, group(complete(), req("capacity.overall", "healthy"), req("fees.overall", "base"), req("outcomes", "none"), req("activity", "typical"))),
}

func EligibleLedgerCandidates(bands LedgerBands) ([]Candidate, error) {
	if err := ValidateInterpretationRegistry(LedgerInterpretationsV2); err != nil {
		return nil, err
	}
	values := ledgerBandValues(bands)
	out := make([]Candidate, 0, len(LedgerInterpretationsV2))
	for _, item := range LedgerInterpretationsV2 {
		if requirementsMatch(item.AnyOf, values) {
			out = append(out, Candidate{ID: item.ID, Description: item.Description, Claims: append([]ClaimID(nil), item.Claims...), Priority: item.Priority})
		}
	}
	sortCandidates(out)
	return out, nil
}

func SelectLedgerDeterministically(candidates []Candidate) Selection {
	copyOf := append([]Candidate(nil), candidates...)
	sortCandidates(copyOf)
	selection := Selection{Source: "deterministic", RegistryVersion: LedgerInterpretationRegistryVersion}
	if len(copyOf) == 0 {
		return selection
	}
	selection.Lead = copyOf[0].ID
	for _, candidate := range copyOf[1:] {
		selection.Supporting = append(selection.Supporting, candidate.ID)
	}
	return selection
}

func ValidateInterpretationRegistry(registry []InterpretationSpec) error {
	seen := map[InterpretationID]struct{}{}
	knownClaims := ledgerClaimCatalog()
	knownPaths := ledgerBandPathCatalog()
	for _, item := range registry {
		if item.ID == "" || item.Description == "" || item.TemplateVersion == "" {
			return fmt.Errorf("interpretation spec has an empty required field")
		}
		if _, exists := seen[item.ID]; exists {
			return fmt.Errorf("duplicate interpretation id %q", item.ID)
		}
		seen[item.ID] = struct{}{}
		if len(item.Claims) == 0 || len(item.AnyOf) == 0 {
			return fmt.Errorf("interpretation %q has no claims or requirements", item.ID)
		}
		for _, claim := range item.Claims {
			if _, ok := knownClaims[claim]; !ok {
				return fmt.Errorf("interpretation %q uses unknown claim %q", item.ID, claim)
			}
		}
		for _, group := range item.AnyOf {
			if len(group.All) == 0 {
				return fmt.Errorf("interpretation %q has an empty requirement group", item.ID)
			}
			for _, requirement := range group.All {
				allowed, ok := knownPaths[requirement.Path]
				if !ok {
					return fmt.Errorf("interpretation %q uses unknown band path %q", item.ID, requirement.Path)
				}
				if len(requirement.Allowed) == 0 {
					return fmt.Errorf("interpretation %q has no allowed values for %q", item.ID, requirement.Path)
				}
				for _, value := range requirement.Allowed {
					if _, ok := allowed[value]; !ok {
						return fmt.Errorf("interpretation %q uses invalid value %q for %q", item.ID, value, requirement.Path)
					}
				}
			}
		}
	}
	return nil
}

func spec(id InterpretationID, description string, priority int, claims []ClaimID, groups ...RequirementGroup) InterpretationSpec {
	return InterpretationSpec{ID: id, Description: description, Claims: claims, AnyOf: groups, Priority: priority, TemplateVersion: LedgerTemplateVersion}
}
func group(requirements ...BandRequirement) RequirementGroup {
	return RequirementGroup{All: requirements}
}
func req(path string, values ...string) BandRequirement {
	return BandRequirement{Path: path, Allowed: values}
}
func complete() BandRequirement { return req("evidence", "complete") }

func requirementsMatch(groups []RequirementGroup, values map[string]string) bool {
	for _, group := range groups {
		matched := true
		for _, requirement := range group.All {
			actual, ok := values[requirement.Path]
			if !ok {
				matched = false
				break
			}
			allowed := false
			for _, value := range requirement.Allowed {
				if actual == value {
					allowed = true
					break
				}
			}
			if !allowed {
				matched = false
				break
			}
		}
		if matched {
			return true
		}
	}
	return false
}

func ledgerBandValues(b LedgerBands) map[string]string {
	return map[string]string{
		"evidence": string(b.Evidence), "capacity.overall": string(b.Capacity.Overall), "fees.overall": string(b.Fees.Overall),
		"outcomes": string(b.Outcomes), "activity": string(b.Activity), "composition": string(b.Composition), "state_change.volume": string(b.StateChange.Volume),
	}
}

// LedgerBandValues returns the small semantic state safe for a closed-choice
// selector. It intentionally excludes raw facts, measurements, and prose.
func LedgerBandValues(b LedgerBands) map[string]string {
	return ledgerBandValues(b)
}

func sortCandidates(values []Candidate) {
	sort.Slice(values, func(i, j int) bool {
		if values[i].Priority != values[j].Priority {
			return values[i].Priority > values[j].Priority
		}
		return values[i].ID < values[j].ID
	})
}

func ledgerClaimCatalog() map[ClaimID]struct{} {
	return map[ClaimID]struct{}{ClaimEvidenceIncomplete: {}, ClaimCapacityHealthy: {}, ClaimCapacityPressure: {}, ClaimFeesBase: {}, ClaimFeesElevated: {}, ClaimNoFailures: {}, ClaimNoTransactions: {}, ClaimIsolatedFailures: {}, ClaimClusteredFailures: {}, ClaimWidespreadFailures: {}, ClaimActivityBusy: {}, ClaimSorobanHeavy: {}, ClaimStateChangeHeavy: {}}
}

func ledgerBandPathCatalog() map[string]map[string]struct{} {
	set := func(values ...string) map[string]struct{} {
		out := map[string]struct{}{}
		for _, value := range values {
			out[value] = struct{}{}
		}
		return out
	}
	return map[string]map[string]struct{}{
		"evidence":            set("complete", "partial", "stale", "unavailable"),
		"capacity.overall":    set("unavailable", "not_applicable", "partial", "healthy", "elevated", "near_limit", "exceeded"),
		"fees.overall":        set("unavailable", "partial", "not_evaluated", "base", "elevated", "extreme"),
		"outcomes":            set("unavailable", "partial", "empty", "none", "isolated", "clustered", "widespread"),
		"activity":            set("unavailable", "partial", "not_evaluated", "quiet", "typical", "busy"),
		"composition":         set("unavailable", "partial", "not_evaluated", "classic_heavy", "mixed", "soroban_heavy"),
		"state_change.volume": set("unavailable", "partial", "not_evaluated", "quiet", "typical", "heavy"),
	}
}
