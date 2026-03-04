package handlers

import (
	"fmt"
	"net/http"
)

// AssetDirectory renders the asset listing page.
//
// Route: GET /assets
// Template: pages/asset_directory.templ
// Design: Filterable table with trustline counts, volume, issuer info
func (h *Handlers) AssetDirectory(w http.ResponseWriter, r *http.Request) {
	// TODO: Fetch asset list from Obsrvr Lake (trustlines_current, trades)
	fmt.Fprint(w, "Asset Directory")
}

// AssetDetail renders a single asset page.
//
// Route: GET /assets/{code}-{issuer}
// Template: pages/asset_detail.templ
func (h *Handlers) AssetDetail(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")
	issuer := r.PathValue("issuer")
	fmt.Fprintf(w, "Asset: %s-%s", code, issuer)
}
