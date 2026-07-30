package intent

import (
	"strings"
	"testing"
)

func TestMatchExpiringContracts(t *testing.T) {
	m, ok := DefaultRegistry().Match("which contracts are expiring this week?")
	if !ok {
		t.Fatal("expected match")
	}
	if m.ID != ExpiringContracts {
		t.Fatalf("intent=%s want %s", m.ID, ExpiringContracts)
	}
	if m.Slots["requested_time"] != "7d" || m.Slots["window"] != "current_ttl_snapshot" {
		t.Fatalf("TTL slots=%v", m.Slots)
	}
}

func TestMatchContractArchivalWording(t *testing.T) {
	match, ok := DefaultRegistry().Match("Which contracts are nearing archival?")
	if !ok || match.ID != ExpiringContracts {
		t.Fatalf("archival wording match = %+v, ok=%t", match, ok)
	}
}

func TestMatchTransactionFailureWithEmbeddedHash(t *testing.T) {
	hash := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	match, ok := DefaultRegistry().Match("why did " + hash + " fail?")
	if !ok || match.ID != TransactionFailure || match.Slots["tx_hash"] != hash {
		t.Fatalf("transaction failure match = %+v, ok=%t", match, ok)
	}
}

func TestMatchContractAssetAndRecentFailureAnswers(t *testing.T) {
	contract := "CAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"
	tests := []struct {
		query string
		id    ID
	}{
		{"How active is contract " + contract + " today?", ContractActivity},
		{"How active is XLM today?", AssetActivity},
		{"Are any recent transactions failing?", RecentFailures},
	}
	for _, test := range tests {
		match, ok := DefaultRegistry().Match(test.query)
		if !ok || match.ID != test.id {
			t.Errorf("Match(%q) = %+v, ok=%t, want %s", test.query, match, ok, test.id)
		}
	}
}

func TestAssetActivityRequiresUnambiguousIdentity(t *testing.T) {
	registry := DefaultRegistry()
	if match, ok := registry.Match("How active is USDC today?"); ok && match.ID == AssetActivity {
		t.Fatalf("ambiguous asset code matched as answer: %+v", match)
	}
	issuer := "G" + strings.Repeat("A", 55)
	match, ok := registry.Match("How active is USDC:" + issuer + " today?")
	if !ok || match.ID != AssetActivity || match.Slots["asset"] != "USDC:"+issuer {
		t.Fatalf("issued asset identity match = %+v, ok=%t", match, ok)
	}
	if got := assetDisplayLabel(match.Slots["asset"]); got != "USDC" {
		t.Fatalf("issued asset display label = %q, want USDC", got)
	}
}

func TestMatchProtocolBusy(t *testing.T) {
	m, ok := DefaultRegistry().Match("is Soroswap busy?")
	if !ok {
		t.Fatal("expected match")
	}
	if m.ID != ProtocolBusy {
		t.Fatalf("intent=%s want %s", m.ID, ProtocolBusy)
	}
	if m.Slots["protocol"] != "soroswap" {
		t.Fatalf("protocol=%q want soroswap", m.Slots["protocol"])
	}
	if m.Slots["time"] != "24h" {
		t.Fatalf("time=%q want 24h", m.Slots["time"])
	}
}

func TestDoesNotMatchExactEntities(t *testing.T) {
	inputs := []string{
		"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		"GAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
		"CAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
		"12345",
	}
	for _, in := range inputs {
		if m, ok := DefaultRegistry().Match(in); ok {
			t.Fatalf("Match(%q)=%+v, want no intent", in, m)
		}
	}
}

func TestDoesNotMatchBareProtocol(t *testing.T) {
	if m, ok := DefaultRegistry().Match("soroswap"); ok {
		t.Fatalf("bare protocol matched %+v", m)
	}
}
