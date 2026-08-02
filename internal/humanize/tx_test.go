package humanize

import (
	"strings"
	"testing"

	"github.com/withObsrvr/prism/internal/gateway"
)

func strptr(s string) *string { return &s }

func contains(s, sub string) bool { return strings.Contains(s, sub) }

func TestBuildTxNarrative_SmartWalletSwap(t *testing.T) {
	resp := &gateway.SemanticTransactionResponse{
		Classification: gateway.SemanticTransactionClassification{
			TxType:         "smart_wallet_swap",
			Subtype:        "dex_swap",
			Confidence:     "0.94",
			WalletInvolved: true,
			OperationTypes: []string{"invoke_host_function"},
		},
		Actors: []gateway.SemanticActor{
			{ActorID: "CACTOR1", ActorType: "smart_wallet", Label: strptr("Aquabot #7"), Roles: []string{"effective_actor"}},
			{ActorID: "CPROTOCOL", ActorType: "protocol", Label: strptr("Soroswap Router"), Roles: []string{"protocol"}},
		},
		Assets: gateway.SemanticAssetContext{
			Sent:     &gateway.SemanticAssetMovement{Amount: "5000", Asset: "XLM"},
			Received: &gateway.SemanticAssetMovement{Amount: "485", Asset: "USDC"},
		},
	}

	n := BuildTxNarrative(resp)
	if n.Title != "Smart wallet swap" {
		t.Fatalf("expected title %q, got %q", "Smart wallet swap", n.Title)
	}
	if n.Narrative == "" || n.Narrative == "Aquabot #7 submitted a smart wallet swap." {
		t.Fatalf("expected richer swap narrative, got %q", n.Narrative)
	}
	if len(n.Signals) == 0 {
		t.Fatalf("expected signals")
	}
}

func TestBuildTxNarrative_PolicyUpdate(t *testing.T) {
	resp := &gateway.SemanticTransactionResponse{
		Classification: gateway.SemanticTransactionClassification{
			TxType:         "smart_wallet_policy_update",
			Subtype:        "allow_signing_key",
			Confidence:     "0.91",
			WalletInvolved: true,
		},
		Actors: []gateway.SemanticActor{
			{ActorID: "CWALLET", ActorType: "smart_wallet", Label: strptr("Treasury Wallet"), Roles: []string{"effective_actor"}},
		},
	}

	n := BuildTxNarrative(resp)
	if n.Title != "Smart wallet policy update" {
		t.Fatalf("unexpected title: %q", n.Title)
	}
	if n.Narrative == "" {
		t.Fatalf("expected narrative")
	}
	found := false
	for _, s := range n.Signals {
		if s.Title == "Policy change" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected policy change signal, got %#v", n.Signals)
	}
}

func TestLookupFunctionNarration(t *testing.T) {
	rule, ok := LookupFunctionNarration("reset_all_circuit_breakers")
	if !ok {
		t.Fatalf("expected rule lookup to succeed")
	}
	if rule.Phrase != "reset all circuit breakers" {
		t.Fatalf("unexpected phrase: %q", rule.Phrase)
	}
	if rule.Signal == nil || rule.Signal.Title != "Administrative action" {
		t.Fatalf("expected administrative signal rule, got %#v", rule.Signal)
	}
}

func TestInferProject(t *testing.T) {
	if got := InferProject("Blend Risk Engine"); got != "blend" {
		t.Fatalf("expected blend project inference, got %q", got)
	}
}

func TestLookupFunctionNarrationWithContext_ProjectOverride(t *testing.T) {
	rule, ok := LookupFunctionNarrationWithContext(
		"reset_all_circuit_breakers",
		"",
		"Blend Risk Engine",
		"blend",
	)
	if !ok {
		t.Fatalf("expected project override lookup to succeed")
	}
	if rule.Signal == nil || rule.Signal.Title != "Risk controls updated" {
		t.Fatalf("expected project override signal title, got %#v", rule.Signal)
	}
}

func TestLookupFunctionNarrationWithContext_Override(t *testing.T) {
	rule, ok := LookupFunctionNarrationWithContext(
		"reset_all_circuit_breakers",
		"CBMXIVKLQ245TZ7SWQUUFVJJBUXYHM7RBABKBFBGDTPBQMABARGCQSMC",
		"CBMX...QSMC",
		"blend",
	)
	if !ok {
		t.Fatalf("expected contextual override lookup to succeed")
	}
	if rule.Signal == nil || rule.Signal.Title != "Risk controls updated" {
		t.Fatalf("expected override signal title, got %#v", rule.Signal)
	}
}

func TestHumanizeFunctionName(t *testing.T) {
	got := HumanizeFunctionName("reset_all_circuit_breakers")
	if got != "reset all circuit breakers" {
		t.Fatalf("unexpected humanized function name: %q", got)
	}
}

func TestBuildTxNarrative_AdminFunctionNarration(t *testing.T) {
	resp := &gateway.SemanticTransactionResponse{
		Classification: gateway.SemanticTransactionClassification{
			TxType:         "contract_call",
			Confidence:     "high",
			OperationTypes: []string{"invoke_host_function"},
		},
		Actors: []gateway.SemanticActor{
			{ActorID: "CBMXIVKLQ245TZ7SWQUUFVJJBUXYHM7RBABKBFBGDTPBQMABARGCQSMC", ActorType: "contract", Roles: []string{"protocol"}},
			{ActorID: "GA4AWUKYO6WAN4ROGQVPDJRR2VNKHMDEQTBOC7LCF5N3IGXUNV5DXP6R", ActorType: "classic_account", Roles: []string{"effective_actor"}},
		},
		Operations: []gateway.DecodedOperation{{FunctionName: "reset_all_circuit_breakers", TypeName: "invoke_host_function", ContractID: "CBMXIVKLQ245TZ7SWQUUFVJJBUXYHM7RBABKBFBGDTPBQMABARGCQSMC"}},
	}

	n := BuildTxNarrative(resp)
	if n.Narrative == "" || !contains(n.Narrative, "reset all circuit breakers") {
		t.Fatalf("expected admin narration, got %q", n.Narrative)
	}
	found := false
	for _, s := range n.Signals {
		if s.Title == "Administrative action" || s.Title == "Risk controls updated" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected administrative-style signal, got %#v", n.Signals)
	}
	if n.ConfidenceLabel != "High confidence" {
		t.Fatalf("expected high confidence label, got %q", n.ConfidenceLabel)
	}
}

func TestBuildTxNarrative_GenericCallUsesTransactionSourceWhenEffectiveActorsAreAmbiguous(t *testing.T) {
	source := "GAJDZBVLYPJOJRUQI2UL2C2BB52CHMKK2TWXPIFWDB4WGDKWD4K4DJBE"
	resp := &gateway.SemanticTransactionResponse{
		Transaction: gateway.SemanticTransactionInfo{SourceAccount: &source},
		Classification: gateway.SemanticTransactionClassification{
			TxType: "contract_call", Confidence: "high", OperationTypes: []string{"invoke_host_function"},
		},
		Actors: []gateway.SemanticActor{
			{ActorID: "CCBD4XHNU2W6FTBZMEQSYPYUTXZNKMYV2HFWGPIWLSEP7GT5HAEI54S3", ActorType: "contract", Roles: []string{"effective_actor", "receiver"}},
			{ActorID: "CCBDP6MHOAASY2SJW3U3ZXYPV7YBQOJ2XOFVJMD3W5CKK2IOZ4I7L443", ActorType: "contract", Roles: []string{"effective_actor", "sender"}},
			{ActorID: source, ActorType: "classic_account", Roles: []string{"effective_actor", "submitter"}},
		},
		Operations: []gateway.DecodedOperation{{FunctionName: "create_and_try_fill_with_fee", TypeName: "invoke_host_function", ContractID: "CDFWQCDY34ODL4IHTGOV6XSBKS3CRSCUYOITRKU5M3GQZBPO5UDH4364"}},
	}

	narrative := BuildTxNarrative(resp).Narrative
	if !strings.HasPrefix(narrative, "GAJD...DJBE called") {
		t.Fatalf("narrative did not use transaction source: %q", narrative)
	}
	if strings.Contains(narrative, "CCBD...54S3") {
		t.Fatalf("narrative promoted an ambiguous contract actor: %q", narrative)
	}
}

func TestBuildTxNarrative_GenericCallUsesInvokedContractInsteadOfFirstProtocolActor(t *testing.T) {
	const (
		invokedContract = "CDFWQCDY34ODL4IHTGOV6XSBKS3CRSCUYOITRKU5M3GQZBPO5UDH4364"
		tokenContract   = "CDA7SDCEQK2R6TTR655VNGEAONMNO3BSSRCFZDFNIJPADSMEKNEWRRBN"
	)
	source := "GAJDZBVLYPJOJRUQI2UL2C2BB52CHMKK2TWXPIFWDB4WGDKWD4K4DJBE"
	resp := &gateway.SemanticTransactionResponse{
		Transaction: gateway.SemanticTransactionInfo{SourceAccount: &source},
		Classification: gateway.SemanticTransactionClassification{
			TxType: "contract_call", Confidence: "high", OperationTypes: []string{"invoke_host_function"},
		},
		Actors: []gateway.SemanticActor{
			{ActorID: tokenContract, ActorType: "contract", Roles: []string{"protocol"}},
			{ActorID: invokedContract, ActorType: "contract", Roles: []string{"protocol"}},
		},
		Operations: []gateway.DecodedOperation{{
			FunctionName: "create_and_try_fill_with_fee",
			TypeName:     "invoke_host_function",
			ContractID:   invokedContract,
		}},
	}

	narrative := BuildTxNarrative(resp).Narrative
	if !strings.Contains(narrative, "CDFW...4364") {
		t.Fatalf("narrative omitted invoked contract: %q", narrative)
	}
	if strings.Contains(narrative, "CDA7...RRBN") {
		t.Fatalf("narrative promoted supporting token contract: %q", narrative)
	}
}

func TestBuildTxNarrative_GenericFallback(t *testing.T) {
	resp := &gateway.SemanticTransactionResponse{
		Classification: gateway.SemanticTransactionClassification{
			TxType:         "unknown_custom_type",
			Confidence:     "0.55",
			OperationTypes: []string{"payment"},
		},
		Actors: []gateway.SemanticActor{
			{ActorID: "GABC1234", ActorType: "classic_account", Roles: []string{"effective_actor"}},
		},
	}

	n := BuildTxNarrative(resp)
	if n.Title == "" || n.Narrative == "" {
		t.Fatalf("expected fallback title and narrative")
	}
	if n.ConfidenceLabel == "" {
		t.Fatalf("expected confidence label")
	}
	if len(n.Evidence) == 0 {
		t.Fatalf("expected evidence")
	}
}
