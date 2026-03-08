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
