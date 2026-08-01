package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/withObsrvr/prism/internal/gateway"
	"github.com/withObsrvr/prism/internal/insight"
	pagesv2 "github.com/withObsrvr/prism/internal/templates/v2/pages"
	vmv2 "github.com/withObsrvr/prism/internal/templates/v2/viewmodel"
)

const insightV2Timeout = 3 * time.Second

func (h *Handlers) InsightDetailV2(w http.ResponseWriter, r *http.Request) {
	network := networkFromRequest(r)
	insightID := strings.TrimSpace(r.PathValue("id"))
	if !gateway.ValidHomeInsightID(insightID) {
		renderInsightDetail(w, r, invalidInsightDetailData(network, insightID), http.StatusBadRequest, h)
		return
	}
	if h.Gateway == nil || h.useExplicitMockData(r) {
		renderInsightDetail(w, r, unavailableInsightDetailData(network, insightID), http.StatusServiceUnavailable, h)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), insightV2Timeout)
	defer cancel()
	packet, err := h.Gateway.GetHomeInsight(ctx, network, insightID)
	if err != nil {
		data, status := insightDetailErrorData(network, insightID, err)
		if h.Logger != nil {
			h.Logger.Warn("insight detail unavailable", "network", network, "insight_id", insightID, "status", status, "error", err)
		}
		renderInsightDetail(w, r, data, status, h)
		return
	}

	interpreted, err := insight.InterpretDetail(*packet)
	if err != nil {
		if h.Logger != nil {
			h.Logger.Warn("insight detail failed Prism evidence checks", "network", network, "insight_id", insightID, "error", err)
		}
		renderInsightDetail(w, r, unverifiableInsightDetailData(network, insightID), http.StatusServiceUnavailable, h)
		return
	}
	renderInsightDetail(w, r, buildInsightDetailData(*packet, interpreted, network), http.StatusOK, h)
}

func buildInsightDetailData(packet gateway.HomeInsightDetailResponse, interpreted insight.DetailInterpretation, network string) vmv2.InsightDetailData {
	provenance := packet.EvidenceProvenance
	subjectLabel := interpreted.Subject.Label
	identityDetail := interpreted.Subject.IdentityDetail
	if packet.Subject.ID != "" && homeDisplayIsIdentifier(subjectLabel, packet.Subject.ID) {
		subjectLabel = shortHomeID(packet.Subject.ID)
		identityDetail = ""
	}
	data := vmv2.InsightDetailData{
		Network:               network,
		InsightID:             packet.InsightID,
		InsightLabel:          shortHomeID(packet.InsightID),
		State:                 packet.Status,
		StatusLabel:           insightDetailStatusLabel(packet.Status),
		Tone:                  interpreted.Severity,
		Title:                 interpreted.Title,
		Summary:               interpreted.Summary,
		Detail:                interpreted.Detail,
		SubjectLabel:          subjectLabel,
		SubjectID:             interpreted.Subject.ID,
		SubjectHref:           insightNetworkHref(interpreted.Subject.Href, network),
		IdentityDetail:        identityDetail,
		RuleLabel:             humanizeHomeCode(interpreted.RuleID) + " v" + interpreted.RuleVersion,
		RuleSummary:           interpreted.RuleSummary,
		MatchSummary:          interpreted.MatchSummary,
		ObservedWindowLabel:   insightWindowLabel(packet.Observed.WindowStart, packet.Observed.WindowEnd),
		BaselineWindowLabel:   insightWindowLabel(packet.Baseline.WindowStart, packet.Baseline.WindowEnd),
		ObservedLedgerRange:   insightLedgerRange(packet.Observed.FirstLedger, packet.Observed.LastLedger),
		SourceLedgerLabel:     gateway.FormatNumber(packet.Observed.SourceLedger),
		Caveats:               append([]string(nil), interpreted.Caveats...),
		ProvenanceSources:     append([]string(nil), provenance.Sources...),
		CompleteThroughLedger: gateway.FormatNumber(provenance.CompleteThroughLedger),
		UpdatedLabel:          formatHomeEvidenceTime(provenance.UpdatedAt),
		HomeHref:              "/v2/home?network=" + url.QueryEscape(network) + "#changes",
	}
	for _, metric := range interpreted.Metrics {
		label := metric.Label
		switch label {
		case "Observed":
			label = "Measured hour"
		case "Baseline":
			label = "Typical hour"
		case "Ratio":
			label = "Change"
		case "Evidence":
			label = "Evidence rows"
		}
		data.Metrics = append(data.Metrics, vmv2.InsightDetailMetric{Label: label, Value: metric.Value})
	}
	for _, value := range interpreted.Contributors {
		data.Contributors = append(data.Contributors, vmv2.InsightDetailContributor{
			Rank:        fmt.Sprintf("%02d", value.Rank),
			Dimension:   value.Dimension,
			Label:       value.Label,
			Key:         value.Key,
			ShowKey:     value.Key != value.Label && !homeDisplayIsIdentifier(value.Label, value.Key),
			CountLabel:  value.CountLabel,
			ShareLabel:  value.ShareLabel,
			LedgerLabel: value.LedgerLabel,
			Href:        insightNetworkHref(value.Href, network),
		})
	}
	for _, value := range interpreted.Samples {
		data.Samples = append(data.Samples, vmv2.InsightDetailSample{
			Rank:          fmt.Sprintf("%02d", value.Rank),
			KindLabel:     value.KindLabel,
			TxHash:        value.TxHash,
			TxLabel:       value.TxLabel,
			TxHref:        insightNetworkHref(value.TxHref, network),
			LedgerLabel:   value.LedgerLabel,
			LedgerHref:    insightNetworkHref(value.LedgerHref, network),
			Context:       value.Context,
			ContractID:    value.ContractID,
			ContractLabel: value.ContractLabel,
			ContractHref:  insightNetworkHref(value.ContractHref, network),
		})
	}
	for _, value := range interpreted.Evidence {
		data.Evidence = append(data.Evidence, vmv2.HomeInsightEvidenceLink{Label: value.Label, Href: insightNetworkHref(value.Href, network)})
	}
	return data
}

func insightDetailErrorData(network, insightID string, err error) (vmv2.InsightDetailData, int) {
	var apiErr *gateway.APIError
	if errors.As(err, &apiErr) {
		switch apiErr.StatusCode {
		case http.StatusBadRequest:
			return invalidInsightDetailData(network, insightID), http.StatusBadRequest
		case http.StatusNotFound:
			return missingInsightDetailData(network, insightID), http.StatusNotFound
		}
	}
	return unavailableInsightDetailData(network, insightID), http.StatusServiceUnavailable
}

func baseInsightErrorData(network, insightID, title, message string) vmv2.InsightDetailData {
	return vmv2.InsightDetailData{
		Network:      network,
		InsightID:    insightID,
		InsightLabel: shortHomeID(insightID),
		State:        "unavailable",
		ErrorTitle:   title,
		ErrorMessage: message,
		RetryHref:    "/v2/insight/" + url.PathEscape(insightID) + "?network=" + url.QueryEscape(network),
		HomeHref:     "/v2/home?network=" + url.QueryEscape(network) + "#changes",
	}
}

func invalidInsightDetailData(network, insightID string) vmv2.InsightDetailData {
	data := baseInsightErrorData(network, insightID, "This insight link is invalid", "Prism only opens versioned insight IDs produced by the evidence service. No substitute query was run.")
	data.RetryHref = ""
	return data
}

func missingInsightDetailData(network, insightID string) vmv2.InsightDetailData {
	data := baseInsightErrorData(network, insightID, "Insight evidence was not found", "The evidence service has no retained packet for this exact insight ID. Prism cannot distinguish an expired link from an ID that never existed.")
	data.RetryHref = ""
	return data
}

func unavailableInsightDetailData(network, insightID string) vmv2.InsightDetailData {
	return baseInsightErrorData(network, insightID, "Insight evidence is temporarily unavailable", "Prism could not read the retained evidence packet. The insight has not been reconstructed from homepage text or fallback data.")
}

func unverifiableInsightDetailData(network, insightID string) vmv2.InsightDetailData {
	return baseInsightErrorData(network, insightID, "Prism could not verify this insight", "The returned packet did not pass Prism's version, identity, or reconciliation checks, so no interpretation is shown.")
}

func insightDetailStatusLabel(status string) string {
	switch status {
	case "ready":
		return "Evidence ready"
	case "partial":
		return "Evidence limited"
	case "stale":
		return "Evidence delayed"
	default:
		return "Evidence unavailable"
	}
}

func insightWindowLabel(startRaw, endRaw string) string {
	start, startErr := time.Parse(time.RFC3339, startRaw)
	end, endErr := time.Parse(time.RFC3339, endRaw)
	if startErr != nil || endErr != nil {
		return "Exact window unavailable"
	}
	return start.UTC().Format("Jan 2, 2006 15:04") + " to " + end.UTC().Format("Jan 2, 2006 15:04 UTC")
}

func insightLedgerRange(first, last int64) string {
	if first <= 0 || last < first {
		return "Ledger range unavailable"
	}
	if first == last {
		return "Ledger " + gateway.FormatNumber(first)
	}
	return "Ledgers " + gateway.FormatNumber(first) + " to " + gateway.FormatNumber(last)
}

func insightNetworkHref(href, network string) string {
	if strings.TrimSpace(href) == "" {
		return ""
	}
	parsed, err := url.Parse(href)
	if err != nil || parsed.IsAbs() {
		return href
	}
	query := parsed.Query()
	query.Set("network", network)
	parsed.RawQuery = query.Encode()
	return parsed.String()
}

func renderInsightDetail(w http.ResponseWriter, r *http.Request, data vmv2.InsightDetailData, status int, h *Handlers) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	if err := pagesv2.InsightDetail(data).Render(r.Context(), w); err != nil && h.Logger != nil {
		h.Logger.Error("render insight detail", "error", err, "status", status)
	}
}
