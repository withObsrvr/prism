package server

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLedgerDetailRoutePromotion(t *testing.T) {
	app := &Application{
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		Config: Config{DataSource: "mock"},
	}
	routes := app.Routes()

	t.Run("canonical route uses evidence-only handler", func(t *testing.T) {
		res := httptest.NewRecorder()
		routes.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/v2/ledger/123", nil))
		if res.Code != http.StatusServiceUnavailable {
			t.Fatalf("status = %d, want %d from promoted handler", res.Code, http.StatusServiceUnavailable)
		}
	})

	t.Run("former candidate route redirects", func(t *testing.T) {
		res := httptest.NewRecorder()
		routes.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/v2/ledger/123/v3?network=testnet", nil))
		if res.Code != http.StatusPermanentRedirect {
			t.Fatalf("status = %d, want %d", res.Code, http.StatusPermanentRedirect)
		}
		if got := res.Header().Get("Location"); got != "/v2/ledger/123?network=testnet" {
			t.Fatalf("Location = %q", got)
		}
	})
}
