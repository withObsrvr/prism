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

func TestHomeSummaryUsesBoundedLastGoodSnapshotAfterRefreshFailure(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/lake/v1/mainnet/api/v1/home/summary" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		if calls.Add(1) == 1 {
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"network":"mainnet","generated_at":"2026-07-31T12:00:00Z","components":{"leaders":{"status":"ready"}},"leaders":[{"contract_id":"CAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAD2KM"}]}`)
			return
		}
		http.Error(w, "temporarily unavailable", http.StatusServiceUnavailable)
	}))
	defer server.Close()

	client := New(Config{BaseURL: server.URL, APIKey: "test", Timeout: time.Second}, slog.New(slog.NewTextHandler(io.Discard, nil)), context.Background())
	defer client.Stop()

	fresh, err := client.GetHomeSummary(context.Background(), "mainnet")
	if err != nil {
		t.Fatalf("initial GetHomeSummary error: %v", err)
	}
	if fresh.Delivery.UsedLastGood {
		t.Fatal("fresh response was marked as a retained snapshot")
	}

	cacheKey := "mainnet:home_summary"
	client.cache.Set(cacheKey, fresh, -time.Second)
	retained, err := client.GetHomeSummary(context.Background(), "mainnet")
	if err != nil {
		t.Fatalf("fallback GetHomeSummary error: %v", err)
	}
	if !retained.Delivery.UsedLastGood || retained.GeneratedAt != fresh.GeneratedAt || len(retained.Leaders) != 1 {
		t.Fatalf("retained response = %+v", retained)
	}
	if fresh.Delivery.UsedLastGood {
		t.Fatal("fallback delivery state mutated the cached source packet")
	}

	// The negative cache should shield the upstream while the separately held
	// last-good packet remains available.
	again, err := client.GetHomeSummary(context.Background(), "mainnet")
	if err != nil || !again.Delivery.UsedLastGood {
		t.Fatalf("negative-cache fallback = %+v, %v", again, err)
	}
	if got := calls.Load(); got != 2 {
		t.Fatalf("upstream calls = %d, want 2", got)
	}
}

func TestHomeSummaryRefreshFailureWithoutLastGoodReturnsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "temporarily unavailable", http.StatusServiceUnavailable)
	}))
	defer server.Close()

	client := New(Config{BaseURL: server.URL, APIKey: "test", Timeout: time.Second}, slog.New(slog.NewTextHandler(io.Discard, nil)), context.Background())
	defer client.Stop()

	if summary, err := client.GetHomeSummary(context.Background(), "mainnet"); err == nil || summary != nil {
		t.Fatalf("GetHomeSummary = %+v, %v; want an error without retained evidence", summary, err)
	}
}

func TestClientBalanceEndpoints(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/lake/v1/testnet/api/v1/silver/addresses/CABC/balances":
			_, _ = io.WriteString(w, `{"address":"CABC","balances":[{"asset_type":"native","asset_code":"XLM","balance_raw":"99950000000","balance":"9995.0000000","decimals":7,"decimals_source":"stellar_native","balance_source":"contract_storage_state","last_updated_ledger":3689307,"last_updated_at":"2026-07-19T12:00:00Z"}],"total_balances":1,"sources":["contract_storage_state"],"partial":true,"warnings":["optional transfer history omitted"]}`)
		case "/lake/v1/testnet/api/v1/silver/smart-wallets/CABC/balances":
			_, _ = io.WriteString(w, `{"contract_id":"CABC","native_balance":"9995.0000000","native_balance_source":"contract_storage_state","balances":[{"asset_code":"XLM","asset_type":"native","balance":"9995.0000000","decimals":7,"symbol":"XLM","balance_source":"contract_storage_state"},{"asset_code":"USDC","asset_type":"credit_alphanum4","asset_issuer":"GISSUER","balance":"7.0000000","decimals":7,"symbol":"USDC","token_contract_id":"CTOKEN","balance_source":"contract_storage_state"}],"count":2,"partial":false,"balance_status":"materialized"}`)
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := New(Config{BaseURL: server.URL, APIKey: "test", Timeout: time.Second}, slog.New(slog.NewTextHandler(io.Discard, nil)), context.Background())
	defer client.Stop()

	address, err := client.GetAddressBalances(context.Background(), "testnet", "CABC")
	if err != nil {
		t.Fatalf("GetAddressBalances error: %v", err)
	}
	if address.Address != "CABC" || address.TotalBalances != 1 || !address.Partial {
		t.Fatalf("address response = %+v", address)
	}
	if got := address.Balances[0]; got.BalanceRaw != "99950000000" || got.LastUpdatedLedger == nil || *got.LastUpdatedLedger != 3689307 {
		t.Fatalf("address balance = %+v", got)
	}

	wallet, err := client.GetSmartWalletBalances(context.Background(), "testnet", "CABC")
	if err != nil {
		t.Fatalf("GetSmartWalletBalances error: %v", err)
	}
	if wallet.ContractID != "CABC" || wallet.NativeBalance != "9995.0000000" || wallet.Count != 2 || wallet.BalanceStatus != "materialized" {
		t.Fatalf("wallet response = %+v", wallet)
	}
	if got := wallet.Balances[1]; got.AssetCode != "USDC" || got.AssetIssuer != "GISSUER" || got.TokenContractID != "CTOKEN" {
		t.Fatalf("wallet balance = %+v", got)
	}
}

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

func TestClientDecodesEnrichedRecentLedger(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/lake/v1/testnet/api/v1/silver/ledgers/recent" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"latest_sequence":3707457,"count":1,"generated_at":"2026-07-28T12:00:10Z","source_ledger":{"sequence":3707457,"closed_at":"2026-07-28T12:00:05Z","age_seconds":5,"freshness":"fresh"},"ledgers":[{"ledger_sequence":3707457,"successful_tx_count":13,"failed_tx_count":1,"operation_count":15,"transaction_count":14,"transaction_set_operation_count":19,"successful_operation_count":15,"failed_operation_count":4,"validator":{"public_key":"GVALIDATOR","attribution_available":true,"status":"resolved","display_name":"SDF Testnet 3","source":"radar"},"operations":{"included":19,"successful":15,"failed":4,"classification_status":"materialized","categories":{"account_creation":1,"payments":2,"offers_and_amms":3,"trustlines":1,"claimable_balances":1,"sponsorship":1,"soroban":8,"other":2},"successful_categories":{"account_creation":1,"payments":2,"offers_and_amms":2,"trustlines":1,"claimable_balances":1,"sponsorship":1,"soroban":5,"other":2},"soroban_detail":{"contract_calls":5,"contract_deployments":1,"other":2},"successful_soroban_detail":{"contract_calls":4,"contract_deployments":1,"other":0}}}],"provenance":{"data_source":"serving.sv_ledger_stats_recent","complete_through_ledger":3707457,"partial":false,"warnings":[]}}`)
	}))
	defer server.Close()

	client := New(Config{BaseURL: server.URL, APIKey: "test", Timeout: time.Second}, slog.New(slog.NewTextHandler(io.Discard, nil)), context.Background())
	defer client.Stop()

	resp, err := client.GetSilverRecentLedgers(context.Background(), "testnet", 6)
	if err != nil {
		t.Fatalf("GetSilverRecentLedgers error: %v", err)
	}
	if len(resp.Ledgers) != 1 {
		t.Fatalf("ledgers = %d, want 1", len(resp.Ledgers))
	}
	got := resp.Ledgers[0]
	if got.TransactionCount != 14 || got.TransactionSetOperationCount != 19 || got.Operations.Included != 19 || got.Operations.Failed != 4 {
		t.Fatalf("explicit ledger counts were not decoded: %+v", got)
	}
	if got.Validator.Status != "resolved" || got.Validator.DisplayName != "SDF Testnet 3" {
		t.Fatalf("validator identity was not decoded: %+v", got.Validator)
	}
	if got.Operations.ClassificationStatus != "materialized" || got.Operations.Categories.Soroban != 8 {
		t.Fatalf("operation categories were not decoded: %+v", got.Operations)
	}
	if got.Operations.SorobanDetail.ContractCalls != 5 || got.Operations.SorobanDetail.ContractDeployments != 1 {
		t.Fatalf("Soroban detail was not decoded: %+v", got.Operations.SorobanDetail)
	}
	if resp.SourceLedger.Freshness != "fresh" || resp.SourceLedger.Sequence != 3707457 || resp.Provenance.CompleteThroughLedger != 3707457 {
		t.Fatalf("recent-ledger envelope was not decoded: source=%+v provenance=%+v", resp.SourceLedger, resp.Provenance)
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

func TestClientDoesNotCachePartialLedgerFullResponse(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/lake/v1/testnet/api/v1/silver/ledger/123/full" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		if calls.Add(1) == 1 {
			_, _ = io.WriteString(w, `{"ledger_sequence":123,"ledger":{"sequence":123,"transaction_count":1},"partial":true,"warnings":["transactions data unavailable"]}`)
			return
		}
		_, _ = io.WriteString(w, `{"ledger_sequence":123,"ledger":{"sequence":123,"transaction_count":1},"transactions":[{"transaction_hash":"abc","ledger_sequence":123,"successful":true}]}`)
	}))
	defer server.Close()

	client := New(Config{BaseURL: server.URL, APIKey: "test", Timeout: time.Second}, slog.New(slog.NewTextHandler(io.Discard, nil)), context.Background())
	defer client.Stop()

	first, err := client.GetSilverLedgerFull(context.Background(), "testnet", 123)
	if err != nil {
		t.Fatalf("first GetSilverLedgerFull error: %v", err)
	}
	if !first.Partial || len(first.Transactions) != 0 {
		t.Fatalf("first response = %+v, want partial response without transactions", first)
	}

	second, err := client.GetSilverLedgerFull(context.Background(), "testnet", 123)
	if err != nil {
		t.Fatalf("second GetSilverLedgerFull error: %v", err)
	}
	if len(second.Transactions) != 1 {
		t.Fatalf("second transactions = %d, want 1", len(second.Transactions))
	}
	if got := calls.Load(); got != 2 {
		t.Fatalf("upstream calls = %d, want 2", got)
	}

	if _, err := client.GetSilverLedgerFull(context.Background(), "testnet", 123); err != nil {
		t.Fatalf("cached GetSilverLedgerFull error: %v", err)
	}
	if got := calls.Load(); got != 2 {
		t.Fatalf("upstream calls after complete response = %d, want 2", got)
	}
}

func TestClientDoesNotCacheIncompleteLedgerFullResponse(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if calls.Add(1) == 1 {
			_, _ = io.WriteString(w, `{"ledger_sequence":123,"ledger":{"sequence":123,"transaction_count":2},"transactions":[{"transaction_hash":"abc","ledger_sequence":123,"successful":true}]}`)
			return
		}
		_, _ = io.WriteString(w, `{"ledger_sequence":123,"ledger":{"sequence":123,"transaction_count":2},"transactions":[{"transaction_hash":"abc","ledger_sequence":123,"successful":true},{"transaction_hash":"def","ledger_sequence":123,"successful":true}]}`)
	}))
	defer server.Close()

	client := New(Config{BaseURL: server.URL, APIKey: "test", Timeout: time.Second}, slog.New(slog.NewTextHandler(io.Discard, nil)), context.Background())
	defer client.Stop()

	if _, err := client.GetSilverLedgerFull(context.Background(), "testnet", 123); err != nil {
		t.Fatalf("first GetSilverLedgerFull error: %v", err)
	}
	second, err := client.GetSilverLedgerFull(context.Background(), "testnet", 123)
	if err != nil {
		t.Fatalf("second GetSilverLedgerFull error: %v", err)
	}
	if len(second.Transactions) != 2 {
		t.Fatalf("second transactions = %d, want 2", len(second.Transactions))
	}
	if got := calls.Load(); got != 2 {
		t.Fatalf("upstream calls = %d, want 2", got)
	}
}

func TestClientRejectsLedgerFullSequenceMismatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"ledger_sequence":123,"ledger":{"sequence":0}}`)
	}))
	defer server.Close()

	client := New(Config{BaseURL: server.URL, APIKey: "test", Timeout: time.Second}, slog.New(slog.NewTextHandler(io.Discard, nil)), context.Background())
	defer client.Stop()

	if _, err := client.GetSilverLedgerFull(context.Background(), "testnet", 123); err == nil {
		t.Fatal("GetSilverLedgerFull error = nil, want sequence mismatch error")
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

func TestClientSmartAccountEndpoints(t *testing.T) {
	var paths []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.RequestURI())
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/lake/v1/testnet/api/v1/silver/smart-accounts/lookup/address/GABC":
			if r.URL.Query().Get("limit") != "50" {
				t.Errorf("address limit = %q, want 50", r.URL.Query().Get("limit"))
			}
			_, _ = w.Write([]byte(`{"lookup_type":"address","lookup":"GABC","normalized":"GABC","source":"silver.smart_account_state","count":1,"contracts":[{"contract_id":"CCONTRACT","wallet_type":"openzeppelin","context_rule_count":2,"active_signer_count":3,"credential_signer_count":1,"address_signer_count":2,"active_policy_count":4,"context_rule_ids":[0,3],"last_modified_ledger":123}]}`))
		case "/lake/v1/testnet/api/v1/silver/smart-accounts/lookup/credential/deadbeef":
			_, _ = w.Write([]byte(`{"lookup_type":"credential","lookup":"deadbeef","normalized":"deadbeef","source":"silver.smart_account_state","count":0,"contracts":[]}`))
		case "/lake/v1/testnet/api/v1/silver/smart-accounts/CCONTRACT/rules":
			_, _ = w.Write([]byte(`{"contract_id":"CCONTRACT","source":"silver.smart_account_state","summary":{"contract_id":"CCONTRACT","context_rule_count":1,"active_signer_count":1,"active_policy_count":1},"count":1,"context_rules":[{"context_rule_id":7,"active":true,"event_type":"context_rule_added","last_modified_ledger":123,"signers":[{"signer_id":10,"signer_type":"external","credential_id":"deadbeef","last_modified_ledger":123,"registry_resolved":true}],"policies":[{"policy_id":2,"policy_address":"CPOLICY","last_modified_ledger":123,"registry_resolved":true}]}]}`))
		case "/lake/v1/testnet/api/v1/silver/smart-accounts/stats":
			_, _ = w.Write([]byte(`{"source":"silver.smart_account_state","contract_count":8,"active_rule_count":9,"active_signer_count":10,"credential_count":3,"address_signer_count":4,"active_policy_count":5,"last_modified_ledger":123}`))
		default:
			t.Errorf("unexpected path %s", r.URL.RequestURI())
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := New(Config{BaseURL: server.URL, APIKey: "test", Timeout: time.Second}, slog.New(slog.NewTextHandler(io.Discard, nil)), context.Background())
	defer client.Stop()

	addr, err := client.LookupSmartAccountsByAddress(context.Background(), "testnet", "GABC", 50)
	if err != nil {
		t.Fatalf("LookupSmartAccountsByAddress error: %v", err)
	}
	if addr.Count != 1 || len(addr.Contracts) != 1 || addr.Contracts[0].ActivePolicyCount != 4 {
		t.Fatalf("address lookup parsed unexpected response: %+v", addr)
	}

	cred, err := client.LookupSmartAccountsByCredential(context.Background(), "testnet", "deadbeef", 100)
	if err != nil {
		t.Fatalf("LookupSmartAccountsByCredential error: %v", err)
	}
	if cred.LookupType != "credential" || cred.Count != 0 {
		t.Fatalf("credential lookup parsed unexpected response: %+v", cred)
	}

	state, err := client.GetSmartAccountRules(context.Background(), "testnet", "CCONTRACT", nil)
	if err != nil {
		t.Fatalf("GetSmartAccountRules error: %v", err)
	}
	if state.Count != 1 || len(state.ContextRules) != 1 || len(state.ContextRules[0].Signers) != 1 {
		t.Fatalf("rules parsed unexpected response: %+v", state)
	}

	stats, err := client.GetSmartAccountStats(context.Background(), "testnet")
	if err != nil {
		t.Fatalf("GetSmartAccountStats error: %v", err)
	}
	if stats.ContractCount != 8 || stats.LastModifiedLedger == nil || *stats.LastModifiedLedger != 123 {
		t.Fatalf("stats parsed unexpected response: %+v", stats)
	}

	if len(paths) != 4 {
		t.Fatalf("upstream call count = %d, want 4, paths=%v", len(paths), paths)
	}
}

func writeAccountOverview(t *testing.T, w http.ResponseWriter, accountID string) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	_, _ = fmt.Fprintf(w, `{"account":{"account_id":%q,"balance":"1.0000000","sequence_number":"1"},"recent_operations":[],"recent_transfers":[]}`, accountID)
}

func TestHasCompleteTransactionsHonorsPartialFlag(t *testing.T) {
	full := &LedgerFullResponse{
		Ledger:       Ledger{TransactionCount: 1},
		Transactions: []Transaction{{TransactionHash: "aa"}},
	}
	if !full.HasCompleteTransactions() {
		t.Fatal("complete non-partial response reported incomplete")
	}
	full.Partial = true
	if full.HasCompleteTransactions() {
		t.Fatal("partial response must never report complete, even with a satisfying transaction count")
	}
	var nilResp *LedgerFullResponse
	if nilResp.HasCompleteTransactions() {
		t.Fatal("nil response reported complete")
	}
}
