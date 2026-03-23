package handlers

import (
	"net/http"

	"github.com/withObsrvr/prism/internal/templates/fragments"
)

// AssetTableFragment returns the asset directory table.
func (h *Handlers) AssetTableFragment(w http.ResponseWriter, r *http.Request) {
	data := mockAssetDirectoryData()

	if err := fragments.AssetTable(data).Render(r.Context(), w); err != nil {
		h.Logger.Error("render fragment", "error", err)
		fragments.FragmentError("Could not load assets", r.URL.Path).Render(r.Context(), w)
	}
}
