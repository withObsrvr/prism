package handlers

import (
	"net/http"
	"strconv"

	pagesv2 "github.com/withObsrvr/prism/internal/templates/v2/pages"
)

// LedgerDetailV3 renders the ledger detail v3 prototype.
//
// The page starts from the mock and overlays whatever the lake can actually
// serve for the requested ledger, so the two are visibly distinguishable: a
// section backed by live data says so in its provenance, and a section still
// waiting on a design decision keeps the mock's own marking. It leaves
// LedgerDetailV2 untouched.
func (h *Handlers) LedgerDetailV3(w http.ResponseWriter, r *http.Request) {
	sequence := r.PathValue("sequence")
	network := networkFromRequest(r)

	data := mockLedgerDetailV3Data(sequence, network)
	if seq, err := strconv.ParseInt(sequence, 10, 64); err == nil {
		ctx := r.Context()
		h.overlayLedgerV3Header(ctx, network, seq, &data)
		h.overlayLedgerV3Capacity(ctx, network, seq, &data)
		changes := h.overlayLedgerV3Changes(ctx, network, seq, &data)
		h.overlayLedgerV3Fees(ctx, network, seq, &data)

		// Panes run last: the state pane reuses the change cells the changes
		// overlay has just built, and the transactions pane reuses the same
		// cached ledger response the header read.
		if full, err := h.Gateway.GetSilverLedgerFull(ctx, network, seq); err == nil && full != nil {
			h.overlayLedgerV3Panes(&data, network, full.Transactions, full.Operations, changes)
		}
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := pagesv2.LedgerDetailV3(data).Render(r.Context(), w); err != nil {
		if h.Logger != nil {
			h.Logger.Error("render ledger detail v3", "error", err)
		}
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}
