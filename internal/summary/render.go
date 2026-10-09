package summary

import (
	"fmt"
	"sort"
	"strings"
)

type Headline struct {
	Lead     string `json:"lead"`
	Emphasis string `json:"emphasis"`
	Trail    string `json:"trail"`
}

type Action struct {
	Label string `json:"label"`
	Href  string `json:"href"`
}

type RenderedLedgerSummary struct {
	Headline        Headline  `json:"headline"`
	Detail          string    `json:"detail"`
	Supporting      []string  `json:"supporting,omitempty"`
	Caveats         []string  `json:"caveats,omitempty"`
	NextActions     []Action  `json:"next_actions,omitempty"`
	Interpretation  Selection `json:"interpretation"`
	Provenance      Evidence  `json:"provenance"`
	TemplateVersion string    `json:"template_version"`
}

// RenderLedgerSummary applies controlled templates to an already validated
// selection. It refuses unknown or ineligible IDs so neither a caller nor a
// future model can turn arbitrary text into a Prism claim.
func RenderLedgerSummary(envelope LedgerSummaryEnvelope) (RenderedLedgerSummary, error) {
	if envelope.TemplateVersion != LedgerTemplateVersion || envelope.RegistryVersion != LedgerInterpretationRegistryVersion || envelope.BandVersion != LedgerBandVersion || envelope.ThresholdVersion != LedgerThresholdVersion {
		return RenderedLedgerSummary{}, fmt.Errorf("unsupported ledger summary versions")
	}
	if envelope.BaselineVersion != LedgerBaselineVersion {
		return RenderedLedgerSummary{}, fmt.Errorf("unsupported ledger baseline version %q", envelope.BaselineVersion)
	}
	if envelope.SelectionTrace.SelectorVersion != LedgerSelectorVersion || envelope.SelectionTrace.AppliedLead != envelope.Selection.Lead || envelope.SelectionTrace.AppliedSource != envelope.Selection.Source {
		return RenderedLedgerSummary{}, fmt.Errorf("ledger selection trace does not match the applied selection")
	}
	derived, err := EligibleLedgerCandidates(envelope.State.Bands)
	if err != nil {
		return RenderedLedgerSummary{}, err
	}
	eligible := map[InterpretationID]Candidate{}
	for _, candidate := range derived {
		eligible[candidate.ID] = candidate
	}
	if len(envelope.SelectionTrace.Eligible) != len(derived) {
		return RenderedLedgerSummary{}, fmt.Errorf("ledger selection trace has a stale eligible set")
	}
	for i, candidate := range derived {
		if envelope.SelectionTrace.Eligible[i] != candidate.ID {
			return RenderedLedgerSummary{}, fmt.Errorf("ledger selection trace has a stale eligible set")
		}
	}
	if envelope.Selection.Lead == "" {
		return renderNoCandidate(envelope), nil
	}
	if _, ok := eligible[envelope.Selection.Lead]; !ok {
		return RenderedLedgerSummary{}, fmt.Errorf("selected interpretation %q is not eligible", envelope.Selection.Lead)
	}
	for _, id := range envelope.Selection.Supporting {
		if _, ok := eligible[id]; !ok {
			return RenderedLedgerSummary{}, fmt.Errorf("supporting interpretation %q is not eligible", id)
		}
	}

	rendered, err := renderLead(envelope.Selection.Lead, envelope.State)
	if err != nil {
		return RenderedLedgerSummary{}, err
	}
	rendered.Interpretation = envelope.Selection
	rendered.Provenance = envelope.State.Facts.Evidence
	rendered.TemplateVersion = LedgerTemplateVersion
	for _, id := range envelope.Selection.Supporting {
		if len(rendered.Supporting) == 2 {
			break
		}
		if text := renderSupporting(id, envelope.State); text != "" {
			rendered.Supporting = append(rendered.Supporting, text)
		}
	}
	rendered.Caveats = ledgerCaveats(envelope.State)
	rendered.NextActions = ledgerActions(envelope.Selection.Lead)
	return rendered, nil
}

func renderLead(id InterpretationID, state LedgerSummaryState) (RenderedLedgerSummary, error) {
	f, m := state.Facts, state.Metrics
	total, failed := formatInteger(f.Transactions.Value), formatInteger(f.Failed.Value)
	switch id {
	case InterpretationEvidenceIncomplete:
		return rendered(Headline{"Some ledger", "evidence", "is incomplete"}, fmt.Sprintf("The ledger header reports %s included transactions, but one or more evidence families are partial or unavailable. Missing measurements are not treated as zero.", total)), nil
	case InterpretationWidespreadFailures:
		return rendered(Headline{"Failures affected", "a material share", "of this ledger"}, fmt.Sprintf("%s of %s included transactions failed (%s).", failed, total, formatPercent(m.FailureRate.Ratio))), nil
	case InterpretationClusteredFailures:
		return rendered(Headline{"Several failures share", "one recorded cause", ""}, fmt.Sprintf("%s transactions failed; the largest recorded cause accounts for %s of those failures.", failed, formatPercent(m.LargestCauseShare.Ratio))), nil
	case InterpretationCapacityPressure:
		name, metric := bindingMetric(state)
		return rendered(Headline{"This ledger put pressure on", resourceLabel(name), "capacity"}, fmt.Sprintf("The most utilized measured resource reached %s of its effective ledger limit.", formatPercent(metric.Ratio))), nil
	case InterpretationHealthyCapacityElevatedFees:
		return rendered(Headline{"Capacity stayed healthy while", "fees rose", "above base"}, fmt.Sprintf("Every applicable measured resource retained substantial headroom, while the inclusion fee reached %s the base fee.", formatMultiple(m.Fees.InclusionMultiple.Ratio))), nil
	case InterpretationElevatedFees:
		return rendered(Headline{"Inclusion fees were", "elevated", "above base"}, fmt.Sprintf("The measured inclusion fee was %s the ledger base fee.", formatMultiple(m.Fees.InclusionMultiple.Ratio))), nil
	case InterpretationStateChangeHeavy:
		return rendered(Headline{"This ledger produced", "heavy state change", ""}, fmt.Sprintf("Transactions produced %s recorded entry changes, %s the versioned baseline.", formatInteger(m.StateChangeTotal.Value), formatMultiple(m.StateChangeRatio.Ratio))), nil
	case InterpretationSorobanHeavy:
		return rendered(Headline{"This ledger was", "Soroban-heavy", ""}, fmt.Sprintf("Soroban transactions represented %s of %s included transactions.", formatPercent(m.SorobanShare.Ratio), total)), nil
	case InterpretationEmptyLedger:
		return rendered(Headline{"This ledger included", "no transactions", ""}, "The ledger closed without including a transaction or operation."), nil
	case InterpretationActiveHealthy:
		activity, metric := bindingActivityMetric(state)
		if metric.Status != Available {
			return RenderedLedgerSummary{}, fmt.Errorf("busy activity interpretation has no available baseline ratio")
		}
		return rendered(Headline{"This ledger was", "busy and healthy", ""}, fmt.Sprintf("%s reached %s the median across the preceding ledger window; all %s transactions succeeded while applicable measured resources retained substantial headroom.", activity, formatMultiple(metric.Ratio), total)), nil
	case InterpretationIsolatedFailure:
		return rendered(Headline{"Failures were", "isolated", "in this ledger"}, fmt.Sprintf("%s of %s included transactions failed; the evidence does not show a ledger-wide failure pattern.", failed, total)), nil
	case InterpretationAllSucceeded:
		return rendered(Headline{"Every included transaction", "succeeded", ""}, fmt.Sprintf("All %s included transactions applied successfully.", total)), nil
	case InterpretationHealthyCapacity:
		return rendered(Headline{"This ledger retained", "capacity headroom", ""}, "Every applicable measured resource remained below the elevated-pressure threshold."), nil
	case InterpretationRoutineLedger:
		return rendered(Headline{"This ledger was", "routine", "across measured signals"}, fmt.Sprintf("All %s included transactions succeeded; activity remained within its versioned baseline band, fees remained at base, and measured capacity retained substantial headroom.", total)), nil
	default:
		return RenderedLedgerSummary{}, fmt.Errorf("no controlled ledger template for %q", id)
	}
}

func rendered(headline Headline, detail string) RenderedLedgerSummary {
	return RenderedLedgerSummary{Headline: headline, Detail: detail}
}

func renderSupporting(id InterpretationID, state LedgerSummaryState) string {
	f, m := state.Facts, state.Metrics
	switch id {
	case InterpretationElevatedFees:
		return "Inclusion fees reached " + formatMultiple(m.Fees.InclusionMultiple.Ratio) + " the base fee."
	case InterpretationHealthyCapacity:
		return "Applicable measured resources retained substantial headroom."
	case InterpretationAllSucceeded:
		return "All " + formatInteger(f.Transactions.Value) + " included transactions succeeded."
	case InterpretationSorobanHeavy:
		return "Soroban represented " + formatPercent(m.SorobanShare.Ratio) + " of included transactions."
	case InterpretationEmptyLedger:
		return "The ledger closed without including a transaction."
	case InterpretationStateChangeHeavy:
		return formatInteger(m.StateChangeTotal.Value) + " recorded entry changes were heavy relative to baseline."
	case InterpretationIsolatedFailure:
		return formatInteger(f.Failed.Value) + " failures were isolated in the complete transaction set."
	case InterpretationClusteredFailures:
		return "The largest recorded cause accounts for " + formatPercent(m.LargestCauseShare.Ratio) + " of failures."
	case InterpretationCapacityPressure:
		_, metric := bindingMetric(state)
		return "The binding measured resource reached " + formatPercent(metric.Ratio) + " of its limit."
	case InterpretationRoutineLedger:
		return "Fees, outcomes, activity, and measured capacity remained in routine bands."
	case InterpretationActiveHealthy:
		activity, metric := bindingActivityMetric(state)
		return activity + " reached " + formatMultiple(metric.Ratio) + " the preceding-ledger median without failures or measured capacity pressure."
	default:
		return ""
	}
}

func ledgerCaveats(state LedgerSummaryState) []string {
	values := append([]string(nil), state.Facts.Evidence.Caveats...)
	if state.Bands.Evidence == EvidencePartial {
		values = append(values, "The transaction or operation evidence is partial; complete-ledger pattern claims are disabled.")
	}
	if state.Bands.Capacity.Overall == PressurePartial {
		values = append(values, "Some applicable capacity dimensions are unavailable, so Prism does not make a complete capacity claim.")
	}
	if state.Bands.Fees.Inclusion == FeeUnavailable {
		values = append(values, "The inclusion-fee component is unavailable or cannot be separated from Soroban resource fees.")
	}
	if state.Bands.Activity == ActivityNotEvaluated {
		values = append(values, "Activity is not compared with a baseline yet.")
	}
	if state.Bands.StateChange.Volume == ChangeNotEvaluated {
		values = append(values, "State-change volume is not compared with a baseline yet.")
	}
	return uniqueStrings(values)
}

func ledgerActions(lead InterpretationID) []Action {
	switch lead {
	case InterpretationEvidenceIncomplete:
		return []Action{{"Inspect provenance", "#notes"}, {"Review transaction evidence", "#s1"}}
	case InterpretationCapacityPressure, InterpretationHealthyCapacity:
		return []Action{{"Inspect capacity", "#s2"}, {"Review transaction evidence", "#s1"}}
	case InterpretationElevatedFees, InterpretationHealthyCapacityElevatedFees:
		return []Action{{"Inspect fees", "#s3"}, {"Inspect capacity", "#s2"}}
	case InterpretationWidespreadFailures, InterpretationClusteredFailures, InterpretationIsolatedFailure:
		return []Action{{"Inspect failures", "#s4"}, {"Review transaction evidence", "#s1"}}
	case InterpretationStateChangeHeavy:
		return []Action{{"Inspect state changes", "#s5"}, {"Inspect provenance", "#notes"}}
	default:
		return []Action{{"Review transaction evidence", "#s1"}, {"Inspect provenance", "#notes"}}
	}
}

func renderNoCandidate(envelope LedgerSummaryEnvelope) RenderedLedgerSummary {
	return RenderedLedgerSummary{Headline: Headline{"Ledger", formatInteger(envelope.State.Facts.Sequence), ""}, Detail: "Prism has no registered interpretation for the available semantic state.", Caveats: ledgerCaveats(envelope.State), NextActions: []Action{{"Inspect provenance", "#notes"}}, Interpretation: envelope.Selection, Provenance: envelope.State.Facts.Evidence, TemplateVersion: LedgerTemplateVersion}
}

func bindingMetric(state LedgerSummaryState) (string, RatioMetric) {
	name := state.Bands.Capacity.BindingResource
	m := state.Metrics.Capacity
	switch name {
	case "cpu":
		return name, m.CPU
	case "read_entries":
		return name, m.ReadEntries
	case "write_entries":
		return name, m.WriteEntries
	case "read_bytes":
		return name, m.ReadBytes
	case "write_bytes":
		return name, m.WriteBytes
	case "transaction_bytes":
		return name, m.TransactionBytes
	case "event_return_bytes":
		return name, m.EventReturnBytes
	default:
		return name, RatioMetric{}
	}
}

func bindingActivityMetric(state LedgerSummaryState) (string, RatioMetric) {
	tx, operations := state.Metrics.Activity.TransactionRatio, state.Metrics.Activity.OperationRatio
	if operations.Status == Available && (tx.Status != Available || operations.Ratio > tx.Ratio) {
		return "Operation activity", operations
	}
	return "Transaction activity", tx
}

func resourceLabel(value string) string {
	if value == "" {
		return "measured"
	}
	return strings.ReplaceAll(value, "_", " ")
}
func formatInteger(value int64) string {
	raw := fmt.Sprintf("%d", value)
	start := 0
	if strings.HasPrefix(raw, "-") {
		start = 1
	}
	for i := len(raw) - 3; i > start; i -= 3 {
		raw = raw[:i] + "," + raw[i:]
	}
	return raw
}
func formatPercent(value float64) string { return fmt.Sprintf("%.1f%%", value*100) }
func formatMultiple(value float64) string {
	if value >= 10 {
		return fmt.Sprintf("%.0f×", value)
	}
	return fmt.Sprintf("%.1f×", value)
}
func uniqueStrings(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}
