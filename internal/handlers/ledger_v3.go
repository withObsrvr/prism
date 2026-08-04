package handlers

import (
	"net/http"

	pagesv2 "github.com/withObsrvr/prism/internal/templates/v2/pages"
)

// LedgerDetailV3 renders the ledger detail v3 prototype.
//
// Mock data only, by design: the page exists to validate the data contract of
// the v3 design against what obsrvr-lake can actually serve. It never calls the
// gateway, and it leaves LedgerDetailV2 untouched.
func (h *Handlers) LedgerDetailV3(w http.ResponseWriter, r *http.Request) {
	sequence := r.PathValue("sequence")
	network := networkFromRequest(r)

	data := mockLedgerDetailV3Data(sequence, network)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := pagesv2.LedgerDetailV3(data).Render(r.Context(), w); err != nil {
		if h.Logger != nil {
			h.Logger.Error("render ledger detail v3", "error", err)
		}
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}
