package handlers

import (
	"net/http"

	"github.com/withObsrvr/prism/internal/templates/fragments"
)

// NetworkStatsGridFragment returns the throughput chart + consensus/soroban/fees/protocol grid.
func (h *Handlers) NetworkStatsGridFragment(w http.ResponseWriter, r *http.Request) {
	data := mockNetworkHealthData()

	if err := fragments.NetworkStatsGrid(data).Render(r.Context(), w); err != nil {
		h.Logger.Error("render fragment", "error", err)
		fragments.FragmentError("Could not load network stats", r.URL.Path).Render(r.Context(), w)
	}
}

// NetworkValidatorsFragment returns the validators table.
func (h *Handlers) NetworkValidatorsFragment(w http.ResponseWriter, r *http.Request) {
	data := mockNetworkHealthData()

	if err := fragments.NetworkValidators(data).Render(r.Context(), w); err != nil {
		h.Logger.Error("render fragment", "error", err)
		fragments.FragmentError("Could not load validators", r.URL.Path).Render(r.Context(), w)
	}
}

// NetworkRecentLedgersFragment returns the recent ledgers table.
func (h *Handlers) NetworkRecentLedgersFragment(w http.ResponseWriter, r *http.Request) {
	data := mockNetworkHealthData()

	if err := fragments.NetworkRecentLedgers(data).Render(r.Context(), w); err != nil {
		h.Logger.Error("render fragment", "error", err)
		fragments.FragmentError("Could not load recent ledgers", r.URL.Path).Render(r.Context(), w)
	}
}
