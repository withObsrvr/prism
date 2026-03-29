package handlers

import (
	"net/http"

	"github.com/withObsrvr/prism/internal/templates/fragments"
	"github.com/withObsrvr/prism/internal/templates/pages"
)

// AssetTableFragment returns the asset directory table.
func (h *Handlers) AssetTableFragment(w http.ResponseWriter, r *http.Request) {
	network := networkFromRequest(r)

	var data pages.AssetDirectoryData

	if h.useLiveData(r) {
		if live, err := h.buildAssetDirectoryData(r, network); err == nil {
			data = live
		} else {
			h.Logger.Warn("live asset data failed, falling back to mock", "error", err)
		}
	}

	if data.Assets == nil {
		data = mockAssetDirectoryData()
	}

	if err := fragments.AssetTable(data).Render(r.Context(), w); err != nil {
		h.renderFragmentError(w, r, "Could not load assets", err)
	}
}
