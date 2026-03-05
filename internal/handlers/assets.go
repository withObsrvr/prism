package handlers

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/withObsrvr/prism/internal/templates/pages"
)

func (h *Handlers) AssetDirectory(w http.ResponseWriter, r *http.Request) {
	data := pages.AssetDirectoryData{
		TotalAssets:   "12,847",
		ClassicCount:  "11,203",
		SEP41Count:    "1,644",
		NetworkVolume: "$24.8M",
		VolumeChange:  "+12.4%",
		Trustlines:    "8.2M",
		DEXLiquidity:  "$142M",
		ActiveFilter:  "all",
		CurrentPage:   1,
		TotalPages:    128,
		Assets: []pages.AssetRow{
			{Rank: 1, Code: "XLM", Name: "Stellar Lumens", BgColor: "bg-gray-900", Initial: "X", Price: "$0.097", Change: "+2.4%", IsUp: true, IsVerified: true, MarketCap: "$3.2B", Volume: "$89.2M", Supply: "50.0B", Holders: "8.2M", TypeBadge: "Native", TypeColor: "gray"},
			{Rank: 2, Code: "USDC", Name: "USD Coin", BgColor: "bg-blue-600", Initial: "U", Price: "$1.00", Change: "+0.01%", IsUp: true, IsVerified: true, MarketCap: "$142M", Volume: "$18.4M", Supply: "142M", Holders: "284K", TypeBadge: "Classic", TypeColor: "gray"},
			{Rank: 3, Code: "yUSDC", Name: "Blend USDC", BgColor: "bg-emerald-600", Initial: "y", Price: "$1.03", Change: "+3.2%", IsUp: true, IsVerified: true, MarketCap: "$84M", Volume: "$4.2M", Supply: "81.6M", Holders: "12.4K", TypeBadge: "SEP-41", TypeColor: "violet"},
			{Rank: 4, Code: "AQUA", Name: "Aquarius", BgColor: "bg-cyan-600", Initial: "A", Price: "$0.0049", Change: "-1.8%", IsUp: false, IsVerified: true, MarketCap: "$49M", Volume: "$2.1M", Supply: "10B", Holders: "142K", TypeBadge: "Classic", TypeColor: "gray"},
			{Rank: 5, Code: "BLND", Name: "Blend Token", BgColor: "bg-violet-600", Initial: "B", Price: "$0.039", Change: "+12.4%", IsUp: true, IsVerified: true, MarketCap: "$39M", Volume: "$1.8M", Supply: "1B", Holders: "8.4K", TypeBadge: "SEP-41", TypeColor: "violet"},
			{Rank: 6, Code: "SHX", Name: "Stronghold", BgColor: "bg-amber-600", Initial: "S", Price: "$0.0028", Change: "-0.5%", IsUp: false, MarketCap: "$28M", Volume: "$890K", Supply: "10B", Holders: "92K", TypeBadge: "Classic", TypeColor: "gray"},
			{Rank: 7, Code: "EURC", Name: "Euro Coin", BgColor: "bg-blue-500", Initial: "E", Price: "$1.08", Change: "+0.3%", IsUp: true, IsVerified: true, MarketCap: "$21M", Volume: "$1.2M", Supply: "19.4M", Holders: "18K", TypeBadge: "Classic", TypeColor: "gray"},
			{Rank: 8, Code: "RWA", Name: "Real World Asset", BgColor: "bg-rose-600", Initial: "R", Price: "$12.40", Change: "+5.1%", IsUp: true, MarketCap: "$12.4M", Volume: "$420K", Supply: "1M", Holders: "2.1K", TypeBadge: "SEP-41", TypeColor: "violet"},
			{Rank: 9, Code: "FIDR", Name: "Fidelity Fund", BgColor: "bg-green-700", Initial: "F", Price: "$100.00", Change: "+0.8%", IsUp: true, IsVerified: true, MarketCap: "$10M", Volume: "$200K", Supply: "100K", Holders: "847", TypeBadge: "Classic", TypeColor: "gray"},
			{Rank: 10, Code: "BTC", Name: "Wrapped Bitcoin", BgColor: "bg-orange-500", Initial: "B", Price: "$97,240", Change: "+1.2%", IsUp: true, MarketCap: "$9.7M", Volume: "$340K", Supply: "100", Holders: "1.2K", TypeBadge: "Classic", TypeColor: "gray"},
		},
	}
	pages.AssetDirectory(data).Render(r.Context(), w)
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
