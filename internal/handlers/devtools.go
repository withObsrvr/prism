package handlers

import (
	"fmt"
	"net/http"

	"github.com/withObsrvr/prism/internal/events"
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
		Events: buildFirehoseEvents(),
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

// buildFirehoseEvents creates mock firehose events using the event decoder
// to generate human-readable TopicsHTML from structured event data.
func buildFirehoseEvents() []pages.FirehoseEvent {
	type mockEvent struct {
		pages.FirehoseEvent
		raw events.RawEvent
	}

	mocks := []mockEvent{
		{
			FirehoseEvent: pages.FirehoseEvent{Time: "0.4s", Type: "transfer", TypeColor: "transfer", ContractName: "USDC Token", ContractAddr: "CCW6...7YMK", Ledger: "5,104,938", TxShort: "8f2a...1b", TxHash: "8f2a1b4c", IsNew: true, AlertBadge: "WHALE", AlertColor: "amber", DetailJSON: "{\n  \"type\": \"transfer\",\n  \"from\": \"GABC...7X92\",\n  \"to\": \"GDEF...9R23\",\n  \"amount\": \"2500.0000000\",\n  \"asset\": \"USDC:GA5Z...CCW6\"\n}", DetailMeta: "Large USDC transfer (>$1,000). Source account is a known DeFi power user with 142 contract interactions this month."},
			raw: events.RawEvent{Type: "transfer", From: "GABC...7X", To: "GDEF...9R", Amount: "2,500.00", Asset: "USDC"},
		},
		{
			FirehoseEvent: pages.FirehoseEvent{Time: "0.4s", Type: "swap", TypeColor: "contract", ContractName: "Soroswap Router", ContractAddr: "CAXY...Z10P", Ledger: "5,104,938", TxShort: "c4e9...3d", TxHash: "c4e93d7f", IsNew: true, DetailJSON: "{\n  \"type\": \"swap\",\n  \"pair\": \"XLM/USDC\",\n  \"amount_in\": \"12400.0000000\",\n  \"amount_out\": \"1202.8000000\",\n  \"rate\": \"0.097\"\n}", DetailMeta: "Soroswap Router swap via XLM/USDC direct pool. Effective rate: 0.097 USDC/XLM."},
			raw: events.RawEvent{Type: "swap", From: "GNOP...3W", PairIn: "12,400 XLM", PairOut: "1,202.80 USDC", Router: "Soroswap"},
		},
		{
			FirehoseEvent: pages.FirehoseEvent{Time: "2.1s", Type: "transfer", TypeColor: "transfer", ContractName: "USDC Token", ContractAddr: "CCW6...7YMK", Ledger: "5,104,937", TxShort: "a1b2...8e", TxHash: "a1b28e5a"},
			raw:           events.RawEvent{Type: "transfer", From: "GKLM...1V", To: "GHIJ...2M", Amount: "500.00", Asset: "USDC"},
		},
		{
			FirehoseEvent: pages.FirehoseEvent{Time: "2.1s", Type: "mint", TypeColor: "mint", ContractName: "BLND Token", ContractAddr: "CBLND...E84K", Ledger: "5,104,937", TxShort: "f7d2...4c", TxHash: "f7d24c9b"},
			raw:           events.RawEvent{Type: "mint", To: "GDEF...9R", Amount: "45,000", Asset: "BLND"},
		},
		{
			FirehoseEvent: pages.FirehoseEvent{Time: "7.3s", Type: "swap", TypeColor: "contract", ContractName: "Soroswap Router", ContractAddr: "CAXY...Z10P", Ledger: "5,104,936", TxShort: "9e8f...2a", TxHash: "9e8f2a1b"},
			raw:           events.RawEvent{Type: "swap", From: "GQRS...5X", PairIn: "500,000 AQUA", PairOut: "6,200 XLM", Router: "Soroswap"},
		},
		{
			FirehoseEvent: pages.FirehoseEvent{Time: "7.3s", Type: "approve", TypeColor: "approve", ContractName: "USDC Token", ContractAddr: "CCW6...7YMK", Ledger: "5,104,936", TxShort: "d3c2...7f", TxHash: "d3c27f1a"},
			raw:           events.RawEvent{Type: "approve", From: "GABC...7X", Spender: "CAXY...Z10P", Amount: "∞", Asset: "USDC"},
		},
	}

	result := make([]pages.FirehoseEvent, len(mocks))
	for i, m := range mocks {
		fe := m.FirehoseEvent
		if decoded := events.Decode(m.raw); decoded != nil {
			fe.TopicsHTML = decoded.TopicsHTML()
		}
		result[i] = fe
	}
	return result
}
