package viewmodelv2

import (
	"fmt"
	"html"
	"strconv"
	"strings"

	legacy "github.com/withObsrvr/prism/internal/templates/pages"
)

type TxHeroKind string

const (
	TxHeroValueFlow   TxHeroKind = "value_flow"
	TxHeroStateChange TxHeroKind = "state_change"
	TxHeroLifecycle   TxHeroKind = "lifecycle"
	TxHeroFailure     TxHeroKind = "failure"
	TxHeroOperations  TxHeroKind = "operations"
	TxHeroGenericCall TxHeroKind = "generic_call"
)

type TxHeroModel struct {
	Kind         TxHeroKind
	Status       string
	TitleHTML    string
	SubtitleHTML string
	Tags         []string
	Meta         TxHeroMeta

	ValueFlow   *TxValueFlowHero
	StateChange *TxStateChangeHero
	Lifecycle   *TxLifecycleHero
	Failure     *TxFailureHero
	Operations  *TxOperationsHero
	GenericCall *TxGenericCallHero
}

type TxHeroMeta struct {
	Fee       string
	Status    string
	Ledger    string
	Resources string
}

type TxValueFlowHero struct {
	FromLabel   string
	FromAddress string
	ToLabel     string
	ToAddress   string
	Protocol    string
	Route       string
	SentAmount  string
	Received    string
	Rate        string
	Slippage    string
}

type TxStateChangeHero struct {
	Caller      string
	Contract    string
	Function    string
	SummaryHTML string
	Entries     []TxStateEntry
}

type TxStateEntry struct {
	Key        string
	EntryType  string
	ChangeType string
	Before     string
	After      string
}

type TxLifecycleHero struct {
	Verb        string
	Actor       string
	Entity      string
	SummaryHTML string
	Entries     []TxStateEntry
}

type TxFailureHero struct {
	Actor       string
	Contract    string
	Function    string
	ErrorCode   string
	SummaryHTML string
	Frames      []TxFailureFrame
}

type TxFailureFrame struct {
	Name   string
	Detail string
	Status string
	Failed bool
}

type TxGenericCallHero struct {
	Actor       string
	Contract    string
	Function    string
	SummaryHTML string
}

type TxOperationsHero struct {
	Actor       string
	Count       int
	SummaryHTML string
}

func BuildTxHero(data legacy.TxReceiptData) TxHeroModel {
	facts := factsFromReceipt(data)
	kind := classifyTxHero(facts)
	model := TxHeroModel{
		Kind:         kind,
		Status:       statusLabel(facts.Successful),
		TitleHTML:    titleHTML(kind, facts),
		SubtitleHTML: subtitleHTML(kind, facts),
		Tags:         tags(facts),
		Meta: TxHeroMeta{
			Fee:       firstNonEmpty(data.FeePaidXLM, data.FeePaid),
			Status:    statusLabel(facts.Successful),
			Ledger:    data.Ledger,
			Resources: firstNonEmpty(data.SorobanCPU, "—"),
		},
	}
	switch kind {
	case TxHeroFailure:
		model.Failure = buildFailureHero(facts)
	case TxHeroLifecycle:
		model.Lifecycle = buildLifecycleHero(facts)
	case TxHeroStateChange:
		model.StateChange = buildStateChangeHero(facts)
	case TxHeroValueFlow:
		model.ValueFlow = buildValueFlowHero(facts)
	case TxHeroOperations:
		model.Operations = buildOperationsHero(facts)
	default:
		model.GenericCall = buildGenericCallHero(facts)
	}
	return model
}

type txHeroFacts struct {
	Data           legacy.TxReceiptData
	Successful     bool
	TxType         string
	Subtype        string
	OperationTypes []string
	Actor          string
	ActorFull      string
	Contract       string
	Function       string
	HasValue       bool
	HasState       bool
	HasLifecycle   bool
	HasSoroban     bool
	LifecycleVerb  string
	StateChanges   []legacy.TxStateChange
	BalanceChanges []legacy.TxBalanceChange
}

func factsFromReceipt(data legacy.TxReceiptData) txHeroFacts {
	f := txHeroFacts{Data: data, Successful: !strings.EqualFold(data.Status, "failed"), TxType: strings.ToLower(data.SemanticTxType), Subtype: strings.ToLower(data.SemanticSubtype), StateChanges: data.StateChanges, BalanceChanges: data.BalanceChanges}
	f.Actor = firstNonEmpty(data.EffectiveActorShort, firstNonEmpty(data.SourceAddr, data.SubmitterShort))
	f.ActorFull = firstNonEmpty(data.EffectiveActorAddr, firstNonEmpty(data.SourceAddrFull, data.SubmitterAddr))
	f.Contract = firstNonEmpty(data.DownstreamContractShort, firstNonEmpty(data.ContractAddr, data.ContractAddrFull))
	f.Function = strings.TrimSuffix(firstNonEmpty(data.DownstreamFunctionName, data.ContractFn), "()")
	for _, op := range data.Operations {
		name := strings.ToLower(strings.TrimSpace(op.Type))
		if name != "" {
			f.OperationTypes = append(f.OperationTypes, strings.ReplaceAll(name, " ", "_"))
		}
		if f.Function == "" && op.Function != "" {
			f.Function = strings.TrimSuffix(op.Function, "()")
		}
		if f.Contract == "" && op.Contract != "" {
			f.Contract = op.Contract
		}
		if op.IsSoroban {
			f.HasSoroban = true
		}
		if strings.Contains(name, "extend") || strings.Contains(name, "restore") || strings.Contains(name, "create") || strings.Contains(name, "merge") {
			f.HasLifecycle = true
		}
	}
	joinedOps := strings.Join(f.OperationTypes, ",")
	if strings.Contains(joinedOps, "extend_footprint_ttl") || strings.Contains(joinedOps, "extend") || strings.Contains(f.TxType, "extend") || strings.Contains(f.Subtype, "ttl") {
		f.HasLifecycle = true
		f.LifecycleVerb = "extended"
	} else if strings.Contains(joinedOps, "restore") || strings.Contains(f.TxType, "restore") {
		f.HasLifecycle = true
		f.LifecycleVerb = "restored"
	} else if strings.Contains(joinedOps, "create") || strings.Contains(f.TxType, "create") || strings.Contains(f.Subtype, "deploy") {
		f.HasLifecycle = true
		f.LifecycleVerb = "created"
	} else if strings.Contains(joinedOps, "merge") || strings.Contains(f.TxType, "merge") {
		f.HasLifecycle = true
		f.LifecycleVerb = "merged"
	}
	if f.LifecycleVerb == "" {
		f.LifecycleVerb = "changed"
	}
	f.HasState = len(data.StateChanges) > 0
	f.HasValue = strings.Contains(f.TxType, "swap") || strings.Contains(f.Subtype, "swap") || strings.Contains(f.TxType, "transfer") || strings.Contains(f.TxType, "payment")
	// Raw ledger diffs often include the source account fee debit. Treat balance
	// changes as value-flow evidence only when there is more than one non-fee
	// participant; a lone XLM debit on a state-writing contract call is just fee.
	nonFeeBalanceChanges := 0
	for _, b := range data.BalanceChanges {
		if b.Change != "" && b.Change != "0" && !b.IsFee {
			nonFeeBalanceChanges++
		}
	}
	if nonFeeBalanceChanges >= 2 {
		f.HasValue = true
	}
	return f
}

func classifyTxHero(f txHeroFacts) TxHeroKind {
	if !f.Successful {
		return TxHeroFailure
	}
	if f.HasLifecycle {
		return TxHeroLifecycle
	}
	if f.HasValue {
		return TxHeroValueFlow
	}
	if f.HasState {
		return TxHeroStateChange
	}
	if !f.Data.IsSoroban && !f.HasSoroban && (f.TxType == "multi_op" || (len(f.OperationTypes) > 0 && f.Contract == "")) {
		return TxHeroOperations
	}
	return TxHeroGenericCall
}

func buildValueFlowHero(f txHeroFacts) *TxValueFlowHero {
	d := f.Data
	return &TxValueFlowHero{FromLabel: firstNonEmpty(f.Actor, "Source"), FromAddress: d.SourceAddr, ToLabel: "Destination", ToAddress: d.DestAddr, Protocol: firstNonEmpty(d.ContractName, f.Contract), Route: d.Route, SentAmount: d.SourceAmount, Received: d.DestAmount, Rate: d.EffectiveRate, Slippage: d.Slippage}
}

func buildStateChangeHero(f txHeroFacts) *TxStateChangeHero {
	entries := stateEntries(f.StateChanges)
	summary := fmt.Sprintf("<b>%s</b> called <code>%s()</code> on <b>%s</b>. No value flow was detected; this transaction wrote contract state.", esc(firstNonEmpty(f.Actor, "An actor")), esc(firstNonEmpty(f.Function, "function")), esc(firstNonEmpty(f.Contract, "a contract")))
	if len(entries) > 0 {
		summary = fmt.Sprintf("<b>%s</b> updated <b>%d</b> contract state entr%s on <b>%s</b>.", esc(firstNonEmpty(f.Actor, "An actor")), len(entries), pluralY(len(entries)), esc(firstNonEmpty(f.Contract, "a contract")))
	}
	return &TxStateChangeHero{Caller: f.Actor, Contract: f.Contract, Function: f.Function, SummaryHTML: summary, Entries: entries}
}

func buildLifecycleHero(f txHeroFacts) *TxLifecycleHero {
	entries := stateEntries(f.StateChanges)
	verb := f.LifecycleVerb
	summary := fmt.Sprintf("<b>%s</b> %s <b>%s</b>. This is a lifecycle transaction, so Prism renders what changed structurally instead of a value flow.", esc(firstNonEmpty(f.Actor, "An actor")), esc(verb), esc(firstNonEmpty(f.Contract, "an on-chain entity")))
	return &TxLifecycleHero{Verb: verb, Actor: f.Actor, Entity: f.Contract, SummaryHTML: summary, Entries: entries}
}

func buildFailureHero(f txHeroFacts) *TxFailureHero {
	frames := []TxFailureFrame{{Name: firstNonEmpty(f.Actor, "Source"), Detail: f.ActorFull, Status: "source"}}
	if f.Contract != "" || f.Function != "" {
		frames = append(frames, TxFailureFrame{Name: firstNonEmpty(f.Contract, "Contract"), Detail: callLabel(f.Function), Status: "failed here", Failed: true})
	}
	summary := fmt.Sprintf("Execution reverted while calling <code>%s()</code> on <b>%s</b>. Fees were still charged, but successful value/state changes were not applied.", esc(firstNonEmpty(f.Function, "function")), esc(firstNonEmpty(f.Contract, "the target contract")))
	return &TxFailureHero{Actor: f.Actor, Contract: f.Contract, Function: f.Function, ErrorCode: "reverted", SummaryHTML: summary, Frames: frames}
}

func buildGenericCallHero(f txHeroFacts) *TxGenericCallHero {
	summary := fmt.Sprintf("<b>%s</b> called <code>%s()</code> on <b>%s</b>. Prism did not detect value movement, lifecycle changes, or decoded state changes for a specialized hero.", esc(firstNonEmpty(f.Actor, "An actor")), esc(firstNonEmpty(f.Function, "function")), esc(firstNonEmpty(f.Contract, "a contract")))
	return &TxGenericCallHero{Actor: f.Actor, Contract: f.Contract, Function: f.Function, SummaryHTML: summary}
}

func buildOperationsHero(f txHeroFacts) *TxOperationsHero {
	count := operationCount(f)
	actor := firstNonEmpty(f.Actor, "The source account")
	breakdown := classicOperationBreakdown(f.OperationTypes)
	summary := fmt.Sprintf("<b>%s</b> submitted <b>%d classic operation%s</b> in one transaction.", esc(actor), count, pluralS(count))
	if breakdown != "" {
		summary = fmt.Sprintf("%s The envelope contains %s.", summary, breakdown)
	}
	return &TxOperationsHero{Actor: actor, Count: count, SummaryHTML: summary}
}

func operationCount(f txHeroFacts) int {
	if len(f.Data.Operations) > 0 {
		return len(f.Data.Operations)
	}
	value := strings.Fields(strings.TrimSpace(f.Data.OpsCount))
	if len(value) > 0 {
		if count, err := strconv.Atoi(value[0]); err == nil && count > 0 {
			return count
		}
	}
	return len(f.OperationTypes)
}

func classicOperationBreakdown(operationTypes []string) string {
	if len(operationTypes) == 0 {
		return ""
	}
	order := make([]string, 0, len(operationTypes))
	counts := make(map[string]int, len(operationTypes))
	for _, operationType := range operationTypes {
		label := strings.ToLower(strings.TrimSpace(strings.ReplaceAll(operationType, "_", " ")))
		if label == "" {
			continue
		}
		if counts[label] == 0 {
			order = append(order, label)
		}
		counts[label]++
	}
	parts := make([]string, 0, len(order))
	for _, label := range order {
		count := counts[label]
		parts = append(parts, fmt.Sprintf("<b>%d</b> %s operation%s", count, esc(label), pluralS(count)))
	}
	if len(parts) == 1 {
		return parts[0]
	}
	return strings.Join(parts[:len(parts)-1], ", ") + " and " + parts[len(parts)-1]
}

func stateEntries(changes []legacy.TxStateChange) []TxStateEntry {
	out := make([]TxStateEntry, 0, len(changes))
	for _, c := range changes {
		out = append(out, TxStateEntry{Key: firstNonEmpty(c.Key, "state entry"), EntryType: firstNonEmpty(c.Contract, "contract_data"), ChangeType: firstNonEmpty(c.Action, "updated"), Before: beforeAfter(c.DetailHTML, true), After: beforeAfter(c.DetailHTML, false)})
	}
	return out
}

func titleHTML(kind TxHeroKind, f txHeroFacts) string {
	switch kind {
	case TxHeroFailure:
		return fmt.Sprintf(`<span class="px-tx-actor">%s</span> <span class="px-tx-verb">reverted</span>`, esc(firstNonEmpty(f.Function, "Transaction")))
	case TxHeroLifecycle:
		return fmt.Sprintf(`<span class="px-tx-verb">%s</span> <span class="px-tx-actor">%s</span>`, strings.Title(f.LifecycleVerb), esc(firstNonEmpty(f.Contract, "on-chain state")))
	case TxHeroStateChange:
		return fmt.Sprintf(`<span class="px-tx-verb">Called</span> <span class="px-tx-actor">%s</span> <span class="px-tx-verb">on</span> <span class="px-tx-amt">%s</span>`, esc(firstNonEmpty(f.Function, "contract function")), esc(firstNonEmpty(f.Contract, "contract")))
	case TxHeroValueFlow:
		if f.Data.HeroTitle != "" {
			return esc(f.Data.HeroTitle)
		}
		return f.Data.SummaryHTML
	case TxHeroOperations:
		return fmt.Sprintf(`<span class="px-tx-actor">%s</span> <span class="px-tx-verb">submitted</span> <span class="px-tx-amt">%d operations</span>`, esc(firstNonEmpty(f.Actor, "Source account")), operationCount(f))
	default:
		return fmt.Sprintf(`<span class="px-tx-verb">Called</span> <span class="px-tx-actor">%s</span>`, esc(firstNonEmpty(f.Function, "contract")))
	}
}

func subtitleHTML(kind TxHeroKind, f txHeroFacts) string {
	if kind == TxHeroValueFlow && f.Data.AISummaryHTML != "" {
		return f.Data.AISummaryHTML
	}
	if f.Data.HumanNarrative != "" {
		return esc(f.Data.HumanNarrative)
	}
	if f.Data.SummaryHTML != "" {
		return f.Data.SummaryHTML
	}
	return "Prism selected the transaction hero from semantic classification, operations, events, and ledger-entry changes."
}

func tags(f txHeroFacts) []string {
	var out []string
	if f.Data.IsSoroban {
		out = append(out, "Soroban")
	}
	if f.TxType != "" {
		out = append(out, f.TxType)
	}
	if f.Subtype != "" {
		out = append(out, f.Subtype)
	}
	return out
}

func statusLabel(ok bool) string {
	if ok {
		return "Successful"
	}
	return "Reverted"
}
func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" && strings.TrimSpace(v) != "—" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
func esc(v string) string { return html.EscapeString(v) }
func pluralY(n int) string {
	if n == 1 {
		return "y"
	}
	return "ies"
}
func pluralS(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}
func callLabel(fn string) string {
	if fn == "" {
		return "contract call"
	}
	return fn + "()"
}
func beforeAfter(detail string, before bool) string {
	if strings.TrimSpace(detail) == "" {
		if before {
			return "previous value"
		}
		return "new value"
	}
	return detail
}
