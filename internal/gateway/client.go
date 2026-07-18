package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"sync"
	"time"
)

// Cache TTLs per data type.
const (
	TTLNetworkStats       = 10 * time.Second
	TTLBronzeNetworkStats = 3 * time.Second // tracks ledger close (~5s, may decrease)
	TTLRecentList         = 10 * time.Second
	TTLHomeSummary        = 15 * time.Second
	TTLHomeSummaryFailure = 10 * time.Second
	TTLLedgerFeedFailure  = 30 * time.Second
	TTLTxReceiptFailure   = 2 * time.Minute
	TTLAccountFailure     = 15 * time.Second
	TTLImmutable          = 24 * time.Hour
	TTLAccount            = 30 * time.Second
	TTLContracts          = 2 * time.Minute
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
	inflight   *inflightGroup
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
		inflight: &inflightGroup{
			calls: make(map[string]*inflightCall),
		},
	}
}

// Stop cleans up the client's background resources.
func (c *Client) Stop() {
	if c != nil && c.cache != nil {
		c.cache.Stop()
	}
}

type inflightCall struct {
	done chan struct{}
	val  any
	err  error
}

type inflightGroup struct {
	mu    sync.Mutex
	calls map[string]*inflightCall
}

func (g *inflightGroup) Do(key string, fn func() (any, error)) (any, error, bool) {
	return g.DoContext(context.Background(), key, fn)
}

func (g *inflightGroup) DoContext(ctx context.Context, key string, fn func() (any, error)) (any, error, bool) {
	if ctx == nil {
		ctx = context.Background()
	}
	g.mu.Lock()
	if call, ok := g.calls[key]; ok {
		g.mu.Unlock()
		select {
		case <-call.done:
			return call.val, call.err, true
		case <-ctx.Done():
			return nil, ctx.Err(), true
		}
	}
	call := &inflightCall{done: make(chan struct{})}
	g.calls[key] = call
	g.mu.Unlock()

	call.val, call.err = fn()
	close(call.done)

	g.mu.Lock()
	delete(g.calls, key)
	g.mu.Unlock()
	return call.val, call.err, false
}

// buildURL constructs a gateway API URL for the given network.
// Pattern: {baseURL}/lake/v1/{network}/api/v1{path}
func (c *Client) buildURL(network, path string) string {
	return fmt.Sprintf("%s/lake/v1/%s/api/v1%s", c.baseURL, network, path)
}

// doRequest executes an HTTP request with authentication.
func (c *Client) doRequest(ctx context.Context, method, rawURL string) ([]byte, error) {
	start := time.Now()
	endpoint := gatewayLogEndpoint(rawURL)
	req, err := http.NewRequestWithContext(ctx, method, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("gateway: creating request: %w", err)
	}
	req.Header.Set("Authorization", "Api-Key "+c.apiKey)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Warn("gateway request failed", "method", method, "endpoint", endpoint, "duration", time.Since(start), "error", err)
		return nil, fmt.Errorf("gateway: request failed %s %s after %s: %w", method, endpoint, time.Since(start), err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.logger.Warn("gateway response read failed", "method", method, "endpoint", endpoint, "status", resp.StatusCode, "duration", time.Since(start), "error", err)
		return nil, fmt.Errorf("gateway: reading response %s %s after %s: %w", method, endpoint, time.Since(start), err)
	}

	if resp.StatusCode >= 400 {
		c.logger.Warn("gateway returned error", "method", method, "endpoint", endpoint, "status", resp.StatusCode, "duration", time.Since(start), "body", string(body))
		return nil, &APIError{
			StatusCode: resp.StatusCode,
			Message:    string(body),
		}
	}

	if elapsed := time.Since(start); elapsed > 2*time.Second {
		c.logger.Warn("slow gateway request", "method", method, "endpoint", endpoint, "status", resp.StatusCode, "duration", elapsed, "bytes", len(body))
	} else {
		c.logger.Debug("gateway request", "method", method, "endpoint", endpoint, "status", resp.StatusCode, "duration", elapsed, "bytes", len(body))
	}

	return body, nil
}

func gatewayLogEndpoint(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	if u.RawQuery != "" {
		return u.Path + "?" + u.RawQuery
	}
	return u.Path
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
	v, err, _ := c.inflight.DoContext(ctx, cacheKey, func() (any, error) {
		if v, ok := c.cache.Get(cacheKey); ok {
			return v, nil
		}

		body, err := c.doRequest(ctx, http.MethodGet, c.buildURL(network, fmt.Sprintf("/silver/ledger/%d/full", sequence)))
		if err != nil {
			return nil, err
		}

		var resp LedgerFullResponse
		if err := json.Unmarshal(body, &resp); err != nil {
			return nil, fmt.Errorf("gateway: parsing silver ledger full: %w", err)
		}
		if resp.LedgerSequence != sequence || resp.Ledger.Sequence != sequence {
			return nil, fmt.Errorf(
				"gateway: silver ledger full sequence mismatch: requested %d, response %d, ledger %d",
				sequence,
				resp.LedgerSequence,
				resp.Ledger.Sequence,
			)
		}

		// Only complete responses are immutable. Partial responses are transient
		// products of the backend query budget and must be retried.
		// HasCompleteTransactions already treats partial responses as incomplete.
		if resp.HasCompleteTransactions() {
			c.cache.Set(cacheKey, &resp, TTLImmutable)
		} else {
			c.logger.Debug(
				"skipping incomplete silver ledger full cache",
				"sequence", sequence,
				"partial", resp.Partial,
				"transactions", len(resp.Transactions),
				"expected_transactions", min(resp.Ledger.TransactionCount, LedgerFullTransactionLimit),
				"warnings", resp.Warnings,
			)
		}
		return &resp, nil
	})
	if err != nil {
		return nil, err
	}
	return v.(*LedgerFullResponse), nil
}

// GetSilverRecentLedgers returns the most recent ledgers from a single serving-backed endpoint.
// This replaces the bronze/stats + bronze/ledgers multi-call pattern.
func (c *Client) GetSilverRecentLedgers(ctx context.Context, network string, limit int) (*RecentLedgersResponse, error) {
	cacheKey := fmt.Sprintf("%s:silver_ledgers_recent:%d", network, limit)
	if v, ok := c.cache.Get(cacheKey); ok {
		return v.(*RecentLedgersResponse), nil
	}
	v, err, _ := c.inflight.DoContext(ctx, cacheKey, func() (any, error) {
		if v, ok := c.cache.Get(cacheKey); ok {
			return v, nil
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
	})
	if err != nil {
		return nil, err
	}
	return v.(*RecentLedgersResponse), nil
}

// GetSilverRecentTransactions returns the most recent transactions with decoded summaries
// from a single serving-backed endpoint. This replaces the bronze/stats + bronze/transactions
// + silver/tx/batch/decoded multi-call pattern.
func (c *Client) GetSilverRecentTransactions(ctx context.Context, network string, limit int) (*RecentTransactionsResponse, error) {
	cacheKey := fmt.Sprintf("%s:silver_txs_recent:%d", network, limit)
	if v, ok := c.cache.Get(cacheKey); ok {
		return v.(*RecentTransactionsResponse), nil
	}
	v, err, _ := c.inflight.DoContext(ctx, cacheKey, func() (any, error) {
		if v, ok := c.cache.Get(cacheKey); ok {
			return v, nil
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
	})
	if err != nil {
		return nil, err
	}
	return v.(*RecentTransactionsResponse), nil
}

// GetSilverLedgerSummary returns the older compact per-ledger snapshot used by the ledger-first v2 page.
func (c *Client) GetSilverLedgerSummary(ctx context.Context, network string, sequence int64) (*LedgerSummary, error) {
	cacheKey := fmt.Sprintf("%s:silver_ledger_summary:%d", network, sequence)
	if v, ok := c.cache.Get(cacheKey); ok {
		return v.(*LedgerSummary), nil
	}
	v, err, _ := c.inflight.DoContext(ctx, cacheKey, func() (any, error) {
		if v, ok := c.cache.Get(cacheKey); ok {
			return v, nil
		}

		body, err := c.doRequest(ctx, http.MethodGet, c.buildURL(network, fmt.Sprintf("/silver/ledger/%d/summary", sequence)))
		if err != nil {
			return nil, err
		}

		var resp LedgerSummary
		if err := json.Unmarshal(body, &resp); err != nil {
			return nil, fmt.Errorf("gateway: parsing silver ledger summary: %w", err)
		}
		if ledgerSummaryLooksEmpty(resp) {
			var feedResp LedgerFeedSummary
			if err := json.Unmarshal(body, &feedResp); err == nil && !ledgerFeedSummaryLooksEmpty(feedResp) {
				resp = ledgerSummaryFromFeedSummary(feedResp)
			}
		}

		c.cache.Set(cacheKey, &resp, TTLImmutable)
		return &resp, nil
	})
	if err != nil {
		return nil, err
	}
	return v.(*LedgerSummary), nil
}

// ledgerFeedNegativeEntry is cached briefly on per-ledger summary failures to
// avoid repeated 30s waits for the same recent ledgers while the serving
// endpoint is unhealthy or lagging.
type ledgerFeedNegativeEntry struct{ err error }

// GetSilverLedgerFeedSummary returns the current rich per-ledger summary used by the /v2/home feed.
func (c *Client) GetSilverLedgerFeedSummary(ctx context.Context, network string, sequence int64) (*LedgerFeedSummary, error) {
	cacheKey := fmt.Sprintf("%s:silver_ledger_feed_summary:%d", network, sequence)
	if v, ok := c.cache.Get(cacheKey); ok {
		if neg, isNeg := v.(*ledgerFeedNegativeEntry); isNeg {
			return nil, neg.err
		}
		return v.(*LedgerFeedSummary), nil
	}
	v, err, _ := c.inflight.DoContext(ctx, cacheKey, func() (any, error) {
		if v, ok := c.cache.Get(cacheKey); ok {
			if neg, isNeg := v.(*ledgerFeedNegativeEntry); isNeg {
				return nil, neg.err
			}
			return v, nil
		}

		body, err := c.doRequest(ctx, http.MethodGet, c.buildURL(network, fmt.Sprintf("/silver/ledger/%d/summary", sequence)))
		if err != nil {
			c.cache.Set(cacheKey, &ledgerFeedNegativeEntry{err: err}, TTLLedgerFeedFailure)
			return nil, err
		}

		var resp LedgerFeedSummary
		if err := json.Unmarshal(body, &resp); err != nil {
			wrapped := fmt.Errorf("gateway: parsing silver ledger feed summary: %w", err)
			c.cache.Set(cacheKey, &ledgerFeedNegativeEntry{err: wrapped}, TTLLedgerFeedFailure)
			return nil, wrapped
		}

		c.cache.Set(cacheKey, &resp, TTLImmutable)
		return &resp, nil
	})
	if err != nil {
		return nil, err
	}
	return v.(*LedgerFeedSummary), nil
}

func ledgerSummaryLooksEmpty(v LedgerSummary) bool {
	return v.TxCount == 0 &&
		v.TransactionCount == 0 &&
		v.Swaps == 0 &&
		v.Calls == 0 &&
		v.Agents == 0 &&
		v.InstructionPct == 0 &&
		v.ReadWritePct == 0 &&
		v.Classifications == nil &&
		v.Utilization == nil
}

func ledgerFeedSummaryLooksEmpty(v LedgerFeedSummary) bool {
	return v.Ledger.Sequence == 0 &&
		v.Totals.TransactionCount == 0 &&
		v.Totals.OperationCount == 0 &&
		v.ClassificationCounts.SwapTxCount == 0 &&
		v.ClassificationCounts.ContractCallTxCount == 0 &&
		v.ClassificationCounts.ClassicTxCount == 0 &&
		v.SorobanUtilization.InstructionsUsed == 0 &&
		v.SorobanUtilization.ReadWriteBytesUsed == 0 &&
		len(v.RepresentativeTransactions) == 0
}

func ledgerSummaryFromFeedSummary(v LedgerFeedSummary) LedgerSummary {
	return LedgerSummary{
		LedgerSequence:   v.Ledger.Sequence,
		TransactionCount: v.Totals.TransactionCount,
		Swaps:            v.ClassificationCounts.SwapTxCount,
		Calls:            v.ClassificationCounts.ContractCallTxCount,
		InstructionPct:   v.SorobanUtilization.InstructionPct,
		ReadWritePct:     v.SorobanUtilization.ReadWritePct,
		Classifications: &LedgerSummaryClassifications{
			Swaps: v.ClassificationCounts.SwapTxCount,
			Calls: v.ClassificationCounts.ContractCallTxCount,
		},
		Utilization: &LedgerSummaryUtilization{
			InstructionPct: v.SorobanUtilization.InstructionPct,
			ReadWritePct:   v.SorobanUtilization.ReadWritePct,
		},
	}
}

// homeSummaryNegativeEntry is cached briefly on home-summary failures so one
// slow gateway endpoint cannot stall every home-page/fragment request.
type homeSummaryNegativeEntry struct{ err error }

// GetHomeSummary returns the aggregated non-feed data used by /v2/home.
func (c *Client) GetHomeSummary(ctx context.Context, network string) (*HomeSummaryResponse, error) {
	cacheKey := fmt.Sprintf("%s:home_summary", network)
	if v, ok := c.cache.Get(cacheKey); ok {
		if neg, isNeg := v.(*homeSummaryNegativeEntry); isNeg {
			return nil, neg.err
		}
		return v.(*HomeSummaryResponse), nil
	}
	v, err, _ := c.inflight.DoContext(ctx, cacheKey, func() (any, error) {
		if v, ok := c.cache.Get(cacheKey); ok {
			if neg, isNeg := v.(*homeSummaryNegativeEntry); isNeg {
				return nil, neg.err
			}
			return v, nil
		}

		body, err := c.doRequest(ctx, http.MethodGet, c.buildURL(network, "/home/summary"))
		if err != nil {
			c.cache.Set(cacheKey, &homeSummaryNegativeEntry{err: err}, TTLHomeSummaryFailure)
			return nil, err
		}

		var resp HomeSummaryResponse
		if err := json.Unmarshal(body, &resp); err != nil {
			wrapped := fmt.Errorf("gateway: parsing home summary: %w", err)
			c.cache.Set(cacheKey, &homeSummaryNegativeEntry{err: wrapped}, TTLHomeSummaryFailure)
			return nil, wrapped
		}

		c.cache.Set(cacheKey, &resp, TTLHomeSummary)
		return &resp, nil
	})
	if err != nil {
		return nil, err
	}
	return v.(*HomeSummaryResponse), nil
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

// GetTransactionReceipt returns the consolidated transaction receipt.
type txReceiptNegativeEntry struct{ err error }

func (c *Client) GetTransactionReceipt(ctx context.Context, network string, hash string) (*TxReceipt, error) {
	cacheKey := network + ":tx_receipt:" + hash
	if v, ok := c.cache.Get(cacheKey); ok {
		if neg, isNeg := v.(*txReceiptNegativeEntry); isNeg {
			return nil, neg.err
		}
		return v.(*TxReceipt), nil
	}

	body, err := c.doRequest(ctx, http.MethodGet, c.buildURL(network, "/silver/tx/"+hash+"/receipt"))
	if err != nil {
		if apiErr, ok := err.(*APIError); ok && apiErr.StatusCode == http.StatusNotFound {
			c.cache.Set(cacheKey, &txReceiptNegativeEntry{err: err}, TTLTxReceiptFailure)
		}
		return nil, err
	}

	var result TxReceipt
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("gateway: parsing tx receipt: %w", err)
	}

	c.cache.Set(cacheKey, &result, TTLImmutable)
	return &result, nil
}

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

// GetSemanticTransaction returns the actor-centric semantic transaction view.
func (c *Client) GetSemanticTransaction(ctx context.Context, network, hash string, includeDeep bool) (*SemanticTransactionResponse, error) {
	cacheKey := fmt.Sprintf("%s:tx_semantic:%s:%t", network, hash, includeDeep)
	if v, ok := c.cache.Get(cacheKey); ok {
		return v.(*SemanticTransactionResponse), nil
	}

	path := c.buildURL(network, "/silver/tx/"+hash+"/semantic")
	if includeDeep {
		path += "?include_deep=true"
	}
	body, err := c.doRequest(ctx, http.MethodGet, path)
	if err != nil {
		return nil, err
	}

	var result SemanticTransactionResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("gateway: parsing tx semantic: %w", err)
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

type accountOverviewNegativeEntry struct{ err error }

func (c *Client) cacheAccountOverviewFailure(ctx context.Context, cacheKey string, err error) {
	if ctx.Err() != nil {
		return
	}
	c.cache.Set(cacheKey, &accountOverviewNegativeEntry{err: err}, TTLAccountFailure)
}

// GetAccountOverview returns comprehensive account info with recent activity.
func (c *Client) GetAccountOverview(ctx context.Context, network string, accountID string) (*AccountOverview, error) {
	cacheKey := network + ":account:" + accountID
	if v, ok := c.cache.Get(cacheKey); ok {
		if neg, isNeg := v.(*accountOverviewNegativeEntry); isNeg {
			return nil, neg.err
		}
		return v.(*AccountOverview), nil
	}

	v, err, _ := c.inflight.DoContext(ctx, cacheKey, func() (any, error) {
		if v, ok := c.cache.Get(cacheKey); ok {
			if neg, isNeg := v.(*accountOverviewNegativeEntry); isNeg {
				return nil, neg.err
			}
			return v, nil
		}

		params := url.Values{"account_id": {accountID}}
		body, err := c.doRequest(ctx, http.MethodGet, c.buildURL(network, "/silver/explorer/account")+"?"+params.Encode())
		if err != nil {
			c.cacheAccountOverviewFailure(ctx, cacheKey, err)
			return nil, err
		}

		var result AccountOverview
		if err := json.Unmarshal(body, &result); err != nil {
			wrapped := fmt.Errorf("gateway: parsing account overview: %w", err)
			c.cacheAccountOverviewFailure(ctx, cacheKey, wrapped)
			return nil, wrapped
		}

		c.cache.Set(cacheKey, &result, TTLAccount)
		return &result, nil
	})
	if err != nil {
		return nil, err
	}
	return v.(*AccountOverview), nil
}

// AccountTransaction matches an item in the /silver/accounts/{id}/transactions response
// (the hot+cold federated account history).
type AccountTransaction struct {
	LedgerSequence  int64    `json:"ledger_sequence"`
	ClosedAt        string   `json:"closed_at"`
	TransactionHash string   `json:"transaction_hash"`
	Successful      *bool    `json:"successful,omitempty"`
	SourceAccount   *string  `json:"source_account,omitempty"`
	ActivityTypes   []string `json:"activity_types"`
	Summary         string   `json:"summary"`
}

// AccountTransactionsResponse is the envelope for /silver/accounts/{id}/transactions.
type AccountTransactionsResponse struct {
	AccountID    string               `json:"account_id"`
	Transactions []AccountTransaction `json:"transactions"`
	Cursor       string               `json:"cursor,omitempty"`
	HasMore      bool                 `json:"has_more"`
}

// GetAccountTransactions returns the account's full hot+cold transaction history from the
// federated /silver/accounts/{id}/transactions endpoint, most-recent first. Unlike the
// hot-only recent operations in the account overview, this includes pre-handoff history
// served from cold storage.
func (c *Client) GetAccountTransactions(ctx context.Context, network, accountID string, limit int, order, cursor string) (*AccountTransactionsResponse, error) {
	if limit <= 0 {
		limit = 400
	}
	if order == "" {
		order = "desc"
	}
	params := url.Values{"limit": {fmt.Sprintf("%d", limit)}, "order": {order}}
	if cursor != "" {
		params.Set("cursor", cursor)
	}
	body, err := c.doRequest(ctx, http.MethodGet, c.buildURL(network, "/silver/accounts/"+accountID+"/transactions")+"?"+params.Encode())
	if err != nil {
		return nil, err
	}
	var result AccountTransactionsResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("gateway: parsing account transactions: %w", err)
	}
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

	// live_only=false is deliberate: the gateway's live-only query path times
	// out (503/504, observed 2026-07-08). Callers filter expired entries
	// instead, which preserves live-only semantics. Revisit once fixed.
	params := url.Values{"limit": {fmt.Sprintf("%d", limit)}, "live_only": {"false"}}
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
		var wrapped struct {
			Calls       []ContractRecentCall `json:"calls"`
			RecentCalls []ContractRecentCall `json:"recent_calls"`
			Invocations []ContractRecentCall `json:"invocations"`
		}
		if wrappedErr := json.Unmarshal(body, &wrapped); wrappedErr != nil {
			return nil, fmt.Errorf("gateway: parsing contract calls: %w", err)
		}
		switch {
		case len(wrapped.Calls) > 0:
			result = wrapped.Calls
		case len(wrapped.RecentCalls) > 0:
			result = wrapped.RecentCalls
		case len(wrapped.Invocations) > 0:
			result = wrapped.Invocations
		default:
			result = []ContractRecentCall{}
		}
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

// GetAssetDetail returns the canonical asset detail endpoint used for native assets,
// classic assets, and token-contract asset identifiers such as C... .
func (c *Client) GetAssetDetail(ctx context.Context, network, asset string) (*AssetDetail, error) {
	if asset == "" {
		return nil, fmt.Errorf("gateway: empty asset")
	}
	cacheKey := fmt.Sprintf("%s:asset_detail:%s", network, asset)
	if v, ok := c.cache.Get(cacheKey); ok {
		return v.(*AssetDetail), nil
	}
	body, err := c.doRequest(ctx, http.MethodGet, c.buildURL(network, "/silver/assets/"+url.PathEscape(asset)))
	if err != nil {
		return nil, err
	}
	var result AssetDetail
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("gateway: parsing asset detail: %w", err)
	}
	c.cache.Set(cacheKey, &result, TTLContracts)
	return &result, nil
}

func (c *Client) GetAssetLinks(ctx context.Context, network, asset string) (*AssetLinksResponse, error) {
	if asset == "" {
		return nil, fmt.Errorf("gateway: empty asset")
	}
	cacheKey := fmt.Sprintf("%s:asset_links:%s", network, asset)
	if v, ok := c.cache.Get(cacheKey); ok {
		return v.(*AssetLinksResponse), nil
	}
	body, err := c.doRequest(ctx, http.MethodGet, c.buildURL(network, "/silver/assets/"+url.PathEscape(asset)+"/links"))
	if err != nil {
		return nil, err
	}
	var result AssetLinksResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("gateway: parsing asset links: %w", err)
	}
	c.cache.Set(cacheKey, &result, TTLContracts)
	return &result, nil
}

func (c *Client) GetAssetPairs(ctx context.Context, network, asset string) (*AssetPairsResponse, error) {
	if asset == "" {
		return nil, fmt.Errorf("gateway: empty asset")
	}
	cacheKey := fmt.Sprintf("%s:asset_pairs:%s:10", network, asset)
	if v, ok := c.cache.Get(cacheKey); ok {
		return v.(*AssetPairsResponse), nil
	}
	params := url.Values{}
	params.Set("limit", "10")
	body, err := c.doRequest(ctx, http.MethodGet, c.buildURL(network, "/silver/assets/"+url.PathEscape(asset)+"/pairs")+"?"+params.Encode())
	if err != nil {
		return nil, err
	}
	var result AssetPairsResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("gateway: parsing asset pairs: %w", err)
	}
	c.cache.Set(cacheKey, &result, TTLRecentList)
	return &result, nil
}

func (c *Client) GetAssetHolders(ctx context.Context, network, asset string) (*AssetHoldersResponse, error) {
	if asset == "" {
		return nil, fmt.Errorf("gateway: empty asset")
	}
	cacheKey := fmt.Sprintf("%s:asset_holders:%s:100", network, asset)
	if v, ok := c.cache.Get(cacheKey); ok {
		return v.(*AssetHoldersResponse), nil
	}
	params := url.Values{}
	params.Set("limit", "100")
	body, err := c.doRequest(ctx, http.MethodGet, c.buildURL(network, "/silver/assets/"+url.PathEscape(asset)+"/holders")+"?"+params.Encode())
	if err != nil {
		return nil, err
	}
	var result AssetHoldersResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("gateway: parsing asset holders: %w", err)
	}
	c.cache.Set(cacheKey, &result, TTLRecentList)
	return &result, nil
}

func (c *Client) GetAssetStats(ctx context.Context, network, asset string) (*AssetStats, error) {
	if asset == "" {
		return nil, fmt.Errorf("gateway: empty asset")
	}
	cacheKey := fmt.Sprintf("%s:asset_stats:%s", network, asset)
	if v, ok := c.cache.Get(cacheKey); ok {
		return v.(*AssetStats), nil
	}
	body, err := c.doRequest(ctx, http.MethodGet, c.buildURL(network, "/silver/assets/"+url.PathEscape(asset)+"/stats"))
	if err != nil {
		return nil, err
	}
	var result AssetStats
	if err := json.Unmarshal(body, &result); err != nil {
		var envelope AssetStatsEnvelope
		if err2 := json.Unmarshal(body, &envelope); err2 != nil {
			return nil, fmt.Errorf("gateway: parsing asset stats: %w", err)
		}
		if envelope.Stats != nil {
			result = *envelope.Stats
		}
	}
	c.cache.Set(cacheKey, &result, TTLRecentList)
	return &result, nil
}

func (c *Client) GetTokenStats(ctx context.Context, network, contractID string) (*TokenStats, error) {
	if contractID == "" {
		return nil, fmt.Errorf("gateway: empty contract id")
	}
	cacheKey := fmt.Sprintf("%s:token_stats:%s", network, contractID)
	if v, ok := c.cache.Get(cacheKey); ok {
		return v.(*TokenStats), nil
	}
	body, err := c.doRequest(ctx, http.MethodGet, c.buildURL(network, "/silver/tokens/"+url.PathEscape(contractID)+"/stats"))
	if err != nil {
		return nil, err
	}
	var result TokenStats
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("gateway: parsing token stats: %w", err)
	}
	c.cache.Set(cacheKey, &result, TTLRecentList)
	return &result, nil
}

func (c *Client) GetTokenTransfers(ctx context.Context, network, contractID string, limit int) (*TokenTransfersResponse, error) {
	if contractID == "" {
		return nil, fmt.Errorf("gateway: empty contract id")
	}
	if limit <= 0 {
		limit = 25
	}
	cacheKey := fmt.Sprintf("%s:token_transfers:%s:%d", network, contractID, limit)
	if v, ok := c.cache.Get(cacheKey); ok {
		return v.(*TokenTransfersResponse), nil
	}
	params := url.Values{}
	params.Set("limit", fmt.Sprintf("%d", limit))
	body, err := c.doRequest(ctx, http.MethodGet, c.buildURL(network, "/silver/tokens/"+url.PathEscape(contractID)+"/transfers")+"?"+params.Encode())
	if err != nil {
		return nil, err
	}
	var result TokenTransfersResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("gateway: parsing token transfers: %w", err)
	}
	c.cache.Set(cacheKey, &result, TTLRecentList)
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
	return c.GetTransfersFiltered(ctx, network, map[string]string{"limit": fmt.Sprintf("%d", limit)})
}

// GetTransfersFiltered returns transfers with optional query filters such as from_account, to_account,
// start_time, end_time, source_type, asset_code, cursor, and order.
func (c *Client) GetTransfersFiltered(ctx context.Context, network string, filters map[string]string) ([]TransferEvent, error) {
	params := url.Values{}
	for k, v := range filters {
		if v != "" {
			params.Set(k, v)
		}
	}
	cacheKey := fmt.Sprintf("%s:transfers:%s", network, params.Encode())
	if v, ok := c.cache.Get(cacheKey); ok {
		return v.([]TransferEvent), nil
	}

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

// GetSmartWalletDetail returns the wallet-centric detail document for a smart wallet contract.
func (c *Client) GetSmartWalletDetail(ctx context.Context, network string, contractID string) (*SmartWalletDetail, error) {
	cacheKey := network + ":smart_wallet_detail:" + contractID
	if v, ok := c.cache.Get(cacheKey); ok {
		return v.(*SmartWalletDetail), nil
	}

	body, err := c.doRequest(ctx, http.MethodGet, c.buildURL(network, "/silver/smart-wallets/"+contractID))
	if err != nil {
		return nil, err
	}

	var result SmartWalletDetail
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("gateway: parsing smart wallet detail: %w", err)
	}

	c.cache.Set(cacheKey, &result, TTLAccount)
	return &result, nil
}

// LookupSmartAccountsByAddress returns smart account contracts where address is an active signer.
func (c *Client) LookupSmartAccountsByAddress(ctx context.Context, network string, address string, limit int) (*SmartAccountLookupResponse, error) {
	if limit <= 0 {
		limit = 100
	}
	cacheKey := fmt.Sprintf("%s:smart_account_lookup_address:%s:%d", network, address, limit)
	if v, ok := c.cache.Get(cacheKey); ok {
		return v.(*SmartAccountLookupResponse), nil
	}

	params := url.Values{}
	params.Set("limit", fmt.Sprintf("%d", limit))
	body, err := c.doRequest(ctx, http.MethodGet, c.buildURL(network, "/silver/smart-accounts/lookup/address/"+url.PathEscape(address))+"?"+params.Encode())
	if err != nil {
		return nil, err
	}

	var result SmartAccountLookupResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("gateway: parsing smart account address lookup: %w", err)
	}

	c.cache.Set(cacheKey, &result, TTLAccount)
	return &result, nil
}

// LookupSmartAccountsByCredential returns smart account contracts where credentialID is an active signer.
func (c *Client) LookupSmartAccountsByCredential(ctx context.Context, network string, credentialID string, limit int) (*SmartAccountLookupResponse, error) {
	if limit <= 0 {
		limit = 100
	}
	cacheKey := fmt.Sprintf("%s:smart_account_lookup_credential:%s:%d", network, credentialID, limit)
	if v, ok := c.cache.Get(cacheKey); ok {
		return v.(*SmartAccountLookupResponse), nil
	}

	params := url.Values{}
	params.Set("limit", fmt.Sprintf("%d", limit))
	body, err := c.doRequest(ctx, http.MethodGet, c.buildURL(network, "/silver/smart-accounts/lookup/credential/"+url.PathEscape(credentialID))+"?"+params.Encode())
	if err != nil {
		return nil, err
	}

	var result SmartAccountLookupResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("gateway: parsing smart account credential lookup: %w", err)
	}

	c.cache.Set(cacheKey, &result, TTLAccount)
	return &result, nil
}

// GetSmartAccountRules returns active context rules, signers, and policies for a smart account contract.
func (c *Client) GetSmartAccountRules(ctx context.Context, network string, contractID string, contextRuleID *int64) (*SmartAccountStateResponse, error) {
	ruleKey := "all"
	if contextRuleID != nil {
		ruleKey = fmt.Sprintf("%d", *contextRuleID)
	}
	cacheKey := fmt.Sprintf("%s:smart_account_rules:%s:%s", network, contractID, ruleKey)
	if v, ok := c.cache.Get(cacheKey); ok {
		return v.(*SmartAccountStateResponse), nil
	}

	rawURL := c.buildURL(network, "/silver/smart-accounts/"+url.PathEscape(contractID)+"/rules")
	if contextRuleID != nil {
		params := url.Values{}
		params.Set("context_rule_id", fmt.Sprintf("%d", *contextRuleID))
		rawURL += "?" + params.Encode()
	}
	body, err := c.doRequest(ctx, http.MethodGet, rawURL)
	if err != nil {
		return nil, err
	}

	var result SmartAccountStateResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("gateway: parsing smart account rules: %w", err)
	}

	c.cache.Set(cacheKey, &result, TTLAccount)
	return &result, nil
}

// GetSmartAccountStats returns aggregate smart account index statistics.
func (c *Client) GetSmartAccountStats(ctx context.Context, network string) (*SmartAccountStatsResponse, error) {
	cacheKey := network + ":smart_account_stats"
	if v, ok := c.cache.Get(cacheKey); ok {
		return v.(*SmartAccountStatsResponse), nil
	}

	body, err := c.doRequest(ctx, http.MethodGet, c.buildURL(network, "/silver/smart-accounts/stats"))
	if err != nil {
		return nil, err
	}

	var result SmartAccountStatsResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("gateway: parsing smart account stats: %w", err)
	}

	c.cache.Set(cacheKey, &result, TTLNetworkStats)
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
	v, err, _ := c.inflight.DoContext(ctx, cacheKey, func() (any, error) {
		if v, ok := c.cache.Get(cacheKey); ok {
			return v, nil
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
	})
	if err != nil {
		return nil, err
	}
	return v.(*BatchDecodedResponse), nil
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
