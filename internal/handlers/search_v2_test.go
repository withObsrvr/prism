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
	asset := "USDC:" + account
	queries := []struct {
		query string
		kind  prismsearch.SearchResolutionKind
	}{
		{tx, prismsearch.SearchOpen},
		{"please open account " + account, prismsearch.SearchOpen},
		{asset, prismsearch.SearchOpen},
		{"please open asset " + asset, prismsearch.SearchOpen},
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
		_, _ = io.WriteString(w, `{"evidence_version":"entity_search_v1","query":"stellar","status":"ready","limit":4,"has_more":true,"results":[{"type":"contract","entity_kind":"contract","id":"C1","canonical_slug":"C1","label":"One","matched_field":"display_name","match_type":"exact","identity_source":"serving.sv_contracts_current","verification_status":"verified"},{"type":"contract","entity_kind":"contract","id":"C2","canonical_slug":"C2","label":"Two","matched_field":"display_name","match_type":"prefix","identity_source":"serving.sv_contracts_current","verification_status":"unverified"},{"type":"asset","entity_kind":"classic_asset","id":"A3","canonical_slug":"A3","label":"Three","matched_field":"display_name","match_type":"prefix","identity_source":"serving.sv_assets_current","verification_status":"unverified"},{"type":"account","entity_kind":"account","id":"G4","canonical_slug":"G4","label":"Four","matched_field":"display_name","match_type":"prefix","identity_source":"serving.sv_accounts_current","verification_status":"unverified"},{"type":"ledger","entity_kind":"ledger","id":"5","canonical_slug":"5","label":"Five","matched_field":"entity_id","match_type":"prefix","identity_source":"serving.sv_ledger_stats_recent","verification_status":"verified"}],"provenance":{"source":"serving.sv_entity_search_current","complete_through_ledger":500,"request_path":"serving_only","fuzzy_threshold":0.6}}`)
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

func TestEntitySearchPreservesAmbiguousAssetsAndEvidence(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"evidence_version":"entity_search_v1","query":"USDC","status":"ready","limit":7,"has_more":true,"results":[{"type":"asset","entity_kind":"classic_asset","id":"USDC:GISSUER1","canonical_slug":"USDC:GISSUER1","label":"USD Coin One","symbol":"USDC","matched_field":"symbol","match_type":"exact","identity_source":"serving.sv_assets_current","verification_status":"verified","details":{"asset_code":"USDC","asset_issuer":"GISSUER1"}},{"type":"asset","entity_kind":"classic_asset","id":"USDC:GISSUER2","canonical_slug":"USDC:GISSUER2","label":"USD Coin Two","symbol":"USDC","matched_field":"symbol","match_type":"exact","identity_source":"serving.sv_assets_current","verification_status":"unverified","details":{"asset_code":"USDC","asset_issuer":"GISSUER2"}}],"provenance":{"source":"serving.sv_entity_search_current","complete_through_ledger":500,"request_path":"serving_only","fuzzy_threshold":0.6}}`)
	}))
	defer server.Close()
	client := gateway.New(gateway.Config{BaseURL: server.URL, APIKey: "test", Timeout: time.Second}, testSearchLogger(), context.Background())
	defer client.Stop()
	handler := &Handlers{Logger: testSearchLogger(), Gateway: client, DataSource: "auto"}

	resolution, evidence := handler.resolveSearchWithEntities(context.Background(), "testnet", "USDC")
	if resolution.Kind != prismsearch.SearchExplore || resolution.RuleID != "gateway.entity.ambiguous" || resolution.Destination != "/v2/explore?q=USDC" {
		t.Fatalf("ambiguous resolution = %+v", resolution)
	}
	if len(evidence.Entities) != 2 || !evidence.HasMore {
		t.Fatalf("ambiguous evidence = %+v", evidence)
	}

	request := httptest.NewRequest(http.MethodGet, "/search/suggest?q=USDC", nil)
	recorder := httptest.NewRecorder()
	handler.SearchSuggest(recorder, request)
	output := recorder.Body.String()
	for _, want := range []string{"USD Coin One", "issuer GISSUER1", "USD Coin Two", "issuer GISSUER2", "Exact symbol match", "Verified identity", "Unverified identity", "More matches exist", "ledger 500", "will not select one implicitly"} {
		if !strings.Contains(output, want) {
			t.Errorf("ambiguous suggestion missing %q: %s", want, output)
		}
	}
	if strings.Contains(output, `href="/v2/assets/USDC"`) {
		t.Fatalf("ambiguous suggestion leaked the built-in single-asset shortcut: %s", output)
	}
}

func TestEntitySearchRoutesSACPoolFuzzyEmptyAndUnavailableTruthfully(t *testing.T) {
	const sacID = "CAW2SVC7HTEFP64JVQSHIZNOYCOKPE54IPCSAD3AKG2ZYMUWQFQB7KVH"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		query := r.URL.Query().Get("q")
		switch query {
		case sacID:
			_, _ = io.WriteString(w, `{"evidence_version":"entity_search_v1","query":"`+sacID+`","status":"ready","limit":7,"has_more":false,"results":[{"type":"contract","entity_kind":"sac","id":"`+sacID+`","canonical_slug":"USDC:GISSUER1","label":"USD Coin","symbol":"USDC","matched_field":"entity_id","match_type":"exact","identity_source":"silver.token_registry","verification_status":"verified","details":{"asset_code":"USDC","asset_issuer":"GISSUER1","is_sac":true}}],"provenance":{"source":"serving.sv_entity_search_current","complete_through_ledger":500,"request_path":"serving_only","fuzzy_threshold":0.6}}`)
		case "4cd1f6de":
			_, _ = io.WriteString(w, `{"evidence_version":"entity_search_v1","query":"4cd1f6de","status":"ready","limit":7,"has_more":false,"results":[{"type":"liquidity_pool","entity_kind":"liquidity_pool","id":"4cd1f6defba237eecbc5fefe259f89ebc4b5edd49116beb5536c4034fc48d63f","canonical_slug":"pool:4cd1f6defba237eecbc5fefe259f89ebc4b5edd49116beb5536c4034fc48d63f","label":"XLM / USDC pool","matched_field":"entity_id","match_type":"prefix","identity_source":"silver.liquidity_pools_current","verification_status":"verified","details":{"asset_pair":"XLM / USDC:GISSUER1"}}],"provenance":{"source":"serving.sv_entity_search_current","complete_through_ledger":500,"request_path":"serving_only","fuzzy_threshold":0.6}}`)
		case "USDCC":
			_, _ = io.WriteString(w, `{"evidence_version":"entity_search_v1","query":"USDCC","status":"ready","limit":7,"has_more":false,"results":[{"type":"asset","entity_kind":"classic_asset","id":"USDC:GISSUER1","canonical_slug":"USDC:GISSUER1","label":"USD Coin","symbol":"USDC","matched_field":"search_terms","match_type":"fuzzy","identity_source":"serving.sv_assets_current","verification_status":"unverified","details":{"asset_code":"USDC","asset_issuer":"GISSUER1"}}],"provenance":{"source":"serving.sv_entity_search_current","complete_through_ledger":500,"request_path":"serving_only","fuzzy_threshold":0.6}}`)
		case "definitely-not-an-entity":
			_, _ = io.WriteString(w, `{"evidence_version":"entity_search_v1","query":"definitely-not-an-entity","status":"ready","limit":7,"has_more":false,"results":[],"provenance":{"source":"serving.sv_entity_search_current","complete_through_ledger":500,"request_path":"serving_only","fuzzy_threshold":0.6}}`)
		case "USDC":
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = io.WriteString(w, `{"evidence_version":"entity_search_v1","query":"USDC","status":"unavailable","limit":7,"has_more":false,"results":[],"warnings":["entity search projection has no complete source watermark"],"provenance":{"source":"serving.sv_entity_search_current","complete_through_ledger":0,"request_path":"serving_only","fuzzy_threshold":0.6}}`)
		case "EURC":
			_, _ = io.WriteString(w, `{"evidence_version":"entity_search_v1","query":"EURC","status":"partial","limit":7,"has_more":false,"results":[],"warnings":["optional exact lookup did not complete"],"provenance":{"source":"serving.sv_entity_search_current","complete_through_ledger":500,"request_path":"serving_only","fuzzy_threshold":0.6}}`)
		default:
			t.Fatalf("unexpected query %q", query)
		}
		filters := r.URL.Query()["type"]
		switch query {
		case sacID:
			if strings.Join(filters, ",") != "contract,sac,protocol_contract" {
				t.Fatalf("contract filters = %#v", filters)
			}
		case "USDCC", "USDC", "EURC":
			if strings.Join(filters, ",") != "asset,sac" {
				t.Fatalf("asset filters = %#v", filters)
			}
		case "4cd1f6de", "definitely-not-an-entity":
			if len(filters) != 0 {
				t.Fatalf("generic filters = %#v", filters)
			}
		}
	}))
	defer server.Close()
	client := gateway.New(gateway.Config{BaseURL: server.URL, APIKey: "test", Timeout: time.Second}, testSearchLogger(), context.Background())
	defer client.Stop()
	handler := &Handlers{Logger: testSearchLogger(), Gateway: client, DataSource: "auto"}

	tests := []struct {
		query       string
		kind        prismsearch.SearchResolutionKind
		rule        string
		destination string
	}{
		{query: sacID, kind: prismsearch.SearchOpen, rule: "gateway.entity.sac", destination: "/v2/assets/USDC:GISSUER1"},
		{query: "4cd1f6de", kind: prismsearch.SearchExplore, rule: "gateway.entity.liquidity_pool", destination: "/v2/explore?q=4cd1f6defba237eecbc5fefe259f89ebc4b5edd49116beb5536c4034fc48d63f"},
		{query: "USDCC", kind: prismsearch.SearchUnsupported, rule: "gateway.entity.close_matches", destination: "/v2/search/unsupported?q=USDCC"},
		{query: "definitely-not-an-entity", kind: prismsearch.SearchUnsupported, rule: "unsupported.no_supported_rule", destination: "/v2/search/unsupported?q=definitely-not-an-entity"},
		{query: "USDC", kind: prismsearch.SearchUnsupported, rule: "gateway.entity.unavailable", destination: "/v2/search/unsupported?q=USDC"},
		{query: "EURC", kind: prismsearch.SearchUnsupported, rule: "gateway.entity.partial", destination: "/v2/search/unsupported?q=EURC"},
	}
	for _, test := range tests {
		t.Run(test.query, func(t *testing.T) {
			resolution := handler.resolveSearch(context.Background(), "testnet", test.query)
			if resolution.Kind != test.kind || resolution.RuleID != test.rule || resolution.Destination != test.destination {
				t.Fatalf("resolution = %+v", resolution)
			}
		})
	}

	for _, test := range []struct {
		query string
		want  []string
	}{
		{query: sacID, want: []string{"Open linked asset", "Stellar Asset Contract", "Exact ID match", "Verified identity", "token registry"}},
		{query: "4cd1f6de", want: []string{"Explore liquidity pool", "Liquidity pool", "Prefix ID match", "XLM / USDC:GISSUER1"}},
		{query: "USDCC", want: []string{"No exact match", "Fuzzy name or symbol match", "will not open a fuzzy match automatically"}},
		{query: "definitely-not-an-entity", want: []string{"No live entity matches", "authoritative empty result"}},
		{query: "USDC", want: []string{"Entity lookup unavailable", "Live entity lookup unavailable", "did not guess", "has not published a complete network snapshot"}},
		{query: "EURC", want: []string{"Entity lookup incomplete", "Some live matches may be missing", "did not treat that as no match", "optional exact lookup did not complete"}},
	} {
		t.Run(test.query+" suggestion", func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/search/suggest?q="+urlQueryEscape(test.query), nil)
			recorder := httptest.NewRecorder()
			handler.SearchSuggest(recorder, request)
			output := recorder.Body.String()
			for _, want := range test.want {
				if !strings.Contains(output, want) {
					t.Errorf("suggestion missing %q: %s", want, output)
				}
			}
		})
	}
}

func TestGatewaySearchEntitySupportsAllE1BKinds(t *testing.T) {
	tests := []struct {
		name string
		in   gateway.SearchResult
		want string
	}{
		{name: "account", in: gateway.SearchResult{EntityKind: "account", ID: "GACCOUNT", Label: "Account"}, want: "/v2/account/GACCOUNT"},
		{name: "asset", in: gateway.SearchResult{EntityKind: "classic_asset", ID: "USDC:GISSUER", CanonicalSlug: "USDC:GISSUER", Label: "USDC"}, want: "/v2/assets/USDC:GISSUER"},
		{name: "contract", in: gateway.SearchResult{EntityKind: "contract", ID: "CCONTRACT", Label: "Contract"}, want: "/v2/contract/CCONTRACT"},
		{name: "sac", in: gateway.SearchResult{EntityKind: "sac", ID: "CSAC", CanonicalSlug: "USDC:GISSUER", Label: "USDC"}, want: "/v2/assets/USDC:GISSUER"},
		{name: "pool", in: gateway.SearchResult{EntityKind: "liquidity_pool", ID: "POOLID", Label: "Pool"}, want: "/v2/explore?q=POOLID"},
		{name: "protocol", in: gateway.SearchResult{EntityKind: "protocol", ID: "soroswap", DisplayName: "Soroswap", Label: "Soroswap"}, want: "/v2/explore?q=Soroswap"},
		{name: "protocol contract", in: gateway.SearchResult{EntityKind: "protocol_contract", ID: "CPROTOCOL", Label: "Router"}, want: "/v2/contract/CPROTOCOL"},
		{name: "transaction", in: gateway.SearchResult{EntityKind: "transaction", ID: strings.Repeat("a", 64), Label: "Transaction"}, want: "/v2/tx/" + strings.Repeat("a", 64)},
		{name: "ledger", in: gateway.SearchResult{EntityKind: "ledger", ID: "500", Label: "Ledger"}, want: "/v2/ledger/500"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			entity, ok := searchEntityFromGateway(test.in)
			if !ok || entity.Href != test.want {
				t.Fatalf("entity = %+v, ok=%t", entity, ok)
			}
		})
	}
}

func TestGatewayAssetSuggestionDisclosesReverseSACMapping(t *testing.T) {
	entity, ok := searchEntityFromGateway(gateway.SearchResult{
		EntityKind:    "classic_asset",
		ID:            "USDC:GISSUER",
		CanonicalSlug: "USDC:GISSUER",
		Label:         "USDC",
		Details: map[string]any{
			"asset_issuer":    "GISSUER",
			"sac_contract_id": "CSACCONTRACT",
		},
	})
	if !ok || entity.SACContractID != "CSACCONTRACT" || !strings.Contains(entity.Subtitle, "SAC CSACCONTRACT") {
		t.Fatalf("asset entity = %+v, ok=%t", entity, ok)
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

	shellRequest := httptest.NewRequest(http.MethodGet, "/v2/explore?asset=XLM", nil)
	shellRecorder := httptest.NewRecorder()
	handler.ExploreV2(shellRecorder, shellRequest)
	if output := shellRecorder.Body.String(); strings.Contains(output, "Alice swapped") || strings.Contains(output, "52,844,201") || !strings.Contains(output, "/v2/explore/live?asset=XLM") {
		t.Fatalf("live Explore shell leaked fixtures or lost hydration: %s", output)
	}

	liveRequest := httptest.NewRequest(http.MethodGet, "/v2/explore/live?asset=XLM", nil)
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
