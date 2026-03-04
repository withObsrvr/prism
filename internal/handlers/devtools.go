package handlers

import (
	"fmt"
	"net/http"
)

// EventsFirehose renders the real-time events stream page.
//
// Route: GET /events
// Template: pages/events_firehose.templ
// Design: Filterable live stream of CAP-67 events
// htmx: Uses hx-trigger="every 2s" or WebSocket for live updates
func (h *Handlers) EventsFirehose(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "Events Firehose")
}

// StateRentTracker renders the Soroban state and rent overview.
//
// Route: GET /state
// Template: pages/state_rent_tracker.templ
// Design: TTL distribution, rent costs, expiring entries
func (h *Handlers) StateRentTracker(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "State & Rent Tracker")
}

// LiveFeed returns an HTML fragment with the latest transactions.
//
// Route: GET /partials/live-feed
// Triggered by: hx-get with hx-trigger="every 3s" on the home page
// Returns: HTML fragment with the latest 5 transactions
func (h *Handlers) LiveFeed(w http.ResponseWriter, r *http.Request) {
	// TODO: Fetch latest transactions from Obsrvr Lake hot path (Postgres)
	// TODO: Render components.LiveFeedRows(txs).Render(r.Context(), w)

	fmt.Fprint(w, `<div class="live-feed">Latest transactions...</div>`)
}
