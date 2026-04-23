# Home v2 Hero Gateway Requirements

This document defines what data Gateway needs to provide in order to correctly populate the Hero area on `/v2/home`.

It also proposes:
- a concrete Gateway endpoint shape
- exact Go types for the Gateway client / API responses

---

# 1. What the Hero currently renders

The Hero UI currently renders three fields:

- `Eyebrow`
- `HeadlineHTML`
- `Body`

In code, this is represented by:

```go
type HeroData struct {
    Eyebrow      string
    HeadlineHTML string
    Body         string
}
```

However, the Hero copy implies underlying live metrics.

Examples from the current mock / role variants include:

- network health
- busier than usual / lightly loaded / normal activity
- transactions every 5 seconds
- active smart contracts today
- instruction budget used
- read / write used
- contracts running out of TTL
- anomalous bursts / no anomalous bursts
- agent activity trend

So Gateway should not return prewritten prose. Gateway should return the facts Prism needs to write the prose.

---

# 2. Recommended responsibility split

## Gateway should provide

- normalized network facts
- counts
- percentages
- trend deltas
- health bands
- TTL summary
- anomaly / agent summary

## Prism should provide

- final hero wording
- role-specific copy variants
- HTML emphasis / highlights
- fallback wording when data is partial

This keeps the split clean:

- **Gateway = facts**
- **Prism = wording**

---

# 3. Data categories Gateway needs for the Hero

## 3.1 Network identity

Needed for selecting network-specific wording.

Example:

```json
{
  "network": "testnet"
}
```

---

## 3.2 Health and load classification

Needed for wording such as:

- healthy
- degraded
- halted
- lightly loaded
- busier than usual
- normal activity

Example:

```json
{
  "health": {
    "status": "healthy",
    "load_band": "light",
    "activity_band": "normal"
  }
}
```

Recommended values:

- `status`: `healthy | degraded | halted | unknown`
- `load_band`: `light | moderate | heavy | unknown`
- `activity_band`: `quiet | normal | busy | unknown`

---

## 3.3 Recent cadence / throughput

Needed for wording such as:

- `187 transactions every 5 seconds`
- average throughput over recent ledgers

Example:

```json
{
  "latest_ledger": {
    "sequence": 2144030,
    "closed_at": "2026-04-20T22:51:10Z",
    "transaction_count": 12,
    "operation_count": 11
  },
  "cadence": {
    "avg_close_seconds": 5.0,
    "tx_per_ledger_recent_avg": 187,
    "ops_per_ledger_recent_avg": 424
  }
}
```

---

## 3.4 Active contract count

Needed for wording such as:

- `2,314 smart contracts active today`

Example:

```json
{
  "contracts": {
    "active_24h": 2314
  }
}
```

---

## 3.5 Soroban utilization snapshot

Needed for wording such as:

- `Instruction budget 64% used`
- `read / write 60%`

Example:

```json
{
  "soroban": {
    "instruction_pct": 64,
    "read_write_pct": 60
  }
}
```

These should be explicit percentages, so Prism does not need to know protocol limits.

---

## 3.6 Trend deltas vs baseline

Needed for wording such as:

- busier than usual
- up 22% week-over-week
- activity is normal

Example:

```json
{
  "trends": {
    "tx_vs_24h_avg_pct": 14.0,
    "agent_activity_wow_pct": 22.0,
    "anomaly_detected": false
  }
}
```

---

## 3.7 TTL risk summary

Needed for operator-oriented Hero copy such as:

- `Three contracts are running out of TTL`
- `One has under 17 hours`

Example:

```json
{
  "ttl": {
    "expiring_contract_count": 3,
    "worst_remaining_hours": 17,
    "worst_remaining_ledgers": 12418
  }
}
```

---

## 3.8 Agent / anomaly summary

Needed for compliance-oriented wording such as:

- agent volume is up 22% week-over-week
- no anomalous bursts detected

Example:

```json
{
  "activity_mix": {
    "agent_tx_24h": 1204,
    "swap_tx_24h": 84201,
    "contract_call_tx_24h": 52318
  },
  "trends": {
    "agent_activity_wow_pct": 22.0,
    "anomaly_detected": false
  }
}
```

---

# 4. Minimum viable Hero payload

If the goal is to support all current Hero variants with a minimal Gateway response, this is enough:

```json
{
  "network": "testnet",
  "health_status": "healthy",
  "activity_band": "normal",
  "tx_per_ledger_recent_avg": 187,
  "active_contracts_24h": 2314,
  "instruction_pct": 64,
  "read_write_pct": 60,
  "expiring_contract_count": 3,
  "worst_remaining_hours": 17,
  "agent_activity_wow_pct": 22.0,
  "anomaly_detected": false
}
```

This is the absolute minimum Prism needs to generate the current Hero messaging without inventing numbers.

---

# 5. Recommended Gateway endpoint shape

## Proposed endpoint

`GET /api/v1/home/hero`

Alternative if you want it to live under a more generic network summary API:

`GET /api/v1/network/hero`

Recommended query/context model:
- network chosen via existing network scoping / base path conventions
- no query params needed initially

Example full URL through Gateway conventions:

`/lake/v1/{network}/api/v1/home/hero`

---

# 6. Recommended response shape

```json
{
  "network": "testnet",
  "generated_at": "2026-04-20T22:51:12Z",
  "health": {
    "status": "healthy",
    "load_band": "light",
    "activity_band": "normal"
  },
  "latest_ledger": {
    "sequence": 2144030,
    "closed_at": "2026-04-20T22:51:10Z",
    "transaction_count": 12,
    "operation_count": 11
  },
  "cadence": {
    "avg_close_seconds": 5.0,
    "tx_per_ledger_recent_avg": 187,
    "ops_per_ledger_recent_avg": 424
  },
  "contracts": {
    "active_24h": 2314
  },
  "soroban": {
    "instruction_pct": 64,
    "read_write_pct": 60
  },
  "trends": {
    "tx_vs_24h_avg_pct": 14.0,
    "agent_activity_wow_pct": 22.0,
    "anomaly_detected": false
  },
  "ttl": {
    "expiring_contract_count": 3,
    "worst_remaining_hours": 17,
    "worst_remaining_ledgers": 12418
  },
  "activity_mix": {
    "agent_tx_24h": 1204,
    "swap_tx_24h": 84201,
    "contract_call_tx_24h": 52318
  }
}
```

---

# 7. Exact proposed Go types

These types are intended for Prism's `internal/gateway/types.go`.

```go
// HomeHeroResponse matches /home/hero.
type HomeHeroResponse struct {
    Network      string                `json:"network"`
    GeneratedAt  string                `json:"generated_at,omitempty"`
    Health       HomeHeroHealth        `json:"health"`
    LatestLedger HomeHeroLatestLedger  `json:"latest_ledger"`
    Cadence      HomeHeroCadence       `json:"cadence"`
    Contracts    HomeHeroContracts     `json:"contracts"`
    Soroban      HomeHeroSoroban       `json:"soroban"`
    Trends       HomeHeroTrends        `json:"trends"`
    TTL          HomeHeroTTL           `json:"ttl"`
    ActivityMix  HomeHeroActivityMix   `json:"activity_mix"`
}

type HomeHeroHealth struct {
    Status       string `json:"status,omitempty"`        // healthy, degraded, halted, unknown
    LoadBand     string `json:"load_band,omitempty"`     // light, moderate, heavy, unknown
    ActivityBand string `json:"activity_band,omitempty"` // quiet, normal, busy, unknown
}

type HomeHeroLatestLedger struct {
    Sequence         int64  `json:"sequence"`
    ClosedAt         string `json:"closed_at,omitempty"`
    TransactionCount int64  `json:"transaction_count,omitempty"`
    OperationCount   int64  `json:"operation_count,omitempty"`
}

type HomeHeroCadence struct {
    AvgCloseSeconds       float64 `json:"avg_close_seconds,omitempty"`
    TxPerLedgerRecentAvg  int64   `json:"tx_per_ledger_recent_avg,omitempty"`
    OpsPerLedgerRecentAvg int64   `json:"ops_per_ledger_recent_avg,omitempty"`
}

type HomeHeroContracts struct {
    Active24h int64 `json:"active_24h,omitempty"`
}

type HomeHeroSoroban struct {
    InstructionPct int `json:"instruction_pct,omitempty"`
    ReadWritePct   int `json:"read_write_pct,omitempty"`
}

type HomeHeroTrends struct {
    TxVs24hAvgPct       float64 `json:"tx_vs_24h_avg_pct,omitempty"`
    AgentActivityWoWPct float64 `json:"agent_activity_wow_pct,omitempty"`
    AnomalyDetected     bool    `json:"anomaly_detected"`
}

type HomeHeroTTL struct {
    ExpiringContractCount int64 `json:"expiring_contract_count,omitempty"`
    WorstRemainingHours   int64 `json:"worst_remaining_hours,omitempty"`
    WorstRemainingLedgers int64 `json:"worst_remaining_ledgers,omitempty"`
}

type HomeHeroActivityMix struct {
    AgentTx24h        int64 `json:"agent_tx_24h,omitempty"`
    SwapTx24h         int64 `json:"swap_tx_24h,omitempty"`
    ContractCallTx24h int64 `json:"contract_call_tx_24h,omitempty"`
}
```

---

# 8. Proposed Gateway client method

This is the corresponding client method shape for Prism's `internal/gateway/client.go`.

```go
// GetHomeHero returns the network summary used by the /v2/home Hero area.
func (c *Client) GetHomeHero(ctx context.Context, network string) (*HomeHeroResponse, error)
```

Recommended caching:
- short TTL
- around 5 seconds to 15 seconds

Reasoning:
- the Hero is near-real-time but does not need per-request recomputation
- it should stay roughly aligned with the home feed cadence

---

# 9. Example client implementation shape

```go
func (c *Client) GetHomeHero(ctx context.Context, network string) (*HomeHeroResponse, error) {
    cacheKey := fmt.Sprintf("%s:home_hero", network)
    if v, ok := c.cache.Get(cacheKey); ok {
        return v.(*HomeHeroResponse), nil
    }

    body, err := c.doRequest(ctx, http.MethodGet, c.buildURL(network, "/home/hero"))
    if err != nil {
        return nil, err
    }

    var resp HomeHeroResponse
    if err := json.Unmarshal(body, &resp); err != nil {
        return nil, fmt.Errorf("gateway: parsing home hero: %w", err)
    }

    c.cache.Set(cacheKey, &resp, TTLRecentList)
    return &resp, nil
}
```

---

# 10. How Prism would use this response

Prism should map the raw Gateway data into role-based Hero copy.

## Curious/default Hero

Needs:
- `health.status`
- `health.activity_band`
- `cadence.tx_per_ledger_recent_avg`
- `contracts.active_24h`

## Developer Hero

Needs:
- `health.status`
- `soroban.instruction_pct`
- `soroban.read_write_pct`
- `contracts.active_24h`

## Operator Hero

Needs:
- `ttl.expiring_contract_count`
- `ttl.worst_remaining_hours`

## Compliance Hero

Needs:
- `trends.anomaly_detected`
- `trends.agent_activity_wow_pct`

## User Hero

Can remain mostly static, optionally using:
- `health.status`

---

# 11. Recommendation

Recommended implementation path:

## Phase 1

Add a single Gateway endpoint:

`GET /api/v1/home/hero`

with the response shape above.

## Phase 2

Update Prism `/v2/home` to:
- fetch `GetHomeHero(...)`
- generate Hero copy from the response
- keep static fallback copy when data is missing

## Phase 3

Optionally reuse the same Hero response for:
- status chips
- top-bar network summary
- search/prompt contextual hints

---

# 12. Bottom line

For `/v2/home` Hero, Gateway needs to provide these categories of data:

1. network identity
2. health/load classification
3. recent cadence / tx activity
4. active contract count
5. Soroban utilization
6. trend deltas vs baseline
7. TTL risk summary
8. agent/anomaly summary

That is enough for Prism to generate all current Hero variants cleanly and consistently.
