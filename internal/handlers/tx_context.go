package handlers

import (
	"context"
	"fmt"
	"strings"

	"github.com/withObsrvr/prism/internal/gateway"
)

type NormalizedTxActor struct {
	ID        string
	Short     string
	Label     string
	ActorType string
	Role      string
	Href      string
}

type NormalizedSmartWalletContext struct {
	Detected        bool
	ContractID      string
	ContractShort   string
	WalletType      string
	Implementation  string
	Confidence      float64
	DetectionSource string // semantic, smart_wallet_endpoint, mixed
}

type NormalizedTxContext struct {
	Submitter          *NormalizedTxActor
	FeePayer           *NormalizedTxActor
	EffectiveActor     *NormalizedTxActor
	DownstreamContract *NormalizedTxActor
	DownstreamFunction string
	Wallet             *NormalizedSmartWalletContext
	WalletInvolved     bool
	SourceOfTruth      string // semantic, fallback, mixed
}

func (h *Handlers) buildNormalizedTxContext(ctx context.Context, network string, txFull *gateway.TxFull, semanticTx *gateway.SemanticTransactionResponse) (*NormalizedTxContext, error) {
	if txFull == nil {
		return nil, nil
	}

	norm := &NormalizedTxContext{SourceOfTruth: "fallback"}
	source := txFull.Transaction.SourceAccount
	if source == "" && semanticTx != nil && semanticTx.Transaction.SourceAccount != nil {
		source = *semanticTx.Transaction.SourceAccount
	}
	if source != "" {
		norm.Submitter = newNormalizedActor(source, "classic_account", "submitter")
		// The receipt does not currently expose the outer fee source. Assuming
		// the submitter paid would be false for fee-bump transactions, so leave
		// this role unknown until the API supplies authoritative evidence.
	}

	var semanticProtocol *gateway.SemanticActor
	var semanticWallet *gateway.SemanticActor
	if semanticTx != nil {
		for _, actor := range semanticTx.Actors {
			if semanticWallet == nil && (actor.ActorType == "smart_wallet" || actor.Wallet != nil) {
				copy := actor
				semanticWallet = &copy
			}
			if semanticProtocol == nil && (hasRole(actor.Roles, "protocol") || hasRole(actor.Roles, "callee")) {
				copy := actor
				semanticProtocol = &copy
			}
		}
	}
	var semanticEffective *gateway.SemanticActor
	if semanticTx != nil {
		semanticEffective = selectSemanticEffectiveActor(semanticTx.Actors)
	}

	wallet := &NormalizedSmartWalletContext{}
	if semanticWallet != nil {
		wallet.Detected = true
		wallet.ContractID = semanticWallet.ActorID
		wallet.ContractShort = gateway.ShortAddress(semanticWallet.ActorID)
		wallet.DetectionSource = "semantic"
		if semanticWallet.Wallet != nil {
			wallet.WalletType = semanticWallet.Wallet.WalletType
			wallet.Implementation = semanticWallet.Wallet.Implementation
			wallet.Confidence = semanticWallet.Wallet.Confidence
		}
	}

	candidateIDs := candidateSmartWalletContracts(txFull, semanticTx)
	if !wallet.Detected {
		for _, cid := range candidateIDs {
			info, err := h.Gateway.GetSmartWalletInfo(ctx, network, cid)
			if err != nil || info == nil || !info.IsSmartWallet {
				continue
			}
			wallet.Detected = true
			wallet.ContractID = cid
			wallet.ContractShort = gateway.ShortAddress(cid)
			wallet.WalletType = info.WalletType
			wallet.Implementation = info.Implementation
			wallet.Confidence = info.Confidence
			wallet.DetectionSource = "smart_wallet_endpoint"
			break
		}
	} else {
		for _, cid := range candidateIDs {
			if cid == wallet.ContractID {
				continue
			}
			info, err := h.Gateway.GetSmartWalletInfo(ctx, network, cid)
			if err == nil && info != nil && info.IsSmartWallet {
				wallet.DetectionSource = "mixed"
				if wallet.WalletType == "" {
					wallet.WalletType = info.WalletType
				}
				if wallet.Implementation == "" {
					wallet.Implementation = info.Implementation
				}
				if wallet.Confidence == 0 {
					wallet.Confidence = info.Confidence
				}
				break
			}
		}
	}

	if wallet.Detected {
		norm.Wallet = wallet
		norm.WalletInvolved = true
		norm.EffectiveActor = newNormalizedActor(wallet.ContractID, "smart_wallet", "effective_actor")
		switch wallet.DetectionSource {
		case "semantic":
			norm.SourceOfTruth = "semantic"
		case "mixed":
			norm.SourceOfTruth = "mixed"
		default:
			norm.SourceOfTruth = "fallback"
		}
	} else if semanticEffective != nil {
		norm.EffectiveActor = newNormalizedActor(semanticEffective.ActorID, semanticEffective.ActorType, "effective_actor")
		norm.SourceOfTruth = "semantic"
	}

	// An invoke-host-function operation names the contract that received the
	// top-level call. That is stronger primary-target evidence than actor-array
	// order: token and helper contracts touched during execution may all carry a
	// generic "protocol" role. A smart-wallet execute() operation is the one
	// exception; its observed wallet-to-protocol call-graph edge remains the
	// useful downstream target.
	for _, op := range txFull.Operations {
		if op.ContractID == "" || (norm.Wallet != nil && op.ContractID == norm.Wallet.ContractID) {
			continue
		}
		norm.DownstreamContract = newNormalizedActor(op.ContractID, "contract", "protocol")
		if op.FunctionName != "" {
			norm.DownstreamFunction = op.FunctionName
		}
		break
	}

	if norm.DownstreamContract == nil && semanticProtocol != nil && semanticProtocol.ActorID != "" {
		norm.DownstreamContract = newNormalizedActor(semanticProtocol.ActorID, semanticProtocol.ActorType, "protocol")
		norm.SourceOfTruth = mergeSourceOfTruth(norm.SourceOfTruth, "semantic")
	}
	if semanticTx != nil && len(semanticTx.CallGraph) > 0 {
		for _, edge := range semanticTx.CallGraph {
			if edge.To == "" {
				continue
			}
			if norm.Wallet != nil && norm.Wallet.ContractID != "" && edge.From != norm.Wallet.ContractID {
				continue
			}
			if norm.DownstreamContract == nil || (norm.Wallet != nil && edge.To != norm.Wallet.ContractID) {
				norm.DownstreamContract = newNormalizedActor(edge.To, "contract", "protocol")
			}
			if norm.DownstreamFunction == "" && edge.Function != "" {
				norm.DownstreamFunction = edge.Function
			}
			norm.SourceOfTruth = mergeSourceOfTruth(norm.SourceOfTruth, "semantic")
			break
		}
	}

	for _, op := range txFull.Operations {
		if norm.DownstreamFunction == "" && op.FunctionName != "" {
			norm.DownstreamFunction = op.FunctionName
		}
		if norm.DownstreamContract == nil && op.ContractID != "" {
			if norm.Wallet == nil || op.ContractID != norm.Wallet.ContractID {
				norm.DownstreamContract = newNormalizedActor(op.ContractID, "contract", "protocol")
			}
		}
	}

	if norm.EffectiveActor == nil {
		if norm.Submitter != nil {
			norm.EffectiveActor = norm.Submitter
		} else if source != "" {
			norm.EffectiveActor = newNormalizedActor(source, "classic_account", "effective_actor")
		}
	}

	// Avoid misleading target display when the only thing we know is the wallet's
	// own execute() entrypoint and we have no downstream delegated call yet.
	if norm.Wallet != nil && norm.DownstreamContract != nil && norm.DownstreamContract.ID == norm.Wallet.ContractID {
		if norm.DownstreamFunction == "" || norm.DownstreamFunction == "execute" {
			norm.DownstreamContract = nil
			norm.DownstreamFunction = ""
		}
	}

	return norm, nil
}

// selectSemanticEffectiveActor returns an actor only when the semantic packet
// identifies one unambiguous address for the role. Event-derived actor sets can
// legitimately contain several senders, receivers, and authorized contracts;
// choosing the first turns response ordering into a false identity claim.
func selectSemanticEffectiveActor(actors []gateway.SemanticActor) *gateway.SemanticActor {
	seen := make(map[string]struct{})
	var selected *gateway.SemanticActor
	for _, actor := range actors {
		if actor.ActorID == "" || !hasRole(actor.Roles, "effective_actor") {
			continue
		}
		if _, duplicate := seen[actor.ActorID]; duplicate {
			continue
		}
		seen[actor.ActorID] = struct{}{}
		if selected != nil {
			return nil
		}
		copy := actor
		selected = &copy
	}
	return selected
}

func candidateSmartWalletContracts(txFull *gateway.TxFull, semanticTx *gateway.SemanticTransactionResponse) []string {
	seen := map[string]bool{}
	out := make([]string, 0, 4)
	add := func(v string) {
		if v == "" || seen[v] {
			return
		}
		seen[v] = true
		out = append(out, v)
	}
	if txFull != nil {
		for _, op := range txFull.Operations {
			add(op.ContractID)
		}
	}
	if semanticTx != nil {
		for _, actor := range semanticTx.Actors {
			add(actor.ActorID)
			if actor.ContractID != nil {
				add(*actor.ContractID)
			}
		}
	}
	return out
}

func newNormalizedActor(id, actorType, role string) *NormalizedTxActor {
	if id == "" {
		return nil
	}
	return &NormalizedTxActor{
		ID:        id,
		Short:     gateway.ShortAddress(id),
		Label:     gateway.ShortAddress(id),
		ActorType: actorType,
		Role:      role,
		Href:      actorHref(id, actorType),
	}
}

func actorHref(id, actorType string) string {
	if id == "" {
		return ""
	}
	switch actorType {
	case "smart_wallet":
		return "/account/" + id + "/smart"
	case "contract", "protocol", "token", "sac":
		return "/contracts/" + id
	default:
		if strings.HasPrefix(id, "C") {
			return "/contracts/" + id
		}
		return "/account/" + id
	}
}

func hasRole(roles []string, role string) bool {
	for _, r := range roles {
		if r == role {
			return true
		}
	}
	return false
}

func mergeSourceOfTruth(current, next string) string {
	if current == "" || current == next {
		return next
	}
	if current == "fallback" && next == "semantic" || current == "semantic" && next == "fallback" {
		return "mixed"
	}
	return current
}

func walletConfidenceLabel(v float64) string {
	if v <= 0 {
		return ""
	}
	return fmt.Sprintf("%.0f%%", v*100)
}

func (h *Handlers) firstSmartWalletForDecodedTx(ctx context.Context, network string, dt *gateway.DecodedTransaction) *gateway.SmartWalletInfo {
	if dt == nil {
		return nil
	}
	for _, op := range dt.Operations {
		if op.ContractID == "" {
			continue
		}
		info, err := h.Gateway.GetSmartWalletInfo(ctx, network, op.ContractID)
		if err == nil && info != nil && info.IsSmartWallet {
			return info
		}
	}
	return nil
}
