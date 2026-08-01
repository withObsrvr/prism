package insight

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/withObsrvr/prism/internal/gateway"
)

const comparisonMethodAdoption = "post_deployment_observation_window"

type v2RuleSpec struct {
	family, direction, subjectKind, ruleID, comparison, ratioComparison string
}

var v2Rules = map[string]v2RuleSpec{
	"successful_activity_growth": {"activity", "positive", "network", "network_successful_activity_growth", comparisonMethodV1, "at_least"},
	"failure_recovery":           {"recovery", "positive", "contract", "contract_failure_recovery", comparisonMethodV1, "at_most"},
	"new_contract_adoption":      {"adoption", "positive", "contract", "contract_new_adoption", comparisonMethodAdoption, "at_least"},
}

func validateV2(item gateway.HomeSummaryInsight) error {
	spec, ok := v2Rules[item.Type]
	if !ok {
		return fmt.Errorf("unsupported insight evidence v2 type %q", item.Type)
	}
	if !insightIDPattern.MatchString(item.InsightID) || !strings.HasPrefix(item.InsightID, "hiev2_") {
		return errors.New("invalid evidence v2 insight ID")
	}
	if item.Definition == nil || item.Observed == nil || item.Baseline == nil || item.Facts == nil || item.EvidenceLocator == nil || item.Caveats == nil || item.EvidenceProvenance == nil {
		return errors.New("incomplete insight evidence v2 packet")
	}
	if item.Family != spec.family || item.Direction != spec.direction || item.Subject.Kind != spec.subjectKind || strings.TrimSpace(item.Subject.ID) == "" || strings.TrimSpace(item.Network) == "" {
		return errors.New("insight evidence v2 classification or subject is invalid")
	}
	if spec.subjectKind == "network" && item.Subject.ID != item.Network {
		return errors.New("network insight subject does not match its network")
	}
	definition := item.Definition
	if definition.RuleID != spec.ruleID || definition.RuleVersion != "1" || definition.ComparisonMethod != spec.comparison || definition.RatioComparison != spec.ratioComparison {
		return errors.New("unsupported insight evidence v2 definition")
	}
	if item.Status != "ready" && item.Status != "partial" && item.Status != "stale" {
		return fmt.Errorf("unsupported insight status %q", item.Status)
	}
	if item.Observed.Value < 0 || item.Baseline.Value <= 0 || item.EvidenceCount < 0 {
		return errors.New("invalid insight evidence v2 measurements")
	}
	if item.Observed.FirstLedger <= 0 || item.Observed.FirstLedger > item.Observed.LastLedger || item.Observed.LastLedger > item.Observed.SourceLedger || item.EvidenceProvenance.CompleteThroughLedger < item.Observed.LastLedger {
		return errors.New("invalid insight evidence v2 ledger bounds")
	}
	observedStart, startErr := time.Parse(time.RFC3339, item.Observed.WindowStart)
	observedEnd, endErr := time.Parse(time.RFC3339, item.Observed.WindowEnd)
	baselineStart, baselineStartErr := time.Parse(time.RFC3339, item.Baseline.WindowStart)
	baselineEnd, baselineEndErr := time.Parse(time.RFC3339, item.Baseline.WindowEnd)
	if startErr != nil || endErr != nil || baselineStartErr != nil || baselineEndErr != nil || !observedEnd.After(observedStart) || !baselineEnd.Equal(observedStart) {
		return errors.New("invalid insight evidence v2 windows")
	}
	if spec.comparison == comparisonMethodV1 && (observedEnd.Sub(observedStart) != time.Hour || baselineEnd.Sub(baselineStart) != 168*time.Hour || item.Baseline.CompleteHourCount != 168) {
		return errors.New("invalid rolling evidence v2 coverage")
	}
	if spec.comparison == comparisonMethodAdoption && (!baselineStart.Equal(baselineEnd) || item.Baseline.CompleteHourCount != 0) {
		return errors.New("invalid adoption evidence coverage")
	}
	if item.Baseline.ZeroBaselinePolicy != "omit_ratio_insight" || !nearlyEqual(item.Ratio, item.Observed.Value/item.Baseline.Value) {
		return errors.New("insight evidence v2 ratio does not reconcile")
	}
	if spec.ratioComparison == "at_least" && (item.Ratio < definition.MinimumRatio || (definition.MinimumObserved != nil && item.Observed.Value < *definition.MinimumObserved)) {
		return errors.New("insight evidence v2 does not cross its minimum")
	}
	if spec.ratioComparison == "at_most" && item.Ratio > definition.MinimumRatio {
		return errors.New("insight evidence v2 is above its maximum")
	}
	if len(item.EvidenceProvenance.Sources) == 0 {
		return errors.New("insight evidence v2 provenance is missing")
	}
	for _, source := range item.EvidenceProvenance.Sources {
		if strings.TrimSpace(source) == "" {
			return errors.New("insight evidence v2 provenance contains an empty source")
		}
	}
	if _, err := time.Parse(time.RFC3339, item.EvidenceProvenance.UpdatedAt); err != nil {
		return errors.New("invalid insight evidence v2 update time")
	}
	if err := validateContributor(item.PrimaryContributor, item.Observed.SourceLedger); err != nil {
		return err
	}
	if err := validateCaveats(*item.Caveats, item.Status); err != nil {
		return err
	}
	if item.EvidenceLocator.LedgerStart != item.Observed.FirstLedger || item.EvidenceLocator.LedgerEnd != item.Observed.LastLedger {
		return errors.New("insight evidence v2 locator is outside the observation")
	}
	return validateV2Facts(item)
}

func validateV2Facts(item gateway.HomeSummaryInsight) error {
	switch item.Type {
	case "successful_activity_growth":
		facts := item.Facts.Growth
		if !floatPointerEquals(item.Definition.MinimumObserved, 500) || !nearlyEqual(item.Definition.MinimumRatio, 2) {
			return errors.New("successful activity rule thresholds are invalid")
		}
		if facts == nil || facts.Kind != item.Type || facts.IncludedTransactionCount != facts.SuccessfulTransactionCount+facts.FailedTransactionCount || facts.IncludedTransactionCount != facts.SorobanTransactionCount+facts.ClassicOnlyTransactionCount || facts.SuccessfulTransactionCount != item.EvidenceCount || float64(facts.SuccessfulTransactionCount) != item.Observed.Value || !nearlyEqual(facts.BaselineSuccessfulTransactionCount, item.Baseline.Value) {
			return errors.New("successful activity evidence does not reconcile")
		}
		failureRate := float64(facts.FailedTransactionCount) / float64(facts.IncludedTransactionCount)
		if !nearlyEqual(facts.CurrentFailureRate, failureRate) || facts.CurrentFailureRate > facts.MaximumFailureRate || facts.CurrentFailureRate > facts.BaselineFailureRate+facts.FailureRateTolerance || item.EvidenceLocator.Kind != "ledger_activity" || item.EvidenceLocator.Status != "successful" {
			return errors.New("successful activity evidence failed its quality guard")
		}
		if item.PrimaryContributor != nil && (item.PrimaryContributor.Dimension != "operation_category" || item.PrimaryContributor.Kind != "category" || item.PrimaryContributor.DenominatorValue != facts.IncludedOperationCount) {
			return errors.New("successful activity contributor is invalid")
		}
	case "failure_recovery":
		facts := item.Facts.Recovery
		if !floatPointerEquals(item.Definition.MinimumObserved, 20) || !nearlyEqual(item.Definition.MinimumRatio, 1) {
			return errors.New("recovery rule thresholds are invalid")
		}
		if facts == nil || facts.Kind != item.Type || facts.CurrentAttemptCount != facts.CurrentSuccessCount+facts.CurrentFailureCount || facts.CurrentAttemptCount != item.EvidenceCount || float64(facts.CurrentFailureCount) != item.Observed.Value || !nearlyEqual(facts.NormalRangeFailureCount, item.Baseline.Value) || float64(facts.CurrentFailureCount) > facts.NormalRangeFailureCount {
			return errors.New("recovery evidence does not reconcile")
		}
		if facts.CurrentAttemptCount < facts.MinimumAttemptCount || float64(facts.CurrentAttemptCount) < facts.BaselineAttemptCount*facts.ActivityFloorRatio || !strings.HasPrefix(facts.PriorInsightID, "hiev1_") || facts.PriorFailureCount <= 0 || item.EvidenceLocator.Kind != "contract_invocations" || item.EvidenceLocator.ContractID != item.Subject.ID || item.EvidenceLocator.Status != "successful" {
			return errors.New("recovery evidence does not prove meaningful restored activity")
		}
		priorStart, startErr := time.Parse(time.RFC3339, facts.PriorWindowStart)
		priorEnd, endErr := time.Parse(time.RFC3339, facts.PriorWindowEnd)
		observedStart, _ := time.Parse(time.RFC3339, item.Observed.WindowStart)
		if startErr != nil || endErr != nil || priorEnd.Sub(priorStart) != time.Hour || priorEnd.After(observedStart) {
			return errors.New("recovery prior-spike window is invalid")
		}
	case "new_contract_adoption":
		facts := item.Facts.Adoption
		if !floatPointerEquals(item.Definition.MinimumObserved, 25) || !nearlyEqual(item.Definition.MinimumRatio, 1) {
			return errors.New("adoption rule thresholds are invalid")
		}
		if facts == nil || facts.Kind != item.Type || facts.ContractID != item.Subject.ID || facts.CallsSinceDeployment != facts.SuccessCount+facts.FailureCount || facts.CallsSinceDeployment != item.EvidenceCount || float64(facts.CallsSinceDeployment) != item.Observed.Value || facts.DistinctCallerCount > facts.CallsSinceDeployment {
			return errors.New("new contract adoption evidence does not reconcile")
		}
		successRate := float64(facts.SuccessCount) / float64(facts.CallsSinceDeployment)
		if !nearlyEqual(facts.SuccessRate, successRate) || facts.CallsSinceDeployment < facts.MinimumCalls || facts.DistinctCallerCount < facts.MinimumDistinctCallers || facts.SuccessRate < facts.MinimumSuccessRate || facts.AdoptionAgeSeconds > facts.MaximumAdoptionAgeSeconds || facts.DeploymentLedger != item.Observed.FirstLedger || item.EvidenceLocator.Kind != "contract_deployments" || item.EvidenceLocator.ContractID != item.Subject.ID {
			return errors.New("new contract adoption evidence does not satisfy its thresholds")
		}
		deployedAt, deployedErr := time.Parse(time.RFC3339, facts.DeployedAt)
		windowEnd, windowErr := time.Parse(time.RFC3339, facts.ObservationWindowEnd)
		observedStart, _ := time.Parse(time.RFC3339, item.Observed.WindowStart)
		observedEnd, _ := time.Parse(time.RFC3339, item.Observed.WindowEnd)
		if deployedErr != nil || windowErr != nil || !deployedAt.Equal(observedStart) || !windowEnd.Equal(observedEnd) || int64(observedEnd.Sub(observedStart).Seconds()) != facts.AdoptionAgeSeconds {
			return errors.New("new contract adoption window is invalid")
		}
	default:
		return errors.New("unsupported positive insight facts")
	}
	return nil
}

func interpretGrowth(item gateway.HomeSummaryInsight) Interpretation {
	facts := item.Facts.Growth
	return Interpretation{
		Title:    "Successful network activity grew",
		Summary:  printer.Sprintf("%d successful transactions were included, %s× the typical hour, while the failure rate remained %s.", facts.SuccessfulTransactionCount, formatDecimal(item.Ratio), formatPercent(facts.CurrentFailureRate)),
		Detail:   printer.Sprintf("%d of %d included transactions succeeded. The comparison only qualifies when the failure rate remains within its normal guardrail.", facts.SuccessfulTransactionCount, facts.IncludedTransactionCount),
		Severity: "positive",
	}
}

func interpretRecovery(item gateway.HomeSummaryInsight) Interpretation {
	facts := item.Facts.Recovery
	return Interpretation{
		Title:    "Contract failures returned to their normal range",
		Summary:  printer.Sprintf("Failures fell from %d in the earlier spike to %d while the contract still processed %d calls.", facts.PriorFailureCount, facts.CurrentFailureCount, facts.CurrentAttemptCount),
		Detail:   printer.Sprintf("%d of %d current calls succeeded, so the improvement is not explained by inactivity.", facts.CurrentSuccessCount, facts.CurrentAttemptCount),
		Severity: "positive",
	}
}

func interpretAdoption(item gateway.HomeSummaryInsight) Interpretation {
	facts := item.Facts.Adoption
	callerWord := "callers"
	if facts.DistinctCallerCount == 1 {
		callerWord = "caller"
	}
	return Interpretation{
		Title:    "A new contract is gaining use",
		Summary:  printer.Sprintf("The contract received %d calls from %d distinct %s after deployment, with a %s success rate.", facts.CallsSinceDeployment, facts.DistinctCallerCount, callerWord, formatPercent(facts.SuccessRate)),
		Detail:   firstNonEmpty(adoptionFunctionDetail(facts.TopFunction), "Usage crossed Prism's adoption floor within the first 72 hours after deployment."),
		Severity: "positive",
	}
}

func adoptionFunctionDetail(function string) string {
	if strings.TrimSpace(function) == "" {
		return ""
	}
	return "The most-used contract function was " + function + "."
}

func nearlyEqual(left, right float64) bool {
	return math.Abs(left-right) <= math.Max(0.000001, math.Abs(right)*0.000001)
}
