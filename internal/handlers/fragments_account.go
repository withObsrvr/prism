package handlers

import (
	"net/http"

	"github.com/withObsrvr/prism/internal/templates/fragments"
)

// AccountBalancesFragment returns the portfolio balances table.
func (h *Handlers) AccountBalancesFragment(w http.ResponseWriter, r *http.Request) {
	data := mockAccountData()

	if err := fragments.AccountBalances(data.Balances).Render(r.Context(), w); err != nil {
		h.Logger.Error("render fragment", "error", err)
		fragments.FragmentError("Could not load balances", r.URL.Path).Render(r.Context(), w)
	}
}

// AccountActivityFragment returns the activity + contracts + offers.
func (h *Handlers) AccountActivityFragment(w http.ResponseWriter, r *http.Request) {
	data := mockAccountData()

	if err := fragments.AccountActivity(data).Render(r.Context(), w); err != nil {
		h.Logger.Error("render fragment", "error", err)
		fragments.FragmentError("Could not load activity", r.URL.Path).Render(r.Context(), w)
	}
}

// AccountSignersFragment returns the signers & thresholds section.
func (h *Handlers) AccountSignersFragment(w http.ResponseWriter, r *http.Request) {
	data := mockAccountData()

	if err := fragments.AccountSigners(data).Render(r.Context(), w); err != nil {
		h.Logger.Error("render fragment", "error", err)
		fragments.FragmentError("Could not load signers", r.URL.Path).Render(r.Context(), w)
	}
}
