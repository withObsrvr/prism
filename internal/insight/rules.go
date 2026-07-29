package insight

import (
	"errors"
	"fmt"
	"math"
	"net/url"
	"regexp"
	"strings"
	"time"

	"golang.org/x/text/language"
	"golang.org/x/text/message"

	"github.com/withObsrvr/prism/internal/gateway"
)

const EvidenceVersionV1 = "home_insight_evidence_v1"

const comparisonMethodV1 = "rolling_7d_median_prior_complete_hour"

var (
	insightIDPattern = regexp.MustCompile(`^hiev1_[A-Za-z0-9_-]{43}$`)
	printer          = message.NewPrinter(language.English)
)

func Interpret(item gateway.HomeSummaryInsight) (Interpretation, error) {
	if item.EvidenceVersion != EvidenceVersionV1 {
		return genericInterpretation(item), nil
	}
	if err := validateCommon(item); err != nil {
		return Interpretation{}, err
	}

	var result Interpretation
	switch item.Type {
	case "failure_spike":
		result = interpretFailure(item)
	case "contract_deployments_spike":
		result = interpretDeployments(item)
	case "transaction_activity_spike":
		result = interpretActivity(item)
	default:
		return genericInterpretation(item), nil
	}

	result.RuleID = item.Definition.RuleID
	result.RuleVersion = item.Definition.RuleVersion
	result.Status = item.Status
	result.WindowLabel = formatWindow(item.Observed.WindowStart, item.Observed.WindowEnd)
	result.ComparisonLabel = "Prior seven-day hourly median"
	result.Subject = subjectFor(item.Subject)
	result.EvidenceCount = item.EvidenceCount
	result.Metrics = commonMetrics(item)
	result.Evidence = evidenceLinks(item.EvidenceLocator)
	result.Caveats = caveatCopy(item.Caveats)
	result.Provenance = Provenance{
		Sources:               append([]string(nil), item.EvidenceProvenance.Sources...),
		CompleteThroughLedger: item.EvidenceProvenance.CompleteThroughLedger,
		UpdatedAt:             item.EvidenceProvenance.UpdatedAt,
	}
	return result, nil
}

func validateCommon(item gateway.HomeSummaryInsight) error {
	if !insightIDPattern.MatchString(item.InsightID) {
		return errors.New("invalid insight ID")
	}
	if item.Definition == nil || item.Observed == nil || item.Baseline == nil || item.Facts == nil || item.EvidenceLocator == nil || item.Caveats == nil || item.EvidenceProvenance == nil {
		return errors.New("incomplete insight evidence packet")
	}
	if strings.TrimSpace(item.Network) == "" || strings.TrimSpace(item.Subject.Kind) == "" || strings.TrimSpace(item.Subject.ID) == "" {
		return errors.New("insight network or subject is missing")
	}
	if item.Status != "ready" && item.Status != "partial" && item.Status != "stale" {
		return fmt.Errorf("unsupported insight status %q", item.Status)
	}
	if item.Definition.RuleVersion != "1" || item.Definition.ComparisonMethod != comparisonMethodV1 || item.Baseline.CompleteHourCount != 168 || item.Baseline.ZeroBaselinePolicy != "omit_ratio_insight" {
		return errors.New("unsupported insight definition")
	}
	if item.Observed.Value < 0 || item.Baseline.Value <= 0 || item.EvidenceCount < 0 {
		return errors.New("invalid insight measurements")
	}
	if item.Observed.FirstLedger <= 0 || item.Observed.FirstLedger > item.Observed.LastLedger || item.Observed.LastLedger > item.Observed.SourceLedger {
		return errors.New("invalid observed ledger bounds")
	}
	if item.EvidenceProvenance.CompleteThroughLedger < item.Observed.LastLedger || len(item.EvidenceProvenance.Sources) == 0 {
		return errors.New("insight provenance does not cover the observation")
	}
	for _, source := range item.EvidenceProvenance.Sources {
		if strings.TrimSpace(source) == "" {
			return errors.New("insight provenance contains an empty source")
		}
	}
	if _, err := time.Parse(time.RFC3339, item.EvidenceProvenance.UpdatedAt); err != nil {
		return errors.New("invalid insight provenance update time")
	}
	observedStart, err := time.Parse(time.RFC3339, item.Observed.WindowStart)
	if err != nil {
		return errors.New("invalid observed window start")
	}
	observedEnd, err := time.Parse(time.RFC3339, item.Observed.WindowEnd)
	if err != nil || observedEnd.Sub(observedStart) != time.Hour {
		return errors.New("invalid observed window end")
	}
	baselineStart, err := time.Parse(time.RFC3339, item.Baseline.WindowStart)
	if err != nil {
		return errors.New("invalid baseline window start")
	}
	baselineEnd, err := time.Parse(time.RFC3339, item.Baseline.WindowEnd)
	if err != nil || !baselineEnd.Equal(observedStart) || baselineEnd.Sub(baselineStart) != 168*time.Hour {
		return errors.New("baseline does not end at the observation")
	}
	wantRatio := item.Observed.Value / item.Baseline.Value
	if math.Abs(wantRatio-item.Ratio) > math.Max(0.000001, math.Abs(wantRatio)*0.000001) {
		return errors.New("insight ratio does not reconcile")
	}
	if item.Ratio < item.Definition.MinimumRatio || (item.Definition.MinimumObserved != nil && item.Observed.Value < *item.Definition.MinimumObserved) {
		return errors.New("insight does not satisfy its declared threshold")
	}
	if err := validateContributor(item.PrimaryContributor, item.Observed.SourceLedger); err != nil {
		return err
	}
	if err := validateCaveats(*item.Caveats, item.Status); err != nil {
		return err
	}
	return validateFacts(item)
}

func validateFacts(item gateway.HomeSummaryInsight) error {
	switch item.Type {
	case "failure_spike":
		facts := item.Facts.Failure
		if facts == nil || facts.Kind != item.Type || facts.AttemptCount != facts.SuccessCount+facts.FailureCount || float64(facts.FailureCount) != item.Observed.Value || facts.FailureCount != item.EvidenceCount || facts.NetworkFailureCount <= 0 || facts.FailureCount > facts.NetworkFailureCount || facts.DistinctTransactionCount > facts.FailureCount || !ratioMatches(facts.SubjectFailureShare, facts.FailureCount, facts.NetworkFailureCount) {
			return errors.New("failure insight facts do not reconcile")
		}
		if item.Subject.Kind != "contract" || item.Definition.RuleID != "contract_failure_spike" || !floatPointerEquals(item.Definition.MinimumObserved, 3) || item.Definition.MinimumRatio != 3 || item.EvidenceLocator.Kind != "contract_invocations" || item.EvidenceLocator.ContractID != item.Subject.ID || item.EvidenceLocator.Status != "failed" {
			return errors.New("failure insight rule or locator is invalid")
		}
		if item.PrimaryContributor != nil && (item.PrimaryContributor.Dimension != "function" || item.PrimaryContributor.Kind != "function") {
			return errors.New("failure insight contributor is invalid")
		}
		if facts.DominantResultCode != nil && validateCountContribution(*facts.DominantResultCode) != nil {
			return errors.New("failure result-code contribution does not reconcile")
		}
	case "contract_deployments_spike":
		facts := item.Facts.Deployment
		if facts == nil || facts.Kind != item.Type || float64(facts.DeploymentCount) != item.Observed.Value || facts.DeploymentCount != item.EvidenceCount || facts.PrimaryContract.CallsSinceDeployment != facts.PrimaryContract.SuccessCount+facts.PrimaryContract.FailureCount {
			return errors.New("deployment insight facts do not reconcile")
		}
		if item.Subject.Kind != "network" || item.Subject.ID != item.Network || item.Definition.RuleID != "network_contract_deployments_spike" || !floatPointerEquals(item.Definition.MinimumObserved, 2) || item.Definition.MinimumRatio != 3 || item.EvidenceLocator.Kind != "contract_deployments" {
			return errors.New("deployment insight rule or locator is invalid")
		}
		if err := validatePrimaryContract(facts.PrimaryContract, item); err != nil {
			return err
		}
		if item.PrimaryContributor != nil && (item.PrimaryContributor.Dimension != "deployed_contract_activity" || item.PrimaryContributor.Kind != "contract" || item.PrimaryContributor.Key != facts.PrimaryContract.ContractID || item.PrimaryContributor.Count != facts.PrimaryContract.CallsSinceDeployment) {
			return errors.New("deployment insight contributor is invalid")
		}
	case "transaction_activity_spike":
		facts := item.Facts.Activity
		if facts == nil || facts.Kind != item.Type || facts.IncludedTransactionCount != facts.SuccessfulTransactionCount+facts.FailedTransactionCount || facts.IncludedTransactionCount != facts.SorobanTransactionCount+facts.ClassicOnlyTransactionCount || float64(facts.IncludedTransactionCount) != item.Observed.Value || facts.IncludedTransactionCount != item.EvidenceCount {
			return errors.New("transaction activity insight facts do not reconcile")
		}
		if item.Subject.Kind != "network" || item.Subject.ID != item.Network || item.Definition.RuleID != "network_transaction_activity_spike" || item.Definition.MinimumObserved != nil || item.Definition.MinimumRatio != 2 || item.EvidenceLocator.Kind != "ledger_activity" {
			return errors.New("transaction activity rule or locator is invalid")
		}
	default:
		return nil
	}
	if item.EvidenceLocator.LedgerStart < item.Observed.FirstLedger || item.EvidenceLocator.LedgerEnd > item.Observed.LastLedger || item.EvidenceLocator.LedgerStart > item.EvidenceLocator.LedgerEnd {
		return errors.New("evidence locator is outside the observation")
	}
	return nil
}

func validateContributor(value *gateway.HomeInsightContribution, sourceLedger int64) error {
	if value == nil {
		return nil
	}
	if strings.TrimSpace(value.Dimension) == "" || strings.TrimSpace(value.Kind) == "" || strings.TrimSpace(value.Key) == "" || strings.TrimSpace(value.DenominatorName) == "" || value.Count < 0 || value.DenominatorValue <= 0 || value.Count > value.DenominatorValue || !ratioMatches(value.Share, value.Count, value.DenominatorValue) {
		return errors.New("insight contributor does not reconcile")
	}
	if value.FirstLedger <= 0 || value.FirstLedger > value.LastLedger || value.LastLedger > sourceLedger {
		return errors.New("insight contributor ledger bounds are invalid")
	}
	return nil
}

func validateCountContribution(value gateway.HomeInsightCountContribution) error {
	if strings.TrimSpace(value.Key) == "" || strings.TrimSpace(value.DenominatorName) == "" || value.Count < 0 || value.DenominatorValue <= 0 || value.Count > value.DenominatorValue || !ratioMatches(value.Share, value.Count, value.DenominatorValue) {
		return errors.New("count contribution does not reconcile")
	}
	return nil
}

func validatePrimaryContract(value gateway.HomeInsightPrimaryContract, item gateway.HomeSummaryInsight) error {
	if strings.TrimSpace(value.ContractID) == "" || value.DeploymentLedger < item.Observed.FirstLedger || value.DeploymentLedger > item.Observed.LastLedger {
		return errors.New("deployment primary contract is invalid")
	}
	deployedAt, deployedErr := time.Parse(time.RFC3339, value.DeployedAt)
	activityStart, startErr := time.Parse(time.RFC3339, value.ActivityWindowStart)
	activityEnd, endErr := time.Parse(time.RFC3339, value.ActivityWindowEnd)
	observedStart, _ := time.Parse(time.RFC3339, item.Observed.WindowStart)
	observedEnd, _ := time.Parse(time.RFC3339, item.Observed.WindowEnd)
	if deployedErr != nil || startErr != nil || endErr != nil || deployedAt.Before(observedStart) || deployedAt.After(observedEnd) || activityStart.Before(deployedAt) || activityEnd.Before(activityStart) || activityEnd.After(observedEnd) {
		return errors.New("deployment activity window is invalid")
	}
	return nil
}

func validateCaveats(caveats []gateway.HomeInsightCaveat, status string) error {
	allowed := map[string]struct{}{
		"identity_unavailable": {}, "function_distribution_unavailable": {}, "result_code_distribution_unavailable": {},
		"contributor_distribution_truncated": {}, "sample_evidence_unavailable": {}, "projection_stale": {},
	}
	hasStale := false
	for _, caveat := range caveats {
		if _, ok := allowed[caveat.Code]; !ok || strings.TrimSpace(caveat.Field) == "" {
			return errors.New("unsupported insight caveat")
		}
		hasStale = hasStale || caveat.Code == "projection_stale"
	}
	if status == "stale" && !hasStale {
		return errors.New("stale insight is missing its projection caveat")
	}
	if status == "partial" && len(caveats) == 0 {
		return errors.New("partial insight is missing its evidence caveat")
	}
	return nil
}

func ratioMatches(value float64, count, denominator int64) bool {
	if denominator <= 0 {
		return false
	}
	want := float64(count) / float64(denominator)
	return math.Abs(value-want) <= math.Max(0.000001, math.Abs(want)*0.000001)
}

func floatPointerEquals(value *float64, expected float64) bool {
	return value != nil && math.Abs(*value-expected) <= 0.000001
}

func interpretFailure(item gateway.HomeSummaryInsight) Interpretation {
	facts := item.Facts.Failure
	result := Interpretation{
		Title:    "Contract failures rose above their usual hour",
		Summary:  printer.Sprintf("Contract invocation failures were %.2g times the seven-day median in the last completed hour: %d failures versus a median of %s.", item.Ratio, facts.FailureCount, formatDecimal(item.Baseline.Value)),
		Severity: "critical",
	}
	if contributor := item.PrimaryContributor; contributor != nil && contributor.Dimension == "function" {
		result.Detail = printer.Sprintf("%s accounted for %d of %d %s (%s).", contributor.Key, contributor.Count, contributor.DenominatorValue, denominatorLabel(contributor.DenominatorName), formatPercent(contributor.Share))
	}
	return result
}

func interpretDeployments(item gateway.HomeSummaryInsight) Interpretation {
	facts := item.Facts.Deployment
	result := Interpretation{
		Title:    "Contract deployments increased",
		Summary:  printer.Sprintf("%d contracts were deployed in the last completed hour, %.2g times the seven-day median of %s.", facts.DeploymentCount, item.Ratio, formatDecimal(item.Baseline.Value)),
		Severity: "signal",
	}
	primary := facts.PrimaryContract
	if primary.ContractID != "" {
		result.Detail = printer.Sprintf("The most active new contract received %d calls from %d callers after deployment; %d succeeded and %d failed.", primary.CallsSinceDeployment, primary.DistinctCallerCount, primary.SuccessCount, primary.FailureCount)
	}
	return result
}

func interpretActivity(item gateway.HomeSummaryInsight) Interpretation {
	facts := item.Facts.Activity
	result := Interpretation{
		Title:    "Transaction activity increased",
		Summary:  printer.Sprintf("Transaction activity reached %d in the last completed hour, %.2g times the seven-day median of %s.", facts.IncludedTransactionCount, item.Ratio, formatDecimal(item.Baseline.Value)),
		Severity: "info",
	}
	if contributor := item.PrimaryContributor; contributor != nil {
		result.Detail = printer.Sprintf("%s represented %d of %d %s (%s).", humanizeKey(contributor.Key), contributor.Count, contributor.DenominatorValue, denominatorLabel(contributor.DenominatorName), formatPercent(contributor.Share))
	}
	return result
}

func genericInterpretation(item gateway.HomeSummaryInsight) Interpretation {
	observed := item.ObservedValue
	baseline := item.BaselineValue
	windowStart := item.WindowStart
	windowEnd := item.WindowEnd
	if item.Observed != nil {
		observed = item.Observed.Value
		windowStart = item.Observed.WindowStart
		windowEnd = item.Observed.WindowEnd
	}
	if item.Baseline != nil {
		baseline = item.Baseline.Value
	}
	result := Interpretation{
		Title:           "Measured change",
		Summary:         printer.Sprintf("The completed-hour observation was %s, compared with a baseline of %s.", formatDecimal(observed), formatDecimal(baseline)),
		Severity:        "neutral",
		Status:          item.Status,
		WindowLabel:     formatWindow(windowStart, windowEnd),
		ComparisonLabel: firstNonEmpty(item.ComparisonMethod, "Comparison supplied by the Query API"),
		Subject:         subjectFor(item.Subject),
		EvidenceCount:   item.EvidenceCount,
		Metrics: []Metric{
			{Label: "Observed", Value: formatDecimal(observed)},
			{Label: "Baseline", Value: formatDecimal(baseline)},
		},
		Caveats: []string{"Prism does not have a supported evidence rule for this packet version, so it is showing measured values without additional interpretation."},
		Generic: true,
	}
	if item.EvidenceVersion == "" {
		result.Caveats[0] = "This is a compatibility preview. Detailed versioned evidence is not available for this item."
	}
	return result
}

func commonMetrics(item gateway.HomeSummaryInsight) []Metric {
	return []Metric{
		{Label: "Observed", Value: formatDecimal(item.Observed.Value)},
		{Label: "Baseline", Value: formatDecimal(item.Baseline.Value)},
		{Label: "Ratio", Value: fmt.Sprintf("%.2g×", item.Ratio)},
		{Label: "Evidence", Value: printer.Sprintf("%d", item.EvidenceCount)},
	}
}

func subjectFor(subject gateway.HomeSummaryInsightSubject) Subject {
	result := Subject{ID: subject.ID, Label: subject.ID}
	if subject.Kind == "contract" && subject.ID != "" {
		result.Href = "/v2/contract/" + url.PathEscape(subject.ID)
	}
	if subject.Identity != nil && strings.TrimSpace(subject.Identity.DisplayName) != "" {
		result.Label = subject.Identity.DisplayName
		result.IdentityDetail = identityDetail(*subject.Identity)
	} else if subject.Kind == "contract" {
		result.IdentityDetail = "Raw contract identity"
	}
	return result
}

func identityDetail(identity gateway.HomeInsightIdentity) string {
	status := humanizeKey(identity.VerificationStatus)
	source := humanizeKey(identity.Source)
	if status == "" {
		return source
	}
	if source == "" {
		return status + " identity"
	}
	return status + " identity from " + source
}

func evidenceLinks(locator *gateway.HomeInsightEvidenceLocator) []EvidenceLink {
	if locator == nil || locator.LedgerStart <= 0 || locator.LedgerEnd < locator.LedgerStart {
		return nil
	}
	query := url.Values{}
	query.Set("from_ledger", fmt.Sprintf("%d", locator.LedgerStart))
	query.Set("to_ledger", fmt.Sprintf("%d", locator.LedgerEnd))
	query.Set("time", "coverage")
	label := "Inspect ledger-range evidence"
	switch locator.Kind {
	case "contract_invocations":
		if locator.ContractID != "" {
			query.Set("contract", locator.ContractID)
		}
		if locator.Status == "failed" {
			query.Set("status", "failed")
		} else if locator.Status == "successful" {
			query.Set("status", "success")
		}
		label = "Inspect matching contract activity"
	case "contract_deployments":
		if locator.ContractID != "" {
			query.Set("contract", locator.ContractID)
		}
		label = "Inspect the deployment ledger range"
	case "transactions":
		if locator.Status == "failed" {
			query.Set("status", "failed")
		} else if locator.Status == "successful" {
			query.Set("status", "success")
		}
	case "ledger_activity":
		label = "Inspect activity in this ledger range"
	default:
		return nil
	}
	return []EvidenceLink{{Label: label, Href: "/v2/explore?" + query.Encode()}}
}

func caveatCopy(caveats *[]gateway.HomeInsightCaveat) []string {
	if caveats == nil {
		return nil
	}
	result := make([]string, 0, len(*caveats))
	for _, caveat := range *caveats {
		var text string
		switch caveat.Code {
		case "identity_unavailable":
			text = "A verified display identity is not available."
		case "function_distribution_unavailable":
			text = "Function-level contribution evidence is incomplete."
		case "result_code_distribution_unavailable":
			text = "Failure-code distribution evidence is incomplete."
		case "contributor_distribution_truncated":
			text = "The contributor list is bounded and does not show every contributor."
		case "sample_evidence_unavailable":
			text = "Representative transaction samples are unavailable."
		case "projection_stale":
			text = "This evidence projection is delayed."
		default:
			text = "Some supporting evidence is incomplete."
		}
		if !contains(result, text) {
			result = append(result, text)
		}
	}
	return result
}

func formatWindow(startRaw, endRaw string) string {
	start, startErr := time.Parse(time.RFC3339, startRaw)
	end, endErr := time.Parse(time.RFC3339, endRaw)
	if startErr != nil || endErr != nil {
		return "Exact comparison window unavailable"
	}
	return start.UTC().Format("Jan 2, 15:04") + " to " + end.UTC().Format("15:04 UTC")
}

func formatDecimal(value float64) string {
	if math.Abs(value-math.Round(value)) < 0.0000001 {
		return printer.Sprintf("%.0f", value)
	}
	return printer.Sprintf("%.2f", value)
}

func formatPercent(value float64) string {
	return fmt.Sprintf("%.0f%%", value*100)
}

func humanizeKey(value string) string {
	return strings.TrimSpace(strings.ReplaceAll(value, "_", " "))
}

func denominatorLabel(value string) string {
	switch value {
	case "subject_failure_count":
		return "failures"
	case "included_operation_count":
		return "included operations"
	case "cohort_calls_since_deployment":
		return "calls since deployment"
	default:
		return humanizeKey(value)
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
}

func contains(values []string, candidate string) bool {
	for _, value := range values {
		if value == candidate {
			return true
		}
	}
	return false
}
