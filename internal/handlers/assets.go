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

func (h *Handlers) AssetDirectoryV2(w http.ResponseWriter, r *http.Request) {
	data := mockAssetDirectoryData()
	pages.AssetDirectoryV2(data).Render(r.Context(), w)
}

func (h *Handlers) AssetPreview(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")
	data := mockAssetPreviewData(code)
	pages.AssetPreview(data).Render(r.Context(), w)
}

func mockAssetPreviewData(code string) pages.AssetPreviewData {
	// Look up the table row for identity fields.
	dir := mockAssetDirectoryData()
	var row pages.AssetRow
	for _, a := range dir.Assets {
		if strings.EqualFold(a.Code, code) {
			row = a
			break
		}
	}
	if row.Code == "" && len(dir.Assets) > 0 {
		row = dir.Assets[0]
	}

	base := pages.AssetPreviewData{
		Code:       row.Code,
		Name:       row.Name,
		Initial:    row.Initial,
		BgColor:    row.BgColor,
		IsVerified: row.IsVerified,
		TypeBadge:  row.TypeBadge,
		TypeColor:  row.TypeColor,
		Price:      row.Price,
		Change:     row.Change,
		IsUp:       row.IsUp,
		// Defaults for fields not in the table
		HomeDomain:    "",
		IssuerAddr:    row.Issuer,
		TomlVerified:  row.IsVerified,
		Top10Pct:      "42.1%",
		Top100Pct:     "68.4%",
		GiniLabel:     "Moderate",
	}

	// Per-asset enrichment — the data the table CAN'T show.
	switch strings.ToUpper(code) {
	case "XLM":
		base.HomeDomain = "stellar.org"
		base.IssuerAddr = "Native"
		base.Top10Pct = "18.2%"
		base.Top100Pct = "31.5%"
		base.GiniLabel = "Low"
		base.HasSAC = true
		base.SACContractID = "CDLZF...XLM0"
		base.SorobanPools = 24
		base.SorobanTransfers = "284,102"
		base.Pairs = []pages.AssetPair{
			{CounterCode: "USDC", Pool: "Soroswap", Liquidity: "$12.4M", Volume24h: "$4.2M"},
			{CounterCode: "EURC", Pool: "Classic DEX", Liquidity: "$2.1M", Volume24h: "$890K"},
			{CounterCode: "AQUA", Pool: "Soroswap", Liquidity: "$1.8M", Volume24h: "$420K"},
		}
		base.RecentTxs = []pages.AssetRecentTx{
			{Type: "buy", Amount: "125,000 XLM", Account: "GABC...7X92", Age: "12s"},
			{Type: "sell", Amount: "50,000 XLM", Account: "GDEF...9R23", Age: "28s"},
			{Type: "send", Amount: "10,000 XLM", Account: "GHIJ...2M56", Age: "1m"},
			{Type: "buy", Amount: "84,200 XLM", Account: "GKLM...4P78", Age: "2m"},
		}
	case "USDC":
		base.HomeDomain = "centre.io"
		base.IssuerAddr = "GA5Z...SUL4"
		base.AuthRequired = true
		base.HasSAC = true
		base.SACContractID = "CCW6...7YMK"
		base.SorobanPools = 18
		base.SorobanTransfers = "142,847"
		base.Top10Pct = "62.4%"
		base.Top100Pct = "81.2%"
		base.GiniLabel = "High"
		base.Pairs = []pages.AssetPair{
			{CounterCode: "XLM", Pool: "Soroswap", Liquidity: "$12.4M", Volume24h: "$4.2M"},
			{CounterCode: "EURC", Pool: "Classic DEX", Liquidity: "$3.8M", Volume24h: "$1.1M"},
		}
		base.RecentTxs = []pages.AssetRecentTx{
			{Type: "send", Amount: "25,000 USDC", Account: "GCKW...NP4X", Age: "8s"},
			{Type: "buy", Amount: "12,400 USDC", Account: "GNOP...3W12", Age: "34s"},
			{Type: "send", Amount: "500 USDC", Account: "GQRS...5X67", Age: "1m"},
		}
	case "YUSDC":
		base.HomeDomain = "blend.capital"
		base.IssuerAddr = "GDBE...KJ3L"
		base.HasSAC = true
		base.SACContractID = "CBLND...YLD1"
		base.SorobanPools = 4
		base.SorobanTransfers = "28,401"
		base.Top10Pct = "38.7%"
		base.Top100Pct = "55.2%"
		base.GiniLabel = "Moderate"
		base.Pairs = []pages.AssetPair{
			{CounterCode: "USDC", Pool: "Blend", Liquidity: "$8.4M", Volume24h: "$1.2M"},
		}
		base.RecentTxs = []pages.AssetRecentTx{
			{Type: "buy", Amount: "8,400 yUSDC", Account: "GA7T...QE5L", Age: "2m"},
			{Type: "sell", Amount: "2,100 yUSDC", Account: "GBXR...KM92", Age: "8m"},
		}
	case "BLND":
		base.HomeDomain = "blend.capital"
		base.IssuerAddr = "CBLND...P2R8"
		base.HasSAC = true
		base.SACContractID = "CBLND...TKN1"
		base.SorobanPools = 3
		base.SorobanTransfers = "12,840"
		base.Pairs = []pages.AssetPair{
			{CounterCode: "XLM", Pool: "Soroswap", Liquidity: "$1.2M", Volume24h: "$340K"},
			{CounterCode: "USDC", Pool: "Soroswap", Liquidity: "$890K", Volume24h: "$210K"},
		}
		base.RecentTxs = []pages.AssetRecentTx{
			{Type: "buy", Amount: "45,000 BLND", Account: "GDEF...9R23", Age: "4m"},
			{Type: "send", Amount: "12,000 BLND", Account: "GAUTH...B4KQ", Age: "18m"},
		}
	default:
		// Generic for assets without specific mock data
		base.Pairs = []pages.AssetPair{
			{CounterCode: "XLM", Pool: "Classic DEX", Liquidity: "$420K", Volume24h: "$84K"},
		}
		base.RecentTxs = []pages.AssetRecentTx{
			{Type: "buy", Amount: "10,000 " + row.Code, Account: "GABC...7X92", Age: "5m"},
			{Type: "sell", Amount: "2,500 " + row.Code, Account: "GDEF...9R23", Age: "12m"},
		}
	}

	return base
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
