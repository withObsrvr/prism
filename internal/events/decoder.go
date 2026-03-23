package events

import (
	"fmt"
	"html"
)

// DecodedEvent holds a human-readable interpretation of a Soroban event.
type DecodedEvent struct {
	Type         string   // "transfer", "approve", "mint", "burn", "swap"
	Summary      string   // Human-readable one-liner
	Participants []string // Addresses involved
	Amount       string   // Formatted amount if applicable
	Asset        string   // Asset code if applicable
	PairIn       string   // For swaps: e.g. "12,400 XLM"
	PairOut      string   // For swaps: e.g. "1,202 USDC"
	Router       string   // For swaps: DEX name
	Raw          string   // Original data for detail panel
}

// RawEvent represents the input data for the decoder.
type RawEvent struct {
	Type         string // event type/topic[0]: "transfer", "approve", "mint", "burn", "swap"
	ContractName string // human name of contract, e.g. "USDC Token"
	From         string // source address (short form)
	To           string // destination address (short form)
	Spender      string // spender address for approve events
	Amount       string // formatted amount string, e.g. "2,500.00"
	Asset        string // asset code, e.g. "USDC"
	PairIn       string // for swaps: input asset description e.g. "12,400 XLM"
	PairOut      string // for swaps: output asset description e.g. "1,202 USDC"
	Router       string // for swaps: DEX name e.g. "Soroswap"
	Raw          string // original JSON for detail panel
}

// Decode takes a RawEvent and returns a DecodedEvent with human-readable summary
// and pre-rendered HTML suitable for TopicsHTML/DataHTML template fields.
// Returns nil if the event type is not recognized (caller should fall back to raw display).
func Decode(e RawEvent) *DecodedEvent {
	switch e.Type {
	case "transfer":
		return decodeTransfer(e)
	case "approve":
		return decodeApprove(e)
	case "mint":
		return decodeMint(e)
	case "burn":
		return decodeBurn(e)
	case "swap":
		return decodeSwap(e)
	default:
		return nil
	}
}

// TopicsHTML returns pre-rendered HTML matching the existing badge+pill format
// used by FirehoseEvent.TopicsHTML and TxEvent.DataHTML.
func (d *DecodedEvent) TopicsHTML() string {
	switch d.Type {
	case "transfer":
		return d.transferHTML()
	case "approve":
		return d.approveHTML()
	case "mint":
		return d.mintHTML()
	case "burn":
		return d.burnHTML()
	case "swap":
		return d.swapHTML()
	default:
		return ""
	}
}

func decodeTransfer(e RawEvent) *DecodedEvent {
	summary := fmt.Sprintf("Sent %s %s → %s", e.Amount, e.Asset, e.To)
	return &DecodedEvent{
		Type:         "transfer",
		Summary:      summary,
		Participants: []string{e.From, e.To},
		Amount:       e.Amount,
		Asset:        e.Asset,
		Raw:          e.Raw,
	}
}

func decodeApprove(e RawEvent) *DecodedEvent {
	summary := fmt.Sprintf("Approved %s to spend %s %s", e.Spender, e.Amount, e.Asset)
	return &DecodedEvent{
		Type:         "approve",
		Summary:      summary,
		Participants: []string{e.From, e.Spender},
		Amount:       e.Amount,
		Asset:        e.Asset,
		Raw:          e.Raw,
	}
}

func decodeMint(e RawEvent) *DecodedEvent {
	summary := fmt.Sprintf("Minted %s %s to %s", e.Amount, e.Asset, e.To)
	return &DecodedEvent{
		Type:         "mint",
		Summary:      summary,
		Participants: []string{e.To},
		Amount:       e.Amount,
		Asset:        e.Asset,
		Raw:          e.Raw,
	}
}

func decodeBurn(e RawEvent) *DecodedEvent {
	summary := fmt.Sprintf("Burned %s %s from %s", e.Amount, e.Asset, e.From)
	return &DecodedEvent{
		Type:         "burn",
		Summary:      summary,
		Participants: []string{e.From},
		Amount:       e.Amount,
		Asset:        e.Asset,
		Raw:          e.Raw,
	}
}

func decodeSwap(e RawEvent) *DecodedEvent {
	summary := fmt.Sprintf("Swapped %s for %s via %s", e.PairIn, e.PairOut, e.Router)
	return &DecodedEvent{
		Type:         "swap",
		Summary:      summary,
		Participants: []string{e.From},
		Amount:       e.PairOut,
		PairIn:       e.PairIn,
		PairOut:      e.PairOut,
		Router:       e.Router,
		Raw:          e.Raw,
	}
}

// HTML rendering helpers — output matches existing TopicsHTML/DataHTML pill format.

func (d *DecodedEvent) transferHTML() string {
	return fmt.Sprintf(`<div class="flex items-center gap-1.5 flex-wrap">`+
		`<span class="rounded bg-surface-subtle px-1.5 py-0.5 font-mono text-[10px] text-text-body">Sent %s %s</span>`+
		`<span class="rounded bg-surface-subtle px-1.5 py-0.5 font-mono text-[10px] text-text-body">→ <span class="text-violet-600 dark:text-violet-400">%s</span></span>`+
		`</div>`,
		html.EscapeString(d.Amount), html.EscapeString(d.Asset),
		html.EscapeString(d.Participants[1]))
}

func (d *DecodedEvent) approveHTML() string {
	return fmt.Sprintf(`<div class="flex items-center gap-1.5 flex-wrap">`+
		`<span class="rounded bg-surface-subtle px-1.5 py-0.5 font-mono text-[10px] text-text-body">Approved <span class="text-violet-600 dark:text-violet-400">%s</span></span>`+
		`<span class="rounded bg-surface-subtle px-1.5 py-0.5 font-mono text-[10px] text-text-body">to spend %s %s</span>`+
		`</div>`,
		html.EscapeString(d.Participants[1]),
		html.EscapeString(d.Amount), html.EscapeString(d.Asset))
}

func (d *DecodedEvent) mintHTML() string {
	return fmt.Sprintf(`<div class="flex items-center gap-1.5 flex-wrap">`+
		`<span class="rounded bg-emerald-50 dark:bg-emerald-950/30 px-1.5 py-0.5 font-mono text-[10px] font-semibold text-emerald-700 dark:text-emerald-400">Minted +%s %s</span>`+
		`<span class="rounded bg-surface-subtle px-1.5 py-0.5 font-mono text-[10px] text-text-body">→ <span class="text-violet-600 dark:text-violet-400">%s</span></span>`+
		`</div>`,
		html.EscapeString(d.Amount), html.EscapeString(d.Asset),
		html.EscapeString(d.Participants[0]))
}

func (d *DecodedEvent) burnHTML() string {
	return fmt.Sprintf(`<div class="flex items-center gap-1.5 flex-wrap">`+
		`<span class="rounded bg-red-50 dark:bg-red-950/30 px-1.5 py-0.5 font-mono text-[10px] font-semibold text-red-600 dark:text-red-400">Burned %s %s</span>`+
		`<span class="rounded bg-surface-subtle px-1.5 py-0.5 font-mono text-[10px] text-text-body">from <span class="text-violet-600 dark:text-violet-400">%s</span></span>`+
		`</div>`,
		html.EscapeString(d.Amount), html.EscapeString(d.Asset),
		html.EscapeString(d.Participants[0]))
}

func (d *DecodedEvent) swapHTML() string {
	return fmt.Sprintf(`<div class="flex items-center gap-1.5 flex-wrap">`+
		`<span class="rounded bg-red-50 dark:bg-red-950/30 px-1.5 py-0.5 font-mono text-[10px] text-red-600 dark:text-red-400">−%s</span>`+
		`<span class="rounded bg-emerald-50 dark:bg-emerald-950/30 px-1.5 py-0.5 font-mono text-[10px] font-semibold text-emerald-700 dark:text-emerald-400">+%s</span>`+
		`<span class="rounded bg-surface-subtle px-1.5 py-0.5 font-mono text-[10px] text-text-body">via %s</span>`+
		`</div>`,
		html.EscapeString(d.PairIn),
		html.EscapeString(d.PairOut),
		html.EscapeString(d.Router))
}
