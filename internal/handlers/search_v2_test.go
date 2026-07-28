package handlers

import (
	"context"
	"html"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/withObsrvr/prism/internal/gateway"
	prismsearch "github.com/withObsrvr/prism/internal/search"
)

func TestSearchSuggestAndSubmitUseSameResolution(t *testing.T) {
	handler := &Handlers{Logger: testSearchLogger(), DataSource: "auto"}
	tx := strings.Repeat("a", 64)
	account := "G" + strings.Repeat("A", 55)
	queries := []struct {
		query string
		kind  prismsearch.SearchResolutionKind
	}{
		{tx, prismsearch.SearchOpen},
		{"please open account " + account, prismsearch.SearchOpen},
		{"failed USDC transfers last hour", prismsearch.SearchExplore},
		{"USDC transfers this week", prismsearch.SearchExplore},
		{"How active is XLM today?", prismsearch.SearchAnswer},
		{"How active is USDC today?", prismsearch.SearchUnsupported},
		{"Which contracts are nearing archival?", prismsearch.SearchAnswer},
		{"tell me whether the ecosystem feels healthy", prismsearch.SearchUnsupported},
	}
	for _, test := range queries {
		t.Run(test.query, func(t *testing.T) {
			resolution := handler.resolveSearch(context.Background(), "mainnet", test.query)
			if resolution.Kind != test.kind {
				t.Fatalf("resolution kind = %s, want %s: %+v", resolution.Kind, test.kind, resolution)
			}

			suggestRequest := httptest.NewRequest(http.MethodGet, "/search/suggest?q="+urlQueryEscape(test.query), nil)
			suggestRecorder := httptest.NewRecorder()
			handler.SearchSuggest(suggestRecorder, suggestRequest)
			suggestBody := suggestRecorder.Body.String()
			if !strings.Contains(suggestBody, `href="`+html.EscapeString(resolution.Destination)+`"`) {
				t.Fatalf("suggestion did not use destination %q: %s", resolution.Destination, suggestBody)
			}
			if !strings.Contains(suggestBody, resolution.ActionLabel) || !strings.Contains(suggestBody, searchKindLabel(resolution.Kind)) {
				t.Fatalf("suggestion did not disclose resolution %+v: %s", resolution, suggestBody)
			}

			submitRequest := httptest.NewRequest(http.MethodGet, "/search/submit?q="+urlQueryEscape(test.query), nil)
			submitRecorder := httptest.NewRecorder()
			handler.SearchSubmit(submitRecorder, submitRequest)
			if got := submitRecorder.Result().Header.Get("Location"); got != resolution.Destination {
				t.Fatalf("submit destination = %q, suggest resolution = %q", got, resolution.Destination)
			}
		})
	}
}

func TestUnsupportedSearchDoesNotFallThroughToExplore(t *testing.T) {
	handler := &Handlers{Logger: testSearchLogger(), DataSource: "auto"}
	query := "tell me whether the ecosystem feels healthy"
	resolution := handler.resolveSearch(context.Background(), "mainnet", query)
	if resolution.Kind != prismsearch.SearchUnsupported || strings.Contains(resolution.Destination, "/v2/explore") {
		t.Fatalf("unsupported query resolved as %+v", resolution)
	}

	request := httptest.NewRequest(http.MethodGet, resolution.Destination, nil)
	recorder := httptest.NewRecorder()
	handler.SearchUnsupportedV2(recorder, request)
	output := recorder.Body.String()
	for _, want := range []string{"Prism does not have a safe answer", query, "unsupported.no_supported_rule", "Try a query Prism can prove"} {
		if !strings.Contains(output, want) {
			t.Errorf("unsupported page missing %q", want)
		}
	}
}

func TestDynamicSearchSuggestionsAreGatewayBackedAndBounded(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/lake/v1/testnet/api/v1/silver/search" {
			t.Fatalf("unexpected dynamic search request %s?%s", r.URL.Path, r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"query":"stellar","results":[{"type":"contract","id":"C1","label":"One"},{"type":"contract","id":"C2","label":"Two"},{"type":"asset","id":"A3","label":"Three"},{"type":"account","id":"G4","label":"Four"},{"type":"ledger","id":"5","label":"Five"}]}`)
	}))
	defer server.Close()
	client := gateway.New(gateway.Config{BaseURL: server.URL, APIKey: "test", Timeout: time.Second}, testSearchLogger(), context.Background())
	defer client.Stop()
	handler := &Handlers{Logger: testSearchLogger(), Gateway: client, DataSource: "auto"}

	entities := handler.dynamicSearchEntities(context.Background(), "testnet", "stellar", 4)
	if len(entities) != 4 {
		t.Fatalf("dynamic entities = %d, want 4", len(entities))
	}
	if entities[0].Href != "/v2/contract/C1" || entities[3].Href != "/v2/account/G4" {
		t.Fatalf("dynamic entity routes = %+v", entities)
	}
	resolution := handler.resolveSearch(context.Background(), "testnet", "One")
	if resolution.Kind != prismsearch.SearchOpen || resolution.Destination != "/v2/contract/C1" || resolution.RuleID != "gateway.entity.contract" {
		t.Fatalf("exact dynamic entity resolution = %+v", resolution)
	}
}

func TestGatewaySearchEntitySupportsSmartAccountsAndTokens(t *testing.T) {
	smart, ok := searchEntityFromGateway(gateway.SearchResult{Type: "smart_wallet", ID: "CSMART", Label: "Passkey wallet"})
	if !ok || smart.Href != "/v2/account/CSMART/smart" || smart.Type != prismsearch.EntityAccount {
		t.Fatalf("smart account entity = %+v, ok=%t", smart, ok)
	}
	token, ok := searchEntityFromGateway(gateway.SearchResult{Type: "token", ID: "CTOKEN", Label: "Example token", Details: map[string]any{"canonical_slug": "EXAMPLE"}})
	if !ok || token.Href != "/v2/assets/EXAMPLE" || token.Type != prismsearch.EntityAsset {
		t.Fatalf("token entity = %+v, ok=%t", token, ok)
	}
}

func TestExploreLiveModeNeverFallsBackToMockRows(t *testing.T) {
	handler := &Handlers{Logger: testSearchLogger(), DataSource: "auto"}

	shellRequest := httptest.NewRequest(http.MethodGet, "/v2/explore?asset=USDC", nil)
	shellRecorder := httptest.NewRecorder()
	handler.ExploreV2(shellRecorder, shellRequest)
	if output := shellRecorder.Body.String(); strings.Contains(output, "Alice swapped") || strings.Contains(output, "52,844,201") || !strings.Contains(output, "/v2/explore/live?asset=USDC") {
		t.Fatalf("live Explore shell leaked fixtures or lost hydration: %s", output)
	}

	liveRequest := httptest.NewRequest(http.MethodGet, "/v2/explore/live?asset=USDC", nil)
	liveRecorder := httptest.NewRecorder()
	handler.ExploreV2Live(liveRecorder, liveRequest)
	output := liveRecorder.Body.String()
	if strings.Contains(output, "Alice swapped") || strings.Contains(output, "Design fallback") || !strings.Contains(output, "Evidence unavailable") {
		t.Fatalf("unavailable Explore response was not truthful: %s", output)
	}
}

func TestExploreExplicitMockModeIsLabeled(t *testing.T) {
	handler := &Handlers{Logger: testSearchLogger(), DataSource: "auto"}
	request := httptest.NewRequest(http.MethodGet, "/v2/explore/live?mock=true", nil)
	recorder := httptest.NewRecorder()
	handler.ExploreV2Live(recorder, request)
	output := recorder.Body.String()
	if !strings.Contains(output, "Demo fixture") || !strings.Contains(output, "Alice swapped") {
		t.Fatalf("explicit Explore demo was not labeled: %s", output)
	}
}

func TestAskUnsupportedQueryUsesEducationalState(t *testing.T) {
	handler := &Handlers{Logger: testSearchLogger(), DataSource: "auto"}
	request := httptest.NewRequest(http.MethodGet, "/v2/ask?q=explain+the+whole+ecosystem", nil)
	recorder := httptest.NewRecorder()
	handler.AskV2(recorder, request)
	location := recorder.Result().Header.Get("Location")
	if !strings.HasPrefix(location, "/v2/search/unsupported?") {
		t.Fatalf("unsupported Ask location = %q", location)
	}
}

func TestAskUnavailableEvidenceDoesNotClaimGatewayBackedAnswer(t *testing.T) {
	handler := &Handlers{Logger: testSearchLogger(), DataSource: "auto"}
	request := httptest.NewRequest(http.MethodGet, "/v2/ask?q=Which+contracts+have+the+least+TTL+runway%3F", nil)
	recorder := httptest.NewRecorder()
	handler.AskV2(recorder, request)
	output := recorder.Body.String()
	for _, want := range []string{"Evidence unavailable", "Routing confidence", "gateway evidence unavailable", "Gateway client is not configured"} {
		if !strings.Contains(output, want) {
			t.Errorf("unavailable Ask page missing %q", want)
		}
	}
	if strings.Contains(output, "gateway-backed deterministic input") || strings.Contains(output, ">Confidence<") {
		t.Fatalf("unavailable Ask page overstated evidence: %s", output)
	}
}

func TestNetworkFromRequestUsesConfiguredDefault(t *testing.T) {
	t.Setenv("PRISM_NETWORK", "testnet")
	request := httptest.NewRequest(http.MethodGet, "/v2/home", nil)
	if got := networkFromRequest(request); got != "testnet" {
		t.Fatalf("default network = %q, want testnet", got)
	}
	request = httptest.NewRequest(http.MethodGet, "/v2/home?network=mainnet", nil)
	if got := networkFromRequest(request); got != "mainnet" {
		t.Fatalf("query network = %q, want mainnet", got)
	}
}

func urlQueryEscape(value string) string { return url.QueryEscape(value) }

func testSearchLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
