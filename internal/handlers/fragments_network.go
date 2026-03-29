package handlers

import (
	"net/http"

	"github.com/withObsrvr/prism/internal/templates/fragments"
	"github.com/withObsrvr/prism/internal/templates/pages"
)

// buildNetworkFragmentData fetches network health data for fragment rendering.
func (h *Handlers) buildNetworkFragmentData(r *http.Request, network string) *pages.NetworkHealthData {
	if !h.useLiveData(r) {
		return nil
	}
	data, err := h.buildNetworkHealthData(r, network)
	if err != nil {
		h.Logger.Warn("live network data failed, falling back to mock", "error", err)
		return nil
	}
	return &data
}

// NetworkStatsGridFragment returns the throughput chart + consensus/soroban/fees/protocol grid.
func (h *Handlers) NetworkStatsGridFragment(w http.ResponseWriter, r *http.Request) {
	network := networkFromRequest(r)

	data := h.buildNetworkFragmentData(r, network)
	if data == nil {
		mock := mockNetworkHealthData()
		data = &mock
	}

	if err := fragments.NetworkStatsGrid(*data).Render(r.Context(), w); err != nil {
		h.renderFragmentError(w, r, "Could not load network stats", err)
	}
}

// NetworkValidatorsFragment returns the validators table.
// Validators data requires Radar client (not yet implemented) — always uses
// the mock validators from buildNetworkHealthData or mockNetworkHealthData.
func (h *Handlers) NetworkValidatorsFragment(w http.ResponseWriter, r *http.Request) {
	network := networkFromRequest(r)

	data := h.buildNetworkFragmentData(r, network)
	if data == nil {
		mock := mockNetworkHealthData()
		data = &mock
	}

	if err := fragments.NetworkValidators(*data).Render(r.Context(), w); err != nil {
		h.renderFragmentError(w, r, "Could not load validators", err)
	}
}

// NetworkRecentLedgersFragment returns the recent ledgers table.
func (h *Handlers) NetworkRecentLedgersFragment(w http.ResponseWriter, r *http.Request) {
	network := networkFromRequest(r)

	data := h.buildNetworkFragmentData(r, network)
	if data == nil {
		mock := mockNetworkHealthData()
		data = &mock
	}

	if err := fragments.NetworkRecentLedgers(*data).Render(r.Context(), w); err != nil {
		h.renderFragmentError(w, r, "Could not load recent ledgers", err)
	}
}
