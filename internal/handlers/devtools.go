package handlers

import (
	"fmt"
	"net/http"

	"github.com/withObsrvr/prism/internal/gateway"
	"github.com/withObsrvr/prism/internal/templates/pages"
)

func (h *Handlers) EventsFirehose(w http.ResponseWriter, r *http.Request) {
	network := networkFromRequest(r)
	var data pages.EventsFirehoseData

	if h.Gateway != nil {
		d, err := h.buildEventsFirehoseData(r, network)
		if err != nil {
			h.Logger.Warn("gateway error, using mock data for events firehose", "error", err)
			data = mockEventsFirehoseData()
		} else {
			data = d
		}
	} else {
		data = mockEventsFirehoseData()
	}

	pages.EventsFirehose(data).Render(r.Context(), w)
}

func (h *Handlers) buildEventsFirehoseData(r *http.Request, network string) (pages.EventsFirehoseData, error) {
	ctx := r.Context()

	transfers, err := h.Gateway.GetTransfers(ctx, network, 20)
	if err != nil {
		return pages.EventsFirehoseData{}, fmt.Errorf("fetching transfers: %w", err)
	}

	events := make([]pages.FirehoseEvent, 0, len(transfers))
	for _, t := range transfers {
		typeColor := "transfer"
		evtType := "transfer"

		events = append(events, pages.FirehoseEvent{
			Time:         t.Timestamp,
			Type:         evtType,
			TypeColor:    typeColor,
			ContractName: t.AssetCode,
			Ledger:       gateway.FormatNumber(t.LedgerSequence),
			TxShort:      gateway.ShortHash(t.TransactionHash),
			TxHash:       t.TransactionHash,
			DetailJSON: fmt.Sprintf("{\n  \"type\": \"%s\",\n  \"from\": \"%s\",\n  \"to\": \"%s\",\n  \"amount\": \"%s\",\n  \"asset\": \"%s\"\n}",
				evtType, gateway.ShortAddress(t.FromAccount), gateway.ShortAddress(t.ToAccount), t.Amount, t.AssetCode),
		})
	}

	data := pages.EventsFirehoseData{
		MatchedEvents: fmt.Sprintf("%d", len(transfers)),
		Events:        events,
	}

	return data, nil
}

func (h *Handlers) StateRentTracker(w http.ResponseWriter, r *http.Request) {
	// State rent requires contract storage TTL data, not in gateway.
	// Always uses mock data.
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
