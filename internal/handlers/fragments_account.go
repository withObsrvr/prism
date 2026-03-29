package handlers

import (
	"net/http"

	"github.com/withObsrvr/prism/internal/templates/fragments"
	"github.com/withObsrvr/prism/internal/templates/pages"
)

// buildAccountFragmentData fetches account detail data for fragment rendering.
// Returns nil if live data is unavailable or not requested.
func (h *Handlers) buildAccountFragmentData(r *http.Request, network, accountID string) *pages.AccountData {
	if !h.useLiveData(r) {
		return nil
	}
	data, err := h.buildAccountData(r, network, accountID)
	if err != nil {
		h.Logger.Warn("live account data failed, falling back to mock", "error", err, "account", accountID)
		return nil
	}
	return &data
}

// AccountBalancesFragment returns the portfolio balances table.
func (h *Handlers) AccountBalancesFragment(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	network := networkFromRequest(r)

	data := h.buildAccountFragmentData(r, network, id)
	if data == nil {
		mock := mockAccountData()
		data = &mock
	}

	if err := fragments.AccountBalances(data.Balances).Render(r.Context(), w); err != nil {
		h.renderFragmentError(w, r, "Could not load balances", err)
	}
}

// AccountActivityFragment returns the activity + contracts + offers.
func (h *Handlers) AccountActivityFragment(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	network := networkFromRequest(r)

	data := h.buildAccountFragmentData(r, network, id)
	if data == nil {
		mock := mockAccountData()
		data = &mock
	}

	if err := fragments.AccountActivity(*data).Render(r.Context(), w); err != nil {
		h.renderFragmentError(w, r, "Could not load activity", err)
	}
}

// AccountSignersFragment returns the signers & thresholds section.
func (h *Handlers) AccountSignersFragment(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	network := networkFromRequest(r)

	data := h.buildAccountFragmentData(r, network, id)
	if data == nil {
		mock := mockAccountData()
		data = &mock
	}

	if err := fragments.AccountSigners(*data).Render(r.Context(), w); err != nil {
		h.renderFragmentError(w, r, "Could not load signers", err)
	}
}
