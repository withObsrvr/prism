package gateway

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestGetSorobanConfigAtLedgerUsesLimitsEndpointAndDirectSchema(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.RequestURI(); got != "/lake/v1/testnet/api/v1/silver/soroban/config/limits?ledger=123" {
			t.Fatalf("request URI = %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"instructions":{"ledger_max":100000000,"tx_max":10000000,"fee_rate_per_increment":1000},"ledger":{"max_read_entries":200,"max_read_bytes":133120,"max_write_entries":125,"max_write_bytes":66560},"transaction":{"max_read_entries":40,"max_write_entries":25},"contract":{"max_size_bytes":65536},"updated_at":"2026-10-09T12:00:00Z"}`)
	}))
	defer server.Close()

	client := New(Config{BaseURL: server.URL, APIKey: "test", Timeout: time.Second}, slog.New(slog.NewTextHandler(io.Discard, nil)), context.Background())
	defer client.Stop()
	config, err := client.GetSorobanConfigAtLedger(context.Background(), "testnet", 123)
	if err != nil {
		t.Fatal(err)
	}
	if config.Instructions.LedgerMax != 100000000 || config.LedgerLimits.MaxWriteEntries != 125 || config.TxLimits.MaxWriteEntries != 25 {
		t.Fatalf("decoded config = %+v", config)
	}
}
