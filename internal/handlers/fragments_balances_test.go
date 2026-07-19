package handlers

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/withObsrvr/prism/internal/gateway"
)

func TestContractCurrentBalancesFragmentUsesGenericAddressEndpoint(t *testing.T) {
	const contractID = "CCONTRACT"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/lake/v1/testnet/api/v1/silver/addresses/"+contractID+"/balances" {
			t.Fatalf("unexpected endpoint %s", r.URL.String())
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"address":%q,"balances":[{"asset_type":"native","asset_code":"XLM","balance":"1000.0000000","balance_source":"contract_storage_state"}],"total_balances":1,"sources":["contract_storage_state"],"partial":false}`, contractID)
	}))
	defer server.Close()

	h, stop := balanceFragmentTestHandlers(t, server.URL)
	defer stop()
	req := httptest.NewRequest(http.MethodGet, "/fragments/contract/"+contractID+"/balances?network=testnet&surface=v2", nil)
	req.SetPathValue("id", contractID)
	rec := httptest.NewRecorder()

	h.ContractCurrentBalancesFragment(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	for _, want := range []string{"Current balances", "1,000", "Current contract storage"} {
		if !strings.Contains(rec.Body.String(), want) {
			t.Errorf("fragment missing %q: %s", want, rec.Body.String())
		}
	}
}

func TestSmartAccountCurrentBalancesFragmentUsesWalletEndpointAndUpdatesHero(t *testing.T) {
	const contractID = "CWALLET"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/lake/v1/testnet/api/v1/silver/smart-wallets/"+contractID+"/balances" {
			t.Fatalf("unexpected endpoint %s", r.URL.String())
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"contract_id":%q,"native_balance":"9995.0000000","native_balance_source":"contract_storage_state","balances":[{"asset_type":"native","asset_code":"XLM","balance":"9995.0000000","balance_source":"contract_storage_state"}],"count":1,"balance_status":"materialized"}`, contractID)
	}))
	defer server.Close()

	h, stop := balanceFragmentTestHandlers(t, server.URL)
	defer stop()
	req := httptest.NewRequest(http.MethodGet, "/fragments/smart-account/"+contractID+"/balances?network=testnet&surface=v2", nil)
	req.SetPathValue("id", contractID)
	rec := httptest.NewRecorder()

	h.SmartAccountCurrentBalancesFragment(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	for _, want := range []string{"Current balances", "9,995", `id="px-smart-balance-hero"`, `hx-swap-oob="outerHTML"`} {
		if !strings.Contains(rec.Body.String(), want) {
			t.Errorf("fragment missing %q: %s", want, rec.Body.String())
		}
	}
}

func TestContractCurrentBalancesFragmentRendersUnavailableOnGatewayFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "temporarily unavailable", http.StatusServiceUnavailable)
	}))
	defer server.Close()

	h, stop := balanceFragmentTestHandlers(t, server.URL)
	defer stop()
	req := httptest.NewRequest(http.MethodGet, "/fragments/contract/CCONTRACT/balances?network=testnet", nil)
	req.SetPathValue("id", "CCONTRACT")
	rec := httptest.NewRecorder()

	h.ContractCurrentBalancesFragment(rec, req)

	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "Balances are temporarily unavailable") {
		t.Fatalf("soft failure response = %d %q", rec.Code, rec.Body.String())
	}
}

func TestSmartAccountCurrentBalancesFragmentWithoutGatewayRendersUnavailable(t *testing.T) {
	h := &Handlers{Logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
	req := httptest.NewRequest(http.MethodGet, "/fragments/smart-account/CWALLET/balances?surface=v2", nil)
	req.SetPathValue("id", "CWALLET")
	rec := httptest.NewRecorder()

	h.SmartAccountCurrentBalancesFragment(rec, req)

	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "Balances are temporarily unavailable") {
		t.Fatalf("unavailable response = %d %q", rec.Code, rec.Body.String())
	}
}

func balanceFragmentTestHandlers(t *testing.T, baseURL string) (*Handlers, func()) {
	t.Helper()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	gw := gateway.New(gateway.Config{BaseURL: baseURL, APIKey: "test", Timeout: time.Second}, logger, context.Background())
	return &Handlers{Logger: logger, Gateway: gw}, gw.Stop
}
