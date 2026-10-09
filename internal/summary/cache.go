package summary

import (
	"fmt"
	"sync"
	"time"
)

const (
	LedgerSummaryCacheTTL        = 5 * time.Minute
	ledgerSummaryCacheMaxEntries = 2048
)

type LedgerSummaryCacheKey struct {
	Network          string
	Sequence         int64
	BandVersion      string
	BaselineVersion  string
	ThresholdVersion string
	RegistryVersion  string
	TemplateVersion  string
	SelectorVersion  string
	ShadowIdentity   string
}

func NewLedgerSummaryCacheKey(network string, sequence int64, shadowIdentity ...string) LedgerSummaryCacheKey {
	shadow := "disabled"
	if len(shadowIdentity) > 0 && shadowIdentity[0] != "" {
		shadow = shadowIdentity[0]
	}
	return LedgerSummaryCacheKey{
		Network: network, Sequence: sequence,
		BandVersion: LedgerBandVersion, BaselineVersion: LedgerBaselineVersion,
		ThresholdVersion: LedgerThresholdVersion, RegistryVersion: LedgerInterpretationRegistryVersion,
		TemplateVersion: LedgerTemplateVersion, SelectorVersion: LedgerSelectorVersion, ShadowIdentity: shadow,
	}
}

func (k LedgerSummaryCacheKey) String() string {
	return fmt.Sprintf("%s:%d:%s:%s:%s:%s:%s:%s:%s", k.Network, k.Sequence, k.BandVersion, k.BaselineVersion, k.ThresholdVersion, k.RegistryVersion, k.TemplateVersion, k.SelectorVersion, k.ShadowIdentity)
}

type ledgerSummaryCacheEntry struct {
	envelope  LedgerSummaryEnvelope
	expiresAt time.Time
}

// LedgerSummaryCache is a bounded in-process snapshot cache. The short TTL
// avoids pinning an incomplete projection forever while the version-complete
// key prevents behavior changes from reusing an older interpretation.
type LedgerSummaryCache struct {
	mu         sync.Mutex
	entries    map[LedgerSummaryCacheKey]ledgerSummaryCacheEntry
	ttl        time.Duration
	maxEntries int
	now        func() time.Time
}

func NewLedgerSummaryCache() *LedgerSummaryCache {
	return &LedgerSummaryCache{entries: make(map[LedgerSummaryCacheKey]ledgerSummaryCacheEntry), ttl: LedgerSummaryCacheTTL, maxEntries: ledgerSummaryCacheMaxEntries, now: time.Now}
}

func (c *LedgerSummaryCache) Get(key LedgerSummaryCacheKey) (LedgerSummaryEnvelope, bool) {
	if c == nil {
		return LedgerSummaryEnvelope{}, false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	entry, ok := c.entries[key]
	if !ok {
		return LedgerSummaryEnvelope{}, false
	}
	if !c.now().Before(entry.expiresAt) {
		delete(c.entries, key)
		return LedgerSummaryEnvelope{}, false
	}
	return entry.envelope, true
}

func (c *LedgerSummaryCache) Set(key LedgerSummaryCacheKey, envelope LedgerSummaryEnvelope) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	now := c.now()
	for existing, entry := range c.entries {
		if !now.Before(entry.expiresAt) {
			delete(c.entries, existing)
		}
	}
	for len(c.entries) >= c.maxEntries {
		for existing := range c.entries {
			delete(c.entries, existing)
			break
		}
	}
	c.entries[key] = ledgerSummaryCacheEntry{envelope: envelope, expiresAt: now.Add(c.ttl)}
}
