# Tier 2 Gateway API Requirements

> **Date:** 2026-03-06
> **Status:** Draft
> **Author:** Prism team

## Context

Prism's UI was built against rich mock data. With the Obsrvr Gateway wired in (Phase 1), many fields still show "—" or fall back to mocks because the gateway doesn't return what the UI needs. This spec defines the endpoint enhancements and new endpoints required to close the gap, organized by priority.

**Conventions:**
- Gateway paths are relative to `/lake/v1/{network}/api/v1`
- Radar paths use `https://radar.withobsrvr.com/api/v1` (separate service, no network prefix)
- Field names use `snake_case` matching existing gateway conventions
- `stroops` = 1/10,000,000 XLM (integer)
- "Unlocks" = which Prism UI fields become real data

---

## Priority 1 — Enhance Existing Endpoints

### 1.1 `GET /silver/stats/network`

**Current response (abbreviated):**
```json
{
  "generated_at": "2026-03-06T12:00:00Z",
  "accounts": { "funded": 8234567, "total": 12345678 },
  "ledger": {
    "current_sequence": 55000000,
    "avg_close_time_seconds": 5.2
  },
  "operations_24h": {
    "total": 1234567,
    "contract_invoke": 456789,
    "payments": 234567,
    "create_account": 12345,
    "account_merge": 678,
    "change_trust": 45678,
    "manage_offer": 89012,
    "other": 34567
  }
}
```

**Proposed additions:**

```json
{
  "ledger": {
    "current_sequence": 55000000,
    "avg_close_time_seconds": 5.2,
    "protocol_version": 21
  },
  "operations_24h": {
    "total": 1234567,
    "contract_invoke": 456789,
    "previous_total": 1198000,
    "previous_contract_invoke": 412000
  },
  "transactions_24h": {
    "total": 890000,
    "failed": 12340,
    "failure_rate": 0.0139
  },
  "fees_24h": {
    "median_stroops": 100,
    "p99_stroops": 34000,
    "surge_active": false,
    "daily_total_stroops": 89000000
  },
  "soroban": {
    "active_contracts_24h": 3456,
    "total_state_bytes": 1073741824,
    "rent_burned_24h_stroops": 45000000,
    "avg_cpu_insns": 12500000
  }
}
```

**Unlocks:**

| Prism field | Page | Currently |
|---|---|---|
| `TxChange` (24h delta %) | Home | `""` — needs `previous_total` to compute delta |
| `SorobanChange` (24h delta %) | Home | `""` — needs `previous_contract_invoke` |
| `FeeStandard` / `FeePriority` | Home | Hardcoded `100` / `34000` |
| `SurgeActive` / `SurgeContext` | Home | Always `false` / `"uncongested"` |
| `FailureRate` | Network Health | Hardcoded `"0.00%"` |
| `FeeMedian` / `FeeP99` / `DailyFees` | Network Health | `"—"` |
| `ActiveContracts` / `TotalState` / `RentBurned` / `AvgCPU` | Network Health | `"—"` |
| `ProtocolVersion` | Network Health | `"—"` |

---

### 1.2 `GET /silver/tx/{hash}/full`

**Current `tx_info` shape:**
```json
{
  "tx_info": {
    "hash": "abc123...",
    "ledger_sequence": 55000000,
    "fee_charged": 100,
    "created_at": "2026-03-06T12:00:00Z",
    "memo": "",
    "memo_type": "none",
    "successful": true
  }
}
```

**Proposed additions to `tx_info`:**
```json
{
  "tx_info": {
    "source_account": "GABCD...WXYZ",
    "account_sequence": 1234567890,
    "soroban_resources": {
      "cpu_insns": 25000000,
      "mem_bytes": 1048576,
      "read_bytes": 4096,
      "write_bytes": 2048
    }
  }
}
```

**Unlocks:**

| Prism field | Page | Currently |
|---|---|---|
| `SeqNumber` | Transaction Receipt | `"—"` |
| `SourceAccount` (in header) | Transaction Receipt | Derived from ops, fragile |
| `SorobanCPU` / `SorobanMem` | Transaction Receipt | `"—"` |
| `SorobanReads` / `SorobanWrites` | Transaction Receipt | `"—"` |

---

### 1.3 `GET /bronze/ledgers`

**Current role:** still useful for raw bronze ledger range/detail access and network-health style drill-downs.

**Current per-ledger shape (abbreviated):**
```json
{
  "sequence": 55000000,
  "ledger_hash": "abc...",
  "closed_at": "2026-03-06T12:00:00Z",
  "successful_tx_count": 120,
  "failed_tx_count": 3,
  "operation_count": 456,
  "protocol_version": 21,
  "base_fee": 100,
  "max_tx_set_size": 1000,
  "fee_pool": 12345678
}
```

**Note:** For Prism's home-page "Latest Ledgers" widget, this older bronze range flow has now been superseded by:
- `GET /silver/ledgers/recent?limit=6`

That serving-backed endpoint collapses the previous 2-call pattern:
- `GET /bronze/stats/network`
- `GET /bronze/ledgers?start={N-5}&end={N}&limit=6&order=desc`

into a single request.

**Proposed additions per ledger (still applicable for bronze detail/range usage):**
```json
{
  "soroban_op_count": 87,
  "total_fee_charged": 5678900,
  "events_emitted": 234,
  "soroban_resources": {
    "total_cpu_insns": 500000000,
    "total_read_bytes": 102400,
    "total_write_bytes": 51200,
    "total_rent_stroops": 1234567
  }
}
```

**Unlocks:**

| Prism field | Page | Currently |
|---|---|---|
| `SorobanCalls` per ledger row | Network Health | `"—"` — eliminates need for separate ops fetch |
| `SorobanPct` per ledger | Ledger Detail | Derived from ops fetch; would be direct |
| `EventsEmitted` | Ledger Detail | `"—"` |
| `TotalCPU` / `StateReads` / `StateWrites` / `RentBurned` | Ledger Detail | `"—"` |
| `TotalFeeCharged` (accurate) | Ledger Detail | Uses `fee_pool` as proxy |

---

### 1.4 `GET /silver/explorer/account`

**Current `account` shape:**
```json
{
  "account": {
    "account_id": "GABCD...WXYZ",
    "balance": "1234.5678900",
    "sequence_number": "123456789",
    "num_subentries": 5,
    "last_modified_ledger": 54999000,
    "updated_at": "2026-03-05T18:00:00Z"
  }
}
```

**Proposed additions:**
```json
{
  "account": {
    "created_at": "2024-01-15T10:30:00Z",
    "home_domain": "example.com"
  }
}
```

**Unlocks:**

| Prism field | Page | Currently |
|---|---|---|
| `CreatedAt` | Account Portfolio | Uses `updated_at` (wrong semantics) |
| `HomeDomain` | Account Portfolio | Unset |

---

## Priority 2 — New Endpoints

### 2.1 `GET /silver/stats/fees`

Fee distribution over a configurable period. Powers Home fee economy panel and Network Health fee stats.

**Parameters:**

| Param | Type | Default | Description |
|---|---|---|---|
| `period` | string | `24h` | `1h`, `24h`, `7d` |

**Proposed response:**
```json
{
  "period": "24h",
  "generated_at": "2026-03-06T12:00:00Z",
  "fee_distribution": {
    "base_stroops": 100,
    "median_stroops": 100,
    "p75_stroops": 200,
    "p90_stroops": 1200,
    "p99_stroops": 34000,
    "max_stroops": 100000
  },
  "surge_pricing": {
    "active": false,
    "surge_pct_of_ledgers": 0.02,
    "context": "uncongested"
  },
  "total_fees_stroops": 89000000
}
```

**Unlocks:**

| Prism field | Page | Currently |
|---|---|---|
| `FeeEconomy` / `FeeStandard` / `FeePriority` | Home | Hardcoded |
| `SurgeActive` / `SurgeContext` | Home | Always false |
| `FeeMedian` / `FeeP99` | Network Health | `"—"` |

---

### 2.2 `GET /silver/contracts/{id}/metadata`

Static contract metadata. Powers the Contract Detail header and info panel.

**Proposed response:**
```json
{
  "contract_id": "CABCD...WXYZ",
  "wasm_hash": "abc123def456...",
  "wasm_size_bytes": 65536,
  "creator": "GABCD...WXYZ",
  "deploy_ledger": 52000000,
  "deploy_timestamp": "2025-06-15T14:30:00Z",
  "storage_summary": {
    "persistent_entries": 45,
    "temporary_entries": 12,
    "instance_entries": 3,
    "total_size_bytes": 32768,
    "estimated_monthly_rent_stroops": 5000000
  },
  "exported_functions": [
    { "name": "initialize", "param_count": 3 },
    { "name": "transfer", "param_count": 4 },
    { "name": "balance", "param_count": 1 }
  ]
}
```

**Unlocks:**

| Prism field | Page | Currently |
|---|---|---|
| `Creator` | Contract Detail | Unset |
| `DeployLedger` / deploy date | Contract Detail | Unset |
| `WASMSize` / `WASMHash` | Contract Detail | Unset |
| `StorageEntries` / `StateSize` / `MonthlyRent` | Contract Detail | Unset |
| `ExportedFunctions` list | Contract Detail | Unset |

---

### 2.3 `GET /silver/contracts/{id}/storage`

Contract storage entries with TTL info. Powers the State Rent Tracker page.

**Parameters:**

| Param | Type | Default | Description |
|---|---|---|---|
| `limit` | int | `100` | Max entries |
| `durability` | string | — | Filter: `persistent`, `temporary`, `instance` |
| `cursor` | string | — | Pagination cursor |

**Proposed response:**
```json
{
  "contract_id": "CABCD...WXYZ",
  "entries": [
    {
      "key_xdr": "AAAAAQ==",
      "key_decoded": "Balance:GABCD...WXYZ",
      "durability": "persistent",
      "size_bytes": 128,
      "ttl_ledgers": 518400,
      "ttl_expires_at": "2026-06-15T00:00:00Z",
      "last_modified_ledger": 54990000,
      "rent_per_period_stroops": 1234
    }
  ],
  "total_entries": 60,
  "cursor": "next_page_token",
  "has_more": false
}
```

**Unlocks:**

| Prism field | Page | Currently |
|---|---|---|
| `StorageItems` table | Contract Detail | Unset |
| Full State Rent Tracker page | State Rent Tracker | Entirely mock |

---

### 2.4 `GET /silver/accounts/{id}/offers`

Active DEX offers for an account.

**Parameters:**

| Param | Type | Default | Description |
|---|---|---|---|
| `limit` | int | `20` | Max offers |
| `cursor` | string | — | Pagination cursor |

**Proposed response:**
```json
{
  "account_id": "GABCD...WXYZ",
  "offers": [
    {
      "offer_id": 123456789,
      "selling": { "asset_code": "USDC", "asset_issuer": "GA5Z..." },
      "buying": { "asset_code": "native" },
      "amount": "1000.0000000",
      "price": "0.0833333",
      "last_modified_ledger": 54999500
    }
  ],
  "total_offers": 3
}
```

**Unlocks:**

| Prism field | Page | Currently |
|---|---|---|
| `ActiveOffers` count | Account Portfolio | Hardcoded `"0"` |
| Offers section | Account Portfolio | Entirely mock |

---

### 2.5 `GET /silver/transactions/summaries`

Batch-fetch enriched transaction summaries. Powers the home page transactions panel and ledger detail transaction rows without requiring per-tx `/silver/tx/{hash}/full` calls.

**Parameters (one required):**

| Param | Type | Description |
|---|---|---|
| `hashes` | string | Comma-separated tx hashes (max 25) |
| `ledger` | int | Fetch all txs in a ledger |
| `limit` | int | Max results (default 10, max 50) |

**Proposed response:**
```json
{
  "transactions": [
    {
      "hash": "abc123...",
      "ledger_sequence": 55000000,
      "created_at": "2026-03-06T12:00:00Z",
      "source_account": "GABCD...WXYZ",
      "fee_charged": 100,
      "operation_count": 3,
      "successful": true,
      "tx_type": "contract_invoke",
      "summary": "Invoked transfer on CABCD...WXYZ",
      "primary_asset": { "code": "USDC", "issuer": "GA5Z..." },
      "primary_amount": "500.0000000"
    }
  ]
}
```

**Unlocks:**

| Prism field | Page | Currently |
|---|---|---|
| `Transactions` panel | Home | Entirely mock |
| `TxType` / `OpType` in tx rows | Ledger Detail | Generic guess |
| `Summary` per tx | Ledger Detail | Not available |

---

## Priority 2b — Radar API Integration (Validators)

Validator data comes from the Obsrvr Radar API (formerly Stellarbeat), not the gateway. Base URL: `https://radar.withobsrvr.com/api/v1`.

### 2.6 `GET /api/v1/nodes` (Radar)

Returns all known Stellar nodes/validators. Already available — no changes needed.

**Key response fields per node:**
```json
{
  "publicKey": "GCGB2S2KGYARPVIA37HYZXVRM2YZUEXA6S33ZU5BUDC6THSB62LZSTYH",
  "name": "SDF 1",
  "alias": "sdf1",
  "homeDomain": "www.stellar.org",
  "host": "core-live-a.stellar.org:11625",
  "ip": "34.227.72.189",
  "port": 11625,
  "isp": "Amazon.com Inc.",
  "active": true,
  "isValidating": true,
  "isFullValidator": true,
  "activeInScp": true,
  "overLoaded": false,
  "connectivityError": false,
  "index": 0.83,
  "versionStr": "v22.0.0",
  "overlayVersion": 35,
  "overlayMinVersion": 33,
  "ledgerVersion": 22,
  "lag": 0,
  "historyUrl": "https://history.stellar.org/prd/core-live/core_live_001/",
  "historyArchiveHasError": false,
  "dateDiscovered": "2019-05-31T10:35:09.274Z",
  "dateUpdated": "2024-11-10T00:59:39.573Z",
  "organizationId": "9860311160b56412668f572a6d9454d0",
  "stellarCoreVersionBehind": false,
  "geoData": {
    "latitude": 40.7128,
    "longitude": -74.006,
    "countryCode": "US",
    "countryName": "United States"
  },
  "statistics": {
    "has24HourStats": true,
    "has30DayStats": true,
    "active24HoursPercentage": 100,
    "validating24HoursPercentage": 100,
    "overLoaded24HoursPercentage": 0,
    "active30DaysPercentage": 99.95,
    "validating30DaysPercentage": 99.95,
    "overLoaded30DaysPercentage": 0
  },
  "quorumSet": {
    "threshold": 5,
    "validators": [],
    "innerQuorumSets": [
      {
        "threshold": 2,
        "validators": ["GCGB2S...", "GCM6QM...", "GABMKJ..."],
        "innerQuorumSets": []
      }
    ]
  },
  "trustCentralityScore": "1.06043570",
  "pageRankScore": "0.73133496",
  "trustRank": 1
}
```

**Unlocks:**

| Prism field | Page | Radar field |
|---|---|---|
| `Name` | Validator Detail | `name` |
| `NodeID` / `ShortNodeID` | Validator Detail | `publicKey` |
| `NodeType` | Validator Detail | `isFullValidator` → "Full Validator" / "Watcher" |
| `StatusBadge` | Validator Detail | `isValidating` → "Validating" / "Not Validating" |
| `Badges` | Validator Detail | Derived: `isFullValidator` → "Full Validator", `historyUrl` → "Archive Publisher" |
| `NodeIndex` / `IndexWidth` | Validator Detail | `index` (0-1 scale) |
| `Validating24H` / `Val24HWidth` | Validator Detail | `statistics.validating24HoursPercentage` |
| `Validating30D` / `Val30DWidth` | Validator Detail | `statistics.validating30DaysPercentage` |
| `CrawlerReject` / `CrawlerWidth` | Validator Detail | `statistics.overLoaded30DaysPercentage` |
| `Host` / `IP` / `ISP` | Validator Detail | `host`, `ip`, `isp` |
| `Domain` | Validator Detail | `homeDomain` |
| `Country` | Validator Detail | `geoData.countryName` |
| `Version` | Validator Detail | `versionStr` |
| `Overlay` | Validator Detail | `overlayVersion` + `overlayMinVersion` |
| `LedgerVer` | Validator Detail | `ledgerVersion` |
| `HistoryURL` | Validator Detail | `historyUrl` |
| `Discovered` | Validator Detail | `dateDiscovered` |
| `ExtLag` | Validator Detail | `lag` |
| `QuorumThreshold` | Validator Detail | `quorumSet.threshold` |
| `QuorumSets` | Validator Detail | `quorumSet.innerQuorumSets` (resolve publicKeys to names via the full node list) |
| `Trusts` table (all columns) | Validator Detail | Cross-reference quorumSet validators against full node list |
| `ValidatorCount` | Network Health | `count(where isValidating=true)` |
| `Agreement` | Network Health | Derived from quorum intersection analysis |

### 2.7 `GET /api/v1/organizations` (Radar)

Returns all validator organizations. Already available — no changes needed.

**Key response fields per org:**
```json
{
  "id": "9860311160b56412668f572a6d9454d0",
  "name": "Stellar Development Foundation",
  "homeDomain": "www.stellar.org",
  "url": "https://www.stellar.org",
  "description": "SDF is a non-profit...",
  "validators": [
    "GCGB2S2KGYARPVIA37HYZXVRM2YZUEXA6S33ZU5BUDC6THSB62LZSTYH",
    "GCM6QMP3XKFL46LPIRHL3GFSP7FWZSRZSKKXMOLCD3YEAIFTHPJOPM5Q",
    "GABMKJM6I25XI4K7U6XWMULOUQIQ27BCTMLS6BYYSOWCTCI43SEDRATS"
  ],
  "subQuorumAvailable": true,
  "has24HourStats": true,
  "subQuorum24HoursAvailability": 100,
  "has30DayStats": true,
  "subQuorum30DaysAvailability": 99.97,
  "hasReliableUptime": true,
  "logo": "https://...",
  "tomlState": "valid"
}
```

**Unlocks:**

| Prism field | Page | Radar field |
|---|---|---|
| `Organization` | Validator Detail | Resolve `organizationId` → `organizations.name` |
| `OrgName` | Validator Detail | `name` |
| `OrgNodes` (list with uptimes) | Validator Detail | `validators` array → resolve each to node, get `statistics.validating30DaysPercentage` |
| `OrgUptime` | Validator Detail | `subQuorum30DaysAvailability` |
| `OrgValidators` | Validator Detail | `len(validators)` |
| `TrustsOrgs` | Validator Detail | Count distinct orgs in quorum set |
| Home `Validators` count | Home | `count(where isValidating=true)` from `/nodes` |
| Network Health validators table | Network Health | Combined nodes + orgs data |

### Integration Notes

- **Caching:** Radar data changes slowly (node crawl every ~5 minutes). Cache for 2-5 minutes.
- **Network selection:** Radar serves mainnet data at the base URL. For testnet, check if a separate endpoint exists (e.g., `/api/v1/nodes?network=testnet`) or if testnet nodes are flagged in the response.
- **Node updates timeline:** The Radar API does not expose a change history per node. The "Latest Node Updates" section on Validator Detail will need to be derived by Prism by periodically snapshotting node state and diffing (version changes, IP changes, overlay updates). This could be a background job or stored in a local cache.
- **Trusted-by relationship:** Not directly in the API. Requires inverting the quorum set graph: for node X, scan all nodes whose `quorumSet` contains X's publicKey. The `TrustedByCount` field is derived this way.

---

## Priority 3 — Nice-to-Have

### 3.1 `GET /silver/ledgers/{seq}/fees`

Per-ledger fee histogram. Powers the Ledger Detail fee distribution panel.

**Proposed response:**
```json
{
  "ledger_sequence": 55000000,
  "fee_distribution": {
    "min_stroops": 100,
    "median_stroops": 100,
    "p99_stroops": 5000,
    "max_stroops": 50000,
    "surge_pct": 0.05
  },
  "histogram": [
    { "range": "100-200", "count": 95 },
    { "range": "200-1000", "count": 15 },
    { "range": "1000-10000", "count": 7 },
    { "range": "10000+", "count": 3 }
  ]
}
```

**Unlocks:**

| Prism field | Page | Currently |
|---|---|---|
| `FeeMedian` / `FeeP99` / `SurgePct` | Ledger Detail | `"—"` |

---

### 3.2 Enhance `GET /silver/contracts/{id}/analytics`

**Current response includes:** `stats`, `timeline`, `top_functions` (name + count), `daily_calls_7d`.

**Proposed additions:**
```json
{
  "stats": {
    "success_count": 450000,
    "failure_count": 5600,
    "success_rate": 0.9877
  },
  "top_functions": [
    {
      "name": "transfer",
      "count": 234567,
      "calls_7d": 45000,
      "calls_30d": 180000,
      "avg_cpu_insns": 15000000,
      "last_called": "2026-03-06T11:55:00Z"
    }
  ],
  "daily_calls_30d": [
    { "date": "2026-02-04", "count": 5678 }
  ]
}
```

**Unlocks:**

| Prism field | Page | Currently |
|---|---|---|
| `SuccessRate` | Contract Detail | Unset |
| `Calls7d` / `Calls30d` / `AvgCPU` per function | Contract Detail | Unset |
| 30-day sparkline | Contract Detail | Only 7 data points |

---

### 3.3 `GET /silver/stats/soroban`

Deep Soroban network metrics. Powers a potential Soroban-focused dashboard.

**Proposed response:**
```json
{
  "generated_at": "2026-03-06T12:00:00Z",
  "period": "24h",
  "contracts": {
    "total_deployed": 12345,
    "active_24h": 3456,
    "new_24h": 78
  },
  "execution": {
    "total_invocations": 456789,
    "avg_cpu_insns": 12500000,
    "avg_mem_bytes": 524288,
    "total_cpu_insns": 5712362500000
  },
  "state": {
    "total_entries": 8900000,
    "total_size_bytes": 2147483648,
    "persistent_entries": 6700000,
    "temporary_entries": 2200000
  },
  "rent": {
    "total_burned_24h_stroops": 45000000,
    "avg_per_contract_stroops": 13020
  }
}
```

**Unlocks:**

| Prism field | Page | Currently |
|---|---|---|
| Deep Soroban panel | Network Health | `"—"` for all Soroban metrics |

---

## Summary Matrix

| # | Endpoint | Source | Type | Priority | Pages Unlocked |
|---|---|---|---|---|---|
| 1.1 | `/silver/stats/network` | Gateway | Enhance | P1 | Home, Network Health |
| 1.2 | `/silver/tx/{hash}/full` | Gateway | Enhance | P1 | Transaction Receipt |
| 1.3 | `/bronze/ledgers` | Gateway | Enhance | P1 | Network Health, Ledger Detail |
| 1.3a | `/silver/ledgers/recent` | Gateway | New | P1 | Home |
| 1.4 | `/silver/explorer/account` | Gateway | Enhance | P1 | Account Portfolio |
| 2.1 | `/silver/stats/fees` | Gateway | New | P2 | Home, Network Health |
| 2.2 | `/silver/contracts/{id}/metadata` | Gateway | New | P2 | Contract Detail |
| 2.3 | `/silver/contracts/{id}/storage` | Gateway | New | P2 | Contract Detail, State Rent Tracker |
| 2.4 | `/silver/accounts/{id}/offers` | Gateway | New | P2 | Account Portfolio |
| 2.5 | `/silver/transactions/summaries` | Gateway | New | P2 | Home, Ledger Detail |
| 2.6 | `/api/v1/nodes` | Radar | Existing | P2 | Validator Detail, Network Health, Home |
| 2.7 | `/api/v1/organizations` | Radar | Existing | P2 | Validator Detail, Network Health |
| 3.1 | `/silver/ledgers/{seq}/fees` | Gateway | New | P3 | Ledger Detail |
| 3.2 | `/silver/contracts/{id}/analytics` | Gateway | Enhance | P3 | Contract Detail |
| 3.3 | `/silver/stats/soroban` | Gateway | New | P3 | Network Health |

---

## Out of Scope

These gaps require external data sources beyond the gateway and Radar:

| Gap | Required source | Affected pages |
|---|---|---|
| USD pricing | Price oracle / aggregator | All pages showing USD values |
| Contract names / verification | Contract registry / toml files | Contract Detail, Home |
| Asset metadata (name, icon, verified) | Asset directory / toml files | Asset Directory, Account Portfolio |
| NFT metadata | IPFS / custom indexer | NFT Gallery |
| Core/Horizon/Soroban-RPC versions | Horizon `/info` endpoint | Network Health |
