package intent

import "testing"

func TestMatchExpiringContracts(t *testing.T) {
	m, ok := DefaultRegistry().Match("which contracts are expiring this week?")
	if !ok {
		t.Fatal("expected match")
	}
	if m.ID != ExpiringContracts {
		t.Fatalf("intent=%s want %s", m.ID, ExpiringContracts)
	}
	if m.Slots["time"] != "7d" {
		t.Fatalf("time=%q want 7d", m.Slots["time"])
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
