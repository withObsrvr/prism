package gateway

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestClientCoalescesRecentLedgerCacheMisses(t *testing.T) {
	var calls atomic.Int32
	started := make(chan struct{})
	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/lake/v1/mainnet/api/v1/silver/ledgers/recent" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		if calls.Add(1) == 1 {
			close(started)
		}
		<-release
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"latest_sequence":123,"count":0,"ledgers":[]}`))
	}))
	defer server.Close()

	client := New(Config{BaseURL: server.URL, APIKey: "test", Timeout: time.Second}, slog.New(slog.NewTextHandler(io.Discard, nil)), context.Background())
	defer client.Stop()

	const n = 5
	var wg sync.WaitGroup
	errs := make(chan error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := client.GetSilverRecentLedgers(context.Background(), "mainnet", 6)
			errs <- err
		}()
	}

	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("upstream request did not start")
	}
	time.Sleep(50 * time.Millisecond)
	close(release)
	wg.Wait()
	close(errs)

	for err := range errs {
		if err != nil {
			t.Fatalf("GetSilverRecentLedgers error: %v", err)
		}
	}
	if got := calls.Load(); got != 1 {
		t.Fatalf("upstream calls = %d, want 1", got)
	}
}

func TestCacheEvictsWhenFull(t *testing.T) {
	cache := &Cache{entries: make(map[string]*cacheEntry)}
	for i := 0; i < cacheMaxEntries+10; i++ {
		cache.Set(fmt.Sprintf("key-%d", i), i, time.Hour)
	}
	if got := len(cache.entries); got > cacheMaxEntries {
		t.Fatalf("cache entries = %d, want <= %d", got, cacheMaxEntries)
	}
}
