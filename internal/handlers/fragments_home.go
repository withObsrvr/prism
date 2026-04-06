package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/withObsrvr/prism/internal/gateway"
	"github.com/withObsrvr/prism/internal/templates/fragments"
	"github.com/withObsrvr/prism/internal/templates/pages"
)

// ── Fragment 1: Network Pulse ──

// HomeNetworkPulseFragment returns the 5-metric Network Pulse grid.
func (h *Handlers) HomeNetworkPulseFragment(w http.ResponseWriter, r *http.Request) {
	network := networkFromRequest(r)

	var data pages.HomeData

	if h.useLiveData(r) {
		if live, err := h.buildHomeNetworkPulse(r, network); err == nil {
			data = live
		} else {
			h.Logger.Warn("live network pulse failed, falling back to mock", "error", err, "fragment", "network-pulse")
		}
	}

	if data.LatestLedger == "" {
		data = mockHomeData(network)
	}

	if err := fragments.HomeNetworkPulse(data).Render(r.Context(), w); err != nil {
		h.renderFragmentError(w, r, "Could not load network stats", err)
	}
}

// buildHomeNetworkPulse fetches network stats for the 5-metric pulse grid.
func (h *Handlers) buildHomeNetworkPulse(r *http.Request, network string) (pages.HomeData, error) {
	ctx := r.Context()

	// Bronze stats for accurate latest ledger and 24h counts.
	bronze, err := h.Gateway.GetBronzeNetworkStats(ctx, network)
	if err != nil {
		return pages.HomeData{}, fmt.Errorf("fetching bronze stats: %w", err)
	}

	latestSeq := bronze.Ledger.LatestSequence
	txCount24H := bronze.Transactions24H.Total
	sorobanCalls := bronze.Transactions24H.SorobanCount

	// Ledger age.
	ledgerAge := "just now"
	if t, err := time.Parse(time.RFC3339, bronze.Ledger.ClosedAt); err == nil {
		ledgerAge = gateway.FormatAge(t)
	}

	// TPS from 24h count.
	tpsAvg := float64(txCount24H) / 86400
	tpsPeak := tpsAvg * 3 // estimate

	data := pages.HomeData{
		Network:      network,
		LatestLedger: gateway.FormatNumber(latestSeq),
		LedgerAge:    ledgerAge,
		TxCount24H:   gateway.FormatNumber(txCount24H),
		TxChange:     "", // Not available without historical comparison
		TPSAvg:       fmt.Sprintf("%.1f", tpsAvg),
		TPSPeak:      fmt.Sprintf("%.0f", tpsPeak),
		SorobanCalls: gateway.FormatNumber(sorobanCalls),
		SorobanChange: "",
		Validators:   35, // Needs Radar client — static for now
	}

	// Try fee stats for the fee guide values.
	if fees, err := h.Gateway.GetFeeStats(ctx, network, "24h"); err == nil {
		data.FeeEconomy = gateway.FormatNumber(fees.MinFee)
		data.FeeStandard = gateway.FormatNumber(fees.MedianFee)
		data.FeePriority = gateway.FormatNumber(fees.P99Fee)
		data.SurgeActive = fees.SurgeActive
	} else {
		// Fallback fee values.
		data.FeeEconomy = "100"
		data.FeeStandard = "200"
		data.FeePriority = "500"
	}

	return data, nil
}

// ── Fragment 2: Recent Transactions ──

// HomeRecentTxsFragment returns the Latest Transactions table.
func (h *Handlers) HomeRecentTxsFragment(w http.ResponseWriter, r *http.Request) {
	network := networkFromRequest(r)

	var txs []pages.HomeTx

	if h.useLiveData(r) {
		if live, err := h.buildHomeRecentTxs(r, network); err == nil {
			txs = live
		} else {
			h.Logger.Warn("live recent txs failed, falling back to mock", "error", err, "fragment", "recent-txs")
		}
	}

	if txs == nil {
		txs = mockHomeData(network).Transactions
	}

	if err := fragments.HomeRecentTxs(txs).Render(r.Context(), w); err != nil {
		h.renderFragmentError(w, r, "Could not load transactions", err)
	}
}

// buildHomeRecentTxs fetches recent transactions with decoded summaries.
// It tries the batch decode endpoint first (rich summaries from silver),
// then falls back to basic bronze transaction data if silver is lagging.
func (h *Handlers) buildHomeRecentTxs(r *http.Request, network string) ([]pages.HomeTx, error) {
	ctx := r.Context()

	// Get latest sequence to compute range.
	bronze, err := h.Gateway.GetBronzeNetworkStats(ctx, network)
	if err != nil {
		return nil, fmt.Errorf("fetching bronze stats: %w", err)
	}

	latestSeq := bronze.Ledger.LatestSequence
	startSeq := latestSeq - 3
	if startSeq < 1 {
		startSeq = 1
	}

	// Fetch raw txs from bronze (real-time).
	rawTxs, err := h.Gateway.GetTransactions(ctx, network, startSeq, latestSeq, 6, "desc")
	if err != nil {
		return nil, fmt.Errorf("fetching transactions: %w", err)
	}

	// Try batch decoded endpoint for rich summaries (may fail if silver is lagging).
	hashes := make([]string, 0, len(rawTxs))
	for _, tx := range rawTxs {
		hashes = append(hashes, tx.TransactionHash)
	}

	decodedMap := make(map[string]*gateway.DecodedTransaction)
	if len(hashes) > 0 {
		if resp, err := h.Gateway.GetBatchDecodedTransactions(ctx, network, hashes, 0, 0); err == nil {
			for i := range resp.Transactions {
				dt := &resp.Transactions[i]
				decodedMap[dt.TxHash] = dt
			}
		} else {
			h.Logger.Debug("batch decode unavailable (silver may be lagging), using basic data", "error", err)
		}
	}

	result := make([]pages.HomeTx, 0, len(rawTxs))
	for _, tx := range rawTxs {
		age := "—"
		if t, err := time.Parse(time.RFC3339, tx.CreatedAt); err == nil {
			age = gateway.FormatAge(t)
		}

		shortHash := gateway.ShortHash(tx.TransactionHash)
		txType := "classic"
		typeLabel := "Classic"
		summary := fmt.Sprintf("From %s", gateway.ShortAddress(tx.SourceAccount))

		// Enrich from decoded data if available.
		if dt, ok := decodedMap[tx.TransactionHash]; ok && dt.Summary != nil {
			txType = dt.Summary.Type
			summary = dt.Summary.Description

			switch txType {
			case "transfer":
				typeLabel = "Transfer"
			case "mint":
				typeLabel = "Mint"
			case "burn":
				typeLabel = "Burn"
			case "swap":
				typeLabel = "Swap"
			case "contract_call":
				typeLabel = "Contract"
			case "multi_op":
				typeLabel = "Multi-Op"
			case "multi_transfer":
				typeLabel = "Transfers"
			default:
				typeLabel = "Classic"
			}
		} else {
			// Basic classification from bronze data only.
			if tx.OperationCount > 3 {
				txType = "multi_op"
				typeLabel = "Multi-Op"
				summary = fmt.Sprintf("%d operations from %s", tx.OperationCount, gateway.ShortAddress(tx.SourceAccount))
			}
		}

		result = append(result, pages.HomeTx{
			Hash:      tx.TransactionHash,
			ShortHash: shortHash,
			Type:      txType,
			TypeLabel: typeLabel,
			Summary:   summary,
			From:      gateway.ShortAddress(tx.SourceAccount),
			Ops:       fmt.Sprintf("%d ops", tx.OperationCount),
			Fee:       gateway.FormatNumber(tx.MaxFee) + " str",
			Age:       age,
		})
	}

	return result, nil
}

// ── Fragment 3: Recent Ledgers ──

// HomeRecentLedgersFragment returns the Latest Ledgers sidebar.
func (h *Handlers) HomeRecentLedgersFragment(w http.ResponseWriter, r *http.Request) {
	network := networkFromRequest(r)

	var ledgers []pages.HomeLedger

	if h.useLiveData(r) {
		if live, err := h.buildHomeLedgers(r, network); err == nil {
			ledgers = live
		} else {
			h.Logger.Warn("live ledgers failed, falling back to mock", "error", err, "fragment", "recent-ledgers")
		}
	}

	if ledgers == nil {
		ledgers = mockHomeData(network).Ledgers
	}

	if err := fragments.HomeRecentLedgers(ledgers).Render(r.Context(), w); err != nil {
		h.renderFragmentError(w, r, "Could not load ledgers", err)
	}
}

// buildHomeLedgers fetches recent ledgers from the gateway.
func (h *Handlers) buildHomeLedgers(r *http.Request, network string) ([]pages.HomeLedger, error) {
	ctx := r.Context()

	bronze, err := h.Gateway.GetBronzeNetworkStats(ctx, network)
	if err != nil {
		return nil, fmt.Errorf("fetching bronze stats: %w", err)
	}

	latestSeq := bronze.Ledger.LatestSequence
	startSeq := latestSeq - 5
	if startSeq < 1 {
		startSeq = 1
	}

	ledgers, err := h.Gateway.GetLedgers(ctx, network, startSeq, latestSeq, 6, "desc")
	if err != nil {
		return nil, fmt.Errorf("fetching ledgers: %w", err)
	}

	result := make([]pages.HomeLedger, 0, len(ledgers))
	for i, l := range ledgers {
		age := "—"
		if t, err := time.Parse(time.RFC3339, l.ClosedAt); err == nil {
			age = gateway.FormatAge(t)
		}
		result = append(result, pages.HomeLedger{
			Sequence:    gateway.FormatNumber(l.Sequence),
			SequenceRaw: fmt.Sprintf("%d", l.Sequence),
			Age:         age,
			TxCount:     fmt.Sprintf("%d txs", l.SuccessfulTxCount),
			OpCount:     fmt.Sprintf("%d ops", l.OperationCount),
			IsLatest:    i == 0,
		})
	}

	return result, nil
}

// ── Fragment 4: Trending Contracts ──

// HomeTrendingContractsFragment returns the Trending Contracts table.
func (h *Handlers) HomeTrendingContractsFragment(w http.ResponseWriter, r *http.Request) {
	network := networkFromRequest(r)

	var contracts []pages.HomeContract

	if h.useLiveData(r) {
		if live, err := h.buildHomeContracts(r, network); err == nil {
			contracts = live
		} else {
			h.Logger.Warn("live contracts failed, falling back to mock", "error", err, "fragment", "trending-contracts")
		}
	}

	if contracts == nil {
		contracts = mockHomeData(network).Contracts
	}

	if err := fragments.HomeTrendingContracts(contracts).Render(r.Context(), w); err != nil {
		h.renderFragmentError(w, r, "Could not load contracts", err)
	}
}

// buildHomeContracts fetches top contracts and enriches with names and tags
// from the semantic contracts registry.
func (h *Handlers) buildHomeContracts(r *http.Request, network string) ([]pages.HomeContract, error) {
	ctx := r.Context()

	contracts, err := h.Gateway.GetTopContracts(ctx, network, 5)
	if err != nil {
		return nil, fmt.Errorf("fetching top contracts: %w", err)
	}

	// Build a lookup map from semantic contracts for name enrichment.
	semanticMap := make(map[string]*gateway.SemanticContract)
	if semContracts, err := h.Gateway.GetSemanticContracts(ctx, network, 200); err == nil {
		for i := range semContracts {
			semanticMap[semContracts[i].ContractID] = &semContracts[i]
		}
	}

	result := make([]pages.HomeContract, 0, len(contracts))
	for i, c := range contracts {
		name := gateway.ShortAddress(c.ContractID)
		tag, tagColor := inferContractTag(c.TopFunction, nil)

		// Enrich from semantic registry if available.
		if sem, ok := semanticMap[c.ContractID]; ok {
			if sem.TokenName != nil && *sem.TokenName != "" {
				name = *sem.TokenName
			} else if sem.TokenSymbol != nil && *sem.TokenSymbol != "" {
				name = *sem.TokenSymbol
			}
			tag, tagColor = inferContractTag(c.TopFunction, sem.ObservedFunctions)
			if sem.ContractType == "sep41_token" {
				tag = "Token"
				tagColor = "cyan"
			}
			// Smart wallet detection from semantic registry.
			if sem.WalletType != nil && *sem.WalletType != "" {
				tag = walletTypeLabel(*sem.WalletType)
				tagColor = "violet"
			}
			if sem.ContractType == "smart_wallet" {
				tag = walletTypeLabel("")
				tagColor = "violet"
				if sem.WalletType != nil && *sem.WalletType != "" {
					tag = walletTypeLabel(*sem.WalletType)
				}
			}
		}

		result = append(result, pages.HomeContract{
			Rank:        i + 1,
			Name:        name,
			Tag:         tag,
			TagColor:    tagColor,
			Address:     gateway.ShortAddress(c.ContractID),
			Invocations: gateway.FormatNumber(c.TotalCalls),
			Change:      "",
			IsPositive:  true,
		})
	}

	return result, nil
}

// inferContractTag determines a human-readable tag and color from function names.
func inferContractTag(topFunction string, observedFunctions []string) (string, string) {
	// Check all function names for patterns.
	allFns := observedFunctions
	if topFunction != "" {
		allFns = append([]string{topFunction}, allFns...)
	}

	hasFn := func(names ...string) bool {
		for _, fn := range allFns {
			for _, name := range names {
				if fn == name {
					return true
				}
			}
		}
		return false
	}

	switch {
	case hasFn("swap", "swap_exact_tokens_for_tokens", "swap_tokens_for_exact_tokens",
		"add_liquidity", "remove_liquidity", "router_swap"):
		return "DEX", "emerald"
	case hasFn("transfer", "mint", "burn", "approve", "balance", "decimals", "name", "symbol"):
		return "Token", "cyan"
	case hasFn("deposit", "withdraw", "borrow", "repay", "liquidate"):
		return "Lending", "amber"
	case hasFn("set_price", "write_prices", "get_price"):
		return "Oracle", "blue"
	case hasFn("stake", "unstake", "claim_rewards"):
		return "Staking", "violet"
	case hasFn("set_manual_override", "reset_all_circuit_breakers"):
		return "Controller", "gray"
	case hasFn("add_claim", "claim"):
		return "Airdrop", "emerald"
	default:
		return "Contract", "violet"
	}
}

// ── Fragment 5: Sidebar (Assets + Fees) ──

// HomeSidebarFragment returns the Top Assets + Fee Guide sidebar.
func (h *Handlers) HomeSidebarFragment(w http.ResponseWriter, r *http.Request) {
	network := networkFromRequest(r)

	var data pages.HomeData

	if h.useLiveData(r) {
		if live, err := h.buildHomeSidebar(r, network); err == nil {
			data = live
		} else {
			h.Logger.Warn("live sidebar failed, falling back to mock", "error", err, "fragment", "sidebar")
		}
	}

	if data.FeeEconomy == "" {
		data = mockHomeData(network)
	}

	if err := fragments.HomeSidebar(data).Render(r.Context(), w); err != nil {
		h.renderFragmentError(w, r, "Could not load sidebar", err)
	}
}

// buildHomeSidebar fetches top assets and fee stats for the sidebar.
func (h *Handlers) buildHomeSidebar(r *http.Request, network string) (pages.HomeData, error) {
	ctx := r.Context()

	data := pages.HomeData{Network: network}

	// Fetch top assets by 24h volume.
	assetsResp, err := h.Gateway.GetAssets(ctx, network, 4, "volume_24h", "desc")
	if err != nil {
		return data, fmt.Errorf("fetching assets: %w", err)
	}

	colors := []string{"bg-amber-500", "bg-blue-500", "bg-emerald-500", "bg-violet-500", "bg-cyan-500", "bg-red-500"}
	for i, a := range assetsResp.Assets {
		code := a.AssetCode
		if code == "" {
			code = "XLM"
		}
		initial := string(code[0])
		bgColor := colors[i%len(colors)]

		data.Assets = append(data.Assets, pages.HomeAsset{
			Code:       code,
			Issuer:     gateway.ShortAddress(a.AssetIssuer),
			Initial:    initial,
			BgColor:    bgColor,
			Volume:     gateway.FormatAbbrev(parseVolume(a.Volume24H)),
			Change:     "", // Not available
			IsPositive: true,
		})
	}

	// Fetch fee stats.
	if fees, err := h.Gateway.GetFeeStats(ctx, network, "24h"); err == nil {
		data.FeeEconomy = gateway.FormatNumber(fees.MinFee)
		data.FeeStandard = gateway.FormatNumber(fees.MedianFee)
		data.FeePriority = gateway.FormatNumber(fees.P99Fee)
		data.SurgeActive = fees.SurgeActive
	} else {
		// Fallback — at least provide the data structure.
		data.FeeEconomy = "100"
		data.FeeStandard = "200"
		data.FeePriority = "500"
	}

	return data, nil
}

// parseVolume converts a string volume (possibly decimal) to an int64 for formatting.
func parseVolume(s string) int64 {
	var v float64
	fmt.Sscanf(s, "%f", &v)
	return int64(v)
}

// walletTypeLabel returns a human-readable label for a smart wallet type.
func walletTypeLabel(walletType string) string {
	switch walletType {
	case "crossmint":
		return "Crossmint Wallet"
	case "openzeppelin":
		return "OZ Wallet"
	case "sep50":
		return "Smart Wallet"
	default:
		return "Smart Wallet"
	}
}
