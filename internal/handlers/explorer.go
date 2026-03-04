package handlers

import (
	"fmt"
	"net/http"
)

// Home renders the search-first landing page.
//
// Route: GET /
// Template: pages/home.templ
// Design: Search input + live network pulse + curated entry points
func (h *Handlers) Home(w http.ResponseWriter, r *http.Request) {
	// Reject non-root paths (Go 1.22+ serves "/" as a catch-all).
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	// TODO: Fetch live network stats (latest ledger, TPS, recent txs)
	// TODO: Render pages.Home(data).Render(r.Context(), w)

	fmt.Fprintf(w, "Prism — Home [network=%s]", h.Network)
}

// Search renders the full search results page.
//
// Route: GET /search?q={query}
// Template: pages/search.templ (dual-view: transactions + accounts)
// htmx: On keystroke, hits /partials/search-results for live results
func (h *Handlers) Search(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	h.Logger.Debug("search", "query", query)

	// TODO: Detect query type (address, tx hash, ledger, asset)
	// TODO: Render pages.Search(query, results).Render(r.Context(), w)

	fmt.Fprintf(w, "Search: %s", query)
}

// SearchResults returns an HTML fragment for live search.
//
// Route: GET /partials/search-results?q={query}
// Triggered by: hx-get on the search input with hx-trigger="keyup changed delay:300ms"
// Returns: HTML fragment, not a full page
func (h *Handlers) SearchResults(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// TODO: Query Obsrvr Lake, return components.SearchResultList(results)
	fmt.Fprintf(w, `<div class="search-results">Results for: %s</div>`, query)
}

// LedgerDetail renders a single ledger page.
//
// Route: GET /ledger/{sequence}
// Template: pages/ledger_detail.templ
// Design: Header with ledger stats, tab for transactions
func (h *Handlers) LedgerDetail(w http.ResponseWriter, r *http.Request) {
	sequence := r.PathValue("sequence")

	// TODO: Fetch ledger from Obsrvr Lake
	// TODO: Render pages.LedgerDetail(ledger).Render(r.Context(), w)

	fmt.Fprintf(w, "Ledger: %s", sequence)
}

// TransactionReceipt renders a single transaction page.
//
// Route: GET /tx/{hash}
// Template: pages/transaction_receipt.templ
// Design: Human-readable summary + inline operations + progressive disclosure
func (h *Handlers) TransactionReceipt(w http.ResponseWriter, r *http.Request) {
	hash := r.PathValue("hash")

	// TODO: Fetch transaction from Obsrvr Lake
	// TODO: Generate human-readable summary via SEP-41/SEP-50
	// TODO: Detect patterns (self-payment, DEX routing, swap path)
	// TODO: Render pages.TransactionReceipt(tx).Render(r.Context(), w)

	fmt.Fprintf(w, "Transaction: %s", hash)
}
