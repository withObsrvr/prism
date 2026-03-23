package handlers

import (
	"net/http"

	"github.com/withObsrvr/prism/internal/templates/fragments"
)

// NetworkStatsGridFragment returns the throughput chart + consensus/soroban/fees/protocol grid.
func (h *Handlers) NetworkStatsGridFragment(w http.ResponseWriter, r *http.Request) {
	_ = networkFromRequest(r) // Will be used when wiring live data.
	data := mockNetworkHealthData()

	if err := fragments.NetworkStatsGrid(data).Render(r.Context(), w); err != nil {
		h.renderFragmentError(w, r, "Could not load network stats", err)
	}
}

// NetworkValidatorsFragment returns the validators table.
func (h *Handlers) NetworkValidatorsFragment(w http.ResponseWriter, r *http.Request) {
	_ = networkFromRequest(r)
	data := mockNetworkHealthData()

	if err := fragments.NetworkValidators(data).Render(r.Context(), w); err != nil {
		h.renderFragmentError(w, r, "Could not load validators", err)
	}
}

// NetworkRecentLedgersFragment returns the recent ledgers table.
func (h *Handlers) NetworkRecentLedgersFragment(w http.ResponseWriter, r *http.Request) {
	_ = networkFromRequest(r)
	data := mockNetworkHealthData()

	if err := fragments.NetworkRecentLedgers(data).Render(r.Context(), w); err != nil {
		h.renderFragmentError(w, r, "Could not load recent ledgers", err)
	}
}
