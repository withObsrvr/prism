package handlers

import (
	"net/http"

	"github.com/withObsrvr/prism/internal/templates/fragments"
)

// AccountBalancesFragment returns the portfolio balances table.
func (h *Handlers) AccountBalancesFragment(w http.ResponseWriter, r *http.Request) {
	_ = r.PathValue("id") // Will be used when wiring live data.
	data := mockAccountData()

	if err := fragments.AccountBalances(data.Balances).Render(r.Context(), w); err != nil {
		h.renderFragmentError(w, r, "Could not load balances", err)
	}
}

// AccountActivityFragment returns the activity + contracts + offers.
func (h *Handlers) AccountActivityFragment(w http.ResponseWriter, r *http.Request) {
	_ = r.PathValue("id")
	data := mockAccountData()

	if err := fragments.AccountActivity(data).Render(r.Context(), w); err != nil {
		h.renderFragmentError(w, r, "Could not load activity", err)
	}
}

// AccountSignersFragment returns the signers & thresholds section.
func (h *Handlers) AccountSignersFragment(w http.ResponseWriter, r *http.Request) {
	_ = r.PathValue("id")
	data := mockAccountData()

	if err := fragments.AccountSigners(data).Render(r.Context(), w); err != nil {
		h.renderFragmentError(w, r, "Could not load signers", err)
	}
}
