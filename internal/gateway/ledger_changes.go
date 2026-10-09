package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// TTLLedgerFacts covers per-ledger facts that never change once the ledger is
// closed.
const TTLLedgerFacts = 5 * time.Minute

// LedgerChanges is a ledger's ledger-entry change counts.
//
// Created, Updated, Deleted and Restored sum to Total. Evicted sits outside
// that sum: an eviction is the protocol sweeping an expired entry rather than
// a transaction changing state, and it can legitimately exceed the total.
type LedgerChanges struct {
	LedgerSequence int64 `json:"ledger_sequence"`

	Created  int64 `json:"created"`
	Updated  int64 `json:"updated"`
	Deleted  int64 `json:"deleted"`
	Restored int64 `json:"restored"`
	Total    int64 `json:"total"`
	Evicted  int64 `json:"evicted"`

	ByType        map[string]LedgerChangeTypeCounts `json:"by_type,omitempty"`
	EvictedByType map[string]int64                  `json:"evicted_by_type,omitempty"`

	// Available separates a ledger that changed nothing from one ingested
	// before change statistics were captured. Both are a row of zeros.
	Available bool `json:"available"`
}

type LedgerChangeTypeCounts struct {
	Total    int64 `json:"total"`
	Created  int64 `json:"created"`
	Updated  int64 `json:"updated"`
	Deleted  int64 `json:"deleted"`
	Restored int64 `json:"restored"`
}

// GetLedgerChanges returns a ledger's entry-change counts.
func (c *Client) GetLedgerChanges(ctx context.Context, network string, sequence int64) (*LedgerChanges, error) {
	cacheKey := fmt.Sprintf("%s:ledger_changes:%d", network, sequence)
	if v, ok := c.cache.Get(cacheKey); ok {
		return v.(*LedgerChanges), nil
	}
	v, err, _ := c.inflight.DoContext(ctx, cacheKey, func() (any, error) {
		if v, ok := c.cache.Get(cacheKey); ok {
			return v, nil
		}

		body, err := c.doRequest(ctx, http.MethodGet,
			c.buildURL(network, fmt.Sprintf("/silver/ledgers/%d/changes", sequence)))
		if err != nil {
			return nil, err
		}

		var resp LedgerChanges
		if err := json.Unmarshal(body, &resp); err != nil {
			return nil, fmt.Errorf("gateway: parsing ledger changes: %w", err)
		}

		c.cache.Set(cacheKey, &resp, TTLLedgerFacts)
		return &resp, nil
	})
	if err != nil {
		return nil, err
	}
	return v.(*LedgerChanges), nil
}

