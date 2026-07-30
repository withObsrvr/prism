package gateway

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestClientDecodesAndCachesExplorerEventsV1(t *testing.T) {
	var calls int
	actor := "G" + strings.Repeat("A", 55)
	start := time.Date(2026, 7, 29, 12, 0, 0, 0, time.UTC)
	successful := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if got, want := r.URL.Path, "/lake/v1/testnet/api/v1/explorer/events"; got != want {
			t.Fatalf("path = %q, want %q", got, want)
		}
		checks := map[string]string{
			"type": "transfer,swap", "function": "transfer", "asset": "XLM", "actor": actor,
			"start_time": start.Format(time.RFC3339), "successful": "false", "limit": "48", "order": "desc",
		}
		for key, want := range checks {
			if got := r.URL.Query().Get(key); got != want {
				t.Errorf("%s = %q, want %q", key, got, want)
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{
  "evidence_version":"explorer_events_v1","status":"partial",
  "coverage":{"source":"serving.sv_explorer_events_recent","status":"complete","complete_from":3335980,"complete_thru":3853497,"updated_at":"2026-07-29T12:01:00Z"},
  "provenance":{"source":"serving.sv_explorer_events_recent","request_path":"serving_only","applied_filters":{"type":["transfer","swap"],"function":"transfer","asset":"XLM","successful":false},"count_cap":10000,"available_from_time":"2026-07-01T00:00:00Z","available_through_time":"2026-07-29T12:01:00Z"},
  "meta":{"matched_count":1,"count_capped":false,"ledger_range":{"min":3853490,"max":3853490}},
  "events":[{"event_id":"evt-1","type":"transfer","contract_id":"C1","function_name":"transfer","asset_key":"XLM","actors":[{"address":"`+actor+`","type":"account","role":"from"}],"from_address":"`+actor+`","ledger_sequence":3853490,"transaction_hash":"`+strings.Repeat("a", 64)+`","closed_at":"2026-07-29T12:00:30Z","transaction_successful":false,"event_index":2,"operation_index":1}],
  "count":1,"has_more":true,"next_cursor":"cursor-1","warnings":["start_time precedes retained serving coverage"]
}`)
	}))
	defer server.Close()

	client := New(Config{BaseURL: server.URL, APIKey: "test", Timeout: time.Second}, slog.New(slog.NewTextHandler(io.Discard, nil)), context.Background())
	defer client.Stop()
	params := ExplorerEventsParams{Types: []string{"transfer", "swap"}, Function: "transfer", Asset: "XLM", Actor: actor, StartTime: start, Successful: &successful, Limit: 48, Order: "desc"}
	for index := 0; index < 2; index++ {
		result, err := client.GetExplorerEvents(context.Background(), "testnet", params)
		if err != nil {
			t.Fatalf("GetExplorerEvents: %v", err)
		}
		if result.EvidenceVersion != "explorer_events_v1" || result.Status != "partial" || result.Coverage == nil || result.Coverage.CompleteThru != 3853497 {
			t.Fatalf("envelope = %+v", result)
		}
		if len(result.Events) != 1 || deref(result.Events[0].FunctionName) != "transfer" || deref(result.Events[0].AssetKey) != "XLM" || len(result.Events[0].Actors) != 1 {
			t.Fatalf("events = %+v", result.Events)
		}
		if result.NextCursor == nil || *result.NextCursor != "cursor-1" || result.Provenance.RequestPath != "serving_only" {
			t.Fatalf("pagination/provenance = %+v / %+v", result.NextCursor, result.Provenance)
		}
	}
	if calls != 1 {
		t.Fatalf("upstream calls = %d, want 1 cached call", calls)
	}
}

func TestClientReturnsTypedUnavailableExplorerEvents(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = io.WriteString(w, `{"evidence_version":"explorer_events_v1","status":"unavailable","provenance":{"source":"serving.sv_explorer_events_recent","request_path":"serving_only","applied_filters":{},"count_cap":10000},"meta":{"matched_count":0,"count_capped":false,"ledger_range":{"min":0,"max":0}},"events":[],"count":0,"has_more":false,"warnings":["complete watermark is not readable"]}`)
	}))
	defer server.Close()

	client := New(Config{BaseURL: server.URL, APIKey: "test", Timeout: time.Second}, slog.New(slog.NewTextHandler(io.Discard, nil)), context.Background())
	defer client.Stop()
	result, err := client.GetExplorerEvents(context.Background(), "testnet", ExplorerEventsParams{})
	if err != nil {
		t.Fatalf("GetExplorerEvents: %v", err)
	}
	if result.Status != "unavailable" || len(result.Warnings) != 1 || len(result.Events) != 0 {
		t.Fatalf("unavailable packet = %+v", result)
	}
}

func TestClientRejectsUnknownExplorerEvidenceVersionAndStatus(t *testing.T) {
	tests := []struct{ name, body, want string }{
		{name: "version", body: `{"evidence_version":"explorer_events_v2","status":"ready"}`, want: "unsupported explorer events evidence version"},
		{name: "status", body: `{"evidence_version":"explorer_events_v1","status":"stale"}`, want: "unsupported explorer events status"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = io.WriteString(w, test.body) }))
			defer server.Close()
			client := New(Config{BaseURL: server.URL, APIKey: "test", Timeout: time.Second}, slog.New(slog.NewTextHandler(io.Discard, nil)), context.Background())
			defer client.Stop()
			_, err := client.GetExplorerEvents(context.Background(), "testnet", ExplorerEventsParams{})
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want %q", err, test.want)
			}
		})
	}
}

func deref(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
