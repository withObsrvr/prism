package gateway

import "testing"

// allOperationTypes is the complete Stellar XDR OperationType enum, indexed by
// its discriminant. Keeping the list here rather than deriving it from
// operationFamilySources is the point: if a protocol upgrade adds a type, this
// slice is what has to be updated, and the test below fails until the new type
// is given a family.
var allOperationTypes = []string{
	0:  "create_account",
	1:  "payment",
	2:  "path_payment_strict_receive",
	3:  "manage_sell_offer",
	4:  "create_passive_sell_offer",
	5:  "set_options",
	6:  "change_trust",
	7:  "allow_trust",
	8:  "account_merge",
	9:  "inflation",
	10: "manage_data",
	11: "bump_sequence",
	12: "manage_buy_offer",
	13: "path_payment_strict_send",
	14: "create_claimable_balance",
	15: "claim_claimable_balance",
	16: "begin_sponsoring_future_reserves",
	17: "end_sponsoring_future_reserves",
	18: "revoke_sponsorship",
	19: "clawback",
	20: "clawback_claimable_balance",
	21: "set_trust_line_flags",
	22: "liquidity_pool_deposit",
	23: "liquidity_pool_withdraw",
	24: "invoke_host_function",
	25: "extend_footprint_ttl",
	26: "restore_footprint",
}

// Every operation the protocol can produce must land in a real family. A type
// falling through to "other" means the bar paints it neutral, which claims
// "unclassified" about an operation we do in fact understand.
func TestOperationFamilyCoversAllTypes(t *testing.T) {
	want := map[string]string{
		"create_account":                   OpFamilyTransfer,
		"payment":                          OpFamilyTransfer,
		"path_payment_strict_receive":      OpFamilyTransfer,
		"path_payment_strict_send":         OpFamilyTransfer,
		"inflation":                        OpFamilyTransfer,
		"create_claimable_balance":         OpFamilyTransfer,
		"claim_claimable_balance":          OpFamilyTransfer,
		"manage_sell_offer":                OpFamilyMarket,
		"create_passive_sell_offer":        OpFamilyMarket,
		"manage_buy_offer":                 OpFamilyMarket,
		"liquidity_pool_deposit":           OpFamilyMarket,
		"liquidity_pool_withdraw":          OpFamilyMarket,
		"set_options":                      OpFamilyControls,
		"change_trust":                     OpFamilyControls,
		"allow_trust":                      OpFamilyControls,
		"manage_data":                      OpFamilyControls,
		"bump_sequence":                    OpFamilyControls,
		"begin_sponsoring_future_reserves": OpFamilyControls,
		"end_sponsoring_future_reserves":   OpFamilyControls,
		"set_trust_line_flags":             OpFamilyControls,
		"account_merge":                    OpFamilyRevoke,
		"revoke_sponsorship":               OpFamilyRevoke,
		"clawback":                         OpFamilyRevoke,
		"clawback_claimable_balance":       OpFamilyRevoke,
		"invoke_host_function":             OpFamilyContract,
		"extend_footprint_ttl":             OpFamilyContract,
		"restore_footprint":                OpFamilyContract,
	}

	if len(allOperationTypes) != len(want) {
		t.Fatalf("enum has %d types but %d are classified", len(allOperationTypes), len(want))
	}
	for discriminant, typeName := range allOperationTypes {
		got := OperationFamily(typeName)
		if got == OpFamilyOther {
			t.Errorf("op %d (%s) fell through to %q", discriminant, typeName, OpFamilyOther)
			continue
		}
		if got != want[typeName] {
			t.Errorf("op %d (%s) = %q, want %q", discriminant, typeName, got, want[typeName])
		}
	}
}

// The display names FormatOperationType/OperationDisplayName emit have to
// resolve too — the mock path and any pre-formatted gateway response classify
// from those rather than from raw protocol names.
func TestOperationFamilyResolvesDisplayNames(t *testing.T) {
	for _, typeName := range allOperationTypes {
		display := OperationDisplayName(typeName)
		if display == typeName {
			continue // no display mapping, nothing extra to check
		}
		raw := OperationFamily(typeName)
		if got := OperationFamily(display); got != raw {
			t.Errorf("display name %q = %q but raw %q = %q", display, got, typeName, raw)
		}
	}
}

func TestOperationFamilySpellings(t *testing.T) {
	tests := []struct {
		name     string
		typeName string
		want     string
	}{
		{"raw", "invoke_host_function", OpFamilyContract},
		{"api-prefixed", "OperationTypeInvokeHostFunction", OpFamilyContract},
		{"display", "Contract Call", OpFamilyContract},
		{"mock display", "Invoke Contract", OpFamilyContract},
		{"api-prefixed merge", "OperationTypeAccountMerge", OpFamilyRevoke},
		{"api-prefixed trust flags", "OperationTypeSetTrustLineFlags", OpFamilyControls},
		{"aggregate offer label", "Manage Offer", OpFamilyMarket},
		{"unknown", "some_future_operation", OpFamilyOther},
		{"empty", "", OpFamilyOther},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := OperationFamily(tt.typeName); got != tt.want {
				t.Errorf("OperationFamily(%q) = %q, want %q", tt.typeName, got, tt.want)
			}
		})
	}
}

// These two normalize to near-identical keys but belong to opposite families —
// one is a holder claiming funds, the other an issuer seizing them.
func TestOperationFamilyClaimableBalanceKeysDistinct(t *testing.T) {
	if normalizeOpKey("claim_claimable_balance") == normalizeOpKey("clawback_claimable_balance") {
		t.Fatal("claimable-balance keys collided")
	}
	if got := OperationFamily("claim_claimable_balance"); got != OpFamilyTransfer {
		t.Errorf("claim_claimable_balance = %q, want %q", got, OpFamilyTransfer)
	}
	if got := OperationFamily("clawback_claimable_balance"); got != OpFamilyRevoke {
		t.Errorf("clawback_claimable_balance = %q, want %q", got, OpFamilyRevoke)
	}
}

func TestNormalizeOpKeySpellingsAgree(t *testing.T) {
	want := normalizeOpKey("invoke_host_function")
	for _, spelling := range []string{"InvokeHostFunction", "OperationTypeInvokeHostFunction", "invoke host function"} {
		if got := normalizeOpKey(spelling); got != want {
			t.Errorf("normalizeOpKey(%q) = %q, want %q", spelling, got, want)
		}
	}
}
