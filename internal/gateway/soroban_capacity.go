package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// SorobanConfig is the network's Soroban configuration: the denominators a
// capacity meter divides by.
//
// Each setting is its own ledger entry upstream, so the serving layer folds
// several rows together to produce this. A zero here means the limit was not
// recorded, which is indistinguishable from a genuine zero — callers should
// treat zero as "no denominator" rather than rendering a meter against it.
type SorobanConfig struct {
	Instructions       SorobanInstructionLimits `json:"instructions"`
	Memory             SorobanMemoryLimits      `json:"memory"`
	LedgerLimits       SorobanIOLimits          `json:"ledger_limits"`
	TxLimits           SorobanIOLimits          `json:"tx_limits"`
	Contract           SorobanContractLimits    `json:"contract"`
	LastModifiedLedger int64                    `json:"last_modified_ledger"`
	UpdatedAt          time.Time                `json:"updated_at"`
}

type SorobanInstructionLimits struct {
	LedgerMax           int64 `json:"ledger_max"`
	TxMax               int64 `json:"tx_max"`
	FeeRatePerIncrement int64 `json:"fee_rate_per_increment"`
}

type SorobanMemoryLimits struct {
	TxLimitBytes int64 `json:"tx_limit_bytes"`
}

type SorobanIOLimits struct {
	MaxReadEntries  int64 `json:"max_read_entries"`
	MaxReadBytes    int64 `json:"max_read_bytes"`
	MaxWriteEntries int64 `json:"max_write_entries"`
	MaxWriteBytes   int64 `json:"max_write_bytes"`
}

type SorobanContractLimits struct {
	MaxSizeBytes int64 `json:"max_size_bytes"`
}

type sorobanConfigEnvelope struct {
	Config *SorobanConfig `json:"config"`
}

const (
	// Limits change only on protocol upgrade, so this can be held far longer
	// than a per-ledger fact.
	TTLSorobanConfig = 10 * time.Minute
)


// GetSorobanConfig returns the Soroban limits currently in force.
func (c *Client) GetSorobanConfig(ctx context.Context, network string) (*SorobanConfig, error) {
	return c.getSorobanConfig(ctx, network, "/silver/soroban/config", fmt.Sprintf("%s:soroban_config", network))
}

// GetSorobanConfigAtLedger returns the limits that were in force at a given
// ledger. Capacity caps change on protocol upgrade, so a historical ledger
// divided by today's caps yields a percentage that looks entirely reasonable
// and is wrong.
func (c *Client) GetSorobanConfigAtLedger(ctx context.Context, network string, sequence int64) (*SorobanConfig, error) {
	return c.getSorobanConfig(ctx, network,
		fmt.Sprintf("/silver/soroban/config?ledger=%d", sequence),
		fmt.Sprintf("%s:soroban_config:%d", network, sequence))
}

func (c *Client) getSorobanConfig(ctx context.Context, network, path, cacheKey string) (*SorobanConfig, error) {
	if v, ok := c.cache.Get(cacheKey); ok {
		return v.(*SorobanConfig), nil
	}
	v, err, _ := c.inflight.DoContext(ctx, cacheKey, func() (any, error) {
		if v, ok := c.cache.Get(cacheKey); ok {
			return v, nil
		}

		body, err := c.doRequest(ctx, http.MethodGet, c.buildURL(network, path))
		if err != nil {
			return nil, err
		}

		var envelope sorobanConfigEnvelope
		if err := json.Unmarshal(body, &envelope); err != nil {
			return nil, fmt.Errorf("gateway: parsing soroban config: %w", err)
		}
		if envelope.Config == nil {
			return nil, fmt.Errorf("gateway: soroban config absent for %s", network)
		}

		c.cache.Set(cacheKey, envelope.Config, TTLSorobanConfig)
		return envelope.Config, nil
	})
	if err != nil {
		return nil, err
	}
	return v.(*SorobanConfig), nil
}

