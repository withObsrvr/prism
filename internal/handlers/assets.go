package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

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
	network := networkFromRequest(r)
	var data pages.AssetDirectoryData

	if h.useLiveData(r) {
		d, err := h.buildAssetDirectoryData(r, network)
		if err != nil {
			h.Logger.Warn("gateway error, using mock data for assets v2", "error", err)
			data = mockAssetDirectoryData()
		} else {
			data = d
		}
	} else {
		data = mockAssetDirectoryData()
	}

	pages.AssetDirectoryV2(data).Render(r.Context(), w)
}

func (h *Handlers) AssetPreview(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	network := networkFromRequest(r)
	code, _, _ := parseAssetSlug(slug)
	if code == "" {
		code = slug
	}

	var data pages.AssetPreviewData
	if h.useLiveData(r) {
		if live, err := h.buildAssetPreviewData(r, network, slug); err == nil {
			data = live
		} else {
			h.Logger.Warn("live asset preview failed, falling back to mock", "error", err, "slug", slug)
		}
	}
	if data.Code == "" {
		data = mockAssetPreviewData(code)
	}

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
		Slug:       row.Slug,
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
	var (
		classicCount int64
		sep41Count   int64
		trustlines   int64
		networkVol   int64
	)
	palette := []string{"bg-gray-900", "bg-blue-600", "bg-emerald-600", "bg-violet-600", "bg-cyan-600", "bg-amber-600", "bg-rose-600", "bg-indigo-600"}

	for i, a := range resp.Assets {
		code := a.AssetCode
		if code == "" {
			code = "XLM"
		}

		typeBadge := "Classic"
		typeColor := "gray"
		switch strings.ToLower(a.AssetType) {
		case "native":
			typeBadge = "Native"
		case "sep41", "sep-41", "token", "soroban":
			typeBadge = "SEP-41"
			typeColor = "violet"
		}
		if typeBadge == "SEP-41" {
			sep41Count++
		} else if typeBadge != "Native" {
			classicCount++
		}

		trustlines += int64(a.HolderCount)
		networkVol += parseVolume(a.Volume24H)
		bgColor := palette[i%len(palette)]
		initial := string([]rune(code)[0])
		if initial == "" {
			initial = "?"
		}

		rows = append(rows, pages.AssetRow{
			Rank:       i + 1,
			Code:       code,
			Slug:       assetSlug(code, a.AssetIssuer),
			Name:       liveAssetDisplayName(code, a.AssetType),
			Issuer:     gateway.ShortAddress(a.AssetIssuer),
			BgColor:    bgColor,
			Initial:    initial,
			Price:      "—",
			Change:     "—",
			IsUp:       true,
			IsVerified: a.AssetIssuer == "" || strings.EqualFold(a.AssetType, "native"),
			MarketCap:  "—",
			Holders:    gateway.FormatNumber(int64(a.HolderCount)),
			Supply:     gateway.FormatDecimalAmount(a.CirculatingSupply),
			Volume:     "$" + gateway.FormatAbbrev(parseVolume(a.Volume24H)),
			TypeBadge:  typeBadge,
			TypeColor:  typeColor,
		})
	}

	if classicCount == 0 && sep41Count == 0 {
		classicCount = int64(resp.TotalAssets)
	}

	data := pages.AssetDirectoryData{
		TotalAssets:   gateway.FormatNumber(int64(resp.TotalAssets)),
		ClassicCount:  gateway.FormatNumber(classicCount),
		SEP41Count:    gateway.FormatNumber(sep41Count),
		NetworkVolume: "$" + gateway.FormatAbbrev(networkVol),
		VolumeChange:  "Live snapshot",
		Trustlines:    gateway.FormatNumber(trustlines),
		DEXLiquidity:  "—",
		ActiveFilter:  "all",
		CurrentPage:   1,
		TotalPages:    1,
		Assets:        rows,
	}

	return data, nil
}

func (h *Handlers) AssetDetail(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	network := networkFromRequest(r)
	code, _, _ := parseAssetSlug(slug)
	if code == "" {
		code = slug
	}

	var data pages.AssetPreviewData
	if h.useLiveData(r) {
		if live, err := h.buildAssetPreviewData(r, network, slug); err == nil {
			data = live
		} else {
			h.Logger.Warn("live asset detail failed, falling back to mock", "error", err, "slug", slug)
		}
	}
	if data.Code == "" {
		data = mockAssetPreviewData(code)
	}

	if err := pages.AssetDetail(data).Render(r.Context(), w); err != nil {
		h.Logger.Error("render asset detail", "error", err, "slug", slug)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func (h *Handlers) buildAssetPreviewData(r *http.Request, network, slug string) (pages.AssetPreviewData, error) {
	ctx := r.Context()
	code, _, canonical := parseAssetSlug(slug)
	if code == "" {
		code = slug
	}

	detail, err := h.Gateway.GetAssetDetail(ctx, network, canonical)
	if err != nil {
		return pages.AssetPreviewData{}, fmt.Errorf("fetching asset detail %q: %w", canonical, err)
	}
	if detail == nil {
		return pages.AssetPreviewData{}, fmt.Errorf("asset detail %q returned nil", canonical)
	}

	data := mapAssetDetailToPreview(detail, canonical)
	h.enrichAssetPreviewData(ctx, network, canonical, &data)
	return data, nil
}

func liveAssetDisplayName(code, assetType string) string {
	switch strings.ToUpper(code) {
	case "XLM":
		return "Stellar Lumens"
	case "USDC":
		return "USD Coin"
	case "EURC":
		return "Euro Coin"
	}
	if strings.EqualFold(assetType, "native") {
		return "Stellar Lumens"
	}
	return code
}

func liveAssetTypeBadge(assetType string) string {
	switch strings.ToLower(assetType) {
	case "native":
		return "Native"
	case "sep41", "sep-41", "token", "soroban":
		return "SEP-41"
	default:
		return "Classic"
	}
}

func liveAssetTypeColor(assetType string) string {
	if liveAssetTypeBadge(assetType) == "SEP-41" {
		return "violet"
	}
	return "gray"
}

func assetPreviewColor(code string) string {
	switch strings.ToUpper(code) {
	case "XLM":
		return "bg-gray-900"
	case "USDC", "EURC":
		return "bg-blue-600"
	case "AQUA":
		return "bg-cyan-600"
	case "BLND":
		return "bg-violet-600"
	default:
		return "bg-emerald-600"
	}
}

func formatAssetTransferAmount(raw, assetType, assetCode string) string {
	if strings.EqualFold(assetType, "native") || strings.EqualFold(assetCode, "XLM") {
		if stroops, err := strconv.ParseInt(strings.Split(raw, ".")[0], 10, 64); err == nil && !strings.Contains(raw, ".") {
			return gateway.FormatXLM(stroops)
		}
	}
	return gateway.FormatDecimalAmount(raw)
}

func classifyAssetTransferType(tx gateway.TransferEvent, issuer string) string {
	switch {
	case tx.FromAccount != "" && tx.FromAccount == tx.ToAccount:
		return "self"
	case issuer != "" && tx.FromAccount == issuer:
		return "issue"
	case issuer != "" && tx.ToAccount == issuer:
		return "redeem"
	case tx.SourceType == "soroban":
		return "soroban"
	default:
		return "transfer"
	}
}

func parseAssetSlug(slug string) (code string, issuer string, canonical string) {
	slug = strings.TrimSpace(slug)
	switch {
	case slug == "":
		return "", "", ""
	case strings.EqualFold(slug, "XLM"):
		return "XLM", "", "XLM"
	case strings.HasPrefix(slug, "C"):
		return slug, "", slug
	case strings.Contains(slug, ":"):
		code, issuer, _ = strings.Cut(slug, ":")
		return code, issuer, assetSlug(code, issuer)
	case strings.Contains(slug, "-"):
		code, issuer, _ = strings.Cut(slug, "-")
		if strings.HasPrefix(issuer, "G") {
			return code, issuer, assetSlug(code, issuer)
		}
	}
	return slug, "", slug
}

func assetSlug(code, issuer string) string {
	code = strings.TrimSpace(code)
	issuer = strings.TrimSpace(issuer)
	if code == "" {
		return ""
	}
	if issuer == "" {
		return code
	}
	return code + ":" + issuer
}

func mapAssetDetailToPreview(detail *gateway.AssetDetail, canonical string) pages.AssetPreviewData {
	assetCode := firstNonEmpty(detail.AssetCode, detail.Symbol, gateway.ShortAddress(firstNonEmpty(detail.ContractID, canonical)))
	assetType := detail.AssetType
	if assetType == "" && strings.HasPrefix(firstNonEmpty(detail.ContractID, canonical), "C") {
		assetType = "sep41"
	}
	name := firstNonEmpty(detail.DisplayName, detail.Name, detail.Symbol, liveAssetDisplayName(assetCode, assetType))
	issuerAddr := firstNonEmpty(detail.AssetIssuer, detail.ContractID, "Native")
	data := pages.AssetPreviewData{
		Code:         assetCode,
		Slug:         firstNonEmpty(detail.CanonicalSlug, canonical, assetSlug(assetCode, detail.AssetIssuer)),
		Name:         name,
		Initial:      string([]rune(firstNonEmpty(assetCode, "A"))[0]),
		BgColor:      assetPreviewColor(assetCode),
		IsVerified:   detail.TomlVerified || strings.EqualFold(assetType, "native") || detail.ContractID != "",
		TypeBadge:    liveAssetTypeBadge(assetType),
		TypeColor:    liveAssetTypeColor(assetType),
		Price:        "—",
		Change:       "Live asset",
		IsUp:         true,
		HomeDomain:   detail.HomeDomain,
		IssuerAddr:   issuerAddr,
		AuthRequired: detail.AuthRequired,
		AuthRevocable: detail.AuthRevocable,
		TomlVerified: detail.TomlVerified,
		HasSAC:       detail.LinkedTokenContract != "" || strings.EqualFold(detail.LinkedTokenType, "sac"),
		SACContractID: firstNonEmpty(detail.LinkedTokenContract, func() string { if strings.EqualFold(detail.LinkedTokenType, "sac") { return detail.ContractID }; return "" }()),
		SorobanPools: len(firstNonNilPairs(detail.TopPairs, detail.PairsPreview)),
		SorobanTransfers: func() string { if detail.Transfers24H > 0 { return gateway.FormatNumber(detail.Transfers24H) }; return "—" }(),
		Top10Pct:     formatPct(detail.Top10Concentration),
		Top100Pct:    formatPct(detail.Top100Concentration),
		GiniLabel:    concentrationLabel(detail.Top10Concentration),
		Pairs:        []pages.AssetPair{},
		RecentTxs:    []pages.AssetRecentTx{},
	}
	for _, p := range firstNonNilPairs(detail.TopPairs, detail.PairsPreview) {
		counter := firstNonEmpty(p.CounterCode, p.CounterAsset)
		data.Pairs = append(data.Pairs, pages.AssetPair{
			CounterCode: counter,
			Pool:        firstNonEmpty(p.Pool, "Market"),
			Liquidity:   firstNonEmpty(p.Liquidity, "—"),
			Volume24h:   firstNonEmpty(p.Volume24H, "—"),
		})
	}
	for _, tx := range firstNonNilTransfers(detail.RecentTransfers, detail.TransferPreview) {
		age := "—"
		if t, err := time.Parse(time.RFC3339, tx.Timestamp); err == nil {
			age = gateway.FormatAge(t)
		}
		data.RecentTxs = append(data.RecentTxs, pages.AssetRecentTx{
			Type:    classifyAssetTransferType(gateway.TransferEvent(tx), detail.AssetIssuer),
			Amount:  strings.TrimSpace(formatAssetTransferAmount(tx.Amount, assetType, tx.AssetCode) + " " + firstNonEmpty(tx.AssetCode, assetCode)),
			Account: gateway.ShortAddress(firstNonEmpty(tx.ToAccount, tx.FromAccount)),
			Age:     age,
		})
	}
	return data
}

func (h *Handlers) enrichAssetPreviewData(ctx context.Context, network, asset string, data *pages.AssetPreviewData) {
	if data == nil {
		return
	}
	if links, err := h.Gateway.GetAssetLinks(ctx, network, asset); err == nil && links != nil && len(links.Links) > 0 {
		for _, link := range links.Links {
			if data.SACContractID == "" && link.ContractID != "" {
				data.SACContractID = link.ContractID
				data.HasSAC = true
				break
			}
		}
	}
	if len(data.Pairs) == 0 {
		if pairs, err := h.Gateway.GetAssetPairs(ctx, network, asset); err == nil && pairs != nil {
			for _, p := range pairs.Pairs {
				data.Pairs = append(data.Pairs, pages.AssetPair{
					CounterCode: firstNonEmpty(p.CounterCode, p.CounterAsset),
					Pool:        firstNonEmpty(p.Pool, "Market"),
					Liquidity:   firstNonEmpty(p.Liquidity, "—"),
					Volume24h:   firstNonEmpty(p.Volume24H, "—"),
				})
			}
			if len(pairs.Pairs) > 0 {
				data.SorobanPools = len(pairs.Pairs)
			}
		}
	}
	if data.Top100Pct == "—" {
		if holders, err := h.Gateway.GetAssetHolders(ctx, network, asset); err == nil && holders != nil && len(holders.Holders) > 0 {
			data.Top100Pct = gateway.FormatNumber(int64(len(holders.Holders))) + " holders"
		}
	}
}

func firstNonNilPairs(primary, fallback []gateway.AssetPairSummary) []gateway.AssetPairSummary {
	if len(primary) > 0 {
		return primary
	}
	return fallback
}

func firstNonNilTransfers(primary, fallback []gateway.AssetTransferBrief) []gateway.AssetTransferBrief {
	if len(primary) > 0 {
		return primary
	}
	return fallback
}

func formatPct(v float64) string {
	if v <= 0 {
		return "—"
	}
	return fmt.Sprintf("%.1f%%", v)
}

func concentrationLabel(top10 float64) string {
	switch {
	case top10 >= 75:
		return "High"
	case top10 >= 40:
		return "Moderate"
	case top10 > 0:
		return "Low"
	default:
		return "—"
	}
}
