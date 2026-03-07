package gateway

import (
	"context"
	"sync"
	"time"
)

type cacheEntry struct {
	value     any
	expiresAt time.Time
}

// Cache is a simple in-memory TTL cache.
type Cache struct {
	mu      sync.RWMutex
	entries map[string]*cacheEntry
	cancel  context.CancelFunc
}

// NewCache creates a cache with a background cleanup goroutine.
func NewCache(ctx context.Context) *Cache {
	ctx, cancel := context.WithCancel(ctx)
	c := &Cache{
		entries: make(map[string]*cacheEntry),
		cancel:  cancel,
	}
	go c.cleanup(ctx)
	return c
}

// Get returns the cached value for key, or nil if missing/expired.
func (c *Cache) Get(key string) (any, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	e, ok := c.entries[key]
	if !ok || time.Now().After(e.expiresAt) {
		return nil, false
	}
	return e.value, true
}

// Set stores a value with the given TTL.
func (c *Cache) Set(key string, value any, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[key] = &cacheEntry{
		value:     value,
		expiresAt: time.Now().Add(ttl),
	}
}

// Stop cancels the background cleanup goroutine.
func (c *Cache) Stop() {
	c.cancel()
}

func (c *Cache) cleanup(ctx context.Context) {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.mu.Lock()
			now := time.Now()
			for k, e := range c.entries {
				if now.After(e.expiresAt) {
					delete(c.entries, k)
				}
			}
			c.mu.Unlock()
		}
	}
}
