package handlers

import (
	"strings"
	"testing"

	"github.com/withObsrvr/prism/internal/gateway"
)

func TestAddressBalancePortfolioPreservesDuplicateSymbols(t *testing.T) {
	decimals := 7
	ledger := int64(3689307)
	response := &gateway.AddressBalancesResponse{
		Address:       "CCONTRACT",
		TotalBalances: 3,
		Sources:       []string{"contract_storage_state"},
		Partial:       true,
		Balances: []gateway.AddressBalance{
			{AssetType: "credit_alphanum4", AssetCode: "USDC", AssetIssuer: "GISSUER2", Balance: "615.1000000", Decimals: &decimals, BalanceSource: "contract_storage_state", LastUpdatedLedger: &ledger},
			{AssetType: "native", AssetCode: "XLM", Balance: "19971.7336815", Decimals: &decimals, BalanceSource: "contract_storage_state", LastUpdatedLedger: &ledger},
			{AssetType: "credit_alphanum4", AssetCode: "USDC", AssetIssuer: "GISSUER1", Balance: "975.0000000", Decimals: &decimals, BalanceSource: "contract_storage_state", LastUpdatedLedger: &ledger},
		},
	}

	portfolio := addressBalancePortfolio(response, "testnet")
	if !portfolio.Available || !portfolio.Partial || portfolio.NativeBalance != "19,971.7336815" {
		t.Fatalf("portfolio state = %+v", portfolio)
	}
	if len(portfolio.Items) != 3 || portfolio.Items[0].AssetCode != "XLM" {
		t.Fatalf("native balance was not sorted first: %+v", portfolio.Items)
	}
	if portfolio.Items[1].AssetIssuer != "GISSUER1" || portfolio.Items[2].AssetIssuer != "GISSUER2" {
		t.Fatalf("duplicate symbols lost identity or stable ordering: %+v", portfolio.Items)
	}
	if portfolio.Items[1].IssuerHref != "/v2/account/GISSUER1?network=testnet" {
		t.Fatalf("issuer href = %q", portfolio.Items[1].IssuerHref)
	}
}

func TestBalanceSourceLabelTrimsUnknownSource(t *testing.T) {
	if got := balanceSourceLabel("  custom_balance_source  "); got != "custom balance source" {
		t.Fatalf("balanceSourceLabel = %q, want trimmed label", got)
	}
}

func TestCurrentBalanceFragmentHrefPreservesPageContext(t *testing.T) {
	got := currentBalanceFragmentHref("smart-account", "CWALLET", "testnet", "v2", true)
	for _, want := range []string{
		"/fragments/smart-account/CWALLET/balances?",
		"mock=true",
		"network=testnet",
		"surface=v2",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("fragment href %q missing %q", got, want)
		}
	}
}

func TestFormatBalanceAmountUsesDecimalStrings(t *testing.T) {
	decimals := 7
	tests := map[string]string{
		"0009995.0000000":  "9,995",
		"19971.7336815":    "19,971.7336815",
		"-1000000.5000000": "-1,000,000.5",
		"0.0000001":        "0.0000001",
	}
	for input, want := range tests {
		if got := formatBalanceAmount(input, &decimals); got != want {
			t.Errorf("formatBalanceAmount(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestSmartWalletBalancePortfolioUsesMaterializedStatus(t *testing.T) {
	portfolio := smartWalletBalancePortfolio(&gateway.SmartWalletBalancesResponse{
		ContractID: "CWALLET", NativeBalance: "500.0000000", NativeBalanceSource: "contract_storage_state", Count: 2, BalanceStatus: "materialized",
		Balances: []gateway.SmartWalletBalance{
			{AssetType: "native", AssetCode: "XLM", Balance: "500.0000000", BalanceSource: "contract_storage_state"},
			{AssetType: "credit_alphanum4", AssetCode: "EURC", AssetIssuer: "GISSUER", Balance: "10.0000000", BalanceSource: "contract_storage_state"},
		},
	}, "testnet")

	if portfolio.Status != "materialized" || portfolio.NativeBalance != "500" || portfolio.DisplayCount() != 2 {
		t.Fatalf("portfolio = %+v", portfolio)
	}
	if portfolio.HeroLabel() != "Native balance" || portfolio.HeroValue() != "500" || portfolio.HeroUnit() != "XLM" {
		t.Fatalf("hero = %q %q %q", portfolio.HeroLabel(), portfolio.HeroValue(), portfolio.HeroUnit())
	}
}
