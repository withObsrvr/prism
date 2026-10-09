package summary

const LedgerSelectorVersion = "ledger_deterministic_selector_v1"

type SelectionTrace struct {
	Surface           string                `json:"surface"`
	SelectorVersion   string                `json:"selector_version"`
	Mode              string                `json:"mode"`
	Eligible          []InterpretationID    `json:"eligible"`
	DeterministicLead InterpretationID      `json:"deterministic_lead,omitempty"`
	AppliedLead       InterpretationID      `json:"applied_lead,omitempty"`
	AppliedSource     string                `json:"applied_source"`
	CacheStatus       string                `json:"cache_status"`
	Shadow            *ShadowSelectionTrace `json:"shadow,omitempty"`
}

type ShadowSelectionTrace struct {
	Status          string             `json:"status"`
	RegistryVersion string             `json:"registry_version,omitempty"`
	Model           string             `json:"model,omitempty"`
	Lead            InterpretationID   `json:"lead,omitempty"`
	Probabilities   map[string]float64 `json:"probabilities,omitempty"`
	Confidence      float64            `json:"confidence,omitempty"`
	Agrees          bool               `json:"agrees"`
	Reason          string             `json:"reason,omitempty"`
	InputTokens     int                `json:"input_tokens,omitempty"`
	OutputTokens    int                `json:"output_tokens,omitempty"`
}

type LedgerSummaryEnvelope struct {
	State            LedgerSummaryState `json:"state"`
	Candidates       []Candidate        `json:"candidates"`
	Selection        Selection          `json:"selection"`
	SelectionTrace   SelectionTrace     `json:"selection_trace"`
	BandVersion      string             `json:"band_version"`
	BaselineVersion  string             `json:"baseline_version"`
	ThresholdVersion string             `json:"threshold_version"`
	RegistryVersion  string             `json:"registry_version"`
	TemplateVersion  string             `json:"template_version"`
}

func BuildLedgerSummaryEnvelope(facts LedgerFacts) (LedgerSummaryEnvelope, error) {
	state, err := BuildLedgerSummaryState(facts)
	if err != nil {
		return LedgerSummaryEnvelope{}, err
	}
	candidates, err := EligibleLedgerCandidates(state.Bands)
	if err != nil {
		return LedgerSummaryEnvelope{}, err
	}
	selection := SelectLedgerDeterministically(candidates)
	eligible := make([]InterpretationID, 0, len(candidates))
	for _, candidate := range candidates {
		eligible = append(eligible, candidate.ID)
	}
	return LedgerSummaryEnvelope{
		State: state, Candidates: candidates,
		Selection: selection,
		SelectionTrace: SelectionTrace{
			Surface: "ledger_summary", SelectorVersion: LedgerSelectorVersion,
			Mode: "deterministic", Eligible: eligible,
			DeterministicLead: selection.Lead, AppliedLead: selection.Lead,
			AppliedSource: selection.Source, CacheStatus: "bypass",
		},
		BandVersion: LedgerBandVersion, BaselineVersion: LedgerBaselineVersion, ThresholdVersion: LedgerThresholdVersion,
		RegistryVersion: LedgerInterpretationRegistryVersion, TemplateVersion: LedgerTemplateVersion,
	}, nil
}
