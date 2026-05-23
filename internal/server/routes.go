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
	h := handlers.New(app.Logger, app.Gateway)

	// ─────────────────────────────────────────────
	// Explorer routes
	// ─────────────────────────────────────────────

	// Home — search-first landing page
	mux.HandleFunc("GET /", h.Home)
	mux.HandleFunc("GET /v2/home", h.HomeV2)
	mux.HandleFunc("GET /v2/home/feed", h.HomeV2Feed)
	mux.HandleFunc("GET /v2/home/ledger", h.HomeLedgerFirstV2)
	mux.HandleFunc("GET /v2/explore", h.ExploreV2)
	mux.HandleFunc("GET /v2/explore/header", h.ExploreV2Header)
	mux.HandleFunc("GET /v2/explore/live", h.ExploreV2Live)
	mux.HandleFunc("GET /search", h.Search)
	mux.HandleFunc("GET /search/suggest", h.SearchSuggest)
	mux.HandleFunc("GET /search/submit", h.SearchSubmit)
	mux.HandleFunc("POST /search/submit", h.SearchSubmit)
	mux.HandleFunc("GET /v2/ask", h.AskV2)

	// Ledgers
	mux.HandleFunc("GET /ledger/{sequence}", h.LedgerDetail)
	mux.HandleFunc("GET /v2/ledger/{sequence}", h.LedgerDetailV2)
	mux.HandleFunc("GET /v2/ledger/{sequence}/card.png", h.LedgerCardV2)
	mux.HandleFunc("GET /v2/ledger/{sequence}/card.svg", h.LedgerCardSVGV2)

	// Transactions
	mux.HandleFunc("GET /tx/{hash}", h.TransactionReceipt)
	mux.HandleFunc("GET /v2/tx/{hash}", h.TransactionReceiptV2)
	mux.HandleFunc("GET /v2/tx/{hash}/card.png", h.TransactionCardV2)
	mux.HandleFunc("GET /v2/tx/{hash}/card.svg", h.TransactionCardSVGV2)
	mux.HandleFunc("GET /tx/v3/{hash}", h.TransactionReceiptV3)
	mux.HandleFunc("GET /tx/v2/{hash}", h.TransactionReceiptV2)
	mux.HandleFunc("GET /tx/v1/{hash}", h.TransactionReceiptV1)

	// ─────────────────────────────────────────────
	// Network routes
	// ─────────────────────────────────────────────

	mux.HandleFunc("GET /network", h.NetworkHealth)
	mux.HandleFunc("GET /network/v1", h.NetworkHealthV1)
	mux.HandleFunc("GET /network/validators/{id}", h.ValidatorDetail)

	// htmx partial: slide-out panel preview
	mux.HandleFunc("GET /network/validators/{id}/preview", h.ValidatorPreview)

	// ─────────────────────────────────────────────
	// Assets routes
	// ─────────────────────────────────────────────

	mux.HandleFunc("GET /assets", h.AssetDirectoryV2)
	mux.HandleFunc("GET /v2/assets", h.AssetDirectoryV2)
	mux.HandleFunc("GET /v2/assets/{slug}", h.AssetDetailV2)
	mux.HandleFunc("GET /assets/v1", h.AssetDirectory)
	mux.HandleFunc("GET /assets/{slug}/preview", h.AssetPreview)
	mux.HandleFunc("GET /assets/{slug}", h.AssetDetail)

	// ─────────────────────────────────────────────
	// Contracts routes (Soroban)
	// ─────────────────────────────────────────────

	mux.HandleFunc("GET /contracts", h.ContractList)
	mux.HandleFunc("GET /contracts/{id}", h.ContractDetail)
	mux.HandleFunc("GET /contracts/{id}/events", h.ContractEvents)
	mux.HandleFunc("GET /v2/contract/{id}", h.ContractDetailV2)
	mux.HandleFunc("GET /v2/contracts/{id}", h.ContractDetailV2)

	// ─────────────────────────────────────────────
	// Account routes
	// ─────────────────────────────────────────────

	mux.HandleFunc("GET /account/{id}", h.AccountPortfolio)
	mux.HandleFunc("GET /v2/account/{id}", h.GAccountDetailV2)
	mux.HandleFunc("GET /account/{id}/smart", h.SmartAccountDashboard)
	mux.HandleFunc("GET /v2/account/{id}/smart", h.SmartAccountDashboardV2)

	// ─────────────────────────────────────────────
	// Dev Tools routes
	// ─────────────────────────────────────────────

	mux.HandleFunc("GET /events", h.EventsFirehose)
	mux.HandleFunc("GET /events/v1", h.EventsFirehoseV1)
	mux.HandleFunc("GET /state", h.StateRentTracker)

	// ─────────────────────────────────────────────
	// NFT routes
	// ─────────────────────────────────────────────

	mux.HandleFunc("GET /nft", h.NftGallery)

	// ─────────────────────────────────────────────
	// htmx partial endpoints (legacy)
	// ─────────────────────────────────────────────
	// These return HTML fragments, not full pages.
	// Triggered by hx-get on the client.

	mux.HandleFunc("GET /partials/search-results", h.SearchResults)
	mux.HandleFunc("GET /partials/live-feed", h.LiveFeed)
	mux.HandleFunc("GET /partials/latest-ledger", h.LatestLedgerPartial)

	// ─────────────────────────────────────────────
	// htmx fragment endpoints
	// ─────────────────────────────────────────────
	// Per-section HTML fragments loaded by shell pages.

	// Home page fragments
	mux.HandleFunc("GET /fragments/home/network-pulse", h.HomeNetworkPulseFragment)
	mux.HandleFunc("GET /fragments/home/recent-txs", h.HomeRecentTxsFragment)
	mux.HandleFunc("GET /fragments/home/recent-ledgers", h.HomeRecentLedgersFragment)
	mux.HandleFunc("GET /fragments/home/trending-contracts", h.HomeTrendingContractsFragment)
	mux.HandleFunc("GET /fragments/home/sidebar", h.HomeSidebarFragment)

	// Transaction detail fragments
	mux.HandleFunc("GET /fragments/tx/v2/{hash}/hero", h.TxV2HeroFragment)
	mux.HandleFunc("GET /fragments/tx/v2/{hash}/detail", h.TxV2DetailFragment)
	mux.HandleFunc("GET /fragments/tx/v2/{hash}/sidebar", h.TxV2SidebarFragment)
	mux.HandleFunc("GET /fragments/tx/{hash}/overview", h.TxOverviewFragment)
	mux.HandleFunc("GET /fragments/tx/{hash}/operations", h.TxOperationsFragment)
	mux.HandleFunc("GET /fragments/tx/{hash}/events", h.TxEventsFragment)
	mux.HandleFunc("GET /fragments/tx/{hash}/effects", h.TxEffectsFragment)
	mux.HandleFunc("GET /fragments/tx/{hash}/balance-changes", h.TxBalanceChangesFragment)
	mux.HandleFunc("GET /fragments/tx/{hash}/timeline", h.TxTimelineFragment)
	mux.HandleFunc("GET /fragments/tx/{hash}/state-changes", h.TxStateChangesFragment)

	// Ledger detail fragments
	mux.HandleFunc("GET /fragments/ledger/{sequence}/txs", h.LedgerTxsFragment)
	mux.HandleFunc("GET /fragments/ledger/{sequence}/ops-fees", h.LedgerOpsAndFeesFragment)
	mux.HandleFunc("GET /fragments/ledger/{sequence}/soroban", h.LedgerSorobanFragment)

	// Account fragments
	mux.HandleFunc("GET /fragments/account/{id}/balances", h.AccountBalancesFragment)
	mux.HandleFunc("GET /fragments/account/{id}/activity", h.AccountActivityFragment)
	mux.HandleFunc("GET /fragments/account/{id}/signers", h.AccountSignersFragment)

	// Contract detail fragments
	mux.HandleFunc("GET /fragments/contract/{id}/info", h.ContractInfoFragment)
	mux.HandleFunc("GET /fragments/contract/{id}/functions", h.ContractFunctionsFragment)
	mux.HandleFunc("GET /fragments/contract/{id}/invocations", h.ContractInvocationsFragment)

	// Network health fragments
	mux.HandleFunc("GET /fragments/network/stats-grid", h.NetworkStatsGridFragment)
	mux.HandleFunc("GET /fragments/network/validators", h.NetworkValidatorsFragment)
	mux.HandleFunc("GET /fragments/network/recent-ledgers", h.NetworkRecentLedgersFragment)

	// Asset directory fragments
	mux.HandleFunc("GET /fragments/assets/table", h.AssetTableFragment)
	mux.HandleFunc("GET /fragments/assets/{slug}/links", h.AssetLinksFragment)
	mux.HandleFunc("GET /fragments/assets/{slug}/holders", h.AssetHoldersFragment)
	mux.HandleFunc("GET /fragments/assets/{slug}/pairs", h.AssetPairsFragment)

	// Events firehose fragments
	mux.HandleFunc("GET /fragments/events/stream", h.EventsStreamFragment)

	// ─────────────────────────────────────────────
	// API endpoints
	// ─────────────────────────────────────────────

	mux.HandleFunc("PUT /api/network", h.SetNetwork)

	// ─────────────────────────────────────────────
	// Health check (for load balancers / Nomad)
	// ─────────────────────────────────────────────

	mux.HandleFunc("GET /healthz", h.Healthz)

	// Wrap with middleware.
	return app.recoverPanic(app.logRequest(mux))
}
