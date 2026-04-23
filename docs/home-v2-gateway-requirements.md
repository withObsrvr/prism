# Home v2 Gateway Requirements

This document gathers the Gateway data requirements for the full `/v2/home` page, not just the Hero area.

It is organized by page section so it is easy to see:
- what Prism already has mocked
- what Gateway must provide
- what Prism should derive or humanize
- which fields are optional for V1

---

# 1. Responsibility split

## Gateway should provide

- normalized network facts
- counts, percentages, and trend deltas
- indexed ledger summaries
- representative transactions
- TTL / contract health facts
- contract activity rankings
- utilization metrics
- optional marketplace or advanced analytics later

## Prism should provide

- final wording
- human-readable summaries
- role-based copy
- labels, emphasis, and presentation
- fallbacks when data is partial

In short:

- **Gateway = facts and indexed summaries**
- **Prism = humanized presentation**

---

# 2. Full page sections

The current `/v2/home` page has these major areas:

1. Header
2. Hero
3. Prompt / ask bar
4. Alert strip
5. Ledger feed
6. Contracts needing attention
7. Leaders / most-used protocols
8. Network utilization
9. Footer meta

---

# 3. Header requirements

The header currently shows:

- latest ledger number
- age label
- network selector

## Gateway needs to provide

```json
{
  "network": "testnet",
  "latest_ledger": {
    "sequence": 2144030,
    "closed_at": "2026-04-20T22:51:10Z"
  }
}
```

## Prism derives

- human age label like `2.1s ago`
- selected network label formatting

## Minimum required fields

- `network`
- `latest_ledger.sequence`
- `latest_ledger.closed_at`

---

# 4. Hero requirements

Covered in more detail in:

- `docs/home-v2-hero-gateway-requirements.md`

## Gateway needs to provide

- network identity
- health/load classification
- recent cadence / tx activity
- active contract count
- Soroban utilization
- trend deltas vs baseline
- TTL risk summary
- agent/anomaly summary

## Proposed endpoint

- `GET /api/v1/home/hero`

---

# 5. Prompt / ask bar requirements

The prompt bar currently has:

- placeholder text
- static example prompts
- mock quick answers

There are two possible scopes here.

## V1 minimum

Gateway does not need to provide anything.

Prism can keep:
- static placeholder
- static sample prompts
- static quick-answer examples

## V2 / real search-assisted prompt

Gateway would eventually need:

- natural-language query routing
- query suggestions
- answer cards / result clusters
- search hits across accounts/contracts/transactions/NFTs

## Possible future endpoint

`GET /api/v1/home/prompt/suggest?q=...`

or

`GET /api/v1/search?q=...&mode=home`

## Not required for V1 live home

This section can remain Prism-owned for now.

---

# 6. Alert strip requirements

Current mock alert:

- urgent ecosystem issue
- TTL-driven narrative
- title/body/meta/CTA

This is effectively a **network alert / notable condition summary**.

## Gateway needs to provide

At minimum, a normalized top alert summary:

```json
{
  "alert": {
    "type": "ttl_risk",
    "severity": "warn",
    "title": "Three contracts are running out of room",
    "body": "Blend’s lending pool, Soroswap’s router, and Phoenix’s AMM have under four days...",
    "meta": "Why this matters: contract operators need to extend TTL before it hits zero.",
    "cta_label": "Review →",
    "cta_href": "/contracts?filter=expiring"
  }
}
```

## Better normalized shape

Instead of full prose, Gateway can provide:

```json
{
  "alert": {
    "type": "ttl_risk",
    "severity": "warn",
    "affected_contract_count": 3,
    "worst_remaining_hours": 17,
    "top_contracts": [
      "Blend",
      "Soroswap",
      "Phoenix"
    ]
  }
}
```

Then Prism writes the prose.

## Recommendation

Prefer **structured facts** over Gateway-authored prose.

---

# 7. Ledger feed requirements

This is the most advanced live section already partially wired.

The feed needs, per ledger:

- ledger number
- transaction count
- operation count
- validator / closer label if available
- classification counts
- utilization percentages
- close time
- age
- representative transactions

## Gateway currently needed

This is already close to:

- recent ledger list
- per-ledger summary

## Recommended endpoints

1. `GET /api/v1/silver/ledgers/recent?limit=5`
2. `GET /api/v1/silver/ledger/{seq}/summary`

## Required per-ledger fields

```json
{
  "ledger": {
    "sequence": 2144030,
    "closed_at": "2026-04-20T22:51:10Z",
    "closed_by_validator": "GC...",
    "protocol_version": 26
  },
  "totals": {
    "transaction_count": 12,
    "operation_count": 11,
    "successful_tx_count": 11,
    "failed_tx_count": 1
  },
  "classification_counts": {
    "swap_tx_count": 1,
    "contract_call_tx_count": 2,
    "classic_tx_count": 10,
    "payment_tx_count": 9,
    "soroban_tx_count": 2
  },
  "soroban_utilization": {
    "instructions_used": 2933987,
    "read_write_bytes_used": 1468
  },
  "representative_transactions": [
    {
      "tx_hash": "...",
      "category": "classic_payment",
      "category_label": "Classic Payment",
      "coverage_count": 8,
      "summary": {
        "description": "Sent 29310 USDC..."
      }
    }
  ]
}
```

## Prism derives

- age labels
- close time string
- chip wording
- human-readable representative rows

## V1 status

Mostly covered already.

---

# 8. Contracts needing attention requirements

This section currently shows TTL-risk cards.

Each card needs:

- contract / protocol name
- contract id
- severity
- remaining ledgers
- remaining human time
- health tone
- explanatory copy
- progress / runway percentage
- CTA target

## Gateway should provide

A ranked list of contracts nearing TTL expiration.

## Proposed endpoint

`GET /api/v1/home/contracts-needing-attention?limit=4`

## Suggested response

```json
{
  "contracts": [
    {
      "contract_id": "C...",
      "protocol_name": "Blend",
      "contract_name": "Lending Pool",
      "severity": "bad",
      "remaining_ledgers": 12418,
      "remaining_hours": 17,
      "remaining_human": "17 hours",
      "runway_pct": 12,
      "status": "expiring_soon"
    }
  ]
}
```

## Prism derives / writes

- title line like `Blend · Lending Pool`
- warning narrative
- bar width / color mapping
- CTA wording

## Required raw facts

- contract identity
- remaining TTL
- severity / status
- ranking order

---

# 9. Leaders / most-used protocols requirements

This section currently shows:

- most active contracts / protocols in the last day
- call counts
- unique callers
- short explanatory body

## Gateway should provide

A ranked protocol / contract activity leaderboard.

## Proposed endpoint

`GET /api/v1/home/leaders?limit=4&period=24h`

## Suggested response

```json
{
  "leaders": [
    {
      "contract_id": "C...",
      "protocol_name": "Soroswap",
      "contract_name": "Router",
      "call_count_24h": 84201,
      "unique_callers_24h": 412,
      "dominant_actions": ["swap", "deposit"],
      "growth_pct": 0.0
    },
    {
      "contract_id": "C...",
      "protocol_name": "Blend",
      "contract_name": "Lending Pool",
      "call_count_24h": 52318,
      "unique_callers_24h": 187,
      "dominant_actions": ["deposit", "borrow"],
      "growth_pct": 0.0
    }
  ]
}
```

## Prism derives / writes

- labels like `Most active`, `Runner-up`, `Fastest growing`
- card tone / badge color
- human sentence body

## Required raw facts

- rank order
- name / identity
- call count
- unique callers
- dominant actions
- optional growth metric

---

# 10. Network utilization requirements

This section currently shows cards for:

- instruction budget
- read / write
- transaction size

## Gateway should provide

A single current utilization snapshot.

## Proposed endpoint

Either fold into Hero endpoint or expose separately:

- `GET /api/v1/home/utilization`

## Suggested response

```json
{
  "instruction_pct": 64,
  "instruction_used": 64200000,
  "instruction_limit": 100000000,
  "read_write_pct": 60,
  "read_write_used_bytes": 2100000,
  "read_write_limit_bytes": 3500000,
  "tx_size_pct": 48,
  "avg_tx_size_bytes": 89000,
  "tx_size_limit_bytes": 131072
}
```

## Prism derives / writes

- value strings
- `% used` labels
- warning tones
- body copy like `64.2M of 100M instructions this ledger`

## Recommendation

This can likely live inside the Hero response to avoid another request.

---

# 11. Footer meta requirements

Current footer items are largely product / infra claims:

- complete history from protocol v20
- lookup latency
- known protocols count
- open source link

These split into two categories.

## Product constants

Can remain Prism-owned for now:
- open source link
- protocol-history marketing copy

## Dynamic infra metrics
n
If you want them live, Gateway would need:

- lookup avg latency
- p99 latency
- known protocol count / semantic contract count

## Possible endpoint

`GET /api/v1/home/meta`

## Suggested response

```json
{
  "lookup_avg_ms": 94,
  "lookup_p99_ms": 312,
  "known_protocol_count": 412,
  "history_start_protocol": 20
}
```

## Recommendation

Not needed for first live home pass.

---

# 12. Consolidated recommended endpoint plan

There are two good ways to structure this.

## Option A: one home-summary endpoint

`GET /api/v1/home/summary`

that returns all sections:

```json
{
  "header": { ... },
  "hero": { ... },
  "alert": { ... },
  "feed": { ... },
  "contracts_needing_attention": [ ... ],
  "leaders": [ ... ],
  "utilization": { ... },
  "meta": { ... }
}
```

### Pros
- one request
- simple for Prism
- easy consistency between sections

### Cons
- bigger payload
- harder to iterate section-by-section

## Option B: home-specific section endpoints

1. `GET /api/v1/home/hero`
2. `GET /api/v1/home/alert`
3. `GET /api/v1/home/contracts-needing-attention`
4. `GET /api/v1/home/leaders`
5. `GET /api/v1/home/utilization`
6. existing feed endpoints remain as-is

### Pros
- incremental rollout
- each section can evolve independently
- easier caching by section

### Cons
- more requests unless Gateway aggregates internally

## Recommendation

For Prism, best practical shape is:

- keep existing ledger feed endpoints
- add a **single compact home-summary endpoint** for everything else

Suggested:

`GET /api/v1/home/summary`

with fields for:
- header
- hero
- alert
- contracts needing attention
- leaders
- utilization
- optional meta

---

# 13. Proposed Home summary response shape

```json
{
  "network": "testnet",
  "generated_at": "2026-04-20T22:51:12Z",
  "header": {
    "latest_ledger_sequence": 2144030,
    "latest_ledger_closed_at": "2026-04-20T22:51:10Z"
  },
  "hero": {
    "health": {
      "status": "healthy",
      "load_band": "light",
      "activity_band": "normal"
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
  },
  "alert": {
    "type": "ttl_risk",
    "severity": "warn",
    "affected_contract_count": 3,
    "worst_remaining_hours": 17,
    "top_contracts": ["Blend", "Soroswap", "Phoenix"]
  },
  "contracts_needing_attention": [
    {
      "contract_id": "C...",
      "protocol_name": "Blend",
      "contract_name": "Lending Pool",
      "severity": "bad",
      "remaining_ledgers": 12418,
      "remaining_hours": 17,
      "remaining_human": "17 hours",
      "runway_pct": 12,
      "status": "expiring_soon"
    }
  ],
  "leaders": [
    {
      "contract_id": "C...",
      "protocol_name": "Soroswap",
      "contract_name": "Router",
      "call_count_24h": 84201,
      "unique_callers_24h": 412,
      "dominant_actions": ["swap", "deposit"],
      "growth_pct": 0.0
    }
  ],
  "utilization": {
    "instruction_pct": 64,
    "instruction_used": 64200000,
    "instruction_limit": 100000000,
    "read_write_pct": 60,
    "read_write_used_bytes": 2100000,
    "read_write_limit_bytes": 3500000,
    "tx_size_pct": 48,
    "avg_tx_size_bytes": 89000,
    "tx_size_limit_bytes": 131072
  },
  "meta": {
    "lookup_avg_ms": 94,
    "lookup_p99_ms": 312,
    "known_protocol_count": 412,
    "history_start_protocol": 20
  }
}
```

---

# 14. Implementation priority

## P0

Required to make most of `/v2/home` live:

1. header latest ledger fields
2. hero summary fields
3. alert structured summary
4. contracts needing attention list
5. leaders list
6. utilization snapshot
7. existing feed endpoints

## P1

Nice to have:

1. footer meta live metrics
2. prompt suggestions / natural-language search hooks
3. richer anomaly explanations
4. richer protocol growth stats

## P2

Advanced:

1. personalized prompt suggestions
2. per-role tailored home summaries
3. section-specific freshness indicators
4. real-time push instead of polling

---

# 15. Bottom line

To fully support `/v2/home`, Gateway needs to provide data for:

- **Header**: latest ledger and age source
- **Hero**: health, cadence, active contracts, utilization, trends, TTL risk, anomaly/agent signals
- **Alert strip**: top notable network condition
- **Ledger feed**: recent ledgers plus per-ledger summaries and representative transactions
- **Contracts needing attention**: ranked TTL-risk list
- **Leaders**: top-used protocols/contracts over 24h
- **Utilization**: current instruction/read-write/tx-size usage
- **Footer meta**: optional dynamic infra/product metrics

That gives Prism enough structured data to make the whole home page live while keeping wording and presentation in Prism.
