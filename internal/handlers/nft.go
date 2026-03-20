package handlers

import (
	"net/http"

	"github.com/withObsrvr/prism/internal/templates/pages"
)

func (h *Handlers) NftGallery(w http.ResponseWriter, r *http.Request) {
	data := pages.NftGalleryData{
		CollectionName: "Stellar Geometries",
		Description:    "Generative on-chain geometric art stored entirely on Soroban. 256 unique pieces, algorithmically composed at mint.",
		ContractID:     "CNFT...G7K2",
		IsVerified:     true,
		Standard:       "OZ Non-Fungible",
		FloorPrice:     "1,200",
		FloorUSD:       "$116.40",
		TotalVolume:    "84,200",
		VolumeUSD:      "$8,167.40",
		ItemCount:      "256",
		HolderCount:    "142",
		HolderPct:      "55%",
		RentStatus:     "Healthy",
		RentTTL:        "120 days",
		Featured: pages.NftItem{
			TokenID:         "0042",
			Name:            "Convergence Point",
			Description:     "A study in rotational symmetry and negative space. Seed: 0x8f2a1b. Composed at ledger 4,102,847.",
			Owner:           "GABC...7X92",
			TokenStandard:   "OZ Non-Fungible (Soroban)",
			LastTransfer:    "Oct 18, 2026",
			MetadataStorage: "On-chain",
			Traits: []pages.NftTrait{
				{Name: "Palette", Value: "Midnight Rose", Rarity: "12%"},
				{Name: "Complexity", Value: "High", Rarity: "8%"},
				{Name: "Shape", Value: "Triangle", Rarity: "25%"},
				{Name: "Symmetry", Value: "Rotational", Rarity: "18%"},
				{Name: "Elements", Value: "7", Rarity: "5%"},
			},
			Provenance: []pages.NftProvenance{
				{Action: "Transferred to you", Date: "Oct 18, 2026", Detail: "from GXYZ...4R21"},
				{Action: "Sold for 1,800 XLM", Date: "Sep 4, 2026", Detail: "GXYZ...4R21"},
				{Action: "Minted", Date: "Aug 12, 2026 · Ledger 4,102,847", Detail: "by GMINT...8K32"},
			},
		},
		Items: []pages.NftCard{
			{TokenID: "0042", Name: "Convergence Point", Floor: "1,200 XLM", IsOwned: true, ArtClass: "hero"},
			{TokenID: "0107", Name: "Radial Drift", Floor: "1,400 XLM", IsOwned: false, OwnerAddr: "GXYZ...4R21", ArtClass: "cool"},
			{TokenID: "0003", Name: "Axiom", Floor: "2,800 XLM", IsOwned: false, OwnerAddr: "GHIJ...2M56", ArtClass: "warm"},
			{TokenID: "0199", Name: "Fracture Line", Floor: "980 XLM", IsOwned: false, OwnerAddr: "GKLM...1V83", ArtClass: "nature"},
			{TokenID: "0088", Name: "Nested Echo", Floor: "1,100 XLM", IsOwned: true, ArtClass: "hero"},
			{TokenID: "0241", Name: "Phase Shift", Floor: "1,500 XLM", IsOwned: false, OwnerAddr: "GDEF...9R23", ArtClass: "cool"},
			{TokenID: "0156", Name: "Orbital Decay", Floor: "1,300 XLM", IsOwned: false, OwnerAddr: "GABC...7X92", ArtClass: "warm"},
			{TokenID: "0012", Name: "Primordial", Floor: "3,200 XLM", IsOwned: false, OwnerAddr: "GMINT...8K32", ArtClass: "nature"},
		},
	}
	pages.NftGalleryV2(data).Render(r.Context(), w)
}
