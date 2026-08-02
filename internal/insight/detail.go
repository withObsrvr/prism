package insight

import (
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/withObsrvr/prism/internal/gateway"
)

const detailSampleSelectionV1 = "highest_impact_then_earliest_latest"

var transactionHashPattern = regexp.MustCompile(`^[0-9a-fA-F]{64}$`)

type DetailInterpretation struct {
	Interpretation
	RuleSummary  string
	MatchSummary string
	Contributors []DetailContributor
	Samples      []DetailSample
}

type DetailContributor struct {
	Rank        int
	Dimension   string
	Label       string
	Key         string
	Count       int64
	CountLabel  string
	ShareLabel  string
	LedgerLabel string
	Href        string
}

type DetailSample struct {
	Rank           int
	Kind           string
	KindLabel      string
	LedgerSequence int64
	LedgerLabel    string
	LedgerHref     string
	TxHash         string
	TxLabel        string
	TxHref         string
	Context        string
	ContractID     string
	ContractLabel  string
	ContractHref   string
}

// InterpretDetail validates a retained evidence packet before turning its
// bounded contributor and sample evidence into Prism-owned presentation facts.
func InterpretDetail(packet gateway.HomeInsightDetailResponse) (DetailInterpretation, error) {
	if packet.EvidenceVersion != EvidenceVersionV1 && packet.EvidenceVersion != EvidenceVersionV2 {
		return DetailInterpretation{}, fmt.Errorf("unsupported insight evidence version %q", packet.EvidenceVersion)
	}
	if err := validateStableInsightID(packet.HomeSummaryInsight); err != nil {
		return DetailInterpretation{}, err
	}
	base, err := Interpret(packet.HomeSummaryInsight)
	if err != nil {
		return DetailInterpretation{}, err
	}
	if base.Generic {
		return DetailInterpretation{}, errors.New("unsupported detail interpretation")
	}
	if packet.Subject.Identity != nil && !completeDetailIdentity(packet.Subject.Identity) {
		return DetailInterpretation{}, errors.New("insight subject identity is incomplete")
	}
	if packet.PrimaryContributor != nil && packet.PrimaryContributor.Identity != nil && !completeDetailIdentity(packet.PrimaryContributor.Identity) {
		return DetailInterpretation{}, errors.New("insight primary contributor identity is incomplete")
	}
	if err := validateDetailContributors(packet); err != nil {
		return DetailInterpretation{}, err
	}
	if err := validateDetailSamples(packet); err != nil {
		return DetailInterpretation{}, err
	}

	ruleSummary := "Prism compared one completed UTC hour with the median of the previous 168 complete hourly observations."
	if packet.Definition.ComparisonMethod == comparisonMethodAdoption {
		ruleSummary = "Prism measured verified use from contract deployment through the end of the supplied observation window."
	}
	result := DetailInterpretation{
		Interpretation: base,
		RuleSummary:    ruleSummary,
		MatchSummary:   detailMatchSummary(packet.HomeSummaryInsight),
	}
	for _, value := range packet.Contributors {
		if value.Identity == nil && packet.PrimaryContributor != nil && value.Dimension == packet.PrimaryContributor.Dimension && value.Key == packet.PrimaryContributor.Key {
			value.Identity = packet.PrimaryContributor.Identity
		}
		result.Contributors = append(result.Contributors, detailContributor(value))
	}
	sort.SliceStable(result.Contributors, func(i, j int) bool {
		if result.Contributors[i].Dimension == result.Contributors[j].Dimension {
			return result.Contributors[i].Rank < result.Contributors[j].Rank
		}
		return result.Contributors[i].Dimension < result.Contributors[j].Dimension
	})
	for _, value := range packet.Samples {
		result.Samples = append(result.Samples, detailSample(value))
	}
	sort.SliceStable(result.Samples, func(i, j int) bool {
		if result.Samples[i].Kind == result.Samples[j].Kind {
			return result.Samples[i].Rank < result.Samples[j].Rank
		}
		return result.Samples[i].Kind < result.Samples[j].Kind
	})
	return result, nil
}

func validateStableInsightID(item gateway.HomeSummaryInsight) error {
	if item.Definition == nil || item.Observed == nil {
		return errors.New("insight identity inputs are incomplete")
	}
	windowEnd, err := time.Parse(time.RFC3339, item.Observed.WindowEnd)
	if err != nil {
		return errors.New("invalid insight identity window")
	}
	canonical := strings.Join([]string{
		item.Network,
		item.Type,
		item.Subject.Kind,
		item.Subject.ID,
		windowEnd.UTC().Format(time.RFC3339),
		item.Definition.RuleID + "@" + item.Definition.RuleVersion,
	}, "\n")
	digest := sha256.Sum256([]byte(canonical))
	prefix := "hiev1_"
	if item.EvidenceVersion == EvidenceVersionV2 {
		prefix = "hiev2_"
	}
	want := prefix + base64.RawURLEncoding.EncodeToString(digest[:])
	if item.InsightID != want {
		return errors.New("insight ID does not match its evidence identity")
	}
	return nil
}

func validateDetailContributors(packet gateway.HomeInsightDetailResponse) error {
	seen := make(map[string]map[int]struct{})
	counts := make(map[string]int)
	for index := range packet.Contributors {
		value := &packet.Contributors[index]
		if value.Rank < 1 || value.Rank > 25 {
			return errors.New("insight contributor rank is outside the bounded detail contract")
		}
		if err := validateContributor(value, packet.Observed.SourceLedger); err != nil {
			return err
		}
		if value.Identity != nil && !completeDetailIdentity(value.Identity) {
			return errors.New("insight contributor identity is incomplete")
		}
		if seen[value.Dimension] == nil {
			seen[value.Dimension] = make(map[int]struct{})
		}
		if _, duplicate := seen[value.Dimension][value.Rank]; duplicate {
			return errors.New("insight contributor rank is duplicated within its dimension")
		}
		seen[value.Dimension][value.Rank] = struct{}{}
		counts[value.Dimension]++
		if counts[value.Dimension] > 25 {
			return errors.New("insight contributor dimension exceeds its bounded cap")
		}
	}
	return nil
}

func completeDetailIdentity(value *gateway.HomeInsightIdentity) bool {
	return value != nil && strings.TrimSpace(value.DisplayName) != "" && strings.TrimSpace(value.Kind) != "" && strings.TrimSpace(value.VerificationStatus) != "" && strings.TrimSpace(value.Source) != ""
}

func validateDetailSamples(packet gateway.HomeInsightDetailResponse) error {
	seen := make(map[string]map[int]struct{})
	counts := make(map[string]int)
	allowed := detailSampleKind(packet.Type)
	for _, value := range packet.Samples {
		if value.SampleKind != allowed || value.Rank < 1 || value.Rank > 10 || value.SelectionMethod != detailSampleSelectionV1 {
			return errors.New("insight sample is outside the bounded detail contract")
		}
		if value.LedgerSequence < packet.Observed.FirstLedger || value.LedgerSequence > packet.Observed.LastLedger || !transactionHashPattern.MatchString(value.TransactionHash) {
			return errors.New("insight sample evidence is outside the observation or malformed")
		}
		if value.OperationIndex != nil && *value.OperationIndex < 0 {
			return errors.New("insight sample operation index is invalid")
		}
		if value.EventIndex != nil && *value.EventIndex < 0 {
			return errors.New("insight sample event index is invalid")
		}
		if seen[value.SampleKind] == nil {
			seen[value.SampleKind] = make(map[int]struct{})
		}
		if _, duplicate := seen[value.SampleKind][value.Rank]; duplicate {
			return errors.New("insight sample rank is duplicated within its kind")
		}
		seen[value.SampleKind][value.Rank] = struct{}{}
		counts[value.SampleKind]++
		if counts[value.SampleKind] > 10 {
			return errors.New("insight sample kind exceeds its bounded cap")
		}
	}
	return nil
}

func detailSampleKind(insightType string) string {
	switch insightType {
	case "failure_spike":
		return "failed_transaction"
	case "contract_deployments_spike":
		return "deployment_transaction"
	case "transaction_activity_spike":
		return "activity_transaction"
	case "successful_activity_growth":
		return "successful_transaction"
	case "failure_recovery":
		return "successful_transaction"
	case "new_contract_adoption":
		return "adoption_transaction"
	default:
		return ""
	}
}

func detailMatchSummary(item gateway.HomeSummaryInsight) string {
	if item.Type == "failure_recovery" {
		return fmt.Sprintf("This rule requires failures at or below %s× the normal range while meaningful call activity continues. The measured hour reached %s×.", formatDecimal(item.Definition.MinimumRatio), formatDecimal(item.Ratio))
	}
	if item.Type == "new_contract_adoption" {
		return fmt.Sprintf("This rule requires at least %s calls from three callers with an 80%% success rate during the first 72 hours. The contract reached %s calls.", formatDecimal(*item.Definition.MinimumObserved), formatDecimal(item.Observed.Value))
	}
	minimum := ""
	if item.Definition.MinimumObserved != nil {
		minimum = fmt.Sprintf(" and at least %s observations", formatDecimal(*item.Definition.MinimumObserved))
	}
	return fmt.Sprintf("This rule requires at least %s× the typical hour%s. The measured hour reached %s×.", formatDecimal(item.Definition.MinimumRatio), minimum, formatDecimal(item.Ratio))
}

func detailContributor(value gateway.HomeInsightContribution) DetailContributor {
	label := humanizeKey(value.Key)
	href := ""
	if value.Dimension == "result_code" {
		label = detailResultLabel(value.Key)
	}
	if value.Identity != nil && strings.TrimSpace(value.Identity.DisplayName) != "" {
		label = value.Identity.DisplayName
	}
	if value.Kind == "contract" {
		href = "/v2/contract/" + url.PathEscape(value.Key)
		if label == value.Key {
			label = shortEvidenceID(value.Key)
		}
	}
	return DetailContributor{
		Rank:        value.Rank,
		Dimension:   detailDimensionLabel(value.Dimension),
		Label:       label,
		Key:         value.Key,
		Count:       value.Count,
		CountLabel:  printer.Sprintf("%d of %d %s", value.Count, value.DenominatorValue, denominatorLabel(value.DenominatorName)),
		ShareLabel:  formatPercent(value.Share),
		LedgerLabel: detailLedgerRange(value.FirstLedger, value.LastLedger),
		Href:        href,
	}
}

func detailSample(value gateway.HomeInsightSample) DetailSample {
	result := DetailSample{
		Rank:           value.Rank,
		Kind:           value.SampleKind,
		KindLabel:      detailSampleLabel(value.SampleKind),
		LedgerSequence: value.LedgerSequence,
		LedgerLabel:    printer.Sprintf("%d", value.LedgerSequence),
		LedgerHref:     fmt.Sprintf("/v2/ledger/%d", value.LedgerSequence),
		TxHash:         value.TransactionHash,
		TxLabel:        shortEvidenceID(value.TransactionHash),
		TxHref:         "/v2/tx/" + url.PathEscape(value.TransactionHash),
		ContractID:     value.ContractID,
		ContractLabel:  shortEvidenceID(value.ContractID),
	}
	if value.ContractID != "" {
		result.ContractHref = "/v2/contract/" + url.PathEscape(value.ContractID)
	}
	context := make([]string, 0, 3)
	if value.FunctionName != "" {
		context = append(context, "Function "+value.FunctionName)
	}
	if value.ResultCode != "" {
		context = append(context, "Result "+detailResultLabel(value.ResultCode))
	}
	if value.OperationIndex != nil {
		context = append(context, fmt.Sprintf("Operation %d", *value.OperationIndex+1))
	}
	result.Context = strings.Join(context, " · ")
	return result
}

func detailDimensionLabel(value string) string {
	switch value {
	case "function", "primary_contract_function":
		return "Function"
	case "result_code":
		return "Result category"
	case "deployed_contract_activity":
		return "New contract"
	case "operation_category":
		return "Operation category"
	case "adopted_contract_function":
		return "Contract function"
	default:
		return humanizeKey(value)
	}
}

func detailSampleLabel(value string) string {
	switch value {
	case "failed_transaction":
		return "Failed transaction"
	case "deployment_transaction":
		return "Deployment transaction"
	case "activity_transaction":
		return "Activity transaction"
	case "successful_transaction":
		return "Successful transaction"
	case "adoption_transaction":
		return "Adoption transaction"
	default:
		return "Representative transaction"
	}
}

// detailResultLabel keeps protocol codes available as evidence keys while
// leading with language that explains the category to a non-protocol reader.
func detailResultLabel(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	normalized = strings.TrimPrefix(normalized, "invoke_")
	switch normalized {
	case "host_function_trapped", "host_function_result_code_invoke_host_function_trapped":
		return "Contract stopped unexpectedly"
	case "host_function_resource_limit_exceeded":
		return "Resource limit exceeded"
	case "soroban_invalid":
		return "Invalid Soroban transaction"
	case "insufficient_fee":
		return "Insufficient fee"
	case "bad_sequence":
		return "Invalid account sequence"
	case "internal_error":
		return "Network processing error"
	default:
		return humanizeKey(value)
	}
}

func detailLedgerRange(first, last int64) string {
	if first == last {
		return "Ledger " + printer.Sprintf("%d", first)
	}
	return "Ledgers " + printer.Sprintf("%d", first) + " to " + printer.Sprintf("%d", last)
}

func shortEvidenceID(value string) string {
	value = strings.TrimSpace(value)
	if len(value) <= 16 {
		return value
	}
	return value[:8] + "…" + value[len(value)-6:]
}
