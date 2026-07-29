package handlers

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/withObsrvr/prism/internal/gateway"
)

func TestExploreV2UsesE1CExactFiltersAndTypedEvidence(t *testing.T) {
	t.Setenv("PRISM_NETWORK", "testnet")
	const contractID = "CBHHWFAYLB3SXJCE232DC6WNSK74IBEOROAGCI2AFBA2H5NQOH2KYKNN"
	const actor = "GAHM2GC2QJRAM7EGGYIPGRVOLZTHUEDWMIGDMKXQESLKMSDLGZWPSU5E"
	txHash := strings.Repeat("a", 64)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got, want := r.URL.Path, "/lake/v1/testnet/api/v1/explorer/events"; got != want {
			t.Fatalf("path = %q, want %q", got, want)
		}
		checks := map[string]string{
			"type": "transfer", "function": "transfer", "asset": "XLM", "actor": actor,
			"contract_id": contractID, "successful": "false", "start_ledger": "3852000", "end_ledger": "3853000", "limit": "48", "order": "desc",
		}
		for key, want := range checks {
			if got := r.URL.Query().Get(key); got != want {
				t.Errorf("%s = %q, want %q", key, got, want)
			}
		}
		if r.URL.Query().Get("topic_match") != "" || r.URL.Query().Get("tab") != "" {
			t.Fatalf("live query used broad or compatibility filters: %s", r.URL.RawQuery)
		}
		startTime, err := time.Parse(time.RFC3339, r.URL.Query().Get("start_time"))
		if err != nil || time.Since(startTime) < 23*time.Hour || time.Since(startTime) > 25*time.Hour {
			t.Errorf("start_time = %q (%v)", r.URL.Query().Get("start_time"), err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{
  "evidence_version":"explorer_events_v1","status":"partial",
  "coverage":{"source":"serving.sv_explorer_events_recent","status":"complete","complete_from":3335980,"complete_thru":3853497,"updated_at":"2026-07-29T12:01:00Z"},
  "provenance":{"source":"serving.sv_explorer_events_recent","request_path":"serving_only","applied_filters":{"type":["transfer"],"function":"transfer","asset":"XLM","actor":"`+actor+`","contract_id":"`+contractID+`","successful":false},"count_cap":10000},
  "meta":{"matched_count":10000,"count_capped":true,"ledger_range":{"min":3852615,"max":3852615}},
  "events":[{"event_id":"evt-1","type":"transfer","protocol":"SEP-41 token","contract_id":"`+contractID+`","function_name":"transfer","asset_key":"XLM","actors":[{"address":"`+actor+`","type":"account","role":"from"}],"from_address":"`+actor+`","ledger_sequence":3852615,"transaction_hash":"`+txHash+`","closed_at":"2026-07-29T12:00:30Z","transaction_successful":false,"event_index":2,"operation_index":1}],
  "count":1,"has_more":true,"next_cursor":"cursor-1","warnings":["start_time precedes retained serving coverage"]
}`)
	}))
	defer server.Close()

	client := gateway.New(gateway.Config{BaseURL: server.URL, APIKey: "test", Timeout: time.Second}, testSearchLogger(), context.Background())
	defer client.Stop()
	handler := &Handlers{Logger: testSearchLogger(), Gateway: client, DataSource: "auto"}
	query := url.Values{
		"q": {contractID}, "topic": {"transfer"}, "fn": {"transfer"}, "asset": {"XLM"}, "actor": {actor},
		"time": {"24h"}, "status": {"failed"}, "start_ledger": {"3852000"}, "end_ledger": {"3853000"},
	}
	request := httptest.NewRequest(http.MethodGet, "/v2/explore/live?"+query.Encode(), nil)
	recorder := httptest.NewRecorder()
	handler.ExploreV2Live(recorder, request)
	body := recorder.Body.String()
	for _, want := range []string{
		"10,000+", "Coverage ledgers 3,335,980 to 3,853,497", "Serving-only, partial coverage",
		"Coverage is incomplete", "start_time precedes retained serving coverage", "Live gateway, partial",
		"SEP-41 token", "transfer", "Failed", "3,852,615", "cursor-1",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("Explore response missing %q: %s", want, body)
		}
	}
	if strings.Contains(body, "No matching activity yet") || strings.Contains(body, "Demo fixture") {
		t.Fatalf("typed E1C response leaked a legacy state: %s", body)
	}
}

func TestExploreV2AuthoritativeEmptyDoesNotFallBack(t *testing.T) {
	t.Setenv("PRISM_NETWORK", "testnet")
	var calls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.Path != "/lake/v1/testnet/api/v1/explorer/events" {
			t.Fatalf("authoritative empty fell back to %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"evidence_version":"explorer_events_v1","status":"empty","coverage":{"source":"serving.sv_explorer_events_recent","status":"complete","complete_from":3335980,"complete_thru":3853497},"provenance":{"source":"serving.sv_explorer_events_recent","request_path":"serving_only","applied_filters":{"contract_id":"CUNKNOWN"},"count_cap":10000},"meta":{"matched_count":0,"count_capped":false,"ledger_range":{"min":0,"max":0}},"events":[],"count":0,"has_more":false}`)
	}))
	defer server.Close()
	client := gateway.New(gateway.Config{BaseURL: server.URL, APIKey: "test", Timeout: time.Second}, testSearchLogger(), context.Background())
	defer client.Stop()
	handler := &Handlers{Logger: testSearchLogger(), Gateway: client, DataSource: "auto"}
	request := httptest.NewRequest(http.MethodGet, "/v2/explore/live?q="+strings.Repeat("C", 56), nil)
	recorder := httptest.NewRecorder()
	handler.ExploreV2Live(recorder, request)
	body := recorder.Body.String()
	if calls != 1 || !strings.Contains(body, "No matches in retained coverage") || !strings.Contains(body, "authoritative empty result") || !strings.Contains(body, "Live gateway, no matches") {
		t.Fatalf("authoritative empty handling calls=%d body=%s", calls, body)
	}
}

func TestExploreV2RejectsAmbiguousOrInvalidFiltersWithoutBroadQuery(t *testing.T) {
	t.Setenv("PRISM_NETWORK", "testnet")
	var calls int
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { calls++ }))
	defer server.Close()
	client := gateway.New(gateway.Config{BaseURL: server.URL, APIKey: "test", Timeout: time.Second}, testSearchLogger(), context.Background())
	defer client.Stop()
	handler := &Handlers{Logger: testSearchLogger(), Gateway: client, DataSource: "auto"}

	tests := []struct{ rawQuery, want string }{
		{rawQuery: "asset=USDC", want: "A bare code can refer to multiple issuers"},
		{rawQuery: "start_ledger=200&end_ledger=100", want: "From ledger must be less than or equal"},
		{rawQuery: "actor=not-an-address", want: "Actor must be a complete"},
		{rawQuery: "scope=classic", want: "Classic operations are not part"},
	}
	for index, test := range tests {
		t.Run(strconv.Itoa(index), func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/v2/explore/live?"+test.rawQuery, nil)
			recorder := httptest.NewRecorder()
			handler.ExploreV2Live(recorder, request)
			body := recorder.Body.String()
			for _, want := range []string{"Query needs more detail", "Query not run", "No evidence claimed", test.want} {
				if !strings.Contains(body, want) {
					t.Errorf("invalid query missing %q: %s", want, body)
				}
			}
		})
	}
	if calls != 0 {
		t.Fatalf("invalid filters made %d upstream calls", calls)
	}
}

func TestExploreLedgerBoundsDefaultToRetainedCoverage(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/v2/explore?start_ledger=3852615&end_ledger=3852615", nil)
	filters := exploreFiltersFromRequest(request)
	if filters.Time != "coverage" {
		t.Fatalf("time filter = %q, want retained coverage", filters.Time)
	}
	params, message := explorerParamsForFilters(filters, "", time.Now())
	if message != "" || !params.StartTime.IsZero() || params.StartLedger != 3852615 || params.EndLedger != 3852615 {
		t.Fatalf("ledger-bound params = %+v, message = %q", params, message)
	}
}
