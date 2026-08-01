package insight

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/withObsrvr/prism/internal/gateway"
)

const evaluationVersionV1 = "home_insight_evaluation_v1"

type EvaluationCheck struct {
	Label  string
	Value  string
	Detail string
	State  string
}

var evaluationRegistries = map[string][]string{
	"home_insight_detector_registry_v1": {"failure_spike", "contract_deployments_spike", "transaction_activity_spike"},
	"home_insight_detector_registry_v2": {"failure_spike", "contract_deployments_spike", "transaction_activity_spike", "successful_activity_growth"},
	"home_insight_detector_registry_v3": {"failure_spike", "contract_deployments_spike", "transaction_activity_spike", "successful_activity_growth", "failure_recovery"},
	"home_insight_detector_registry_v4": {"failure_spike", "contract_deployments_spike", "transaction_activity_spike", "successful_activity_growth", "failure_recovery", "new_contract_adoption"},
}

var evaluationRuleSpecs = map[string]v2RuleSpec{
	"failure_spike":              {"risk", "negative", "contract", "contract_failure_spike", comparisonMethodV1, "at_least"},
	"contract_deployments_spike": {"activity", "neutral", "network", "network_contract_deployments_spike", comparisonMethodV1, "at_least"},
	"transaction_activity_spike": {"activity", "neutral", "network", "network_transaction_activity_spike", comparisonMethodV1, "at_least"},
	"successful_activity_growth": v2Rules["successful_activity_growth"],
	"failure_recovery":           v2Rules["failure_recovery"],
	"new_contract_adoption":      v2Rules["new_contract_adoption"],
}

func ValidateEvaluation(value *gateway.HomeInsightEvaluationEnvelope) error {
	if value == nil {
		return errors.New("insight evaluation is missing")
	}
	expected, ok := evaluationRegistries[value.RegistryVersion]
	if value.EvidenceVersion != evaluationVersionV1 || !ok {
		return errors.New("unsupported insight evaluation version or registry")
	}
	if value.Status != "ready" && value.Status != "partial" && value.Status != "unavailable" {
		return errors.New("unsupported insight evaluation status")
	}
	windowStart, startErr := time.Parse(time.RFC3339, value.WindowStart)
	windowEnd, endErr := time.Parse(time.RFC3339, value.WindowEnd)
	if startErr != nil || endErr != nil || windowEnd.Sub(windowStart) != time.Hour || value.ComparisonMethod != comparisonMethodV1 || value.CompleteThroughLedger <= 0 {
		return errors.New("invalid insight evaluation window")
	}
	if len(value.Rules) != len(expected) {
		return errors.New("insight evaluation registry is incomplete")
	}
	if value.Status == "ready" && len(value.Caveats) != 0 {
		return errors.New("ready insight evaluation contains envelope caveats")
	}
	if value.Status != "ready" && len(value.Caveats) == 0 {
		return errors.New("limited insight evaluation is missing its caveat")
	}
	if value.Provenance.CompleteThroughLedger != value.CompleteThroughLedger || len(value.Provenance.Sources) == 0 {
		return errors.New("insight evaluation provenance does not reconcile")
	}
	if _, err := time.Parse(time.RFC3339, value.Provenance.UpdatedAt); err != nil {
		return errors.New("invalid insight evaluation provenance time")
	}
	seen := make(map[string]struct{}, len(value.Rules))
	for _, rule := range value.Rules {
		if _, duplicate := seen[rule.Type]; duplicate {
			return errors.New("insight evaluation contains a duplicate rule")
		}
		seen[rule.Type] = struct{}{}
		if err := validateEvaluationRule(rule, value.CompleteThroughLedger); err != nil {
			return fmt.Errorf("%s: %w", rule.Type, err)
		}
	}
	for _, insightType := range expected {
		if _, found := seen[insightType]; !found {
			return errors.New("insight evaluation omits a registered rule")
		}
	}
	return nil
}

func validateEvaluationRule(rule gateway.HomeInsightEvaluationRule, completeThrough int64) error {
	spec, ok := evaluationRuleSpecs[rule.Type]
	if !ok || rule.Family != spec.family || rule.Direction != spec.direction || rule.RuleID != spec.ruleID || rule.RuleVersion != "1" || rule.ComparisonMethod != spec.comparison || rule.RatioComparison != spec.ratioComparison {
		return errors.New("rule identity diverges from the frozen registry")
	}
	allowedStatus := rule.Status == "ready" || rule.Status == "partial" || rule.Status == "unavailable"
	allowedOutcome := map[string]bool{"evaluated": true, "valid_zero": true, "no_eligible_subject": true, "baseline_zero": true, "source_partial": true, "source_unavailable": true}
	if !allowedStatus || !allowedOutcome[rule.EvaluationOutcome] || rule.EvaluatedSubjectCount < 0 || rule.QualifyingSubjectCount < 0 || rule.QualifyingSubjectCount > rule.EvaluatedSubjectCount {
		return errors.New("invalid rule evaluation state")
	}
	if rule.ThresholdCrossed != (rule.QualifyingSubjectCount > 0) {
		return errors.New("threshold result does not match the qualifying count")
	}
	if rule.Status == "partial" && rule.EvaluationOutcome != "source_partial" || rule.Status == "unavailable" && rule.EvaluationOutcome != "source_unavailable" {
		return errors.New("rule status and outcome disagree")
	}
	if rule.Status != "ready" && len(rule.Caveats) == 0 {
		return errors.New("limited rule is missing a caveat")
	}
	if rule.Subject != nil && (rule.Subject.Kind != spec.subjectKind || strings.TrimSpace(rule.Subject.ID) == "") {
		return errors.New("invalid evaluated subject")
	}
	if (rule.ObservedFirstLedger == nil) != (rule.ObservedLastLedger == nil) {
		return errors.New("incomplete observed ledger bounds")
	}
	if rule.ObservedFirstLedger != nil && (*rule.ObservedFirstLedger <= 0 || *rule.ObservedFirstLedger > *rule.ObservedLastLedger || *rule.ObservedLastLedger > completeThrough) {
		return errors.New("observed ledger bounds are invalid")
	}
	if rule.ObservedValue != nil && *rule.ObservedValue < 0 || rule.BaselineValue != nil && *rule.BaselineValue < 0 || rule.Ratio != nil && *rule.Ratio < 0 {
		return errors.New("negative evaluation measurement")
	}
	if rule.BaselineValue != nil && *rule.BaselineValue > 0 && rule.ObservedValue != nil {
		if rule.Ratio == nil || !nearlyEqual(*rule.Ratio, *rule.ObservedValue / *rule.BaselineValue) {
			return errors.New("evaluation ratio does not reconcile")
		}
	}
	if rule.EvaluationOutcome == "baseline_zero" && (rule.BaselineValue == nil || *rule.BaselineValue != 0 || rule.Ratio != nil) {
		return errors.New("zero-baseline evaluation is malformed")
	}
	if rule.EvaluationOutcome == "evaluated" && (rule.ObservedValue == nil || rule.BaselineValue == nil || rule.Ratio == nil) {
		return errors.New("evaluated rule is missing measurements")
	}
	if rule.ThresholdCrossed {
		if rule.Ratio == nil || rule.MinimumRatio == nil || rule.ObservedValue == nil {
			return errors.New("threshold crossing is missing its measurements")
		}
		if rule.RatioComparison == "at_least" && *rule.Ratio < *rule.MinimumRatio || rule.RatioComparison == "at_most" && *rule.Ratio > *rule.MinimumRatio {
			return errors.New("threshold crossing contradicts its ratio")
		}
		if rule.MinimumObserved != nil && *rule.ObservedValue < *rule.MinimumObserved {
			return errors.New("threshold crossing contradicts its observation floor")
		}
	}
	return nil
}

func ValidateInsightDelivery(delivery *gateway.HomeInsightDelivery, evaluation *gateway.HomeInsightEvaluationEnvelope) error {
	if delivery == nil {
		return errors.New("insight delivery is missing")
	}
	if delivery.Mode != "current" && delivery.Mode != "last_good" && delivery.Mode != "unavailable" || delivery.MaxAgeSeconds <= 0 || delivery.ProjectionLagSecond < 0 || delivery.ProjectionLedgerLag < 0 {
		return errors.New("invalid insight delivery state")
	}
	if delivery.Mode == "unavailable" {
		if evaluation != nil {
			return errors.New("unavailable insight delivery contains an evaluation")
		}
		return nil
	}
	if evaluation == nil || delivery.EvaluatedWindowEnd != evaluation.WindowEnd {
		return errors.New("insight delivery does not match its evaluation window")
	}
	if _, err := time.Parse(time.RFC3339, delivery.RetainedAt); err != nil {
		return errors.New("invalid insight retention time")
	}
	return nil
}

// ValidateEvaluationInsights prevents the detector ledger and the selected
// preview rows from making contradictory claims about the same completed hour.
func ValidateEvaluationInsights(evaluation *gateway.HomeInsightEvaluationEnvelope, insights []gateway.HomeSummaryInsight, componentStatus string) error {
	if err := ValidateEvaluation(evaluation); err != nil {
		return err
	}
	rules := make(map[string]gateway.HomeInsightEvaluationRule, len(evaluation.Rules))
	anyCrossed := false
	for _, rule := range evaluation.Rules {
		rules[rule.Type] = rule
		anyCrossed = anyCrossed || rule.ThresholdCrossed
	}
	for _, item := range insights {
		rule, ok := rules[item.Type]
		if !ok || !rule.ThresholdCrossed {
			return errors.New("current insight has no matching threshold crossing")
		}
		if item.Observed != nil && item.Observed.WindowEnd != evaluation.WindowEnd {
			return errors.New("current insight and evaluation windows disagree")
		}
	}
	if strings.EqualFold(componentStatus, "empty") && (anyCrossed || len(insights) > 0) {
		return errors.New("empty insight component contains a threshold crossing")
	}
	if anyCrossed && len(insights) == 0 {
		return errors.New("threshold crossing has no selected current insight")
	}
	if !anyCrossed && len(insights) > 0 {
		return errors.New("current insight exists without a detector crossing")
	}
	return nil
}

func EvaluationChecks(value *gateway.HomeInsightEvaluationEnvelope) []EvaluationCheck {
	if ValidateEvaluation(value) != nil {
		return nil
	}
	checks := make([]EvaluationCheck, 0, len(value.Rules))
	for _, rule := range value.Rules {
		check := EvaluationCheck{Label: evaluationRuleLabel(rule.Type), State: rule.Status}
		switch {
		case rule.Status != "ready":
			check.Value, check.Detail = "Incomplete", "Evidence for this check is still arriving"
		case rule.EvaluationOutcome == "no_eligible_subject":
			check.Value, check.Detail = "Not applicable", evaluationNoSubjectLabel(rule.Type)
		case rule.EvaluationOutcome == "baseline_zero":
			check.Value = evaluationNumber(rule.ObservedValue)
			check.Detail = "No non-zero typical hour yet"
		case rule.Ratio != nil:
			check.Value = fmt.Sprintf("%s× typical", formatDecimal(*rule.Ratio))
			check.Detail = evaluationMeasuredLabel(rule)
		default:
			check.Value, check.Detail = evaluationNumber(rule.ObservedValue), "Measured in the completed hour"
		}
		if rule.ThresholdCrossed {
			check.State = "crossed"
		}
		checks = append(checks, check)
	}
	return checks
}

func evaluationRuleLabel(value string) string {
	switch value {
	case "failure_spike":
		return "Contract failures"
	case "contract_deployments_spike":
		return "New contracts"
	case "transaction_activity_spike":
		return "Transactions"
	case "successful_activity_growth":
		return "Successful activity"
	case "failure_recovery":
		return "Failure recovery"
	case "new_contract_adoption":
		return "New contract adoption"
	default:
		return humanizeKey(value)
	}
}

func evaluationNoSubjectLabel(value string) string {
	switch value {
	case "failure_recovery":
		return "No recent failure spike to evaluate"
	case "new_contract_adoption":
		return "No newly deployed contract met the observation criteria"
	default:
		return "No eligible subject in this hour"
	}
}

func evaluationMeasuredLabel(rule gateway.HomeInsightEvaluationRule) string {
	if rule.ObservedValue == nil || rule.BaselineValue == nil {
		return "Measured against the supplied baseline"
	}
	return formatDecimal(*rule.ObservedValue) + " now, " + formatDecimal(*rule.BaselineValue) + " typical"
}

func evaluationNumber(value *float64) string {
	if value == nil || math.IsNaN(*value) || math.IsInf(*value, 0) {
		return "Not measured"
	}
	return formatDecimal(*value)
}
