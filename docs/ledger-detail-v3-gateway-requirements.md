# Ledger Detail v3 Gateway Requirements

This document gathers the Gateway data requirements for the `/v2/ledger/{seq}/v3`
page — the interpretive ledger detail design prototyped at
`internal/templates/v2/pages/ledger_detail_v3.templ`.

It is organized so it is easy to see:
- what bronze and serving already carry
- what needs adding to extraction
- what needs carrying through the projectors
- what changes at the endpoint
- what Prism must wire on its own side

Every claim was checked against the running API and the ingester source, not
inferred from schema files. Measurements are cited inline.

---

# 1. On the nebu processors

Two nebu origins — `soroban-tx-resources` and `ledger-change-stats` — were
built and merged in `nebu-processor-registry`. **They are a prototyping vehicle,
not the production origin.** Their purpose was to prove, against live mainnet,
that the facts this page needs are extractable from `LedgerCloseMeta` at all,
before anyone committed to a schema migration and a projector.

They did that job. Everything below is expressed in terms of the existing
obsrvr-lake pipeline, and the nebu work is referenced only where it is useful as
a reference implementation.

The production path is unchanged:

```
stellar-postgres-ingester   →  bronze  (transactions_row_v2, *_state_v1)
serving-projection-processor →  serving (serving.sv_*)
stellar-query-api            →  endpoints
Prism gateway client         →  page
```

---

# 2. Responsibility split

## Gateway should provide

- per-ledger aggregate facts (counts, totals, resource sums)
- the network limits in force **at that ledger**
- per-transaction declared resources and decoded result codes
- per-ledger entry-change counts and archival counts
- raw numbers, never percentages

## Prism should provide

- all division into utilisation percentages
- which of the four limits was binding
- failure grouping and the narrative around it
- every piece of wording on the page

**Gateway = numerators, denominators, and codes. Prism = ratios, verdicts, and prose.**

The caps change on protocol upgrade, so the Gateway should not bake a
moment-in-time ratio into a historical record.

---

# 3. Status at a glance

The scope is considerably smaller than a first read suggests, because bronze
already carries most of it.

| Bucket | Items |
|---|---|
| Already in bronze **and** serving | 4 |
| In bronze, not carried to serving | 2 |
| Needs new bronze extraction | 3 |
| Endpoint-only changes | 2 |
| Prism-side wiring | 1 |

---

# 4. What already exists

## 4.1 Bronze — `transactions_row_v2`

Confirmed in `stellar-postgres-ingester/go/writer.go:1644`:

| Column | Serves |
|---|---|
| `transaction_result_code` | failure grouping — **but see §6.1 on format** |
| `envelope_size_bytes` | transaction-size meter |
| `max_fee` | "highest bid was 17.8× the clearing price" |
| `fee_charged` | clearing price |
| `soroban_resources_instructions` | CPU meter numerator |
| `soroban_resources_read_bytes` / `_write_bytes` | byte-level I/O |
| `rent_fee_charged` | archival cost, partially |

## 4.2 Serving — `sv_ledger_stats_recent`

Confirmed in `serving-projection-processor/go/schema/serving_schema.sql:949`
and `ledgers_projector.go`:

- `total_cpu_insns`, `total_read_bytes`, `total_write_bytes`, `total_rent_stroops`
- `avg_tx_size_bytes`, `p95_tx_size_bytes`, `max_tx_size_bytes`, `tx_size_sample_count`
- full operation-category breakdown, `close_time_seconds`, `base_fee_stroops`

Transaction size is **already aggregated**. What is missing is only the sum —
see §7.1.

## 4.3 Endpoints in place

| Page area | Endpoint | Prism client |
|---|---|---|
| Header, counts, contents | `/api/v1/silver/ledger/{seq}` · `/summary` · `/full` | `GetSilverLedgerFull`, `GetSilverLedgerSummary` |
| Fee panel | `/api/v1/silver/ledgers/{seq}/fees` | `GetLedgerFees` |
| CPU, byte I/O, rent | `/api/v1/silver/ledgers/{seq}/soroban` | `GetLedgerSoroban` |
| Neighbour chain | `/api/v1/silver/ledgers/recent` | `GetSilverRecentLedgers` |
| Capacity **caps** | `/api/v1/silver/soroban/config/limits` | none — §9 |

`/silver/ledger/{seq}/full` already composites `ledger`, `transactions`,
`fees`, `soroban` and `operations`. Prefer extending it over adding siblings.

---

# 5. What needs new bronze extraction

Three items. Two are small; one has no precedent.

## 5.1 Footprint entry counts

**Blocking.** This is the page's central claim — the masthead asserts *"This
ledger ran out of writes"* and the binding-limit chip reads "124 of 128
entries".

Bronze has resource **bytes** but not **entry counts**. Stellar charges ledger
capacity against the entries declared in each transaction's readWrite
footprint.

### Why the distinction is not academic

Counting entries observed to change instead of entries declared undercounts by
20–70%. Measured on mainnet:

| ledger | declared write entries | observed-change proxy |
|---|---|---|
| 60200000 | 325 | 112 |
| 60200002 | 345 | 246 |
| 60200008 | 352 | 286 |

A meter built on the proxy understates congestion precisely when congestion
matters.

### Required

Add to `transactions_row_v2`:

| Column | Source |
|---|---|
| `soroban_footprint_read_entries` | `len(SorobanData.Resources.Footprint.ReadOnly)` |
| `soroban_footprint_write_entries` | `len(SorobanData.Resources.Footprint.ReadWrite)` |

`extractors_soroban.go:1256` already dereferences
`envelope.V1.Tx.Ext.SorobanData.Resources.Footprint` for restored-key
extraction, so the struct is already in hand at the right point. Remember the
fee-bump case: `FeeBump.Tx.InnerTx.V1.Tx.Ext`.

Reference: `soroban_tx_resources.go` in the processor registry.

## 5.2 Contract error type and code

**Blocking.** The failure grouping needs the contract's own error number — the
`#4` in `#4 HealthFactorTooLow` — and the type that distinguishes a contract
declining to proceed from a host fault.

This is the one item with **no existing precedent in the ingester**: there is no
`ScError` handling anywhere in `stellar-postgres-ingester`.

### Required

Scan the transaction's diagnostic events for an `ScError` and add:

| Column | Example |
|---|---|
| `contract_error_type` | `contract`, `wasm_vm`, `budget`, `storage`, `auth` |
| `contract_error_code` | `4` — only meaningful when type is `contract` |

Semantics worth preserving from the prototype:
- **First error wins.** A contract that errors unwinds; later diagnostic events
  describe the unwinding rather than the cause.
- Non-contract errors carry a type but no number. Report the type anyway, so a
  host fault stays distinguishable from a contract refusal.

Reference: `contractError()` in `soroban_tx_resources.go`.

## 5.3 Per-ledger entry change counts

**Blocking.** Nothing counts created / modified / deleted / restored per ledger,
the entry-type split, or evictions per ledger.

This follows an established pattern rather than inventing one:
`ingest.NewLedgerChangeReaderFromLedgerCloseMeta` is already used in six
extractors (`extractors_accounts.go`, `extractors_defi.go`,
`extractors_market.go`), and `evicted_keys_state_v1` / `restored_keys_state_v1`
are already written by `writer.go`.

### Required

A new per-ledger row — `ledger_change_stats_v1` or equivalent — carrying:

- `created`, `updated`, `deleted`, `restored`, `state`, `total_changes`
- counts by reason: fee, fee refund, transaction, operation, upgrade, unknown
- per-entry-type breakdown across all ten `LedgerEntryType` values
- `evicted_keys` and `eviction_available`

### Three semantics that must survive

1. **Read the change type from XDR, do not derive it.** A restore and a create
   both present as "no Pre, some Post". Deriving from nil-ness silently counts
   restores as creations. Use `ingest.Change.ChangeType`.
2. **`eviction_available` is required.** `LedgerCloseMeta` V0 returns
   `(nil, nil)` from `EvictedLedgerKeys()` — indistinguishable from a real zero
   — and panics on an unrecognised version. Check `ledger.V` before calling.
3. **`evicted_keys` is not part of `total_changes`.** Eviction is a ledger-level
   sweep, not a change any transaction made, which is why it can legitimately
   exceed the total.

Reference: `ledger_change_stats.go` in the processor registry, including its
tests for these three cases.

---

# 6. What is in bronze but not carried to serving

## 6.1 `transaction_result_code` → `sv_transactions_recent`

Bronze populates it at `extractors.go:75`:

```go
TransactionResultCode: tx.Result.Result.Result.Code.String(),
```

Two problems:

1. **`transactions_projector.go` does not carry it** into
   `sv_transactions_recent`. The column does not exist there.
2. **The format is the Go identifier**, not the Horizon-style string.
   `Code.String()` yields `TransactionResultCodeTxBadSeq`, not `tx_BAD_SEQ`.
   Nobody reads the former in a UI and no projection should store it.

### Required

Add `result_code` to `sv_transactions_recent`, carrying the canonical form.
Canonicalise either at extraction or in the projector — pick one and be
consistent. Unknown future codes must keep their numeric identity rather than
being folded into an existing bucket; a mislabelled failure code is worse than
an obviously unrecognised one.

Reference: `canonicalEnumCode` in `soroban_tx_resources.go` does this
generically for the whole enum family, including operation result codes.

## 6.2 `max_fee` → `/fees` as the bid

`sv_transactions_recent.max_fee_stroops` exists. `LedgerFeesResponse.MaxFee` is
`MAX(fee_charged)`, which in a contested ledger equals the median — so the
panel's central contrast collapses to nothing.

### Required

Add `max_bid` = `MAX(max_fee_stroops)` to
`GET /api/v1/silver/ledgers/{seq}/fees`. One line in an existing query.

---

# 7. Projector work

## 7.1 `LedgersRecentProjector`

Extend `serving.sv_ledger_stats_recent` and the projector's aggregate:

| Column | Source |
|---|---|
| `total_envelope_bytes` | `SUM(envelope_size_bytes)` — avg/p95/max already exist |
| `soroban_envelope_bytes` | same, filtered to Soroban transactions — see §8.2 |
| `total_read_entries` | `SUM(soroban_footprint_read_entries)` (§5.1) |
| `total_write_entries` | `SUM(soroban_footprint_write_entries)` (§5.1) |
| `max_write_entries_single_tx` | `MAX(...)` — backs "a single oracle update wrote 14 entries" |
| entry-change counts | from §5.3 |
| `evicted_keys`, `eviction_available` | from §5.3 |

The projector already computes size percentiles from `envelope_size_bytes`, so
this is an extension of an existing CTE, not new plumbing.

## 7.2 `TransactionsProjector`

Carry into `sv_transactions_recent`:

`result_code` (§6.1), `contract_error_type`, `contract_error_code` (§5.2),
`soroban_footprint_read_entries`, `soroban_footprint_write_entries` (§5.1).

## 7.3 Cold path

`serving-cold-backfill` builds the same aggregates over DuckDB
(`main.go:1852`). Whatever lands in the hot projector must land there too, or
historical ledgers will return nulls for the new fields and the page will show
gaps that look like data loss.

---

# 8. Endpoint changes

Most of the above surfaces through existing endpoints once the columns exist.
Only these need endpoint-level work.

## 8.1 Evicted and restored keys cannot filter by ledger

**Blocking.** The Archived and Restored cells ask "what happened in ledger N".
`EvictionFilters` has no ledger dimension:

```go
type EvictionFilters struct {
    ContractID string
    Limit      int
    Cursor     *EvictionCursor
}
```

### Required

Add `ledger_sequence` to the filter and accept it as a query parameter on
`/api/v1/silver/soroban/evicted-keys` and `/restored-keys`. A range
(`from_ledger` / `to_ledger`) additionally serves the six-ledger neighbour strip
in one call.

The underlying tables already carry `ledger_sequence`. This is a filter that was
never plumbed, not missing data.

## 8.2 Config limits are current-only, not effective-at-ledger

**Blocking.** `/api/v1/silver/soroban/config/limits` returns the *current*
configuration. Caps change on protocol upgrade, so a historical ledger gets the
wrong denominator and the page silently mis-states utilisation.

The correct resolution already exists — `getServingLedgerLimits` in
`silver_hot_reader.go` selects config settings `WHERE ledger_sequence <= $1`.
It is not wired to this endpoint.

### Required

Accept `?ledger={seq}` and route through the effective-at-ledger lookup. Absent
the parameter, keep current behaviour.

Response shape is otherwise correct and already carries what the page needs:

```json
{
  "instructions": { "ledger_max": 100000000, "tx_max": 100000000, "fee_rate_per_increment": 1000 },
  "ledger":       { "max_read_entries": 200, "max_read_bytes": 133120,
                    "max_write_entries": 125, "max_write_bytes": 66560 },
  "transaction":  { "max_read_entries": 40, "max_write_entries": 25 },
  "contract":     { "max_size_bytes": 65536 },
  "updated_at":   "..."
}
```

## 8.3 Per-ledger changes — new or extended

The entry-change data from §5.3 needs a home. Two options:

- **Extend** `/api/v1/silver/ledgers/{seq}/soroban` — but the data is not
  Soroban-specific; it covers accounts, trustlines and offers.
- **Add** `GET /api/v1/silver/ledgers/{seq}/changes` — cleaner separation.

Recommended: add the endpoint, and include it as a `changes` key in
`/ledger/{seq}/full` so the page still loads in one call.

### Suggested response

```json
{
  "ledger_sequence": 61500126,
  "created": 35, "modified": 3131, "deleted": 32, "restored": 22,
  "total_changes": 3220,
  "by_reason": { "fee": 282, "fee_refund": 107, "transaction": 403,
                 "operation": 1184, "upgrade": 0, "unknown": 0 },
  "entry_types": [
    { "entry_type": "account",        "created": 5,  "updated": 1634, "deleted": 0,  "restored": 0,  "total": 1639 },
    { "entry_type": "trustline",      "created": 1,  "updated": 742,  "deleted": 0,  "restored": 0,  "total": 743  },
    { "entry_type": "offer",          "created": 25, "updated": 256,  "deleted": 26, "restored": 0,  "total": 307  },
    { "entry_type": "contract_data",  "created": 2,  "updated": 447,  "deleted": 3,  "restored": 11, "total": 463  },
    { "entry_type": "ttl",            "created": 2,  "updated": 20,   "deleted": 3,  "restored": 11, "total": 36   },
    { "entry_type": "liquidity_pool", "created": 0,  "updated": 32,   "deleted": 0,  "restored": 0,  "total": 32   }
  ],
  "evicted_keys": 2000,
  "evicted_by_type": [
    { "entry_type": "contract_data", "count": 1000 },
    { "entry_type": "ttl", "count": 1000 }
  ],
  "eviction_available": true
}
```

That payload is the measured profile of mainnet ledger 61,500,126, not an
illustration — a concrete parity target. Both dimensions reconcile to 3,220.

Entry-type ordering must be stable: order by the XDR enum, not map iteration, so
identical input produces identical bytes.

## 8.4 Failure grouping — do not build

The page groups failures by error code with counts and a fee-charged flag. Once
§6.1 and §5.2 land, Prism groups client-side from the `transactions` block of
`/full`. The grouping is presentational and the wording is Prism's job.

**Recommend no endpoint** until that proves slow at 118+ transactions.

---

# 9. Prism-side wiring

`/api/v1/silver/soroban/config/limits` is live and returns exactly the caps the
capacity meters need, but Prism has no client method for it.

### Required in Prism

- `Client.GetSorobanConfigLimits(ctx, network, ledger)` in
  `internal/gateway/client.go`
- A `SorobanConfigLimits` type in `internal/gateway/types.go`

Cheapest item on the list: no backend work, unblocks the CPU meter denominator
immediately, independent of everything else.

---

# 10. Decisions needed before implementation

## 10.1 Which contract archived or restored these entries

The "Notable changes" rows name a contract — *"Twelve storage entries archived
from CNFT8823…QP41"*. Per-ledger counts alone cannot serve that.

`evicted_keys_state_v1` does carry `contract_id`, so this is reachable — but it
means the changes endpoint returns rows, not just counts, or the page makes a
second call to the filtered eviction endpoint from §8.1.

**Decision:** rows in the changes payload, a second call, or drop contract
attribution from those rows.

## 10.2 The transaction-size meter compares two populations

`envelope_size_bytes` summed over a ledger covers **every** transaction. The
only published cap, `txMaxSizeBytes`, is **Soroban-only**.
`silver_hot_reader.go` already declines to publish it as a limit for exactly
this reason.

**Decision:** scope the meter to Soroban transactions so the cap applies (hence
`soroban_envelope_bytes` in §7.1), or show the byte total with no cap. A design
decision, not missing data.

## 10.3 Restore and archival cost

Rows show *"restorable for 0.4 XLM"* and *"cost 1.82 XLM"*. `rent_fee_charged`
is per transaction, not per entry.

**Decision:** compute rent from entry size and the state-archival config
settings, or drop cost from those rows and keep the counts.

---

# 11. Sequencing

1. **Prism wires `config/limits`** (§9). No backend dependency.
2. **Bronze extraction** — §5.1 footprint entries, §5.2 contract error, §5.3
   change stats. Everything downstream depends on this.
3. **Projectors** — §7.1, §7.2, and the cold path §7.3.
4. **Carry-through** — §6.1 result code, §6.2 `max_bid`.
5. **Endpoints** — §8.1 ledger filters, §8.2 effective-at-ledger caps, §8.3
   changes endpoint.

Steps 1–3 make roughly the whole page servable. 4–5 finish it.

---

# 12. Verification

The nebu processors remain useful here: they read the same `LedgerCloseMeta` and
can be run against any ledger to produce the expected values, so each bronze
extraction can be checked against a known-good reference before the pipeline
catches up.

```bash
# Expected footprint entry counts and envelope bytes for a ledger range
nebu-sql --file obsrvr-lake/scripts/nebu-recreations/ledger_capacity.sql

# Expected entry-change and eviction counts
nebu-sql --file obsrvr-lake/scripts/nebu-recreations/ledger_changes.sql
```

Parity targets already measured:

| Ledger | Fact | Value |
|---|---|---|
| 60200000 | declared write entries | 325 |
| 61500126 | total changes / restored / evicted | 3,220 / 22 / 2,000 |
| 61500126 | entry types | account 1,639 · trustline 743 · contract_data 463 |

Acceptance: the v3 page renders with zero `NO SOURCE` chips outside the three
decisions in §10.

---

# 13. Not observable — do not build

The page states these plainly rather than estimating, which is correct:

- **Transactions excluded from the ledger.** A closed ledger records inclusions
  only. Transactions that bid below the clearing price are written nowhere.
- **The oracle price behind a Blend failure.** Prism can see identical error
  codes and their timing; not the price that caused them.

---

# 14. One protocol note that affects extractor design

Restores do **not** come from `RestoreFootprintOp`.

Under [CAP-0062](https://github.com/stellar/stellar-protocol/blob/master/core/cap-0062.md),
archived entries in a transaction's readWrite footprint are restored
automatically during `InvokeHostFunction`, emitting `LedgerEntryRestored`
changes with no dedicated operation. Testnet ledger 3966108 restored four
entries while containing no restore operation at all.

Scanning operation result codes for `RESTORE` finds nothing across roughly
5,400 mainnet and 1,400 testnet ledgers, including ledgers that demonstrably
restored entries. **Detect restores from the change stream, never from
operation codes.**

Note this also means `extractors_soroban.go`'s current restored-key extraction —
which reads `footprint.ReadWrite` on the assumption those entries are being
restored — is a different and broader signal than an actual
`LedgerEntryRestored` change. Worth confirming the two agree before §5.3 lands
next to it.

Restores also cluster immediately after eviction sweeps and are otherwise
absent, so sampling to validate should target a sweep rather than a random
range.
