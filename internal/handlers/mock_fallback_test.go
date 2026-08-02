package handlers

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/withObsrvr/prism/internal/gateway"
)

func quietHandlers(gw *gateway.Client, source string) *Handlers {
	return &Handlers{Gateway: gw, DataSource: source, Logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
}

// A gateway failure must never render fixtures. It used to: buildTxFragmentData
// returned nil for both "fixtures were requested" and "the load failed", and the
// caller filled the nil with mockTxReceiptData. A reader then got a confident,
// entirely fabricated receipt for a transaction that had not loaded.
func TestGatewayFailureDoesNotRenderFixtures(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = io.WriteString(w, `{"error":{"code":"internal_error","message":"boom"}}`)
	}))
	defer srv.Close()

	gw := gateway.New(gateway.Config{BaseURL: srv.URL, APIKey: "k", Timeout: time.Second}, slog.New(slog.NewTextHandler(io.Discard, nil)), context.Background())
	h := quietHandlers(gw, "auto")

	req := httptest.NewRequest(http.MethodGet, "/fragments/tx/v2/abc/hero?network=testnet", nil)
	res := h.buildTxFragmentData(req, "testnet", "abc", "abc")

	if res.Err == nil {
		t.Fatal("gateway failure produced no error: the caller cannot tell it failed")
	}
	if res.Data != nil {
		t.Errorf("gateway failure produced fixture data: %+v", res.Data)
	}
	if res.Demo {
		t.Error("a failed load was reported as deliberate demo data")
	}
}

// Fixtures are legitimate when asked for, but must be flagged so the page can
// say so.
func TestExplicitMockIsFlaggedAsDemo(t *testing.T) {
	for _, tt := range []struct {
		name   string
		source string
		url    string
	}{
		{"data_source=mock", "mock", "/fragments/tx/v2/abc/hero?network=testnet"},
		{"?mock=true", "auto", "/fragments/tx/v2/abc/hero?network=testnet&mock=true"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			h := quietHandlers(nil, tt.source)
			res := h.buildTxFragmentData(httptest.NewRequest(http.MethodGet, tt.url, nil), "testnet", "abc", "abc")
			if res.Err != nil {
				t.Fatalf("unexpected error: %v", res.Err)
			}
			if !res.Demo || res.Data == nil || !res.Data.Demo {
				t.Errorf("fixtures not flagged: demo=%v dataDemo=%v", res.Demo, res.Data != nil && res.Data.Demo)
			}
		})
	}
}

// With no gateway configured at all there is nothing else to show, but the page
// still has to disclose it.
func TestNoGatewayRendersFlaggedFixtures(t *testing.T) {
	h := quietHandlers(nil, "auto")
	res := h.buildTxFragmentData(httptest.NewRequest(http.MethodGet, "/fragments/tx/v2/abc/hero", nil), "testnet", "abc", "abc")
	if res.Data == nil || !res.Data.Demo {
		t.Fatalf("no-gateway path did not flag fixtures: %+v", res)
	}
}

// The ledger path had the identical bug.
func TestLedgerGatewayFailureDoesNotRenderFixtures(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = io.WriteString(w, `{"error":{"code":"internal_error"}}`)
	}))
	defer srv.Close()

	gw := gateway.New(gateway.Config{BaseURL: srv.URL, APIKey: "k", Timeout: time.Second}, slog.New(slog.NewTextHandler(io.Discard, nil)), context.Background())
	h := quietHandlers(gw, "auto")

	req := httptest.NewRequest(http.MethodGet, "/fragments/ledger/1/txs?network=testnet", nil)
	res := h.buildLedgerFragmentData(req, "testnet", "1")
	if res.Err == nil || res.Data != nil {
		t.Fatalf("ledger failure produced fixtures: err=%v data=%v", res.Err, res.Data != nil)
	}
}

// End to end: the failing-gateway fragment must not contain fixture copy.
func TestFailedFragmentResponseContainsNoFixtureCopy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = io.WriteString(w, `{"error":{"code":"internal_error"}}`)
	}))
	defer srv.Close()

	gw := gateway.New(gateway.Config{BaseURL: srv.URL, APIKey: "k", Timeout: time.Second}, slog.New(slog.NewTextHandler(io.Discard, nil)), context.Background())
	h := quietHandlers(gw, "auto")

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/fragments/tx/v2/abc/hero?network=testnet", nil)
	req.SetPathValue("hash", "abc")
	h.TxV2HeroFragment(rec, req)

	body := rec.Body.String()
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", rec.Code)
	}
	for _, fixture := range []string{"Soroswap", "GABC...7X92", "485"} {
		if strings.Contains(body, fixture) {
			t.Errorf("failed fragment leaked fixture copy %q", fixture)
		}
	}
}
