package handlers

import (
	"context"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/withObsrvr/prism/internal/gateway"
	ui "github.com/withObsrvr/prism/internal/viewmodel"
)

const detailBalanceTimeout = 1500 * time.Millisecond

type balancePortfolioResult struct {
	portfolio ui.BalancePortfolio
	err       error
}

func (h *Handlers) startAddressBalanceLookup(ctx context.Context, network, address string) <-chan balancePortfolioResult {
	result := make(chan balancePortfolioResult, 1)
	go func() {
		lookupCtx, cancel := context.WithTimeout(ctx, detailBalanceTimeout)
		defer cancel()
		response, err := h.Gateway.GetAddressBalances(lookupCtx, network, address)
		if err != nil {
			result <- balancePortfolioResult{portfolio: unavailableBalancePortfolio(address), err: err}
			return
		}
		result <- balancePortfolioResult{portfolio: addressBalancePortfolio(response, network)}
	}()
	return result
}

func (h *Handlers) startSmartWalletBalanceLookup(ctx context.Context, network, contractID string) <-chan balancePortfolioResult {
	result := make(chan balancePortfolioResult, 1)
	go func() {
		lookupCtx, cancel := context.WithTimeout(ctx, detailBalanceTimeout)
		defer cancel()
		response, err := h.Gateway.GetSmartWalletBalances(lookupCtx, network, contractID)
		if err != nil {
			result <- balancePortfolioResult{portfolio: unavailableBalancePortfolio(contractID), err: err}
			return
		}
		result <- balancePortfolioResult{portfolio: smartWalletBalancePortfolio(response, network)}
	}()
	return result
}

func unavailableBalancePortfolio(ownerID string) ui.BalancePortfolio {
	return ui.BalancePortfolio{OwnerID: ownerID}
}

func addressBalancePortfolio(response *gateway.AddressBalancesResponse, network string) ui.BalancePortfolio {
	if response == nil {
		return ui.BalancePortfolio{}
	}
	portfolio := ui.BalancePortfolio{
		OwnerID:     response.Address,
		Count:       response.TotalBalances,
		Available:   true,
		Partial:     response.Partial,
		Status:      "materialized",
		SourceLabel: balanceSourcesLabel(response.Sources),
		Warnings:    append([]string(nil), response.Warnings...),
	}
	for _, balance := range response.Balances {
		code := balanceAssetCode(balance.AssetType, balance.AssetCode, balance.Symbol, balance.ContractID)
		item := ui.BalanceItem{
			AssetCode:         code,
			AssetType:         balance.AssetType,
			AssetIssuer:       balance.AssetIssuer,
			IssuerHref:        balanceEntityHref(balance.AssetIssuer, network),
			TokenContractID:   balance.ContractID,
			TokenHref:         balanceEntityHref(balance.ContractID, network),
			Balance:           formatBalanceAmount(balance.Balance, balance.Decimals),
			Decimals:          balance.Decimals,
			Source:            balance.BalanceSource,
			SourceLabel:       balanceSourceLabel(balance.BalanceSource),
			LastUpdatedLedger: balance.LastUpdatedLedger,
			LastUpdatedAt:     balance.LastUpdatedAt,
		}
		if isNativeBalance(item.AssetType, item.AssetCode) {
			portfolio.NativeBalance = item.Balance
		}
		portfolio.Items = append(portfolio.Items, item)
	}
	finalizeBalancePortfolio(&portfolio)
	return portfolio
}

func smartWalletBalancePortfolio(response *gateway.SmartWalletBalancesResponse, network string) ui.BalancePortfolio {
	if response == nil {
		return ui.BalancePortfolio{}
	}
	portfolio := ui.BalancePortfolio{
		OwnerID:       response.ContractID,
		NativeBalance: formatBalanceAmount(response.NativeBalance, nil),
		Count:         response.Count,
		Available:     true,
		Partial:       response.Partial,
		Status:        response.BalanceStatus,
		SourceLabel:   balanceSourceLabel(response.NativeBalanceSource),
	}
	for _, balance := range response.Balances {
		code := balanceAssetCode(balance.AssetType, balance.AssetCode, balance.Symbol, balance.TokenContractID)
		item := ui.BalanceItem{
			AssetCode:       code,
			AssetType:       balance.AssetType,
			AssetIssuer:     balance.AssetIssuer,
			IssuerHref:      balanceEntityHref(balance.AssetIssuer, network),
			TokenContractID: balance.TokenContractID,
			TokenHref:       balanceEntityHref(balance.TokenContractID, network),
			Balance:         formatBalanceAmount(balance.Balance, balance.Decimals),
			Decimals:        balance.Decimals,
			Source:          balance.BalanceSource,
			SourceLabel:     balanceSourceLabel(balance.BalanceSource),
		}
		if portfolio.NativeBalance == "" && isNativeBalance(item.AssetType, item.AssetCode) {
			portfolio.NativeBalance = item.Balance
		}
		portfolio.Items = append(portfolio.Items, item)
	}
	finalizeBalancePortfolio(&portfolio)
	return portfolio
}

func finalizeBalancePortfolio(portfolio *ui.BalancePortfolio) {
	if portfolio.Count < len(portfolio.Items) {
		portfolio.Count = len(portfolio.Items)
	}
	if portfolio.SourceLabel == "" {
		for _, item := range portfolio.Items {
			if item.SourceLabel != "" {
				portfolio.SourceLabel = item.SourceLabel
				break
			}
		}
	}
	if portfolio.Status == "" && len(portfolio.Items) > 0 {
		portfolio.Status = "materialized"
	}
	sort.SliceStable(portfolio.Items, func(i, j int) bool {
		left, right := portfolio.Items[i], portfolio.Items[j]
		leftNative := isNativeBalance(left.AssetType, left.AssetCode)
		rightNative := isNativeBalance(right.AssetType, right.AssetCode)
		if leftNative != rightNative {
			return leftNative
		}
		leftKey := strings.ToUpper(left.AssetCode) + "\x00" + left.AssetIssuer + "\x00" + left.TokenContractID
		rightKey := strings.ToUpper(right.AssetCode) + "\x00" + right.AssetIssuer + "\x00" + right.TokenContractID
		return leftKey < rightKey
	})
}

func formatBalanceAmount(value string, decimals *int) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	sign := ""
	if value[0] == '-' || value[0] == '+' {
		sign, value = value[:1], value[1:]
	}
	parts := strings.Split(value, ".")
	if len(parts) > 2 || value == "" {
		return sign + value
	}
	integer := strings.TrimLeft(parts[0], "0")
	if integer == "" {
		integer = "0"
	}
	for i := len(integer) - 3; i > 0; i -= 3 {
		integer = integer[:i] + "," + integer[i:]
	}
	fraction := ""
	if len(parts) == 2 {
		fraction = parts[1]
		if decimals != nil && *decimals >= 0 && len(fraction) > *decimals {
			overflow := fraction[*decimals:]
			if strings.Trim(overflow, "0") == "" {
				fraction = fraction[:*decimals]
			}
		}
		fraction = strings.TrimRight(fraction, "0")
	}
	if fraction == "" {
		return sign + integer
	}
	return sign + integer + "." + fraction
}

func balanceAssetCode(assetType, assetCode, symbol, contractID string) string {
	if isNativeBalance(assetType, assetCode) {
		return "XLM"
	}
	for _, candidate := range []string{assetCode, symbol} {
		if strings.TrimSpace(candidate) != "" {
			return strings.ToUpper(strings.TrimSpace(candidate))
		}
	}
	if contractID != "" {
		return gateway.ShortAddress(contractID)
	}
	return "Unknown asset"
}

func isNativeBalance(assetType, assetCode string) bool {
	return strings.EqualFold(assetType, "native") || strings.EqualFold(assetCode, "XLM")
}

func balanceEntityHref(id, network string) string {
	if id == "" {
		return ""
	}
	path := ""
	switch id[0] {
	case 'G':
		path = "/v2/account/" + id
	case 'C':
		path = "/v2/contract/" + id
	default:
		return ""
	}
	if network != "" {
		path += "?network=" + url.QueryEscape(network)
	}
	return path
}

func balanceSourceLabel(source string) string {
	switch strings.ToLower(strings.TrimSpace(source)) {
	case "contract_storage_state":
		return "Current contract storage"
	case "stellar_native":
		return "Current ledger state"
	case "":
		return ""
	default:
		return strings.ReplaceAll(source, "_", " ")
	}
}

func balanceSourcesLabel(sources []string) string {
	for _, source := range sources {
		if label := balanceSourceLabel(source); label != "" {
			return label
		}
	}
	return "Current materialized balances"
}
