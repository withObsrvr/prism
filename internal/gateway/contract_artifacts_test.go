package gateway

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestClientContractArtifactEndpoints(t *testing.T) {
	const (
		contractID = "CREFERENCE"
		wasmHash   = "01fa5b70fc7aa4c6e6fcbaffb63024d8458e06c3e5aded11badb06e96b717a19"
		etag       = `"` + wasmHash + `"`
	)
	var interfaceCalls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Api-Key test" {
			t.Errorf("authorization = %q", got)
		}
		switch r.URL.Path {
		case "/lake/v1/testnet/api/v1/silver/contracts/" + contractID + "/interface":
			if r.URL.Query().Get("format") == "rust" {
				w.Header().Set("Content-Type", "text/plain; charset=utf-8")
				w.Header().Set("X-Contract-ID", contractID)
				w.Header().Set("X-Wasm-SHA256", wasmHash)
				_, _ = io.WriteString(w, "fn get_price(asset: BytesN<8>) -> Option<PriceEntry>\n")
				return
			}
			interfaceCalls.Add(1)
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{
				"contract_id":"CREFERENCE",
				"network":"testnet",
				"detected_type":"unknown",
				"executable":{"type":"wasm","wasm_hash":"`+wasmHash+`","wasm_size_bytes":10027,"instance_last_modified_ledger":3673498,"live_until_ledger":3794457,"resolved_at_ledger":3693849},
				"interface":{
					"functions":[{"name":"get_price","doc":"Read the latest price.","inputs":[{"name":"asset","type":"BytesN<8>"}],"outputs":["Option<PriceEntry>"]}],
					"structs":[{"name":"PriceEntry","fields":[{"name":"price","type":"i128"}]}],
					"unions":[{"name":"DataKey","cases":[{"name":"PricePers","values":["BytesN<8>"]}]}],
					"enums":[],
					"errors":[{"name":"Error","cases":[{"name":"NotInitialized","value":2}]}],
					"events":[]
				},
				"metadata":[{"key":"rsver","value":"1.94.1"}],
				"environment":{"interface_version":{"protocol":26,"pre_release":0}},
				"provenance":{"executable_source":"stellar_rpc","code_source":"file_cache","code_last_modified_ledger":3673498,"resolved_at_ledger":3693849},
				"observed_functions":["get_price"]
			}`)
		case "/lake/v1/testnet/api/v1/silver/contracts/" + contractID + "/wasm":
			if r.Header.Get("If-None-Match") == etag {
				w.Header().Set("ETag", etag)
				w.WriteHeader(http.StatusNotModified)
				return
			}
			w.Header().Set("Content-Type", "application/wasm")
			w.Header().Set("Content-Disposition", `attachment; filename="contract.wasm"`)
			w.Header().Set("ETag", etag)
			w.Header().Set("X-Contract-ID", contractID)
			w.Header().Set("X-Wasm-SHA256", wasmHash)
			w.Header().Set("X-Resolved-At-Ledger", "3693849")
			_, _ = w.Write([]byte("\x00asm"))
		default:
			t.Errorf("unexpected path %s", r.URL.String())
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := New(Config{BaseURL: server.URL, APIKey: "test", Timeout: time.Second}, slog.New(slog.NewTextHandler(io.Discard, nil)), context.Background())
	defer client.Stop()

	for range 2 {
		result, err := client.GetContractInterface(context.Background(), "testnet", contractID)
		if err != nil {
			t.Fatalf("GetContractInterface: %v", err)
		}
		if result.Executable.WASMSizeBytes != 10027 || result.Executable.WASMHash != wasmHash {
			t.Fatalf("executable = %+v", result.Executable)
		}
		if len(result.Interface.Functions) != 1 || result.Interface.Functions[0].Inputs[0].Type != "BytesN<8>" {
			t.Fatalf("interface = %+v", result.Interface)
		}
		if result.Interface.Errors[0].Cases[0].Value != 2 || result.Environment.InterfaceVersion.Protocol != 26 {
			t.Fatalf("typed interface response = %+v", result)
		}
	}
	if got := interfaceCalls.Load(); got != 2 {
		t.Fatalf("contract-ID interface was cached: calls = %d, want 2", got)
	}

	rust, err := client.GetContractInterfaceRust(context.Background(), "testnet", contractID)
	if err != nil {
		t.Fatalf("GetContractInterfaceRust: %v", err)
	}
	if rust.WASMHash != wasmHash || !strings.Contains(rust.Text, "fn get_price") {
		t.Fatalf("rust response = %+v", rust)
	}

	wasm, err := client.GetContractWASM(context.Background(), "testnet", contractID, "")
	if err != nil {
		t.Fatalf("GetContractWASM: %v", err)
	}
	if wasm.StatusCode != http.StatusOK || string(wasm.Body) != "\x00asm" || wasm.ETag != etag || wasm.WASMHash != wasmHash {
		t.Fatalf("wasm response = %+v", wasm)
	}
	notModified, err := client.GetContractWASM(context.Background(), "testnet", contractID, etag)
	if err != nil {
		t.Fatalf("conditional GetContractWASM: %v", err)
	}
	if notModified.StatusCode != http.StatusNotModified || len(notModified.Body) != 0 {
		t.Fatalf("conditional wasm response = %+v", notModified)
	}
}
