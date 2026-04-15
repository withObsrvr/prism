package gateway

import (
	"fmt"
	"strings"
	"time"
)

// FormatNumber formats an integer with thousand separators: 1284392 → "1,284,392"
func FormatNumber(n int64) string {
	if n < 0 {
		return "-" + FormatNumber(-n)
	}
	s := fmt.Sprintf("%d", n)
	if len(s) <= 3 {
		return s
	}
	var b strings.Builder
	rem := len(s) % 3
	if rem > 0 {
		b.WriteString(s[:rem])
	}
	for i := rem; i < len(s); i += 3 {
		if b.Len() > 0 {
			b.WriteByte(',')
		}
		b.WriteString(s[i : i+3])
	}
	return b.String()
}

// FormatAbbrev formats a number with abbreviation: 1284392 → "1.28M"
func FormatAbbrev(n int64) string {
	switch {
	case n >= 1_000_000_000:
		return fmt.Sprintf("%.1fB", float64(n)/1_000_000_000)
	case n >= 1_000_000:
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
	case n >= 1_000:
		return fmt.Sprintf("%.1fK", float64(n)/1_000)
	default:
		return fmt.Sprintf("%d", n)
	}
}

// FormatAge returns a human-readable age string: "3.8s ago", "2 min ago", etc.
func FormatAge(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%.1fs ago", d.Seconds())
	case d < time.Hour:
		return fmt.Sprintf("%d min ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
}

// ShortHash truncates a hex hash: "8f2a7c1b3c4d5e6f" → "8f2a7c...5e6f"
func ShortHash(h string) string {
	if len(h) <= 12 {
		return h
	}
	return h[:6] + "..." + h[len(h)-4:]
}

// ShortAddress truncates a Stellar address: "GABC7DEF..." → "GABC...MNOP"
func ShortAddress(a string) string {
	if len(a) <= 12 {
		return a
	}
	return a[:4] + "..." + a[len(a)-4:]
}

// FormatStroops formats stroops as a comma-separated number string.
func FormatStroops(s int64) string {
	return FormatNumber(s)
}

// FormatXLM converts stroops to XLM with 7 decimal places.
func FormatXLM(stroops int64) string {
	xlm := float64(stroops) / 10_000_000
	return fmt.Sprintf("%.7f", xlm)
}

// FormatCloseTime formats a close time in seconds.
func FormatCloseTime(seconds float64) string {
	return fmt.Sprintf("%.1fs", seconds)
}

// OperationDisplayName returns a human-readable name for a Stellar operation type.
func OperationDisplayName(typeName string) string {
	if display, ok := operationDisplayNames[typeName]; ok {
		return display
	}
	// Fallback: replace underscores with spaces, capitalize first letter.
	s := strings.ReplaceAll(typeName, "_", " ")
	if len(s) > 0 {
		return strings.ToUpper(s[:1]) + s[1:]
	}
	return s
}

var operationDisplayNames = map[string]string{
	"create_account":                    "Create Account",
	"payment":                           "Payment",
	"path_payment_strict_receive":       "Path Payment",
	"path_payment_strict_send":          "Path Payment",
	"manage_sell_offer":                 "Sell Offer",
	"manage_buy_offer":                  "Buy Offer",
	"create_passive_sell_offer":         "Passive Offer",
	"set_options":                       "Set Options",
	"change_trust":                      "Change Trust",
	"allow_trust":                       "Allow Trust",
	"account_merge":                     "Account Merge",
	"manage_data":                       "Manage Data",
	"bump_sequence":                     "Bump Sequence",
	"create_claimable_balance":          "Create Claimable Balance",
	"claim_claimable_balance":           "Claim Balance",
	"begin_sponsoring_future_reserves":  "Begin Sponsorship",
	"end_sponsoring_future_reserves":    "End Sponsorship",
	"revoke_sponsorship":                "Revoke Sponsorship",
	"clawback":                          "Clawback",
	"clawback_claimable_balance":        "Clawback Balance",
	"set_trust_line_flags":              "Set Trust Flags",
	"liquidity_pool_deposit":            "Pool Deposit",
	"liquidity_pool_withdraw":           "Pool Withdraw",
	"invoke_host_function":              "Contract Call",
	"extend_footprint_ttl":              "Extend TTL",
	"restore_footprint":                 "Restore Footprint",
	// Handle OperationType-prefixed names from the API too.
	"OperationTypeCreateAccount":        "Create Account",
	"OperationTypePayment":              "Payment",
	"OperationTypeSetOptions":           "Set Options",
	"OperationTypeChangeTrust":          "Change Trust",
	"OperationTypeAccountMerge":         "Account Merge",
	"OperationTypeManageData":           "Manage Data",
	"OperationTypeManageSellOffer":      "Sell Offer",
	"OperationTypeManageBuyOffer":       "Buy Offer",
	"OperationTypeSetTrustLineFlags":    "Set Trust Flags",
	"OperationTypeInvokeHostFunction":   "Contract Call",
}

// FormatDecimalAmount formats a decimal amount string with thousand separators
// and trims unnecessary trailing zeros.
// "10000.5000000" → "10,000.5", "100.00" → "100"
func FormatDecimalAmount(amount string) string {
	if amount == "" {
		return ""
	}
	neg := false
	s := amount
	if strings.HasPrefix(s, "-") {
		neg = true
		s = s[1:]
	}

	parts := strings.SplitN(s, ".", 2)
	intPart := addCommas(parts[0])

	result := intPart
	if len(parts) == 2 {
		dec := strings.TrimRight(parts[1], "0")
		if dec != "" {
			result += "." + dec
		}
	}
	if neg {
		return "-" + result
	}
	return result
}

func addCommas(s string) string {
	if len(s) <= 3 {
		return s
	}
	var b strings.Builder
	rem := len(s) % 3
	if rem > 0 {
		b.WriteString(s[:rem])
	}
	for i := rem; i < len(s); i += 3 {
		if b.Len() > 0 {
			b.WriteByte(',')
		}
		b.WriteString(s[i : i+3])
	}
	return b.String()
}

// FormatAmountWithAsset formats an amount with its asset symbol.
func FormatAmountWithAsset(amount, assetCode string) string {
	if assetCode == "" {
		assetCode = "XLM"
	}
	return FormatDecimalAmount(amount) + " " + assetCode
}

// FormatFeeXLM formats a fee in stroops as a human-readable XLM string.
func FormatFeeXLM(stroops int64) string {
	xlm := float64(stroops) / 10_000_000
	if xlm < 0.001 {
		return fmt.Sprintf("%.7f XLM", xlm)
	}
	if xlm < 1 {
		return fmt.Sprintf("%.4f XLM", xlm)
	}
	return fmt.Sprintf("%.2f XLM", xlm)
}

// EffectDisplayName returns a human-readable name for an effect type.
func EffectDisplayName(effectType string) string {
	if display, ok := effectDisplayNames[effectType]; ok {
		return display
	}
	s := strings.ReplaceAll(effectType, "_", " ")
	if len(s) > 0 {
		return strings.ToUpper(s[:1]) + s[1:]
	}
	return s
}

var effectDisplayNames = map[string]string{
	// Account (0-6)
	"account_created":              "Account Created",
	"account_removed":              "Account Removed",
	"account_credited":             "Credited",
	"account_debited":              "Debited",
	"account_thresholds_updated":   "Thresholds Updated",
	"account_home_domain_updated":  "Home Domain Updated",
	"account_flags_updated":        "Flags Updated",
	// Signer (10-12)
	"signer_created":               "Signer Created",
	"signer_removed":               "Signer Removed",
	"signer_updated":               "Signer Updated",
	// Trustline (20-26)
	"trustline_created":            "Trustline Created",
	"trustline_removed":            "Trustline Removed",
	"trustline_updated":            "Trustline Updated",
	"trustline_flags_updated":      "Trust Flags Updated",
	// DEX (30-33)
	"offer_created":                "Offer Created",
	"offer_removed":                "Offer Removed",
	"offer_updated":                "Offer Updated",
	"trade":                        "Trade",
	// Data (40-43)
	"data_created":                 "Data Created",
	"data_removed":                 "Data Removed",
	"data_updated":                 "Data Updated",
	"sequence_bumped":              "Sequence Bumped",
	// Claimable Balance (50-52, 80)
	"claimable_balance_created":           "Balance Created",
	"claimable_balance_claimant_created":  "Claimant Created",
	"claimable_balance_claimed":           "Balance Claimed",
	"claimable_balance_clawed_back":       "Balance Clawed Back",
	// Sponsorship (60-74)
	"account_sponsorship_created":           "Sponsorship Created",
	"account_sponsorship_updated":           "Sponsorship Updated",
	"account_sponsorship_removed":           "Sponsorship Removed",
	"trustline_sponsorship_created":         "Trust Sponsorship Created",
	"trustline_sponsorship_updated":         "Trust Sponsorship Updated",
	"trustline_sponsorship_removed":         "Trust Sponsorship Removed",
	"data_sponsorship_created":              "Data Sponsorship Created",
	"data_sponsorship_updated":              "Data Sponsorship Updated",
	"data_sponsorship_removed":              "Data Sponsorship Removed",
	"claimable_balance_sponsorship_created": "Balance Sponsorship Created",
	"claimable_balance_sponsorship_updated": "Balance Sponsorship Updated",
	"claimable_balance_sponsorship_removed": "Balance Sponsorship Removed",
	"signer_sponsorship_created":            "Signer Sponsorship Created",
	"signer_sponsorship_updated":            "Signer Sponsorship Updated",
	"signer_sponsorship_removed":            "Signer Sponsorship Removed",
	// Liquidity Pool (90-94)
	"liquidity_pool_deposited": "Pool Deposit",
	"liquidity_pool_withdrew":  "Pool Withdraw",
	"liquidity_pool_trade":     "Pool Trade",
	"liquidity_pool_created":   "Pool Created",
	"liquidity_pool_removed":   "Pool Removed",
	// Soroban (96-99)
	"contract_credited":       "Contract Credited",
	"contract_debited":        "Contract Debited",
	"extend_footprint_ttl":    "Extend TTL",
	"restore_footprint":       "Restore Footprint",
}

// EffectTypeColor returns a badge color for an effect type.
func EffectTypeColor(effectType string) string {
	switch effectType {
	// Created
	case "account_created", "signer_created", "trustline_created",
		"offer_created", "data_created", "claimable_balance_created",
		"claimable_balance_claimant_created",
		"liquidity_pool_created":
		return "emerald"
	// Credited / received
	case "account_credited", "contract_credited",
		"claimable_balance_claimed", "liquidity_pool_deposited":
		return "cyan"
	// Debited / sent
	case "account_debited", "contract_debited",
		"liquidity_pool_withdrew":
		return "amber"
	// Removed / clawback
	case "account_removed", "signer_removed", "trustline_removed",
		"offer_removed", "data_removed",
		"claimable_balance_clawed_back",
		"liquidity_pool_removed":
		return "red"
	// Trade
	case "trade", "liquidity_pool_trade":
		return "violet"
	// Sponsorship
	case "account_sponsorship_created", "account_sponsorship_updated", "account_sponsorship_removed",
		"trustline_sponsorship_created", "trustline_sponsorship_updated", "trustline_sponsorship_removed",
		"data_sponsorship_created", "data_sponsorship_updated", "data_sponsorship_removed",
		"claimable_balance_sponsorship_created", "claimable_balance_sponsorship_updated", "claimable_balance_sponsorship_removed",
		"signer_sponsorship_created", "signer_sponsorship_updated", "signer_sponsorship_removed":
		return "blue"
	// Soroban
	case "extend_footprint_ttl", "restore_footprint":
		return "violet"
	default:
		return "gray"
	}
}

// FormatStroopsToXLM converts a stroops amount string (from the API) to XLM.
// "100000000000" → "10,000"
func FormatStroopsToXLM(stroops string) string {
	return FormatTokenAmount(stroops, 7)
}

// FormatTokenAmount converts a raw integer amount string to a human-readable decimal string,
// scaled by `decimals`. Use 7 for native XLM and most Stellar SACs; tokens may use other scales.
// "500000" with decimals=7 → "0.05"
// "100000000000" with decimals=7 → "10,000"
// Falls back to the raw string if parsing fails.
func FormatTokenAmount(raw string, decimals int) string {
	if raw == "" {
		return ""
	}
	if decimals <= 0 {
		// No scaling — just add commas.
		return FormatDecimalAmount(raw)
	}

	neg := false
	s := raw
	if strings.HasPrefix(s, "-") {
		neg = true
		s = s[1:]
	}

	// Pad with leading zeros so we always have at least decimals+1 digits.
	if len(s) <= decimals {
		s = strings.Repeat("0", decimals-len(s)+1) + s
	}
	cut := len(s) - decimals
	intPart := s[:cut]
	fracPart := strings.TrimRight(s[cut:], "0")

	intFormatted := addCommas(intPart)
	result := intFormatted
	if fracPart != "" {
		result += "." + fracPart
	}
	if neg {
		return "-" + result
	}
	return result
}
