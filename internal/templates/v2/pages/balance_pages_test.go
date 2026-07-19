package pagesv2

import (
	"context"
	"strings"
	"testing"

	legacy "github.com/withObsrvr/prism/internal/templates/pages"
	ui "github.com/withObsrvr/prism/internal/viewmodel"
)

func TestSmartAccountRendersMaterializedPortfolioWithoutUSDEstimate(t *testing.T) {
	data := legacy.SmartAccountData{
		Name: "Smart account", ContractID: "CWALLET", MinSigners: "2 of 3",
		Portfolio: ui.BalancePortfolio{
			Available: true, NativeBalance: "19,971.7336815", Count: 3, SourceLabel: "Current contract storage", Status: "materialized",
			Items: []ui.BalanceItem{
				{AssetCode: "XLM", AssetType: "native", Balance: "19,971.7336815", SourceLabel: "Current contract storage"},
				{AssetCode: "USDC", AssetType: "credit_alphanum4", AssetIssuer: "GISSUER1", IssuerHref: "/v2/account/GISSUER1?network=testnet", Balance: "71.0157688", SourceLabel: "Current contract storage"},
				{AssetCode: "USDC", AssetType: "credit_alphanum4", AssetIssuer: "GISSUER2", IssuerHref: "/v2/account/GISSUER2?network=testnet", Balance: "7", SourceLabel: "Current contract storage"},
			},
		},
	}
	var html strings.Builder
	if err := SmartAccount(data, "testnet").Render(context.Background(), &html); err != nil {
		t.Fatalf("render SmartAccount: %v", err)
	}
	output := html.String()
	for _, want := range []string{"Current balances", "19,971.7336815", "GISSUER1", "GISSUER2", "3 holdings"} {
		if !strings.Contains(output, want) {
			t.Errorf("rendered smart-account page missing %q", want)
		}
	}
	for _, unwanted := range []string{"Estimated value", " USD"} {
		if strings.Contains(output, unwanted) {
			t.Errorf("rendered smart-account page contains misleading copy %q", unwanted)
		}
	}
}

func TestContractRendersBalanceUnavailableWithoutHidingDetail(t *testing.T) {
	data := legacy.ContractDetailData{
		Name: "Market contract", Address: "CCONTRACT", Narrative: "A market contract with observed activity.",
		Portfolio: ui.BalancePortfolio{OwnerID: "CCONTRACT"},
	}
	var html strings.Builder
	if err := ContractDetail(data, "testnet").Render(context.Background(), &html); err != nil {
		t.Fatalf("render ContractDetail: %v", err)
	}
	output := html.String()
	for _, want := range []string{"Market contract", "A market contract with observed activity.", "Balances are temporarily unavailable"} {
		if !strings.Contains(output, want) {
			t.Errorf("rendered contract page missing %q", want)
		}
	}
}

func TestSmartAccountShellLoadsBalancesWithoutRenderingUnavailableState(t *testing.T) {
	data := legacy.SmartAccountData{
		Name:                "Smart account",
		ContractID:          "CWALLET",
		BalanceFragmentHref: "/fragments/smart-account/CWALLET/balances?network=testnet&surface=v2",
	}
	var html strings.Builder
	if err := SmartAccount(data, "testnet").Render(context.Background(), &html); err != nil {
		t.Fatalf("render SmartAccount: %v", err)
	}
	output := html.String()
	for _, want := range []string{
		`hx-get="/fragments/smart-account/CWALLET/balances?network=testnet&amp;surface=v2"`,
		`hx-trigger="load"`,
		"Loading current balances",
		`id="px-smart-balance-hero"`,
		"Balance data",
	} {
		if !strings.Contains(output, want) {
			t.Errorf("rendered smart-account shell missing %q", want)
		}
	}
	if strings.Contains(output, "Balances are temporarily unavailable") {
		t.Error("loading shell rendered a failure state before the fragment completed")
	}
}

func TestContractShellLoadsBalancesWithoutBlockingDetail(t *testing.T) {
	data := legacy.ContractDetailData{
		Name:                "Market contract",
		Address:             "CCONTRACT",
		Narrative:           "A market contract with observed activity.",
		BalanceFragmentHref: "/fragments/contract/CCONTRACT/balances?network=testnet&surface=v2",
	}
	var html strings.Builder
	if err := ContractDetail(data, "testnet").Render(context.Background(), &html); err != nil {
		t.Fatalf("render ContractDetail: %v", err)
	}
	output := html.String()
	for _, want := range []string{"Market contract", "A market contract with observed activity.", `hx-trigger="load"`, "Loading current balances"} {
		if !strings.Contains(output, want) {
			t.Errorf("rendered contract shell missing %q", want)
		}
	}
	if strings.Contains(output, "Balances are temporarily unavailable") {
		t.Error("loading shell rendered a failure state before the fragment completed")
	}
}
