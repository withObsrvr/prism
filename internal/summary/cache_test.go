package summary

import (
	"testing"
	"time"
)

func TestLedgerSummaryCacheKeyIncludesEveryBehaviorVersion(t *testing.T) {
	key := NewLedgerSummaryCacheKey("testnet", 123)
	if key.BandVersion == "" || key.BaselineVersion == "" || key.ThresholdVersion == "" || key.RegistryVersion == "" || key.TemplateVersion == "" || key.SelectorVersion == "" || key.ShadowIdentity == "" {
		t.Fatalf("incomplete cache key: %+v", key)
	}
	changed := key
	changed.TemplateVersion = "next"
	if changed == key || changed.String() == key.String() {
		t.Fatal("template version did not change cache identity")
	}
	shadow := NewLedgerSummaryCacheKey("testnet", 123, "jev:v1")
	if shadow == key || shadow.String() == key.String() {
		t.Fatal("shadow selector identity did not change cache identity")
	}
}

func TestLedgerSummaryCacheExpiresIncompleteSnapshots(t *testing.T) {
	now := time.Unix(100, 0)
	cache := NewLedgerSummaryCache()
	cache.ttl = time.Minute
	cache.now = func() time.Time { return now }
	key := NewLedgerSummaryCacheKey("testnet", 123)
	want := LedgerSummaryEnvelope{Selection: Selection{Lead: InterpretationAllSucceeded}}
	cache.Set(key, want)
	if got, ok := cache.Get(key); !ok || got.Selection.Lead != want.Selection.Lead {
		t.Fatalf("cache hit = %+v, %v", got, ok)
	}
	now = now.Add(time.Minute)
	if _, ok := cache.Get(key); ok {
		t.Fatal("expired snapshot remained cached")
	}
}
