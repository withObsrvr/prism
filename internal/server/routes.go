package server

import (
	"net/http"

	"github.com/withObsrvr/prism/internal/handlers"
)

// Routes returns the complete HTTP handler with all routes registered.
// Following Alex Edwards' pattern from "Let's Go" — all routes defined
// in a single method, returning an http.Handler for testability.
func (app *Application) Routes() http.Handler {
	mux := http.NewServeMux()

	// Static files (CSS, JS, images).
	fileServer := http.FileServer(http.Dir("./web/static"))
	mux.Handle("GET /static/", http.StripPrefix("/static", fileServer))

	// Create handlers with shared dependencies.
	h := handlers.New(app.Logger, app.Config.Network)

	// ─────────────────────────────────────────────
	// Explorer routes
	// ─────────────────────────────────────────────

	// Home — search-first landing page
	mux.HandleFunc("GET /", h.Home)
	mux.HandleFunc("GET /search", h.Search)

	// Ledgers
	mux.HandleFunc("GET /ledger/{sequence}", h.LedgerDetail)

	// Transactions
	mux.HandleFunc("GET /tx/{hash}", h.TransactionReceipt)

	// ─────────────────────────────────────────────
	// Network routes
	// ─────────────────────────────────────────────

	mux.HandleFunc("GET /network", h.NetworkHealth)
	mux.HandleFunc("GET /network/validators/{id}", h.ValidatorDetail)

	// htmx partial: slide-out panel preview
	mux.HandleFunc("GET /network/validators/{id}/preview", h.ValidatorPreview)

	// ─────────────────────────────────────────────
	// Assets routes
	// ─────────────────────────────────────────────

	mux.HandleFunc("GET /assets", h.AssetDirectory)
	mux.HandleFunc("GET /assets/{slug}", h.AssetDetail)

	// ─────────────────────────────────────────────
	// Contracts routes (Soroban)
	// ─────────────────────────────────────────────

	mux.HandleFunc("GET /contracts", h.ContractList)
	mux.HandleFunc("GET /contracts/{id}", h.ContractDetail)
	mux.HandleFunc("GET /contracts/{id}/events", h.ContractEvents)

	// ─────────────────────────────────────────────
	// Account routes
	// ─────────────────────────────────────────────

	mux.HandleFunc("GET /account/{id}", h.AccountPortfolio)
	mux.HandleFunc("GET /account/{id}/smart", h.SmartAccountDashboard)

	// ─────────────────────────────────────────────
	// Dev Tools routes
	// ─────────────────────────────────────────────

	mux.HandleFunc("GET /events", h.EventsFirehose)
	mux.HandleFunc("GET /state", h.StateRentTracker)

	// ─────────────────────────────────────────────
	// NFT routes
	// ─────────────────────────────────────────────

	mux.HandleFunc("GET /nft", h.NftGallery)

	// ─────────────────────────────────────────────
	// htmx partial endpoints
	// ─────────────────────────────────────────────
	// These return HTML fragments, not full pages.
	// Triggered by hx-get on the client.

	mux.HandleFunc("GET /partials/search-results", h.SearchResults)
	mux.HandleFunc("GET /partials/live-feed", h.LiveFeed)

	// ─────────────────────────────────────────────
	// Health check (for load balancers / Nomad)
	// ─────────────────────────────────────────────

	mux.HandleFunc("GET /healthz", h.Healthz)

	// Wrap with middleware.
	return app.recoverPanic(app.logRequest(mux))
}
