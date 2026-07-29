package handlers

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/withObsrvr/prism/internal/gateway"
	vmv2 "github.com/withObsrvr/prism/internal/templates/v2/viewmodel"
)

func TestHomeEvidenceBuildersRenderOneCoherentSummary(t *testing.T) {
	summary := mockHomeSummaryResponse("testnet")

	insights := buildHomeInsightsData(summary, "testnet", "/v2/home/insights?network=testnet")
	if insights.Status.State != vmv2.HomeSectionReady || len(insights.Cards) != 1 {
		t.Fatalf("insights = %+v", insights)
	}
	if !strings.Contains(insights.Cards[0].Summary, "42 failures") || !strings.Contains(insights.Cards[0].Evidence[0].Href, "status=failed") {
		t.Fatalf("insight evidence was not interpreted: %+v", insights.Cards[0])
	}

	ttl := buildHomeTTLData(summary, "testnet", "/v2/home/ttl?network=testnet")
	if ttl.Status.State != vmv2.HomeSectionReady || len(ttl.Cards) != 2 || ttl.Cards[0].LiveUntilLedger == "" || !strings.Contains(ttl.Cards[0].RunwayLabel, "ledgers remaining") {
		t.Fatalf("TTL = %+v", ttl)
	}

	leaders := buildHomeLeadersData(summary, "testnet", "/v2/home/leaders?network=testnet")
	if leaders.Status.State != vmv2.HomeSectionReady || len(leaders.Cards) != 1 || leaders.Cards[0].CallCount != "91" || !strings.Contains(leaders.Cards[0].OutcomeLabel, "87 successful") {
		t.Fatalf("leaders = %+v", leaders)
	}

	utilization := buildHomeUtilizationData(summary, "testnet", "/v2/home/utilization?network=testnet")
	if utilization.Status.State != vmv2.HomeSectionReady || len(utilization.Metrics) != 3 || utilization.Metrics[0].PercentLabel != "64.0%" {
		t.Fatalf("utilization = %+v", utilization)
	}
}

func TestHomeEvidenceDistinguishesEmptyPartialAndInvalidStates(t *testing.T) {
	summary := mockHomeSummaryResponse("testnet")
	summary.Components.Insights.Status = "empty"
	summary.Insights = nil
	empty := buildHomeInsightsData(summary, "testnet", "/v2/home/insights?network=testnet")
	if empty.Status.State != vmv2.HomeSectionEmpty || !strings.Contains(empty.Status.Message, "No material changes") {
		t.Fatalf("authoritative empty = %+v", empty)
	}

	summary = mockHomeSummaryResponse("testnet")
	summary.Components.Insights.Status = "partial"
	summary.Insights[0].Facts.Failure.SuccessCount--
	partialWithoutFacts := buildHomeInsightsData(summary, "testnet", "/v2/home/insights?network=testnet")
	if partialWithoutFacts.Status.State != vmv2.HomeSectionUnavailable || len(partialWithoutFacts.Cards) != 0 {
		t.Fatalf("invalid partial packet masqueraded as evidence: %+v", partialWithoutFacts)
	}

	summary = mockHomeSummaryResponse("testnet")
	summary.Components.Leaders.Status = "future_state"
	invalid := buildHomeLeadersData(summary, "testnet", "/v2/home/leaders?network=testnet")
	if invalid.Status.State != vmv2.HomeSectionUnavailable || !strings.Contains(invalid.Status.Message, "unsupported component state") {
		t.Fatalf("unknown state was accepted: %+v", invalid)
	}
}

func TestHomeInsightsPreservesPartialAndStaleEvidence(t *testing.T) {
	for _, state := range []string{"partial", "stale"} {
		t.Run(state, func(t *testing.T) {
			summary := mockHomeSummaryResponse("testnet")
			summary.Components.Insights.Status = state
			summary.Insights[0].Status = state
			code := "function_distribution_unavailable"
			if state == "stale" {
				code = "projection_stale"
			}
			caveats := []gateway.HomeInsightCaveat{{Code: code, Field: "primary_contributor", Retryable: true}}
			summary.Insights[0].Caveats = &caveats
			data := buildHomeInsightsData(summary, "testnet", "/v2/home/insights?network=testnet")
			if string(data.Status.State) != state || len(data.Cards) != 1 || data.Status.Message == "" {
				t.Fatalf("%s evidence was not preserved: %+v", state, data)
			}
		})
	}

	summary := mockHomeSummaryResponse("testnet")
	summary.Components.Insights.Status = "unavailable"
	summary.Insights = nil
	unavailable := buildHomeInsightsData(summary, "testnet", "/v2/home/insights?network=testnet")
	if unavailable.Status.State != vmv2.HomeSectionUnavailable || len(unavailable.Cards) != 0 {
		t.Fatalf("unavailable evidence was not isolated: %+v", unavailable)
	}
}

func TestHomeInsightsUnknownVersionRendersGenericFacts(t *testing.T) {
	summary := mockHomeSummaryResponse("testnet")
	item := &summary.Insights[0]
	item.EvidenceVersion = "home_insight_evidence_v2"
	item.ObservedValue = 42
	item.BaselineValue = 7
	item.WindowStart = item.Observed.WindowStart
	item.WindowEnd = item.Observed.WindowEnd
	data := buildHomeInsightsData(summary, "testnet", "/v2/home/insights?network=testnet")
	if data.Status.State != vmv2.HomeSectionReady || len(data.Cards) != 1 || !data.Cards[0].Generic || len(data.Cards[0].Evidence) != 0 {
		t.Fatalf("unknown version claimed a supported narrative: %+v", data)
	}
}

func TestHomeUtilizationMissingMetricDoesNotSuppressOtherEvidence(t *testing.T) {
	summary := mockHomeSummaryResponse("testnet")
	summary.Components.Utilization.Status = "partial"
	summary.Utilization.TransactionEnvelopeSize = &gateway.HomeSummaryTxSizeMetric{Status: "unavailable"}
	data := buildHomeUtilizationData(summary, "testnet", "/v2/home/utilization?network=testnet")
	if data.Status.State != vmv2.HomeSectionPartial || len(data.Metrics) != 2 {
		t.Fatalf("available utilization metrics were suppressed: %+v", data)
	}
}

func TestHomeEvidenceFragmentsShareCachedSummaryAndNeverFallBackToMock(t *testing.T) {
	summary := mockHomeSummaryResponse("testnet")
	summary.Leaders[0].Identity.Source = "semantic_contract_registry"
	summary.Insights[0].Subject.Identity.Source = "semantic_contract_registry"
	summary.Insights[0].EvidenceProvenance.Sources = []string{"serving.sv_home_insight_history"}
	summary.Provenance.DataSource = "serving"
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(summary); err != nil {
			t.Fatalf("encode summary: %v", err)
		}
	}))
	defer server.Close()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	client := gateway.New(gateway.Config{BaseURL: server.URL, APIKey: "test", Timeout: time.Second}, logger, context.Background())
	h := &Handlers{Logger: logger, Gateway: client, DataSource: "auto"}

	requests := []struct {
		path    string
		handler http.HandlerFunc
		want    string
	}{
		{path: "/v2/home/insights?network=testnet", handler: h.HomeV2Insights, want: "Contract failures rose above their usual hour"},
		{path: "/v2/home/ttl?network=testnet", handler: h.HomeV2TTL, want: "Absolute ledger runway"},
		{path: "/v2/home/leaders?network=testnet", handler: h.HomeV2Leaders, want: "87 successful"},
		{path: "/v2/home/utilization?network=testnet", handler: h.HomeV2Utilization, want: "64.0%"},
	}
	for _, test := range requests {
		recorder := httptest.NewRecorder()
		test.handler(recorder, httptest.NewRequest(http.MethodGet, test.path, nil))
		if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), test.want) || strings.Contains(recorder.Body.String(), "Demo fixture") {
			t.Fatalf("%s response = %d %s", test.path, recorder.Code, recorder.Body.String())
		}
	}
	if calls != 1 {
		t.Fatalf("home summary calls = %d, want one cached snapshot", calls)
	}
}

func TestHomeEvidenceWithoutGatewayRendersUnavailable(t *testing.T) {
	h := &Handlers{Logger: testHomeLogger(), DataSource: "auto"}
	recorder := httptest.NewRecorder()
	h.HomeV2Insights(recorder, httptest.NewRequest(http.MethodGet, "/v2/home/insights?network=mainnet", nil))
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), "temporarily unavailable") || strings.Contains(recorder.Body.String(), "Demo router") {
		t.Fatalf("unavailable fragment was not truthful: %s", recorder.Body.String())
	}
}
