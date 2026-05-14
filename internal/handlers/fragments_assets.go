package handlers

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/withObsrvr/prism/internal/gateway"
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

func (h *Handlers) AssetLinksFragment(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	network := networkFromRequest(r)
	links := []pages.AssetLink{}

	if h.useLiveData(r) {
		if resp, err := h.Gateway.GetAssetLinks(r.Context(), network, slug); err == nil && resp != nil {
			for _, link := range resp.Links {
				href := firstNonEmpty(link.Route, "/assets/"+firstNonEmpty(link.CanonicalSlug, link.ContractID, assetSlug(link.AssetCode, link.AssetIssuer)))
				meta := firstNonEmpty(link.TokenType, gateway.ShortAddress(firstNonEmpty(link.ContractID, link.AssetIssuer)), "asset")
				links = append(links, pages.AssetLink{
					Relation: strings.ReplaceAll(firstNonEmpty(link.Relation, "related_asset"), "_", " "),
					Label:    firstNonEmpty(link.Label, link.AssetCode, gateway.ShortAddress(firstNonEmpty(link.ContractID, link.AssetIssuer))),
					Href:     href,
					Meta:     meta,
				})
			}
		} else if err != nil {
			h.Logger.Warn("live asset links failed", "error", err, "slug", slug)
		}
	}

	if err := fragments.AssetLinks("Related Links", links).Render(r.Context(), w); err != nil {
		h.renderFragmentError(w, r, "Could not load related links", err)
	}
}

func (h *Handlers) AssetHoldersFragment(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	network := networkFromRequest(r)
	holders := []pages.AssetHolder{}

	if h.useLiveData(r) {
		if resp, err := h.Gateway.GetAssetHolders(r.Context(), network, slug); err == nil && resp != nil {
			for _, holder := range resp.Holders {
				holders = append(holders, pages.AssetHolder{
					Account: gateway.ShortAddress(holder.Account),
					Balance: firstNonEmpty(holder.Balance, "—"),
					Share:   func() string { if holder.SharePct > 0 { return fmt.Sprintf("%.2f%%", holder.SharePct) }; return "—" }(),
					Href:    accountOrContractHref(holder.Account),
				})
			}
		} else if err != nil {
			h.Logger.Warn("live asset holders failed", "error", err, "slug", slug)
		}
	}

	if err := fragments.AssetHolders("Top Holders", holders).Render(r.Context(), w); err != nil {
		h.renderFragmentError(w, r, "Could not load holders", err)
	}
}

func (h *Handlers) AssetPairsFragment(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	network := networkFromRequest(r)
	pairs := []pages.AssetPair{}

	if h.useLiveData(r) {
		if resp, err := h.Gateway.GetAssetPairs(r.Context(), network, slug); err == nil && resp != nil {
			for _, p := range resp.Pairs {
				pairs = append(pairs, pages.AssetPair{
					CounterCode: firstNonEmpty(p.CounterCode, p.CounterAsset),
					Pool:        firstNonEmpty(p.Pool, "Market"),
					Liquidity:   firstNonEmpty(p.Liquidity, "—"),
					Volume24h:   firstNonEmpty(p.Volume24H, "—"),
				})
			}
		} else if err != nil {
			h.Logger.Warn("live asset pairs failed", "error", err, "slug", slug)
		}
	}

	if err := fragments.AssetPairs("Markets & Pools", pairs).Render(r.Context(), w); err != nil {
		h.renderFragmentError(w, r, "Could not load markets", err)
	}
}

func accountOrContractHref(id string) string {
	if strings.HasPrefix(id, "C") {
		return "/contracts/" + id
	}
	return "/account/" + id
}
