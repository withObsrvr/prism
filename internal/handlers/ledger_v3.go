package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/withObsrvr/prism/internal/gateway"
	pagesv2 "github.com/withObsrvr/prism/internal/templates/v2/pages"
	vmv2 "github.com/withObsrvr/prism/internal/templates/v2/viewmodel"
)

// LedgerDetailV3 renders the canonical ledger detail page from live evidence.
// Illustrative fixture values are scrubbed before any overlay runs, so an
// unavailable projection can never masquerade as evidence for this ledger.
func (h *Handlers) LedgerDetailV3(w http.ResponseWriter, r *http.Request) {
	sequence := r.PathValue("sequence")
	network := networkFromRequest(r)
	seq, err := strconv.ParseInt(sequence, 10, 64)
	if err != nil || seq <= 0 {
		http.Error(w, "invalid ledger sequence", http.StatusBadRequest)
		return
	}
	if h.Gateway == nil {
		http.Error(w, "ledger evidence unavailable", http.StatusServiceUnavailable)
		return
	}

	ctx := r.Context()
	full, err := h.Gateway.GetSilverLedgerFull(ctx, network, seq)
	if err != nil || full == nil {
		status := http.StatusBadGateway
		var apiErr *gateway.APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound {
			status = http.StatusNotFound
		}
		if h.Logger != nil {
			h.Logger.Warn("ledger v3: required ledger evidence unavailable", "sequence", seq, "error", err)
		}
		http.Error(w, "ledger not available", status)
		return
	}

	data := liveLedgerDetailV3Data(sequence, network)
	h.overlayLedgerV3Header(ctx, network, seq, &data)
	h.overlayLedgerV3Capacity(ctx, network, seq, &data)
	changes := h.overlayLedgerV3Changes(ctx, network, seq, &data)
	h.overlayLedgerV3Fees(ctx, network, seq, &data)

	// The composite endpoint is deliberately bounded. Ask the range endpoints
	// for the ledger's declared totals, then let the pane model disclose any
	// remaining server-side truncation rather than calling a sample complete.
	txs, ops := full.Transactions, full.Operations
	if full.Ledger.TransactionCount > len(txs) {
		if fetched, fetchErr := h.Gateway.GetTransactions(ctx, network, seq, seq, full.Ledger.TransactionCount, "asc"); fetchErr == nil && len(fetched) > len(txs) {
			txs = fetched
		}
	}
	if full.Ledger.OperationCount > len(ops) {
		if fetched, fetchErr := h.Gateway.GetOperations(ctx, network, seq, seq, full.Ledger.OperationCount); fetchErr == nil && len(fetched) > len(ops) {
			ops = fetched
		}
	}
	applyLedgerV3Ticks(&data, network, txs)
	h.overlayLedgerV3Panes(&data, network, txs, ops, full.Ledger.TransactionCount, full.Ledger.OperationCount, changes)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := pagesv2.LedgerDetailV3(data).Render(r.Context(), w); err != nil {
		if h.Logger != nil {
			h.Logger.Error("render ledger detail v3", "error", err)
		}
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// RedirectLedgerDetailV3 preserves links to the former candidate route.
func (h *Handlers) RedirectLedgerDetailV3(w http.ResponseWriter, r *http.Request) {
	target := "/v2/ledger/" + r.PathValue("sequence")
	if r.URL.RawQuery != "" {
		target += "?" + r.URL.RawQuery
	}
	http.Redirect(w, r, target, http.StatusPermanentRedirect)
}

var provLedgerV3Unavailable = vmv2.Provenance{
	Kind: vmv2.ProvenanceGap, Origin: "obsrvr Gateway",
	Note: "The evidence needed for this value was not available; Prism does not substitute fixture data.",
}

func liveLedgerDetailV3Data(sequence, network string) vmv2.LedgerDetailV3Data {
	d := mockLedgerDetailV3Data(sequence, network)
	d.CanonicalPath = fmt.Sprintf("/v2/ledger/%s", sequence)
	d.Header.Hash, d.Header.ClosedAt, d.Header.ClosedRelative = "—", "—", "—"
	d.Header.CloseTime, d.Header.ProtocolVersion, d.Header.Kicker = "—", "—", "Ledger evidence"
	d.Header.HeadlineLead, d.Header.HeadlineEmphasis, d.Header.HeadlineTrail = "Ledger", d.Header.Sequence, ""
	d.Header.HeadlineSource = provLedgerV3Unavailable
	d.Header.Badges = []vmv2.LedgerV3Badge{{Label: "Capacity unavailable"}, {Label: "Fees unavailable"}, {Label: "Failures unavailable"}}
	d.Header.TxTabCount, d.Header.StateTabCount = "—", "—"
	d.Lede = []string{`Prism is loading each interpretation from this ledger's recorded evidence. Values whose projections are unavailable are marked rather than estimated.`}
	for i := range d.Standing {
		d.Standing[i].Value, d.Standing[i].Detail, d.Standing[i].Dot = "Unavailable", "Evidence not available", "none"
		d.Standing[i].Source = provLedgerV3Unavailable
	}
	d.Strip.Ticks, d.Strip.Legend = nil, nil
	for i := range d.Strip.Foot {
		d.Strip.Foot[i].Value, d.Strip.Foot[i].Source = "—", provLedgerV3Unavailable
	}
	d.Strip.Note, d.Strip.Source = "Transaction evidence is unavailable.", provLedgerV3Unavailable
	d.Capacity.Meters = nil
	d.Capacity.Note = "<b>Capacity evidence is unavailable.</b> Prism will not infer utilization from unrelated ledgers."
	d.Fees = vmv2.LedgerV3Fees{Heading: "Fee evidence unavailable", Aside: "not recorded", ClearingLabel: "Clearing fee", ClearingValue: "—", ClearingUnit: "", BaseFee: "—", Multiple: "—", TotalCollected: "—", HighestBidNote: "Fee evidence is unavailable.", CannotTitle: "What a fee buys", CannotBody: "Excluded transactions are not recorded in a closed ledger.", Source: provLedgerV3Unavailable, ExcludedSource: provLedgerV3Unavailable}
	d.Failures = vmv2.LedgerV3Failures{Aside: "not available", Intro: "Failure evidence is unavailable.", Note: "No failure interpretation is shown without result evidence.", Source: provLedgerV3Unavailable}
	d.Changes = vmv2.LedgerV3Changes{Total: "—", Note: "State-change evidence is unavailable.", Source: provLedgerV3Unavailable}
	d.Chain = vmv2.LedgerV3Chain{Heading: "Where it sits", Aside: "context unavailable", Intro: "Neighboring-ledger evidence is not yet available for this view.", Note: "Prism does not infer a trend from one ledger.", Source: provLedgerV3Unavailable}
	d.Notes = nil
	for i := range d.Rail.Groups {
		for j := range d.Rail.Groups[i].Rows {
			d.Rail.Groups[i].Rows[j].Value, d.Rail.Groups[i].Rows[j].IsGap = "—", true
			if d.Rail.Groups[i].Heading == "Header" {
				switch d.Rail.Groups[i].Rows[j].Label {
				case "Sequence":
					d.Rail.Groups[i].Rows[j].Value, d.Rail.Groups[i].Rows[j].IsGap = d.Header.Sequence, false
				case "Previous":
					d.Rail.Groups[i].Rows[j].Value, d.Rail.Groups[i].Rows[j].IsGap = d.Header.PrevSequence, false
				}
			}
		}
	}
	d.Rail.Title, d.Rail.Subtitle = "Ledger "+d.Header.Sequence, "Awaiting evidence"
	d.Rail.TOC[2].Label = "Fees"
	d.TxPane = vmv2.LedgerV3TxPane{Title: "Transactions", Intro: "Transaction evidence is unavailable.", SaidLead: "No transaction evidence available.", TotalLabel: "0 transactions", ShownLabel: "0 shown"}
	d.StatePane = vmv2.LedgerV3StatePane{Title: "State changes", Intro: "State-change evidence is unavailable.", SaidLead: "No state-change evidence available.", TotalLabel: "0 changes", ShownLabel: "0 shown"}
	return d
}
