package handlers

import (
	"net/http"

	"github.com/withObsrvr/prism/internal/templates/fragments"
)

// ── Transaction Receipt Fragments ──

// TxOverviewFragment returns the transaction key-value overview.
func (h *Handlers) TxOverviewFragment(w http.ResponseWriter, r *http.Request) {
	hash := r.PathValue("hash")
	shortHash := hash
	if len(hash) > 8 {
		shortHash = hash[:4] + "…" + hash[len(hash)-4:]
	}
	data := mockTxReceiptData(hash, shortHash)

	if err := fragments.TxOverview(data).Render(r.Context(), w); err != nil {
		h.Logger.Error("render fragment", "error", err)
		fragments.FragmentError("Could not load overview", r.URL.Path).Render(r.Context(), w)
	}
}

// TxOperationsFragment returns the operations table.
func (h *Handlers) TxOperationsFragment(w http.ResponseWriter, r *http.Request) {
	hash := r.PathValue("hash")
	shortHash := hash
	if len(hash) > 8 {
		shortHash = hash[:4] + "…" + hash[len(hash)-4:]
	}
	data := mockTxReceiptData(hash, shortHash)

	if err := fragments.TxOperations(data.Operations).Render(r.Context(), w); err != nil {
		h.Logger.Error("render fragment", "error", err)
		fragments.FragmentError("Could not load operations", r.URL.Path).Render(r.Context(), w)
	}
}

// TxEventsFragment returns the events table.
func (h *Handlers) TxEventsFragment(w http.ResponseWriter, r *http.Request) {
	hash := r.PathValue("hash")
	shortHash := hash
	if len(hash) > 8 {
		shortHash = hash[:4] + "…" + hash[len(hash)-4:]
	}
	data := mockTxReceiptData(hash, shortHash)

	if err := fragments.TxEvents(data.Events).Render(r.Context(), w); err != nil {
		h.Logger.Error("render fragment", "error", err)
		fragments.FragmentError("Could not load events", r.URL.Path).Render(r.Context(), w)
	}
}

// TxBalanceChangesFragment returns the balance changes table.
func (h *Handlers) TxBalanceChangesFragment(w http.ResponseWriter, r *http.Request) {
	hash := r.PathValue("hash")
	shortHash := hash
	if len(hash) > 8 {
		shortHash = hash[:4] + "…" + hash[len(hash)-4:]
	}
	data := mockTxReceiptData(hash, shortHash)

	if err := fragments.TxBalanceChanges(data.BalanceChanges).Render(r.Context(), w); err != nil {
		h.Logger.Error("render fragment", "error", err)
		fragments.FragmentError("Could not load balance changes", r.URL.Path).Render(r.Context(), w)
	}
}

// TxStateChangesFragment returns the Soroban state changes table.
func (h *Handlers) TxStateChangesFragment(w http.ResponseWriter, r *http.Request) {
	hash := r.PathValue("hash")
	shortHash := hash
	if len(hash) > 8 {
		shortHash = hash[:4] + "…" + hash[len(hash)-4:]
	}
	data := mockTxReceiptData(hash, shortHash)

	if err := fragments.TxStateChanges(data.StateChanges).Render(r.Context(), w); err != nil {
		h.Logger.Error("render fragment", "error", err)
		fragments.FragmentError("Could not load state changes", r.URL.Path).Render(r.Context(), w)
	}
}

// ── Ledger Detail Fragments ──

// LedgerTxsFragment returns the ledger transactions table.
func (h *Handlers) LedgerTxsFragment(w http.ResponseWriter, r *http.Request) {
	sequence := r.PathValue("sequence")
	data := mockLedgerDetailData(sequence)

	if err := fragments.LedgerTxs(data).Render(r.Context(), w); err != nil {
		h.Logger.Error("render fragment", "error", err)
		fragments.FragmentError("Could not load transactions", r.URL.Path).Render(r.Context(), w)
	}
}

// LedgerOpsAndFeesFragment returns the operation breakdown + fee distribution.
func (h *Handlers) LedgerOpsAndFeesFragment(w http.ResponseWriter, r *http.Request) {
	sequence := r.PathValue("sequence")
	data := mockLedgerDetailData(sequence)

	if err := fragments.LedgerOpsAndFees(data).Render(r.Context(), w); err != nil {
		h.Logger.Error("render fragment", "error", err)
		fragments.FragmentError("Could not load operation breakdown", r.URL.Path).Render(r.Context(), w)
	}
}

// LedgerSorobanFragment returns the Soroban runtime details.
func (h *Handlers) LedgerSorobanFragment(w http.ResponseWriter, r *http.Request) {
	sequence := r.PathValue("sequence")
	data := mockLedgerDetailData(sequence)

	if err := fragments.LedgerSoroban(data).Render(r.Context(), w); err != nil {
		h.Logger.Error("render fragment", "error", err)
		fragments.FragmentError("Could not load Soroban runtime", r.URL.Path).Render(r.Context(), w)
	}
}
