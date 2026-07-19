package handlers

import (
	"net/http"

	"github.com/a-h/templ"
	"github.com/withObsrvr/prism/internal/templates/pages"
	componentsv2 "github.com/withObsrvr/prism/internal/templates/v2/components"
	pagesv2 "github.com/withObsrvr/prism/internal/templates/v2/pages"
	ui "github.com/withObsrvr/prism/internal/viewmodel"
)

// ContractCurrentBalancesFragment loads regular contract balances independently
// from the contract detail shell, so a slow balance source cannot delay the page.
func (h *Handlers) ContractCurrentBalancesFragment(w http.ResponseWriter, r *http.Request) {
	contractID := r.PathValue("id")
	network := networkFromRequest(r)
	portfolio := unavailableBalancePortfolio(contractID)

	if r.URL.Query().Get("mock") == "true" {
		portfolio = mockContractDetailData().Portfolio
		portfolio.OwnerID = contractID
	} else if h.Gateway != nil {
		var err error
		portfolio, err = h.getAddressBalancePortfolio(r.Context(), network, contractID)
		if err != nil && h.Logger != nil {
			h.Logger.Warn("contract balances unavailable", "contract", contractID, "network", network, "error", err)
		}
	}

	var component templ.Component
	if r.URL.Query().Get("surface") == "v2" {
		component = componentsv2.BalancePortfolio(portfolio)
	} else {
		component = pages.BalancePortfolio(portfolio)
	}
	if err := component.Render(r.Context(), w); err != nil {
		h.renderFragmentError(w, r, "Could not render contract balances", err)
	}
}

// SmartAccountCurrentBalancesFragment loads the wallet-specific balance
// document and updates both the portfolio table and smart-account hero.
func (h *Handlers) SmartAccountCurrentBalancesFragment(w http.ResponseWriter, r *http.Request) {
	contractID := r.PathValue("id")
	network := networkFromRequest(r)
	portfolio := unavailableBalancePortfolio(contractID)

	if r.URL.Query().Get("mock") == "true" {
		portfolio = mockSmartAccountDetailData(contractID, network).Portfolio
	} else if h.Gateway != nil {
		var err error
		portfolio, err = h.getSmartWalletBalancePortfolio(r.Context(), network, contractID)
		if err != nil && h.Logger != nil {
			h.Logger.Warn("smart account balances unavailable", "contract", contractID, "network", network, "error", err)
		}
	}

	if err := renderSmartAccountBalanceFragment(r, w, portfolio); err != nil {
		h.renderFragmentError(w, r, "Could not render smart account balances", err)
	}
}

func renderSmartAccountBalanceFragment(r *http.Request, w http.ResponseWriter, portfolio ui.BalancePortfolio) error {
	if r.URL.Query().Get("surface") == "v2" {
		return pagesv2.SmartAccountBalancesFragment(portfolio).Render(r.Context(), w)
	}
	return pages.SmartAccountBalancesFragment(portfolio).Render(r.Context(), w)
}
