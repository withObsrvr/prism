package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/withObsrvr/prism/internal/gateway"
)

func TestBuildContractDetailDataDoesNotWaitForBalanceSoftDependency(t *testing.T) {
	const contractID = "CABYF6CONTRACT"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/lake/v1/testnet/api/v1/silver/contracts/" + contractID + "/storage":
			_, _ = io.WriteString(w, `{"entries":[]}`)
		case "/lake/v1/testnet/api/v1/silver/contracts/" + contractID + "/metadata":
			_, _ = fmt.Fprintf(w, `{"contract_id":%q,"display_name":"Market contract","exported_functions":[{"name":"swap"}]}`, contractID)
		case "/lake/v1/testnet/api/v1/silver/contracts/" + contractID + "/analytics":
			_, _ = fmt.Fprintf(w, `{"contract_id":%q,"top_functions":[{"name":"swap","count":2}]}`, contractID)
		case "/lake/v1/testnet/api/v1/silver/contracts/" + contractID + "/recent-calls":
			_, _ = io.WriteString(w, `[]`)
		case "/lake/v1/testnet/api/v1/silver/addresses/" + contractID + "/balances":
			t.Fatalf("initial contract shell must not call the balance fragment dependency")
		default:
			t.Fatalf("unexpected endpoint %s", r.URL.String())
		}
	}))
	defer server.Close()

	gw := gateway.New(gateway.Config{BaseURL: server.URL, APIKey: "test", Timeout: time.Second}, slog.New(slog.NewTextHandler(io.Discard, nil)), context.Background())
	defer gw.Stop()
	h := &Handlers{Logger: slog.New(slog.NewTextHandler(io.Discard, nil)), Gateway: gw}
	req := httptest.NewRequest(http.MethodGet, "/v2/contract/"+contractID+"?network=testnet", nil)

	data, err := h.buildContractDetailData(req, "testnet", contractID)
	if err != nil {
		t.Fatalf("buildContractDetailData error = %v", err)
	}
	if data.Name != "Market contract" || data.Portfolio.Available {
		t.Fatalf("detail data = %+v, portfolio = %+v", data, data.Portfolio)
	}
}

func TestContractDetailWithoutGatewayKeepsRequestedIdentity(t *testing.T) {
	const contractID = "CREQUESTEDCONTRACT"
	h := &Handlers{Logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
	req := httptest.NewRequest(http.MethodGet, "/v2/contract/"+contractID+"?network=testnet", nil)
	req.SetPathValue("id", contractID)

	data, ok := h.contractDetailDataForRequest(httptest.NewRecorder(), req)
	if !ok {
		t.Fatal("contract detail unexpectedly redirected")
	}
	if data.Address != contractID || data.Name != gateway.ShortAddress(contractID) || data.Portfolio.Available {
		t.Fatalf("contract detail = %+v", data)
	}
}

// Fixtures mirror the real serving-endpoint response shape:
// key_decoded / value_decoded are {type, value, display} wrappers,
// `type` holds the clean durability, `durability` the XDR enum name.
func TestBuildStorageExplorerMapsServingEntries(t *testing.T) {
	entries := []gateway.ContractStorageEntry{
		{
			Key:          "0030b9a50d287d23",
			KeyHash:      "a1b2c3d4e5",
			Type:         "instance",
			Durability:   "ContractDataDurabilityInstance",
			SizeBytes:    64,
			TTLRemaining: 604800, // ~35 days
			KeyDecoded:   json.RawMessage(`{"type":"symbol","value":"Admin","display":"Admin"}`),
			ValueDecoded: json.RawMessage(`{"type":"address","value":"GABCDEFGHIJKLMNOPQRSTUVWXYZ234567ABCDEFGHIJKLMNOPQRSTUVW","display":"GABC…TUVW"}`),
		},
		{
			Key:          "003efa2b74a6e64b",
			KeyHash:      "b0c1d2e3f4",
			Type:         "persistent",
			Durability:   "ContractDataDurabilityPersistent",
			SizeBytes:    172,
			TTLRemaining: 43200, // ~2.5 days -> critical
			KeyDecoded:   json.RawMessage(`{"type":"vec","value":[{"type":"symbol","value":"Balance","display":"Balance"},{"type":"address","value":"GDKX","display":"GDKX"}],"display":"[Balance, GDKX]"}`),
			ValueDecoded: json.RawMessage(`{"type":"bytes","value":{"base64":"j7GwRg==","hex":"8fb1b046","length":4},"display":"bytes[4]"}`),
		},
		{
			Key:        "0063f7595cfb7fe5",
			KeyHash:    "c7d8e9f0a1",
			Durability: "ContractDataDurabilityTemporary", // no clean `type` -> enum fallback
			SizeBytes:  32,
			// no TTL info -> health unknown, excluded from counts
			DataValue: "AAAAAQAAAAY=",
		},
		{
			// The fetch uses live_only=false, so expired entries appear in
			// the response and must be dropped by the mapper.
			Key:       "00ffdeadbeef0000",
			KeyHash:   "d9e8f7a6b5",
			Type:      "temporary",
			SizeBytes: 64,
			Expired:   true,
		},
	}

	items, stats, types := buildStorageExplorer(entries)

	if len(items) != 3 {
		t.Fatalf("items = %d, want 3", len(items))
	}

	admin := items[0]
	if admin.Key != "Admin" {
		t.Errorf("decoded key = %q, want Admin", admin.Key)
	}
	if admin.ValueType != "Address" {
		t.Errorf("value type = %q, want Address", admin.ValueType)
	}
	if admin.Value != "GABC…TUVW" {
		t.Errorf("value = %q, want display string", admin.Value)
	}
	if admin.TTLDays != 35 {
		t.Errorf("ttl days = %d, want 35", admin.TTLDays)
	}
	if admin.HealthPct != 58 { // 604800 / 1036800
		t.Errorf("health pct = %d, want 58", admin.HealthPct)
	}
	if admin.Type != "Instance" {
		t.Errorf("type = %q, want Instance", admin.Type)
	}

	balance := items[1]
	if balance.Key != "Balance:GDKX" {
		t.Errorf("decoded vec key = %q, want Balance:GDKX", balance.Key)
	}
	if balance.ValueType != "Bytes" {
		t.Errorf("value type = %q, want Bytes", balance.ValueType)
	}
	if !strings.HasPrefix(balance.Value, "0x8fb1b046") {
		t.Errorf("bytes value = %q, want 0x-prefixed hex", balance.Value)
	}
	if balance.HealthPct >= 15 {
		t.Errorf("health pct = %d, want critical (<15)", balance.HealthPct)
	}

	nonce := items[2]
	if nonce.Type != "Temporary" {
		t.Errorf("durability enum fallback = %q, want Temporary", nonce.Type)
	}
	if nonce.HealthPct != 0 || nonce.TTLLedgers != 0 {
		t.Errorf("entry without TTL should have unknown health, got pct=%d ledgers=%d", nonce.HealthPct, nonce.TTLLedgers)
	}
	if nonce.ValueType != "XDR" {
		t.Errorf("raw fallback value type = %q, want XDR", nonce.ValueType)
	}

	if stats.Healthy != "1" || stats.Critical != "1" || stats.AtRisk != "0" {
		t.Errorf("stats = %+v, want 1 healthy / 0 at risk / 1 critical", stats)
	}

	if len(types) != 3 {
		t.Fatalf("type breakdown = %d groups, want 3", len(types))
	}
	if types[0].Name != "Instance" || types[1].Name != "Persistent" || types[2].Name != "Temporary" {
		t.Errorf("type order = %s/%s/%s, want Instance/Persistent/Temporary", types[0].Name, types[1].Name, types[2].Name)
	}
	if types[2].MinTTL != "—" {
		t.Errorf("temporary min TTL = %q, want — for unknown TTL", types[2].MinTTL)
	}
}

// Real shapes from the testnet price oracle CARSV4BT...Z3LP and the
// contract-instance ledger entry.
func TestDecodeStorageValueRecursesNestedTypes(t *testing.T) {
	priceMap := gateway.ContractStorageEntry{
		ValueDecoded: json.RawMessage(`{"type":"map","value":{"entries":[
			{"key":{"type":"symbol","value":"asset","display":"asset"},"value":{"type":"symbol","value":"USDC","display":"USDC"}},
			{"key":{"type":"symbol","value":"price","display":"price"},"value":{"type":"i128","value":"999850000000000000","display":"999850000000000000"}},
			{"key":{"type":"symbol","value":"publish_time","display":"publish_time"},"value":{"type":"u64","value":"1783424328","display":"1783424328"}},
			{"key":{"type":"symbol","value":"source","display":"source"},"value":{"type":"vec","value":[{"type":"symbol","value":"RedStone","display":"RedStone"}],"display":"[RedStone]"}}
		]},"display":"map{4}"}`),
	}
	val, vt := decodeStorageValue(priceMap)
	if vt != "Map" {
		t.Errorf("value type = %q, want Map", vt)
	}
	for _, want := range []string{"asset: USDC", "price: 999850000000000000", "source: [RedStone]"} {
		if !strings.Contains(val, want) {
			t.Errorf("map render %q missing %q", val, want)
		}
	}
	if !strings.Contains(val, "\n") {
		t.Errorf("top-level map with >3 entries should be multi-line, got %q", val)
	}

	instance := gateway.ContractStorageEntry{
		KeyDecoded:   json.RawMessage(`{"type":"ledger_key_contract_instance","value":"instance","display":"instance"}`),
		ValueDecoded: json.RawMessage(`{"type":"contract_instance","value":{"executable_type":"ContractExecutableTypeContractExecutableWasm","storage_entries":22},"display":"contract_instance"}`),
	}
	if got := decodeStorageKey(instance); got != "Contract instance" {
		t.Errorf("instance key = %q, want Contract instance", got)
	}
	val, vt = decodeStorageValue(instance)
	if vt != "Instance" {
		t.Errorf("instance value type = %q, want Instance", vt)
	}
	if val != "Wasm executable · 22 instance storage entries" {
		t.Errorf("instance value = %q, want Wasm summary", val)
	}
}

func TestPrettyScType(t *testing.T) {
	cases := map[string]string{
		"u64": "U64", "i128": "i128", "bool": "Bool",
		"vec": "Vec", "map": "Map", "address": "Address", "symbol": "Symbol",
	}
	for in, want := range cases {
		if got := prettyScType(in); got != want {
			t.Errorf("prettyScType(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestFormatRentXLM(t *testing.T) {
	cases := map[int64]string{
		542991360:      "54.3", // real testnet metadata value
		12345678901234: "1,234,568",
		9_500_000:      "0.95",
		5_000_000:      "0.50",
		100_000_000:    "10.0",
		10_000_000_000: "1,000",
	}
	for in, want := range cases {
		if got := formatRentXLM(in); got != want {
			t.Errorf("formatRentXLM(%d) = %q, want %q", in, got, want)
		}
	}
}
