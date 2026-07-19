package viewmodel

import (
	"strconv"
	"strings"
)

// BalancePortfolio is the presentation contract shared by account and contract pages.
// Balance strings remain decimal strings so the UI never loses ledger precision.
type BalancePortfolio struct {
	OwnerID       string
	NativeBalance string
	Items         []BalanceItem
	Count         int
	Available     bool
	Partial       bool
	Status        string
	SourceLabel   string
	Warnings      []string
}

type BalanceItem struct {
	AssetCode         string
	AssetType         string
	AssetIssuer       string
	IssuerHref        string
	TokenContractID   string
	TokenHref         string
	Balance           string
	Decimals          *int
	Source            string
	SourceLabel       string
	LastUpdatedLedger *int64
	LastUpdatedAt     string
}

func (p BalancePortfolio) DisplayCount() int {
	if p.Count > 0 {
		return p.Count
	}
	return len(p.Items)
}

func (p BalancePortfolio) HeroLabel() string {
	if p.NotMaterialized() {
		return "Balance data"
	}
	if p.NativeBalance != "" {
		return "Native balance"
	}
	if p.Available {
		return "Positive holdings"
	}
	return "Balance data"
}

func (p BalancePortfolio) HeroValue() string {
	if p.NotMaterialized() {
		return "Pending"
	}
	if p.NativeBalance != "" {
		return p.NativeBalance
	}
	if p.Available {
		return strconv.Itoa(p.DisplayCount())
	}
	return "Unavailable"
}

func (p BalancePortfolio) HeroUnit() string {
	if p.NotMaterialized() {
		return ""
	}
	if p.NativeBalance != "" {
		return "XLM"
	}
	if p.Available {
		return "assets"
	}
	return ""
}

func (p BalancePortfolio) NotMaterialized() bool {
	return strings.EqualFold(strings.TrimSpace(p.Status), "not_materialized")
}

func (p BalancePortfolio) HasMore(limit int) bool {
	return limit > 0 && len(p.Items) > limit
}

func (p BalancePortfolio) Remaining(limit int) int {
	if !p.HasMore(limit) {
		return 0
	}
	return len(p.Items) - limit
}
