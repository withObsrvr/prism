# Home v2 Endpoint Strategy

This document proposes the endpoint strategy for feeding `/v2/home`.

## Recommendation

Ideally, the home page should be powered by **3 endpoints total**.

This gives a good balance between:
- keeping the API simple
- avoiding an overly fragmented home page backend
- preserving the existing ledger-feed model
- allowing different refresh/caching strategies for the summary versus the feed

---

# Target endpoint count

## Ideal target: 3 endpoints

1. `GET /api/v1/home/summary`
2. `GET /api/v1/silver/ledgers/recent`
3. `GET /api/v1/silver/ledger/{seq}/summary`

---

# Why 3 endpoints is the right shape

The `/v2/home` page naturally splits into two data domains:

## 1. Home summary domain

This includes everything except the scrolling ledger feed:

- header
- hero
- alert strip
- contracts needing attention
- leaders / most-used contracts
- utilization
- optional footer meta

These can be returned together in one compact summary payload.

## 2. Ledger feed domain

This is different from the summary because it is:

- ledger-driven
- more dynamic
- heavier than the rest of the page
- naturally modeled as a recent-ledger list plus per-ledger summaries

That makes it a good candidate to remain separate from the page-wide summary.

---

# Proposed endpoint breakdown

## Endpoint 1: Home summary

### Route

`GET /api/v1/home/summary`

### Purpose

Returns all the non-feed data for `/v2/home`.

### Covers

- header latest-ledger source fields
- hero summary fields
- alert structured summary
- contracts needing attention
- leaders
- utilization
- optional footer/meta stats

### Why this should be aggregated

These sections are all small, summary-oriented, and rendered together on initial page load.

A single summary endpoint:
- reduces request count
- keeps section data consistent
- is easy to cache briefly
- keeps Prism code simpler

---

## Endpoint 2: Recent ledgers

### Route

`GET /api/v1/silver/ledgers/recent?limit=5`

### Purpose

Returns the most recent ledger list for the home feed.

### Covers

- ledger sequences
- close times
- ledger hashes
- operation counts
- successful / failed transaction counts

### Why it stays separate

This is already a good primitive for the feed.

Prism uses it to know:
- which ledgers to show
- which ledger is newest
- what to poll for refresh behavior

---

## Endpoint 3: Per-ledger summary

### Route

`GET /api/v1/silver/ledger/{seq}/summary`

### Purpose

Returns the rich, human-friendly per-ledger summary used by the feed cards.

### Covers

- ledger metadata
- transaction totals
- classification counts
- Soroban utilization
- representative transactions
- composition / provenance

### Why it stays separate

The home feed needs a richer per-ledger object than the recent-ledger list alone.

Keeping this as a separate endpoint lets Gateway:
- compute summaries per immutable ledger
- cache them aggressively
- reuse them elsewhere later

---

# What each endpoint feeds in the UI

## `GET /api/v1/home/summary`

Feeds:
- top header ledger age source
- hero
- alert strip
- contracts needing attention
- leaders
- utilization
- optional footer stats

## `GET /api/v1/silver/ledgers/recent`

Feeds:
- the feed's top-level recent-ledger list
- newest ledger detection
- polling / refresh logic

## `GET /api/v1/silver/ledger/{seq}/summary`

Feeds:
- each ledger card in the feed
- chip counts
- representative transactions
- utilization percentages per ledger

---

# Why not 1 endpoint?

A single endpoint such as:

`GET /api/v1/home/summary`

could theoretically include the entire feed as well.

That would reduce the page to one request, but it has downsides:

- the feed payload is heavier than the rest of the page
- feed data changes more frequently than summary sections
- per-ledger summaries are naturally reusable objects
- feed polling becomes less efficient
- caching behavior becomes less clean

So while **1 endpoint is possible**, it is not the preferred design.

---

# Why not 5–7 endpoints?

You could also split the page into many small endpoints like:

- `/home/hero`
- `/home/alert`
- `/home/leaders`
- `/home/utilization`
- `/home/contracts-needing-attention`

This would work, but it is probably too fragmented.

Downsides:
- too many requests for one page
- more client orchestration in Prism
- more failure modes
- more coordination between similar summary sections

So while this is flexible, it is not ideal for `/v2/home`.

---

# Why 3 endpoints is the sweet spot

Three endpoints gives you:

- one compact summary endpoint for the rest of the page
- one recent-ledger list endpoint for feed polling
- one per-ledger summary endpoint for rich feed rows

That is enough structure to:
- keep home simple
- keep feed performant
- keep caching sane
- avoid over-fragmentation

---

# Recommended implementation model

## Short term

Keep the current feed model and add:

- `GET /api/v1/home/summary`

Then `/v2/home` is powered by:

1. `GET /api/v1/home/summary`
2. `GET /api/v1/silver/ledgers/recent`
3. `GET /api/v1/silver/ledger/{seq}/summary`

## Medium term

If Gateway later wants to optimize further, it could introduce:

- `GET /api/v1/home/feed`

That would let the feed be fetched in a single call.

At that point, the home page could move to just:

1. `GET /api/v1/home/summary`
2. `GET /api/v1/home/feed`

But with the current architecture, **3 endpoints total** is the preferred target.

---

# Suggested summary endpoint shape

The new summary endpoint should likely aggregate:

```json
{
  "network": "testnet",
  "generated_at": "2026-04-20T22:51:12Z",
  "header": { ... },
  "hero": { ... },
  "alert": { ... },
  "contracts_needing_attention": [ ... ],
  "leaders": [ ... ],
  "utilization": { ... },
  "meta": { ... }
}
```

This keeps all non-feed sections under one API surface.

---

# Bottom line

The ideal endpoint strategy for `/v2/home` is:

## 3 endpoints total

1. **Home summary**
   - `GET /api/v1/home/summary`
2. **Recent ledgers list**
   - `GET /api/v1/silver/ledgers/recent`
3. **Per-ledger rich summary**
   - `GET /api/v1/silver/ledger/{seq}/summary`

This is the recommended target because it keeps the home page simple without collapsing everything into a single oversized response or splitting it into too many tiny endpoints.
