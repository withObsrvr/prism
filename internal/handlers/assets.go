package handlers

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/withObsrvr/prism/internal/gateway"
	"github.com/withObsrvr/prism/internal/templates/pages"
)

func (h *Handlers) AssetDirectory(w http.ResponseWriter, r *http.Request) {
	network := networkFromRequest(r)
	var data pages.AssetDirectoryData

	if h.Gateway != nil {
		d, err := h.buildAssetDirectoryData(r, network)
		if err != nil {
			h.Logger.Warn("gateway error, using mock data for assets", "error", err)
			data = mockAssetDirectoryData()
		} else {
			data = d
		}
	} else {
		data = mockAssetDirectoryData()
	}

	pages.AssetDirectory(data).Render(r.Context(), w)
}

func (h *Handlers) buildAssetDirectoryData(r *http.Request, network string) (pages.AssetDirectoryData, error) {
	ctx := r.Context()

	resp, err := h.Gateway.GetAssets(ctx, network, 20, "holder_count", "desc")
	if err != nil {
		return pages.AssetDirectoryData{}, fmt.Errorf("fetching assets: %w", err)
	}

	rows := make([]pages.AssetRow, 0, len(resp.Assets))
	for i, a := range resp.Assets {
		code := a.AssetCode
		if code == "" {
			code = "XLM"
		}
		typeBadge := "Classic"
		typeColor := "gray"
		if a.AssetType == "native" {
			typeBadge = "Native"
		}

		rows = append(rows, pages.AssetRow{
			Rank:      i + 1,
			Code:      code,
			Name:      code,
			BgColor:   "bg-gray-600",
			Initial:   string([]rune(code)[0]),
			Holders:   gateway.FormatNumber(int64(a.HolderCount)),
			Supply:    a.CirculatingSupply,
			Volume:    a.Volume24H,
			TypeBadge: typeBadge,
			TypeColor: typeColor,
		})
	}

	data := pages.AssetDirectoryData{
		TotalAssets:  fmt.Sprintf("%d", resp.TotalAssets),
		ActiveFilter: "all",
		CurrentPage:  1,
		TotalPages:   1,
		Assets:       rows,
	}

	return data, nil
}

func (h *Handlers) AssetDetail(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	code, issuer, ok := strings.Cut(slug, "-")
	if !ok {
		http.NotFound(w, r)
		return
	}
	fmt.Fprintf(w, "Asset: %s-%s", code, issuer)
}
