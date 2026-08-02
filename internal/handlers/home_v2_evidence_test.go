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
	// Scoped to one insight; this asserts how a packet is distilled, not how
	// many the fixture carries.
	summary.Insights = summary.Insights[:1]

	insights := buildHomeInsightsData(summary, "testnet", "/v2/home/insights?network=testnet")
	if insights.Status.State != vmv2.HomeSectionReady || len(insights.Cards) != 1 {
		t.Fatalf("insights = %+v", insights)
	}
	if insights.Cards[0].Summary != "" || !strings.Contains(insights.Cards[0].Detail, "swap") || !strings.Contains(insights.Cards[0].Evidence[0].Href, "status=failed") {
		t.Fatalf("insight evidence was not interpreted: %+v", insights.Cards[0])
	}
	if len(insights.Cards[0].Metrics) != 3 || insights.Cards[0].Metrics[0].Label != "Last hour" || insights.Cards[0].Metrics[1].Label != "Typical hour" || insights.Cards[0].Metrics[2].Label != "Change" {
		t.Fatalf("insight facts were not distilled for the homepage: %+v", insights.Cards[0].Metrics)
	}
	if !strings.Contains(insights.Cards[0].DetailHref, "/v2/insight/hiev1_") || !strings.Contains(insights.Cards[0].DetailHref, "network=testnet") {
		t.Fatalf("versioned insight did not expose its retained detail packet: %+v", insights.Cards[0])
	}

	ttl := buildHomeTTLData(summary, "testnet", "/v2/home/ttl?network=testnet")
	if ttl.Status.State != vmv2.HomeSectionReady || len(ttl.Cards) != 2 || ttl.Cards[0].LiveUntilLedger == "" || !strings.Contains(ttl.Cards[0].RunwayLabel, "ledgers left") {
		t.Fatalf("TTL = %+v", ttl)
	}
	if ttl.Cards[0].ContractLabel != "CAAAAAAA…AAD2KM" || ttl.Cards[0].ContractLabel == ttl.Cards[0].ContractID {
		t.Fatalf("TTL contract label was not compacted: %+v", ttl.Cards[0])
	}

	leaders := buildHomeLeadersData(summary, "testnet", "/v2/home/leaders?network=testnet")
	if leaders.Status.State != vmv2.HomeSectionReady || len(leaders.Cards) != 1 || leaders.Cards[0].CallCount != "91" || leaders.Cards[0].FailureLabel != "4.4% failed" || leaders.Cards[0].FailureTone != "healthy" {
		t.Fatalf("leaders = %+v", leaders)
	}
	if leaders.Cards[0].ContractLabel != "CBBBBBBB…BBBVQG" || leaders.Cards[0].ContractLabel == leaders.Cards[0].ContractID {
		t.Fatalf("leader contract label was not compacted: %+v", leaders.Cards[0])
	}

	utilization := buildHomeUtilizationData(summary, "testnet", "/v2/home/utilization?network=testnet")
	if utilization.Status.State != vmv2.HomeSectionReady || len(utilization.Metrics) != 3 || utilization.Metrics[0].Label != "Contract computation" || utilization.Metrics[1].Label != "Contract state access" || utilization.Metrics[0].PercentLabel != "64.0%" {
		t.Fatalf("utilization = %+v", utilization)
	}
}

func TestHomeInsightCardCompactsRawContractIdentity(t *testing.T) {
	summary := mockHomeSummaryResponse("mainnet")
	// Scoped to one insight; the assertion is about identity compaction.
	summary.Insights = summary.Insights[:1]
	item := &summary.Insights[0]
	item.Subject.Identity.DisplayName = item.Subject.ID
	item.Subject.Identity.VerificationStatus = "inferred"
	item.Subject.Identity.Source = "semantic_entities_contracts"

	data := buildHomeInsightsData(summary, "mainnet", "/v2/home/insights?network=mainnet")
	if len(data.Cards) != 1 {
		t.Fatalf("cards = %+v", data.Cards)
	}
	card := data.Cards[0]
	if card.SubjectLabel != "CAAAAAAA…AAD2KM" {
		t.Fatalf("raw contract identity was not compacted: %+v", card)
	}
	if card.IdentityDetail != "" {
		t.Fatalf("raw contract identity kept redundant provenance in the scan path: %+v", card)
	}
}

func TestHomeEvidenceDistinguishesEmptyPartialAndInvalidStates(t *testing.T) {
	summary := mockHomeSummaryResponse("testnet")
	summary.Components.Insights.Status = "empty"
	summary.Insights = nil
	empty := buildHomeInsightsData(summary, "testnet", "/v2/home/insights?network=testnet")
	if empty.Status.State != vmv2.HomeSectionEmpty || !strings.Contains(empty.Status.Message, "No significant change") {
		t.Fatalf("authoritative empty = %+v", empty)
	}

	summary = mockHomeSummaryResponse("testnet")
	summary.Components.Insights.Status = "partial"
	// One insight, deliberately corrupted: the property under test is that an
	// invalid packet is not narrated, which a second valid card would mask.
	summary.Insights = summary.Insights[:1]
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

func TestHomeInsightsUsesEvaluationAndKeepsRecentSignalSubordinate(t *testing.T) {
	summary := mockHomeSummaryResponse("testnet")
	recent := append([]gateway.HomeSummaryInsight(nil), summary.Insights...)
	summary.Insights = nil
	summary.RecentInsights = &recent
	summary.Components.Insights.Status = "empty"
	summary.InsightEvaluation = quietHomeEvaluationFixture()
	summary.InsightDelivery = &gateway.HomeInsightDelivery{Mode: "current", EvaluatedWindowEnd: summary.InsightEvaluation.WindowEnd, RetainedAt: "2026-07-31T22:00:19Z", MaxAgeSeconds: 21600, ProjectionLagSecond: 41}

	data := buildHomeInsightsData(summary, "testnet", "/v2/home/insights?network=testnet")
	if data.Status.State != vmv2.HomeSectionEmpty || len(data.Cards) != 0 || len(data.Checks) != 3 {
		t.Fatalf("quiet evaluation = %+v", data)
	}
	if data.Checks[0].Label != "Contract failures" || data.Checks[0].Value != "0.50× typical" || data.WindowLabel == "" {
		t.Fatalf("evaluation comparisons were not distilled: %+v", data)
	}
	if data.RecentLabel == "" || !strings.Contains(data.RecentDetailHref, recent[0].InsightID) {
		t.Fatalf("recent insight was not retained as history: %+v", data)
	}
}

func TestHomeInsightsRejectsContradictoryEvaluationDelivery(t *testing.T) {
	summary := mockHomeSummaryResponse("testnet")
	summary.Insights = nil
	summary.Components.Insights.Status = "empty"
	summary.InsightEvaluation = quietHomeEvaluationFixture()
	summary.InsightDelivery = &gateway.HomeInsightDelivery{Mode: "current", EvaluatedWindowEnd: "2026-07-31T23:00:00Z", RetainedAt: "2026-07-31T22:00:19Z", MaxAgeSeconds: 21600}

	data := buildHomeInsightsData(summary, "testnet", "/v2/home/insights?network=testnet")
	if data.Status.State != vmv2.HomeSectionUnavailable || len(data.Checks) != 0 {
		t.Fatalf("contradictory delivery was presented as a quiet hour: %+v", data)
	}
}

func TestHomeEvidenceMarksRetainedSummaryAsDelayed(t *testing.T) {
	summary := mockHomeSummaryResponse("mainnet")
	summary.Delivery.UsedLastGood = true

	insights := buildHomeInsightsData(summary, "mainnet", "/v2/home/insights?network=mainnet")
	leaders := buildHomeLeadersData(summary, "mainnet", "/v2/home/leaders?network=mainnet")
	utilization := buildHomeUtilizationData(summary, "mainnet", "/v2/home/utilization?network=mainnet")
	for name, status := range map[string]vmv2.HomeSectionStatus{
		"insights":    insights.Status,
		"leaders":     leaders.Status,
		"utilization": utilization.Status,
	} {
		if status.State != vmv2.HomeSectionStale || !status.Retryable || !containsString(status.Warnings, homeLastGoodWarning) {
			t.Errorf("%s retained status = %+v", name, status)
		}
	}

	ttl := buildHomeTTLData(summary, "mainnet", "/v2/home/ttl?network=mainnet")
	if ttl.Status.State != vmv2.HomeSectionStale || len(ttl.Cards) == 0 {
		t.Fatalf("retained TTL = %+v", ttl)
	}
	if ttl.Cards[0].RunwayLabel != "Availability may have changed" || ttl.Cards[0].RemainingLedgers != "" || ttl.Cards[0].LiveUntilLedger == "" || strings.Contains(ttl.Cards[0].Detail, "hours") {
		t.Fatalf("retained TTL asserted a stale relative countdown: %+v", ttl.Cards[0])
	}
	if ttl.Cards[0].Tone != "neutral" {
		t.Fatalf("retained TTL kept an urgency tone from a stale countdown: %+v", ttl.Cards[0])
	}
}

func TestHomeEvidenceDoesNotPresentRetainedEmptyAsCurrent(t *testing.T) {
	summary := mockHomeSummaryResponse("mainnet")
	summary.Delivery.UsedLastGood = true
	summary.Components.Insights.Status = "empty"
	summary.Insights = nil

	data := buildHomeInsightsData(summary, "mainnet", "/v2/home/insights?network=mainnet")
	if data.Status.State != vmv2.HomeSectionUnavailable || !data.Status.Retryable || !containsString(data.Status.Warnings, homeLastGoodWarning) {
		t.Fatalf("retained empty evidence = %+v", data)
	}
}

func TestHomeInsightsPreservesPartialAndStaleEvidence(t *testing.T) {
	for _, state := range []string{"partial", "stale"} {
		t.Run(state, func(t *testing.T) {
			summary := mockHomeSummaryResponse("testnet")
			// Scoped to one insight: this test is about how a single packet is
			// interpreted, not about the fixture size.
			summary.Insights = summary.Insights[:1]
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
	// Scoped to one insight; the assertion is about narration, not count.
	summary.Insights = summary.Insights[:1]
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

func quietHomeEvaluationFixture() *gateway.HomeInsightEvaluationEnvelope {
	number := func(value float64) *float64 { return &value }
	ledger := func(value int64) *int64 { return &value }
	first, last := int64(3903100), int64(3903157)
	rules := []gateway.HomeInsightEvaluationRule{
		{Type: "failure_spike", Family: "risk", Direction: "negative", RuleID: "contract_failure_spike", RuleVersion: "1", ComparisonMethod: "rolling_7d_median_prior_complete_hour", Status: "ready", EvaluationOutcome: "evaluated", Subject: &gateway.HomeSummaryInsightSubject{Kind: "contract", ID: "CAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAD2KM"}, EvaluatedSubjectCount: 3, ObservedValue: number(1), BaselineValue: number(2), Ratio: number(.5), MinimumObserved: number(3), MinimumRatio: number(3), RatioComparison: "at_least", ObservedFirstLedger: ledger(first), ObservedLastLedger: ledger(last), Caveats: []gateway.HomeInsightCaveat{}},
		{Type: "contract_deployments_spike", Family: "activity", Direction: "neutral", RuleID: "network_contract_deployments_spike", RuleVersion: "1", ComparisonMethod: "rolling_7d_median_prior_complete_hour", Status: "ready", EvaluationOutcome: "evaluated", Subject: &gateway.HomeSummaryInsightSubject{Kind: "network", ID: "testnet"}, EvaluatedSubjectCount: 1, ObservedValue: number(1), BaselineValue: number(2), Ratio: number(.5), MinimumObserved: number(2), MinimumRatio: number(3), RatioComparison: "at_least", ObservedFirstLedger: ledger(first), ObservedLastLedger: ledger(last), Caveats: []gateway.HomeInsightCaveat{}},
		{Type: "transaction_activity_spike", Family: "activity", Direction: "neutral", RuleID: "network_transaction_activity_spike", RuleVersion: "1", ComparisonMethod: "rolling_7d_median_prior_complete_hour", Status: "ready", EvaluationOutcome: "evaluated", Subject: &gateway.HomeSummaryInsightSubject{Kind: "network", ID: "testnet"}, EvaluatedSubjectCount: 1, ObservedValue: number(10), BaselineValue: number(20), Ratio: number(.5), MinimumRatio: number(2), RatioComparison: "at_least", ObservedFirstLedger: ledger(first), ObservedLastLedger: ledger(last), Caveats: []gateway.HomeInsightCaveat{}},
	}
	return &gateway.HomeInsightEvaluationEnvelope{EvidenceVersion: "home_insight_evaluation_v1", RegistryVersion: "home_insight_detector_registry_v1", Status: "ready", WindowStart: "2026-07-31T21:00:00Z", WindowEnd: "2026-07-31T22:00:00Z", ComparisonMethod: "rolling_7d_median_prior_complete_hour", CompleteThroughLedger: last, Rules: rules, Caveats: []gateway.HomeInsightCaveat{}, Provenance: gateway.HomeInsightEvidenceProvenance{Sources: []string{"serving.sv_home_insight_evaluations_current"}, CompleteThroughLedger: last, UpdatedAt: "2026-07-31T22:00:19Z"}}
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
		{path: "/v2/home/ttl?network=testnet", handler: h.HomeV2TTL, want: "Expires in"},
		{path: "/v2/home/leaders?network=testnet", handler: h.HomeV2Leaders, want: "4.4% failed"},
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

func TestHomeEvidenceFragmentsStayGlanceableAndNavigateToDetail(t *testing.T) {
	summary := mockHomeSummaryResponse("testnet")
	logger := testHomeLogger()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(summary); err != nil {
			t.Fatalf("encode summary: %v", err)
		}
	}))
	defer server.Close()

	client := gateway.New(gateway.Config{BaseURL: server.URL, APIKey: "test", Timeout: time.Second}, logger, context.Background())
	h := &Handlers{Logger: logger, Gateway: client, DataSource: "auto"}

	ttl := httptest.NewRecorder()
	h.HomeV2TTL(ttl, httptest.NewRequest(http.MethodGet, "/v2/home/ttl?network=testnet", nil))
	ttlBody := ttl.Body.String()
	for _, want := range []string{
		`class="ph-evidence-row is-critical"`,
		`href="/v2/contract/CAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAD2KM"`,
		`CAAAAAAA…AAD2KM`,
		"Contract data expiring soon",
		"Expires in",
		"9,800 ledgers left",
	} {
		if !strings.Contains(ttlBody, want) {
			t.Errorf("TTL fragment missing %q", want)
		}
	}
	for _, forbidden := range []string{"Shortest runway", "Inspect contract evidence", "tracked entries", "Component snapshot", "Absolute ledger runway"} {
		if strings.Contains(ttlBody, forbidden) {
			t.Errorf("TTL fragment contains redundant detail %q", forbidden)
		}
	}

	leaders := httptest.NewRecorder()
	h.HomeV2Leaders(leaders, httptest.NewRequest(http.MethodGet, "/v2/home/leaders?network=testnet", nil))
	leaderBody := leaders.Body.String()
	for _, want := range []string{
		`class="ph-leader-row"`,
		`href="/v2/contract/CBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBVQG"`,
		`CBBBBBBB…BBBVQG`,
		"Busiest contracts, 24h",
		"Callers / top function",
		"4.4% failed",
		"Top <code>swap</code>",
	} {
		if !strings.Contains(leaderBody, want) {
			t.Errorf("leader fragment missing %q", want)
		}
	}
	for _, forbidden := range []string{"Last completed 24 hours", "Updated", "Component snapshot", "87 successful", "4 failed"} {
		if strings.Contains(leaderBody, forbidden) {
			t.Errorf("leader fragment contains redundant detail %q", forbidden)
		}
	}
}

func TestHomeEvidenceSuppressesDuplicateGeneratedIdentity(t *testing.T) {
	contractID := "CAESC7SCJJY4TQG4B5RH6XK2JGTX27AQRYXMV67LOF4X2FA6KF7QM"
	if !homeDisplayIsIdentifier("CAES...F7QM", contractID) {
		t.Fatal("generated short identity was not recognized")
	}
	if homeDisplayIsIdentifier("Stellar Lumens (XLM)", contractID) {
		t.Fatal("resolved identity was mistaken for an identifier")
	}

	summary := mockHomeSummaryResponse("testnet")
	summary.ContractsNeedingAttention[0].ProtocolName = ""
	summary.ContractsNeedingAttention[0].ContractName = "CAAA...D2KM"
	data := buildHomeTTLData(summary, "testnet", "/v2/home/ttl?network=testnet")
	if data.Cards[0].ShowContractID || data.Cards[0].Name != data.Cards[0].ContractLabel {
		t.Fatalf("duplicate identifier was not collapsed: %+v", data.Cards[0])
	}
}

func TestHomeLeaderFailureUsesComparableRates(t *testing.T) {
	critical := 0.62
	warning := 0.075
	zero := 0.0
	for _, test := range []struct {
		name  string
		rate  *float64
		label string
		tone  string
	}{
		{name: "critical", rate: &critical, label: "62% failed", tone: "critical"},
		{name: "warning", rate: &warning, label: "7.5% failed", tone: "warning"},
		{name: "healthy", rate: &zero, label: "0% failed", tone: "healthy"},
		{name: "unavailable", label: "Unavailable", tone: "neutral"},
	} {
		t.Run(test.name, func(t *testing.T) {
			label, tone := homeLeaderFailure(gateway.HomeSummaryLeader{FailureRate: test.rate})
			if label != test.label || tone != test.tone {
				t.Fatalf("failure presentation = %q/%q, want %q/%q", label, tone, test.label, test.tone)
			}
		})
	}
}
