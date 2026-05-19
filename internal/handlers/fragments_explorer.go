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
func (h *Handlers) buildTxFragmentData(r *http.Request, network, hash, shortHash string) *pages.TxReceiptData {
	if !h.useLiveData(r) {
		return nil
	}
	ctx, cancel := context.WithTimeout(r.Context(), txPageGatewayTimeout)
	defer cancel()
	started := time.Now()
	data, err := h.buildTxReceiptData(r.Clone(ctx), network, hash, shortHash)
	if err != nil {
		h.Logger.Warn("live tx data failed, falling back to mock", "error", err, "hash", shortHash, "duration", time.Since(started))
		return nil
	}
	return &data
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

	data := h.buildTxFragmentData(r, network, hash, shortHash)
	if data == nil {
		mock := mockTxReceiptData(hash, shortHash)
		data = &mock
	}

	if err := pagesv2.TxReceiptHeroFragment(*data).Render(r.Context(), w); err != nil {
		h.renderFragmentError(w, r, "Could not load transaction hero", err)
	}
}

// TxV2DetailFragment returns the v2 transaction operations/events/balance/raw section.
func (h *Handlers) TxV2DetailFragment(w http.ResponseWriter, r *http.Request) {
	hash := r.PathValue("hash")
	network := networkFromRequest(r)
	shortHash := txShortHash(hash)

	data := h.buildTxFragmentData(r, network, hash, shortHash)
	if data == nil {
		mock := mockTxReceiptData(hash, shortHash)
		data = &mock
	}

	if err := pagesv2.TxReceiptDetailFragment(*data).Render(r.Context(), w); err != nil {
		h.renderFragmentError(w, r, "Could not load transaction details", err)
	}
}

// TxV2SidebarFragment returns the v2 transaction summary sidebar.
func (h *Handlers) TxV2SidebarFragment(w http.ResponseWriter, r *http.Request) {
	hash := r.PathValue("hash")
	network := networkFromRequest(r)
	shortHash := txShortHash(hash)

	data := h.buildTxFragmentData(r, network, hash, shortHash)
	if data == nil {
		mock := mockTxReceiptData(hash, shortHash)
		data = &mock
	}

	if err := pagesv2.TxReceiptSidebarFragment(*data).Render(r.Context(), w); err != nil {
		h.renderFragmentError(w, r, "Could not load transaction sidebar", err)
	}
}

// TxOverviewFragment returns the transaction key-value overview.
func (h *Handlers) TxOverviewFragment(w http.ResponseWriter, r *http.Request) {
	hash := r.PathValue("hash")
	network := networkFromRequest(r)
	shortHash := txShortHash(hash)

	data := h.buildTxFragmentData(r, network, hash, shortHash)
	if data == nil {
		mock := mockTxReceiptData(hash, shortHash)
		data = &mock
	}

	if err := fragments.TxOverview(*data).Render(r.Context(), w); err != nil {
		h.renderFragmentError(w, r, "Could not load overview", err)
	}
}

// TxOperationsFragment returns the operations table.
func (h *Handlers) TxOperationsFragment(w http.ResponseWriter, r *http.Request) {
	hash := r.PathValue("hash")
	network := networkFromRequest(r)
	shortHash := txShortHash(hash)

	data := h.buildTxFragmentData(r, network, hash, shortHash)
	if data == nil {
		mock := mockTxReceiptData(hash, shortHash)
		data = &mock
	}

	if err := fragments.TxOperations(data.Operations).Render(r.Context(), w); err != nil {
		h.renderFragmentError(w, r, "Could not load operations", err)
	}
}

// TxEventsFragment returns the events table.
func (h *Handlers) TxEventsFragment(w http.ResponseWriter, r *http.Request) {
	hash := r.PathValue("hash")
	network := networkFromRequest(r)
	shortHash := txShortHash(hash)

	data := h.buildTxFragmentData(r, network, hash, shortHash)
	if data == nil {
		mock := mockTxReceiptData(hash, shortHash)
		data = &mock
	}

	if err := fragments.TxEvents(data.Events).Render(r.Context(), w); err != nil {
		h.renderFragmentError(w, r, "Could not load events", err)
	}
}

// TxBalanceChangesFragment returns the balance changes table.
func (h *Handlers) TxBalanceChangesFragment(w http.ResponseWriter, r *http.Request) {
	hash := r.PathValue("hash")
	network := networkFromRequest(r)
	shortHash := txShortHash(hash)

	data := h.buildTxFragmentData(r, network, hash, shortHash)
	if data == nil {
		mock := mockTxReceiptData(hash, shortHash)
		data = &mock
	}

	if err := fragments.TxBalanceChanges(data.BalanceChanges).Render(r.Context(), w); err != nil {
		h.renderFragmentError(w, r, "Could not load balance changes", err)
	}
}

// TxEffectsFragment returns the effects table.
func (h *Handlers) TxEffectsFragment(w http.ResponseWriter, r *http.Request) {
	hash := r.PathValue("hash")
	network := networkFromRequest(r)
	shortHash := txShortHash(hash)

	data := h.buildTxFragmentData(r, network, hash, shortHash)
	if data == nil {
		mock := mockTxReceiptData(hash, shortHash)
		data = &mock
	}

	if err := fragments.TxEffects(data.Effects).Render(r.Context(), w); err != nil {
		h.renderFragmentError(w, r, "Could not load effects", err)
	}
}

// TxTimelineFragment returns the "What Happened" timeline.
func (h *Handlers) TxTimelineFragment(w http.ResponseWriter, r *http.Request) {
	hash := r.PathValue("hash")
	network := networkFromRequest(r)
	shortHash := txShortHash(hash)

	data := h.buildTxFragmentData(r, network, hash, shortHash)
	if data == nil {
		mock := mockTxReceiptData(hash, shortHash)
		data = &mock
	}

	if err := fragments.TxTimeline(data.Timeline).Render(r.Context(), w); err != nil {
		h.renderFragmentError(w, r, "Could not load timeline", err)
	}
}

// TxStateChangesFragment returns the Soroban state changes table.
func (h *Handlers) TxStateChangesFragment(w http.ResponseWriter, r *http.Request) {
	hash := r.PathValue("hash")
	network := networkFromRequest(r)
	shortHash := txShortHash(hash)

	data := h.buildTxFragmentData(r, network, hash, shortHash)
	if data == nil {
		mock := mockTxReceiptData(hash, shortHash)
		data = &mock
	}

	if err := fragments.TxStateChanges(data.StateChanges).Render(r.Context(), w); err != nil {
		h.renderFragmentError(w, r, "Could not load state changes", err)
	}
}

// ── Ledger Detail Fragments ──

// buildLedgerFragmentData fetches ledger detail data for fragment rendering.
// Returns nil if live data is unavailable or not requested.
func (h *Handlers) buildLedgerFragmentData(r *http.Request, network, sequence string) *pages.LedgerDetailData {
	if !h.useLiveData(r) {
		return nil
	}
	data, err := h.buildLedgerDetailData(r, network, sequence)
	if err != nil {
		h.Logger.Warn("live ledger data failed, falling back to mock", "error", err, "sequence", sequence)
		return nil
	}
	return &data
}

// LedgerTxsFragment returns the ledger transactions table.
func (h *Handlers) LedgerTxsFragment(w http.ResponseWriter, r *http.Request) {
	sequence := r.PathValue("sequence")
	network := networkFromRequest(r)

	data := h.buildLedgerFragmentData(r, network, sequence)
	if data == nil {
		mock := mockLedgerDetailData(sequence)
		data = &mock
	}

	if err := fragments.LedgerTxs(*data).Render(r.Context(), w); err != nil {
		h.renderFragmentError(w, r, "Could not load transactions", err)
	}
}

// LedgerOpsAndFeesFragment returns the operation breakdown + fee distribution.
func (h *Handlers) LedgerOpsAndFeesFragment(w http.ResponseWriter, r *http.Request) {
	sequence := r.PathValue("sequence")
	network := networkFromRequest(r)

	data := h.buildLedgerFragmentData(r, network, sequence)
	if data == nil {
		mock := mockLedgerDetailData(sequence)
		data = &mock
	}

	if err := fragments.LedgerOpsAndFees(*data).Render(r.Context(), w); err != nil {
		h.renderFragmentError(w, r, "Could not load operation breakdown", err)
	}
}

// LedgerSorobanFragment returns the Soroban runtime details.
func (h *Handlers) LedgerSorobanFragment(w http.ResponseWriter, r *http.Request) {
	sequence := r.PathValue("sequence")
	network := networkFromRequest(r)

	data := h.buildLedgerFragmentData(r, network, sequence)
	if data == nil {
		mock := mockLedgerDetailData(sequence)
		data = &mock
	}

	if err := fragments.LedgerSoroban(*data).Render(r.Context(), w); err != nil {
		h.renderFragmentError(w, r, "Could not load Soroban runtime", err)
	}
}
