package handlers

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/withObsrvr/prism/internal/gateway"
	"github.com/withObsrvr/prism/internal/insight"
	pagesv2 "github.com/withObsrvr/prism/internal/templates/v2/pages"
	vmv2 "github.com/withObsrvr/prism/internal/templates/v2/viewmodel"
)

// The Query API fails closed at 2.5 seconds. Leave a small transport margin so
// Prism does not abandon a response that the API can still return on time.
const homeV2EvidenceTimeout = 3 * time.Second

const homeLastGoodWarning = "Prism could not refresh this evidence, so the last available snapshot is shown."

func (h *Handlers) HomeV2Insights(w http.ResponseWriter, r *http.Request) {
	network := networkFromRequest(r)
	data := unavailableHomeInsights(network, homeV2FragmentURL("insights", network, false), "The seven-day comparison is temporarily unavailable.")
	if h.useExplicitMockData(r) {
		data = buildHomeInsightsData(mockHomeSummaryResponse(network), network, homeV2FragmentURL("insights", network, true))
		data.DemoData = true
	} else if summary, err := h.homeSummaryForFragment(r, network); err == nil {
		data = buildHomeInsightsData(summary, network, homeV2FragmentURL("insights", network, false))
	} else if h.Logger != nil {
		h.Logger.Warn("home insights unavailable", "network", network, "error", err)
	}
	renderHomeInsights(w, r, h, data)
}

func (h *Handlers) HomeV2TTL(w http.ResponseWriter, r *http.Request) {
	network := networkFromRequest(r)
	data := unavailableHomeTTL(network, homeV2FragmentURL("ttl", network, false), "Contract archival evidence is temporarily unavailable.")
	if h.useExplicitMockData(r) {
		data = buildHomeTTLData(mockHomeSummaryResponse(network), network, homeV2FragmentURL("ttl", network, true))
		data.DemoData = true
	} else if summary, err := h.homeSummaryForFragment(r, network); err == nil {
		data = buildHomeTTLData(summary, network, homeV2FragmentURL("ttl", network, false))
	} else if h.Logger != nil {
		h.Logger.Warn("home TTL attention unavailable", "network", network, "error", err)
	}
	renderHomeTTL(w, r, h, data)
}

func (h *Handlers) HomeV2Leaders(w http.ResponseWriter, r *http.Request) {
	network := networkFromRequest(r)
	data := unavailableHomeLeaders(network, homeV2FragmentURL("leaders", network, false), "Contract ranking evidence is temporarily unavailable.")
	if h.useExplicitMockData(r) {
		data = buildHomeLeadersData(mockHomeSummaryResponse(network), network, homeV2FragmentURL("leaders", network, true))
		data.DemoData = true
	} else if summary, err := h.homeSummaryForFragment(r, network); err == nil {
		data = buildHomeLeadersData(summary, network, homeV2FragmentURL("leaders", network, false))
	} else if h.Logger != nil {
		h.Logger.Warn("home leaders unavailable", "network", network, "error", err)
	}
	renderHomeLeaders(w, r, h, data)
}

func (h *Handlers) HomeV2Utilization(w http.ResponseWriter, r *http.Request) {
	network := networkFromRequest(r)
	data := unavailableHomeUtilization(network, homeV2FragmentURL("utilization", network, false), "Ledger utilization evidence is temporarily unavailable.")
	if h.useExplicitMockData(r) {
		data = buildHomeUtilizationData(mockHomeSummaryResponse(network), network, homeV2FragmentURL("utilization", network, true))
		data.DemoData = true
	} else if summary, err := h.homeSummaryForFragment(r, network); err == nil {
		data = buildHomeUtilizationData(summary, network, homeV2FragmentURL("utilization", network, false))
	} else if h.Logger != nil {
		h.Logger.Warn("home utilization unavailable", "network", network, "error", err)
	}
	renderHomeUtilization(w, r, h, data)
}

func (h *Handlers) homeSummaryForFragment(r *http.Request, network string) (*gateway.HomeSummaryResponse, error) {
	if h.Gateway == nil {
		return nil, fmt.Errorf("gateway unavailable")
	}
	ctx, cancel := context.WithTimeout(r.Context(), homeV2EvidenceTimeout)
	defer cancel()
	return h.Gateway.GetHomeSummary(ctx, network)
}

func buildHomeInsightsData(summary *gateway.HomeSummaryResponse, network, pollURL string) vmv2.HomeInsightsData {
	data := vmv2.HomeInsightsData{Network: network, PollURL: pollURL}
	if summary == nil {
		return unavailableHomeInsights(network, pollURL, "The seven-day comparison is temporarily unavailable.")
	}
	warnings := componentWarnings(summary, "insights", summary.Components.Insights)
	evaluationValid := false
	acceptCurrentInsights := true
	if summary.InsightEvaluation != nil {
		if err := insight.ValidateEvaluation(summary.InsightEvaluation); err != nil {
			warnings = append(warnings, "The detector evaluation packet failed Prism's evidence checks.")
		} else {
			evaluationValid = true
			data.WindowLabel = insightWindowLabel(summary.InsightEvaluation.WindowStart, summary.InsightEvaluation.WindowEnd)
			for _, check := range insight.EvaluationChecks(summary.InsightEvaluation) {
				data.Checks = append(data.Checks, vmv2.HomeInsightCheck{Label: check.Label, Value: check.Value, Detail: check.Detail, State: check.State})
			}
			if err := insight.ValidateEvaluationInsights(summary.InsightEvaluation, summary.Insights, summary.Components.Insights.Status); err != nil {
				warnings = append(warnings, "The detector evaluation and current insight rows do not reconcile.")
				evaluationValid = false
				acceptCurrentInsights = false
				data.Checks = nil
			}
		}
	}
	if summary.InsightDelivery != nil {
		if err := insight.ValidateInsightDelivery(summary.InsightDelivery, summary.InsightEvaluation); err != nil {
			warnings = append(warnings, "The insight delivery state failed Prism's evidence checks.")
			evaluationValid = false
			data.Checks = nil
		} else if summary.InsightDelivery.Mode == "last_good" {
			warnings = append(warnings, "This is the most recent retained comparison, not the current completed hour.")
		} else if summary.InsightDelivery.Mode == "unavailable" {
			warnings = append(warnings, "The API did not provide a current or retained comparison.")
		}
	}
	currentInsights := summary.Insights
	if !acceptCurrentInsights {
		currentInsights = nil
	}
	for _, item := range currentInsights {
		interpreted, err := insight.Interpret(item)
		if err != nil {
			warnings = append(warnings, "One insight packet failed Prism's evidence checks and was not narrated.")
			continue
		}
		card := homeInsightCard(interpreted)
		if (item.EvidenceVersion == insight.EvidenceVersionV1 || item.EvidenceVersion == insight.EvidenceVersionV2) && gateway.ValidHomeInsightID(item.InsightID) && !strings.Contains(pollURL, "mock=true") {
			card.DetailHref = "/v2/insight/" + url.PathEscape(item.InsightID) + "?network=" + url.QueryEscape(network)
		}
		data.Cards = append(data.Cards, card)
		if len(data.Cards) == 3 {
			break
		}
	}
	if len(data.Cards) == 0 && summary.RecentInsights != nil {
		for _, item := range *summary.RecentInsights {
			interpreted, err := insight.Interpret(item)
			if err != nil || interpreted.Generic || !gateway.ValidHomeInsightID(item.InsightID) {
				continue
			}
			data.RecentLabel = interpreted.Title
			data.RecentTimeLabel = formatHomeEvidenceTime(item.UpdatedAt)
			if !strings.Contains(pollURL, "mock=true") {
				data.RecentDetailHref = "/v2/insight/" + url.PathEscape(item.InsightID) + "?network=" + url.QueryEscape(network)
			}
			break
		}
	}
	data.Status = homeComponentStatus(summary.Components.Insights, len(data.Cards), warnings,
		"No significant changes in the last completed hour.",
		"The seven-day comparison is temporarily unavailable.")
	if evaluationValid && len(data.Cards) == 0 && strings.EqualFold(summary.Components.Insights.Status, "partial") {
		data.Status.State = vmv2.HomeSectionPartial
		data.Status.Message = "Some detector evidence is incomplete; available comparisons are shown without claiming a quiet hour."
		data.Status.Retryable = true
	}
	data.Status = applyHomeSummaryDeliveryStatus(data.Status, summary, len(data.Cards))
	if summary.InsightDelivery != nil {
		switch summary.InsightDelivery.Mode {
		case "last_good":
			data.Status.State = vmv2.HomeSectionStale
			data.Status.Retryable = true
		case "unavailable":
			if len(data.Cards) == 0 {
				data.Status.State = vmv2.HomeSectionUnavailable
				data.Status.Retryable = true
				data.Checks = nil
			}
		}
	}
	if !evaluationValid && len(data.Cards) == 0 && summary.InsightEvaluation != nil {
		data.Status.State = vmv2.HomeSectionUnavailable
		data.Status.Retryable = true
	}
	return data
}

func homeInsightCard(value insight.Interpretation) vmv2.HomeInsightCard {
	subjectLabel := value.Subject.Label
	identityDetail := value.Subject.IdentityDetail
	if value.Subject.ID != "" && homeDisplayIsIdentifier(subjectLabel, value.Subject.ID) {
		subjectLabel = shortHomeID(value.Subject.ID)
		identityDetail = ""
	}
	card := vmv2.HomeInsightCard{
		Title:           value.Title,
		Detail:          value.Detail,
		Tone:            value.Severity,
		State:           value.Status,
		WindowLabel:     value.WindowLabel,
		ComparisonLabel: value.ComparisonLabel,
		SubjectLabel:    subjectLabel,
		SubjectID:       value.Subject.ID,
		SubjectHref:     value.Subject.Href,
		IdentityDetail:  identityDetail,
		EvidenceCount:   gateway.FormatNumber(value.EvidenceCount),
		Caveats:         append([]string(nil), value.Caveats...),
		AsOfLedger:      gateway.FormatNumber(value.Provenance.CompleteThroughLedger),
		UpdatedLabel:    formatHomeEvidenceTime(value.Provenance.UpdatedAt),
		Generic:         value.Generic,
	}
	// Supported homepage cards already expose the observed value, baseline, and
	// ratio as comparable facts. Repeating those numbers in a paragraph makes
	// the scan path slower. Unknown versions retain their generic summary because
	// Prism cannot safely substitute the supported interpretation layout.
	if value.Generic {
		card.Summary = value.Summary
	}
	if value.RuleID != "" {
		card.RuleLabel = value.RuleID
		if value.RuleVersion != "" {
			card.RuleLabel += " v" + value.RuleVersion
		}
	}
	for _, metric := range value.Metrics {
		label := metric.Label
		switch label {
		case "Observed":
			label = "Last hour"
		case "Baseline":
			label = "Typical hour"
		case "Ratio":
			label = "Change"
		case "Evidence":
			continue
		}
		card.Metrics = append(card.Metrics, vmv2.HomeInsightMetric{Label: label, Value: metric.Value})
		if len(card.Metrics) == 3 {
			break
		}
	}
	for _, link := range value.Evidence {
		card.Evidence = append(card.Evidence, vmv2.HomeInsightEvidenceLink{Label: link.Label, Href: link.Href})
	}
	return card
}

func buildHomeTTLData(summary *gateway.HomeSummaryResponse, network, pollURL string) vmv2.HomeTTLData {
	data := vmv2.HomeTTLData{Network: network, PollURL: pollURL}
	if summary == nil {
		return unavailableHomeTTL(network, pollURL, "Contract archival evidence is temporarily unavailable.")
	}
	contracts := append([]gateway.HomeSummaryAttentionContract(nil), summary.ContractsNeedingAttention...)
	sort.SliceStable(contracts, func(i, j int) bool {
		return effectiveRemainingLedgers(contracts[i], summary) < effectiveRemainingLedgers(contracts[j], summary)
	})
	for _, contract := range contracts {
		if strings.TrimSpace(contract.ContractID) == "" {
			continue
		}
		remaining, known := remainingLedgerEvidence(contract, summary)
		runway := "Expiration unavailable"
		remainingLabel := ""
		if summary.Delivery.UsedLastGood {
			// A retained packet can safely preserve the absolute live-until ledger,
			// but its relative countdown may already be wrong.
			runway = "Availability may have changed"
		} else if known {
			runway = gateway.FormatNumber(remaining) + " ledgers left"
			if remaining == 1 {
				runway = "1 ledger left"
			}
			remainingLabel = gateway.FormatNumber(remaining)
		}
		contractLabel := shortHomeID(contract.ContractID)
		name := firstNonEmpty(strings.TrimSpace(contract.ProtocolName+" "+contract.ContractName), contract.ContractName, contract.ProtocolName, contractLabel)
		showContractID := !homeDisplayIsIdentifier(name, contract.ContractID)
		if !showContractID {
			name = contractLabel
		}
		detailParts := make([]string, 0, 3)
		if contract.RemainingHuman != "" && !summary.Delivery.UsedLastGood {
			detailParts = append(detailParts, contract.RemainingHuman+" at the serving estimate")
		}
		if contract.ExpiringEntryCount > 0 || contract.TrackedEntryCount > 0 {
			detailParts = append(detailParts, fmt.Sprintf("%s of %s tracked entries near the attention boundary", gateway.FormatNumber(contract.ExpiringEntryCount), gateway.FormatNumber(contract.TrackedEntryCount)))
		}
		if len(contract.DurabilityClasses) > 0 {
			detailParts = append(detailParts, strings.Join(contract.DurabilityClasses, " and ")+" storage")
		}
		tone := homeTTLTone(contract, remaining, known)
		if summary.Delivery.UsedLastGood {
			tone = "neutral"
		}
		data.Cards = append(data.Cards, vmv2.HomeTTLCard{
			Name:             name,
			ContractID:       contract.ContractID,
			ContractLabel:    contractLabel,
			ShowContractID:   showContractID,
			Href:             "/v2/contract/" + url.PathEscape(contract.ContractID),
			Tone:             tone,
			RunwayLabel:      runway,
			RemainingLedgers: remainingLabel,
			LiveUntilLedger:  formatOptionalLedger(contract.NearestLiveUntilLedger),
			Detail:           strings.Join(detailParts, ". "),
		})
		if len(data.Cards) == 4 {
			break
		}
	}
	data.Status = homeComponentStatus(summary.Components.TTLAttention, len(data.Cards), componentWarnings(summary, "ttl_attention", summary.Components.TTLAttention),
		"No contracts are inside the current archival-attention window.",
		"Contract archival evidence is temporarily unavailable.")
	data.Status = applyHomeSummaryDeliveryStatus(data.Status, summary, len(data.Cards))
	return data
}

func buildHomeLeadersData(summary *gateway.HomeSummaryResponse, network, pollURL string) vmv2.HomeLeadersData {
	data := vmv2.HomeLeadersData{Network: network, PollURL: pollURL}
	if summary == nil {
		return unavailableHomeLeaders(network, pollURL, "Contract ranking evidence is temporarily unavailable.")
	}
	leaders := append([]gateway.HomeSummaryLeader(nil), summary.Leaders...)
	sort.SliceStable(leaders, func(i, j int) bool { return homeLeaderCalls(leaders[i]) > homeLeaderCalls(leaders[j]) })
	for _, leader := range leaders {
		if strings.TrimSpace(leader.ContractID) == "" {
			continue
		}
		contractLabel := shortHomeID(leader.ContractID)
		name := homeLeaderName(leader)
		showContractID := !homeDisplayIsIdentifier(name, leader.ContractID)
		if !showContractID {
			name = contractLabel
		}
		failureLabel, failureTone := homeLeaderFailure(leader)
		data.Cards = append(data.Cards, vmv2.HomeLeaderCard{
			Name:           name,
			ContractID:     leader.ContractID,
			ContractLabel:  contractLabel,
			ShowContractID: showContractID,
			Href:           "/v2/contract/" + url.PathEscape(leader.ContractID),
			IdentityDetail: homeLeaderIdentity(leader),
			CallCount:      gateway.FormatNumber(homeLeaderCalls(leader)),
			CallerCount:    gateway.FormatNumber(homeLeaderCallers(leader)),
			CallerUnit:     homeCallerUnit(homeLeaderCallers(leader)),
			OutcomeLabel:   homeLeaderOutcome(leader),
			FailureLabel:   failureLabel,
			FailureTone:    failureTone,
			TopFunction:    leader.TopFunction,
			WindowLabel:    firstNonEmpty(leader.Window, "Window not supplied"),
			UpdatedLabel:   formatHomeEvidenceTime(leader.UpdatedAt),
		})
		if len(data.Cards) == 5 {
			break
		}
	}
	data.Status = homeComponentStatus(summary.Components.Leaders, len(data.Cards), componentWarnings(summary, "leaders", summary.Components.Leaders),
		"No contract calls were found in the completed 24-hour ranking window.",
		"Contract ranking evidence is temporarily unavailable.")
	data.Status = applyHomeSummaryDeliveryStatus(data.Status, summary, len(data.Cards))
	return data
}

func buildHomeUtilizationData(summary *gateway.HomeSummaryResponse, network, pollURL string) vmv2.HomeUtilizationData {
	data := vmv2.HomeUtilizationData{Network: network, PollURL: pollURL}
	if summary == nil {
		return unavailableHomeUtilization(network, pollURL, "Ledger utilization evidence is temporarily unavailable.")
	}
	if metric := homeInstructionMetric(summary.Utilization); metric != nil {
		data.Metrics = append(data.Metrics, *metric)
	}
	if metric := homeReadWriteMetric(summary.Utilization); metric != nil {
		data.Metrics = append(data.Metrics, *metric)
	}
	if metric := homeTxSizeMetric(summary.Utilization); metric != nil {
		data.Metrics = append(data.Metrics, *metric)
	}
	data.Status = homeComponentStatus(summary.Components.Utilization, len(data.Metrics), componentWarnings(summary, "utilization", summary.Components.Utilization),
		"No utilization measurements were emitted for the current serving ledger.",
		"Ledger utilization evidence is temporarily unavailable.")
	data.Status = applyHomeSummaryDeliveryStatus(data.Status, summary, len(data.Metrics))
	return data
}

func homeInstructionMetric(utilization gateway.HomeSummaryUtilization) *vmv2.HomeUtilizationMetric {
	if metric := utilization.Instructions; metric != nil {
		return homeBoundedMetric("Contract computation", metric.Status, metric.Used, metric.Limit, metric.Ratio, metric.Pct, metric.SourceLedger, metric.LimitSource, false)
	}
	return homeBoundedMetric("Contract computation", utilization.InstructionStatus, utilization.InstructionUsed, utilization.InstructionLimit, nil, utilization.InstructionPct, utilization.SourceLedger, utilization.InstructionLimitSource, false)
}

func homeReadWriteMetric(utilization gateway.HomeSummaryUtilization) *vmv2.HomeUtilizationMetric {
	if metric := utilization.ReadWriteBytes; metric != nil {
		return homeBoundedMetric("Contract state access", metric.Status, metric.Used, metric.Limit, metric.Ratio, metric.Pct, metric.SourceLedger, metric.LimitSource, true)
	}
	return homeBoundedMetric("Contract state access", utilization.ReadWriteStatus, utilization.ReadWriteUsedBytes, utilization.ReadWriteLimitBytes, nil, utilization.ReadWritePct, utilization.SourceLedger, utilization.ReadWriteLimitSource, true)
}

func homeTxSizeMetric(utilization gateway.HomeSummaryUtilization) *vmv2.HomeUtilizationMetric {
	metric := utilization.TransactionEnvelopeSize
	status := utilization.TxSizeStatus
	avg := utilization.AvgTxSizeBytes
	limit := utilization.TxSizeLimitBytes
	ratio := utilization.TxSizePct
	sourceLedger := utilization.SourceLedger
	limitSource := utilization.TxSizeLimitSource
	if metric != nil {
		status = metric.Status
		avg = metric.AvgTxSizeBytes
		limit = metric.TxSizeLimitBytes
		ratio = metric.AvgRatio
		sourceLedger = metric.SourceLedger
		limitSource = metric.LimitSource
	}
	if status == "empty" || status == "unavailable" || avg == nil || limit == nil || *limit <= 0 {
		return nil
	}
	resolvedRatio := float64(0)
	if ratio != nil {
		resolvedRatio = *ratio
		if resolvedRatio > 1 {
			resolvedRatio /= 100
		}
	} else {
		resolvedRatio = *avg / float64(*limit)
	}
	return &vmv2.HomeUtilizationMetric{
		Label:        "Transaction envelope size",
		State:        normalizeMetricState(status),
		PercentLabel: formatHomeRatio(resolvedRatio),
		BarStyle:     homeMeterStyle(resolvedRatio),
		UsedLabel:    "Average " + formatBytes(int64(mathRound(*avg))),
		LimitLabel:   "Limit " + formatBytes(*limit),
		Detail:       "Average transaction envelope size compared with the ledger-specific limit.",
		SourceLedger: formatOptionalLedger(sourceLedger),
		LimitSource:  humanizeHomeCode(limitSource),
	}
}

func homeBoundedMetric(label, status string, used, limit *int64, ratio, pct *float64, sourceLedger int64, limitSource string, bytes bool) *vmv2.HomeUtilizationMetric {
	if status == "empty" || status == "unavailable" || used == nil || limit == nil || *limit <= 0 {
		return nil
	}
	resolvedRatio := float64(*used) / float64(*limit)
	if ratio != nil {
		resolvedRatio = *ratio
	} else if pct != nil {
		resolvedRatio = *pct
		if resolvedRatio > 1 {
			resolvedRatio /= 100
		}
	}
	usedLabel := gateway.FormatNumber(*used) + " used"
	limitLabel := gateway.FormatNumber(*limit) + " limit"
	if bytes {
		usedLabel = formatBytes(*used) + " used"
		limitLabel = formatBytes(*limit) + " limit"
	}
	return &vmv2.HomeUtilizationMetric{
		Label:        label,
		State:        normalizeMetricState(status),
		PercentLabel: formatHomeRatio(resolvedRatio),
		BarStyle:     homeMeterStyle(resolvedRatio),
		UsedLabel:    usedLabel,
		LimitLabel:   limitLabel,
		Detail:       label + " used in the source ledger compared with its ledger-specific limit.",
		SourceLedger: formatOptionalLedger(sourceLedger),
		LimitSource:  humanizeHomeCode(limitSource),
	}
}

func homeComponentStatus(component gateway.HomeSummaryComponent, itemCount int, warnings []string, emptyMessage, unavailableMessage string) vmv2.HomeSectionStatus {
	status := vmv2.HomeSectionStatus{Warnings: uniqueStrings(warnings)}
	if component.AsOfLedger != nil {
		status.AsOfLedger = *component.AsOfLedger
	} else if component.CompleteThroughLedger != nil {
		status.AsOfLedger = *component.CompleteThroughLedger
	}
	if status.AsOfLedger > 0 {
		status.AsOfLedgerLabel = gateway.FormatNumber(status.AsOfLedger)
	}
	switch strings.ToLower(strings.TrimSpace(component.Status)) {
	case "ready":
		if itemCount == 0 {
			status.State = vmv2.HomeSectionUnavailable
			status.Message = "The API marked this component ready but did not provide usable evidence."
			status.Retryable = true
		} else {
			status.State = vmv2.HomeSectionReady
		}
	case "empty":
		if itemCount != 0 {
			status.State = vmv2.HomeSectionUnavailable
			status.Message = "The component state conflicts with its returned evidence."
			status.Retryable = true
		} else {
			status.State = vmv2.HomeSectionEmpty
			status.Message = emptyMessage
		}
	case "partial":
		if itemCount == 0 {
			status.State = vmv2.HomeSectionUnavailable
			status.Message = "The API reported incomplete evidence but did not provide usable facts."
			status.Retryable = true
		} else {
			status.State = vmv2.HomeSectionPartial
			status.Message = "Available evidence is shown below, but this component is incomplete."
		}
	case "stale":
		if itemCount == 0 {
			status.State = vmv2.HomeSectionUnavailable
			status.Message = "The delayed component did not provide retained evidence."
			status.Retryable = true
		} else {
			status.State = vmv2.HomeSectionStale
			status.Message = "This retained evidence is delayed; its source ledger remains visible."
		}
	case "unavailable":
		status.State = vmv2.HomeSectionUnavailable
		status.Message = unavailableMessage
		status.Retryable = true
	default:
		status.State = vmv2.HomeSectionUnavailable
		status.Message = "The API returned an unsupported component state, so Prism did not present it as evidence."
		status.Retryable = true
	}
	return status
}

func componentWarnings(summary *gateway.HomeSummaryResponse, componentName string, component gateway.HomeSummaryComponent) []string {
	warnings := make([]string, 0, 4)
	if component.WarningCode != "" {
		warnings = append(warnings, humanizeHomeWarning(component.WarningCode))
	}
	for _, detail := range summary.Provenance.WarningDetails {
		if detail.Component == componentName && detail.Code != component.WarningCode {
			warnings = append(warnings, humanizeHomeWarning(detail.Code))
		}
	}
	if summary.Delivery.UsedLastGood {
		warnings = append(warnings, homeLastGoodWarning)
	}
	return warnings
}

func applyHomeSummaryDeliveryStatus(status vmv2.HomeSectionStatus, summary *gateway.HomeSummaryResponse, itemCount int) vmv2.HomeSectionStatus {
	if summary == nil || !summary.Delivery.UsedLastGood {
		return status
	}
	status.Retryable = true
	if itemCount == 0 {
		// An old empty result cannot establish that the section is still empty.
		status.State = vmv2.HomeSectionUnavailable
		status.Message = "Prism could not refresh this evidence. The last snapshot did not contain evidence that can be presented as current."
		return status
	}
	if status.State == vmv2.HomeSectionReady || status.State == vmv2.HomeSectionPartial || status.State == vmv2.HomeSectionStale {
		status.State = vmv2.HomeSectionStale
		status.Message = "Showing the last available snapshot while Prism reconnects."
	}
	return status
}

func unavailableHomeInsights(network, pollURL, message string) vmv2.HomeInsightsData {
	return vmv2.HomeInsightsData{Network: network, PollURL: pollURL, Status: unavailableHomeEvidenceStatus(message)}
}

func unavailableHomeTTL(network, pollURL, message string) vmv2.HomeTTLData {
	return vmv2.HomeTTLData{Network: network, PollURL: pollURL, Status: unavailableHomeEvidenceStatus(message)}
}

func unavailableHomeLeaders(network, pollURL, message string) vmv2.HomeLeadersData {
	return vmv2.HomeLeadersData{Network: network, PollURL: pollURL, Status: unavailableHomeEvidenceStatus(message)}
}

func unavailableHomeUtilization(network, pollURL, message string) vmv2.HomeUtilizationData {
	return vmv2.HomeUtilizationData{Network: network, PollURL: pollURL, Status: unavailableHomeEvidenceStatus(message)}
}

func unavailableHomeEvidenceStatus(message string) vmv2.HomeSectionStatus {
	return vmv2.HomeSectionStatus{State: vmv2.HomeSectionUnavailable, Message: message, Retryable: true}
}

func renderHomeInsights(w http.ResponseWriter, r *http.Request, h *Handlers, data vmv2.HomeInsightsData) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := pagesv2.HomeInsights(data).Render(r.Context(), w); err != nil {
		if h.Logger != nil {
			h.Logger.Error("render home insights", "error", err)
		}
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func renderHomeTTL(w http.ResponseWriter, r *http.Request, h *Handlers, data vmv2.HomeTTLData) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := pagesv2.HomeTTL(data).Render(r.Context(), w); err != nil {
		if h.Logger != nil {
			h.Logger.Error("render home TTL", "error", err)
		}
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func renderHomeLeaders(w http.ResponseWriter, r *http.Request, h *Handlers, data vmv2.HomeLeadersData) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := pagesv2.HomeLeaders(data).Render(r.Context(), w); err != nil {
		if h.Logger != nil {
			h.Logger.Error("render home leaders", "error", err)
		}
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func renderHomeUtilization(w http.ResponseWriter, r *http.Request, h *Handlers, data vmv2.HomeUtilizationData) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := pagesv2.HomeUtilization(data).Render(r.Context(), w); err != nil {
		if h.Logger != nil {
			h.Logger.Error("render home utilization", "error", err)
		}
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func remainingLedgerEvidence(contract gateway.HomeSummaryAttentionContract, summary *gateway.HomeSummaryResponse) (int64, bool) {
	if contract.RemainingLedgers > 0 {
		return contract.RemainingLedgers, true
	}
	source := summary.Freshness.SourceLedger
	if source <= 0 && summary.Components.TTLAttention.AsOfLedger != nil {
		source = *summary.Components.TTLAttention.AsOfLedger
	}
	if contract.NearestLiveUntilLedger > 0 && source > 0 && contract.NearestLiveUntilLedger >= source {
		return contract.NearestLiveUntilLedger - source, true
	}
	return 0, false
}

func effectiveRemainingLedgers(contract gateway.HomeSummaryAttentionContract, summary *gateway.HomeSummaryResponse) int64 {
	if remaining, ok := remainingLedgerEvidence(contract, summary); ok {
		return remaining
	}
	return int64(^uint64(0) >> 1)
}

func homeTTLTone(contract gateway.HomeSummaryAttentionContract, remaining int64, known bool) string {
	severity := strings.ToLower(contract.Severity)
	if severity == "critical" || severity == "urgent" || (known && remaining <= 17280) {
		return "critical"
	}
	if severity == "warning" || severity == "high" {
		return "warning"
	}
	return "neutral"
}

func homeLeaderCalls(leader gateway.HomeSummaryLeader) int64 {
	if leader.CallCount24h > 0 {
		return leader.CallCount24h
	}
	return int64(leader.TotalCalls)
}

func homeLeaderCallers(leader gateway.HomeSummaryLeader) int64 {
	if leader.UniqueCallers24h > 0 {
		return leader.UniqueCallers24h
	}
	return int64(leader.UniqueCallers)
}

func homeCallerUnit(count int64) string {
	if count == 1 {
		return "unique caller"
	}
	return "unique callers"
}

func homeLeaderIdentity(leader gateway.HomeSummaryLeader) string {
	status := humanizeHomeCode(leader.Identity.VerificationStatus)
	source := humanizeHomeCode(leader.Identity.Source)
	if status == "" && source == "" {
		return "Raw contract identity"
	}
	if source == "" {
		return status + " identity"
	}
	if status == "" {
		return "Identity from " + source
	}
	return status + " identity from " + source
}

func homeLeaderName(leader gateway.HomeSummaryLeader) string {
	for _, candidate := range []string{leader.DisplayName, strings.TrimSpace(leader.ProtocolName + " " + leader.ContractName), leader.ContractName, leader.ProtocolName} {
		candidate = strings.TrimSpace(candidate)
		if candidate != "" && candidate != leader.ContractID {
			return candidate
		}
	}
	return shortHomeID(leader.ContractID)
}

func homeLeaderOutcome(leader gateway.HomeSummaryLeader) string {
	if leader.SuccessCount != nil && leader.FailureCount != nil {
		return fmt.Sprintf("%s successful · %s failed", gateway.FormatNumber(*leader.SuccessCount), gateway.FormatNumber(*leader.FailureCount))
	}
	if leader.SuccessRate != nil && leader.FailureRate != nil {
		return fmt.Sprintf("%.0f%% successful · %.0f%% failed", normalizeRate(*leader.SuccessRate)*100, normalizeRate(*leader.FailureRate)*100)
	}
	return "Outcome breakdown unavailable"
}

func homeLeaderFailure(leader gateway.HomeSummaryLeader) (string, string) {
	rate, ok := homeLeaderFailureRate(leader)
	if !ok {
		return "Unavailable", "neutral"
	}
	if rate < 0 {
		rate = 0
	} else if rate > 1 {
		rate = 1
	}
	percent := rate * 100
	label := fmt.Sprintf("%.0f%% failed", percent)
	if percent > 0 && percent < 0.1 {
		label = "<0.1% failed"
	} else if percent > 0 && percent < 10 {
		label = fmt.Sprintf("%.1f%% failed", percent)
	}
	tone := "healthy"
	if rate >= 0.2 {
		tone = "critical"
	} else if rate >= 0.05 {
		tone = "warning"
	}
	return label, tone
}

func homeLeaderFailureRate(leader gateway.HomeSummaryLeader) (float64, bool) {
	if leader.FailureRate != nil {
		return normalizeRate(*leader.FailureRate), true
	}
	if leader.SuccessRate != nil {
		return 1 - normalizeRate(*leader.SuccessRate), true
	}
	if leader.SuccessCount != nil && leader.FailureCount != nil {
		total := *leader.SuccessCount + *leader.FailureCount
		if total > 0 {
			return float64(*leader.FailureCount) / float64(total), true
		}
	}
	return 0, false
}

func normalizeRate(value float64) float64 {
	if value > 1 {
		return value / 100
	}
	return value
}

func normalizeMetricState(status string) string {
	status = strings.ToLower(strings.TrimSpace(status))
	if status == "partial" {
		return "partial"
	}
	return "ready"
}

func formatHomeRatio(ratio float64) string {
	return fmt.Sprintf("%.1f%%", ratio*100)
}

func homeMeterStyle(ratio float64) string {
	ratio = max(0, min(1, ratio))
	return fmt.Sprintf("--ph-meter-width:%.2f%%", ratio*100)
}

func mathRound(value float64) float64 {
	if value < 0 {
		return value - 0.5
	}
	return value + 0.5
}

func formatOptionalLedger(value int64) string {
	if value <= 0 {
		return ""
	}
	return gateway.FormatNumber(value)
}

func formatHomeEvidenceTime(value string) string {
	parsed, ok := parseGatewayTime(value)
	if !ok {
		return ""
	}
	return parsed.UTC().Format("Jan 2, 15:04 UTC")
}

func humanizeHomeWarning(code string) string {
	switch code {
	case "home_insight_evidence_partial":
		return "Some insight evidence is incomplete."
	case "home_insight_evidence_stale":
		return "Insight evidence is delayed."
	case "home_insight_evidence_unavailable":
		return "Insight evidence is unavailable."
	case "ttl_projection_unavailable":
		return "Contract archival evidence is unavailable."
	case "ttl_projection_lagging":
		return "Contract archival evidence is lagging behind the latest serving ledger."
	case "leaders_projection_unavailable":
		return "Contract ranking evidence is unavailable."
	case "leaders_projection_lagging":
		return "Contract ranking evidence is lagging behind the latest serving ledger."
	case "utilization_projection_unavailable":
		return "Utilization evidence is unavailable."
	case "source_ledger_limit_unavailable":
		return "One or more ledger-specific utilization limits are unavailable."
	default:
		return humanizeHomeCode(code)
	}
}

func humanizeHomeCode(value string) string {
	value = strings.TrimSpace(strings.ReplaceAll(value, "_", " "))
	if value == "" {
		return ""
	}
	return strings.ToUpper(value[:1]) + value[1:]
}

func shortHomeID(value string) string {
	value = strings.TrimSpace(value)
	if len(value) <= 16 {
		return value
	}
	return value[:8] + "…" + value[len(value)-6:]
}

func homeDisplayIsIdentifier(name, identifier string) bool {
	name = strings.ToUpper(strings.TrimSpace(name))
	identifier = strings.ToUpper(strings.TrimSpace(identifier))
	if name == "" || identifier == "" {
		return false
	}
	if name == identifier || name == strings.ToUpper(shortHomeID(identifier)) {
		return true
	}
	if len(identifier) < 8 || len(name) > 24 {
		return false
	}
	return strings.HasPrefix(name, identifier[:4]) && strings.HasSuffix(name, identifier[len(identifier)-4:])
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}
