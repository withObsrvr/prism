package handlers

import (
	"net/http"
	"net/url"
	"strings"

	prismsearch "github.com/withObsrvr/prism/internal/search"
	componentsv2 "github.com/withObsrvr/prism/internal/templates/v2/components"
	pagesv2 "github.com/withObsrvr/prism/internal/templates/v2/pages"
	vmv2 "github.com/withObsrvr/prism/internal/templates/v2/viewmodel"
)

func (h *Handlers) SearchUnsupportedV2(w http.ResponseWriter, r *http.Request) {
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	resolution := h.resolveSearch(r.Context(), networkFromRequest(r), query)
	if resolution.Kind != prismsearch.SearchUnsupported {
		hxOrRedirect(w, r, resolution.Destination)
		return
	}
	data := vmv2.SearchUnsupportedData{
		Header:         componentsv2.HeaderData{Network: networkFromRequest(r)},
		Query:          query,
		Interpretation: resolution.Summary,
		RuleID:         resolution.RuleID,
		Examples: []vmv2.SearchUnsupportedExample{
			{Label: "USDC swaps today", Href: searchSubmitHref("USDC swaps today")},
			{Label: "Failed transfers last hour", Href: searchSubmitHref("Failed transfers last hour")},
			{Label: "How active is XLM today?", Href: searchSubmitHref("How active is XLM today?")},
			{Label: "Which contracts have the least TTL runway?", Href: searchSubmitHref("Which contracts have the least TTL runway?")},
		},
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := pagesv2.SearchUnsupported(data).Render(r.Context(), w); err != nil {
		if h.Logger != nil {
			h.Logger.Error("render unsupported search", "error", err)
		}
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func searchSubmitHref(query string) string {
	return "/search/submit?q=" + url.QueryEscape(query)
}
