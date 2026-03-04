package handlers

import (
	"fmt"
	"net/http"
)

// ContractList renders the smart contract directory.
//
// Route: GET /contracts
// Template: pages/contract_list.templ
// Design: Table with call counts, deployed date, verification status
func (h *Handlers) ContractList(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "Contracts")
}

// ContractDetail renders a single contract page.
//
// Route: GET /contracts/{id}
// Template: pages/contract_detail.templ
// Design: Entity header (violet badges), state viewer, call history, source
func (h *Handlers) ContractDetail(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	fmt.Fprintf(w, "Contract: %s", id)
}

// ContractEvents returns contract events, either as a full page
// or as an htmx partial for tab switching.
//
// Route: GET /contracts/{id}/events
// htmx: hx-get for tab switch, returns fragment if HX-Request header present
func (h *Handlers) ContractEvents(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if isHTMX(r) {
		// Return just the events table fragment for tab swap.
		// TODO: Render components.EventsTable(events).Render(r.Context(), w)
		fmt.Fprintf(w, `<div id="tab-content">Events for %s</div>`, id)
		return
	}

	// Full page render with events tab active.
	fmt.Fprintf(w, "Contract Events: %s", id)
}
