package search

import "testing"

func TestClassify(t *testing.T) {
	tests := []struct {
		name, in string
		typ      ClassType
	}{
		{"tx hash", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", ClassTxHash},
		{"account", "GAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA", ClassAccount},
		{"contract", "CAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA", ClassContract},
		{"ledger", "12345", ClassLedger},
		{"federation", "alice*example.com", ClassFederation},
		{"unknown", "USDC swaps", ClassUnknown},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Classify(tt.in)
			if got.Type != tt.typ {
				t.Fatalf("Classify(%q)=%s want %s", tt.in, got.Type, tt.typ)
			}
		})
	}
}

func TestParse(t *testing.T) {
	tests := []struct{ in, topic, fn, asset, tm, status, scope string }{
		{"all USDC swaps today", "swap", "swap", "USDC", "24h", "", "all"},
		{"failed transfers last hour", "transfer", "transfer", "", "1h", "failed", "all"},
		{"classic payments this week", "transfer", "transfer", "", "7d", "", "classic"},
		{"soroban approve EURC", "approve", "approve", "EURC", "1h", "", "soroban"},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got, conf := Parse(tt.in)
			if conf < 0.5 {
				t.Fatalf("confidence %.2f too low", conf)
			}
			if got.Topic != tt.topic || got.Fn != tt.fn || got.Asset != tt.asset || got.Time != tt.tm || got.Status != tt.status || got.Scope != tt.scope {
				t.Fatalf("Parse(%q)=%+v", tt.in, got)
			}
		})
	}
}

func TestRegistrySearch(t *testing.T) {
	got := DefaultRegistry().Search("swap", 3)
	if len(got) == 0 {
		t.Fatal("expected matches")
	}
	if got[0].Name != "swap" {
		t.Fatalf("top match %q, want swap", got[0].Name)
	}
}
