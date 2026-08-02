package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/withObsrvr/prism/internal/templates/fragments"
	"github.com/withObsrvr/prism/internal/templates/pages"
	pagesv2 "github.com/withObsrvr/prism/internal/templates/v2/pages"
)

// ── Transaction Receipt Fragments ──

// buildTxFragmentData fetches transaction detail data for fragment rendering.
// Returns nil if live data is unavailable or not requested.
// txFragmentResult separates the two reasons a page might not have live data.
// Collapsing them into a nil pointer is what made a failed gateway render as a
// confident, fabricated receipt: the caller could not tell "we are deliberately
// showing fixtures" from "we tried to load this transaction and could not".
type txFragmentResult struct {
	Data *pages.TxReceiptData
	// Demo means fixtures were asked for, by data_source=mock or ?mock=true.
	// Legitimate, but it must be visible on the page.
	Demo bool
	// Err means live data was wanted and could not be loaded. Never render
	// fixtures for this: say the load failed.
	Err error
}

func (h *Handlers) buildTxFragmentData(r *http.Request, network, hash, shortHash string) txFragmentResult {
	if h.useExplicitMockData(r) {
		mock := mockTxReceiptData(hash, shortHash)
		mock.Demo = true
		return txFragmentResult{Data: &mock, Demo: true}
	}
	if h.Gateway == nil {
		// No gateway configured at all. Fixtures are the only thing available,
		// and the page has to say so rather than imply this is the network.
		mock := mockTxReceiptData(hash, shortHash)
		mock.Demo = true
		return txFragmentResult{Data: &mock, Demo: true}
	}
	ctx, cancel := context.WithTimeout(r.Context(), txPageGatewayTimeout)
	defer cancel()
	started := time.Now()
	data, err := h.buildTxReceiptData(r.Clone(ctx), network, hash, shortHash)
	if err != nil {
		h.Logger.Warn("live tx data failed", "error", err, "hash", shortHash, "duration", time.Since(started))
		return txFragmentResult{Err: err}
	}
	return txFragmentResult{Data: &data}
}

func txShortHash(hash string) string {
	if len(hash) > 8 {
		return hash[:4] + "…" + hash[len(hash)-4:]
	}
	return hash
}

// TxV2HeroFragment returns the v2 transaction hero/archetype section.
func (h *Handlers) TxV2HeroFragment(w http.ResponseWriter, r *http.Request) {
	hash := r.PathValue("hash")
	network := networkFromRequest(r)
	shortHash := txShortHash(hash)

	res := h.buildTxFragmentData(r, network, hash, shortHash)
	if res.Err != nil {
		h.renderFragmentError(w, r, "Could not load transaction hero", res.Err)
		return
	}

	if err := pagesv2.TxReceiptHeroFragment(*res.Data).Render(r.Context(), w); err != nil {
		h.renderFragmentError(w, r, "Could not load transaction hero", err)
	}
}

// TxV2DetailFragment returns the v2 transaction operations/events/balance/raw section.
func (h *Handlers) TxV2DetailFragment(w http.ResponseWriter, r *http.Request) {
	hash := r.PathValue("hash")
	network := networkFromRequest(r)
	shortHash := txShortHash(hash)

	res := h.buildTxFragmentData(r, network, hash, shortHash)
	if res.Err != nil {
		h.renderFragmentError(w, r, "Could not load transaction details", res.Err)
		return
	}

	if err := pagesv2.TxReceiptDetailFragment(*res.Data).Render(r.Context(), w); err != nil {
		h.renderFragmentError(w, r, "Could not load transaction details", err)
	}
}

// TxV2SidebarFragment returns the v2 transaction summary sidebar.
func (h *Handlers) TxV2SidebarFragment(w http.ResponseWriter, r *http.Request) {
	hash := r.PathValue("hash")
	network := networkFromRequest(r)
	shortHash := txShortHash(hash)

	res := h.buildTxFragmentData(r, network, hash, shortHash)
	if res.Err != nil {
		h.renderFragmentError(w, r, "Could not load transaction sidebar", res.Err)
		return
	}

	if err := pagesv2.TxReceiptSidebarFragment(*res.Data).Render(r.Context(), w); err != nil {
		h.renderFragmentError(w, r, "Could not load transaction sidebar", err)
	}
}

// TxOverviewFragment returns the transaction key-value overview.
func (h *Handlers) TxOverviewFragment(w http.ResponseWriter, r *http.Request) {
	hash := r.PathValue("hash")
	network := networkFromRequest(r)
	shortHash := txShortHash(hash)

	res := h.buildTxFragmentData(r, network, hash, shortHash)
	if res.Err != nil {
		h.renderFragmentError(w, r, "Could not load overview", res.Err)
		return
	}

	if err := fragments.TxOverview(*res.Data).Render(r.Context(), w); err != nil {
		h.renderFragmentError(w, r, "Could not load overview", err)
	}
}

// TxOperationsFragment returns the operations table.
func (h *Handlers) TxOperationsFragment(w http.ResponseWriter, r *http.Request) {
	hash := r.PathValue("hash")
	network := networkFromRequest(r)
	shortHash := txShortHash(hash)

	res := h.buildTxFragmentData(r, network, hash, shortHash)
	if res.Err != nil {
		h.renderFragmentError(w, r, "Could not load operations", res.Err)
		return
	}

	if err := fragments.TxOperations(res.Data.Operations).Render(r.Context(), w); err != nil {
		h.renderFragmentError(w, r, "Could not load operations", err)
	}
}

// TxEventsFragment returns the events table.
func (h *Handlers) TxEventsFragment(w http.ResponseWriter, r *http.Request) {
	hash := r.PathValue("hash")
	network := networkFromRequest(r)
	shortHash := txShortHash(hash)

	res := h.buildTxFragmentData(r, network, hash, shortHash)
	if res.Err != nil {
		h.renderFragmentError(w, r, "Could not load events", res.Err)
		return
	}

	if err := fragments.TxEvents(res.Data.Events).Render(r.Context(), w); err != nil {
		h.renderFragmentError(w, r, "Could not load events", err)
	}
}

// TxBalanceChangesFragment returns the balance changes table.
func (h *Handlers) TxBalanceChangesFragment(w http.ResponseWriter, r *http.Request) {
	hash := r.PathValue("hash")
	network := networkFromRequest(r)
	shortHash := txShortHash(hash)

	res := h.buildTxFragmentData(r, network, hash, shortHash)
	if res.Err != nil {
		h.renderFragmentError(w, r, "Could not load balance changes", res.Err)
		return
	}

	if err := fragments.TxBalanceChanges(res.Data.BalanceChanges).Render(r.Context(), w); err != nil {
		h.renderFragmentError(w, r, "Could not load balance changes", err)
	}
}

// TxEffectsFragment returns the effects table.
func (h *Handlers) TxEffectsFragment(w http.ResponseWriter, r *http.Request) {
	hash := r.PathValue("hash")
	network := networkFromRequest(r)
	shortHash := txShortHash(hash)

	res := h.buildTxFragmentData(r, network, hash, shortHash)
	if res.Err != nil {
		h.renderFragmentError(w, r, "Could not load effects", res.Err)
		return
	}

	if err := fragments.TxEffects(res.Data.Effects).Render(r.Context(), w); err != nil {
		h.renderFragmentError(w, r, "Could not load effects", err)
	}
}

// TxTimelineFragment returns the "What Happened" timeline.
func (h *Handlers) TxTimelineFragment(w http.ResponseWriter, r *http.Request) {
	hash := r.PathValue("hash")
	network := networkFromRequest(r)
	shortHash := txShortHash(hash)

	res := h.buildTxFragmentData(r, network, hash, shortHash)
	if res.Err != nil {
		h.renderFragmentError(w, r, "Could not load timeline", res.Err)
		return
	}

	if err := fragments.TxTimeline(res.Data.Timeline).Render(r.Context(), w); err != nil {
		h.renderFragmentError(w, r, "Could not load timeline", err)
	}
}

// TxStateChangesFragment returns the Soroban state changes table.
func (h *Handlers) TxStateChangesFragment(w http.ResponseWriter, r *http.Request) {
	hash := r.PathValue("hash")
	network := networkFromRequest(r)
	shortHash := txShortHash(hash)

	res := h.buildTxFragmentData(r, network, hash, shortHash)
	if res.Err != nil {
		h.renderFragmentError(w, r, "Could not load state changes", res.Err)
		return
	}

	if err := fragments.TxStateChanges(res.Data.StateChanges).Render(r.Context(), w); err != nil {
		h.renderFragmentError(w, r, "Could not load state changes", err)
	}
}

// ── Ledger Detail Fragments ──

// buildLedgerFragmentData fetches ledger detail data for fragment rendering.
// Returns nil if live data is unavailable or not requested.
// ledgerFragmentResult mirrors txFragmentResult: deliberate fixtures and a
// failed load are different outcomes and must not share a representation.
type ledgerFragmentResult struct {
	Data *pages.LedgerDetailData
	Demo bool
	Err  error
}

func (h *Handlers) buildLedgerFragmentData(r *http.Request, network, sequence string) ledgerFragmentResult {
	if h.useExplicitMockData(r) || h.Gateway == nil {
		mock := mockLedgerDetailData(sequence)
		mock.Demo = true
		return ledgerFragmentResult{Data: &mock, Demo: true}
	}
	data, err := h.buildLedgerDetailData(r, network, sequence)
	if err != nil {
		h.Logger.Warn("live ledger data failed", "error", err, "sequence", sequence)
		return ledgerFragmentResult{Err: err}
	}
	return ledgerFragmentResult{Data: &data}
}

// LedgerTxsFragment returns the ledger transactions table.
func (h *Handlers) LedgerTxsFragment(w http.ResponseWriter, r *http.Request) {
	sequence := r.PathValue("sequence")
	network := networkFromRequest(r)

	res := h.buildLedgerFragmentData(r, network, sequence)
	if res.Err != nil {
		h.renderFragmentError(w, r, "Could not load transactions", res.Err)
		return
	}

	if err := fragments.LedgerTxs(*res.Data).Render(r.Context(), w); err != nil {
		h.renderFragmentError(w, r, "Could not load transactions", err)
	}
}

// LedgerOpsAndFeesFragment returns the operation breakdown + fee distribution.
func (h *Handlers) LedgerOpsAndFeesFragment(w http.ResponseWriter, r *http.Request) {
	sequence := r.PathValue("sequence")
	network := networkFromRequest(r)

	res := h.buildLedgerFragmentData(r, network, sequence)
	if res.Err != nil {
		h.renderFragmentError(w, r, "Could not load operation breakdown", res.Err)
		return
	}

	if err := fragments.LedgerOpsAndFees(*res.Data).Render(r.Context(), w); err != nil {
		h.renderFragmentError(w, r, "Could not load operation breakdown", err)
	}
}

// LedgerSorobanFragment returns the Soroban runtime details.
func (h *Handlers) LedgerSorobanFragment(w http.ResponseWriter, r *http.Request) {
	sequence := r.PathValue("sequence")
	network := networkFromRequest(r)

	res := h.buildLedgerFragmentData(r, network, sequence)
	if res.Err != nil {
		h.renderFragmentError(w, r, "Could not load Soroban runtime", res.Err)
		return
	}

	if err := fragments.LedgerSoroban(*res.Data).Render(r.Context(), w); err != nil {
		h.renderFragmentError(w, r, "Could not load Soroban runtime", err)
	}
}
