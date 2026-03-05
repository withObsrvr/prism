package handlers

import (
	"net/http"

	"github.com/withObsrvr/prism/internal/templates/pages"
)

func (h *Handlers) AccountPortfolio(w http.ResponseWriter, r *http.Request) {
	data := pages.AccountData{
		Address:      "GABC7DEF8GHI9JKL0MNO1PQR2STU3VWX4YZ567890ABCDEFGHIJKLMNOP",
		ShortAddress: "GABC...MNOP",
		TotalValue:   "$58,247",
		TotalCents:   ".82",
		XLMBalance:   "124,500.00 XLM",
		Trustlines:   "12",
		ActiveOffers: "1",
		Subentries:   "14",
		CreatedAt:    "Jan 15, 2024",
		HomeDomain:   "stellar.org",
		IsFunded:     true,
		SignerCount:  "1",
		Balances: []pages.AccountBalance{
			{Code: "XLM", Name: "Stellar Lumens", BgColor: "bg-gray-900", Type: "Native", TypeColor: "gray", Balance: "124,500.00", ValueUSD: "$12,078.50"},
			{Code: "USDC", Name: "USD Coin", Issuer: "Centre", BgColor: "bg-blue-600", Type: "Classic", TypeColor: "gray", Balance: "25,000.00", ValueUSD: "$25,000.00"},
			{Code: "yUSDC", Name: "Blend USDC", Issuer: "Blend", BgColor: "bg-emerald-600", Type: "SEP-41", TypeColor: "violet", Balance: "18,200.00", ValueUSD: "$18,247.32"},
			{Code: "AQUA", Name: "Aquarius", Issuer: "Aquarius", BgColor: "bg-cyan-600", Type: "Classic", TypeColor: "gray", Balance: "500,000.00", ValueUSD: "$2,450.00"},
			{Code: "BLND", Name: "Blend Token", Issuer: "Blend", BgColor: "bg-violet-600", Type: "SEP-41", TypeColor: "violet", Balance: "12,000.00", ValueUSD: "$471.00"},
		},
		Activities: []pages.AccountActivity{
			{IconBg: "bg-emerald-50", IconColor: "text-emerald-600", Summary: "Swap: 12,400 XLM → 1,202.80 USDC via Soroswap", Badge: "Swap", BadgeColor: "violet", TxHash: "8f2a1b4c5d6e7f8a", ShortHash: "8f2a...1b4c", Time: "2 min ago", DateGroup: "Today"},
			{IconBg: "bg-violet-50", IconColor: "text-violet-600", Summary: "Blend Protocol: supply(USDC, 5,000)", Badge: "Contract", BadgeColor: "violet", TxHash: "c4e93d7f2a1b8e5a", ShortHash: "c4e9...3d7f", Time: "1 hour ago", DateGroup: "Today"},
			{IconBg: "bg-blue-50", IconColor: "text-blue-600", Summary: "Sent 500 USDC to GDEF...9R23", Badge: "Payment", BadgeColor: "blue", TxHash: "a1b28e5a9f7d2c4e", ShortHash: "a1b2...8e5a", Time: "18 hours ago", DateGroup: "Yesterday"},
			{IconBg: "bg-gray-50", IconColor: "text-gray-600", Summary: "Added trustline: BLND (Blend Token)", Badge: "Trustline", BadgeColor: "gray", TxHash: "f7d24c9b3a8e1f2d", ShortHash: "f7d2...4c9b", Time: "22 hours ago", DateGroup: "Yesterday"},
		},
		Contracts: []pages.AccountContract{
			{Name: "Soroswap Router", Badge: "DEX", BadgeColor: "violet", Address: "CAXY...Z10P", TopFn: "swap", Calls: "142", Fees: "2,847 XLM", LastCall: "2 min ago"},
			{Name: "Blend Protocol", Badge: "Lending", BadgeColor: "emerald", Address: "CBLND...P2R8", TopFn: "supply", Calls: "38", Fees: "412 XLM", LastCall: "1 hour ago"},
		},
		Offers: []pages.AccountOffer{
			{Side: "Buy", SideColor: "emerald", Pair: "USDC/XLM", Price: "0.097", PriceUnit: "XLM", Amount: "10,000 USDC", OfferID: "892341"},
		},
		Signers: []pages.AccountSigner{
			{Address: "GABC...MNOP", Type: "ed25519", IsSelf: true, Weight: "1"},
		},
		Thresholds: []pages.AccountThreshold{
			{Label: "Low", Value: "1", Pct: "100%", Color: "emerald"},
			{Label: "Medium", Value: "1", Pct: "100%", Color: "emerald"},
			{Label: "High", Value: "1", Pct: "100%", Color: "emerald"},
		},
	}
	pages.AccountPortfolio(data).Render(r.Context(), w)
}

func (h *Handlers) SmartAccountDashboard(w http.ResponseWriter, r *http.Request) {
	data := pages.SmartAccountData{
		Name:         "Treasury Multisig",
		ContractID:   "CDLZ...Q8M4",
		TotalBalance: "$87,204",
		BalanceCents: ".51",
		Signers: []pages.SmartSigner{
			{Name: "Owner Key", Role: "Admin", RoleColor: "amber", Address: "GBXC...4K71", KeyType: "Ed25519", Weight: "10", IconSVG: "key", IconBg: "bg-amber-50 text-amber-600 ring-1 ring-amber-100"},
			{Name: "Operations Signer", Role: "Signer", RoleColor: "blue", Address: "GDEF...9R23", KeyType: "Ed25519", Weight: "10", IconSVG: "user", IconBg: "bg-blue-50 text-blue-600 ring-1 ring-blue-100"},
			{Name: "Recovery Signer", Role: "Recovery", RoleColor: "emerald", Address: "GHIJ...2M56", KeyType: "Ed25519", Weight: "10", IconSVG: "recovery", IconBg: "bg-emerald-50 text-emerald-600 ring-1 ring-emerald-100"},
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
			{Action: "Session key created", Detail: "Today, 11:02 AM · by Owner Key", Time: "2h ago", Status: "OK", StatusColor: "emerald", IconSVG: "check", IconBg: "bg-emerald-50 text-emerald-600"},
			{Action: "Blend Protocol added to allowlist", Detail: "Yesterday, 3:18 PM · 2-of-3 approval", Time: "1d ago", Status: "OK", StatusColor: "blue", IconSVG: "plus", IconBg: "bg-blue-50 text-blue-600"},
			{Action: "Session key expired", Detail: "Oct 23, 2026 · auto-revoked", Time: "3d ago", Status: "Expired", StatusColor: "red", IconSVG: "x", IconBg: "bg-red-50 text-red-600"},
			{Action: "Threshold updated to 2-of-3", Detail: "Oct 2, 2026 · at deployment", Time: "5d ago", Status: "OK", StatusColor: "amber", IconSVG: "key", IconBg: "bg-amber-50 text-amber-600"},
		},
	}
	pages.SmartAccount(data).Render(r.Context(), w)
}
