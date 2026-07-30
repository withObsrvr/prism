package handlers

import (
	"net/http"
	"net/url"

	componentsv2 "github.com/withObsrvr/prism/internal/templates/v2/components"
	pagesv2 "github.com/withObsrvr/prism/internal/templates/v2/pages"
	vmv2 "github.com/withObsrvr/prism/internal/templates/v2/viewmodel"
)

// HomeV2 renders a fast, fact-free shell. Live blockchain sections hydrate
// independently and never inherit synthetic values when the Gateway is absent.
func (h *Handlers) HomeV2(w http.ResponseWriter, r *http.Request) {
	network := networkFromRequest(r)
	data := emptyHomeV2Data(network)
	if h.useExplicitMockData(r) {
		data = mockHomeV2Data(network)
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := pagesv2.Home(data).Render(r.Context(), w); err != nil {
		if h.Logger != nil {
			h.Logger.Error("render home v2", "error", err)
		}
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func emptyHomeV2Data(network string) vmv2.HomeData {
	return vmv2.HomeData{
		Header: componentsv2.HeaderData{
			LedgerNumber: "Unavailable",
			AgeLabel:     "Waiting for ledger data",
			Network:      network,
		},
		TimelineURL:    homeV2FragmentURL("timeline", network, false),
		InsightsURL:    homeV2FragmentURL("insights", network, false),
		TTLURL:         homeV2FragmentURL("ttl", network, false),
		LeadersURL:     homeV2FragmentURL("leaders", network, false),
		UtilizationURL: homeV2FragmentURL("utilization", network, false),
		Prompt: vmv2.PromptData{
			Placeholder: "Transaction, account, contract, asset, or ledger",
		},
	}
}

func homeV2TimelineURL(network string, mock bool) string {
	return homeV2FragmentURL("timeline", network, mock)
}

func homeV2FragmentURL(fragment, network string, mock bool) string {
	query := url.Values{}
	if network != "" {
		query.Set("network", network)
	}
	if mock {
		query.Set("mock", "true")
	}
	return "/v2/home/" + fragment + "?" + query.Encode()
}
