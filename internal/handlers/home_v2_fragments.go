package handlers

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/withObsrvr/prism/internal/gateway"
	pagesv2 "github.com/withObsrvr/prism/internal/templates/v2/pages"
	vmv2 "github.com/withObsrvr/prism/internal/templates/v2/viewmodel"
)

const homeV2RecentLedgersTimeout = 2 * time.Second

func (h *Handlers) HomeV2Timeline(w http.ResponseWriter, r *http.Request) {
	network := networkFromRequest(r)
	pollURL := homeV2TimelineURL(network, h.useExplicitMockData(r))

	var data vmv2.HomeTimelineData
	if h.useExplicitMockData(r) {
		data = mockHomeV2TimelineData(network, time.Now())
	} else if h.Gateway == nil {
		data = unavailableHomeTimeline(network, pollURL, "Ledger activity is temporarily unavailable.")
	} else {
		ctx, cancel := context.WithTimeout(r.Context(), homeV2RecentLedgersTimeout)
		defer cancel()
		response, err := h.Gateway.GetSilverRecentLedgers(ctx, network, 60)
		if err != nil {
			if h.Logger != nil {
				h.Logger.Warn("home timeline unavailable", "network", network, "error", err)
			}
			data = unavailableHomeTimeline(network, pollURL, "Ledger activity is temporarily unavailable.")
		} else {
			data = buildHomeTimelineDataAt(response, network, pollURL, time.Now())
		}
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := pagesv2.HomeTimeline(data).Render(r.Context(), w); err != nil {
		if h.Logger != nil {
			h.Logger.Error("render home timeline", "error", err)
		}
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func unavailableHomeTimeline(network, pollURL, message string) vmv2.HomeTimelineData {
	return vmv2.HomeTimelineData{
		Network:       network,
		PollURL:       pollURL,
		HeaderState:   "Ledger unavailable",
		HeaderLedger:  "Unavailable",
		HeaderAge:     "Retrying",
		HeaderTxCount: "Unavailable",
		Status: vmv2.HomeSectionStatus{
			State:     vmv2.HomeSectionUnavailable,
			Message:   message,
			Retryable: true,
		},
	}
}

func buildHomeTimelineDataAt(response *gateway.RecentLedgersResponse, network, pollURL string, now time.Time) vmv2.HomeTimelineData {
	if response == nil {
		return unavailableHomeTimeline(network, pollURL, "Ledger activity is temporarily unavailable.")
	}
	if len(response.Ledgers) == 0 {
		if response.Provenance.Partial {
			return unavailableHomeTimeline(network, pollURL, "Ledger activity is incomplete, so Prism cannot prove that the window is empty.")
		}
		return vmv2.HomeTimelineData{
			Network:       network,
			PollURL:       pollURL,
			HeaderState:   "No ledger data",
			HeaderLedger:  "Unavailable",
			HeaderAge:     "Waiting for data",
			HeaderTxCount: "Unavailable",
			Status: vmv2.HomeSectionStatus{
				State:   vmv2.HomeSectionEmpty,
				Message: "No recent ledgers were returned for this network.",
			},
		}
	}

	ledgers := append([]gateway.RecentLedger(nil), response.Ledgers...)
	sort.Slice(ledgers, func(i, j int) bool {
		return ledgers[i].LedgerSequence < ledgers[j].LedgerSequence
	})

	latest := ledgers[len(ledgers)-1]
	maxOperations := 0
	maxFailures := 0
	for _, ledger := range ledgers {
		_, included, _, failed := homeTimelineLedgerCounts(ledger)
		maxOperations = max(maxOperations, included)
		maxFailures = max(maxFailures, failed)
	}

	legendCounts := map[string]int{}
	legendLabels := map[string]string{}
	columns := make([]vmv2.HomeSpectrogramColumn, 0, len(ledgers))
	totalIncluded := 0
	totalFailed := 0
	for _, ledger := range ledgers {
		transactions, included, successful, failed := homeTimelineLedgerCounts(ledger)
		segments := homeTimelineSegments(ledger, included)
		for _, segment := range segments {
			legendCounts[segment.Kind] += segment.Count
			legendLabels[segment.Kind] = segment.Label
		}
		totalIncluded += included
		totalFailed += failed

		closedAt, _ := parseGatewayTime(ledger.ClosedAt)
		introducer := homeTimelineIntroducer(ledger.Validator)
		accessible := fmt.Sprintf(
			"Ledger %d, %d transactions, %d included operations, %d successful operations, %d failed operations",
			ledger.LedgerSequence, transactions, included, successful, failed,
		)
		if !closedAt.IsZero() {
			accessible += ", closed " + closedAt.UTC().Format("January 2 at 15:04:05 UTC")
		}
		if introducer != "" {
			accessible += ", introduced by " + introducer
		}

		columns = append(columns, vmv2.HomeSpectrogramColumn{
			Sequence:             ledger.LedgerSequence,
			SequenceLabel:        gateway.FormatNumber(ledger.LedgerSequence),
			Href:                 fmt.Sprintf("/v2/ledger/%d", ledger.LedgerSequence),
			ClosedAt:             formatHomeTimelineTime(closedAt),
			AgeLabel:             relativeHomeTimelineAge(closedAt, now),
			TransactionCount:     transactions,
			IncludedOperations:   included,
			SuccessfulOperations: successful,
			FailedOperations:     failed,
			Introducer:           introducer,
			HeightStyle:          fmt.Sprintf("--ph-column-height:%d%%", normalizedHomeTimelineHeight(included, maxOperations)),
			FailureStyle:         fmt.Sprintf("--ph-failure-height:%d%%", normalizedHomeTimelineFailure(failed, maxFailures)),
			Segments:             segments,
			AccessibleLabel:      accessible,
			Latest:               ledger.LedgerSequence == latest.LedgerSequence,
		})
	}

	legendOrder := []string{"payments", "markets", "calls", "deployments", "soroban", "other"}
	legend := make([]vmv2.HomeSpectrogramLegendItem, 0, len(legendOrder))
	for _, kind := range legendOrder {
		count := legendCounts[kind]
		if count == 0 {
			continue
		}
		legend = append(legend, vmv2.HomeSpectrogramLegendItem{
			Kind:       kind,
			Label:      legendLabels[kind],
			Count:      count,
			Percentage: formatHomeTimelinePercent(count, totalIncluded),
		})
	}

	freshness, ageSeconds := homeTimelineFreshness(response, latest, now)
	state := vmv2.HomeSectionReady
	message := ""
	if response.Provenance.Partial {
		state = vmv2.HomeSectionPartial
		message = "Some ledger evidence is incomplete. Available values remain visible."
	} else if freshness == "stale" {
		state = vmv2.HomeSectionStale
		message = "The latest serving ledger is stale. Values remain visible with their source time."
	} else if freshness == "delayed" {
		message = "Serving data is delayed."
	}

	headerState := "Live"
	switch {
	case response.Provenance.Partial:
		headerState = "Partial data"
	case freshness == "delayed" || freshness == "stale":
		headerState = "Data delayed"
	case freshness == "unknown":
		headerState = "Ledger data"
	}

	asOfTime, _ := parseGatewayTime(response.SourceLedger.ClosedAt)
	if asOfTime.IsZero() {
		asOfTime, _ = parseGatewayTime(latest.ClosedAt)
	}
	asOfLedger := response.SourceLedger.Sequence
	if asOfLedger == 0 {
		asOfLedger = latest.LedgerSequence
	}

	return vmv2.HomeTimelineData{
		Status: vmv2.HomeSectionStatus{
			State:      state,
			Message:    message,
			AsOfLedger: asOfLedger,
			AsOfTime:   asOfTime,
			Warnings:   append([]string(nil), response.Provenance.Warnings...),
			Retryable:  state == vmv2.HomeSectionStale,
		},
		Network:         network,
		PollURL:         pollURL,
		Freshness:       freshness,
		HeaderState:     headerState,
		HeaderLedger:    gateway.FormatNumber(latest.LedgerSequence),
		HeaderAge:       formatHomeTimelineAgeSeconds(ageSeconds),
		HeaderTxCount:   gateway.FormatNumber(int64(homeTimelineTransactionCount(latest))),
		WindowLabel:     fmt.Sprintf("Last %d ledgers · %s", len(ledgers), homeTimelineWindow(ledgers)),
		DetailLabel:     "Column height shows included operations. Select a ledger to inspect its evidence.",
		StartSequence:   gateway.FormatNumber(ledgers[0].LedgerSequence),
		EndSequence:     gateway.FormatNumber(latest.LedgerSequence),
		ColumnGridStyle: fmt.Sprintf("--ph-column-count:%d", len(columns)),
		Columns:         columns,
		Legend:          legend,
		FailureCount:    totalFailed,
		FailurePercent:  formatHomeTimelinePercent(totalFailed, totalIncluded),
		AsOfLedgerLabel: gateway.FormatNumber(asOfLedger),
	}
}

func homeTimelineLedgerCounts(ledger gateway.RecentLedger) (transactions, included, successful, failed int) {
	transactions = ledger.Transactions.Total
	if transactions == 0 {
		transactions = ledger.TransactionCount
	}
	if transactions == 0 {
		transactions = ledger.SuccessfulTxCount + ledger.FailedTxCount
	}
	included = ledger.Operations.Included
	if included == 0 {
		included = ledger.TransactionSetOperationCount
	}
	if included == 0 {
		included = ledger.OperationCount
	}
	successful = ledger.Operations.Successful
	if successful == 0 && included > 0 {
		successful = ledger.SuccessfulOperationCount
	}
	failed = ledger.Operations.Failed
	if failed == 0 {
		failed = ledger.FailedOperationCount
	}
	if successful == 0 && included > 0 && failed <= included {
		successful = included - failed
	}
	return
}

func homeTimelineTransactionCount(ledger gateway.RecentLedger) int {
	transactions, _, _, _ := homeTimelineLedgerCounts(ledger)
	return transactions
}

func homeTimelineSegments(ledger gateway.RecentLedger, included int) []vmv2.HomeSpectrogramSegment {
	categories := ledger.Operations.Categories
	detail := ledger.Operations.SorobanDetail
	detailTotal := detail.ContractCalls + detail.ContractDeployments + detail.Other
	other := categories.AccountCreation + categories.Trustlines + categories.ClaimableBalances + categories.Sponsorship + categories.Other

	counts := []struct {
		kind  string
		label string
		count int
	}{
		{kind: "payments", label: "Payments", count: categories.Payments},
		{kind: "markets", label: "Offers and AMM", count: categories.OffersAndAMMs},
	}
	if categories.Soroban > 0 && detailTotal == categories.Soroban {
		counts = append(counts,
			struct {
				kind  string
				label string
				count int
			}{kind: "calls", label: "Contract calls", count: detail.ContractCalls},
			struct {
				kind  string
				label string
				count int
			}{kind: "deployments", label: "Deployments", count: detail.ContractDeployments},
		)
		other += detail.Other
	} else if categories.Soroban > 0 {
		counts = append(counts, struct {
			kind  string
			label string
			count int
		}{kind: "soroban", label: "Soroban", count: categories.Soroban})
	}
	counts = append(counts, struct {
		kind  string
		label string
		count int
	}{kind: "other", label: "Everything else", count: other})

	total := 0
	for _, item := range counts {
		total += item.count
	}
	if total < included {
		counts[len(counts)-1].count += included - total
	} else if total > included && included > 0 {
		counts = []struct {
			kind  string
			label string
			count int
		}{{kind: "other", label: "Unclassified activity", count: included}}
	}

	segments := make([]vmv2.HomeSpectrogramSegment, 0, len(counts))
	for _, item := range counts {
		if item.count <= 0 {
			continue
		}
		segments = append(segments, vmv2.HomeSpectrogramSegment{
			Kind:  item.kind,
			Label: item.label,
			Count: item.count,
			Style: fmt.Sprintf("--ph-segment-weight:%d", item.count),
		})
	}
	return segments
}

func homeTimelineFreshness(response *gateway.RecentLedgersResponse, latest gateway.RecentLedger, now time.Time) (string, int64) {
	freshness := strings.ToLower(strings.TrimSpace(response.SourceLedger.Freshness))
	ageSeconds := response.SourceLedger.AgeSeconds
	closedAt, _ := parseGatewayTime(response.SourceLedger.ClosedAt)
	if closedAt.IsZero() {
		closedAt, _ = parseGatewayTime(latest.ClosedAt)
	}
	if ageSeconds <= 0 && !closedAt.IsZero() {
		ageSeconds = max(int64(0), int64(now.Sub(closedAt).Seconds()))
	}
	if freshness == "" {
		switch {
		case closedAt.IsZero():
			freshness = "unknown"
		case ageSeconds <= 30:
			freshness = "fresh"
		case ageSeconds <= 120:
			freshness = "delayed"
		default:
			freshness = "stale"
		}
	}
	return freshness, ageSeconds
}

func homeTimelineIntroducer(validator gateway.LedgerValidator) string {
	if !validator.AttributionAvailable {
		return ""
	}
	for _, candidate := range []string{validator.DisplayName, validator.Name, validator.Alias, validator.HomeDomain} {
		if value := strings.TrimSpace(candidate); value != "" {
			return value
		}
	}
	return ""
}

func normalizedHomeTimelineHeight(value, maximum int) int {
	if value <= 0 || maximum <= 0 {
		return 0
	}
	return 10 + int(float64(value)/float64(maximum)*90)
}

func normalizedHomeTimelineFailure(value, maximum int) int {
	if value <= 0 || maximum <= 0 {
		return 0
	}
	return 20 + int(float64(value)/float64(maximum)*80)
}

func formatHomeTimelinePercent(value, total int) string {
	if total <= 0 {
		return "0%"
	}
	percentage := float64(value) / float64(total) * 100
	if percentage >= 10 {
		return fmt.Sprintf("%.0f%%", percentage)
	}
	return fmt.Sprintf("%.1f%%", percentage)
}

func homeTimelineWindow(ledgers []gateway.RecentLedger) string {
	if len(ledgers) < 2 {
		return "one ledger"
	}
	start, startOK := parseGatewayTime(ledgers[0].ClosedAt)
	end, endOK := parseGatewayTime(ledgers[len(ledgers)-1].ClosedAt)
	if !startOK || !endOK || end.Before(start) {
		return fmt.Sprintf("%d ledgers", len(ledgers))
	}
	duration := end.Sub(start)
	if duration < time.Minute {
		return fmt.Sprintf("%d seconds", int(duration.Round(time.Second).Seconds()))
	}
	return fmt.Sprintf("%d minutes", int(duration.Round(time.Minute).Minutes()))
}

func formatHomeTimelineTime(value time.Time) string {
	if value.IsZero() {
		return "Close time unavailable"
	}
	return value.UTC().Format("15:04:05 UTC")
}

func relativeHomeTimelineAge(value, now time.Time) string {
	if value.IsZero() {
		return "Age unavailable"
	}
	seconds := max(int64(0), int64(now.Sub(value).Seconds()))
	return formatHomeTimelineAgeSeconds(seconds)
}

func formatHomeTimelineAgeSeconds(seconds int64) string {
	switch {
	case seconds < 5:
		return "just now"
	case seconds < 60:
		return fmt.Sprintf("%ds ago", seconds)
	case seconds < 3600:
		return fmt.Sprintf("%dm ago", seconds/60)
	default:
		return fmt.Sprintf("%dh ago", seconds/3600)
	}
}

func parseGatewayTime(value string) (time.Time, bool) {
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02 15:04:05.999999-07:00", "2006-01-02 15:04:05-07:00"} {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed, true
		}
	}
	return time.Time{}, false
}
