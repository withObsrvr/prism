package handlers

import (
	"testing"

	"github.com/withObsrvr/prism/internal/gateway"
)

func TestBuildOpBreakdownOrdersFamiliesByTotal(t *testing.T) {
	got := buildOpBreakdown(map[string]int{
		"payment":              30,
		"invoke_host_function": 37,
		"change_trust":         9,
		"bump_sequence":        9,
	}, 85)

	// Families by combined count: contract 37, transfer 30, controls 18. Inside
	// controls the two 9s tie, so they break on type name.
	want := []string{"Contract Call", "Payment", "Bump Sequence", "Change Trust"}
	if len(got) != len(want) {
		t.Fatalf("got %d segments, want %d", len(got), len(want))
	}
	for i, name := range want {
		if got[i].Name != name {
			t.Errorf("segment %d = %q, want %q", i, got[i].Name, name)
		}
	}
}

// A family's shades must sit next to each other, or the ramp reads as unrelated
// colours rather than one family split by count.
func TestBuildOpBreakdownKeepsFamiliesContiguous(t *testing.T) {
	got := buildOpBreakdown(map[string]int{
		"invoke_host_function":        37,
		"payment":                     30,
		"path_payment_strict_receive": 15,
		"manage_sell_offer":           11,
		"create_account":              9,
		"manage_data":                 6,
	}, 108)

	// transfer totals 54, so it leads despite contract holding the single
	// largest type; its three shades must then run consecutively.
	wantFamilies := []string{"transfer", "transfer", "transfer", "contract", "market", "controls"}
	wantTiers := []int{1, 2, 3, 1, 1, 1}
	for i := range wantFamilies {
		if got[i].Family != wantFamilies[i] || got[i].Tier != wantTiers[i] {
			t.Errorf("segment %d (%s) = family %q tier %d, want family %q tier %d",
				i, got[i].Name, got[i].Family, got[i].Tier, wantFamilies[i], wantTiers[i])
		}
	}

	seen := make(map[string]bool)
	prev := ""
	for _, item := range got {
		if item.Family != prev {
			if seen[item.Family] {
				t.Errorf("family %q appears in more than one block", item.Family)
			}
			seen[item.Family] = true
			prev = item.Family
		}
	}
}

// Go randomises map iteration order, which is what made the old positional
// palette repaint the bar differently on every request.
func TestBuildOpBreakdownIsDeterministic(t *testing.T) {
	counts := map[string]int{
		"invoke_host_function": 12,
		"payment":              12,
		"create_account":       12,
		"manage_sell_offer":    12,
		"change_trust":         12,
		"set_options":          12,
		"extend_footprint_ttl": 12,
	}
	first := buildOpBreakdown(counts, 84)
	for i := 0; i < 50; i++ {
		next := buildOpBreakdown(counts, 84)
		for j := range first {
			if first[j].Name != next[j].Name || first[j].Family != next[j].Family || first[j].Tier != next[j].Tier {
				t.Fatalf("run %d differed at segment %d: %+v vs %+v", i, j, first[j], next[j])
			}
		}
	}
}

func TestBuildOpBreakdownAssignsFamilyTiers(t *testing.T) {
	got := buildOpBreakdown(map[string]int{
		"payment":                     30,
		"path_payment_strict_receive": 15,
		"create_account":              9,
		"claim_claimable_balance":     4,
		"invoke_host_function":        37,
	}, 95)

	byName := make(map[string]int, len(got))
	for _, item := range got {
		byName[item.Name] = item.Tier
		if item.Family == "" {
			t.Errorf("segment %q has no family", item.Name)
		}
	}

	// Four transfer-family types rank 1..3, with the fourth sharing tier 3.
	for name, wantTier := range map[string]int{
		"Payment":        1,
		"Path Payment":   2,
		"Create Account": 3,
		"Claim Balance":  3,
		"Contract Call":  1,
	} {
		if byName[name] != wantTier {
			t.Errorf("%s tier = %d, want %d", name, byName[name], wantTier)
		}
	}
}

// Every family the builder can emit needs a Tailwind class, or the v1 ledger
// fragment renders an unpainted segment.
func TestOpFamilyTailwindCoversEveryFamily(t *testing.T) {
	families := []string{
		gateway.OpFamilyContract, gateway.OpFamilyTransfer, gateway.OpFamilyMarket,
		gateway.OpFamilyControls, gateway.OpFamilyAgent, gateway.OpFamilyRevoke,
		gateway.OpFamilyOther,
	}
	for _, family := range families {
		if opFamilyTailwind[family] == "" {
			t.Errorf("family %q has no Tailwind class", family)
		}
	}
}

func TestBuildOpBreakdownEmptyInput(t *testing.T) {
	if got := buildOpBreakdown(map[string]int{}, 0); len(got) != 0 {
		t.Errorf("expected no segments, got %d", len(got))
	}
	if got := buildOpBreakdown(map[string]int{"payment": 3}, 0); len(got) != 0 {
		t.Errorf("expected no segments for zero total, got %d", len(got))
	}
}

// A ledger's two operation counts measure different populations: OperationCount
// covers successful transactions only, TxSetOperationCount the whole set. The
// breakdown bar is built from the operations endpoint, which returns the whole
// set, so the header has to use the same population or the two disagree.
func TestLedgerOperationTotalUsesTxSetCount(t *testing.T) {
	tests := []struct {
		name string
		in   gateway.Ledger
		want int
	}{
		{
			name: "failed transactions included",
			in:   gateway.Ledger{OperationCount: 11, TxSetOperationCount: 15},
			want: 15,
		},
		{
			name: "no failures, counts agree",
			in:   gateway.Ledger{OperationCount: 108, TxSetOperationCount: 108},
			want: 108,
		},
		{
			name: "falls back when the tx-set count is absent",
			in:   gateway.Ledger{OperationCount: 42},
			want: 42,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ledgerOperationTotal(tt.in); got != tt.want {
				t.Errorf("ledgerOperationTotal() = %d, want %d", got, tt.want)
			}
		})
	}
}

// The breakdown is an aggregate, so the operations request is sized to the
// ledger rather than to a fixed cap — a cap would change the proportions
// between families and present them as fact.
func TestLedgerOpsFetchLimitSizesToLedger(t *testing.T) {
	tests := []struct {
		name string
		in   gateway.Ledger
		want int
	}{
		{
			name: "small ledger asks for exactly its operations",
			in:   gateway.Ledger{OperationCount: 11, TxSetOperationCount: 15},
			want: 15,
		},
		{
			name: "busy ledger is no longer truncated at the old 200 cap",
			in:   gateway.Ledger{TxSetOperationCount: 847},
			want: 847,
		},
		{
			name: "absent count degrades to a sample rather than nothing",
			in:   gateway.Ledger{},
			want: ledgerOpsFallback,
		},
		{
			name: "implausible count is bounded",
			in:   gateway.Ledger{TxSetOperationCount: 500000},
			want: ledgerOpsCeiling,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ledgerOpsFetchLimit(tt.in); got != tt.want {
				t.Errorf("ledgerOpsFetchLimit() = %d, want %d", got, tt.want)
			}
		})
	}
}
