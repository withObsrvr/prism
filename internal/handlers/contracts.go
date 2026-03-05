package handlers

import (
	"fmt"
	"net/http"

	"github.com/withObsrvr/prism/internal/templates/pages"
)

func (h *Handlers) ContractList(w http.ResponseWriter, r *http.Request) {
	data := pages.ContractListData{
		Contracts: []pages.ContractListItem{
			{Rank: 1, Name: "Soroswap Router", Address: "CAXY...Z10P", Tag: "DEX", TagColor: "violet", Invocations: "284,102", Change: "+12.4%", IsPositive: true},
			{Rank: 2, Name: "Blend Protocol", Address: "CBLND...P2R8", Tag: "Lending", TagColor: "emerald", Invocations: "142,847", Change: "+8.2%", IsPositive: true},
			{Rank: 3, Name: "USDC Token", Address: "CCW6...7YMK", Tag: "Token", TagColor: "blue", Invocations: "98,204", Change: "+3.1%", IsPositive: true},
			{Rank: 4, Name: "Aquarius AMM", Address: "CAQUA...8K32", Tag: "AMM", TagColor: "cyan", Invocations: "67,421", Change: "-2.4%", IsPositive: false},
			{Rank: 5, Name: "Phoenix DEX", Address: "CPHNX...R4M2", Tag: "DEX", TagColor: "violet", Invocations: "45,892", Change: "+22.1%", IsPositive: true},
			{Rank: 6, Name: "FxDAO Stablecoin", Address: "CFXD...Q7K1", Tag: "Stablecoin", TagColor: "amber", Invocations: "28,104", Change: "+5.7%", IsPositive: true},
			{Rank: 7, Name: "Stellar Geometries", Address: "CNFT...G7K2", Tag: "NFT", TagColor: "rose", Invocations: "12,847", Change: "+142%", IsPositive: true},
			{Rank: 8, Name: "Reflector Oracle", Address: "CREFL...M8P4", Tag: "Oracle", TagColor: "gray", Invocations: "8,421", Change: "-0.8%", IsPositive: false},
		},
	}
	pages.ContractList(data).Render(r.Context(), w)
}

func (h *Handlers) ContractDetail(w http.ResponseWriter, r *http.Request) {
	data := pages.ContractDetailData{
		Name:             "Soroswap Router",
		Address:          "CAXY7K2M8P4Q9R1S2T3U4V5W6X7Y8Z10PABCDEFGHIJKLMNOPQRSTUVWX",
		IsVerified:       true,
		Tags:             []string{"DEX", "AMM"},
		DeployedAt:       "Jan 15, 2024",
		DeployLedger:     "4,201,847",
		TotalInvocations: "284,102",
		StorageEntries:   "1,847",
		StateSize:        "2.4 MB",
		MonthlyRent:      "42,100 XLM",
		LastInvoked:      "2 min ago",
		Creator:          "GABC...7X92",
		CreatorHref:      "/account/GABC7DEF8GHI9JKL0MNO1PQR2STU3VWX4YZ567890",
		WASMSize:         "142 KB",
		FunctionsCount:   "12",
		IsSourceVerified: true,
		ActivityPoints:   []float64{4200, 4800, 5100, 4900, 5400, 6200, 5800, 6500, 7200, 6800, 7500, 8200, 7800, 8500, 7200, 6800, 7500, 8200, 8800, 9200, 8500, 7800, 7200, 6800, 7500, 8200, 8800, 9200, 9500, 8800},
		AvgInvocations:   "9,470",
		PeakInvocations:  "14,200",
		SuccessRate:      "99.2%",
		ExportedFunctions: []string{"swap", "add_liquidity", "remove_liquidity", "get_reserves", "get_pair", "initialize", "set_admin", "get_admin", "deposit", "withdraw", "get_amounts_out", "get_amounts_in"},
		Functions: []pages.ContractFunction{
			{Name: "swap", Calls24h: "4,847", Calls7d: "32,104", Calls30d: "142,847", SuccessRate: "99.4%", AvgCPU: "24.2M insn", LastCalled: "2 min ago"},
			{Name: "add_liquidity", Calls24h: "1,201", Calls7d: "8,402", Calls30d: "48,201", SuccessRate: "98.8%", AvgCPU: "18.4M insn", LastCalled: "8 min ago"},
			{Name: "remove_liquidity", Calls24h: "847", Calls7d: "5,921", Calls30d: "32,104", SuccessRate: "99.1%", AvgCPU: "16.8M insn", LastCalled: "22 min ago"},
			{Name: "get_reserves", Calls24h: "2,421", Calls7d: "12,847", Calls30d: "28,421", SuccessRate: "100%", AvgCPU: "2.1M insn", LastCalled: "30s ago"},
			{Name: "initialize", Calls24h: "0", Calls7d: "0", Calls30d: "1", SuccessRate: "100%", AvgCPU: "42.8M insn", LastCalled: "Jan 15, 2024"},
		},
		Invocations: []pages.ContractInvocation{
			{TxHash: "8f2a1b4c5d6e7f8a", ShortHash: "8f2a...1b4c", Function: "swap", Caller: "GABC...7X92", Status: "Success", StatusColor: "emerald", Summary: "Swapped 5,000 XLM → 485 USDC", CPUUsed: "24.2M", Age: "2 min ago"},
			{TxHash: "c4e93d7f2a1b8e5a", ShortHash: "c4e9...3d7f", Function: "add_liquidity", Caller: "GDEF...9R23", Status: "Success", StatusColor: "emerald", Summary: "Added 10,000 XLM + 970 USDC", CPUUsed: "18.4M", Age: "8 min ago"},
			{TxHash: "a1b28e5a9f7d2c4e", ShortHash: "a1b2...8e5a", Function: "swap", Caller: "GHIJ...2M56", Status: "Success", StatusColor: "emerald", Summary: "Swapped 12,400 XLM → 1,202 USDC", CPUUsed: "24.8M", Age: "15 min ago"},
			{TxHash: "f7d24c9b3a8e1f2d", ShortHash: "f7d2...4c9b", Function: "remove_liquidity", Caller: "GKLM...1V48", Status: "Success", StatusColor: "emerald", Summary: "Removed 5,000 XLM + 485 USDC", CPUUsed: "16.2M", Age: "22 min ago"},
		},
		StorageItems: []pages.ContractStorageItem{
			{Key: "Admin", Type: "Instance", TypeColor: "blue", Size: "64 B", TTL: "42 days"},
			{Key: "ReserveA:XLM/USDC", Type: "Persistent", TypeColor: "cyan", Size: "128 B", TTL: "38 days"},
			{Key: "ReserveB:XLM/USDC", Type: "Persistent", TypeColor: "cyan", Size: "128 B", TTL: "38 days"},
			{Key: "TotalShares:XLM/USDC", Type: "Persistent", TypeColor: "cyan", Size: "64 B", TTL: "38 days"},
			{Key: "SwapRoute:0x8f2a", Type: "Temporary", TypeColor: "amber", Size: "256 B", TTL: "12 days"},
		},
		CodePreview: `pub fn swap(\n    env: Env,\n    token_in: Address,\n    token_out: Address,\n    amount_in: i128,\n    amount_out_min: i128,\n) -> i128 {\n    let pair = get_pair(&env, &token_in, &token_out);\n    let (reserve_in, reserve_out) = pair.get_reserves();\n    let amount_out = get_amount_out(amount_in, reserve_in, reserve_out);\n    require!(amount_out >= amount_out_min, "insufficient output");\n    // ... transfer logic\n    amount_out\n}`,
		WASMHash: "a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2e3f4a5b6c7d8e9f0",
	}
	pages.ContractDetail(data).Render(r.Context(), w)
}

func (h *Handlers) ContractEvents(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if isHTMX(r) {
		fmt.Fprintf(w, `<div id="tab-content">Events for %s</div>`, id)
		return
	}
	fmt.Fprintf(w, "Contract Events: %s", id)
}
