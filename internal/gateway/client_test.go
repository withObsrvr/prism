package gateway

import (
	"context"
	"errors"
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

func TestClientDoesNotNegativeCacheAccountOverviewContextFailure(t *testing.T) {
	var calls atomic.Int32
	accountID := "GTESTACCOUNT"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/lake/v1/mainnet/api/v1/silver/explorer/account" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		if r.URL.Query().Get("account_id") != accountID {
			t.Errorf("account_id = %q, want %q", r.URL.Query().Get("account_id"), accountID)
		}
		if calls.Add(1) == 1 {
			<-r.Context().Done()
			return
		}
		writeAccountOverview(t, w, accountID)
	}))
	defer server.Close()

	client := New(Config{BaseURL: server.URL, APIKey: "test", Timeout: time.Second}, slog.New(slog.NewTextHandler(io.Discard, nil)), context.Background())
	defer client.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	if _, err := client.GetAccountOverview(ctx, "mainnet", accountID); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("first GetAccountOverview error = %v, want context deadline exceeded", err)
	}

	if _, err := client.GetAccountOverview(context.Background(), "mainnet", accountID); err != nil {
		t.Fatalf("second GetAccountOverview error = %v", err)
	}
	if got := calls.Load(); got != 2 {
		t.Fatalf("upstream calls = %d, want 2", got)
	}
}

func TestClientAccountOverviewInflightWaitHonorsCallerContext(t *testing.T) {
	var calls atomic.Int32
	started := make(chan struct{})
	release := make(chan struct{})
	var releaseOnce sync.Once
	t.Cleanup(func() {
		releaseOnce.Do(func() { close(release) })
	})

	accountID := "GTESTACCOUNT"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/lake/v1/mainnet/api/v1/silver/explorer/account" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		if calls.Add(1) == 1 {
			close(started)
		}
		<-release
		writeAccountOverview(t, w, accountID)
	}))
	defer server.Close()

	client := New(Config{BaseURL: server.URL, APIKey: "test", Timeout: time.Second}, slog.New(slog.NewTextHandler(io.Discard, nil)), context.Background())
	defer client.Stop()

	firstErr := make(chan error, 1)
	go func() {
		_, err := client.GetAccountOverview(context.Background(), "mainnet", accountID)
		firstErr <- err
	}()

	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("upstream request did not start")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	if _, err := client.GetAccountOverview(ctx, "mainnet", accountID); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("waiting GetAccountOverview error = %v, want context deadline exceeded", err)
	}
	if got := calls.Load(); got != 1 {
		t.Fatalf("upstream calls while request was in flight = %d, want 1", got)
	}

	releaseOnce.Do(func() { close(release) })
	select {
	case err := <-firstErr:
		if err != nil {
			t.Fatalf("first GetAccountOverview error = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("first GetAccountOverview did not finish")
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

func writeAccountOverview(t *testing.T, w http.ResponseWriter, accountID string) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	_, _ = fmt.Fprintf(w, `{"account":{"account_id":%q,"balance":"1.0000000","sequence_number":"1"},"recent_operations":[],"recent_transfers":[]}`, accountID)
}
