package viewmodelv2

import (
	"fmt"
	"html"
	"strconv"
	"strings"

	legacy "github.com/withObsrvr/prism/internal/templates/pages"
	"github.com/withObsrvr/prism/internal/txoutcome"
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
	Actor                 string
	Contract              string
	Function              string
	Heading               string
	ErrorCode             string
	ReasonLabel           string
	CauseSpecificity      string
	EvidenceStatus        string
	DiagnosticStatus      string
	PhaseLabel            string
	OperationLabel        string
	OperationNumber       int
	Arguments             []string
	RolledBackOperations  int
	NotExecutedOperations int
	Impact                string
	SummaryHTML           string
	Caveats               []string
	Frames                []TxFailureFrame
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
	Actor string
	Count int
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
	Data              legacy.TxReceiptData
	Successful        bool
	TxType            string
	Subtype           string
	OperationTypes    []string
	Actor             string
	ActorFull         string
	Contract          string
	Function          string
	HasValue          bool
	HasState          bool
	HasLifecycle      bool
	SemanticValueFlow bool
	HasSoroban        bool
	LifecycleVerb     string
	StateChanges      []legacy.TxStateChange
	BalanceChanges    []legacy.TxBalanceChange
	Outcome           txoutcome.Interpretation
	HasOutcome        bool
}

// lifecycleOperationVerbs are the operation types that genuinely change what
// exists on-chain, matched exactly.
//
// This used to be a substring test for "extend", "restore", "create", or
// "merge" anywhere in an operation's display name. "Create Account" and
// "Create Claimable Balance" both contain "create", so a single one of them
// among any number of operations reclassified the whole transaction as a
// lifecycle change. Claimable balances alone are 42% of sampled mainnet
// operations. Both of those move value; neither changes what exists.
var lifecycleOperationVerbs = map[string]string{
	"extendfootprintttl": "extended",
	"restorefootprint":   "restored",
	"accountmerge":       "merged",
}

func lifecycleVerbForOperation(op legacy.TxOperation) string {
	return lifecycleOperationVerbs[normalizeOpName(firstNonEmpty(op.TypeName, op.Type))]
}

func factsFromReceipt(data legacy.TxReceiptData) txHeroFacts {
	f := txHeroFacts{Data: data, Successful: !strings.EqualFold(data.Status, "failed"), TxType: strings.ToLower(data.SemanticTxType), Subtype: strings.ToLower(data.SemanticSubtype), StateChanges: data.StateChanges, BalanceChanges: data.BalanceChanges}
	if data.OutcomeEvidence != nil {
		f.Outcome = txoutcome.Interpret(data.OutcomeEvidence)
		f.HasOutcome = true
		f.Successful = f.Outcome.Outcome == "succeeded"
	}
	// The headline says who submitted the operation. An effective actor is
	// execution context and can be a contract authorized inside the call; it is
	// not interchangeable with the transaction-envelope source.
	f.Actor = firstNonEmpty(data.SubmitterShort, firstNonEmpty(data.SourceAddr, data.EffectiveActorShort))
	f.ActorFull = firstNonEmpty(data.SubmitterAddr, firstNonEmpty(data.SourceAddrFull, data.EffectiveActorAddr))
	// ContractAddr is the top-level operation target. DownstreamContract may be
	// a supporting token/helper contract inferred from semantic actors, so only
	// use it when the operation itself did not identify a target.
	f.Contract = firstNonEmpty(data.ContractAddr, firstNonEmpty(data.ContractAddrFull, data.DownstreamContractShort))
	f.Function = strings.TrimSuffix(firstNonEmpty(data.DownstreamFunctionName, data.ContractFn), "()")
	if data.OutcomeEvidence != nil && data.OutcomeEvidence.PrimaryInvocation != nil {
		invocation := data.OutcomeEvidence.PrimaryInvocation
		if invocation.ContractID != "" {
			f.Contract = shortEntity(invocation.ContractID)
		}
		if invocation.FunctionName != "" {
			f.Function = invocation.FunctionName
		}
		f.HasSoroban = true
	}
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
		if verb := lifecycleVerbForOperation(op); verb != "" {
			f.HasLifecycle = true
			if f.LifecycleVerb == "" {
				f.LifecycleVerb = verb
			}
		}
	}
	// Contract deployment is a lifecycle event the operation type alone cannot
	// express: it arrives as invoke_host_function and is only distinguishable
	// from the semantic classification.
	if strings.Contains(f.Subtype, "deploy") || strings.Contains(f.TxType, "deploy") || strings.Contains(f.TxType, "create_contract") {
		f.HasLifecycle = true
		if f.LifecycleVerb == "" {
			f.LifecycleVerb = "deployed"
		}
	}
	if f.LifecycleVerb == "" {
		f.LifecycleVerb = "changed"
	}
	f.HasState = len(data.StateChanges) > 0
	// An explicit semantic classification is evidence; a flag inferred from
	// operation names is a guess. Kept separate so classifyTxHero can rank them.
	f.SemanticValueFlow = strings.Contains(f.TxType, "swap") || strings.Contains(f.Subtype, "swap") ||
		strings.Contains(f.TxType, "transfer") || strings.Contains(f.TxType, "payment")
	f.HasValue = f.SemanticValueFlow
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
	// Lifecycle is checked before value flow, and that is correct now that
	// HasLifecycle comes only from an exact operation match. Ranking the
	// semantic transfer/payment/swap classification above it was tried and
	// reverted: it is redundant once the substring matching is gone, and it
	// reclassified account_merge as a plain transfer, losing the fact that the
	// account ceased to exist. A merge does move the whole balance, but the
	// destruction is the more distinctive fact.
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

// primaryTransferEvent returns the event that best represents this
// transaction's value movement: the first value-bearing transfer with both
// participants named.
func primaryTransferEvent(f txHeroFacts) *legacy.TxEvent {
	for i := range f.Data.Events {
		e := &f.Data.Events[i]
		if e.From != "" && e.To != "" && strings.TrimSpace(e.Amount) != "" {
			return e
		}
	}
	for i := range f.Data.Events {
		e := &f.Data.Events[i]
		if e.From != "" && e.To != "" {
			return e
		}
	}
	return nil
}

// valueFlowParties reports who value actually moved between.
//
// The submitter signs the envelope; the event says whose balance changed, and
// the two are not always the same account. A sponsored or relayed transfer is
// signed by one account and debits another, so reading the sender from
// data.SourceAddr made the headline assert something the events contradicted.
func valueFlowParties(f txHeroFacts) (from, fromFull, to, toFull string) {
	if e := primaryTransferEvent(f); e != nil {
		return e.From, e.FromFull, e.To, e.ToFull
	}
	d := f.Data
	return firstNonEmpty(f.Actor, d.SourceAddr), d.SourceAddrFull, d.DestAddr, d.DestAddr
}

func buildValueFlowHero(f txHeroFacts) *TxValueFlowHero {
	d := f.Data
	from, _, to, _ := valueFlowParties(f)
	return &TxValueFlowHero{FromLabel: firstNonEmpty(from, "Source"), FromAddress: from, ToLabel: "Destination", ToAddress: to, Protocol: firstNonEmpty(d.ContractName, f.Contract), Route: d.Route, SentAmount: d.SourceAmount, Received: d.DestAmount, Rate: d.EffectiveRate, Slippage: d.Slippage}
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
	interpretation := f.Outcome
	if !f.HasOutcome {
		interpretation = txoutcome.Interpret(nil)
	}
	var frames []TxFailureFrame
	if interpretation.RolledBackOperations > 0 || interpretation.NotExecutedOperations > 0 {
		frames = append(frames, TxFailureFrame{Name: firstNonEmpty(f.Actor, "Source"), Detail: f.ActorFull, Status: "source"})
		if interpretation.OperationNumber > 0 {
			frames = append(frames, TxFailureFrame{
				Name:   firstNonEmpty(interpretation.OperationLabel, "Operation"),
				Detail: fmt.Sprintf("Operation #%d · %s", interpretation.OperationNumber, firstNonEmpty(interpretation.PhaseLabel, "execution")),
				Status: "failed here",
				Failed: true,
			})
		}
		if f.Contract != "" || f.Function != "" {
			frames = append(frames, TxFailureFrame{Name: firstNonEmpty(f.Contract, "Contract"), Detail: callLabel(f.Function), Status: invocationFrameStatus(interpretation), Failed: interpretation.OperationNumber == 0})
		}
	}
	summary := esc(interpretation.Summary)
	return &TxFailureHero{
		Actor:                 f.Actor,
		Contract:              f.Contract,
		Function:              f.Function,
		Heading:               firstNonEmpty(interpretation.Heading, "Transaction failed"),
		ErrorCode:             firstNonEmpty(interpretation.ReasonCode, "reason_unavailable"),
		ReasonLabel:           firstNonEmpty(interpretation.ReasonLabel, "Reason unavailable"),
		CauseSpecificity:      firstNonEmpty(interpretation.CauseSpecificity, "unresolved"),
		EvidenceStatus:        firstNonEmpty(interpretation.EvidenceStatus, "unavailable"),
		DiagnosticStatus:      interpretation.DiagnosticStatus,
		PhaseLabel:            interpretation.PhaseLabel,
		OperationLabel:        interpretation.OperationLabel,
		OperationNumber:       interpretation.OperationNumber,
		Arguments:             interpretation.ArgumentLabels,
		RolledBackOperations:  interpretation.RolledBackOperations,
		NotExecutedOperations: interpretation.NotExecutedOperations,
		Impact:                interpretation.Impact,
		SummaryHTML:           summary,
		Caveats:               interpretation.Caveats,
		Frames:                frames,
	}
}

func buildGenericCallHero(f txHeroFacts) *TxGenericCallHero {
	summary := fmt.Sprintf("<b>%s</b> called <code>%s()</code> on <b>%s</b>. Prism did not detect value movement, lifecycle changes, or decoded state changes for a specialized hero.", esc(firstNonEmpty(f.Actor, "An actor")), esc(firstNonEmpty(f.Function, "function")), esc(firstNonEmpty(f.Contract, "a contract")))
	return &TxGenericCallHero{Actor: f.Actor, Contract: f.Contract, Function: f.Function, SummaryHTML: summary}
}

func buildOperationsHero(f txHeroFacts) *TxOperationsHero {
	return &TxOperationsHero{Actor: firstNonEmpty(f.Actor, "The source account"), Count: operationCount(f)}
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

// actorSpan renders the account the headline speaks for. Every hero leads with
// it: a sampling of mainnet and testnet receipts showed every variant putting
// the actor in the subtitle and a bare verb phrase in the headline, so the
// headline read "Called fill_orders" while the subtitle underneath it read
// "GB4F...S777 called fill orders() on CAZV...BTZZ" — the headline was a strict
// subset of the line below it.
func actorSpan(f txHeroFacts) string {
	return fmt.Sprintf(`<span class="px-tx-actor">%s</span>`, esc(firstNonEmpty(f.Actor, "Source account")))
}

func verbSpan(s string) string { return fmt.Sprintf(`<span class="px-tx-verb">%s</span>`, esc(s)) }

// evidenceSpan is for anything a reader might copy, compare, or verify:
// addresses, contract ids, function names. Same treatment as the actor, because
// they are the same kind of thing. amtSpan is reserved for asset-denominated
// values, which additionally carry the asset chip.
func evidenceSpan(s string) string {
	return fmt.Sprintf(`<span class="px-tx-ev">%s</span>`, esc(s))
}

// proseOrEvidence routes a fragment by what it is. A function name is evidence;
// a category label like "payment" is Prism's own wording.
func proseOrEvidence(s string) string {
	if strings.HasSuffix(strings.TrimSpace(s), "()") {
		return evidenceSpan(s)
	}
	return verbSpan(s)
}
func amtSpan(s string) string { return fmt.Sprintf(`<span class="px-tx-amt">%s</span>`, esc(s)) }

// lowerFirst downcases a leading capital so an upstream sentence fragment can
// be spliced in after the actor without reading like two sentences.
func lowerFirst(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	if r[0] >= 'A' && r[0] <= 'Z' && !(len(r) > 1 && r[1] >= 'A' && r[1] <= 'Z') {
		r[0] = r[0] + ('a' - 'A')
	}
	return string(r)
}

func titleHTML(kind TxHeroKind, f txHeroFacts) string {
	switch kind {
	case TxHeroFailure:
		// Without structured outcome evidence Prism does not know what was
		// attempted. Leading with the actor would imply we are describing their
		// action; say what we actually know instead.
		if !f.HasOutcome {
			return `<span class="px-tx-actor">Failure reason unavailable</span>`
		}
		// Prefer the outcome heading, which names the function on a Soroban
		// failure ("swap() failed") where OperationLabel only has the generic
		// category. Strip its trailing "failed" so the verb can carry that.
		action := strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(f.Outcome.Heading), "failed"))
		if action == "" {
			action = strings.TrimSpace(f.Outcome.OperationLabel)
		}
		if action == "" {
			return actorSpan(f) + " " + verbSpan("submitted a transaction that failed")
		}
		return actorSpan(f) + " " + verbSpan("could not complete") + " " + proseOrEvidence(lowerFirst(action))
	case TxHeroLifecycle:
		// Prefer what the operations actually say over the generic
		// "<verb> on-chain state" when no contract is involved.
		if f.Contract == "" {
			if lead := leadOperationIndex(f.Data.Operations); lead >= 0 {
				if phrase := operationPhrase(f.Data.Operations[lead]); phrase != "" {
					return actorSpan(f) + " " + verbSpan(phrase)
				}
			}
		}
		if f.Contract != "" {
			return actorSpan(f) + " " + verbSpan(f.LifecycleVerb) + " " + evidenceSpan(f.Contract)
		}
		return actorSpan(f) + " " + verbSpan(f.LifecycleVerb) + " " + verbSpan("on-chain state")
	case TxHeroStateChange:
		return actorSpan(f) + " " + verbSpan("called") + " " + evidenceSpan(firstNonEmpty(f.Function, "a contract function")+"()") +
			" " + verbSpan("on") + " " + evidenceSpan(firstNonEmpty(f.Contract, "a contract"))
	case TxHeroValueFlow:
		return valueFlowTitleHTML(f)
	case TxHeroOperations:
		return operationsTitleHTML(f)
	default:
		out := actorSpan(f) + " " + verbSpan("called") + " " + evidenceSpan(firstNonEmpty(f.Function, "a contract function")+"()")
		if f.Contract != "" {
			out += " " + verbSpan("on") + " " + evidenceSpan(f.Contract)
		}
		return out
	}
}

// valueFlowTitleHTML puts the actor in front of the movement. The phrase itself
// comes from the gateway's semantic layer, which supplies sentence-cased
// fragments like "Swapped X for Y", so it is spliced in lowercased.
func valueFlowTitleHTML(f txHeroFacts) string {
	// Prefer the event: it names the accounts whose balances changed and the
	// asset by symbol, where the upstream phrase names the signer and renders
	// the token by contract id.
	if e := primaryTransferEvent(f); e != nil && strings.TrimSpace(e.Amount) != "" {
		value := strings.TrimSpace(e.Amount)
		if asset := strings.TrimSpace(e.Asset); asset != "" {
			value += " " + asset
		}
		return fmt.Sprintf(`<span class="px-tx-actor">%s</span> %s %s %s %s`,
			esc(e.From), verbSpan(transferVerb(e.Type)), amtSpan(value), verbSpan("to"), evidenceSpan(e.To))
	}
	phrase := strings.TrimSpace(f.Data.HeroTitle)
	if phrase == "" {
		phrase = strings.TrimSpace(stripTags(f.Data.SummaryHTML))
	}
	// The upstream phrase sometimes leads with the actor already; do not repeat it.
	if phrase != "" && f.Actor != "" && strings.HasPrefix(phrase, f.Actor) {
		return esc(phrase)
	}
	if phrase == "" {
		return actorSpan(f) + " " + verbSpan("moved value")
	}
	if n, ok := genericTransferCount(phrase); ok {
		noun := "transfers"
		if n == 1 {
			noun = "transfer"
		}
		return actorSpan(f) + " " + verbSpan("made") + " " + verbSpan(fmt.Sprintf("%d %s", n, noun))
	}
	return actorSpan(f) + " " + verbSpan(lowerFirst(phrase))
}

// genericTransferCount recognises the gateway's placeholder value-flow phrase,
// "Transaction with N transfers". It names no counterparty and no amount, and
// mis-pluralises at one, so the count is extracted and phrased locally.
func genericTransferCount(phrase string) (int, bool) {
	lower := strings.ToLower(strings.TrimSpace(phrase))
	if !strings.HasPrefix(lower, "transaction with ") || !strings.HasSuffix(lower, "transfers") {
		return 0, false
	}
	digits := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(lower, "transaction with "), "transfers"))
	n, err := strconv.Atoi(digits)
	if err != nil || n <= 0 {
		return 0, false
	}
	return n, true
}

// transferVerb keeps mint and burn distinct from a plain transfer.
func transferVerb(eventType string) string {
	switch strings.ToLower(strings.TrimSpace(eventType)) {
	case "mint":
		return "minted"
	case "burn":
		return "burned"
	}
	return "sent"
}

func stripTags(s string) string {
	var b strings.Builder
	depth := 0
	for _, r := range s {
		switch {
		case r == '<':
			depth++
		case r == '>':
			if depth > 0 {
				depth--
			}
		case depth == 0:
			b.WriteRune(r)
		}
	}
	return html.UnescapeString(b.String())
}

// operationPhrase renders one classic operation as an actor-less verb phrase.
//
// The verbs are deliberately literal. A manage_buy_offer places an offer; it
// does not necessarily execute, so "offered to buy" is the honest phrasing and
// "bought" is not. Same for sell. Overstating an offer as a completed trade
// would be the page asserting an economic event that may never have happened.
func operationPhrase(op legacy.TxOperation) string {
	amount := strings.TrimSpace(op.Amount)
	asset := strings.TrimSpace(op.Asset)
	value := ""
	if amount != "" {
		value = amount
		if asset != "" {
			value += " " + asset
		}
	}

	// Prefer the raw protocol name, but fall back to the display name: not every
	// path that builds a TxOperation populates TypeName, and normalizeOpName
	// folds "Manage Buy Offer" onto the same key as "manage_buy_offer".
	switch normalizeOpName(firstNonEmpty(op.TypeName, op.Type)) {
	case "managebuyoffer":
		if isZeroAmount(amount) {
			return "cancelled a buy offer"
		}
		if value != "" {
			return "offered to buy " + value
		}
		return "placed a buy offer"
	case "manageselloffer", "createpassiveselloffer":
		if isZeroAmount(amount) {
			return "cancelled a sell offer"
		}
		if value != "" {
			return "offered to sell " + value
		}
		return "placed a sell offer"
	case "createclaimablebalance":
		// 42% of sampled mainnet operations. Phrasing it "sent" would be wrong:
		// the funds sit in a balance entry until a claimant claims them.
		if value != "" {
			return "reserved " + value + " as a claimable balance"
		}
		return "created a claimable balance"
	case "claimclaimablebalance":
		if value != "" {
			return "claimed " + value
		}
		return "claimed a balance"
	case "invokehostfunction":
		return "called a contract"
	case "extendfootprintttl":
		return "extended contract state rent"
	case "restorefootprint":
		return "restored archived contract state"
	case "revokesponsorship":
		return "revoked a sponsorship"
	case "beginsponsoringfuturereserves", "endsponsoringfuturereserves":
		return "sponsored account reserves"
	case "inflation":
		return "ran inflation"
	case "payment":
		if value != "" {
			return "sent " + value
		}
		return "sent a payment"
	case "pathpaymentstrictreceive", "pathpaymentstrictsend":
		if value != "" {
			return "path paid " + value
		}
		return "made a path payment"
	case "createaccount":
		if value != "" {
			return "funded a new account with " + value
		}
		return "created an account"
	case "accountmerge":
		return "merged an account"
	case "changetrust":
		return "changed a trustline"
	case "settrustlineflags", "allowtrust":
		return "set trust flags"
	case "clawback", "clawbackclaimablebalance":
		if value != "" {
			return "clawed back " + value
		}
		return "clawed back an asset"
	case "setoptions":
		return "updated account options"
	case "managedata":
		return "updated a data entry"
	case "bumpsequence":
		return "bumped its sequence"
	case "liquiditypooldeposit":
		return "deposited into a liquidity pool"
	case "liquiditypoolwithdraw":
		return "withdrew from a liquidity pool"
	}
	return ""
}

// isZeroAmount reports whether a formatted amount carries no value. Setting an
// offer amount to zero is how Stellar cancels it.
func isZeroAmount(amount string) bool {
	amount = strings.TrimSpace(amount)
	if amount == "" {
		return false
	}
	for _, r := range amount {
		if r >= '1' && r <= '9' {
			return false
		}
	}
	return true
}

// operationPhraseGeneric drops the amount so repeated operations of the same
// kind group together. Grouping on the full phrase keyed every distinct amount
// separately, producing "offered to buy 13,980.65 XLM, offered to buy 27,961.3
// XLM, offered to buy 4..." instead of one entry with a count.
func operationPhraseGeneric(op legacy.TxOperation) string {
	op.Amount = ""
	op.Asset = ""
	return operationPhrase(op)
}

func normalizeOpName(v string) string {
	v = strings.TrimPrefix(v, "OperationType")
	var b strings.Builder
	for _, r := range v {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r + ('a' - 'A'))
		}
	}
	return b.String()
}

// operationsTitleHTML leads with the actor and the most economically
// significant thing they did, rather than restating the operation count that
// already appears in the chips, the sidebar, and the tab badge.
//
// "Most significant" means the first operation carrying an amount; failing
// that, the first operation we can phrase at all. Remaining operations are
// summarised as a trailing count so the headline stays one clause.
// leadOperationIndex picks the operation the headline speaks for: the first one
// carrying value, else the first one that can be phrased at all.
func leadOperationIndex(ops []legacy.TxOperation) int {
	for i, op := range ops {
		if strings.TrimSpace(op.Amount) != "" && !isZeroAmount(op.Amount) && operationPhrase(op) != "" {
			return i
		}
	}
	for i, op := range ops {
		if operationPhrase(op) != "" {
			return i
		}
	}
	return -1
}

func operationsTitleHTML(f txHeroFacts) string {
	actor := esc(firstNonEmpty(f.Actor, "Source account"))
	ops := f.Data.Operations

	lead := leadOperationIndex(ops)
	if lead < 0 {
		// Nothing phraseable: fall back to the count rather than inventing one.
		return fmt.Sprintf(`<span class="px-tx-actor">%s</span> <span class="px-tx-verb">submitted</span> <span class="px-tx-amt">%d operations</span>`,
			actor, operationCount(f))
	}

	phrase := operationPhrase(ops[lead])
	// Split the phrase so the value carries the amount emphasis and the verb
	// stays quiet, matching the other hero variants.
	verb, value := phrase, ""
	if amount := strings.TrimSpace(ops[lead].Amount); amount != "" {
		if idx := strings.LastIndex(phrase, amount); idx >= 0 {
			verb = strings.TrimSpace(phrase[:idx])
			value = strings.TrimSpace(phrase[idx:])
		}
	}

	out := fmt.Sprintf(`<span class="px-tx-actor">%s</span> <span class="px-tx-verb">%s</span>`, actor, esc(verb))
	if value != "" {
		out += fmt.Sprintf(` <span class="px-tx-amt">%s</span>`, esc(value))
	}
	return out
}

// operationsSubtitle names the operations the headline did not lead with, in
// plain language. The default subtitle restated the actor and listed raw
// protocol names ("a multi op involving manage_buy_offer, set_trust_line_flags"),
// which repeated the headline in worse words.
func operationsSubtitle(f txHeroFacts) string {
	ops := f.Data.Operations
	lead := leadOperationIndex(ops)
	if lead < 0 {
		return ""
	}
	counts := map[string]int{}
	var order []string
	for i, op := range ops {
		if i == lead {
			continue
		}
		phrase := operationPhraseGeneric(op)
		if phrase == "" {
			continue
		}
		if _, seen := counts[phrase]; !seen {
			order = append(order, phrase)
		}
		counts[phrase]++
	}
	if len(order) == 0 {
		return ""
	}
	parts := make([]string, 0, len(order))
	for _, phrase := range order {
		if n := counts[phrase]; n > 1 {
			parts = append(parts, fmt.Sprintf("%s (%d)", phrase, n))
		} else {
			parts = append(parts, phrase)
		}
	}
	return "Also " + strings.Join(parts, ", ") + "."
}

func subtitleHTML(kind TxHeroKind, f txHeroFacts) string {
	if kind == TxHeroOperations {
		if sub := operationsSubtitle(f); sub != "" {
			return esc(sub)
		}
	}
	if kind == TxHeroFailure {
		if f.HasOutcome {
			return esc(f.Outcome.Summary)
		}
		return esc(txoutcome.Interpret(nil).Summary)
	}
	// A sub-call that reverted inside a transaction that succeeded is the most
	// notable thing about that transaction, and nothing else on the page says
	// it. The failure hero keeps its own explanation, which is more specific.
	if note := recoveredCallNote(f); note != "" {
		return esc(note)
	}
	if kind == TxHeroValueFlow && f.Data.AISummaryHTML != "" {
		return f.Data.AISummaryHTML
	}
	// Everything below this point used to fall through to HumanNarrative or
	// SummaryHTML, which is where the restatement came from: those render as
	// "GAIH...ZNSR submitted a create account involving create_account", repeating
	// the actor and the action the headline now carries, in raw protocol names.
	// The subtitle earns its place or it does not render.
	if narrative := strings.TrimSpace(stripTags(f.Data.HumanNarrative)); narrative != "" && !restatesTitle(narrative, f) {
		return esc(narrative)
	}
	return ""
}

// restatesTitle reports whether a candidate subtitle only repeats what the
// headline already says. The generated narratives follow a fixed shape,
// "<actor> submitted a <type> involving <op_names>", so they are recognised by
// that shape rather than by comparing rendered strings.
// recoveredCallNote reports sub-calls that trapped and were caught by their
// caller. It says "of N" so the scale is clear: one reverted call out of six is
// a different story from six out of six.
func recoveredCallNote(f txHeroFacts) string {
	caught := 0
	for _, node := range f.Data.CallTree {
		if node.State == "caught" {
			caught++
		}
	}
	if caught == 0 {
		return ""
	}
	// In "N of M sub-calls" the noun agrees with the total and the verb with the
	// count, so one reverted call out of six is "1 of 6 sub-calls ... was".
	noun := "sub-calls"
	if len(f.Data.CallTree) == 1 {
		noun = "sub-call"
	}
	verb := "were"
	if caught == 1 {
		verb = "was"
	}
	return fmt.Sprintf("%d of %d %s reverted and %s recovered.", caught, len(f.Data.CallTree), noun, verb)
}

func restatesTitle(narrative string, f txHeroFacts) bool {
	lower := strings.ToLower(narrative)
	if strings.Contains(lower, " submitted a ") && strings.Contains(lower, " involving ") {
		return true
	}
	// Raw protocol names never belong in prose; the operation list shows them.
	for _, opType := range f.OperationTypes {
		if opType != "" && strings.Contains(lower, opType) {
			return true
		}
	}
	if f.Actor != "" && strings.HasPrefix(narrative, f.Actor) && len(narrative) < 80 {
		return true
	}
	return false
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
	return "Failed"
}

func invocationFrameStatus(interpretation txoutcome.Interpretation) string {
	if interpretation.OperationNumber > 0 {
		return "invoked"
	}
	return "failure target"
}

func shortEntity(value string) string {
	if len(value) <= 12 {
		return value
	}
	return value[:6] + "..." + value[len(value)-4:]
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
