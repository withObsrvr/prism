package handlers

import (
	"fmt"
	"net/http"
)

// NetworkHealth renders the network overview page.
//
// Route: GET /network
// Template: pages/network_health.templ
// Design: 6 stat cards, validator table, consensus map
func (h *Handlers) NetworkHealth(w http.ResponseWriter, r *http.Request) {
	// TODO: Fetch validator list, network stats from Obsrvr Lake
	// TODO: Render pages.NetworkHealth(stats, validators).Render(r.Context(), w)

	fmt.Fprint(w, "Network Health")
}

// ValidatorDetail renders a full validator page.
//
// Route: GET /network/validators/{id}
// Template: pages/validator_detail.templ
// Design: Entity header + stat cards + tabs (Overview, Quorum Set, Trusts)
func (h *Handlers) ValidatorDetail(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	// TODO: Fetch validator from Obsrvr Lake
	// TODO: Render pages.ValidatorDetail(validator).Render(r.Context(), w)

	fmt.Fprintf(w, "Validator: %s", id)
}

// ValidatorPreview returns an HTML fragment for the slide-out panel.
//
// Route: GET /network/validators/{id}/preview
// Triggered by: hx-get on table row click, hx-target="#slideout"
// Returns: HTML fragment rendered inside the 480px slide-out panel
//
// This is the Ramp pattern — preview without full navigation.
func (h *Handlers) ValidatorPreview(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	// TODO: Fetch validator summary
	// TODO: Render components.ValidatorSlideout(validator).Render(r.Context(), w)

	fmt.Fprintf(w, `<div class="slideout-content">Preview: %s</div>`, id)
}
