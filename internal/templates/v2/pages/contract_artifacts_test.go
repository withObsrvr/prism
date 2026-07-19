package pagesv2

import (
	"context"
	"strings"
	"testing"

	legacy "github.com/withObsrvr/prism/internal/templates/pages"
	ui "github.com/withObsrvr/prism/internal/viewmodel"
)

func TestContractInterfaceFragmentRendersDeclaredAndObservedEvidence(t *testing.T) {
	model := ui.ContractInterface{
		Available: true, DetectedType: "Oracle contract", FunctionCount: 1, DeclaredTypeCount: 2,
		ObservedCurrentCount: 1,
		Functions: []ui.ContractInterfaceFunction{{
			Name: "get_price", Signature: "fn get_price(asset: BytesN<8>) -> Option<PriceEntry>", Doc: "Read the latest price.", Observed: true,
			Inputs: []ui.ContractNamedType{{Name: "asset", Type: "BytesN<8>"}}, Outputs: []string{"Option<PriceEntry>"},
		}},
		DeclaredTypes: []ui.ContractDeclaredType{
			{Kind: "Struct", Name: "PriceEntry", Fields: []ui.ContractNamedType{{Name: "price", Type: "i128"}}},
			{Kind: "Error", Name: "Error", Fields: []ui.ContractNamedType{{Name: "NotInitialized", Type: "2"}}},
		},
		ObservedOnlyFunctions: []string{"old_update"},
	}
	var html strings.Builder
	if err := ContractInterfaceFragment(model).Render(context.Background(), &html); err != nil {
		t.Fatalf("render interface: %v", err)
	}
	for _, want := range []string{"Declared interface", "get_price", "BytesN&lt;8&gt;", "PriceEntry", "NotInitialized", "Observed outside the current declaration", "old_update"} {
		if !strings.Contains(html.String(), want) {
			t.Errorf("interface fragment missing %q", want)
		}
	}
}

func TestContractArtifactFragmentLabelsGeneratedInterfaceAndVerifiedDownload(t *testing.T) {
	model := ui.ContractArtifact{
		Available: true, ExecutableType: "Wasm", WASMHash: "abc123", WASMSize: "9.79 KiB (10,027 bytes)",
		ResolvedAtLedger: "3,693,849", ProvenanceLabel: "Hash verified cached artifact", ProtocolVersion: "Protocol 26",
		ExecutableSource: "Stellar rpc", CodeSource: "File cache", HasWASM: true,
		DownloadHref: "/v2/contract/CREFERENCE/wasm?network=testnet", RustHref: "/v2/contract/CREFERENCE/interface.rust?network=testnet",
	}
	var html strings.Builder
	if err := ContractArtifactFragment(model).Render(context.Background(), &html); err != nil {
		t.Fatalf("render artifact: %v", err)
	}
	for _, want := range []string{"Hash verified cached artifact", "abc123", "Download verified WASM", "Open generated Rust interface", "Interface view, not source code", "Protocol 26"} {
		if !strings.Contains(html.String(), want) {
			t.Errorf("artifact fragment missing %q", want)
		}
	}
}

func TestContractArtifactFragmentDoesNotOfferWASMForStellarAssetContract(t *testing.T) {
	model := ui.ContractArtifact{
		Available: true, IsStellarAsset: true, ExecutableType: "Stellar asset", ProvenanceLabel: "Canonical protocol interface",
		RustHref: "/v2/contract/CSAC/interface.rust?network=testnet", ExecutableSource: "Stellar rpc", CodeSource: "Protocol builtin",
	}
	var html strings.Builder
	if err := ContractArtifactFragment(model).Render(context.Background(), &html); err != nil {
		t.Fatalf("render SAC artifact: %v", err)
	}
	if !strings.Contains(html.String(), "has no standalone WASM file") || strings.Contains(html.String(), "Download verified WASM") {
		t.Fatalf("unexpected SAC artifact output: %s", html.String())
	}
}

func TestContractDetailShellLoadsInterfaceAndDefersArtifact(t *testing.T) {
	data := legacy.ContractDetailData{Name: "Oracle", Address: "CREFERENCE", Mock: true}
	var html strings.Builder
	if err := ContractDetail(data, "testnet").Render(context.Background(), &html); err != nil {
		t.Fatalf("render contract detail: %v", err)
	}
	output := html.String()
	for _, want := range []string{
		`hx-get="/fragments/contract/CREFERENCE/interface?mock=true&amp;network=testnet"`,
		`data-cn-src="/fragments/contract/CREFERENCE/artifact?mock=true&amp;network=testnet"`,
		`data-cn-tab="artifact"`,
		">WASM",
		"Loading declared contract interface",
		"Loading active contract artifact",
	} {
		if !strings.Contains(output, want) {
			t.Errorf("contract shell missing %q", want)
		}
	}
	if strings.Contains(output, "No code preview available") {
		t.Error("contract shell still renders the obsolete empty code preview")
	}
}
