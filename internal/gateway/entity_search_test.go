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

func TestClientDecodesAndCachesEntitySearchV1(t *testing.T) {
	var calls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if got, want := r.URL.Path, "/lake/v1/testnet/api/v1/silver/search"; got != want {
			t.Fatalf("path = %q, want %q", got, want)
		}
		if got := r.URL.Query()["type"]; len(got) != 2 || got[0] != "asset" || got[1] != "sac" {
			t.Fatalf("type filters = %#v", got)
		}
		if got := r.URL.Query().Get("limit"); got != "2" {
			t.Fatalf("limit = %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{
  "evidence_version":"entity_search_v1","query":"USDC","status":"ready","limit":2,"type_filters":["asset","sac"],"has_more":true,
  "results":[{"type":"asset","entity_kind":"classic_asset","id":"USDC:GISSUER1","canonical_slug":"USDC:GISSUER1","label":"USD Coin","display_name":"USD Coin","symbol":"USDC","matched_field":"symbol","match_type":"exact","identity_source":"serving.sv_assets_current","verification_status":"verified","details":{"asset_code":"USDC","asset_issuer":"GISSUER1","sac_contract_id":"CSAC"}}],
  "provenance":{"source":"serving.sv_entity_search_current","complete_through_ledger":500,"request_path":"serving_only","fuzzy_threshold":0.6}
}`)
	}))
	defer server.Close()

	client := New(Config{BaseURL: server.URL, APIKey: "test", Timeout: time.Second}, slog.New(slog.NewTextHandler(io.Discard, nil)), context.Background())
	defer client.Stop()
	for index := 0; index < 2; index++ {
		result, err := client.SearchEntities(context.Background(), "testnet", "USDC", SearchOptions{Limit: 2, Types: []string{"asset", "sac"}})
		if err != nil {
			t.Fatalf("SearchEntities: %v", err)
		}
		if result.EvidenceVersion != "entity_search_v1" || result.Status != "ready" || !result.HasMore || result.Provenance.CompleteThroughLedger != 500 {
			t.Fatalf("search envelope = %+v", result)
		}
		if len(result.Results) != 1 || result.Results[0].EntityKind != "classic_asset" || result.Results[0].MatchType != "exact" || result.Results[0].VerificationStatus != "verified" {
			t.Fatalf("search result = %+v", result.Results)
		}
	}
	if calls != 1 {
		t.Fatalf("upstream calls = %d, want 1 cached call", calls)
	}
}

func TestClientReturnsTypedUnavailableEntitySearch(t *testing.T) {
	var calls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = io.WriteString(w, `{"evidence_version":"entity_search_v1","query":"USDC","status":"unavailable","limit":10,"has_more":false,"results":[],"warnings":["entity search projection has no complete source watermark"],"provenance":{"source":"serving.sv_entity_search_current","complete_through_ledger":0,"request_path":"serving_only","fuzzy_threshold":0.6}}`)
	}))
	defer server.Close()

	client := New(Config{BaseURL: server.URL, APIKey: "test", Timeout: time.Second}, slog.New(slog.NewTextHandler(io.Discard, nil)), context.Background())
	defer client.Stop()
	for index := 0; index < 2; index++ {
		result, err := client.Search(context.Background(), "testnet", "USDC")
		if err != nil {
			t.Fatalf("Search: %v", err)
		}
		if result.Status != "unavailable" || len(result.Warnings) != 1 || len(result.Results) != 0 {
			t.Fatalf("unavailable packet = %+v", result)
		}
	}
	if calls != 1 {
		t.Fatalf("upstream calls = %d, want 1 cached call", calls)
	}
}

func TestClientRejectsUnknownEntitySearchVersionAndStatus(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{name: "version", body: `{"evidence_version":"entity_search_v2","status":"ready"}`, want: "unsupported entity search evidence version"},
		{name: "status", body: `{"evidence_version":"entity_search_v1","status":"stale"}`, want: "unsupported entity search status"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = io.WriteString(w, test.body)
			}))
			defer server.Close()
			client := New(Config{BaseURL: server.URL, APIKey: "test", Timeout: time.Second}, slog.New(slog.NewTextHandler(io.Discard, nil)), context.Background())
			defer client.Stop()
			_, err := client.Search(context.Background(), "testnet", "USDC")
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want %q", err, test.want)
			}
		})
	}
}
