package pagesv2

import (
	"context"
	"strings"
	"testing"

	"github.com/withObsrvr/prism/internal/gateway"
	legacy "github.com/withObsrvr/prism/internal/templates/pages"
	txv2 "github.com/withObsrvr/prism/internal/templates/v2/viewmodel"
)

func TestClassicMultiOperationHeroRendersForFullPageAndHTMX(t *testing.T) {
	data := classicMultiOperationReceiptFixture()
	components := map[string]struct {
		render func(*strings.Builder) error
	}{
		"full page": {
			render: func(out *strings.Builder) error {
				return TransactionReceipt(data, "mainnet").Render(context.Background(), out)
			},
		},
		"hero fragment": {
			render: func(out *strings.Builder) error {
				return TxReceiptHeroFragment(data).Render(context.Background(), out)
			},
		},
		"detail fragment": {
			render: func(out *strings.Builder) error {
				return TxReceiptDetailFragment(data).Render(context.Background(), out)
			},
		},
	}

	for name, component := range components {
		t.Run(name, func(t *testing.T) {
			var out strings.Builder
			if err := component.render(&out); err != nil {
				t.Fatalf("render: %v", err)
			}
			html := out.String()
			if strings.Contains(html, "Called contract") || strings.Contains(html, "What was called") {
				t.Fatalf("classic transaction rendered contract-call copy: %s", html)
			}
			if name != "detail fragment" && !strings.Contains(html, "14 operations") {
				t.Fatalf("render missing operation count: %s", html)
			}
			if name == "detail fragment" {
				if !strings.Contains(html, "Manage Buy Offer") || !strings.Contains(html, "Manage Sell Offer") {
					t.Fatalf("detail fragment lost operation evidence: %s", html)
				}
			}
		})
	}
}

func TestTransactionQuickFactsUseV2EntityLinksAndDedupeContractLabel(t *testing.T) {
	const (
		sourceID      = "GBTHAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"
		contractID    = "CAYIAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"
		sourceShort   = "GBTH...GVQG"
		contractShort = "CAYI...4YBG"
	)
	data := legacy.TxReceiptData{
		Hash:                    "abc",
		ShortHash:               "abc",
		Status:                  "success",
		IsSoroban:               true,
		EffectiveActorAddr:      sourceID,
		EffectiveActorShort:     sourceShort,
		EffectiveActorHref:      "/account/" + sourceID,
		DownstreamContractAddr:  contractID,
		DownstreamContractShort: contractShort,
		DownstreamContractHref:  "/contracts/" + contractID,
		DownstreamFunctionName:  "update_ed25519_persistent",
		SourceAddr:              sourceShort,
		SourceAddrFull:          sourceID,
		ContractName:            contractShort,
		ContractAddr:            contractShort,
		ContractAddrFull:        contractID,
		ContractFn:              "update_ed25519_persistent",
		FeePaidXLM:              "0.0007847 XLM",
	}

	var out strings.Builder
	if err := TxReceiptHeroFragment(data).Render(context.Background(), &out); err != nil {
		t.Fatalf("render hero fragment: %v", err)
	}
	html := out.String()
	for _, want := range []string{
		`href="/v2/account/` + sourceID + `"`,
		`href="/v2/contract/` + contractID + `"`,
		`>` + contractShort + `</a>`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("quick facts missing %q: %s", want, html)
		}
	}
	if duplicate := contractShort + " " + contractShort; strings.Contains(html, duplicate) {
		t.Errorf("contract address rendered twice as %q", duplicate)
	}
}

func TestTransactionQuickFactsRouteSmartWalletSourceToV2SmartPage(t *testing.T) {
	const walletID = "CWALLETAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"
	data := legacy.TxReceiptData{
		EffectiveActorAddr:  walletID,
		EffectiveActorType:  "smart_wallet",
		EffectiveActorHref:  "/account/" + walletID + "/smart",
		SmartWalletDetected: true,
		SmartWalletContract: walletID,
	}
	if got, want := txSourceHref(data), "/v2/account/"+walletID+"/smart"; got != want {
		t.Fatalf("txSourceHref = %q, want %q", got, want)
	}
}

func TestTransactionSidebarOmitsContractCardWithoutContractEvidence(t *testing.T) {
	classic := legacy.TxReceiptData{Hash: "abc", ShortHash: "abc", Ledger: "123", LedgerRaw: "123", ContractName: "—", ContractAddr: "—"}
	contract := classic
	contract.ContractName = "Example contract"
	contract.ContractAddr = "CABC...WXYZ"
	contract.ContractAddrFull = "CABCAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAWXYZ"

	components := map[string]struct {
		render func(legacy.TxReceiptData, *strings.Builder) error
	}{
		"full page": {render: func(data legacy.TxReceiptData, out *strings.Builder) error {
			return TransactionReceipt(data, "testnet").Render(context.Background(), out)
		}},
		"sidebar fragment": {render: func(data legacy.TxReceiptData, out *strings.Builder) error {
			return TxReceiptSidebarFragment(data).Render(context.Background(), out)
		}},
	}

	for name, component := range components {
		t.Run(name, func(t *testing.T) {
			var out strings.Builder
			if err := component.render(classic, &out); err != nil {
				t.Fatalf("render classic sidebar: %v", err)
			}
			if strings.Contains(out.String(), "About this contract") || strings.Contains(out.String(), "/contracts/—") {
				t.Fatalf("classic sidebar invented contract context: %s", out.String())
			}

			out.Reset()
			if err := component.render(contract, &out); err != nil {
				t.Fatalf("render contract sidebar: %v", err)
			}
			if !strings.Contains(out.String(), "About this contract") || !strings.Contains(out.String(), contract.ContractAddrFull) {
				t.Fatalf("contract sidebar omitted available evidence: %s", out.String())
			}
		})
	}
}

func TestGenericCallHeroUsesLayeredCardStructure(t *testing.T) {
	hero := txv2.TxHeroModel{
		GenericCall: &txv2.TxGenericCallHero{
			SummaryHTML: `<b>GBTH...GVQG</b> called <code>update_ed25519_persistent()</code> on <b>CAYI...4YBG</b>.`,
		},
		Meta: txv2.TxHeroMeta{
			Fee:       "0.0007847 XLM",
			Resources: "1.4M insn",
			Ledger:    "3,691,421",
			Status:    "Successful",
		},
	}

	var out strings.Builder
	if err := txGenericCallHero(hero).Render(context.Background(), &out); err != nil {
		t.Fatalf("render generic call hero: %v", err)
	}
	html := out.String()
	for _, want := range []string{
		`px-hero-generic-head`,
		`class="px-hero-summary-copy"`,
		`class="px-tx-flow-foot"`,
		"What was called",
		"0.0007847 XLM",
	} {
		if !strings.Contains(html, want) {
			t.Errorf("generic call hero missing %q: %s", want, html)
		}
	}
}

func TestTransactionOperationUsesCompactSummaryAndOneContractLink(t *testing.T) {
	const (
		contractShort = "CAXY...Z10P"
		contractID    = "CAXY7K2M8P4Q9R1S2T3U4V5W6X7Y8Z10PABCDEFGHIJKLMNOPQRSTUVWX"
	)
	data := legacy.TxReceiptData{
		Operations: []legacy.TxOperation{{
			Index:        "1",
			Type:         "Invoke Contract",
			Status:       "Success",
			SummaryHTML:  "Swapped 5,000 XLM for 485 USDC",
			DetailHTML:   "Completed in one invocation",
			Contract:     contractShort,
			ContractFull: contractID,
			Function:     "swap()",
		}},
	}

	var out strings.Builder
	if err := txOperations(data).Render(context.Background(), &out); err != nil {
		t.Fatalf("render operations: %v", err)
	}
	html := out.String()
	for _, want := range []string{
		`class="px-tx-op-summary"`,
		`class="px-tx-op-detail"`,
		`href="/v2/contract/` + contractID + `"`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("compact operation missing %q: %s", want, html)
		}
	}
	if got := strings.Count(html, contractShort); got != 1 {
		t.Errorf("contract label rendered %d times, want once: %s", got, html)
	}
	if strings.Contains(html, `>summary</span>`) {
		t.Errorf("operation includes redundant summary label: %s", html)
	}
}

func TestFailureHeroUsesAuthoritativeOutcomeEvidence(t *testing.T) {
	operationIndex := 1
	contractID := "CAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABSC4"
	data := legacy.TxReceiptData{
		Hash:                strings.Repeat("a", 64),
		ShortHash:           "aaaa...aaaa",
		Status:              "failed",
		EffectiveActorShort: "GABC...WXYZ",
		EffectiveActorAddr:  "GABC",
		OutcomeEvidence: &gateway.TransactionOutcome{
			EvidenceVersion: "transaction_outcome_v1",
			Status:          "partial",
			Outcome:         "failed",
			Failure: &gateway.TransactionFailureEvidence{
				Status:         "ready",
				Phase:          "soroban_host",
				Scope:          "host_function",
				NormalizedCode: "invoke_host_function_trapped",
				OperationIndex: &operationIndex,
				OperationType:  "INVOKE_HOST_FUNCTION",
			},
			Operations: []gateway.TransactionOutcomeOperation{
				{OperationIndex: 0, ExecutionOutcome: "succeeded", AppliedToLedger: false},
				{OperationIndex: 1, ExecutionOutcome: "failed", AppliedToLedger: false},
			},
			PrimaryInvocation: &gateway.TransactionPrimaryInvocation{
				OperationIndex: 1,
				ContractID:     contractID,
				FunctionName:   "swap",
				Arguments: []gateway.DecodedScVal{
					{Type: "address", Display: "GAAA...AWHF"},
					{Type: "u32", Display: "7"},
				},
			},
			Caveats: []gateway.TransactionOutcomeCaveat{{Code: "diagnostic_evidence_incomplete", Message: "Diagnostic coverage is incomplete."}},
		},
	}

	var out strings.Builder
	if err := TxReceiptHeroFragment(data).Render(context.Background(), &out); err != nil {
		t.Fatalf("render hero fragment: %v", err)
	}
	html := out.String()
	for _, want := range []string{
		"swap() failed",
		"Contract stopped unexpectedly",
		"invoke_host_function_trapped",
		"General failure category",
		"Exact cause unavailable",
		"What Prism knows",
		"No changes from this transaction were saved to the ledger.",
		"Operation #2",
		"GAAA...AWHF",
		"Diagnostic coverage is incomplete.",
		"One earlier operation executed successfully",
	} {
		if !strings.Contains(html, want) {
			t.Errorf("failure hero missing %q: %s", want, html)
		}
	}
	if strings.Contains(html, "Execution reverted") || strings.Contains(html, ">reverted<") {
		t.Errorf("failure hero retained generic reverted copy: %s", html)
	}
	if strings.Contains(strings.ToLower(html), "execution trapped") {
		t.Errorf("failure hero exposed unexplained trapped jargon: %s", html)
	}
	if got := strings.Count(html, "swap() failed"); got != 1 {
		t.Errorf("failure heading rendered %d times, want once: %s", got, html)
	}
	if got := strings.Count(html, ">Fee<"); got != 1 {
		t.Errorf("fee rendered %d times in hero, want once: %s", got, html)
	}
}

func TestSingleOperationFailureOmitsRedundantExecutionStack(t *testing.T) {
	operationIndex := 0
	data := legacy.TxReceiptData{
		Hash:                strings.Repeat("b", 64),
		ShortHash:           "bbbb...bbbb",
		Status:              "failed",
		EffectiveActorShort: "GABC...WXYZ",
		EffectiveActorAddr:  "GABC",
		OutcomeEvidence: &gateway.TransactionOutcome{
			EvidenceVersion: "transaction_outcome_v1",
			Status:          "ready",
			Outcome:         "failed",
			Failure: &gateway.TransactionFailureEvidence{
				Status:         "ready",
				Phase:          "soroban_host",
				Scope:          "host_function",
				NormalizedCode: "invoke_host_function_trapped",
				OperationIndex: &operationIndex,
				OperationType:  "INVOKE_HOST_FUNCTION",
			},
			Operations: []gateway.TransactionOutcomeOperation{{OperationIndex: 0, ExecutionOutcome: "failed"}},
			PrimaryInvocation: &gateway.TransactionPrimaryInvocation{
				OperationIndex: 0,
				ContractID:     "CAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABSC4",
				FunctionName:   "open_with_price",
				Arguments:      []gateway.DecodedScVal{{Type: "u32", Display: "7"}},
			},
		},
	}

	var out strings.Builder
	if err := TxReceiptHeroFragment(data).Render(context.Background(), &out); err != nil {
		t.Fatalf("render hero fragment: %v", err)
	}
	html := out.String()
	if strings.Contains(html, "px-hero-stack") {
		t.Fatalf("single-operation failure retained redundant execution stack: %s", html)
	}
	if !strings.Contains(html, "Invocation details") || !strings.Contains(html, "Technical details") {
		t.Fatalf("progressive failure details are missing: %s", html)
	}
}

func TestFailureHeroWithoutE1ADoesNotGuessRevert(t *testing.T) {
	data := legacy.TxReceiptData{Status: "failed", SourceAddr: "GABC...WXYZ"}
	var out strings.Builder
	if err := TxReceiptHeroFragment(data).Render(context.Background(), &out); err != nil {
		t.Fatalf("render hero fragment: %v", err)
	}
	html := out.String()
	if !strings.Contains(html, "Failure reason unavailable") || !strings.Contains(html, "could not load the structured transaction outcome evidence") || !strings.Contains(html, "Structured outcome evidence unavailable") {
		t.Fatalf("missing truthful unavailable state: %s", html)
	}
	if strings.Contains(html, "Execution reverted") {
		t.Fatalf("unavailable evidence was rendered as a revert: %s", html)
	}
}

func TestTransactionOperationsDistinguishRollbackFromFailure(t *testing.T) {
	data := legacy.TxReceiptData{Operations: []legacy.TxOperation{
		{Index: "1", Type: "Manage Data", Status: "Executed, not applied"},
		{Index: "2", Type: "Payment", Status: "Failed"},
		{Index: "3", Type: "Payment", Status: "Not executed"},
	}}
	var out strings.Builder
	if err := txOperations(data).Render(context.Background(), &out); err != nil {
		t.Fatalf("render operations: %v", err)
	}
	html := out.String()
	for _, want := range []string{`px-tx-op-status rolled-back`, `px-tx-op-status failed`, `px-tx-op-status not-executed`, `Executed, not applied`} {
		if !strings.Contains(html, want) {
			t.Errorf("operation states missing %q: %s", want, html)
		}
	}
}

func classicMultiOperationReceiptFixture() legacy.TxReceiptData {
	operations := make([]legacy.TxOperation, 0, 14)
	for i := 0; i < 7; i++ {
		operations = append(operations,
			legacy.TxOperation{Index: "buy", Type: "Manage Buy Offer", Status: "Success"},
			legacy.TxOperation{Index: "sell", Type: "Manage Sell Offer", Status: "Success"},
		)
	}
	return legacy.TxReceiptData{
		Hash:                "abc",
		ShortHash:           "abc",
		Status:              "success",
		SemanticTxType:      "multi_op",
		EffectiveActorShort: "GABC...WXYZ",
		EffectiveActorAddr:  "GABC",
		SourceAddr:          "GABC...WXYZ",
		SourceAddrFull:      "GABC",
		Ledger:              "63,000,000",
		LedgerRaw:           "63000000",
		OpsCount:            "14",
		EventsCount:         "0",
		FeePaidXLM:          "0.00014 XLM",
		Operations:          operations,
	}
}
