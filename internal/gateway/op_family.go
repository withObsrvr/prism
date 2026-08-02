package gateway

import "strings"

// Operation family slugs.
//
// Stellar defines 27 operation types (XDR OperationType 0-26). A categorical
// palette stops being readable well before that, so the UI colours the family
// an operation belongs to rather than the operation itself. Each family answers
// one question a reader actually asks of a ledger.
//
// There are six chromatic families plus a neutral, matching the brand's
// six-colour system. Six is not arbitrary: at seven, the blue-violet region had
// three families inside a 60-degree arc and two of them measured OKLab dE 0.049
// apart — indistinguishable. Six holds every unrelated pair above 0.097.
// Resolution comes from the three-step shade ramp instead, giving 18 slots.
//
// The families are deliberately uneven in size. `controls` is broad because
// "an account changed its own settings" is one fact to a reader whether it
// arrived via set_options or change_trust. `agent` carries no operation types
// at all: x402 machine payments are a semantic classification layered over
// contract calls, not a protocol operation.
const (
	OpFamilyContract = "contract" // Soroban: invocation, TTL, and restore
	OpFamilyTransfer = "transfer" // value moved between accounts
	OpFamilyMarket   = "market"   // offers and liquidity pools
	OpFamilyControls = "controls" // account and trustline configuration
	OpFamilyAgent    = "agent"    // x402 / machine-to-machine payments
	OpFamilyRevoke   = "revoke"   // authority removing or reclaiming something
	OpFamilyOther    = "other"    // unrecognised — genuinely unclassified
)

// operationFamilySources maps every operation type to its family, and lists
// every spelling of a name we expect to receive. The gateway returns raw
// protocol names ("invoke_host_function") on some endpoints and API-prefixed
// names ("OperationTypeInvokeHostFunction") on others, while already-formatted
// and mock data arrive as display names ("Contract Call"). normalizeOpKey
// collapses all three onto one key, so an operation only needs listing once per
// spelling that genuinely differs.
//
// The XDR discriminant is noted against each type. TestOperationFamilyCoversAllTypes
// asserts every one of the 27 is present, so a protocol addition fails a test
// rather than silently rendering as "unclassified".
var operationFamilySources = map[string][]string{
	OpFamilyContract: {
		"invoke_host_function", "Contract Call", "Invoke Contract", "invoke", // 24
		// Footprint operations share the contract hue rather than taking one of
		// their own: all three are Soroban, and the within-family shade ramp
		// still separates them. See the palette note in v2-unified.css.
		"extend_footprint_ttl", "Extend TTL", // 25
		"restore_footprint", "Restore Footprint", // 26
	},
	OpFamilyTransfer: {
		"create_account", "Create Account", // 0
		"payment", "Payment", // 1
		"path_payment_strict_receive", "Path Payment", // 2
		"path_payment_strict_send", // 13
		"inflation", "Inflation", // 9 — disabled since protocol 12, still in history
		"create_claimable_balance", "Create Claimable Balance", // 14
		"claim_claimable_balance", "Claim Balance", // 15
	},
	OpFamilyMarket: {
		"manage_sell_offer", "Sell Offer", // 3
		"create_passive_sell_offer", "Passive Offer", // 4
		"manage_buy_offer", "Buy Offer", // 12
		"Manage Offer",
		"liquidity_pool_deposit", "Pool Deposit", // 22
		"liquidity_pool_withdraw", "Pool Withdraw", // 23
	},
	OpFamilyControls: {
		"set_options", "Set Options", // 5
		"change_trust", "Change Trust", // 6
		"allow_trust", "Allow Trust", // 7
		"manage_data", "Manage Data", // 10
		"bump_sequence", "Bump Sequence", // 11
		"begin_sponsoring_future_reserves", "Begin Sponsorship", // 16
		"end_sponsoring_future_reserves", "End Sponsorship", // 17
		"set_trust_line_flags", "Set Trust Flags", // 21
	},
	// Reclamation and teardown. These read differently from the `controls`
	// operations that set them up: change_trust is a holder opting in, whereas
	// clawback is an issuer taking assets back without the holder's consent.
	OpFamilyRevoke: {
		"account_merge", "Account Merge", // 8 — the account ceases to exist
		"revoke_sponsorship", "Revoke Sponsorship", // 18
		"clawback", "Clawback", // 19
		"clawback_claimable_balance", "Clawback Balance", // 20
	},
}

var operationFamilies = buildOperationFamilies(operationFamilySources)

func buildOperationFamilies(src map[string][]string) map[string]string {
	out := make(map[string]string)
	for family, names := range src {
		for _, name := range names {
			out[normalizeOpKey(name)] = family
		}
	}
	return out
}

// OperationFamily returns the semantic family slug for an operation type name.
// Unrecognised types fall to OpFamilyOther, which the UI paints neutral — that
// neutral means "we could not classify this", not "we ran out of colours".
func OperationFamily(typeName string) string {
	if family, ok := operationFamilies[normalizeOpKey(typeName)]; ok {
		return family
	}
	return OpFamilyOther
}

// normalizeOpKey reduces an operation name to lowercase alphanumerics so the
// snake_case, CamelCase, and display-name spellings of one operation collapse
// onto the same lookup key.
func normalizeOpKey(v string) string {
	v = strings.TrimPrefix(v, "OperationType")
	var b strings.Builder
	b.Grow(len(v))
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
