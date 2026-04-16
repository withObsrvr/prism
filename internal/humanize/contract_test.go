package humanize

import (
	"strings"
	"testing"

	"github.com/withObsrvr/prism/internal/gateway"
)

func TestBuildContractSummary_TokenLike(t *testing.T) {
	meta := &gateway.ContractMetadata{
		DisplayName:         "USDC Token",
		TotalEntries:        12,
		TotalStateSizeBytes: 16384,
		CreatorAddress:      "GABC1234",
		ExportedFunctions: []gateway.ContractExportedFunctionMetadata{
			{Name: "approve"},
			{Name: "transfer"},
			{Name: "transfer_from"},
		},
	}
	s := BuildContractSummary(meta, nil)
	if s.Narrative == "" {
		t.Fatalf("expected narrative")
	}
	if s.FunctionSummary == "" {
		t.Fatalf("expected function summary")
	}
	found := false
	for _, sig := range s.Signals {
		if sig.Title == "Token-like interface" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected token-like signal, got %#v", s.Signals)
	}
}

func TestBuildContractSummary_ProjectInference(t *testing.T) {
	meta := &gateway.ContractMetadata{
		DisplayName: "Blend Risk Engine",
		ExportedFunctions: []gateway.ContractExportedFunctionMetadata{
			{Name: "reset_all_circuit_breakers"},
		},
	}
	s := BuildContractSummary(meta, nil)
	found := false
	for _, sig := range s.Signals {
		if sig.Title == "Known project" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected known project signal, got %#v", s.Signals)
	}
}

func TestBuildContractSummary_ManagedToken(t *testing.T) {
	meta := &gateway.ContractMetadata{
		DisplayName: "XTEST",
		ExportedFunctions: []gateway.ContractExportedFunctionMetadata{
			{Name: "mint_to"},
			{Name: "burn"},
			{Name: "transfer"},
			{Name: "add_to_blocked_list"},
			{Name: "set_revoker"},
		},
	}
	s := BuildContractSummary(meta, nil)
	if got := s.Narrative; got == "" || got == "XTEST exposes 5 functions and looks like an active Soroban contract." {
		t.Fatalf("expected managed token narrative, got %q", got)
	}
	found := false
	for _, sig := range s.Signals {
		if sig.Title == "Managed token controls" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected managed token signal, got %#v", s.Signals)
	}
}

func TestBuildContractSummary_OracleWithObservedActivity(t *testing.T) {
	meta := &gateway.ContractMetadata{
		ContractID:  "CAUGTEST",
		DisplayName: "Price Feed",
		ExportedFunctions: []gateway.ContractExportedFunctionMetadata{
			{Name: "set_price"},
		},
		TotalEntries:        3,
		TotalStateSizeBytes: 660,
	}
	analytics := &gateway.ContractAnalytics{
		TopFunctions: []gateway.ContractFunction{{Name: "set_price", Count: 10368}},
	}
	s := BuildContractSummary(meta, analytics)
	if !strings.Contains(strings.ToLower(s.Narrative), "oracle") {
		t.Fatalf("expected oracle-style narrative, got %q", s.Narrative)
	}
	for _, sig := range s.Signals {
		if sig.Title == "Limited activity" {
			t.Fatalf("did not expect limited activity signal when top function counts exist: %#v", s.Signals)
		}
	}
	if !strings.Contains(strings.ToLower(s.FunctionSummary), "price updates") {
		t.Fatalf("expected function summary to use polished oracle phrasing, got %q", s.FunctionSummary)
	}
}

func TestBuildContractSummary_FactoryStyle(t *testing.T) {
	meta := &gateway.ContractMetadata{
		DisplayName: "Pool Factory",
		ExportedFunctions: []gateway.ContractExportedFunctionMetadata{
			{Name: "create_pool"},
			{Name: "create_vault"},
		},
	}
	analytics := &gateway.ContractAnalytics{
		TopFunctions: []gateway.ContractFunction{{Name: "create_pool", Count: 4671}, {Name: "create_vault", Count: 1}},
	}
	s := BuildContractSummary(meta, analytics)
	if !strings.Contains(strings.ToLower(s.Narrative), "factory") {
		t.Fatalf("expected factory-style narrative, got %q", s.Narrative)
	}
	if !strings.Contains(strings.ToLower(s.FunctionSummary), "pool creation") {
		t.Fatalf("expected function summary to use polished factory phrasing, got %q", s.FunctionSummary)
	}
	found := false
	for _, sig := range s.Signals {
		if sig.Title == "Factory-style behavior" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected factory-style signal, got %#v", s.Signals)
	}
}
