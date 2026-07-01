package handlers

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/withObsrvr/prism/internal/gateway"
	"github.com/withObsrvr/prism/internal/humanize"
	"github.com/withObsrvr/prism/internal/templates/pages"
	pagesv2 "github.com/withObsrvr/prism/internal/templates/v2/pages"
)

func (h *Handlers) AccountPortfolio(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	network := networkFromRequest(r)

	var data pages.AccountData
	if h.useLiveData(r) {
		if live, err := h.buildAccountData(r, network, id); err == nil {
			data = live
		} else {
			h.Logger.Warn("live account shell data failed, falling back to mock", "error", err)
		}
	}
	if data.Address == "" {
		data = mockAccountData()
	}

	pages.AccountPortfolioV2(data).Render(r.Context(), w)
}

func (h *Handlers) AccountPortfolioV1(w http.ResponseWriter, r *http.Request) {
	data := mockAccountData()
	pages.AccountPortfolio(data).Render(r.Context(), w)
}

func (h *Handlers) GAccountDetailV2(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	network := networkFromRequest(r)

	// Render a cheap SSR shell first. Live account evidence is loaded by htmx
	// fragments so a slow gateway cannot turn the whole page into a 503.
	data := accountShellData(id, true)
	if err := pagesv2.GAccountDetail(data, network).Render(r.Context(), w); err != nil {
		h.Logger.Error("render v2 g-account detail", "account", id, "error", err)
	}
}

func (h *Handlers) GAccountDetailMainFragment(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	network := networkFromRequest(r)
	data := h.accountLiveOrFallback(r, network, id)
	if err := pagesv2.GAccountDetailMain(data).Render(r.Context(), w); err != nil {
		h.Logger.Error("render v2 account main fragment", "account", id, "error", err)
		h.renderFragmentError(w, r, "Could not load account sections", err)
	}
}

func (h *Handlers) GAccountDetailRailFragment(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	network := networkFromRequest(r)
	data := h.accountLiveOrFallback(r, network, id)
	if err := pagesv2.GAccountDetailRail(data).Render(r.Context(), w); err != nil {
		h.Logger.Error("render v2 account rail fragment", "account", id, "error", err)
		h.renderFragmentError(w, r, "Could not load account facts", err)
	}
}

func (h *Handlers) accountLiveOrFallback(r *http.Request, network, id string) pages.AccountData {
	if h.useLiveData(r) {
		ctx, cancel := context.WithTimeout(r.Context(), 3500*time.Millisecond)
		defer cancel()
		if live, err := h.buildAccountData(r.WithContext(ctx), network, id); err == nil {
			return live
		} else {
			h.Logger.Warn("live v2 account fragment data failed, rendering safe fallback", "account", id, "error", err)
		}
	}
	return accountShellData(id, false)
}

func accountShellData(id string, loading bool) pages.AccountData {
	short := gateway.ShortAddress(id)
	if strings.TrimSpace(short) == "" {
		short = id
	}
	return pages.AccountData{
		Address:        id,
		ShortAddress:   short,
		TotalValue:     "—",
		XLMBalance:     "— XLM",
		Trustlines:     "0",
		ActiveOffers:   "0",
		Subentries:     "—",
		SequenceNumber: "unknown",
		IsFunded:       false,
		Loading:        loading,
		SignerCount:    "—",
	}
}

// buildFederatedActivities converts the federated hot+cold account-transaction history into the
// account page's activity rows, classifying each as Sent/Received from the source account so
// pre-handoff sends (served from cold storage) are clearly visible.
func buildFederatedActivities(txs []gateway.AccountTransaction, accountID string) []pages.AccountActivity {
	var activities []pages.AccountActivity
	for _, tx := range txs {
		badge, badgeColor := "Activity", "gray"
		if tx.SourceAccount != nil && *tx.SourceAccount == accountID {
			badge, badgeColor = "Sent", "blue"
		} else if tx.SourceAccount != nil && *tx.SourceAccount != "" {
			badge, badgeColor = "Received", "emerald"
		}
		for _, at := range tx.ActivityTypes {
			if strings.Contains(at, "soroban") || strings.Contains(at, "contract") {
				badge, badgeColor = "Contract", "violet"
			}
		}
		age := "—"
		if t, err := time.Parse(time.RFC3339, tx.ClosedAt); err == nil {
			d := time.Since(t)
			switch {
			case d.Hours() > 24:
				age = fmt.Sprintf("%.0fd ago", d.Hours()/24)
			case d.Hours() > 1:
				age = fmt.Sprintf("%.0fh ago", d.Hours())
			default:
				age = fmt.Sprintf("%.0fm ago", d.Minutes())
			}
		}
		summary := tx.Summary
		if summary == "" {
			summary = strings.Join(tx.ActivityTypes, ", ")
		}
		activities = append(activities, pages.AccountActivity{
			Summary:    summary,
			Badge:      badge,
			BadgeColor: badgeColor,
			TxHash:     tx.TransactionHash,
			ShortHash:  gateway.ShortHash(tx.TransactionHash),
			Time:       age,
		})
	}
	return activities
}

func (h *Handlers) buildAccountData(r *http.Request, network, accountID string) (pages.AccountData, error) {
	ctx := r.Context()

	overview, err := h.Gateway.GetAccountOverview(ctx, network, accountID)
	if err != nil {
		return pages.AccountData{}, fmt.Errorf("fetching account overview: %w", err)
	}

	acct := overview.Account
	if strings.HasPrefix(accountID, "C") {
		if walletInfo, err := h.Gateway.GetSmartWalletInfo(ctx, network, accountID); err == nil && walletInfo != nil && walletInfo.IsSmartWallet {
			acct.AccountID = accountID
		}
	}

	// Build balances from the overview (try dedicated endpoint too).
	var balances []pages.AccountBalance
	balResp, balErr := h.Gateway.GetAccountBalances(ctx, network, accountID)
	if balErr == nil && balResp != nil {
		for _, b := range balResp.Balances {
			assetType := "Classic"
			typeColor := "gray"
			code := b.AssetCode
			if b.AssetType == "native" {
				code = "XLM"
				assetType = "Native"
			}
			balances = append(balances, pages.AccountBalance{
				Code:      code,
				Name:      code,
				BgColor:   "bg-gray-600",
				Type:      assetType,
				TypeColor: typeColor,
				Balance:   b.Balance,
				ValueUSD:  "—",
			})
		}
	}

	// Build the activity list from the full hot+cold history (federated endpoint); fall back
	// to the overview's recent (hot-only) operations if the federated endpoint is unavailable.
	// Federated hot+cold history is great when cold reads are indexed, but for accounts whose
	// cold history is sparse/deep the cold scan is slow (no per-account index yet). Cap the call
	// so the page falls back to fast hot-only activity instead of hanging when that happens.
	var activities []pages.AccountActivity
	fctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if txs, ferr := h.Gateway.GetAccountTransactions(fctx, network, accountID, 200, "desc", ""); ferr == nil && txs != nil && len(txs.Transactions) > 0 {
		activities = buildFederatedActivities(txs.Transactions, accountID)
	} else {
		for _, op := range overview.RecentOperations {
			badge := op.TypeName
			badgeColor := "gray"
			if op.IsSorobanOp {
				badge = "Contract"
				badgeColor = "violet"
			} else if op.IsPaymentOp {
				badge = "Payment"
				badgeColor = "blue"
			}

			age := "—"
			if t, err := time.Parse(time.RFC3339, op.LedgerClosedAt); err == nil {
				d := time.Since(t)
				if d.Hours() > 24 {
					age = fmt.Sprintf("%.0fd ago", d.Hours()/24)
				} else if d.Hours() > 1 {
					age = fmt.Sprintf("%.0fh ago", d.Hours())
				} else {
					age = fmt.Sprintf("%.0fm ago", d.Minutes())
				}
			}

			summary := fmt.Sprintf("%s %s", gateway.ShortAddress(op.SourceAccount), op.TypeName)
			if op.Amount != "" {
				summary += fmt.Sprintf(" %s", op.Amount)
			}
			if op.IsSorobanOp && op.SorobanContract != "" {
				if info, err := h.Gateway.GetSmartWalletInfo(ctx, network, op.SorobanContract); err == nil && info != nil && info.IsSmartWallet {
					badge = "Wallet"
					badgeColor = "violet"
					label := walletTypeLabel(firstNonEmpty(info.WalletType, info.Implementation))
					fn := op.SorobanFunction
					if fn == "" {
						fn = "execute"
					}
					summary = fmt.Sprintf("%s wallet %s() via %s", label, fn, gateway.ShortAddress(op.SorobanContract))
				}
			}

			activities = append(activities, pages.AccountActivity{
				Summary:    summary,
				Badge:      badge,
				BadgeColor: badgeColor,
				TxHash:     op.TransactionHash,
				ShortHash:  gateway.ShortHash(op.TransactionHash),
				Time:       age,
			})
		}
	}

	// Build signers.
	var signers []pages.AccountSigner
	var thresholds []pages.AccountThreshold
	sigResp, sigErr := h.Gateway.GetAccountSigners(ctx, network, accountID)
	if sigErr == nil && sigResp != nil {
		for _, s := range sigResp.Signers {
			signers = append(signers, pages.AccountSigner{
				Address: gateway.ShortAddress(s.Key),
				Type:    s.Type,
				IsSelf:  s.Key == accountID,
				Weight:  fmt.Sprintf("%d", s.Weight),
			})
		}
		thresholds = []pages.AccountThreshold{
			{Label: "Low", Value: fmt.Sprintf("%d", sigResp.Thresholds.Low), Color: "emerald"},
			{Label: "Medium", Value: fmt.Sprintf("%d", sigResp.Thresholds.Medium), Color: "emerald"},
			{Label: "High", Value: fmt.Sprintf("%d", sigResp.Thresholds.High), Color: "emerald"},
		}
	}

	// Count trustlines as non-native balances.
	trustlineCount := 0
	for _, b := range balances {
		if b.Type != "Native" {
			trustlineCount++
		}
	}

	data := pages.AccountData{
		Address:        accountID,
		ShortAddress:   gateway.ShortAddress(accountID),
		TotalValue:     "—",
		XLMBalance:     acct.Balance + " XLM",
		Trustlines:     fmt.Sprintf("%d", trustlineCount),
		ActiveOffers:   "0",
		Subentries:     fmt.Sprintf("%d", acct.NumSubentries),
		SequenceNumber: acct.SequenceNumber,
		IsFunded:       true,
		SignerCount:    fmt.Sprintf("%d", len(signers)),
		Balances:       balances,
		Activities:     activities,
		Signers:        signers,
		Thresholds:     thresholds,
	}
	if strings.HasPrefix(accountID, "C") {
		if walletInfo, err := h.Gateway.GetSmartWalletInfo(ctx, network, accountID); err == nil && walletInfo != nil && walletInfo.IsSmartWallet {
			data.IsSmartWallet = true
		}
	}

	// Prefer created_at from gateway; fall back to updated_at.
	createdAtStr := acct.CreatedAt
	if createdAtStr == "" {
		createdAtStr = acct.UpdatedAt
	}
	if createdAtStr != "" {
		if t, err := time.Parse(time.RFC3339, createdAtStr); err == nil {
			data.CreatedAt = t.Format("Jan 2, 2006")
		}
	}

	return data, nil
}

func (h *Handlers) SmartAccountDashboard(w http.ResponseWriter, r *http.Request) {
	data, _ := h.smartAccountDataForRequest(r)
	pages.SmartAccountV2(data).Render(r.Context(), w)
}

func (h *Handlers) SmartAccountDashboardV2(w http.ResponseWriter, r *http.Request) {
	data, network := h.smartAccountDataForRequest(r)
	if err := pagesv2.SmartAccount(data, network).Render(r.Context(), w); err != nil {
		h.Logger.Error("render v2 smart account", "contract", r.PathValue("id"), "error", err)
	}
}

func (h *Handlers) smartAccountDataForRequest(r *http.Request) (pages.SmartAccountData, string) {
	id := r.PathValue("id")
	network := networkFromRequest(r)

	var data pages.SmartAccountData
	if h.useLiveData(r) {
		if live, err := h.buildSmartAccountData(r, network, id); err == nil {
			data = live
		} else {
			h.Logger.Warn("live smart account data failed, falling back to mock", "contract", id, "error", err)
		}
	}
	if data.ContractID == "" {
		data = pages.SmartAccountData{
			Name:                 "Treasury Multisig",
			ContractID:           "CDLZ...Q8M4",
			TotalBalance:         "$87,204",
			BalanceCents:         ".51",
			BadgeLabel:           "Smart Wallet",
			ClassificationSource: "Mock",
			ApprovalSummary:      "2 of 3 signers required for protected actions",
			PartialData:          true,
			ActiveWindows:        []string{"09:00-12:00 UTC"},
			CommonFunctions: []pages.SmartFunctionSummary{
				{Name: "execute", Count: "14"},
				{Name: "approve", Count: "8"},
			},
			Signers: []pages.SmartSigner{
				{Name: "Owner Key", Role: "Admin", RoleColor: "amber", Address: "GBXC...4K71", KeyType: "Ed25519", Weight: "10", IconSVG: "key", IconBg: "bg-amber-50 dark:bg-amber-950/30 text-amber-600 dark:text-amber-400 ring-1 ring-amber-100 dark:ring-amber-800"},
				{Name: "Operations Signer", Role: "Signer", RoleColor: "blue", Address: "GDEF...9R23", KeyType: "Ed25519", Weight: "10", IconSVG: "user", IconBg: "bg-blue-50 dark:bg-blue-950/30 text-blue-600 dark:text-blue-400 ring-1 ring-blue-100 dark:ring-blue-800"},
				{Name: "Recovery Signer", Role: "Recovery", RoleColor: "emerald", Address: "GHIJ...2M56", KeyType: "Ed25519", Weight: "10", IconSVG: "recovery", IconBg: "bg-emerald-50 dark:bg-emerald-950/30 text-emerald-600 dark:text-emerald-400 ring-1 ring-emerald-100 dark:ring-emerald-800"},
			},
			LowThreshold:   "10",
			MedThreshold:   "20",
			HighThreshold:  "20",
			MasterWeight:   "10",
			RequiredWeight: "20",
			TotalWeight:    "30",
			MinSigners:     "2 of 3",
			Health:         pages.ContractHealth{RentStatus: "Healthy", TTLRemaining: "~58 days", WASMHash: "0xa4f2...c8b1", OZVersion: "v0.5.0", Deployed: "Oct 2, 2026"},
		}
		if len(data.SecurityLog) == 0 {
			data.SecurityLog = []pages.SecurityEvent{{Action: "Wallet detected", Detail: "Using fallback mock data", Time: "—", Status: "Info", StatusColor: "blue"}}
		}
	}
	return data, network
}

func (h *Handlers) buildSmartAccountData(r *http.Request, network, contractID string) (pages.SmartAccountData, error) {
	ctx := r.Context()
	detail, err := h.Gateway.GetSmartWalletDetail(ctx, network, contractID)
	if err != nil {
		return h.buildSmartAccountDataLegacy(r, network, contractID)
	}
	if detail == nil {
		return pages.SmartAccountData{}, fmt.Errorf("smart wallet detail returned nil for %s", contractID)
	}

	walletLike := detail.IsSmartWallet || detail.Contract.InterfaceType == "smart_wallet" || len(detail.Timeline) > 0 || len(detail.Contract.ObservedFunctions) > 0
	if !walletLike {
		return h.buildSmartAccountDataLegacy(r, network, contractID)
	}

	name := firstNonEmpty(detail.DisplayName, smartWalletDisplayName(detail.Wallet.WalletType, detail.Wallet.Implementation), gateway.ShortAddress(contractID))
	badgeLabel := "Smart Wallet"
	degraded := !detail.IsSmartWallet
	if degraded {
		badgeLabel = "Wallet-like Contract"
	}

	data := pages.SmartAccountData{
		Name:                 name,
		ContractID:           contractID,
		TotalBalance:         "—",
		BalanceCents:         "",
		BadgeLabel:           badgeLabel,
		OverviewTabLabel:     "Overview",
		SecurityTabLabel:     "Security",
		ActivityTabLabel:     "Activity",
		ClassificationSource: titleCase(firstNonEmpty(detail.Wallet.ClassificationSource, "observed")),
		Confidence:           walletConfidenceLabel(detail.Wallet.Confidence),
		Evidence:             detail.Wallet.Evidence,
		ApprovalSummary:      firstNonEmpty(detail.SignerConfig.ApprovalModel.Summary, "Threshold model not decoded from indexed state"),
		Degraded:             degraded,
		PartialData:          detail.Meta.Partial,
		SignerSourceLabel:    smartWalletSignerSourceLabel(true, detail.SignerConfig.Decoded, detail.SignerConfig.Source),
		PolicyStateLabel:     smartWalletPolicyStateLabel(detail.Policies.Decoded, len(detail.Policies.Items)),
		ActivitySourceLabel:  "incoming and outgoing transfers",
		ActiveWindows:        detail.Activity.ActiveWindows,
		LowThreshold:         "—",
		MedThreshold:         "—",
		HighThreshold:        "—",
		MasterWeight:         "—",
		RequiredWeight:       smartWalletOptionalInt(detail.SignerConfig.RequiredWeight),
		TotalWeight:          smartWalletOptionalInt(detail.SignerConfig.TotalWeight),
		MinSigners:           firstNonEmpty(smartWalletMinSignerLabel(&detail.SignerConfig), fmt.Sprintf("%d detected", detail.SignerConfig.SignerCount)),
		Health: pages.ContractHealth{
			RentStatus:     titleCase(firstNonEmpty(detail.Rent.RentStatus, "unknown")),
			TTLRemaining:   smartWalletTTLLabel(detail.Rent.TTLExpiresAt, detail.Rent.TTLLedgers),
			WASMHash:       firstNonEmpty(truncateMiddleLocal(detail.Contract.WASMHash, 18), "—"),
			OZVersion:      firstNonEmpty(walletTypeLabel(firstNonEmpty(detail.Wallet.Implementation, detail.Wallet.WalletType)), titleCase(detail.Contract.InterfaceType), "Smart Wallet"),
			Deployed:       smartWalletDateLabel(detail.Account.CreatedAt),
			Deployer:       firstNonEmpty(gateway.ShortAddress(detail.Contract.Deployer), "—"),
			DeployerFull:   detail.Contract.Deployer,
			DeployerHref:   smartWalletAccountHref(detail.Contract.Deployer),
			StateSize:      formatBytes(detail.Contract.StateSizeBytes),
			StorageEntries: fmt.Sprintf("%d", detail.Contract.StorageEntries),
		},
	}

	if data.Health.StateSize == "0 B" {
		data.Health.StateSize = "—"
	}
	if detail.Contract.StorageEntries == 0 {
		data.Health.StorageEntries = "—"
	}

	for i, signer := range detail.SignerConfig.Signers {
		role := firstNonEmpty(titleCase(signer.Role), "Signer")
		roleColor := "blue"
		if i == 0 || strings.EqualFold(signer.Role, "primary") {
			roleColor = "amber"
		}
		name := firstNonEmpty(signer.Label, role+" Signer", fmt.Sprintf("Signer %d", i+1))
		weight := "—"
		if signer.Weight != nil {
			weight = fmt.Sprintf("%d", *signer.Weight)
		}
		data.Signers = append(data.Signers, pages.SmartSigner{
			Name:        name,
			Role:        role,
			RoleColor:   roleColor,
			Address:     gateway.ShortAddress(signer.ID),
			AddressFull: signer.ID,
			AddressHref: "/account/" + signer.ID,
			KeyType:     strings.ToUpper(firstNonEmpty(signer.KeyType, "unknown")),
			Weight:      weight,
		})
	}

	for _, policy := range detail.Policies.Items {
		p := pages.SmartPolicy{
			Name:        firstNonEmpty(policy.Name, titleCase(strings.ReplaceAll(policy.PolicyType, "_", " "))),
			Description: firstNonEmpty(policy.Description, "Observed wallet policy"),
			IsActive:    !strings.EqualFold(policy.Status, "inactive"),
		}
		for _, c := range policy.Contracts {
			methods := strings.Join(c.Methods, ", ")
			if methods == "" {
				methods = "observed"
			}
			p.Contracts = append(p.Contracts, pages.AllowedContract{
				Initial:   strings.ToUpper(firstRune(firstNonEmpty(c.DisplayName, c.ContractID, "?"))),
				InitialBg: "bg-violet-100 text-violet-700",
				Name:      firstNonEmpty(c.DisplayName, gateway.ShortAddress(c.ContractID)),
				Address:   gateway.ShortAddress(c.ContractID),
				Methods:   methods,
			})
		}
		data.Policies = append(data.Policies, p)
	}

	for _, item := range detail.SessionKeys.Items {
		data.SessionKeys = append(data.SessionKeys, pages.SmartSessionKey{
			Name:       firstNonEmpty(item.Status, "Session key"),
			Key:        truncateMiddleLocal(item.Key, 20),
			Scope:      firstNonEmpty(item.Scope, "—"),
			SpendLimit: firstNonEmpty(item.SpendLimit, "—"),
			Used:       firstNonEmpty(item.UsedAmount, "—"),
			Expires:    firstNonEmpty(smartWalletDateLabel(item.ExpiresAt), "—"),
		})
	}

	for _, fn := range detail.Activity.CommonFunctions {
		data.CommonFunctions = append(data.CommonFunctions, pages.SmartFunctionSummary{
			Name:  fn.Name,
			Count: gateway.FormatNumber(fn.Count),
		})
	}
	for _, p := range detail.Activity.CommonProtocols {
		data.CommonProtocols = append(data.CommonProtocols, pages.SmartProtocolSummary{
			Name:  firstNonEmpty(p.DisplayName, gateway.ShortAddress(p.ContractID)),
			Count: gateway.FormatNumber(p.InteractionCount),
		})
	}

	for _, ev := range detail.Timeline {
		status := "Info"
		statusColor := "blue"
		if ev.Successful {
			status = "Success"
			statusColor = "emerald"
		}
		if !ev.Successful && ev.TxHash != "" {
			status = "Failed"
			statusColor = "red"
		}
		detailText := ev.Description
		if len(ev.Actors) > 0 {
			detailText = firstNonEmpty(detailText, fmt.Sprintf("actor %s", gateway.ShortAddress(ev.Actors[0])))
		}
		if detailText == "" && ev.TxHash != "" {
			detailText = "Transaction observed"
		}
		iconSVG, iconBg := smartWalletEventStyle(firstNonEmpty(ev.Subtype, ev.Type), status)
		data.SecurityLog = append(data.SecurityLog, pages.SecurityEvent{
			Action:      smartWalletActionLabel(firstNonEmpty(ev.Title, ""), ev.Subtype, ev.Type),
			Detail:      detailText,
			Time:        formatContractAge(ev.Timestamp),
			Status:      status,
			StatusColor: statusColor,
			TxHash:      ev.TxHash,
			ShortHash:   gateway.ShortHash(ev.TxHash),
			IconSVG:     iconSVG,
			IconBg:      iconBg,
		})
	}

	if len(data.SecurityLog) == 0 {
		data.SecurityLog = append(data.SecurityLog, pages.SecurityEvent{
			Action:      "Wallet detail available",
			Detail:      firstNonEmpty(data.ApprovalSummary, "Wallet activity has been observed on-chain."),
			Time:        "now",
			Status:      "Info",
			StatusColor: "blue",
		})
	}

	if bal := smartWalletPrimaryBalance(detail); bal != nil {
		data.TotalBalance = firstNonEmpty(bal.Balance, "—")
		if bal.AssetType == "native" || strings.EqualFold(bal.AssetCode, "XLM") || bal.AssetCode == "" {
			data.BalanceCents = "XLM"
		} else {
			data.BalanceCents = strings.ToUpper(bal.AssetCode)
		}
	}

	if acctData, err := h.buildAccountData(r, network, contractID); err == nil {
		if (data.TotalBalance == "—" || data.TotalBalance == "0" || data.TotalBalance == "0.0000000") && acctData.XLMBalance != "" {
			parts := strings.SplitN(acctData.XLMBalance, " ", 2)
			data.TotalBalance = parts[0]
			if len(parts) > 1 {
				data.BalanceCents = parts[1]
			}
		}
		if data.Health.Deployed == "—" {
			data.Health.Deployed = acctData.CreatedAt
		}
	}

	if len(data.Signers) == 0 {
		if walletInfo, err := h.Gateway.GetSmartWalletInfo(ctx, network, contractID); err == nil && walletInfo != nil {
			data.SignerSourceLabel = smartWalletSignerSourceLabel(true, false, "wallet_detection")
			for i, signer := range walletInfo.Signers {
				role := "Signer"
				roleColor := "blue"
				if i == 0 {
					role = "Primary"
					roleColor = "amber"
				}
				data.Signers = append(data.Signers, pages.SmartSigner{
					Name:        fmt.Sprintf("Signer %d", i+1),
					Role:        role,
					RoleColor:   roleColor,
					Address:     gateway.ShortAddress(signer.ID),
					AddressFull: signer.ID,
					AddressHref: "/account/" + signer.ID,
					KeyType:     strings.ToUpper(signer.KeyType),
					Weight:      "—",
				})
			}
			if data.MinSigners == "0 detected" || data.MinSigners == "" {
				data.MinSigners = fmt.Sprintf("%d detected", walletInfo.SignerCount)
			}
		}
	}

	if data.ClassificationSource == "Observed" && detail.IsSmartWallet {
		data.ClassificationSource = "Classified"
	}
	if degraded && data.Confidence == "" {
		data.Confidence = "low confidence"
	}
	if data.PolicyStateLabel == "" {
		data.PolicyStateLabel = smartWalletPolicyStateLabel(detail.Policies.Decoded, len(detail.Policies.Items))
	}
	if len(data.SecurityLog) < 5 {
		h.fillSmartWalletSecurityFallback(ctx, network, contractID, &data)
	}
	if activity, windowLabel, err := h.buildSmartWalletActivityLog(ctx, network, contractID, detail.Account.CreatedAt); err == nil {
		data.ActivityLog = activity
		data.ActivityWindowLabel = windowLabel
	} else {
		h.Logger.Warn("smart wallet transfer activity unavailable", "contract", contractID, "error", err)
	}
	data.OverviewTabLabel = smartWalletTabLabel("Overview", 0)
	data.SecurityTabLabel = smartWalletTabLabel("Security", len(data.SecurityLog))
	data.ActivityTabLabel = smartWalletTabLabel("Activity", len(data.ActivityLog))
	if len(data.Evidence) == 0 && len(detail.Contract.ObservedFunctions) > 0 {
		for _, fn := range detail.Contract.ObservedFunctions {
			data.Evidence = append(data.Evidence, "function:"+fn)
		}
	}

	return data, nil
}

func (h *Handlers) buildSmartAccountDataLegacy(r *http.Request, network, contractID string) (pages.SmartAccountData, error) {
	ctx := r.Context()
	walletInfo, err := h.Gateway.GetSmartWalletInfo(ctx, network, contractID)
	if err != nil {
		return pages.SmartAccountData{}, fmt.Errorf("fetching smart wallet info: %w", err)
	}
	if walletInfo == nil || !walletInfo.IsSmartWallet {
		return pages.SmartAccountData{}, fmt.Errorf("contract %s is not classified as a smart wallet", contractID)
	}

	metadata, _ := h.Gateway.GetContractMetadata(ctx, network, contractID)
	recentCalls, _ := h.Gateway.GetContractRecentCalls(ctx, network, contractID, 8)

	name := smartWalletDisplayName(walletInfo.WalletType, walletInfo.Implementation)
	if metadata != nil && metadata.DisplayName != "" {
		name = metadata.DisplayName
	}

	data := pages.SmartAccountData{
		Name:                 name,
		ContractID:           contractID,
		TotalBalance:         "—",
		BalanceCents:         "",
		BadgeLabel:           "Smart Wallet",
		OverviewTabLabel:     "Overview",
		SecurityTabLabel:     "Security",
		ActivityTabLabel:     "Activity",
		ClassificationSource: "Detected",
		Confidence:           walletConfidenceLabel(walletInfo.Confidence),
		ApprovalSummary:      "Threshold model not decoded from indexed state",
		PartialData:          true,
		SignerSourceLabel:    smartWalletSignerSourceLabel(false, false, "wallet_detection"),
		PolicyStateLabel:     smartWalletPolicyStateLabel(false, 0),
		ActivitySourceLabel:  "incoming and outgoing transfers",
		LowThreshold:         "—",
		MedThreshold:         "—",
		HighThreshold:        "—",
		MasterWeight:         "—",
		RequiredWeight:       "—",
		TotalWeight:          "—",
		MinSigners:           fmt.Sprintf("%d detected", walletInfo.SignerCount),
		Health: pages.ContractHealth{
			RentStatus:     "Observed",
			TTLRemaining:   "—",
			WASMHash:       "—",
			OZVersion:      walletTypeLabel(firstNonEmpty(walletInfo.Implementation, walletInfo.WalletType)),
			Deployed:       "—",
			Deployer:       "—",
			DeployerFull:   "",
			DeployerHref:   "#",
			StateSize:      "—",
			StorageEntries: "—",
		},
		Policies: []pages.SmartPolicy{{
			Name:        "Wallet provenance",
			Description: fmt.Sprintf("Detected as %s with %s confidence", walletTypeLabel(firstNonEmpty(walletInfo.WalletType, "smart wallet")), firstNonEmpty(walletConfidenceLabel(walletInfo.Confidence), "unknown")),
			IsActive:    true,
		}},
	}

	if metadata != nil {
		if metadata.WASMHash != "" {
			data.Health.WASMHash = truncateMiddleLocal(metadata.WASMHash, 18)
		}
		if metadata.CreatedAt != "" {
			data.Health.Deployed = smartWalletDateLabel(metadata.CreatedAt)
		}
	}

	for i, signer := range walletInfo.Signers {
		role := "Signer"
		roleColor := "blue"
		if i == 0 {
			role = "Primary"
			roleColor = "amber"
		}
		data.Signers = append(data.Signers, pages.SmartSigner{
			Name:        "Signer " + fmt.Sprintf("%d", i+1),
			Role:        role,
			RoleColor:   roleColor,
			Address:     gateway.ShortAddress(signer.ID),
			AddressFull: signer.ID,
			AddressHref: smartWalletAccountHref(signer.ID),
			KeyType:     strings.ToUpper(signer.KeyType),
			Weight:      "—",
		})
	}

	for _, call := range recentCalls {
		action := smartWalletFunctionActionLabel(call.FunctionName)
		status := "Success"
		statusColor := "emerald"
		if !call.Successful {
			status = "Failed"
			statusColor = "red"
		}
		iconSVG, iconBg := smartWalletEventStyle(call.FunctionName, status)
		data.SecurityLog = append(data.SecurityLog, pages.SecurityEvent{
			Action:      action,
			Detail:      fmt.Sprintf("caller %s", gateway.ShortAddress(call.SourceAccount)),
			Time:        formatContractAge(call.ClosedAt),
			Status:      status,
			StatusColor: statusColor,
			TxHash:      call.TransactionHash,
			ShortHash:   gateway.ShortHash(call.TransactionHash),
			IconSVG:     iconSVG,
			IconBg:      iconBg,
		})
	}
	if len(data.SecurityLog) == 0 {
		data.SecurityLog = append(data.SecurityLog, pages.SecurityEvent{
			Action:      "Wallet classification available",
			Detail:      fmt.Sprintf("Prism identified this contract as a %s smart wallet.", walletTypeLabel(firstNonEmpty(walletInfo.WalletType, "smart"))),
			Time:        "now",
			Status:      "Info",
			StatusColor: "blue",
		})
	}
	if len(data.SecurityLog) < 5 {
		h.fillSmartWalletSecurityFallback(ctx, network, contractID, &data)
	}
	if activity, windowLabel, err := h.buildSmartWalletActivityLog(ctx, network, contractID, ""); err == nil {
		data.ActivityLog = activity
		data.ActivityWindowLabel = windowLabel
	}
	data.OverviewTabLabel = smartWalletTabLabel("Overview", 0)
	data.SecurityTabLabel = smartWalletTabLabel("Security", len(data.SecurityLog))
	data.ActivityTabLabel = smartWalletTabLabel("Activity", len(data.ActivityLog))

	return data, nil
}

func smartWalletDisplayName(walletType, implementation string) string {
	label := walletTypeLabel(firstNonEmpty(walletType, implementation))
	if label == "Smart Wallet" && implementation != "" {
		label = titleCase(strings.ReplaceAll(implementation, "_", " "))
	}
	return label
}

func smartWalletOptionalInt(v *int) string {
	if v == nil {
		return "—"
	}
	return fmt.Sprintf("%d", *v)
}

func smartWalletAccountHref(accountID string) string {
	if accountID == "" {
		return "#"
	}
	return "/account/" + accountID
}

func smartWalletSignerSourceLabel(fromDetail, decoded bool, source string) string {
	if decoded {
		return "decoded signer set"
	}
	switch source {
	case "wallet_detection":
		if fromDetail {
			return "detected signer hints"
		}
		return "fallback signer hints"
	case "indexed_state":
		return "indexed signer state"
	default:
		if fromDetail {
			return "partial signer data"
		}
		return "signers not decoded"
	}
}

func smartWalletPolicyStateLabel(decoded bool, count int) string {
	switch {
	case decoded && count > 0:
		return "decoded policies"
	case decoded && count == 0:
		return "decoded, none observed"
	case !decoded && count > 0:
		return "heuristic policies"
	default:
		return "no decoded policies"
	}
}

func smartWalletMinSignerLabel(cfg *gateway.SmartWalletDetailSignerConfig) string {
	if cfg == nil {
		return ""
	}
	if cfg.MinSignersEstimate != nil && cfg.SignerCount > 0 {
		return fmt.Sprintf("%d of %d", *cfg.MinSignersEstimate, cfg.SignerCount)
	}
	if cfg.RequiredWeight != nil && cfg.TotalWeight != nil {
		return fmt.Sprintf("%d / %d weight", *cfg.RequiredWeight, *cfg.TotalWeight)
	}
	if cfg.SignerCount > 0 {
		return fmt.Sprintf("%d detected", cfg.SignerCount)
	}
	return ""
}

func smartWalletTTLLabel(expiresAt string, ttlLedgers int64) string {
	if expiresAt != "" {
		return formatContractAge(expiresAt)
	}
	if ttlLedgers > 0 {
		return fmt.Sprintf("%s ledgers", gateway.FormatNumber(ttlLedgers))
	}
	return "—"
}

func (h *Handlers) buildSmartWalletActivityLog(ctx context.Context, network, contractID, createdAt string) ([]pages.SecurityEvent, string, error) {
	startTime, windowLabel := smartWalletActivityWindow(createdAt)
	filtersBase := map[string]string{
		"start_time": startTime,
		"end_time":   time.Now().UTC().Format(time.RFC3339),
		"limit":      "30",
		"order":      "desc",
	}
	fromFilters := map[string]string{}
	toFilters := map[string]string{}
	for k, v := range filtersBase {
		fromFilters[k] = v
		toFilters[k] = v
	}
	fromFilters["from_account"] = contractID
	toFilters["to_account"] = contractID

	outgoing, errOut := h.Gateway.GetTransfersFiltered(ctx, network, fromFilters)
	incoming, errIn := h.Gateway.GetTransfersFiltered(ctx, network, toFilters)
	if errOut != nil && errIn != nil {
		return nil, windowLabel, fmt.Errorf("outgoing=%v incoming=%v", errOut, errIn)
	}

	seen := map[string]bool{}
	var merged []gateway.TransferEvent
	for _, t := range append(outgoing, incoming...) {
		key := t.TransactionHash + ":" + t.FromAccount + ":" + t.ToAccount + ":" + t.Amount + ":" + t.TokenContractID
		if seen[key] {
			continue
		}
		seen[key] = true
		merged = append(merged, t)
	}
	sort.Slice(merged, func(i, j int) bool { return merged[i].Timestamp > merged[j].Timestamp })
	if len(merged) > 20 {
		merged = merged[:20]
	}

	tokenLabels := map[string]smartWalletTokenInfo{}
	var events []pages.SecurityEvent
	for _, t := range merged {
		action := "Transfer involving wallet"
		assetLabel, decimals := h.smartWalletTransferAssetLabel(ctx, network, t, tokenLabels)
		amountLabel := formatTransferAmount(t.Amount, decimals)
		detail := smartWalletTransferDetail(contractID, t, amountLabel, assetLabel)
		status := "Success"
		statusColor := "emerald"
		if !t.TransactionSuccessful {
			status = "Failed"
			statusColor = "red"
		}
		switch {
		case t.FromAccount == contractID && t.ToAccount == contractID:
			action = "Self-transfer"
		case t.FromAccount == contractID:
			action = "Outgoing transfer"
		case t.ToAccount == contractID:
			action = "Incoming transfer"
		}
		iconSVG, iconBg := smartWalletEventStyle(action, status)
		counterpartyLabel := ""
		counterpartyHref := ""
		relatedLabel := ""
		relatedHref := ""
		if t.FromAccount == contractID && t.ToAccount != contractID {
			counterpartyLabel = gateway.ShortAddress(t.ToAccount)
			counterpartyHref = smartWalletActorHref(t.ToAccount)
		} else if t.ToAccount == contractID && t.FromAccount != contractID {
			counterpartyLabel = gateway.ShortAddress(t.FromAccount)
			counterpartyHref = smartWalletActorHref(t.FromAccount)
		}
		if t.TokenContractID != "" {
			relatedLabel = gateway.ShortAddress(t.TokenContractID)
			relatedHref = "/contracts/" + t.TokenContractID
		}
		events = append(events, pages.SecurityEvent{
			Action:            action,
			Detail:            detail,
			Time:              formatContractAge(t.Timestamp),
			Status:            status,
			StatusColor:       statusColor,
			TxHash:            t.TransactionHash,
			ShortHash:         gateway.ShortHash(t.TransactionHash),
			CounterpartyLabel: counterpartyLabel,
			CounterpartyHref:  counterpartyHref,
			RelatedLabel:      relatedLabel,
			RelatedHref:       relatedHref,
			IconSVG:           iconSVG,
			IconBg:            iconBg,
		})
	}
	return events, windowLabel, nil
}

func (h *Handlers) fillSmartWalletSecurityFallback(ctx context.Context, network, contractID string, data *pages.SmartAccountData) {
	if data == nil {
		return
	}
	recentCalls, err := h.Gateway.GetContractRecentCalls(ctx, network, contractID, 10)
	if err != nil {
		return
	}
	seen := map[string]bool{}
	for _, ev := range data.SecurityLog {
		if ev.TxHash != "" {
			seen[ev.TxHash] = true
		}
	}
	for _, call := range recentCalls {
		if call.TransactionHash == "" || seen[call.TransactionHash] {
			continue
		}
		status := "Success"
		statusColor := "emerald"
		if !call.Successful {
			status = "Failed"
			statusColor = "red"
		}
		action := smartWalletFunctionActionLabel(call.FunctionName)
		iconSVG, iconBg := smartWalletEventStyle(call.FunctionName, status)
		data.SecurityLog = append(data.SecurityLog, pages.SecurityEvent{
			Action:      action,
			Detail:      fmt.Sprintf("caller %s", gateway.ShortAddress(call.SourceAccount)),
			Time:        formatContractAge(call.ClosedAt),
			Status:      status,
			StatusColor: statusColor,
			TxHash:      call.TransactionHash,
			ShortHash:   gateway.ShortHash(call.TransactionHash),
			IconSVG:     iconSVG,
			IconBg:      iconBg,
		})
		seen[call.TransactionHash] = true
		if len(data.SecurityLog) >= 12 {
			break
		}
	}
}

type smartWalletTokenInfo struct {
	Label    string
	Decimals int
}

func (h *Handlers) smartWalletTransferAssetLabel(ctx context.Context, network string, t gateway.TransferEvent, cache map[string]smartWalletTokenInfo) (string, int) {
	if strings.EqualFold(t.AssetCode, "XLM") {
		return "XLM", 7
	}
	if t.TokenContractID != "" {
		if info, ok := cache[t.TokenContractID]; ok {
			return info.Label, info.Decimals
		}
		info := smartWalletTokenInfo{Label: gateway.ShortAddress(t.TokenContractID), Decimals: 7}
		if token, err := h.Gateway.GetSilverToken(ctx, network, t.TokenContractID); err == nil && token != nil {
			info.Decimals = token.Decimals
			symbol := firstNonEmpty(token.Symbol, token.Name)
			if strings.EqualFold(symbol, "native") {
				info.Label = "XLM"
			} else if symbol != "" {
				info.Label = symbol
			}
		}
		cache[t.TokenContractID] = info
		return info.Label, info.Decimals
	}
	if t.AssetCode != "" {
		return t.AssetCode, 7
	}
	return strings.ToUpper(firstNonEmpty(t.SourceType, "asset")), 0
}

func smartWalletTransferDetail(walletID string, t gateway.TransferEvent, amountLabel, assetLabel string) string {
	from := gateway.ShortAddress(t.FromAccount)
	to := gateway.ShortAddress(t.ToAccount)
	isSelf := t.FromAccount == walletID && t.ToAccount == walletID
	isOutgoing := t.FromAccount == walletID && !isSelf
	isIncoming := t.ToAccount == walletID && !isSelf
	toTokenContract := t.TokenContractID != "" && t.ToAccount == t.TokenContractID

	switch {
	case isSelf:
		return fmt.Sprintf("Moved %s %s within this wallet", amountLabel, assetLabel)
	case isOutgoing && toTokenContract:
		return fmt.Sprintf("Sent %s %s to token contract %s", amountLabel, assetLabel, gateway.ShortAddress(t.TokenContractID))
	case isOutgoing:
		return fmt.Sprintf("Sent %s %s to %s", amountLabel, assetLabel, to)
	case isIncoming:
		return fmt.Sprintf("Received %s %s from %s", amountLabel, assetLabel, from)
	default:
		return fmt.Sprintf("%s → %s · %s %s", from, to, amountLabel, assetLabel)
	}
}

func smartWalletActorHref(id string) string {
	if id == "" {
		return "#"
	}
	if strings.HasPrefix(id, "C") {
		return "/contracts/" + id
	}
	return "/account/" + id
}

func smartWalletActionLabel(title, subtype, eventType string) string {
	if title != "" {
		return title
	}
	switch strings.ToLower(firstNonEmpty(subtype, eventType)) {
	case "signer_added":
		return "Added signer"
	case "signer_removed":
		return "Removed signer"
	case "wallet_execution":
		return "Executed wallet action"
	case "policy_update":
		return "Updated wallet policy"
	case "multisig_approval", "approval":
		return "Approved wallet action"
	default:
		label := strings.ReplaceAll(firstNonEmpty(subtype, eventType), "_", " ")
		if label == "" {
			return "Observed wallet event"
		}
		return titleCase(label)
	}
}

func smartWalletFunctionActionLabel(fn string) string {
	switch strings.ToLower(fn) {
	case "add_context_rule":
		return "Added policy rule"
	case "remove_context_rule":
		return "Removed policy rule"
	case "execute":
		return "Executed wallet action"
	case "add_signer":
		return "Added signer"
	case "revoke_signer", "remove_signer":
		return "Removed signer"
	case "approve":
		return "Approved wallet action"
	case "":
		return "Observed smart-wallet call"
	default:
		return "Called " + humanize.HumanizeFunctionName(fn) + "()"
	}
}

func smartWalletEventStyle(kind, status string) (string, string) {
	k := strings.ToLower(kind)
	success := strings.EqualFold(status, "success") || strings.EqualFold(status, "info")
	switch {
	case strings.Contains(k, "incoming"):
		return "plus", "bg-emerald-50 dark:bg-emerald-950/30 text-emerald-600 dark:text-emerald-400 ring-1 ring-emerald-100 dark:ring-emerald-800"
	case strings.Contains(k, "outgoing"):
		return "arrow-up-right", "bg-blue-50 dark:bg-blue-950/30 text-blue-600 dark:text-blue-400 ring-1 ring-blue-100 dark:ring-blue-800"
	case strings.Contains(k, "self-transfer"):
		return "swap", "bg-violet-50 dark:bg-violet-950/30 text-violet-600 dark:text-violet-400 ring-1 ring-violet-100 dark:ring-violet-800"
	case strings.Contains(k, "add") || strings.Contains(k, "create"):
		return "plus", "bg-emerald-50 dark:bg-emerald-950/30 text-emerald-600 dark:text-emerald-400 ring-1 ring-emerald-100 dark:ring-emerald-800"
	case strings.Contains(k, "remove") || strings.Contains(k, "revoke"):
		return "x", "bg-amber-50 dark:bg-amber-950/30 text-amber-600 dark:text-amber-400 ring-1 ring-amber-100 dark:ring-amber-800"
	case strings.Contains(k, "execute") || strings.Contains(k, "approval") || strings.Contains(k, "policy"):
		return "key", "bg-violet-50 dark:bg-violet-950/30 text-violet-600 dark:text-violet-400 ring-1 ring-violet-100 dark:ring-violet-800"
	case !success:
		return "x", "bg-red-50 dark:bg-red-950/30 text-red-600 dark:text-red-400 ring-1 ring-red-100 dark:ring-red-800"
	default:
		return "check", "bg-surface-subtle text-text-body ring-1 ring-border-subtle"
	}
}

func smartWalletTabLabel(base string, count int) string {
	if count <= 0 || base == "Overview" {
		return base
	}
	return fmt.Sprintf("%s (%d)", base, count)
}

func smartWalletActivityWindow(createdAt string) (startTime string, label string) {
	fallbackStart := time.Now().UTC().Add(-14 * 24 * time.Hour)
	start := fallbackStart
	label = "last 14d"
	for _, layout := range []string{time.RFC3339, "2006-01-02 15:04:05", "2006-01-02"} {
		if createdAt == "" {
			break
		}
		if t, err := time.Parse(layout, createdAt); err == nil {
			start = t.Add(-1 * time.Hour).UTC()
			label = "since wallet creation"
			break
		}
	}
	return start.Format(time.RFC3339), label
}

func compactAmount(v string) string {
	if v == "" {
		return "—"
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return strings.TrimSuffix(strings.TrimSuffix(v, ".0"), ".0000000")
	}
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return v
	}
	if math.Abs(f-math.Round(f)) < 0.0000001 {
		return gateway.FormatNumber(int64(math.Round(f)))
	}
	if math.Abs(f) >= 1 {
		return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.2f", f), "0"), ".")
	}
	return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.6f", f), "0"), ".")
}

func formatTransferAmount(raw string, decimals int) string {
	if raw == "" {
		return "—"
	}
	if decimals <= 0 {
		return compactAmount(raw)
	}
	f, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return compactAmount(raw)
	}
	scaled := f / math.Pow10(decimals)
	return compactAmount(fmt.Sprintf("%.12f", scaled))
}

func smartWalletDateLabel(raw string) string {
	if raw == "" {
		return "—"
	}
	for _, layout := range []string{time.RFC3339, "2006-01-02 15:04:05", "2006-01-02"} {
		if t, err := time.Parse(layout, raw); err == nil {
			return t.Format("Jan 2, 2006")
		}
	}
	return raw
}

func smartWalletPrimaryBalance(detail *gateway.SmartWalletDetail) *gateway.SmartWalletDetailBalance {
	if detail == nil || len(detail.Account.Balances) == 0 {
		return nil
	}
	for i := range detail.Account.Balances {
		bal := &detail.Account.Balances[i]
		if bal.AssetType == "native" || strings.EqualFold(bal.AssetCode, "XLM") {
			return bal
		}
	}
	return &detail.Account.Balances[0]
}

func firstRune(s string) string {
	if s == "" {
		return "?"
	}
	return string([]rune(s)[0])
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func truncateMiddleLocal(s string, max int) string {
	if len(s) <= max || max <= 3 {
		return s
	}
	prefix := (max - 3) / 2
	suffix := max - 3 - prefix
	return s[:prefix] + "..." + s[len(s)-suffix:]
}
