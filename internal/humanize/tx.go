package humanize

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/withObsrvr/prism/internal/gateway"
)

type TxNarrative struct {
	Title           string
	Narrative       string
	ConfidenceLabel string
	ConfidenceValue string
	Actors          []TxActor
	Evidence        []TxEvidence
	Signals         []TxSignal
}

type TxActor struct {
	Label     string
	Role      string
	ActorType string
	Href      string
}

type TxEvidence struct {
	Label string
	Value string
}

type TxSignal struct {
	Title    string
	Severity string
	Summary  string
}

type semanticTxContext struct {
	TxType             string
	Subtype            string
	Confidence         string
	OperationTypes     []string
	WalletInvolved     bool
	EffectiveActorType string
	Actors             []gateway.SemanticActor
	Assets             gateway.SemanticAssetContext
	Operations         []gateway.DecodedOperation
	PrimaryFunction    string
	PrimaryContract    string
	PrimaryContractID  string
	PrimaryProject     string
	PrimaryOperation   string
	AssetMovement      string
}

type txNarrator interface {
	Matches(ctx semanticTxContext) bool
	Build(ctx semanticTxContext) TxNarrative
}

func BuildTxNarrative(resp *gateway.SemanticTransactionResponse) TxNarrative {
	if resp == nil {
		return TxNarrative{}
	}
	ctx := semanticTxContext{
		TxType:             resp.Classification.TxType,
		Subtype:            resp.Classification.Subtype,
		Confidence:         resp.Classification.Confidence,
		OperationTypes:     resp.Classification.OperationTypes,
		WalletInvolved:     resp.Classification.WalletInvolved,
		EffectiveActorType: resp.Classification.EffectiveActorType,
		Actors:             resp.Actors,
		Assets:             resp.Assets,
		Operations:         resp.Operations,
	}
	ctx.PrimaryFunction = primaryFunction(ctx)
	ctx.PrimaryContractID = primaryContractID(ctx)
	ctx.PrimaryContract = primaryContract(ctx)
	ctx.PrimaryProject = InferProject(ctx.PrimaryContract)
	ctx.PrimaryOperation = primaryOperation(ctx)
	ctx.AssetMovement = primaryAssetMovement(ctx)

	for _, n := range []txNarrator{
		smartWalletPolicyUpdateNarrator{},
		smartWalletMultisigApprovalNarrator{},
		smartWalletTransferNarrator{},
		smartWalletSwapNarrator{},
		walletMediatedProtocolInteractionNarrator{},
		accountFundingNarrator{},
		genericContractCallNarrator{},
		genericNarrator{},
	} {
		if n.Matches(ctx) {
			return n.Build(ctx)
		}
	}
	return genericNarrator{}.Build(ctx)
}

type smartWalletPolicyUpdateNarrator struct{}

func (smartWalletPolicyUpdateNarrator) Matches(ctx semanticTxContext) bool {
	return ctx.TxType == "smart_wallet_policy_update"
}

func (smartWalletPolicyUpdateNarrator) Build(ctx semanticTxContext) TxNarrative {
	walletLabel := actorLabel(firstActorByRole(ctx.Actors, "effective_actor"))
	action := "updated its policy"
	if ctx.Subtype != "" {
		action = strings.ReplaceAll(ctx.Subtype, "_", " ")
	}
	narrative := fmt.Sprintf("%s %s.", walletLabel, actionPhrase(action))
	if ctx.PrimaryFunction != "" {
		narrative = fmt.Sprintf("%s %s.", walletLabel, describeFunctionCall(ctx.PrimaryFunction, ctx.PrimaryContractID, ctx.PrimaryContract, "", ""))
	}
	return TxNarrative{
		Title:           "Smart wallet policy update",
		Narrative:       narrative,
		ConfidenceLabel: confidenceLabel(ctx.Confidence),
		ConfidenceValue: ctx.Confidence,
		Actors:          buildActors(ctx.Actors),
		Evidence:        buildEvidence(ctx),
		Signals:         buildSignals(ctx),
	}
}

type smartWalletMultisigApprovalNarrator struct{}

func (smartWalletMultisigApprovalNarrator) Matches(ctx semanticTxContext) bool {
	return ctx.TxType == "smart_wallet_multisig_approval"
}

func (smartWalletMultisigApprovalNarrator) Build(ctx semanticTxContext) TxNarrative {
	wallet := actorLabel(firstActorByRole(ctx.Actors, "effective_actor"))
	narrative := fmt.Sprintf("%s submitted a multisig approval.", wallet)
	if ctx.PrimaryFunction != "" {
		narrative = fmt.Sprintf("%s %s as part of a multisig approval flow.", wallet, describeFunctionCall(ctx.PrimaryFunction, ctx.PrimaryContractID, ctx.PrimaryContract, "", ""))
	}
	return TxNarrative{
		Title:           "Smart wallet approval",
		Narrative:       narrative,
		ConfidenceLabel: confidenceLabel(ctx.Confidence),
		ConfidenceValue: ctx.Confidence,
		Actors:          buildActors(ctx.Actors),
		Evidence:        buildEvidence(ctx),
		Signals:         buildSignals(ctx),
	}
}

type smartWalletTransferNarrator struct{}

func (smartWalletTransferNarrator) Matches(ctx semanticTxContext) bool {
	return ctx.TxType == "smart_wallet_transfer"
}

func (smartWalletTransferNarrator) Build(ctx semanticTxContext) TxNarrative {
	wallet := actorLabel(firstActorByRole(ctx.Actors, "effective_actor"))
	receiver := actorLabel(firstActorByRole(ctx.Actors, "receiver"))
	amount, asset := transferAmount(ctx)
	title := "Smart wallet transfer"
	narrative := fmt.Sprintf("%s transferred", wallet)
	if ctx.PrimaryFunction != "" && ctx.PrimaryContract != "" {
		narrative = fmt.Sprintf("%s %s", wallet, describeFunctionCall(ctx.PrimaryFunction, ctx.PrimaryContractID, ctx.PrimaryContract, amount, asset))
		if receiver != "an actor" {
			narrative += " to " + receiver
		}
		narrative += "."
		return TxNarrative{
			Title:           title,
			Narrative:       narrative,
			ConfidenceLabel: confidenceLabel(ctx.Confidence),
			ConfidenceValue: ctx.Confidence,
			Actors:          buildActors(ctx.Actors),
			Evidence:        buildEvidence(ctx),
			Signals:         buildSignals(ctx),
		}
	}
	if amount != "" {
		narrative += " " + amount
		if asset != "" {
			narrative += " " + asset
		}
	}
	if receiver != "an actor" {
		narrative += " to " + receiver
	}
	narrative += "."
	return TxNarrative{
		Title:           title,
		Narrative:       narrative,
		ConfidenceLabel: confidenceLabel(ctx.Confidence),
		ConfidenceValue: ctx.Confidence,
		Actors:          buildActors(ctx.Actors),
		Evidence:        buildEvidence(ctx),
		Signals:         buildSignals(ctx),
	}
}

type smartWalletSwapNarrator struct{}

func (smartWalletSwapNarrator) Matches(ctx semanticTxContext) bool {
	return ctx.TxType == "smart_wallet_swap"
}

func (smartWalletSwapNarrator) Build(ctx semanticTxContext) TxNarrative {
	wallet := actorLabel(firstActorByRole(ctx.Actors, "effective_actor"))
	inAmt, inAsset, outAmt, outAsset := swapAmounts(ctx)
	title := "Smart wallet swap"
	narrative := fmt.Sprintf("%s swapped", wallet)
	if ctx.PrimaryFunction != "" && ctx.PrimaryContract != "" {
		narrative = fmt.Sprintf("%s %s", wallet, describeSwapCall(ctx.PrimaryFunction, ctx.PrimaryContractID, ctx.PrimaryContract, inAmt, inAsset, outAmt, outAsset))
		narrative += "."
		return TxNarrative{
			Title:           title,
			Narrative:       narrative,
			ConfidenceLabel: confidenceLabel(ctx.Confidence),
			ConfidenceValue: ctx.Confidence,
			Actors:          buildActors(ctx.Actors),
			Evidence:        buildEvidence(ctx),
			Signals:         buildSignals(ctx),
		}
	}
	if inAmt != "" {
		narrative += " " + inAmt
		if inAsset != "" {
			narrative += " " + inAsset
		}
	}
	if outAmt != "" {
		narrative += " for " + outAmt
		if outAsset != "" {
			narrative += " " + outAsset
		}
	}
	protocol := actorLabel(firstActorByRole(ctx.Actors, "protocol"))
	if protocol != "an actor" {
		narrative += " through " + protocol
	}
	narrative += "."
	return TxNarrative{
		Title:           title,
		Narrative:       narrative,
		ConfidenceLabel: confidenceLabel(ctx.Confidence),
		ConfidenceValue: ctx.Confidence,
		Actors:          buildActors(ctx.Actors),
		Evidence:        buildEvidence(ctx),
		Signals:         buildSignals(ctx),
	}
}

type walletMediatedProtocolInteractionNarrator struct{}

func (walletMediatedProtocolInteractionNarrator) Matches(ctx semanticTxContext) bool {
	return ctx.TxType == "wallet_mediated_protocol_interaction"
}

func (walletMediatedProtocolInteractionNarrator) Build(ctx semanticTxContext) TxNarrative {
	wallet := actorLabel(firstActorByRole(ctx.Actors, "effective_actor"))
	protocol := actorLabel(firstActorByRole(ctx.Actors, "protocol"))
	title := "Wallet-mediated protocol interaction"
	narrative := fmt.Sprintf("%s interacted with a protocol", wallet)
	if ctx.PrimaryFunction != "" {
		narrative = fmt.Sprintf("%s %s", wallet, describeFunctionCall(ctx.PrimaryFunction, ctx.PrimaryContractID, ctx.PrimaryContract, "", ""))
		if ctx.Subtype != "" {
			narrative += fmt.Sprintf(" for %s", strings.ReplaceAll(ctx.Subtype, "_", " "))
		}
		narrative += "."
		return TxNarrative{
			Title:           title,
			Narrative:       narrative,
			ConfidenceLabel: confidenceLabel(ctx.Confidence),
			ConfidenceValue: ctx.Confidence,
			Actors:          buildActors(ctx.Actors),
			Evidence:        buildEvidence(ctx),
			Signals:         buildSignals(ctx),
		}
	}
	if protocol != "an actor" {
		narrative += " through " + protocol
	}
	if ctx.Subtype != "" {
		narrative += fmt.Sprintf(" for %s", strings.ReplaceAll(ctx.Subtype, "_", " "))
	}
	narrative += "."
	return TxNarrative{
		Title:           title,
		Narrative:       narrative,
		ConfidenceLabel: confidenceLabel(ctx.Confidence),
		ConfidenceValue: ctx.Confidence,
		Actors:          buildActors(ctx.Actors),
		Evidence:        buildEvidence(ctx),
		Signals:         buildSignals(ctx),
	}
}

type accountFundingNarrator struct{}

func (accountFundingNarrator) Matches(ctx semanticTxContext) bool {
	return ctx.TxType == "account_funding"
}

func (accountFundingNarrator) Build(ctx semanticTxContext) TxNarrative {
	sender := actorLabel(firstActorByRole(ctx.Actors, "sender"))
	if sender == "an actor" {
		sender = actorLabel(firstActorByRole(ctx.Actors, "submitter"))
	}
	receiver := actorLabel(firstActorByRole(ctx.Actors, "receiver"))
	amount, asset := fundingAmount(ctx)
	narrative := fmt.Sprintf("%s funded %s", sender, receiver)
	if amount != "" {
		narrative += fmt.Sprintf(" with %s", amount)
		if asset != "" {
			narrative += " " + asset
		}
	}
	narrative += "."
	return TxNarrative{
		Title:           "Account funding",
		Narrative:       narrative,
		ConfidenceLabel: confidenceLabel(ctx.Confidence),
		ConfidenceValue: ctx.Confidence,
		Actors:          buildActors(ctx.Actors),
		Evidence:        buildEvidence(ctx),
		Signals:         buildSignals(ctx),
	}
}

type genericContractCallNarrator struct{}

func (genericContractCallNarrator) Matches(ctx semanticTxContext) bool {
	return ctx.TxType == "contract_call"
}

func (genericContractCallNarrator) Build(ctx semanticTxContext) TxNarrative {
	actor := actorLabel(firstActorByRole(ctx.Actors, "effective_actor"))
	protocol := actorLabel(firstActorByRole(ctx.Actors, "protocol"))
	narrative := fmt.Sprintf("%s called a contract", actor)
	if ctx.PrimaryFunction != "" {
		narrative = fmt.Sprintf("%s %s.", actor, describeFunctionCall(ctx.PrimaryFunction, ctx.PrimaryContractID, ctx.PrimaryContract, "", ""))
		return TxNarrative{
			Title:           "Contract call",
			Narrative:       narrative,
			ConfidenceLabel: confidenceLabel(ctx.Confidence),
			ConfidenceValue: ctx.Confidence,
			Actors:          buildActors(ctx.Actors),
			Evidence:        buildEvidence(ctx),
			Signals:         buildSignals(ctx),
		}
	}
	if protocol != "an actor" {
		narrative += " on " + protocol
	}
	narrative += "."
	return TxNarrative{
		Title:           "Contract call",
		Narrative:       narrative,
		ConfidenceLabel: confidenceLabel(ctx.Confidence),
		ConfidenceValue: ctx.Confidence,
		Actors:          buildActors(ctx.Actors),
		Evidence:        buildEvidence(ctx),
		Signals:         buildSignals(ctx),
	}
}

type genericNarrator struct{}

func (genericNarrator) Matches(ctx semanticTxContext) bool { return true }

func (genericNarrator) Build(ctx semanticTxContext) TxNarrative {
	actor := actorLabel(firstActorByRole(ctx.Actors, "effective_actor"))
	title := titleish(strings.ReplaceAll(ctx.TxType, "_", " "))
	if title == "" {
		title = "Transaction activity"
	}
	narrative := fmt.Sprintf("%s submitted a %s", actor, strings.ReplaceAll(ctx.TxType, "_", " "))
	if len(ctx.OperationTypes) > 0 {
		narrative += fmt.Sprintf(" involving %s", strings.Join(ctx.OperationTypes, ", "))
	}
	narrative += "."
	return TxNarrative{
		Title:           title,
		Narrative:       narrative,
		ConfidenceLabel: confidenceLabel(ctx.Confidence),
		ConfidenceValue: ctx.Confidence,
		Actors:          buildActors(ctx.Actors),
		Evidence:        buildEvidence(ctx),
		Signals:         buildSignals(ctx),
	}
}

func buildActors(actors []gateway.SemanticActor) []TxActor {
	out := make([]TxActor, 0, len(actors))
	for _, a := range actors {
		label := deref(a.Label)
		if label == "" {
			label = gateway.ShortAddress(a.ActorID)
		}
		href := "#"
		if strings.HasPrefix(a.ActorID, "G") {
			href = "/account/" + a.ActorID
		} else if strings.HasPrefix(a.ActorID, "C") {
			href = "/contracts/" + a.ActorID
		}
		out = append(out, TxActor{
			Label:     label,
			Role:      primaryRole(a.Roles),
			ActorType: titleish(strings.ReplaceAll(a.ActorType, "_", " ")),
			Href:      href,
		})
	}
	return out
}

func buildEvidence(ctx semanticTxContext) []TxEvidence {
	evidence := []TxEvidence{{Label: "Type", Value: ctx.TxType}}
	if ctx.PrimaryFunction != "" {
		evidence = append(evidence, TxEvidence{Label: "Function", Value: ctx.PrimaryFunction})
	}
	if ctx.PrimaryContract != "" {
		evidence = append(evidence, TxEvidence{Label: "Contract", Value: ctx.PrimaryContract})
	}
	if ctx.PrimaryOperation != "" {
		evidence = append(evidence, TxEvidence{Label: "Primary operation", Value: ctx.PrimaryOperation})
	}
	if ctx.AssetMovement != "" {
		evidence = append(evidence, TxEvidence{Label: "Asset movement", Value: ctx.AssetMovement})
	}
	if ctx.Subtype != "" {
		evidence = append(evidence, TxEvidence{Label: "Subtype", Value: ctx.Subtype})
	}
	if ctx.Confidence != "" {
		evidence = append(evidence, TxEvidence{Label: "Confidence", Value: ctx.Confidence})
	}
	if len(ctx.OperationTypes) > 0 {
		evidence = append(evidence, TxEvidence{Label: "Operation types", Value: strings.Join(ctx.OperationTypes, ", ")})
	}
	if ctx.EffectiveActorType != "" {
		evidence = append(evidence, TxEvidence{Label: "Effective actor", Value: ctx.EffectiveActorType})
	}
	if ctx.WalletInvolved {
		evidence = append(evidence, TxEvidence{Label: "Wallet involved", Value: "Yes"})
	}
	return evidence
}

func buildSignals(ctx semanticTxContext) []TxSignal {
	signals := []TxSignal{}
	if rule, ok := LookupFunctionNarrationWithContext(ctx.PrimaryFunction, ctx.PrimaryContractID, ctx.PrimaryContract, ctx.PrimaryProject); ok && rule.Signal != nil {
		summary := rule.Signal.Summary
		if summary != "" && ctx.PrimaryContract != "" && !strings.Contains(summary, ctx.PrimaryContract) && strings.Contains(strings.ToLower(summary), "contract") {
			summary = strings.TrimSuffix(summary, ".") + " on " + ctx.PrimaryContract + "."
		}
		signals = append(signals, TxSignal{Title: rule.Signal.Title, Severity: rule.Signal.Severity, Summary: summary})
	}
	if ctx.WalletInvolved {
		signals = append(signals, TxSignal{
			Title:    "Smart wallet involved",
			Severity: "info",
			Summary:  "Prism detected smart-wallet participation in this transaction.",
		})
	}
	switch ctx.TxType {
	case "smart_wallet_policy_update":
		summary := "This transaction changed wallet policy or signer permissions."
		if ctx.Subtype != "" {
			summary = "This transaction performed a wallet policy action: " + strings.ReplaceAll(ctx.Subtype, "_", " ") + "."
		}
		signals = append(signals, TxSignal{Title: "Policy change", Severity: "warn", Summary: summary})
	case "smart_wallet_multisig_approval":
		signals = append(signals, TxSignal{Title: "Approval flow", Severity: "info", Summary: "This transaction looks like part of a multisig approval flow."})
	case "smart_wallet_transfer":
		signals = append(signals, TxSignal{Title: "Value transfer", Severity: "info", Summary: "This smart wallet moved value to another actor."})
	case "smart_wallet_swap":
		signals = append(signals, TxSignal{Title: "DEX interaction", Severity: "info", Summary: "This transaction swapped one asset for another through a protocol route."})
	case "wallet_mediated_protocol_interaction":
		signals = append(signals, TxSignal{Title: "Protocol interaction", Severity: "info", Summary: "A wallet appears to be mediating an interaction with a protocol contract."})
	}
	if strings.Contains(strings.ToLower(strings.Join(ctx.OperationTypes, ",")), "invoke") {
		signals = append(signals, TxSignal{Title: "Contract execution", Severity: "info", Summary: "This transaction includes contract execution activity."})
	}
	return dedupeSignals(signals)
}

func dedupeSignals(in []TxSignal) []TxSignal {
	seen := map[string]bool{}
	out := make([]TxSignal, 0, len(in))
	for _, s := range in {
		key := s.Title + ":" + s.Summary
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, s)
	}
	return out
}

func firstActorByRole(actors []gateway.SemanticActor, role string) *gateway.SemanticActor {
	for _, a := range actors {
		for _, r := range a.Roles {
			if r == role {
				copy := a
				return &copy
			}
		}
	}
	return nil
}

func actorLabel(actor *gateway.SemanticActor) string {
	if actor == nil {
		return "an actor"
	}
	if actor.Label != nil && *actor.Label != "" {
		return *actor.Label
	}
	if actor.ActorID != "" {
		return gateway.ShortAddress(actor.ActorID)
	}
	return "an actor"
}

func primaryFunction(ctx semanticTxContext) string {
	for _, op := range ctx.Operations {
		if op.FunctionName != "" {
			return op.FunctionName
		}
	}
	return ""
}

func primaryContractID(ctx semanticTxContext) string {
	if actor := firstActorByRole(ctx.Actors, "protocol"); actor != nil && actor.ActorID != "" {
		return actor.ActorID
	}
	for _, op := range ctx.Operations {
		if op.ContractID != "" {
			return op.ContractID
		}
	}
	return ""
}

func primaryContract(ctx semanticTxContext) string {
	if actor := firstActorByRole(ctx.Actors, "protocol"); actor != nil {
		return actorLabel(actor)
	}
	for _, op := range ctx.Operations {
		if op.ContractID != "" {
			return gateway.ShortAddress(op.ContractID)
		}
	}
	return ""
}

func primaryOperation(ctx semanticTxContext) string {
	for _, op := range ctx.Operations {
		if op.TypeName != "" {
			return gateway.OperationDisplayName(op.TypeName)
		}
	}
	if len(ctx.OperationTypes) > 0 {
		return strings.Join(ctx.OperationTypes, ", ")
	}
	return ""
}

func primaryAssetMovement(ctx semanticTxContext) string {
	inAmt, inAsset, outAmt, outAsset := swapAmounts(ctx)
	if inAmt != "" && outAmt != "" {
		return strings.TrimSpace(fmt.Sprintf("%s %s → %s %s", inAmt, inAsset, outAmt, outAsset))
	}
	amt, asset := transferAmount(ctx)
	if amt != "" {
		return strings.TrimSpace(fmt.Sprintf("%s %s", amt, asset))
	}
	return ""
}

func formatFunction(name string) string {
	if name == "" {
		return "a contract function"
	}
	return HumanizeFunctionName(name) + "()"
}

func describeFunctionCall(functionName, contractID, contractName, amount, asset string) string {
	if functionName == "" {
		if contractName != "" {
			return "called a contract on " + contractName
		}
		return "called a contract"
	}
	if rule, ok := LookupFunctionNarrationWithContext(functionName, contractID, contractName, InferProject(contractName)); ok && rule.Phrase != "" {
		if contractName != "" {
			return rule.Phrase + " on " + contractName
		}
		return rule.Phrase
	}
	msg := "called " + formatFunction(functionName)
	if contractName != "" {
		msg += " on " + contractName
	}
	if amount != "" {
		msg += " to move " + amount
		if asset != "" {
			msg += " " + asset
		}
	}
	return msg
}

func describeSwapCall(functionName, contractID, contractName, inAmt, inAsset, outAmt, outAsset string) string {
	if functionName == "swap" || strings.Contains(strings.ToLower(functionName), "swap") {
		msg := "swapped"
		if inAmt != "" {
			msg += " " + inAmt
			if inAsset != "" {
				msg += " " + inAsset
			}
		}
		if outAmt != "" {
			msg += " for " + outAmt
			if outAsset != "" {
				msg += " " + outAsset
			}
		}
		if contractName != "" {
			msg += " through " + contractName
		}
		return msg
	}
	msg := describeFunctionCall(functionName, contractID, contractName, inAmt, inAsset)
	if outAmt != "" {
		msg += " for " + outAmt
		if outAsset != "" {
			msg += " " + outAsset
		}
	}
	return msg
}

func fundingAmount(ctx semanticTxContext) (string, string) {
	if ctx.Assets.Received != nil {
		return ctx.Assets.Received.Amount, ctx.Assets.Received.Asset
	}
	if ctx.Assets.Sent != nil {
		return ctx.Assets.Sent.Amount, ctx.Assets.Sent.Asset
	}
	return firstMovementAmount(ctx)
}

func transferAmount(ctx semanticTxContext) (string, string) {
	if ctx.Assets.Sent != nil {
		return ctx.Assets.Sent.Amount, ctx.Assets.Sent.Asset
	}
	return firstMovementAmount(ctx)
}

func swapAmounts(ctx semanticTxContext) (string, string, string, string) {
	inAmt, inAsset := "", ""
	outAmt, outAsset := "", ""
	if ctx.Assets.Sent != nil {
		inAmt, inAsset = ctx.Assets.Sent.Amount, ctx.Assets.Sent.Asset
	}
	if ctx.Assets.Received != nil {
		outAmt, outAsset = ctx.Assets.Received.Amount, ctx.Assets.Received.Asset
	}
	if inAmt == "" && len(ctx.Assets.Movements) > 0 {
		inAmt = ctx.Assets.Movements[0].Amount
		inAsset = ctx.Assets.Movements[0].Asset
	}
	if outAmt == "" && len(ctx.Assets.Movements) > 1 {
		outAmt = ctx.Assets.Movements[1].Amount
		outAsset = ctx.Assets.Movements[1].Asset
	}
	return inAmt, inAsset, outAmt, outAsset
}

func firstMovementAmount(ctx semanticTxContext) (string, string) {
	if len(ctx.Assets.Movements) > 0 {
		return ctx.Assets.Movements[0].Amount, ctx.Assets.Movements[0].Asset
	}
	return "", ""
}

func primaryRole(roles []string) string {
	preferred := []string{"effective_actor", "submitter", "protocol", "sender", "receiver", "beneficiary", "counterparty"}
	for _, want := range preferred {
		for _, have := range roles {
			if have == want {
				return titleish(strings.ReplaceAll(have, "_", " "))
			}
		}
	}
	if len(roles) > 0 {
		return titleish(strings.ReplaceAll(roles[0], "_", " "))
	}
	return "Actor"
}

func actionPhrase(action string) string {
	action = strings.TrimSpace(strings.ReplaceAll(action, "_", " "))
	if action == "" {
		return "updated its policy"
	}
	if strings.HasPrefix(action, "allow ") || strings.HasPrefix(action, "add ") || strings.HasPrefix(action, "remove ") || strings.HasPrefix(action, "update ") {
		return action
	}
	return "performed " + action
}

func confidenceLabel(v string) string {
	if f, err := strconv.ParseFloat(v, 64); err == nil {
		switch {
		case f >= 0.9:
			return "High confidence"
		case f >= 0.7:
			return "Medium confidence"
		default:
			return "Heuristic"
		}
	}
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "high":
		return "High confidence"
	case "medium":
		return "Medium confidence"
	case "low", "heuristic":
		return "Heuristic"
	}
	if strings.Contains(strings.ToLower(v), "heur") {
		return "Heuristic"
	}
	return "Classified"
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func titleish(s string) string {
	if s == "" {
		return s
	}
	parts := strings.Fields(s)
	for i, p := range parts {
		if p == "" {
			continue
		}
		parts[i] = strings.ToUpper(p[:1]) + strings.ToLower(p[1:])
	}
	return strings.Join(parts, " ")
}
