package handlers

import (
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/withObsrvr/prism/internal/events"
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

	pages.EventsFirehoseV2(data).Render(r.Context(), w)
}

func (h *Handlers) EventsFirehoseV1(w http.ResponseWriter, r *http.Request) {
	data := mockEventsFirehoseData()
	pages.EventsFirehose(data).Render(r.Context(), w)
}

func (h *Handlers) buildEventsFirehoseData(r *http.Request, network string) (pages.EventsFirehoseData, error) {
	ctx := r.Context()
	q := r.URL.Query()

	limit := 20
	if l := q.Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 200 {
			limit = parsed
		}
	}

	params := gateway.ExplorerEventsParams{
		Tab:          q.Get("tab"),
		ContractID:   q.Get("contract_id"),
		ContractName: q.Get("contract_name"),
		TxHash:       q.Get("tx_hash"),
		Cursor:       q.Get("cursor"),
		Limit:        limit,
		Order:        "desc",
	}
	if eventTypes := strings.TrimSpace(q.Get("type")); eventTypes != "" {
		params.Types = strings.Split(eventTypes, ",")
	}

	resp, err := h.Gateway.GetExplorerEvents(ctx, network, params)
	if err != nil {
		return pages.EventsFirehoseData{}, fmt.Errorf("fetching explorer events: %w", err)
	}

	events := make([]pages.FirehoseEvent, 0, len(resp.Events))
	for _, e := range resp.Events {
		events = append(events, mapExplorerEvent(e))
	}

	data := pages.EventsFirehoseData{
		MatchedEvents: gateway.FormatNumber(int64(resp.Meta.MatchedCount)),
		LedgerStart:   gateway.FormatNumber(resp.Meta.LedgerRange.Min),
		LedgerEnd:     gateway.FormatNumber(resp.Meta.LedgerRange.Max),
		Events:        events,
		HasMore:       resp.HasMore,
		ActiveTab:     params.Tab,
	}
	if resp.Meta.EventsPerSecond != nil {
		data.EventsPerSec = fmt.Sprintf("%.0f", *resp.Meta.EventsPerSecond)
	}
	if resp.NextCursor != nil {
		data.NextCursor = *resp.NextCursor
	}

	return data, nil
}

func mapExplorerEvent(e gateway.ExplorerEvent) pages.FirehoseEvent {
	age := ""
	if t, err := time.Parse(time.RFC3339, e.ClosedAt); err == nil {
		age = gateway.FormatAge(t)
	}

	contractName := derefOr(e.ContractName, "")
	if contractName == "" {
		contractName = derefOr(e.ContractSymbol, "Unknown")
	}

	asset := derefOr(e.ContractSymbol, derefOr(e.Topic3, ""))
	amount := extractAmount(e.DataDecoded)

	// Build RawEvent for the decoder to produce TopicsHTML.
	raw := events.RawEvent{
		Type:   e.Type,
		From:   gateway.ShortAddress(derefOr(e.Topic1, "")),
		To:     gateway.ShortAddress(derefOr(e.Topic2, "")),
		Amount: amount,
		Asset:  asset,
	}

	topicsHTML := ""
	if decoded := events.Decode(raw); decoded != nil {
		topicsHTML = decoded.TopicsHTML()
	}

	// Build detail JSON from topic and data fields.
	detail := map[string]string{"type": e.Type}
	if v := derefOr(e.Topic0, ""); v != "" {
		detail["topic0"] = v
	}
	if v := derefOr(e.Topic1, ""); v != "" {
		detail["topic1"] = v
	}
	if v := derefOr(e.Topic2, ""); v != "" {
		detail["topic2"] = v
	}
	if v := derefOr(e.Topic3, ""); v != "" {
		detail["topic3"] = v
	}
	if e.DataDecoded != nil {
		detail["data"] = *e.DataDecoded
	}
	detailJSON, _ := json.MarshalIndent(detail, "", "  ")

	fe := pages.FirehoseEvent{
		Time:         age,
		Type:         e.Type,
		TypeColor:    explorerEventTypeColor(e.Type),
		ContractName: contractName,
		ContractAddr: gateway.ShortAddress(derefOr(e.ContractID, "")),
		TopicsHTML:   topicsHTML,
		Ledger:       gateway.FormatNumber(e.LedgerSequence),
		TxShort:      gateway.ShortHash(e.TransactionHash),
		TxHash:       e.TransactionHash,
		DetailJSON:   string(detailJSON),
	}

	if e.Protocol != nil {
		fe.DetailMeta = *e.Protocol
	}

	if !e.PublicSuccessful() {
		fe.AlertBadge = "Failed"
		fe.AlertColor = "red"
	}

	return fe
}

func explorerEventTypeColor(eventType string) string {
	switch eventType {
	case "transfer":
		return "transfer"
	case "swap":
		return "contract"
	case "mint":
		return "mint"
	case "burn":
		return "burn"
	case "approve":
		return "approve"
	default:
		return "system"
	}
}

func derefOr(p *string, fallback string) string {
	if p != nil {
		return *p
	}
	return fallback
}

// extractAmount parses the data_decoded JSON from explorer events
// and returns a human-readable amount string.
// The API returns i128 values like: {"hi":0,"lo":100000000000,"type":"i128","value":"100000000000"}
// Stellar amounts use 7 decimal places (stroops).
func extractAmount(dataDecoded *string) string {
	if dataDecoded == nil {
		return ""
	}
	var data struct {
		Value string `json:"value"`
	}
	if err := json.Unmarshal([]byte(*dataDecoded), &data); err != nil || data.Value == "" {
		return ""
	}
	v, ok := new(big.Int).SetString(data.Value, 10)
	if !ok {
		return data.Value
	}
	// Convert from stroops (7 decimals) to human-readable.
	// Use big.Int arithmetic throughout to avoid int64 overflow on large i128 values.
	stroopsPerUnit := big.NewInt(10_000_000)
	whole := new(big.Int)
	frac := new(big.Int)
	whole.DivMod(v, stroopsPerUnit, frac)
	frac.Abs(frac)
	if frac.Sign() == 0 {
		return whole.String()
	}
	// Format fractional part as 7 digits with leading zeros, then trim trailing zeros.
	fracStr := fmt.Sprintf("%07d", frac.Int64())
	fracStr = strings.TrimRight(fracStr, "0")
	return whole.String() + "." + fracStr
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
	pages.RentTrackerV2(data).Render(r.Context(), w)
}

func (h *Handlers) LiveFeed(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, `<div class="live-feed">Latest transactions...</div>`)
}
