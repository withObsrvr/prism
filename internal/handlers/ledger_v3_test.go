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
	vmv2 "github.com/withObsrvr/prism/internal/templates/v2/viewmodel"
)

func TestLedgerDetailV3RejectsInvalidSequence(t *testing.T) {
	h := New(nil, nil, "live")
	req := httptest.NewRequest(http.MethodGet, "/v2/ledger/nope", nil)
	req.SetPathValue("sequence", "nope")
	res := httptest.NewRecorder()

	h.LedgerDetailV3(res, req)

	if res.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusBadRequest)
	}
}

func TestLedgerDetailV3RequiresGatewayEvidence(t *testing.T) {
	h := New(nil, nil, "live")
	req := httptest.NewRequest(http.MethodGet, "/v2/ledger/123", nil)
	req.SetPathValue("sequence", "123")
	res := httptest.NewRecorder()

	h.LedgerDetailV3(res, req)

	if res.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusServiceUnavailable)
	}
}

func TestLedgerDetailV3NeverLeaksFixtureValuesWhenOptionalEvidenceIsUnavailable(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/lake/v1/testnet/api/v1/silver/ledgers/123/full" {
			_, _ = io.WriteString(w, `{"ledger_sequence":123,"ledger":{"sequence":123,"ledger_hash":"live-hash","closed_at":"2026-10-09T12:00:00Z","protocol_version":23,"transaction_count":0,"operation_count":0,"successful_tx_count":0,"failed_tx_count":0},"transactions":[],"operations":[]}`)
			return
		}
		http.Error(w, `{"error":"not available"}`, http.StatusNotFound)
	}))
	defer upstream.Close()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	client := gateway.New(gateway.Config{BaseURL: upstream.URL, APIKey: "test", Timeout: time.Second}, logger, context.Background())
	defer client.Stop()
	h := New(logger, client, "live")
	req := httptest.NewRequest(http.MethodGet, "/v2/ledger/123?network=testnet", nil)
	req.SetPathValue("sequence", "123")
	res := httptest.NewRecorder()

	h.LedgerDetailV3(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", res.Code, res.Body.String())
	}
	body := res.Body.String()
	for _, forbidden := range []string{"9f4c2b7e18a05d63", "118 transactions", "42×", "97%", "Prototype · mock data"} {
		if strings.Contains(body, forbidden) {
			t.Errorf("response contains fixture value %q", forbidden)
		}
	}
	for _, required := range []string{"live-hash", "Capacity cannot be stated", "Fee evidence unavailable", "Evidence-backed · testnet"} {
		if !strings.Contains(body, required) {
			t.Errorf("response missing %q", required)
		}
	}
	if !strings.Contains(body, `rel="canonical" href="/v2/ledger/123"`) {
		t.Error("response does not identify the promoted canonical route")
	}
}

func TestRedirectLedgerDetailV3PreservesNetwork(t *testing.T) {
	h := New(nil, nil, "live")
	req := httptest.NewRequest(http.MethodGet, "/v2/ledger/123/v3?network=testnet", nil)
	req.SetPathValue("sequence", "123")
	res := httptest.NewRecorder()

	h.RedirectLedgerDetailV3(res, req)

	if res.Code != http.StatusPermanentRedirect {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusPermanentRedirect)
	}
	if got := res.Header().Get("Location"); got != "/v2/ledger/123?network=testnet" {
		t.Fatalf("Location = %q", got)
	}
}

func TestLedgerV3CapacityNarrativeDoesNotRenderMissingFootprintsAsZero(t *testing.T) {
	data := liveLedgerDetailV3Data("123", "testnet")
	applyLedgerV3LedeCapacity(&data,
		vmv2.LedgerV3Meter{Name: "CPU instructions", Pct: 25},
		&gateway.LedgerSoroban{TotalCPUInsns: 25, FootprintEntriesAvailable: false},
		&gateway.SorobanConfig{Instructions: gateway.SorobanInstructionLimits{LedgerMax: 100}},
	)
	if strings.Contains(data.Lede[0], "0% of write capacity") || !strings.Contains(data.Lede[0], "unknown, not zero") {
		t.Fatalf("lede treats unavailable footprints as measured: %s", data.Lede[0])
	}
}

func TestLedgerV3PanesLabelTruncatedEvidenceAsSample(t *testing.T) {
	data := liveLedgerDetailV3Data("123", "testnet")
	txs := []gateway.Transaction{{TransactionHash: "abc", Successful: false, ResultCode: "tx_FAILED"}}
	applyLedgerV3TxPane(&data, "testnet", txs, nil, 2, 3)
	applyLedgerV3Failures(&data, txs, nil, 2)

	if data.TxPane.Title != "Available transaction sample" || !strings.Contains(data.TxPane.Intro, "1 of 2 transactions") {
		t.Fatalf("transaction pane does not disclose sample: %+v", data.TxPane)
	}
	if !strings.Contains(data.Failures.Note, "Sample only") {
		t.Fatalf("failure interpretation does not disclose sample: %+v", data.Failures)
	}
}
