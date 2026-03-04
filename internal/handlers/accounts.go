package handlers

import (
	"fmt"
	"net/http"
)

// AccountPortfolio renders the account overview page.
//
// Route: GET /account/{id}
// Template: pages/account_portfolio.templ
// Design: Balances, trustlines, recent operations, NFTs
func (h *Handlers) AccountPortfolio(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	fmt.Fprintf(w, "Account: %s", id)
}

// SmartAccountDashboard renders the smart account view for
// OpenZeppelin v0.5.0 wallet contracts.
//
// Route: GET /account/{id}/smart
// Template: pages/smart_account.templ
// Design: Owners, policies, execution history, session keys
func (h *Handlers) SmartAccountDashboard(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	fmt.Fprintf(w, "Smart Account: %s", id)
}
