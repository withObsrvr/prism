package handlers

import (
	"fmt"
	"net/http"

	"github.com/withObsrvr/prism/internal/templates/pages"
)

func (h *Handlers) EventsFirehose(w http.ResponseWriter, r *http.Request) {
	data := pages.EventsFirehoseData{
		MatchedEvents: "2,847",
		LedgerStart:   "5,104,892",
		LedgerEnd:     "5,104,938",
		EventsPerSec:  "48",
		Filters: []pages.EventFilter{
			{Label: "contract:Soroswap", Color: "violet"},
			{Label: "type:transfer,swap", Color: "cyan"},
		},
		Events: []pages.FirehoseEvent{
			{Time: "0.4s", Type: "transfer", TypeColor: "transfer", ContractName: "USDC Token", ContractAddr: "CCW6...7YMK", TopicsHTML: `<div class="flex items-center gap-1.5 flex-wrap"><span class="rounded bg-gray-100 px-1.5 py-0.5 font-mono text-[10px] text-gray-600">from:<span class="text-violet-600">GABC...7X</span></span><span class="rounded bg-emerald-50 px-1.5 py-0.5 font-mono text-[10px] font-semibold text-emerald-700">2,500.00 USDC</span></div>`, Ledger: "5,104,938", TxShort: "8f2a...1b", TxHash: "8f2a1b4c", IsNew: true},
			{Time: "0.4s", Type: "swap", TypeColor: "contract", ContractName: "Soroswap Router", ContractAddr: "CAXY...Z10P", TopicsHTML: `<div class="flex items-center gap-1.5 flex-wrap"><span class="rounded bg-gray-100 px-1.5 py-0.5 font-mono text-[10px] text-gray-600">pair:XLM/USDC</span><span class="rounded bg-red-50 px-1.5 py-0.5 font-mono text-[10px] text-red-600">−12,400 XLM</span><span class="rounded bg-emerald-50 px-1.5 py-0.5 font-mono text-[10px] font-semibold text-emerald-700">+1,202.80 USDC</span></div>`, Ledger: "5,104,938", TxShort: "c4e9...3d", TxHash: "c4e93d7f", IsNew: true},
			{Time: "2.1s", Type: "transfer", TypeColor: "transfer", ContractName: "USDC Token", ContractAddr: "CCW6...7YMK", TopicsHTML: `<div class="flex items-center gap-1.5 flex-wrap"><span class="rounded bg-gray-100 px-1.5 py-0.5 font-mono text-[10px] text-gray-600">from:<span class="text-violet-600">GKLM...1V</span></span><span class="rounded bg-emerald-50 px-1.5 py-0.5 font-mono text-[10px] font-semibold text-emerald-700">500.00 USDC</span></div>`, Ledger: "5,104,937", TxShort: "a1b2...8e", TxHash: "a1b28e5a"},
			{Time: "2.1s", Type: "mint", TypeColor: "mint", ContractName: "BLND Token", ContractAddr: "CBLND...E84K", TopicsHTML: `<div class="flex items-center gap-1.5 flex-wrap"><span class="rounded bg-gray-100 px-1.5 py-0.5 font-mono text-[10px] text-gray-600">to:<span class="text-violet-600">GDEF...9R</span></span><span class="rounded bg-emerald-50 px-1.5 py-0.5 font-mono text-[10px] font-semibold text-emerald-700">+45,000 BLND</span></div>`, Ledger: "5,104,937", TxShort: "f7d2...4c", TxHash: "f7d24c9b"},
			{Time: "7.3s", Type: "swap", TypeColor: "contract", ContractName: "Soroswap Router", ContractAddr: "CAXY...Z10P", TopicsHTML: `<div class="flex items-center gap-1.5 flex-wrap"><span class="rounded bg-gray-100 px-1.5 py-0.5 font-mono text-[10px] text-gray-600">pair:AQUA/XLM</span><span class="rounded bg-red-50 px-1.5 py-0.5 font-mono text-[10px] text-red-600">−500,000 AQUA</span><span class="rounded bg-emerald-50 px-1.5 py-0.5 font-mono text-[10px] font-semibold text-emerald-700">+6,200 XLM</span></div>`, Ledger: "5,104,936", TxShort: "9e8f...2a", TxHash: "9e8f2a1b"},
			{Time: "7.3s", Type: "approve", TypeColor: "approve", ContractName: "USDC Token", ContractAddr: "CCW6...7YMK", TopicsHTML: `<div class="flex items-center gap-1.5 flex-wrap"><span class="rounded bg-gray-100 px-1.5 py-0.5 font-mono text-[10px] text-gray-600">spender:<span class="text-violet-600">CAXY...Z10P</span></span><span class="rounded bg-gray-100 px-1.5 py-0.5 font-mono text-[10px] text-gray-600">amount:∞</span></div>`, Ledger: "5,104,936", TxShort: "d3c2...7f", TxHash: "d3c27f1a"},
		},
	}
	pages.EventsFirehose(data).Render(r.Context(), w)
}

func (h *Handlers) StateRentTracker(w http.ResponseWriter, r *http.Request) {
	data := pages.RentTrackerData{
		ContractName:      "Soroswap Router",
		ContractAddr:      "CAXY...Z10P",
		CurrentLedger:     "5,104,892",
		TotalEntries:      "1,847",
		EstMonthlyRent:    "42,100",
		RentUSD:           "$4,083",
		HealthyCount:      "1,692",
		AtRiskCount:       "143",
		CriticalCount:     "12",
		InstanceEntries:   "1",
		InstanceSize:      "2.4 KB",
		InstanceMinTTL:    "42d",
		PersistentEntries: "1,254",
		PersistentSize:    "847 KB",
		PersistentMinTTL:  "3d",
		TemporaryEntries:  "592",
		TemporarySize:     "124 KB",
		TemporaryMinTTL:   "12d",
		BumpCost:          "8,420",
		BumpCostUSD:       "$816.74",
		StorageEntries: []pages.RentStorageEntry{
			{Type: "Persistent", TypeColor: "cyan", Key: "Balance:GDKX...8R42", KeyDesc: "User LP share balance", Size: "128 B", TTLDays: "3 days", TTLLedgers: "43K", HealthPct: "4%", HealthColor: "red", RentPerMonth: "18 XLM"},
			{Type: "Persistent", TypeColor: "cyan", Key: "Balance:GHIJ...2M56", KeyDesc: "User LP share balance", Size: "128 B", TTLDays: "4 days", TTLLedgers: "57K", HealthPct: "5%", HealthColor: "red", RentPerMonth: "18 XLM"},
			{Type: "Persistent", TypeColor: "cyan", Key: "Allowance:GABC...7X92", KeyDesc: "Token allowance", Size: "96 B", TTLDays: "6 days", TTLLedgers: "86K", HealthPct: "8%", HealthColor: "red", RentPerMonth: "14 XLM"},
			{Type: "Temporary", TypeColor: "amber", Key: "SwapRoute:0x8f2a", KeyDesc: "Cached swap route", Size: "256 B", TTLDays: "12 days", TTLLedgers: "172K", HealthPct: "16%", HealthColor: "amber", RentPerMonth: "22 XLM"},
			{Type: "Persistent", TypeColor: "cyan", Key: "ReserveA:XLM/USDC", KeyDesc: "Pool reserve balance", Size: "128 B", TTLDays: "38 days", TTLLedgers: "547K", HealthPct: "52%", HealthColor: "emerald", RentPerMonth: "18 XLM"},
			{Type: "Instance", TypeColor: "blue", Key: "Admin", KeyDesc: "Contract admin config", Size: "64 B", TTLDays: "42 days", TTLLedgers: "604K", HealthPct: "58%", HealthColor: "emerald", RentPerMonth: "8 XLM"},
		},
	}
	pages.RentTracker(data).Render(r.Context(), w)
}

func (h *Handlers) LiveFeed(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, `<div class="live-feed">Latest transactions...</div>`)
}
