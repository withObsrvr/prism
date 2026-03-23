package handlers

import (
	"net/http"
	"time"

	"github.com/withObsrvr/prism/internal/gateway"
	"github.com/withObsrvr/prism/internal/templates/fragments"
)

// HomeNetworkPulseFragment returns the 5-metric Network Pulse grid.
func (h *Handlers) HomeNetworkPulseFragment(w http.ResponseWriter, r *http.Request) {
	network := networkFromRequest(r)
	data := mockHomeData(network)

	// Overlay live latest ledger if gateway is available.
	if h.Gateway != nil {
		ctx := r.Context()
		if bronze, err := h.Gateway.GetBronzeNetworkStats(ctx, network); err == nil {
			data.LatestLedger = gateway.FormatNumber(bronze.Ledger.LatestSequence)
			if t, err := time.Parse(time.RFC3339, bronze.Ledger.ClosedAt); err == nil {
				data.LedgerAge = gateway.FormatAge(t)
			}
		} else if stats, err := h.Gateway.GetNetworkStats(ctx, network); err == nil {
			data.LatestLedger = gateway.FormatNumber(stats.Ledger.CurrentSequence)
			if t, err := time.Parse(time.RFC3339, stats.GeneratedAt); err == nil {
				data.LedgerAge = gateway.FormatAge(t)
			}
		}
	}

	if err := fragments.HomeNetworkPulse(data).Render(r.Context(), w); err != nil {
		h.Logger.Error("render fragment", "error", err)
		fragments.FragmentError("Could not load network stats", "/fragments/home/network-pulse").Render(r.Context(), w)
	}
}

// HomeRecentTxsFragment returns the Latest Transactions table.
func (h *Handlers) HomeRecentTxsFragment(w http.ResponseWriter, r *http.Request) {
	network := networkFromRequest(r)
	data := mockHomeData(network)

	if err := fragments.HomeRecentTxs(data.Transactions).Render(r.Context(), w); err != nil {
		h.Logger.Error("render fragment", "error", err)
		fragments.FragmentError("Could not load transactions", "/fragments/home/recent-txs").Render(r.Context(), w)
	}
}

// HomeRecentLedgersFragment returns the Latest Ledgers sidebar.
func (h *Handlers) HomeRecentLedgersFragment(w http.ResponseWriter, r *http.Request) {
	network := networkFromRequest(r)
	data := mockHomeData(network)

	if err := fragments.HomeRecentLedgers(data.Ledgers).Render(r.Context(), w); err != nil {
		h.Logger.Error("render fragment", "error", err)
		fragments.FragmentError("Could not load ledgers", "/fragments/home/recent-ledgers").Render(r.Context(), w)
	}
}

// HomeTrendingContractsFragment returns the Trending Contracts table.
func (h *Handlers) HomeTrendingContractsFragment(w http.ResponseWriter, r *http.Request) {
	network := networkFromRequest(r)
	data := mockHomeData(network)

	if err := fragments.HomeTrendingContracts(data.Contracts).Render(r.Context(), w); err != nil {
		h.Logger.Error("render fragment", "error", err)
		fragments.FragmentError("Could not load contracts", "/fragments/home/trending-contracts").Render(r.Context(), w)
	}
}

// HomeSidebarFragment returns the Top Assets + Fee Guide sidebar.
func (h *Handlers) HomeSidebarFragment(w http.ResponseWriter, r *http.Request) {
	network := networkFromRequest(r)
	data := mockHomeData(network)

	if err := fragments.HomeSidebar(data).Render(r.Context(), w); err != nil {
		h.Logger.Error("render fragment", "error", err)
		fragments.FragmentError("Could not load sidebar", "/fragments/home/sidebar").Render(r.Context(), w)
	}
}
