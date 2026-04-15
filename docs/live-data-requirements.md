# Prism Live Data Requirements

This document maps every Prism fragment to the data sources needed to replace mock data with live data. There are two primary data sources:

1. **stellar-query-api** (via obsrvr-gateway) — blockchain data (ledgers, transactions, accounts, contracts, assets, events)
2. **OBSRVR Radar** (`radar.withobsrvr.com/api/v1`) — validator and network consensus data

---

## Data Sources

### stellar-query-api (via obsrvr-gateway)

**Base URL pattern**: `https://gateway.withobsrvr.com/lake/v1/{network}/api/v1`
**Auth**: `Authorization: Api-Key {key}`

### OBSRVR Radar

**Base URL**: `https://radar.withobsrvr.com/api/v1`
**Auth**: None (public API)

**Endpoints**:
| Endpoint | Returns |
|----------|---------|
| `GET /api/v1` | Full network: id, name, passPhrase, maxLedgerVersion, stellarCoreVersion, quorumSetConfiguration, nodes[] |
| `GET /api/v1/nodes` | Array of validator nodes |
| `GET /api/v1/organizations` | Array of validator organizations |
| `GET /api/v1/node-snapshots` | Historical node snapshots with startDate/endDate |

**Node fields**: publicKey, name, alias, homeDomain, ip, port, host, isp, active, isValidating, isFullValidator, overLoaded, activeInScp, lag, versionStr, ledgerVersion, overlayVersion, organizationId, historyUrl, historyArchiveHasError, connectivityError, stellarCoreVersionBehind, geoData (latitude, longitude, countryCode, countryName), statistics (has24HourStats, active24HoursPercentage, validating24HoursPercentage, overLoaded24HoursPercentage, has30DayStats, active30DaysPercentage, validating30DaysPercentage, overLoaded30DaysPercentage), quorumSet (threshold, validators, innerQuorumSets), trustCentralityScore, pageRankScore, trustRank

**Organization fields**: id, name, dba, url, officialEmail, homeDomain, twitter, github, description, validators[], subQuorumAvailable, subQuorum24HoursAvailability, subQuorum30DaysAvailability, hasReliableUptime, tomlState

---

## Fragment → Data Source Mapping

### Home Page

#### HomeNetworkPulse
**Displays**: Latest ledger, 24h tx count, TPS, Soroban calls, validator count

| Field | Source | Endpoint | Response Field |
|-------|--------|----------|----------------|
| `LatestLedger` | stellar-query-api | `GET /silver/ledgers/recent?limit=1` | `latest_sequence` |
| `LedgerAge` | stellar-query-api | `GET /silver/ledgers/recent?limit=1` | `ledgers[0].closed_at` → compute age |
| `TxCount24H` | stellar-query-api | `GET /bronze/stats/network` | `transactions_24h.total` |
| `TxChange` | stellar-query-api | Needs historical comparison — not directly available. Could compute from 2 stats calls 24h apart, or add a `change_pct` field to the stats endpoint |
| `TPSAvg` | stellar-query-api | `GET /silver/stats/network` | `transactions_24h.total / 86400` (compute) |
| `TPSPeak` | stellar-query-api | Not directly available — would need per-ledger tx counts and compute max(tx_count / close_time) |
| `SorobanCalls` | stellar-query-api | `GET /silver/stats/soroban?period=24h` | `invocations` |
| `SorobanChange` | stellar-query-api | Same gap as TxChange — needs historical comparison |
| `Validators` | Radar | `GET /api/v1/nodes` | `count where isValidating == true` |

**Gateway methods**: `GetBronzeNetworkStats`, `GetNetworkStats`
**New integration needed**: Radar client for validator count

---

#### HomeRecentTxs
**Displays**: Last 6 transactions with hash, type, summary, ops, fee, age

| Field | Source | Endpoint | Response Field |
|-------|--------|----------|----------------|
| `Hash` | stellar-query-api | `GET /silver/transactions/recent?limit=6` | `transactions[].tx_hash` |
| `ShortHash` | — | Computed from Hash | `hash[:4] + "…" + hash[-4:]` |
| `Type` | stellar-query-api | `GET /silver/transactions/recent?limit=6` | `transactions[].summary.type` |
| `TypeLabel` | — | Computed from Type | Map type → human label |
| `Summary` | stellar-query-api | `GET /silver/transactions/recent?limit=6` | `transactions[].summary.description` |
| `From` | stellar-query-api | `GET /silver/transactions/recent?limit=6` | `transactions[].source_account` |
| `Ops` | stellar-query-api | `GET /silver/transactions/recent?limit=6` | `transactions[].operation_count` |
| `Fee` | stellar-query-api | `GET /silver/transactions/recent?limit=6` | `transactions[].fee` |
| `Age` | stellar-query-api | `GET /silver/transactions/recent?limit=6` | `transactions[].closed_at` → compute age |

**Gateway method**: `GetRecentTransactions`
**Note**: This is now a single-call serving-backed flow. It replaces the older bronze transactions + decoded summary fan-out pattern for the home-page latest-transactions widget.

---

#### HomeRecentLedgers
**Displays**: Last 6 ledgers with sequence, age, tx count, op count

| Field | Source | Endpoint | Response Field |
|-------|--------|----------|----------------|
| `Sequence` | stellar-query-api | `GET /silver/ledgers/recent?limit=6` | `ledgers[].ledger_sequence` |
| `Age` | stellar-query-api | same | `ledgers[].closed_at` → compute age |
| `TxCount` | stellar-query-api | same | `ledgers[].successful_tx_count + ledgers[].failed_tx_count` |
| `OpCount` | stellar-query-api | same | `ledgers[].operation_count` |
| `IsLatest` | — | Computed | First item in desc-ordered list |

**Gateway method**: `GetRecentLedgers`
**Note**: This is now a single-call serving-backed flow. It replaces the older bronze stats + bronze ledgers 2-call pattern for the home-page latest-ledgers widget.

---

#### HomeTrendingContracts
**Displays**: Top 5 contracts by invocations

| Field | Source | Endpoint | Response Field |
|-------|--------|----------|----------------|
| `Rank` | — | Computed | Index + 1 |
| `Name` | stellar-query-api | `GET /semantic/contracts` or contract interface | `token_name` or `contract_type` — fallback to short address |
| `Tag` | stellar-query-api | `GET /semantic/contracts` | `contract_type` (e.g., "DEX", "Token", "Lending") |
| `TagColor` | — | Computed from Tag | Map contract_type → color |
| `Address` | stellar-query-api | `GET /silver/contracts/top?limit=5` | `contract_id` |
| `Invocations` | stellar-query-api | same | `total_calls` |
| `Change` | stellar-query-api | Not directly available — needs 24h comparison |
| `IsPositive` | — | Computed from Change | |

**Gateway method**: `GetTopContracts`
**Enrichment needed**: Semantic contracts registry for human-readable names/tags

---

#### HomeSidebar (Assets + Fees)
**Displays**: Top 4 assets by volume, fee guide with economy/standard/priority

| Field | Source | Endpoint | Response Field |
|-------|--------|----------|----------------|
| **Assets** | | | |
| `Code` | stellar-query-api | `GET /silver/assets?limit=4&sort_by=volume_24h&order=desc` | `asset_code` |
| `Issuer` | stellar-query-api | same | `asset_issuer` → short format |
| `Volume` | stellar-query-api | same | `volume_24h` |
| `Change` | stellar-query-api | Not directly available | |
| **Fees** | | | |
| `FeeEconomy` | stellar-query-api | `GET /silver/stats/fees?period=24h` | `min_fee` or base fee (100 stroops) |
| `FeeStandard` | stellar-query-api | same | `median_fee` |
| `FeePriority` | stellar-query-api | same | `p99_fee` |
| `SurgeActive` | stellar-query-api | same | `surge_active` |

**Gateway methods**: `GetAssets`, `GetNetworkStats` (or new `GetFeeStats`)
**New gateway method needed**: `GetFeeStats` → `GET /silver/stats/fees`

---

### Transaction Detail

#### TxOverview
**Displays**: Source, contract, fee, Soroban resources, swap info

| Field | Source | Endpoint | Response Field |
|-------|--------|----------|----------------|
| `SourceAddr` | stellar-query-api | `GET /silver/tx/{hash}/full` | `tx_info.source_account` |
| `ContractAddr` | stellar-query-api | same | `contracts_involved[0]` or from operations |
| `ContractName` | stellar-query-api | `GET /semantic/contracts` lookup | `token_name` or short address |
| `ContractFn` | stellar-query-api | `GET /silver/tx/{hash}/full` | `operations[].function` (Soroban invoke) |
| `FeePaid` | stellar-query-api | same | `tx_info.fee_charged` |
| `FeeUSD` | External price API | Not in stellar-query-api | Needs XLM/USD price feed |
| `IsSoroban` | stellar-query-api | same | `operations[].is_soroban` or `soroban_resources != null` |
| `SorobanCPU` | stellar-query-api | same | `soroban_resources.cpu_instructions` |
| `SorobanMem` | stellar-query-api | same | `soroban_resources.memory_bytes` |
| `SorobanReads` | stellar-query-api | same | `soroban_resources.read_entries` |
| `SorobanWrites` | stellar-query-api | same | `soroban_resources.write_entries` |
| `EffectiveRate` | stellar-query-api | Computed from swap operations if applicable |
| `Slippage` | — | Computed | Requires expected vs actual amounts |
| `Route` | — | Computed from call graph | `GET /silver/tx/{hash}/call-graph` |

**Gateway method**: `GetTransactionFull`

---

#### TxOperations
**Displays**: Operations table with index, type, summary, status

| Field | Source | Endpoint | Response Field |
|-------|--------|----------|----------------|
| `Index` | stellar-query-api | `GET /silver/tx/{hash}/full` | `operations[].index` |
| `Type` | stellar-query-api | same | `operations[].type` or `type_name` |
| `IsSoroban` | stellar-query-api | same | `operations[].is_soroban` |
| `IsPrimary` | — | Computed | First operation or invoke_host_function |
| `Status` | stellar-query-api | same | `operations[].successful` → "Success"/"Failed" |
| `SummaryHTML` | stellar-query-api | same | `operations[].description` → format to HTML |
| `Contract` | stellar-query-api | same | `operations[].contract` |
| `Function` | stellar-query-api | same | `operations[].function` |

**Gateway method**: `GetTransactionFull` — operations are included in the full response.

---

#### TxEvents
**Displays**: Events table with type, contract, data

| Field | Source | Endpoint | Response Field |
|-------|--------|----------|----------------|
| `Index` | stellar-query-api | `GET /silver/tx/{hash}/events` | Event index |
| `Type` | stellar-query-api | same | `event_type` (transfer/mint/burn) |
| `TypeColor` | — | Computed from Type | |
| `Contract` | stellar-query-api | same | `contract_id` → short format |
| `DataHTML` | stellar-query-api | same | `from`, `to`, `amount`, `asset` → format to HTML |

**Gateway method**: `GetTransactionFull` — events included, or dedicated `GET /silver/tx/{hash}/events`

---

#### TxBalanceChanges
**Displays**: Net balance changes per account

| Field | Source | Endpoint | Response Field |
|-------|--------|----------|----------------|
| `Account` | stellar-query-api | `GET /silver/tx/{hash}/diffs` | `balance_changes[].account` |
| `Asset` | stellar-query-api | same | `balance_changes[].asset_code` |
| `AssetType` | stellar-query-api | same | `balance_changes[].asset_type` |
| `Change` | stellar-query-api | same | `balance_changes[].after - balance_changes[].before` (compute) |
| `IsPositive` | — | Computed from Change | |
| `IsFee` | stellar-query-api | same | Infer from fee account pattern |
| `IsPool` | stellar-query-api | same | Infer from liquidity pool address pattern |

**Gateway method**: `GetTransactionDiffs`

---

#### TxStateChanges
**Displays**: Soroban state mutations

| Field | Source | Endpoint | Response Field |
|-------|--------|----------|----------------|
| `Action` | stellar-query-api | `GET /silver/tx/{hash}/diffs` | `state_changes[].action` (created/updated/deleted) |
| `Key` | stellar-query-api | same | `state_changes[].key` (decoded ScVal) |
| `Contract` | stellar-query-api | same | `state_changes[].contract_id` |
| `DetailHTML` | stellar-query-api | same | `state_changes[].before/after` → format ScVal to HTML |

**Gateway method**: `GetTransactionDiffs`

---

### Ledger Detail

#### LedgerTxs
**Displays**: All transactions in a ledger

| Field | Source | Endpoint | Response Field |
|-------|--------|----------|----------------|
| `Index` | — | Computed | Row number |
| `Hash` / `ShortHash` | stellar-query-api | `GET /bronze/transactions?start={seq}&end={seq}&limit=50&order=asc` | `tx_hash` |
| `OpType` | stellar-query-api | `GET /silver/operations/enriched?start_ledger={seq}&end_ledger={seq}` | Group by tx_hash, primary op type |
| `Summary` | stellar-query-api | `POST /silver/tx/batch` with hashes | `summary.description` per tx |
| `Ops` | stellar-query-api | `GET /bronze/transactions` | `operation_count` |
| `Fee` | stellar-query-api | same | `fee_charged` |
| `IsFailed` | stellar-query-api | same | `!successful` |

**Gateway method**: `GetTransactions` + batch decode for summaries

---

#### LedgerOpsAndFees
**Displays**: Operation type breakdown chart + fee distribution

| Field | Source | Endpoint | Response Field |
|-------|--------|----------|----------------|
| `OpBreakdown[]` | stellar-query-api | `GET /silver/operations/enriched?start_ledger={seq}&end_ledger={seq}&limit=200` | Group by `type_name`, count per type |
| `FeeBase` | stellar-query-api | `GET /bronze/ledgers?start={seq}&end={seq}` | `base_fee_in_stroops` |
| `FeeMedian` | stellar-query-api | Compute from tx fees in this ledger | Median of `fee_charged` across txs |
| `FeeP99` | stellar-query-api | Compute from tx fees | P99 percentile |
| `SurgePct` | stellar-query-api | `tx_count / max_tx_set_size` from ledger | Compute |

**Gateway methods**: `GetOperations` + `GetLedgers`

---

#### LedgerSoroban
**Displays**: Soroban runtime stats for this ledger

| Field | Source | Endpoint | Response Field |
|-------|--------|----------|----------------|
| `TotalCPU` | stellar-query-api | Not directly available per-ledger | Would need to sum from individual tx soroban_resources |
| `SorobanCalls` | stellar-query-api | `GET /bronze/ledgers` | `soroban_op_count` (if available in bronze ledger) |
| `StateReads` / `StateReadKB` | stellar-query-api | Not available per-ledger | Aggregate from tx diffs |
| `StateWrites` / `StateWriteKB` | stellar-query-api | Not available per-ledger | Aggregate from tx diffs |
| `RentBurned` | stellar-query-api | Not available per-ledger | |

**Gap**: Per-ledger Soroban runtime aggregates are not in the current API. Options:
1. Add a `/silver/ledgers/{seq}/soroban-stats` endpoint to stellar-query-api
2. Compute client-side by fetching all Soroban txs in the ledger and summing resources
3. Keep as mock/placeholder until endpoint exists

---

### Account

#### AccountBalances
**Displays**: Portfolio table with assets, balances, USD values

| Field | Source | Endpoint | Response Field |
|-------|--------|----------|----------------|
| `Code` | stellar-query-api | `GET /silver/accounts/{id}/balances` | `balances[].asset_code` |
| `Name` | stellar-query-api | `GET /semantic/contracts` or TOML lookup | Asset issuer's TOML metadata |
| `BgColor` | — | Computed | Hash asset code → color |
| `Type` | stellar-query-api | `GET /silver/accounts/{id}/balances` | `balances[].asset_type` (native/credit_alphanum4/12) |
| `Balance` | stellar-query-api | same | `balances[].balance` |
| `ValueUSD` | External price API | Not in stellar-query-api | Needs price feed per asset |

**Gateway method**: `GetAccountBalances`
**Gap**: USD values need external price data

---

#### AccountActivity
**Displays**: Recent activity, contract interactions, active offers

| Field | Source | Endpoint | Response Field |
|-------|--------|----------|----------------|
| **Activities** | | | |
| `Summary` | stellar-query-api | `GET /silver/accounts/{id}/activity` or `GET /silver/operations/enriched?account_id={id}&limit=10` | `type_name` + `destination` + `amount` → format |
| `Badge` / `BadgeColor` | — | Computed from operation type | |
| `TxHash` / `ShortHash` | stellar-query-api | same | `transaction_hash` |
| `Time` | stellar-query-api | same | `ledger_closed_at` → compute age |
| **Contracts** | | | |
| `Name` | stellar-query-api | `GET /semantic/contracts` | `token_name` or short address |
| `TopFn` | stellar-query-api | `GET /silver/contracts/{id}/analytics` per contract | `top_functions[0].name` |
| `Calls` | stellar-query-api | same | `total_calls` for this account (not directly available — need account-scoped contract calls) |
| **Offers** | | | |
| All fields | stellar-query-api | `GET /silver/accounts/{id}/offers` | `selling_asset`, `buying_asset`, `amount`, `price` |

**Gateway method**: `GetAccountOverview` (wraps multiple queries)
**Gap**: Per-account contract interaction stats not directly available. May need to query operations filtered by account + contract.

---

#### AccountSigners
**Displays**: Signers table + thresholds

| Field | Source | Endpoint | Response Field |
|-------|--------|----------|----------------|
| `Signers[].Address` | stellar-query-api | `GET /silver/accounts/signers?account_id={id}` | `signers[].signer` |
| `Signers[].Type` | stellar-query-api | same | `signers[].type` |
| `Signers[].Weight` | stellar-query-api | same | `signers[].weight` |
| `Signers[].IsSelf` | — | Computed | `signer == account_id` |
| `Thresholds[].Label` | stellar-query-api | same | `thresholds` keys (low, med, high, master_weight) |
| `Thresholds[].Value` | stellar-query-api | same | `thresholds` values |

**Gateway method**: `GetAccountSigners` — clean match.

---

### Contract Detail

#### ContractInfo
**Displays**: Creator, WASM, storage, success rate

| Field | Source | Endpoint | Response Field |
|-------|--------|----------|----------------|
| `Creator` | stellar-query-api | `GET /semantic/contracts` | `deployer_account` |
| `CreatorHref` | — | Computed | `/account/{deployer_account}` |
| `WASMHash` | stellar-query-api | `GET /silver/contracts/{id}/interface` | WASM hash from contract metadata |
| `WASMSize` | stellar-query-api | same | Not directly available — may need contract storage lookup |
| `StorageEntries` | stellar-query-api | Not directly available | Would need storage enumeration endpoint |
| `StateSize` | stellar-query-api | Not directly available | |
| `MonthlyRent` | stellar-query-api | Not directly available | Would need rent calculation |
| `SuccessRate` | stellar-query-api | `GET /silver/contracts/{id}/analytics` | Compute from `total_calls` and failed calls |

**Gateway method**: `GetContractAnalytics`
**Gaps**: WASM size, storage entries, state size, monthly rent — need new endpoints or compute from contract state reads

---

#### ContractFunctions
**Displays**: Function call stats with 24h/7d/30d breakdowns

| Field | Source | Endpoint | Response Field |
|-------|--------|----------|----------------|
| `Name` | stellar-query-api | `GET /silver/contracts/{id}/analytics` | `top_functions[].name` |
| `Calls24h` | stellar-query-api | same | `top_functions[].count` (current period only) |
| `Calls7d` | stellar-query-api | Not available | Would need `?period=7d` param or separate calls |
| `Calls30d` | stellar-query-api | Not available | Would need `?period=30d` param |
| `SuccessRate` | stellar-query-api | Not available per-function | |
| `AvgCPU` | stellar-query-api | Not available per-function | |
| `LastCalled` | stellar-query-api | `GET /silver/contracts/{id}/recent-calls?limit=1` | `closed_at` for most recent call per function |

**Gateway method**: `GetContractAnalytics`
**Gap**: Per-function time-bucketed stats and success rates not available. Options:
1. Add period params to analytics endpoint
2. Show only total counts and remove time columns

---

#### ContractInvocations
**Displays**: Recent invocation history

| Field | Source | Endpoint | Response Field |
|-------|--------|----------|----------------|
| `TxHash` | stellar-query-api | `GET /silver/contracts/{id}/recent-calls?limit=10` | `transaction_hash` |
| `ShortHash` | — | Computed | |
| `Function` | stellar-query-api | same | `function_name` |
| `Caller` | stellar-query-api | same | `source_account` → short format |
| `Status` | stellar-query-api | same | `successful` → "Success"/"Failed" |
| `StatusColor` | — | Computed | |
| `Age` | stellar-query-api | same | `closed_at` → compute age |

**Gateway method**: `GetContractRecentCalls` — clean match.

---

### Network Health

#### NetworkStatsGrid
**Displays**: Throughput chart, consensus, fees, Soroban, protocol info

| Field | Source | Endpoint | Response Field |
|-------|--------|----------|----------------|
| **Consensus** | | | |
| `Agreement` | Radar | `GET /api/v1` | Compute from node `isValidating` percentages |
| `ValidatorCount` | Radar | `GET /api/v1/nodes` | `count where isValidating == true` |
| `QuorumSets` | Radar | `GET /api/v1` | Count unique `innerQuorumSets` in `quorumSetConfiguration` |
| `AvgLatency` | Radar | `GET /api/v1/nodes` | Average of `lag` across active nodes |
| `ConsensusHalted` | Radar | `GET /api/v1/nodes` | "No" if any node `isValidating`, else "Yes" |
| **Fee Market** | | | |
| `FeeBase` | stellar-query-api | `GET /silver/stats/fees?period=24h` | `min_fee` or base fee constant |
| `FeeMedian` | stellar-query-api | same | `median_fee` |
| `FeeP99` | stellar-query-api | same | `p99_fee` |
| `DailyFees` | stellar-query-api | same | `total_fees` → format |
| `SurgePricing` | stellar-query-api | same | `surge_active` / `surge_pct_of_ledgers` |
| **Soroban** | | | |
| `SorobanInvocations` | stellar-query-api | `GET /silver/stats/soroban?period=24h` | `invocations` |
| `ActiveContracts` | stellar-query-api | same | `active_contracts` |
| `TotalState` | stellar-query-api | same | `total_state_bytes` → format |
| `RentBurned` | stellar-query-api | same | `rent_burned` |
| `AvgCPU` | stellar-query-api | same | `avg_cpu_per_invocation` |
| **Protocol** | | | |
| `ProtocolVer` | Radar | `GET /api/v1` | `maxLedgerVersion` |
| `CoreVersion` | Radar | `GET /api/v1` | `stellarCoreVersion` |
| `HorizonVer` | stellar-query-api | `GET /silver/stats/network` | May not be available — static config |
| `SorobanRPCVer` | — | Static config or not available | |
| `NextUpgrade` | — | Not available from either API | Community knowledge |

**New integration needed**: Radar client for consensus data

---

#### NetworkValidators
**Displays**: Validator table with name, org, uptime, version, latency, status

| Field | Source | Endpoint | Response Field |
|-------|--------|----------|----------------|
| `Name` | Radar | `GET /api/v1/nodes` | `name` or `alias` |
| `Org` | Radar | `GET /api/v1/organizations` | Match via `organizationId` → `organization.name` |
| `Address` | Radar | `GET /api/v1/nodes` | `publicKey` → short format |
| `Uptime` | Radar | same | `statistics.validating30DaysPercentage` → format as % |
| `Version` | Radar | same | `versionStr` |
| `Latency` | Radar | same | `lag` → format as ms |
| `Status` | Radar | same | `isValidating` + `active` → "Validating"/"Active"/"Inactive" |
| `StatusColor` | — | Computed | validating → emerald, active → amber, inactive → red |

**New integration needed**: Full Radar client with nodes + organizations

---

#### NetworkRecentLedgers
**Displays**: Recent ledgers with Soroban breakdown

| Field | Source | Endpoint | Response Field |
|-------|--------|----------|----------------|
| `Sequence` | stellar-query-api | `GET /bronze/ledgers?limit=10&order=desc` | `ledger_sequence` |
| `Age` | stellar-query-api | same | `closed_at` → compute age |
| `TxCount` | stellar-query-api | same | `tx_count` |
| `OpsCount` | stellar-query-api | same | `operation_count` |
| `SorobanCalls` | stellar-query-api | same | `soroban_op_count` (if in bronze) |
| `SorobanPct` | — | Computed | `soroban_op_count / operation_count` |
| `Fees` | stellar-query-api | same | `total_fee_charged` |
| `CloseTime` | stellar-query-api | same | Diff between consecutive `closed_at` timestamps |
| `IsLatest` | — | Computed | First in list |
| `IsSlow` | — | Computed | `close_time > 7s` |

**Gateway method**: `GetLedgers` — mostly clean match.

---

### Asset Directory

#### AssetTable
**Displays**: Paginated asset table with volume, holders, supply

| Field | Source | Endpoint | Response Field |
|-------|--------|----------|----------------|
| `Code` | stellar-query-api | `GET /silver/assets?limit=20&sort_by=volume_24h&order=desc` | `asset_code` |
| `Name` | stellar-query-api | same or TOML lookup | `asset_code` (use as name, or TOML for verified assets) |
| `Issuer` | stellar-query-api | same | `asset_issuer` → short format |
| `Supply` | stellar-query-api | same | `circulating_supply` |
| `Holders` | stellar-query-api | same | `holder_count` |
| `Volume` | stellar-query-api | same | `volume_24h` |
| `Change` | stellar-query-api | Not directly available | Need historical volume comparison |
| `IsVerified` | — | TOML / known issuers list | |
| `TypeBadge` | stellar-query-api | same | `asset_type` → "Classic" or "SEP-41" |

**Gateway method**: `GetAssets` — good match, Change % is the main gap.

---

### Events Firehose

#### EventsStream
**Displays**: Real-time event stream with contract, topics, expandable details

| Field | Source | Endpoint | Response Field |
|-------|--------|----------|----------------|
| `Time` | stellar-query-api | `GET /silver/events?limit=20&order=desc` | `timestamp` → format time |
| `Type` | stellar-query-api | same | `event_type` (transfer/mint/burn) |
| `TypeColor` | — | Computed from Type | |
| `ContractName` | stellar-query-api | `GET /semantic/contracts` | `token_name` — batch lookup |
| `ContractAddr` | stellar-query-api | `GET /silver/events` | `contract_id` → short format |
| `TopicsHTML` | stellar-query-api | same | `from`, `to`, `amount` → format as HTML |
| `Ledger` | stellar-query-api | same | `ledger_sequence` |
| `TxHash` / `TxShort` | stellar-query-api | same | `transaction_hash` |
| `DetailJSON` | stellar-query-api | `GET /silver/events/generic` for raw data | Full event JSON |
| `DetailMeta` | — | Computed | Context from contract registry |
| `IsNew` | — | Client-side | Compare with previous poll |

**Gateway method**: `GetEvents`
**Enrichment**: Contract names from semantic registry

---

## New Gateway Methods Needed

These methods don't exist yet in `internal/gateway/client.go` but are needed:

| Method | Endpoint | Purpose |
|--------|----------|---------|
| `GetFeeStats(ctx, network, period)` | `GET /silver/stats/fees?period={period}` | Fee percentiles, surge detection |
| `GetSorobanStats(ctx, network, period)` | `GET /silver/stats/soroban?period={period}` | Soroban invocations, active contracts, state size |
| `GetSemanticContracts(ctx, network, limit)` | `GET /semantic/contracts?limit={limit}` | Contract names, types, deployers |
| `GetContractInterface(ctx, network, id)` | `GET /silver/contracts/{id}/interface` | WASM hash, detected type, functions |
| `GetAccountActivity(ctx, network, id)` | `GET /silver/accounts/{id}/activity` | Recent account operations |
| `GetAccountOffers(ctx, network, id)` | `GET /silver/accounts/{id}/offers` | Active DEX offers |
| `BatchDecodeTxs(ctx, network, hashes)` | `POST /silver/tx/batch` | Batch transaction summaries |
| `GetGenericEvents(ctx, network, params)` | `GET /silver/events/generic` | Raw contract events with topics |

## New Radar Client Needed

A new `internal/radar/client.go` package to fetch validator/consensus data:

| Method | Endpoint | Purpose |
|--------|----------|---------|
| `GetNetwork(ctx)` | `GET /api/v1` | Full network info + quorum config |
| `GetNodes(ctx)` | `GET /api/v1/nodes` | All validator nodes |
| `GetOrganizations(ctx)` | `GET /api/v1/organizations` | Validator organizations |

Cache TTLs: 30s for nodes (validator status changes), 5m for organizations (rarely changes).

---

## Data Gaps That Need Resolution

### Requires New stellar-query-api Endpoints
1. **Per-ledger Soroban aggregates** — CPU, state reads/writes, rent per ledger
2. **Per-function time-bucketed stats** — 7d/30d call counts per contract function
3. **Per-function success rates** — Success/fail ratio per function

### Requires External Data
1. **USD prices** — XLM/USD and asset/USD for portfolio values and fee USD equivalents
2. **Change percentages** — 24h volume change, tx count change (needs historical comparison or precomputed in API)

### Can Be Computed Client-Side
1. **TPS** — `tx_count_24h / 86400`
2. **Op breakdown** — Group enriched operations by type
3. **Balance change signs** — `after - before` from diffs
4. **Close time** — Diff consecutive ledger timestamps
5. **Soroban percentage** — `soroban_ops / total_ops`

### Can Use Fallback/Placeholder
1. **Contract names** — Fall back to short address if semantic registry has no entry
2. **Asset names** — Fall back to asset code
3. **Horizon/Soroban RPC versions** — Static config
4. **Next upgrade** — "No scheduled upgrade" placeholder
