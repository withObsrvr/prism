package handlers

import (
	"net/http"

	"github.com/withObsrvr/prism/internal/templates/fragments"
)

// AssetTableFragment returns the asset directory table.
func (h *Handlers) AssetTableFragment(w http.ResponseWriter, r *http.Request) {
	_ = networkFromRequest(r) // Will be used when wiring live data.
	data := mockAssetDirectoryData()

	if err := fragments.AssetTable(data).Render(r.Context(), w); err != nil {
		h.renderFragmentError(w, r, "Could not load assets", err)
	}
}
