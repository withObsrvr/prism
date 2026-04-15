package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"time"
)

// Cache TTLs per data type.
const (
	TTLNetworkStats      = 10 * time.Second
	TTLBronzeNetworkStats = 3 * time.Second // tracks ledger close (~5s, may decrease)
	TTLRecentList        = 5 * time.Second
	TTLImmutable         = 5 * time.Minute
	TTLAccount           = 30 * time.Second
	TTLContracts         = 2 * time.Minute
)

// Config holds gateway connection settings.
type Config struct {
	BaseURL string
	APIKey  string
	Timeout time.Duration
}

// Client is the gateway HTTP client.
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
	logger     *slog.Logger
	cache      *Cache
}

// New creates a gateway client. Returns nil if no API key is provided.
func New(cfg Config, logger *slog.Logger, ctx context.Context) *Client {
	if cfg.APIKey == "" {
		return nil
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://gateway.withobsrvr.com"
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}
	return &Client{
		baseURL: cfg.BaseURL,
		apiKey:  cfg.APIKey,
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
		logger: logger,
		cache:  NewCache(ctx),
	}
}

// Stop cleans up the client's background resources.
func (c *Client) Stop() {
	if c != nil && c.cache != nil {
		c.cache.Stop()
	}
}

// buildURL constructs a gateway API URL for the given network.
// Pattern: {baseURL}/lake/v1/{network}/api/v1{path}
func (c *Client) buildURL(network, path string) string {
	return fmt.Sprintf("%s/lake/v1/%s/api/v1%s", c.baseURL, network, path)
}

// doRequest executes an HTTP request with authentication.
func (c *Client) doRequest(ctx context.Context, method, rawURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, method, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("gateway: creating request: %w", err)
	}
	req.Header.Set("Authorization", "Api-Key "+c.apiKey)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("gateway: request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("gateway: reading response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, &APIError{
			StatusCode: resp.StatusCode,
			Message:    string(body),
		}
	}

	return body, nil
}

// Health checks gateway connectivity.
func (c *Client) Health(ctx context.Context, network string) error {
	u := fmt.Sprintf("%s/lake/v1/%s/health", c.baseURL, network)
	_, err := c.doRequest(ctx, http.MethodGet, u)
	return err
}

// GetNetworkStats returns network-wide statistics.
func (c *Client) GetNetworkStats(ctx context.Context, network string) (*NetworkStats, error) {
	cacheKey := network + ":network_stats"
	if v, ok := c.cache.Get(cacheKey); ok {
		return v.(*NetworkStats), nil
	}

	body, err := c.doRequest(ctx, http.MethodGet, c.buildURL(network, "/silver/stats/network"))
	if err != nil {
		return nil, err
	}

	var stats NetworkStats
	if err := json.Unmarshal(body, &stats); err != nil {
		return nil, fmt.Errorf("gateway: parsing network stats: %w", err)
	}

	c.cache.Set(cacheKey, &stats, TTLNetworkStats)
	return &stats, nil
}

// bronzeNegativeEntry is a sentinel cached on bronze fetch failure to avoid stampedes.
type bronzeNegativeEntry struct{ err error }

// GetBronzeNetworkStats returns network stats from the bronze endpoint (accurate latest ledger).
// Caches both successes and failures to prevent thundering herd during outages.
func (c *Client) GetBronzeNetworkStats(ctx context.Context, network string) (*BronzeNetworkStats, error) {
	cacheKey := network + ":bronze_network_stats"
	if v, ok := c.cache.Get(cacheKey); ok {
		if neg, isNeg := v.(*bronzeNegativeEntry); isNeg {
			return nil, neg.err
		}
		return v.(*BronzeNetworkStats), nil
	}

	body, err := c.doRequest(ctx, http.MethodGet, c.buildURL(network, "/bronze/stats/network"))
	if err != nil {
		c.cache.Set(cacheKey, &bronzeNegativeEntry{err: err}, TTLBronzeNetworkStats)
		return nil, err
	}

	var stats BronzeNetworkStats
	if err := json.Unmarshal(body, &stats); err != nil {
		wrapped := fmt.Errorf("gateway: parsing bronze network stats: %w", err)
		c.cache.Set(cacheKey, &bronzeNegativeEntry{err: wrapped}, TTLBronzeNetworkStats)
		return nil, wrapped
	}

	c.cache.Set(cacheKey, &stats, TTLBronzeNetworkStats)
	return &stats, nil
}

// GetLedgers returns ledgers in the given range.
func (c *Client) GetLedgers(ctx context.Context, network string, start, end int64, limit int, order string) ([]Ledger, error) {
	cacheKey := fmt.Sprintf("%s:ledgers:%d:%d:%d:%s", network, start, end, limit, order)
	if v, ok := c.cache.Get(cacheKey); ok {
		return v.([]Ledger), nil
	}

	params := url.Values{
		"start": {fmt.Sprintf("%d", start)},
		"end":   {fmt.Sprintf("%d", end)},
		"limit": {fmt.Sprintf("%d", limit)},
		"order": {order},
	}
	body, err := c.doRequest(ctx, http.MethodGet, c.buildURL(network, "/bronze/ledgers")+"?"+params.Encode())
	if err != nil {
		return nil, err
	}

	var resp LedgersResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("gateway: parsing ledgers: %w", err)
	}

	ttl := TTLRecentList
	if start == end {
		ttl = TTLImmutable // single ledger is immutable
	}
	c.cache.Set(cacheKey, resp.Ledgers, ttl)
	return resp.Ledgers, nil
}

// GetTransactions returns transactions in the given ledger range.
func (c *Client) GetTransactions(ctx context.Context, network string, start, end int64, limit int, order string) ([]Transaction, error) {
	cacheKey := fmt.Sprintf("%s:txs:%d:%d:%d:%s", network, start, end, limit, order)
	if v, ok := c.cache.Get(cacheKey); ok {
		return v.([]Transaction), nil
	}

	params := url.Values{
		"start": {fmt.Sprintf("%d", start)},
		"end":   {fmt.Sprintf("%d", end)},
		"limit": {fmt.Sprintf("%d", limit)},
		"order": {order},
	}
	body, err := c.doRequest(ctx, http.MethodGet, c.buildURL(network, "/bronze/transactions")+"?"+params.Encode())
	if err != nil {
		return nil, err
	}

	var resp TransactionsResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("gateway: parsing transactions: %w", err)
	}

	ttl := TTLRecentList
	if start == end {
		ttl = TTLImmutable
	}
	c.cache.Set(cacheKey, resp.Transactions, ttl)
	return resp.Transactions, nil
}

// GetSilverToken returns metadata for a SAC or custom Soroban token by contract ID.
// Used to resolve an event's contract_id into a human-readable asset symbol and decimal scale.
// Token metadata is essentially immutable, so cached aggressively.
func (c *Client) GetSilverToken(ctx context.Context, network, contractID string) (*Token, error) {
	if contractID == "" {
		return nil, fmt.Errorf("gateway: empty contract id")
	}
	cacheKey := fmt.Sprintf("%s:silver_token:%s", network, contractID)
	if v, ok := c.cache.Get(cacheKey); ok {
		return v.(*Token), nil
	}

	body, err := c.doRequest(ctx, http.MethodGet, c.buildURL(network, "/silver/tokens/"+contractID))
	if err != nil {
		return nil, err
	}

	var resp Token
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("gateway: parsing silver token: %w", err)
	}

	c.cache.Set(cacheKey, &resp, TTLImmutable)
	return &resp, nil
}

// GetSilverLedgerFull returns a composite ledger response with header, transactions, operations,
// fees, and Soroban stats in a single call. Replaces the 6-call fan-out on the ledger detail page.
func (c *Client) GetSilverLedgerFull(ctx context.Context, network string, sequence int64) (*LedgerFullResponse, error) {
	cacheKey := fmt.Sprintf("%s:silver_ledger_full:%d", network, sequence)
	if v, ok := c.cache.Get(cacheKey); ok {
		return v.(*LedgerFullResponse), nil
	}

	body, err := c.doRequest(ctx, http.MethodGet, c.buildURL(network, fmt.Sprintf("/silver/ledger/%d/full", sequence)))
	if err != nil {
		return nil, err
	}

	var resp LedgerFullResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("gateway: parsing silver ledger full: %w", err)
	}

	// A past ledger is immutable, so we can cache it aggressively.
	c.cache.Set(cacheKey, &resp, TTLImmutable)
	return &resp, nil
}

// GetSilverRecentLedgers returns the most recent ledgers from a single serving-backed endpoint.
// This replaces the bronze/stats + bronze/ledgers multi-call pattern.
func (c *Client) GetSilverRecentLedgers(ctx context.Context, network string, limit int) (*RecentLedgersResponse, error) {
	cacheKey := fmt.Sprintf("%s:silver_ledgers_recent:%d", network, limit)
	if v, ok := c.cache.Get(cacheKey); ok {
		return v.(*RecentLedgersResponse), nil
	}

	params := url.Values{
		"limit": {fmt.Sprintf("%d", limit)},
	}
	body, err := c.doRequest(ctx, http.MethodGet, c.buildURL(network, "/silver/ledgers/recent")+"?"+params.Encode())
	if err != nil {
		return nil, err
	}

	var resp RecentLedgersResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("gateway: parsing silver recent ledgers: %w", err)
	}

	c.cache.Set(cacheKey, &resp, TTLRecentList)
	return &resp, nil
}

// GetSilverRecentTransactions returns the most recent transactions with decoded summaries
// from a single serving-backed endpoint. This replaces the bronze/stats + bronze/transactions
// + silver/tx/batch/decoded multi-call pattern.
func (c *Client) GetSilverRecentTransactions(ctx context.Context, network string, limit int) (*RecentTransactionsResponse, error) {
	cacheKey := fmt.Sprintf("%s:silver_txs_recent:%d", network, limit)
	if v, ok := c.cache.Get(cacheKey); ok {
		return v.(*RecentTransactionsResponse), nil
	}

	params := url.Values{
		"limit": {fmt.Sprintf("%d", limit)},
	}
	body, err := c.doRequest(ctx, http.MethodGet, c.buildURL(network, "/silver/transactions/recent")+"?"+params.Encode())
	if err != nil {
		return nil, err
	}

	var resp RecentTransactionsResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("gateway: parsing silver recent transactions: %w", err)
	}

	c.cache.Set(cacheKey, &resp, TTLRecentList)
	return &resp, nil
}

// GetTopContracts returns the most active contracts.
func (c *Client) GetTopContracts(ctx context.Context, network string, limit int) ([]Contract, error) {
	cacheKey := fmt.Sprintf("%s:contracts_top:%d", network, limit)
	if v, ok := c.cache.Get(cacheKey); ok {
		return v.([]Contract), nil
	}

	params := url.Values{
		"limit": {fmt.Sprintf("%d", limit)},
	}
	body, err := c.doRequest(ctx, http.MethodGet, c.buildURL(network, "/silver/contracts/top")+"?"+params.Encode())
	if err != nil {
		return nil, err
	}

	var resp ContractsResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("gateway: parsing contracts: %w", err)
	}

	c.cache.Set(cacheKey, resp.Contracts, TTLContracts)
	return resp.Contracts, nil
}

// GetPayments returns recent payments.
func (c *Client) GetPayments(ctx context.Context, network string, limit int) ([]Payment, error) {
	cacheKey := fmt.Sprintf("%s:payments:%d", network, limit)
	if v, ok := c.cache.Get(cacheKey); ok {
		return v.([]Payment), nil
	}

	params := url.Values{
		"limit": {fmt.Sprintf("%d", limit)},
	}
	body, err := c.doRequest(ctx, http.MethodGet, c.buildURL(network, "/silver/payments")+"?"+params.Encode())
	if err != nil {
		return nil, err
	}

	var resp PaymentsResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("gateway: parsing payments: %w", err)
	}

	c.cache.Set(cacheKey, resp.Payments, TTLRecentList)
	return resp.Payments, nil
}

// --- Phase 2: Transaction Detail ---

// GetTransactionFull returns the full decoded transaction with operations, events, and call graph.
func (c *Client) GetTransactionFull(ctx context.Context, network string, hash string) (*TxFull, error) {
	cacheKey := network + ":tx_full:" + hash
	if v, ok := c.cache.Get(cacheKey); ok {
		return v.(*TxFull), nil
	}

	body, err := c.doRequest(ctx, http.MethodGet, c.buildURL(network, "/silver/tx/"+hash+"/full"))
	if err != nil {
		return nil, err
	}

	var result TxFull
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("gateway: parsing tx full: %w", err)
	}

	c.cache.Set(cacheKey, &result, TTLImmutable)
	return &result, nil
}

// GetTransactionDiffs returns balance changes and state changes for a transaction.
func (c *Client) GetTransactionDiffs(ctx context.Context, network string, hash string) (*TxDiffs, error) {
	cacheKey := network + ":tx_diffs:" + hash
	if v, ok := c.cache.Get(cacheKey); ok {
		return v.(*TxDiffs), nil
	}

	body, err := c.doRequest(ctx, http.MethodGet, c.buildURL(network, "/silver/tx/"+hash+"/diffs"))
	if err != nil {
		return nil, err
	}

	var result TxDiffs
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("gateway: parsing tx diffs: %w", err)
	}

	c.cache.Set(cacheKey, &result, TTLImmutable)
	return &result, nil
}

// GetOperations returns operations for a ledger range.
func (c *Client) GetOperations(ctx context.Context, network string, start, end int64, limit int) ([]Operation, error) {
	cacheKey := fmt.Sprintf("%s:ops:%d:%d:%d", network, start, end, limit)
	if v, ok := c.cache.Get(cacheKey); ok {
		return v.([]Operation), nil
	}

	params := url.Values{
		"start_ledger": {fmt.Sprintf("%d", start)},
		"end_ledger":   {fmt.Sprintf("%d", end)},
		"limit":        {fmt.Sprintf("%d", limit)},
	}
	body, err := c.doRequest(ctx, http.MethodGet, c.buildURL(network, "/silver/operations/enriched")+"?"+params.Encode())
	if err != nil {
		return nil, err
	}

	var resp OperationsResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("gateway: parsing operations: %w", err)
	}

	ttl := TTLRecentList
	if start == end {
		ttl = TTLImmutable
	}
	c.cache.Set(cacheKey, resp.Operations, ttl)
	return resp.Operations, nil
}

// --- Phase 3: Account ---

// GetAccountOverview returns comprehensive account info with recent activity.
func (c *Client) GetAccountOverview(ctx context.Context, network string, accountID string) (*AccountOverview, error) {
	cacheKey := network + ":account:" + accountID
	if v, ok := c.cache.Get(cacheKey); ok {
		return v.(*AccountOverview), nil
	}

	params := url.Values{"account_id": {accountID}}
	body, err := c.doRequest(ctx, http.MethodGet, c.buildURL(network, "/silver/explorer/account")+"?"+params.Encode())
	if err != nil {
		return nil, err
	}

	var result AccountOverview
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("gateway: parsing account overview: %w", err)
	}

	c.cache.Set(cacheKey, &result, TTLAccount)
	return &result, nil
}

// GetAccountBalances returns all balances for an account.
func (c *Client) GetAccountBalances(ctx context.Context, network string, accountID string) (*AccountBalances, error) {
	cacheKey := network + ":account_bal:" + accountID
	if v, ok := c.cache.Get(cacheKey); ok {
		return v.(*AccountBalances), nil
	}

	body, err := c.doRequest(ctx, http.MethodGet, c.buildURL(network, "/silver/accounts/"+accountID+"/balances"))
	if err != nil {
		return nil, err
	}

	var result AccountBalances
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("gateway: parsing account balances: %w", err)
	}

	c.cache.Set(cacheKey, &result, TTLAccount)
	return &result, nil
}

// GetAccountSigners returns signers and thresholds for an account.
func (c *Client) GetAccountSigners(ctx context.Context, network string, accountID string) (*AccountSignersResp, error) {
	cacheKey := network + ":account_sig:" + accountID
	if v, ok := c.cache.Get(cacheKey); ok {
		return v.(*AccountSignersResp), nil
	}

	params := url.Values{"account_id": {accountID}}
	body, err := c.doRequest(ctx, http.MethodGet, c.buildURL(network, "/silver/accounts/signers")+"?"+params.Encode())
	if err != nil {
		return nil, err
	}

	var result AccountSignersResp
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("gateway: parsing account signers: %w", err)
	}

	c.cache.Set(cacheKey, &result, TTLAccount)
	return &result, nil
}

// --- Phase 4: Contract Analytics ---

// GetContractAnalytics returns comprehensive contract analytics.
func (c *Client) GetContractAnalytics(ctx context.Context, network string, contractID string) (*ContractAnalytics, error) {
	cacheKey := network + ":contract_analytics:" + contractID
	if v, ok := c.cache.Get(cacheKey); ok {
		return v.(*ContractAnalytics), nil
	}

	body, err := c.doRequest(ctx, http.MethodGet, c.buildURL(network, "/silver/contracts/"+contractID+"/analytics"))
	if err != nil {
		return nil, err
	}

	var result ContractAnalytics
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("gateway: parsing contract analytics: %w", err)
	}

	c.cache.Set(cacheKey, &result, TTLContracts)
	return &result, nil
}

// GetContractMetadata returns static metadata for a contract.
func (c *Client) GetContractMetadata(ctx context.Context, network string, contractID string) (*ContractMetadata, error) {
	cacheKey := network + ":contract_metadata:" + contractID
	if v, ok := c.cache.Get(cacheKey); ok {
		return v.(*ContractMetadata), nil
	}

	body, err := c.doRequest(ctx, http.MethodGet, c.buildURL(network, "/silver/contracts/"+contractID+"/metadata"))
	if err != nil {
		return nil, err
	}

	var result ContractMetadata
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("gateway: parsing contract metadata: %w", err)
	}

	c.cache.Set(cacheKey, &result, TTLContracts)
	return &result, nil
}

// GetContractStorage returns a preview of contract storage entries.
func (c *Client) GetContractStorage(ctx context.Context, network string, contractID string, limit int) (*ContractStorageResponse, error) {
	cacheKey := fmt.Sprintf("%s:contract_storage:%s:%d", network, contractID, limit)
	if v, ok := c.cache.Get(cacheKey); ok {
		return v.(*ContractStorageResponse), nil
	}

	params := url.Values{"limit": {fmt.Sprintf("%d", limit)}}
	body, err := c.doRequest(ctx, http.MethodGet, c.buildURL(network, "/silver/contracts/"+contractID+"/storage")+"?"+params.Encode())
	if err != nil {
		return nil, err
	}

	var result ContractStorageResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("gateway: parsing contract storage: %w", err)
	}

	c.cache.Set(cacheKey, &result, TTLContracts)
	return &result, nil
}

// GetContractRecentCalls returns recent calls for a contract.
func (c *Client) GetContractRecentCalls(ctx context.Context, network string, contractID string, limit int) ([]ContractRecentCall, error) {
	cacheKey := fmt.Sprintf("%s:contract_calls:%s:%d", network, contractID, limit)
	if v, ok := c.cache.Get(cacheKey); ok {
		return v.([]ContractRecentCall), nil
	}

	params := url.Values{"limit": {fmt.Sprintf("%d", limit)}}
	body, err := c.doRequest(ctx, http.MethodGet, c.buildURL(network, "/silver/contracts/"+contractID+"/recent-calls")+"?"+params.Encode())
	if err != nil {
		return nil, err
	}

	var result []ContractRecentCall
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("gateway: parsing contract calls: %w", err)
	}

	c.cache.Set(cacheKey, result, TTLRecentList)
	return result, nil
}

// --- Phase 4: Assets ---

// GetAssets returns a paginated list of assets sorted by holder count.
func (c *Client) GetAssets(ctx context.Context, network string, limit int, sortBy, order string) (*AssetsResponse, error) {
	cacheKey := fmt.Sprintf("%s:assets:%d:%s:%s", network, limit, sortBy, order)
	if v, ok := c.cache.Get(cacheKey); ok {
		return v.(*AssetsResponse), nil
	}

	params := url.Values{
		"limit":   {fmt.Sprintf("%d", limit)},
		"sort_by": {sortBy},
		"order":   {order},
	}
	body, err := c.doRequest(ctx, http.MethodGet, c.buildURL(network, "/silver/assets")+"?"+params.Encode())
	if err != nil {
		return nil, err
	}

	var result AssetsResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("gateway: parsing assets: %w", err)
	}

	c.cache.Set(cacheKey, &result, TTLContracts)
	return &result, nil
}

// --- Phase 5: Events + Search ---

// GetEvents returns the unified CAP-67 event stream.
func (c *Client) GetEvents(ctx context.Context, network string, limit int) ([]UnifiedEvent, error) {
	cacheKey := fmt.Sprintf("%s:events:%d", network, limit)
	if v, ok := c.cache.Get(cacheKey); ok {
		return v.([]UnifiedEvent), nil
	}

	params := url.Values{"limit": {fmt.Sprintf("%d", limit)}}
	body, err := c.doRequest(ctx, http.MethodGet, c.buildURL(network, "/silver/events")+"?"+params.Encode())
	if err != nil {
		return nil, err
	}

	var resp EventsResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("gateway: parsing events: %w", err)
	}

	c.cache.Set(cacheKey, resp.Events, TTLRecentList)
	return resp.Events, nil
}

// GetTransfers returns recent token transfers.
func (c *Client) GetTransfers(ctx context.Context, network string, limit int) ([]TransferEvent, error) {
	cacheKey := fmt.Sprintf("%s:transfers:%d", network, limit)
	if v, ok := c.cache.Get(cacheKey); ok {
		return v.([]TransferEvent), nil
	}

	params := url.Values{"limit": {fmt.Sprintf("%d", limit)}}
	body, err := c.doRequest(ctx, http.MethodGet, c.buildURL(network, "/silver/transfers")+"?"+params.Encode())
	if err != nil {
		return nil, err
	}

	var resp TransfersResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("gateway: parsing transfers: %w", err)
	}

	c.cache.Set(cacheKey, resp.Transfers, TTLRecentList)
	return resp.Transfers, nil
}

// Search performs a unified search across accounts, contracts, transactions, ledgers, and assets.
func (c *Client) Search(ctx context.Context, network string, query string) (*SearchResults, error) {
	params := url.Values{"q": {query}}
	body, err := c.doRequest(ctx, http.MethodGet, c.buildURL(network, "/silver/search")+"?"+params.Encode())
	if err != nil {
		return nil, err
	}

	var result SearchResults
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("gateway: parsing search: %w", err)
	}

	return &result, nil
}

// --- Smart Wallet Detection ---

// GetSmartWalletInfo returns smart wallet detection info for a contract.
func (c *Client) GetSmartWalletInfo(ctx context.Context, network string, contractID string) (*SmartWalletInfo, error) {
	cacheKey := network + ":smart_wallet:" + contractID
	if v, ok := c.cache.Get(cacheKey); ok {
		return v.(*SmartWalletInfo), nil
	}

	body, err := c.doRequest(ctx, http.MethodGet, c.buildURL(network, "/silver/smart-wallet/"+contractID))
	if err != nil {
		return nil, err
	}

	var result SmartWalletInfo
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("gateway: parsing smart wallet info: %w", err)
	}

	c.cache.Set(cacheKey, &result, TTLContracts)
	return &result, nil
}

// --- Fee & Soroban Stats ---

// GetFeeStats returns fee percentiles and surge detection for a period.
func (c *Client) GetFeeStats(ctx context.Context, network string, period string) (*FeeStats, error) {
	cacheKey := fmt.Sprintf("%s:fee_stats:%s", network, period)
	if v, ok := c.cache.Get(cacheKey); ok {
		return v.(*FeeStats), nil
	}

	params := url.Values{"period": {period}}
	body, err := c.doRequest(ctx, http.MethodGet, c.buildURL(network, "/silver/stats/fees")+"?"+params.Encode())
	if err != nil {
		return nil, err
	}

	var result FeeStats
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("gateway: parsing fee stats: %w", err)
	}

	c.cache.Set(cacheKey, &result, TTLNetworkStats)
	return &result, nil
}

// GetSorobanStats returns Soroban contract, execution, and state statistics.
func (c *Client) GetSorobanStats(ctx context.Context, network string) (*SorobanStats, error) {
	cacheKey := network + ":soroban_stats"
	if v, ok := c.cache.Get(cacheKey); ok {
		return v.(*SorobanStats), nil
	}

	body, err := c.doRequest(ctx, http.MethodGet, c.buildURL(network, "/silver/stats/soroban"))
	if err != nil {
		return nil, err
	}

	var result SorobanStats
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("gateway: parsing soroban stats: %w", err)
	}

	c.cache.Set(cacheKey, &result, TTLNetworkStats)
	return &result, nil
}

// --- Semantic Contracts ---

// GetSemanticContracts returns the contract registry with human-readable names and types.
func (c *Client) GetSemanticContracts(ctx context.Context, network string, limit int) ([]SemanticContract, error) {
	cacheKey := fmt.Sprintf("%s:semantic_contracts:%d", network, limit)
	if v, ok := c.cache.Get(cacheKey); ok {
		return v.([]SemanticContract), nil
	}

	params := url.Values{"limit": {fmt.Sprintf("%d", limit)}}
	body, err := c.doRequest(ctx, http.MethodGet, c.buildURL(network, "/semantic/contracts")+"?"+params.Encode())
	if err != nil {
		return nil, err
	}

	var resp SemanticContractsResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("gateway: parsing semantic contracts: %w", err)
	}

	c.cache.Set(cacheKey, resp.Contracts, TTLContracts)
	return resp.Contracts, nil
}

// --- Contract Interface ---

// GetContractInterface returns the detected interface and observed functions for a contract.
func (c *Client) GetContractInterface(ctx context.Context, network string, contractID string) (*ContractInterface, error) {
	cacheKey := network + ":contract_iface:" + contractID
	if v, ok := c.cache.Get(cacheKey); ok {
		return v.(*ContractInterface), nil
	}

	body, err := c.doRequest(ctx, http.MethodGet, c.buildURL(network, "/silver/contracts/"+contractID+"/interface"))
	if err != nil {
		return nil, err
	}

	var result ContractInterface
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("gateway: parsing contract interface: %w", err)
	}

	c.cache.Set(cacheKey, &result, TTLContracts)
	return &result, nil
}

// --- Account Activity & Offers ---

// GetAccountActivity returns recent activity (payments, contract calls) for an account.
func (c *Client) GetAccountActivity(ctx context.Context, network string, accountID string, limit int) (*AccountActivityResponse, error) {
	cacheKey := fmt.Sprintf("%s:account_activity:%s:%d", network, accountID, limit)
	if v, ok := c.cache.Get(cacheKey); ok {
		return v.(*AccountActivityResponse), nil
	}

	params := url.Values{"limit": {fmt.Sprintf("%d", limit)}}
	body, err := c.doRequest(ctx, http.MethodGet, c.buildURL(network, "/silver/accounts/"+accountID+"/activity")+"?"+params.Encode())
	if err != nil {
		return nil, err
	}

	var result AccountActivityResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("gateway: parsing account activity: %w", err)
	}

	c.cache.Set(cacheKey, &result, TTLAccount)
	return &result, nil
}

// GetAccountOffers returns active DEX offers for an account.
func (c *Client) GetAccountOffers(ctx context.Context, network string, accountID string) (*AccountOffersResponse, error) {
	cacheKey := network + ":account_offers:" + accountID
	if v, ok := c.cache.Get(cacheKey); ok {
		return v.(*AccountOffersResponse), nil
	}

	body, err := c.doRequest(ctx, http.MethodGet, c.buildURL(network, "/silver/accounts/"+accountID+"/offers"))
	if err != nil {
		return nil, err
	}

	var result AccountOffersResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("gateway: parsing account offers: %w", err)
	}

	c.cache.Set(cacheKey, &result, TTLAccount)
	return &result, nil
}

// --- Transaction Summaries ---

// GetTransactionSummaries returns decoded summaries for a batch of transactions (by hashes or ledger).
func (c *Client) GetTransactionSummaries(ctx context.Context, network string, hashes []string, ledger int64) (*TransactionSummariesResponse, error) {
	params := url.Values{}
	var cacheKey string

	if ledger > 0 {
		params.Set("ledger", fmt.Sprintf("%d", ledger))
		cacheKey = fmt.Sprintf("%s:tx_summaries:ledger:%d", network, ledger)
	} else if len(hashes) > 0 {
		for _, h := range hashes {
			params.Add("hashes", h)
		}
		cacheKey = fmt.Sprintf("%s:tx_summaries:%s", network, joinHashes(hashes))
	}

	if v, ok := c.cache.Get(cacheKey); ok {
		return v.(*TransactionSummariesResponse), nil
	}

	body, err := c.doRequest(ctx, http.MethodGet, c.buildURL(network, "/silver/transactions/summaries")+"?"+params.Encode())
	if err != nil {
		return nil, err
	}

	var result TransactionSummariesResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("gateway: parsing transaction summaries: %w", err)
	}

	ttl := TTLRecentList
	if ledger > 0 {
		ttl = TTLImmutable // summaries for a specific ledger are immutable
	}
	c.cache.Set(cacheKey, &result, ttl)
	return &result, nil
}

// --- Generic Events ---

// GetGenericEvents returns raw contract events with topic filtering.
// Supports filtering by contractID and/or txHash.
func (c *Client) GetGenericEvents(ctx context.Context, network string, contractID string, txHash string, limit int) (*GenericEventsResponse, error) {
	cacheKey := fmt.Sprintf("%s:generic_events:%s:%s:%d", network, contractID, txHash, limit)
	if v, ok := c.cache.Get(cacheKey); ok {
		return v.(*GenericEventsResponse), nil
	}

	params := url.Values{"limit": {fmt.Sprintf("%d", limit)}}
	if contractID != "" {
		params.Set("contract_id", contractID)
	}
	if txHash != "" {
		params.Set("tx_hash", txHash)
	}

	body, err := c.doRequest(ctx, http.MethodGet, c.buildURL(network, "/silver/events/generic")+"?"+params.Encode())
	if err != nil {
		return nil, err
	}

	var result GenericEventsResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("gateway: parsing generic events: %w", err)
	}

	c.cache.Set(cacheKey, &result, TTLRecentList)
	return &result, nil
}

// --- Batch Decoded Transactions ---

// GetBatchDecodedTransactions returns decoded transaction data with human-readable summaries.
// Supports two modes: by hash list (up to 25) or by ledger number.
func (c *Client) GetBatchDecodedTransactions(ctx context.Context, network string, hashes []string, ledger int64, limit int) (*BatchDecodedResponse, error) {
	params := url.Values{}
	var cacheKey string

	if ledger > 0 {
		params.Set("ledger", fmt.Sprintf("%d", ledger))
		if limit > 0 {
			params.Set("limit", fmt.Sprintf("%d", limit))
		}
		cacheKey = fmt.Sprintf("%s:batch_decoded:ledger:%d:%d", network, ledger, limit)
	} else if len(hashes) > 0 {
		params.Set("hashes", joinHashes(hashes))
		cacheKey = fmt.Sprintf("%s:batch_decoded:hashes:%s", network, joinHashes(hashes))
	}

	if v, ok := c.cache.Get(cacheKey); ok {
		return v.(*BatchDecodedResponse), nil
	}

	body, err := c.doRequest(ctx, http.MethodGet, c.buildURL(network, "/silver/tx/batch/decoded")+"?"+params.Encode())
	if err != nil {
		return nil, err
	}

	var result BatchDecodedResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("gateway: parsing batch decoded txs: %w", err)
	}

	ttl := TTLRecentList
	if ledger > 0 {
		ttl = TTLImmutable
	}
	c.cache.Set(cacheKey, &result, ttl)
	return &result, nil
}

// joinHashes joins a slice of hashes with commas.
func joinHashes(hashes []string) string {
	result := ""
	for i, h := range hashes {
		if i > 0 {
			result += ","
		}
		result += h
	}
	return result
}

// --- Per-Ledger Stats ---

// GetLedgerFees returns fee distribution for a specific ledger.
func (c *Client) GetLedgerFees(ctx context.Context, network string, sequence int64) (*LedgerFees, error) {
	cacheKey := fmt.Sprintf("%s:ledger_fees:%d", network, sequence)
	if v, ok := c.cache.Get(cacheKey); ok {
		return v.(*LedgerFees), nil
	}

	body, err := c.doRequest(ctx, http.MethodGet, c.buildURL(network, fmt.Sprintf("/silver/ledgers/%d/fees", sequence)))
	if err != nil {
		return nil, err
	}

	var result LedgerFees
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("gateway: parsing ledger fees: %w", err)
	}

	c.cache.Set(cacheKey, &result, TTLImmutable)
	return &result, nil
}

// GetLedgerSoroban returns Soroban resource usage for a specific ledger.
func (c *Client) GetLedgerSoroban(ctx context.Context, network string, sequence int64) (*LedgerSoroban, error) {
	cacheKey := fmt.Sprintf("%s:ledger_soroban:%d", network, sequence)
	if v, ok := c.cache.Get(cacheKey); ok {
		return v.(*LedgerSoroban), nil
	}

	body, err := c.doRequest(ctx, http.MethodGet, c.buildURL(network, fmt.Sprintf("/silver/ledgers/%d/soroban", sequence)))
	if err != nil {
		return nil, err
	}

	var result LedgerSoroban
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("gateway: parsing ledger soroban: %w", err)
	}

	c.cache.Set(cacheKey, &result, TTLImmutable)
	return &result, nil
}

// --- Transaction Effects ---

// GetTransactionEffects returns effects for a transaction.
func (c *Client) GetTransactionEffects(ctx context.Context, network string, hash string) (*TransactionEffectsResponse, error) {
	cacheKey := network + ":tx_effects:" + hash
	if v, ok := c.cache.Get(cacheKey); ok {
		return v.(*TransactionEffectsResponse), nil
	}

	body, err := c.doRequest(ctx, http.MethodGet, c.buildURL(network, "/silver/effects/transaction/"+hash))
	if err != nil {
		return nil, err
	}

	var result TransactionEffectsResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("gateway: parsing transaction effects: %w", err)
	}

	c.cache.Set(cacheKey, &result, TTLImmutable)
	return &result, nil
}

// --- Explorer Events ---

// GetExplorerEvents returns enriched explorer events with classification and pagination.
func (c *Client) GetExplorerEvents(ctx context.Context, network string, p ExplorerEventsParams) (*ExplorerEventsResponse, error) {
	params := url.Values{}
	if p.Type != "" {
		params.Set("type", p.Type)
	}
	if p.Tab != "" {
		params.Set("tab", p.Tab)
	}
	if p.ContractID != "" {
		params.Set("contract_id", p.ContractID)
	}
	if p.ContractName != "" {
		params.Set("contract_name", p.ContractName)
	}
	if p.TxHash != "" {
		params.Set("tx_hash", p.TxHash)
	}
	if p.TopicMatch != "" {
		params.Set("topic_match", p.TopicMatch)
	}
	if p.StartLedger > 0 {
		params.Set("start_ledger", fmt.Sprintf("%d", p.StartLedger))
	}
	if p.EndLedger > 0 {
		params.Set("end_ledger", fmt.Sprintf("%d", p.EndLedger))
	}
	if p.Limit > 0 {
		params.Set("limit", fmt.Sprintf("%d", p.Limit))
	}
	if p.Cursor != "" {
		params.Set("cursor", p.Cursor)
	}
	if p.Order != "" {
		params.Set("order", p.Order)
	}

	cacheKey := fmt.Sprintf("%s:explorer_events:%s", network, params.Encode())
	if v, ok := c.cache.Get(cacheKey); ok {
		return v.(*ExplorerEventsResponse), nil
	}

	body, err := c.doRequest(ctx, http.MethodGet, c.buildURL(network, "/explorer/events")+"?"+params.Encode())
	if err != nil {
		return nil, err
	}

	var result ExplorerEventsResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("gateway: parsing explorer events: %w", err)
	}

	c.cache.Set(cacheKey, &result, TTLRecentList)
	return &result, nil
}
