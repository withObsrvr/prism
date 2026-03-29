package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/withObsrvr/prism/internal/gateway"
	"github.com/withObsrvr/prism/internal/templates/pages"
)

func (h *Handlers) AccountPortfolio(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	network := networkFromRequest(r)

	var data pages.AccountData
	if h.useLiveData(r) {
		if live, err := h.buildAccountData(r, network, id); err == nil {
			data = live
		} else {
			h.Logger.Warn("live account shell data failed, falling back to mock", "error", err)
		}
	}
	if data.Address == "" {
		data = mockAccountData()
	}

	pages.AccountPortfolioV2(data).Render(r.Context(), w)
}

func (h *Handlers) AccountPortfolioV1(w http.ResponseWriter, r *http.Request) {
	data := mockAccountData()
	pages.AccountPortfolio(data).Render(r.Context(), w)
}

func (h *Handlers) buildAccountData(r *http.Request, network, accountID string) (pages.AccountData, error) {
	ctx := r.Context()

	overview, err := h.Gateway.GetAccountOverview(ctx, network, accountID)
	if err != nil {
		return pages.AccountData{}, fmt.Errorf("fetching account overview: %w", err)
	}

	acct := overview.Account

	// Build balances from the overview (try dedicated endpoint too).
	var balances []pages.AccountBalance
	balResp, balErr := h.Gateway.GetAccountBalances(ctx, network, accountID)
	if balErr == nil && balResp != nil {
		for _, b := range balResp.Balances {
			assetType := "Classic"
			typeColor := "gray"
			code := b.AssetCode
			if b.AssetType == "native" {
				code = "XLM"
				assetType = "Native"
			}
			balances = append(balances, pages.AccountBalance{
				Code:      code,
				Name:      code,
				BgColor:   "bg-gray-600",
				Type:      assetType,
				TypeColor: typeColor,
				Balance:   b.Balance,
				ValueUSD:  "—",
			})
		}
	}

	// Build activities from recent operations.
	var activities []pages.AccountActivity
	for _, op := range overview.RecentOperations {
		badge := op.TypeName
		badgeColor := "gray"
		if op.IsSorobanOp {
			badge = "Contract"
			badgeColor = "violet"
		} else if op.IsPaymentOp {
			badge = "Payment"
			badgeColor = "blue"
		}

		age := "—"
		if t, err := time.Parse(time.RFC3339, op.LedgerClosedAt); err == nil {
			d := time.Since(t)
			if d.Hours() > 24 {
				age = fmt.Sprintf("%.0fd ago", d.Hours()/24)
			} else if d.Hours() > 1 {
				age = fmt.Sprintf("%.0fh ago", d.Hours())
			} else {
				age = fmt.Sprintf("%.0fm ago", d.Minutes())
			}
		}

		summary := fmt.Sprintf("%s %s", gateway.ShortAddress(op.SourceAccount), op.TypeName)
		if op.Amount != "" {
			summary += fmt.Sprintf(" %s", op.Amount)
		}

		activities = append(activities, pages.AccountActivity{
			Summary:    summary,
			Badge:      badge,
			BadgeColor: badgeColor,
			TxHash:     op.TransactionHash,
			ShortHash:  gateway.ShortHash(op.TransactionHash),
			Time:       age,
		})
	}

	// Build signers.
	var signers []pages.AccountSigner
	var thresholds []pages.AccountThreshold
	sigResp, sigErr := h.Gateway.GetAccountSigners(ctx, network, accountID)
	if sigErr == nil && sigResp != nil {
		for _, s := range sigResp.Signers {
			signers = append(signers, pages.AccountSigner{
				Address: gateway.ShortAddress(s.Key),
				Type:    s.Type,
				IsSelf:  s.Key == accountID,
				Weight:  fmt.Sprintf("%d", s.Weight),
			})
		}
		thresholds = []pages.AccountThreshold{
			{Label: "Low", Value: fmt.Sprintf("%d", sigResp.Thresholds.Low), Color: "emerald"},
			{Label: "Medium", Value: fmt.Sprintf("%d", sigResp.Thresholds.Medium), Color: "emerald"},
			{Label: "High", Value: fmt.Sprintf("%d", sigResp.Thresholds.High), Color: "emerald"},
		}
	}

	// Count trustlines as non-native balances.
	trustlineCount := 0
	for _, b := range balances {
		if b.Type != "Native" {
			trustlineCount++
		}
	}

	data := pages.AccountData{
		Address:      accountID,
		ShortAddress: gateway.ShortAddress(accountID),
		TotalValue:   "—",
		XLMBalance:   acct.Balance + " XLM",
		Trustlines:   fmt.Sprintf("%d", trustlineCount),
		ActiveOffers: "0",
		Subentries:   fmt.Sprintf("%d", acct.NumSubentries),
		IsFunded:     true,
		SignerCount:  fmt.Sprintf("%d", len(signers)),
		Balances:     balances,
		Activities:   activities,
		Signers:      signers,
		Thresholds:   thresholds,
	}

	// Prefer created_at from gateway; fall back to updated_at.
	createdAtStr := acct.CreatedAt
	if createdAtStr == "" {
		createdAtStr = acct.UpdatedAt
	}
	if createdAtStr != "" {
		if t, err := time.Parse(time.RFC3339, createdAtStr); err == nil {
			data.CreatedAt = t.Format("Jan 2, 2006")
		}
	}

	return data, nil
}

func (h *Handlers) SmartAccountDashboard(w http.ResponseWriter, r *http.Request) {
	// Smart account requires Soroban-specific contract introspection.
	// Always uses mock data.
	data := pages.SmartAccountData{
		Name:         "Treasury Multisig",
		ContractID:   "CDLZ...Q8M4",
		TotalBalance: "$87,204",
		BalanceCents: ".51",
		Signers: []pages.SmartSigner{
			{Name: "Owner Key", Role: "Admin", RoleColor: "amber", Address: "GBXC...4K71", KeyType: "Ed25519", Weight: "10", IconSVG: "key", IconBg: "bg-amber-50 dark:bg-amber-950/30 text-amber-600 dark:text-amber-400 ring-1 ring-amber-100 dark:ring-amber-800"},
			{Name: "Operations Signer", Role: "Signer", RoleColor: "blue", Address: "GDEF...9R23", KeyType: "Ed25519", Weight: "10", IconSVG: "user", IconBg: "bg-blue-50 dark:bg-blue-950/30 text-blue-600 dark:text-blue-400 ring-1 ring-blue-100 dark:ring-blue-800"},
			{Name: "Recovery Signer", Role: "Recovery", RoleColor: "emerald", Address: "GHIJ...2M56", KeyType: "Ed25519", Weight: "10", IconSVG: "recovery", IconBg: "bg-emerald-50 dark:bg-emerald-950/30 text-emerald-600 dark:text-emerald-400 ring-1 ring-emerald-100 dark:ring-emerald-800"},
		},
		Policies: []pages.SmartPolicy{
			{
				Name: "Daily Spending Limit", Description: "Caps single-signer withdrawals per 24-hour period", IsActive: true,
				Limit: "10,000", Spent: "2,400", Remaining: "7,600", SpentPct: "24%", ResetsIn: "14h 22m",
			},
			{
				Name: "Contract Allowlist", Description: "Only pre-approved contracts can be invoked", IsActive: true,
				Contracts: []pages.AllowedContract{
					{Initial: "S", InitialBg: "bg-[#9B6568]/10 text-[#9B6568]", Name: "Soroswap Router", Address: "CAXY...Z10P", Methods: "swap, add_liq"},
					{Initial: "B", InitialBg: "bg-emerald-50 text-emerald-600", Name: "Blend Protocol", Address: "CBLND...P2R8", Methods: "supply, borrow"},
					{Initial: "U", InitialBg: "bg-blue-50 text-blue-600", Name: "USDC Token", Address: "CCTP...W5A3", Methods: "transfer, approve"},
				},
			},
		},
		SessionKeys: []pages.SmartSessionKey{
			{Name: "DeFi Trading Session", Description: "Temporary key for Soroswap interactions", Key: "GSESS...K8P2", SpendLimit: "500", Used: "127", UsedPct: "25%", Expires: "2h 41m", Scope: "Soroswap Router only"},
		},
		LowThreshold:   "10",
		MedThreshold:   "20",
		HighThreshold:  "20",
		MasterWeight:   "10",
		RequiredWeight: "20",
		TotalWeight:    "30",
		MinSigners:     "2 of 3",
		Health: pages.ContractHealth{
			RentStatus:   "Healthy",
			TTLRemaining: "~58 days",
			WASMHash:     "0xa4f2...c8b1",
			OZVersion:    "v0.5.0",
			Deployed:     "Oct 2, 2026",
		},
		SecurityLog: []pages.SecurityEvent{
			{Action: "Session key created", Detail: "Today, 11:02 AM · by Owner Key", Time: "2h ago", Status: "OK", StatusColor: "emerald", IconSVG: "check", IconBg: "bg-emerald-50 dark:bg-emerald-950/30 text-emerald-600 dark:text-emerald-400"},
			{Action: "Blend Protocol added to allowlist", Detail: "Yesterday, 3:18 PM · 2-of-3 approval", Time: "1d ago", Status: "OK", StatusColor: "blue", IconSVG: "plus", IconBg: "bg-blue-50 dark:bg-blue-950/30 text-blue-600 dark:text-blue-400"},
			{Action: "Session key expired", Detail: "Oct 23, 2026 · auto-revoked", Time: "3d ago", Status: "Expired", StatusColor: "red", IconSVG: "x", IconBg: "bg-red-50 dark:bg-red-950/30 text-red-600 dark:text-red-400"},
			{Action: "Threshold updated to 2-of-3", Detail: "Oct 2, 2026 · at deployment", Time: "5d ago", Status: "OK", StatusColor: "amber", IconSVG: "key", IconBg: "bg-amber-50 dark:bg-amber-950/30 text-amber-600 dark:text-amber-400"},
		},
	}
	pages.SmartAccountV2(data).Render(r.Context(), w)
}
