package handlers

import (
	"fmt"
	"net/http"

	"github.com/withObsrvr/prism/internal/gateway"
	pagesv2 "github.com/withObsrvr/prism/internal/templates/v2/pages"
)

func (h *Handlers) HomeLedgerFirstV2(w http.ResponseWriter, r *http.Request) {
	network := networkFromRequest(r)
	data := mockHomeLedgerFirstV2Data()

	if h.useLiveData(r) {
		if live, err := h.buildHomeLedgerFirstV2Data(r, network); err == nil {
			data = live
		} else {
			h.Logger.Warn("live ledger-first v2 failed, falling back to mock", "error", err)
		}
	}

	if err := pagesv2.HomeLedgerFirst(data).Render(r.Context(), w); err != nil {
		h.Logger.Error("render ledger-first v2", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func (h *Handlers) buildHomeLedgerFirstV2Data(r *http.Request, network string) (pagesv2.HomeLedgerFirstData, error) {
	ctx := r.Context()
	if h.Gateway == nil {
		return pagesv2.HomeLedgerFirstData{}, fmt.Errorf("gateway unavailable")
	}

	bronze, err := h.Gateway.GetBronzeNetworkStats(ctx, network)
	if err != nil {
		return pagesv2.HomeLedgerFirstData{}, err
	}

	data := mockHomeLedgerFirstV2Data()
	data.LedgerNumber = gateway.FormatNumber(bronze.Ledger.LatestSequence)

	if summary, err := h.Gateway.GetSilverLedgerSummary(ctx, network, bronze.Ledger.LatestSequence); err == nil && summary != nil {
		applyLedgerSummaryV2(&data, summary)
	}
	return data, nil
}

func applyLedgerSummaryV2(data *pagesv2.HomeLedgerFirstData, summary *gateway.LedgerSummary) {
	if data == nil || summary == nil {
		return
	}
	if tx := firstNonZero64(summary.TxCount, summary.TransactionCount); tx > 0 {
		data.TransactionCount = gateway.FormatNumber(tx)
		if tx == 1 {
			data.TransactionLabel = "transaction"
		} else {
			data.TransactionLabel = "transactions"
		}
	}
	if summary.Classifications != nil {
		data.SwapCount = gateway.FormatNumber(summary.Classifications.Swaps)
		data.CallCount = gateway.FormatNumber(summary.Classifications.Calls)
		data.AgentCount = gateway.FormatNumber(summary.Classifications.Agents)
	}
	if summary.Swaps != 0 || data.SwapCount == "" {
		data.SwapCount = gateway.FormatNumber(summary.Swaps)
	}
	if summary.Calls != 0 || data.CallCount == "" {
		data.CallCount = gateway.FormatNumber(summary.Calls)
	}
	if summary.Agents != 0 || data.AgentCount == "" {
		data.AgentCount = gateway.FormatNumber(summary.Agents)
	}
	if summary.Utilization != nil {
		if summary.Utilization.InstructionPct > 0 {
			data.InstructionPct = summary.Utilization.InstructionPct
			data.InstructionPctLabel = fmt.Sprintf("%d%%", data.InstructionPct)
		}
		if summary.Utilization.ReadWritePct > 0 {
			data.ReadWritePct = summary.Utilization.ReadWritePct
			data.ReadWritePctLabel = fmt.Sprintf("%d%%", data.ReadWritePct)
		}
	}
	if summary.InstructionPct > 0 {
		data.InstructionPct = summary.InstructionPct
		data.InstructionPctLabel = fmt.Sprintf("%d%%", data.InstructionPct)
	}
	if summary.ReadWritePct > 0 {
		data.ReadWritePct = summary.ReadWritePct
		data.ReadWritePctLabel = fmt.Sprintf("%d%%", data.ReadWritePct)
	}
}

func mockHomeLedgerFirstV2Data() pagesv2.HomeLedgerFirstData {
	return pagesv2.HomeLedgerFirstData{
		LedgerNumber:        "52,844,201",
		TransactionCount:    "245",
		TransactionLabel:    "transactions",
		SwapCount:           "68",
		CallCount:           "86",
		AgentCount:          "39",
		InstructionPct:      64,
		InstructionPctLabel: "64%",
		ReadWritePct:        60,
		ReadWritePctLabel:   "60%",
		Tagline:             "Prism — a different kind of block explorer",
	}
}

func firstNonZero64(values ...int64) int64 {
	for _, v := range values {
		if v != 0 {
			return v
		}
	}
	return 0
}
