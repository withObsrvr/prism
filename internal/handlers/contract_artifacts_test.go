package handlers

import (
	"context"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/withObsrvr/prism/internal/gateway"
)

func TestContractArtifactViewsKeepDeclarationAndObservationSeparate(t *testing.T) {
	const (
		contractID = "CREFERENCE"
		wasmHash   = "01fa5b70fc7aa4c6e6fcbaffb63024d8458e06c3e5aded11badb06e96b717a19"
	)
	liveUntil := int64(3_794_457)
	response := &gateway.ContractInterface{
		ContractID:   contractID,
		DetectedType: "oracle_contract",
		Executable: gateway.ContractExecutable{
			Type: "wasm", WASMHash: wasmHash, WASMSizeBytes: 10_027,
			InstanceLastModifiedLedger: 3_673_498, LiveUntilLedger: &liveUntil, ResolvedAtLedger: 3_693_849,
		},
		Interface: gateway.ContractDeclaredInterface{
			Functions: []gateway.ContractSpecFunction{
				{Name: "get_price", Doc: "Read the latest price.", Inputs: []gateway.ContractSpecField{{Name: "asset", Type: "BytesN<8>"}}, Outputs: []string{"Option<PriceEntry>"}},
				{Name: "set_publishers", Inputs: []gateway.ContractSpecField{{Name: "publishers", Type: "Vec<BytesN<32>>"}}, Outputs: []string{"Result<Void, Error>"}},
			},
			Structs: []gateway.ContractSpecStruct{{Name: "PriceEntry", Fields: []gateway.ContractSpecField{{Name: "price", Type: "i128"}}}},
			Unions:  []gateway.ContractSpecUnion{{Name: "DataKey", Cases: []gateway.ContractSpecUnionCase{{Name: "PricePers", Values: []string{"BytesN<8>"}}}}},
			Errors:  []gateway.ContractSpecEnum{{Name: "Error", Cases: []gateway.ContractSpecEnumCase{{Name: "NotInitialized", Value: 2}}}},
		},
		Metadata:          []gateway.ContractInterfaceMetadata{{Key: "rssdkver", Value: "26.0.0"}, {Key: "rsver", Value: "1.94.1"}},
		Environment:       gateway.ContractInterfaceEnvironment{InterfaceVersion: &gateway.ContractInterfaceVersion{Protocol: 26}},
		Provenance:        gateway.ContractArtifactProvenance{ExecutableSource: "stellar_rpc", CodeSource: "file_cache", CodeLedger: 3_673_498},
		ObservedFunctions: []string{"get_price", "old_update"},
	}

	interfaceModel, artifactModel := contractArtifactViews(response, contractID, "testnet")
	if !interfaceModel.Available || interfaceModel.FunctionCount != 2 || interfaceModel.DeclaredTypeCount != 3 {
		t.Fatalf("interface model = %+v", interfaceModel)
	}
	if got := interfaceModel.Functions[0].Signature; got != "fn get_price(asset: BytesN<8>) -> Option<PriceEntry>" {
		t.Fatalf("signature = %q", got)
	}
	if !interfaceModel.Functions[0].Observed || interfaceModel.Functions[1].Observed || interfaceModel.ObservedCurrentCount != 1 {
		t.Fatalf("observed declaration mapping = %+v", interfaceModel.Functions)
	}
	if len(interfaceModel.ObservedOnlyFunctions) != 1 || interfaceModel.ObservedOnlyFunctions[0] != "old_update" {
		t.Fatalf("observed-only functions = %v", interfaceModel.ObservedOnlyFunctions)
	}
	if !artifactModel.Available || !artifactModel.HasWASM || artifactModel.WASMHash != wasmHash || artifactModel.ProtocolVersion != "Protocol 26" {
		t.Fatalf("artifact model = %+v", artifactModel)
	}
	if artifactModel.Metadata[0].Key != "rssdkver" || artifactModel.Metadata[1].Key != "rsver" {
		t.Fatalf("metadata order = %+v", artifactModel.Metadata)
	}
	if artifactModel.DownloadHref != "/v2/contract/CREFERENCE/wasm?network=testnet" || artifactModel.RustHref != "/v2/contract/CREFERENCE/interface.rust?network=testnet" {
		t.Fatalf("artifact hrefs = %q, %q", artifactModel.DownloadHref, artifactModel.RustHref)
	}
}

func TestContractArtifactViewsRepresentBuiltInSACWithoutWASM(t *testing.T) {
	_, artifact := contractArtifactViews(&gateway.ContractInterface{
		ContractID: "CSAC",
		Executable: gateway.ContractExecutable{Type: "stellar_asset", ResolvedAtLedger: 42},
		Provenance: gateway.ContractArtifactProvenance{ExecutableSource: "stellar_rpc", CodeSource: "protocol_builtin"},
	}, "CSAC", "testnet")
	if !artifact.IsStellarAsset || artifact.HasWASM || artifact.ProvenanceLabel != "Canonical protocol interface" {
		t.Fatalf("SAC artifact = %+v", artifact)
	}
}

func TestContractInterfaceAndArtifactFragmentsRenderGatewayEvidence(t *testing.T) {
	const contractID = "CREFERENCE"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/lake/v1/testnet/api/v1/silver/contracts/"+contractID+"/interface" {
			t.Fatalf("unexpected endpoint %s", r.URL.String())
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{
			"contract_id":"CREFERENCE","detected_type":"oracle_contract",
			"executable":{"type":"wasm","wasm_hash":"abc123","wasm_size_bytes":10027,"resolved_at_ledger":3693849},
			"interface":{"functions":[{"name":"get_price","doc":"Read the latest price.","inputs":[{"name":"asset","type":"BytesN<8>"}],"outputs":["Option<PriceEntry>"]}],"structs":[{"name":"PriceEntry","fields":[{"name":"price","type":"i128"}]}],"unions":[],"enums":[],"errors":[],"events":[]},
			"metadata":[],"environment":{"interface_version":{"protocol":26}},
			"provenance":{"executable_source":"stellar_rpc","code_source":"file_cache"},
			"observed_functions":["get_price","old_update"]
		}`)
	}))
	defer server.Close()

	h := contractArtifactTestHandlers(t, server.URL)
	for _, tc := range []struct {
		name string
		call func(http.ResponseWriter, *http.Request)
		want []string
	}{
		{name: "interface", call: h.ContractInterfaceFragment, want: []string{"Declared interface", "fn get_price(asset: BytesN&lt;8&gt;)", "PriceEntry", "Observed outside the current declaration", "old_update"}},
		{name: "artifact", call: h.ContractArtifactFragment, want: []string{"Hash verified cached artifact", "abc123", "Download verified WASM", "Interface view, not source code", "Protocol 26"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/fragments/contract/"+contractID+"/"+tc.name+"?network=testnet", nil)
			req.SetPathValue("id", contractID)
			recorder := httptest.NewRecorder()
			tc.call(recorder, req)
			if recorder.Code != http.StatusOK {
				t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
			}
			for _, want := range tc.want {
				if !strings.Contains(recorder.Body.String(), want) {
					t.Errorf("fragment missing %q", want)
				}
			}
		})
	}
}

func TestContractArtifactProxyPreservesValidatedResponseHeaders(t *testing.T) {
	const (
		contractID = "CREFERENCE"
		wasmHash   = "01fa5b70fc7aa4c6e6fcbaffb63024d8458e06c3e5aded11badb06e96b717a19"
		etag       = `"` + wasmHash + `"`
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/lake/v1/testnet/api/v1/silver/contracts/" + contractID + "/interface":
			w.Header().Set("X-Wasm-SHA256", wasmHash)
			_, _ = io.WriteString(w, "fn get_price() -> i128\n")
		case "/lake/v1/testnet/api/v1/silver/contracts/" + contractID + "/wasm":
			w.Header().Set("ETag", etag)
			if r.Header.Get("If-None-Match") == etag {
				w.WriteHeader(http.StatusNotModified)
				return
			}
			w.Header().Set("Content-Type", "application/wasm")
			w.Header().Set("Content-Disposition", `attachment; filename="CREFERENCE.wasm"`)
			w.Header().Set("X-Wasm-SHA256", wasmHash)
			w.Header().Set("X-Resolved-At-Ledger", "3693849")
			_, _ = w.Write([]byte("\x00asm"))
		default:
			t.Fatalf("unexpected endpoint %s", r.URL.String())
		}
	}))
	defer server.Close()
	h := contractArtifactTestHandlers(t, server.URL)

	rustRequest := httptest.NewRequest(http.MethodGet, "/v2/contract/"+contractID+"/interface.rust?network=testnet", nil)
	rustRequest.SetPathValue("id", contractID)
	rustRecorder := httptest.NewRecorder()
	h.ContractInterfaceRust(rustRecorder, rustRequest)
	if rustRecorder.Code != http.StatusOK || !strings.Contains(rustRecorder.Body.String(), "fn get_price") || rustRecorder.Header().Get("X-Wasm-SHA256") != wasmHash {
		t.Fatalf("Rust response status=%d headers=%v body=%q", rustRecorder.Code, rustRecorder.Header(), rustRecorder.Body.String())
	}
	if got, want := rustRecorder.Header().Get("Content-Disposition"), contractInterfaceContentDisposition(contractID); got != want {
		t.Fatalf("Content-Disposition = %q, want %q", got, want)
	}

	wasmRequest := httptest.NewRequest(http.MethodGet, "/v2/contract/"+contractID+"/wasm?network=testnet", nil)
	wasmRequest.SetPathValue("id", contractID)
	wasmRecorder := httptest.NewRecorder()
	h.ContractWASMDownload(wasmRecorder, wasmRequest)
	if wasmRecorder.Code != http.StatusOK || wasmRecorder.Body.String() != "\x00asm" || wasmRecorder.Header().Get("ETag") != etag || wasmRecorder.Header().Get("Content-Type") != "application/wasm" || wasmRecorder.Header().Get("Cache-Control") != "no-cache" {
		t.Fatalf("WASM response status=%d headers=%v body=%q", wasmRecorder.Code, wasmRecorder.Header(), wasmRecorder.Body.String())
	}

	conditionalRequest := httptest.NewRequest(http.MethodGet, "/v2/contract/"+contractID+"/wasm?network=testnet", nil)
	conditionalRequest.SetPathValue("id", contractID)
	conditionalRequest.Header.Set("If-None-Match", etag)
	conditionalRecorder := httptest.NewRecorder()
	h.ContractWASMDownload(conditionalRecorder, conditionalRequest)
	if conditionalRecorder.Code != http.StatusNotModified || conditionalRecorder.Body.Len() != 0 || conditionalRecorder.Header().Get("ETag") != etag {
		t.Fatalf("conditional response status=%d headers=%v body=%q", conditionalRecorder.Code, conditionalRecorder.Header(), conditionalRecorder.Body.String())
	}
}

func TestContractInterfaceContentDispositionSafelyEncodesFilename(t *testing.T) {
	for _, contractID := range []string{
		"CREFERENCE",
		`CREFERENCE"quoted`,
		`CREFERENCE\backslash`,
		"CREFERENCE\r\nX-Injected: true",
		"CREFERENCE-é",
	} {
		t.Run(contractID, func(t *testing.T) {
			header := contractInterfaceContentDisposition(contractID)
			if strings.ContainsAny(header, "\r\n") {
				t.Fatalf("Content-Disposition contains a raw control character: %q", header)
			}
			mediaType, params, err := mime.ParseMediaType(header)
			if err != nil {
				t.Fatalf("ParseMediaType(%q): %v", header, err)
			}
			if mediaType != "inline" {
				t.Errorf("media type = %q, want inline", mediaType)
			}
			if got, want := params["filename"], contractID+"-interface.rs"; got != want {
				t.Errorf("filename = %q, want %q", got, want)
			}
		})
	}
}

func contractArtifactTestHandlers(t *testing.T, baseURL string) *Handlers {
	t.Helper()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	client := gateway.New(gateway.Config{BaseURL: baseURL, APIKey: "test", Timeout: time.Second}, logger, context.Background())
	t.Cleanup(client.Stop)
	return &Handlers{Logger: logger, Gateway: client}
}
