# Contract and Smart Account Balance Detail Plan

Date: 2026-07-19

## Implementation status

Implemented in the current uncommitted Prism worktree on 2026-07-19:

- Smart-account pages use `/silver/smart-wallets/{contract_id}/balances` alongside the fast rules path.
- Regular contract pages use `/silver/addresses/{contract_id}/balances` without `include_tokens=true`.
- Both documents normalize into one precision-safe portfolio view model and one shared balance presentation.
- The smart-account hero now reports native XLM or a holdings count, with no invented USD total.
- Issuer and token-contract identity remain visible as separate fields, so duplicate symbols are not collapsed.
- Balance reads have a 1.5 second deadline and fail softly without replacing the requested identity with unrelated mock data.
- Complete, empty, partial, not-materialized, and unavailable states render explicitly on v2 and legacy detail pages.

Verification completed with gateway decoding tests, handler endpoint/soft-failure tests, template rendering tests, the full race-enabled Go suite, focused `go vet`, a production build, and desktop/mobile browser review.

## Outcome

Show current, materialized asset balances on Prism's v2 smart-account and contract detail pages without inventing a USD total, losing duplicate asset symbols, slowing the rest of the page, or falling back to unrelated mock data.

The first vertical slice should make this testnet fixture useful end to end:

```text
/v2/account/CC3L3ACABZIRMM5OJDQ6CFV27HWP3ITZ5GOAF6ZIAYTNFWY7AM3VXWXW/smart?network=testnet
```

Expected portfolio:

- 19,971.7336815 XLM
- 189.5292097 AQUA
- 71.0157688 USDC

## Verified API behavior

### Dedicated C-address balance document

`GET /api/v1/silver/smart-wallets/{contract_id}/balances`

This is the preferred serving document for the smart-account page. Despite its route name, the handler currently accepts any contract ID and reads materialized address state without first requiring wallet classification. Prism should not depend on that incidental behavior for regular contracts. Reserve this document for classified smart accounts and use the generic address document for other C-addresses.

Response envelope:

```json
{
  "contract_id": "C...",
  "native_balance": "19971.733681500000",
  "native_balance_source": "contract_storage_state",
  "balances": [],
  "count": 3,
  "partial": false,
  "balance_status": "materialized"
}
```

Balance row:

```json
{
  "asset_code": "USDC",
  "asset_type": "credit_alphanum4",
  "asset_issuer": "G...",
  "balance": "71.0157688000000000",
  "decimals": 7,
  "symbol": "USDC",
  "token_contract_id": "C...",
  "balance_source": "contract_storage_state"
}
```

Important properties:

- `balance` is already a decimal display value. Prism must not scale it again.
- `decimals` determines meaningful fractional precision. Formatting must stay string based, not `float64` based.
- Asset identity is not the symbol alone. Use token contract ID when present, otherwise asset type + code + issuer.
- `partial=false` plus `balance_status=materialized` is the strongest current-state result.
- `native_balance` is optional. A multi-asset contract can have no XLM row.
- `balance_source=contract_storage_state` means the balance came from current canonical contract storage.

Verified fixtures:

| Fixture | Shape | Expected result |
| --- | --- | --- |
| `CC3L3...VXWXW` | Recognizable portfolio | XLM, AQUA, USDC; materialized and complete |
| `CAOSM...XCFAH` | Simple portfolio | 500 XLM, 10 EURC, 7 USDC |
| `CB56M...LWXYW` | Alphanum12 assets | XLM, TUSDC, SECF791 |
| `CA4ID...QKUTIO` | Dense portfolio | 9 assets, no XLM, six distinct PHANTOM issuers |

### Unified address balance document

`GET /api/v1/silver/addresses/{address}/balances`

This response is richer and supports both G-addresses and C-addresses:

```json
{
  "address": "C...",
  "balances": [
    {
      "asset_type": "native",
      "asset_code": "XLM",
      "contract_id": "C...",
      "symbol": "native",
      "balance_raw": "99950000000",
      "balance": "9995.0000000000000000",
      "decimals": 7,
      "decimals_source": "asset_metadata",
      "balance_source": "contract_storage_state",
      "last_updated_ledger": 2996166,
      "last_updated_at": "2026-06-09 08:14:29"
    }
  ],
  "total_balances": 1,
  "sources": ["contract_storage_state"],
  "partial": true,
  "warnings": []
}
```

It provides useful provenance that the dedicated document does not: raw value, decimals source, last-updated ledger, last-updated time, aggregate sources, and warnings.

This endpoint is viable for regular contract detail pages. A focused test on 2026-07-19 produced these results:

| Contract | Rows returned | Payload | Five-request median | Five-request maximum |
| --- | ---: | ---: | ---: | ---: |
| `CABYF6...URX7Z` | 57 | 25.7 KB | 174 ms | 220 ms |
| `CDCOGD...BR4GK` | 53 | 23.9 KB | 183 ms | 219 ms |
| `CANG2C...H6PGQ` | 61 | 27.6 KB | 175 ms | 213 ms |
| `CBYX7H...L4BOUE` | 24 | 10.9 KB | 159 ms | 196 ms |

All 20 requests returned HTTP 200. Every response had a stable payload size across the five runs, `address` matched the requested contract, `total_balances` matched the number of rows, and every row had a unique asset identity, decimals, display balance, and balance source. All rows came from `contract_storage_state`.

A same-moment control comparison against `/smart-wallets/{contract_id}/balances` also passed for all four contracts. The two documents returned identical asset identity sets, balances, and balance sources: 57/57, 53/53, 61/61, and 24/24 rows, with zero missing rows and zero value or source differences. Only the envelope semantics differed: the wallet document reported `partial=false` and `balance_status=materialized`, while the generic document reported `partial=true` because it did not run the optional transfer-history portfolio query.

The live `CANG2C...H6PGQ` portfolio had grown from the earlier 41-row snapshot to 61 rows. Live acceptance should test response invariants rather than hard-code a mutable balance count.

The earlier 20-second timeouts observed for `CC3L3...VXWXW` and `CA4ID...QKUTIO` were not reproducible in the follow-up test. Six subsequent generic-endpoint requests completed in 114 to 140 ms with stable payloads. Prism should still treat balances as a soft dependency with a short deadline because endpoint and network failures remain possible.

Default requests intentionally return `partial=true` with this warning:

```text
soroban token holdings omitted; pass include_tokens=true to request transfer-history portfolio data
```

That warning refers to optional transfer-history-derived portfolio data. It does not weaken rows sourced from current `contract_storage_state`. Prism should present those rows as authoritative current balances and translate the warning narrowly rather than labeling the entire table unreliable.

Do not add `include_tokens=true` to the regular page request. Testing it on two dense contracts returned the same 57 and 24 materialized rows, took 660 ms and 1.18 seconds, and still returned `partial=true` because the optional portfolio query timed out.

### Balance history

`GET /api/v1/silver/addresses/{address}/balances/history`

The history endpoint requires exactly one of `asset` or `contract_id` and supports ledger bounds, cursor pagination, and ascending or descending order. It also timed out during this review. Historical charts are therefore outside the first page update. Keep the current-state model extensible, but do not make balance history part of the initial acceptance gate.

## Current Prism gaps

1. The gateway client has classic `GetAccountBalances` and the aggregate `GetSmartWalletDetail`, but no dedicated C-address balance method.
2. `buildSmartAccountData` returns as soon as the fast smart-account rules endpoint succeeds. That skips the wallet-detail document and leaves `TotalBalance` unavailable.
3. Existing tests intentionally prevent the rules-backed render from calling the slow wallet-detail endpoint. Preserve that performance decision while allowing the dedicated balance endpoint.
4. The current smart-account view model only has `TotalBalance` and `BalanceCents`. It cannot represent multiple assets, issuer identity, source, completeness, or status.
5. The v2 smart-account hero renders `Estimated value` and then appends `USD`, even though the handler uses `BalanceCents` for the asset code. A live XLM result would read as `XLM USD`.
6. The contract detail view model and template have no balance data or balance section.
7. Smart-account live failures can fall through to an unrelated `$87,204.51` mock portfolio. A real contract route must never display fabricated holdings.
8. The selected network defaults to mainnet. Testnet acceptance must use the network query parameter or network cookie and a configured Gateway client.

## Product and UX decisions

### Shared portfolio model

Introduce one Prism view model used by both C-address pages:

```go
type BalancePortfolio struct {
    OwnerID       string
    NativeBalance string
    Items         []BalanceItem
    Count         int
    Partial       bool
    Status        string
    SourceLabel   string
    Warning       string
    Available     bool
}

type BalanceItem struct {
    AssetCode       string
    AssetType       string
    AssetIssuer     string
    TokenContractID string
    Balance         string
    Decimals        *int
    Source          string
}
```

Keep transport response types in `internal/gateway`. Map them to presentation data in handlers. Do not let endpoint naming or raw source strings leak into templates.

### Smart-account hero

- If XLM exists, show one hero value labeled `Native balance`, followed by `XLM`.
- If XLM does not exist, show `{count} current holdings` instead of promoting an arbitrary token.
- Remove `Estimated value` and the unconditional `USD` suffix. No price API means no defensible cross-asset total.
- Keep approval, signer count, and TTL as supporting facts.

### Shared balances section

Place a shared `Current balances` section:

- On smart-account pages, immediately after identity and before signer/policy controls.
- On regular contract pages, after the identity/trust strip and before activity.

Use a semantic table, not a grid of metric cards:

| Asset | Balance | Issuer | Token contract | Evidence |
| --- | ---: | --- | --- | --- |
| XLM | 19,971.7336815 | Native | `CDLZ...CYSC` | Current contract storage |
| USDC | 71.0157688 | `GBBD...FLA5` | `CBIE...DAMA` | Current contract storage |

Presentation rules:

- Native XLM first.
- Then sort by asset code, issuer, and token contract ID. Do not sort incomparable asset quantities as though they were values in one currency.
- Always show a shortened issuer or token contract when symbols can collide.
- Preserve every row with a unique asset identity. The six PHANTOM issuers in `CA4ID...QKUTIO` must remain six rows.
- Link G-address issuers to account detail and C-address token IDs to contract detail, preserving the selected network.
- Format with tabular mono numerals and meaningful decimal precision. Trim redundant trailing zeros without converting through floating point.
- Translate `contract_storage_state` to `Current contract storage`.
- Show `balance_status=materialized` as a quiet evidence state such as `Current state`, not as a promotional badge.
- On mobile, preserve Asset and Balance. Move issuer, token ID, and evidence into a secondary line or disclosure without shrinking text below the established scale.
- For long portfolios, show the first 10 rows and a keyboard-accessible `Show N more` control. Do not paginate a 57-row current snapshot unless real-world sizes grow substantially.

### Empty, partial, and failure states

- Materialized complete with rows: show the table and current-state evidence.
- No rows with `not_materialized`: `Current balances have not been materialized for this contract yet.`
- Partial result with rows: show rows, then a compact warning explaining that additional history-derived holdings may be unavailable.
- Empty partial result: `No positive current balances were returned from the available sources.` Do not claim that the contract owns nothing.
- Timeout or API error: render `Balances are temporarily unavailable` inside the balance section. Keep the rest of the detail page usable.
- Gateway disabled: keep the requested contract identity and render the unavailable state. Only explicit `?mock=true` should show demo holdings.

## Implementation plan

### Slice 1: Gateway contract and normalization

Files:

- `internal/gateway/types.go`
- `internal/gateway/client.go`
- `internal/gateway/client_test.go`

Work:

1. Add response types for the dedicated C-address balance document, including issuer, decimals, token contract ID, source, partial, and status.
2. Add `GetAddressBalances(ctx, network, address)` for `/silver/addresses/{address}/balances`. Regular contract pages must use this method without `include_tokens=true`.
3. Add `GetSmartWalletBalances(ctx, network, contractID)` for the wallet-specific document. Smart-account pages should use this tailored response.
4. Normalize both transport documents into the shared `BalancePortfolio` presentation model.
5. Cache by network, endpoint kind, and address for 30 seconds, matching account data freshness.
6. Add a strict request timeout shorter than the page write timeout. Balance failure must be soft at the handler layer.
7. Add string-based balance formatting and stable asset identity/sorting helpers with focused unit tests.

Acceptance:

- The CC3L3 fixture decodes three distinct rows.
- CA4ID decodes nine rows and preserves all six PHANTOM issuers.
- The generic endpoint fixture for CABYF6 decodes 57 unique current-state rows.
- A generic response with `balances: null` normalizes to an empty slice without claiming a confirmed zero portfolio.
- Generic `partial=true` plus `sources=["contract_storage_state"]` retains authoritative current-state presentation and a narrowly scoped optional-history warning.
- No formatter converts balance values to `float64`.

### Slice 2: Smart-account vertical slice

Files:

- `internal/handlers/accounts.go`
- `internal/handlers/accounts_test.go`
- `internal/templates/pages/smart_account.templ`
- `internal/templates/v2/pages/smart_account.templ`
- generated Templ output

Work:

1. Start rules and balance reads concurrently with independent short deadlines.
2. Keep the fast rules-backed smart-account path. Do not reintroduce wallet-detail, transfer-history, or account-overview calls into the primary render.
3. Apply balance data to the rules-backed view model before returning it.
4. Replace the overloaded `TotalBalance`/`BalanceCents` contract with the shared portfolio model.
5. Correct the hero semantics and add the shared balance table.
6. Replace real-route mock fallback with requested-identity unavailable data.

Acceptance:

- The existing test that protects the fast rules path continues to reject wallet-detail and transfer calls, but now expects one dedicated balance request.
- CC3L3 shows XLM in the hero and all three assets in the table.
- CA4ID shows `9 current holdings`, no USD label, and all duplicate symbols remain distinguishable.
- A balance timeout does not remove rules, signers, policies, or health content.

### Slice 3: Regular contract vertical slice

Files:

- `internal/handlers/contracts.go`
- `internal/handlers/contracts_test.go`
- `internal/templates/pages/contract_detail.templ`
- `internal/templates/v2/pages/contract_detail.templ`
- generated Templ output

Work:

1. Add `BalancePortfolio` to `ContractDetailData`.
2. Fetch balances from the generic `/silver/addresses/{contract_id}/balances` document independently from metadata, analytics, storage, and recent calls. Do not request `include_tokens=true`.
3. Prefer an htmx fragment or a concurrent soft dependency so a slow balance response cannot delay the full page.
4. Add the shared section after contract identity and trust evidence.
5. Keep the smart-account redirect unchanged. Classified smart accounts still land on the smart-account detail page.
6. Use the four full non-wallet IDs recorded below as live smoke fixtures.

Acceptance:

- A non-wallet C-address with materialized rows displays the complete current portfolio.
- `partial=true` caused only by omitted transfer-history data does not hide or downgrade authoritative `contract_storage_state` rows.
- A zero-row or timed-out response produces an honest local empty/error state.
- Existing function, call, storage, and code sections render unchanged.

### Slice 4: Shared presentation hardening

Files:

- a shared Templ balance component under `internal/templates/v2/components`
- `web/static/css/v2-unified.css`
- template tests or HTML assertions

Work:

1. Extract one shared balance table and status treatment used by both detail pages.
2. Add desktop, narrow desktop, tablet, and mobile layouts.
3. Add `Show N more` behavior for portfolios larger than 10 rows.
4. Ensure focus states, table headers, link names, contrast, and non-color status cues meet Prism's accessibility rules.
5. Preserve network selection on issuer and token-contract links.

### Slice 5: Live testnet acceptance and rollout

Run Prism with a real Gateway client. Do not validate against the default no-key mock mode.

Live checks:

1. `CC3L3...VXWXW`: three assets, precision, native hero, activity remains present.
2. `CAOSM...XCFAH`: simple XLM/EURC/USDC scanability.
3. `CB56M...LWXYW`: alphanum12 labels and shared issuer.
4. `CA4ID...QKUTIO`: nine assets, no-native hero, duplicate PHANTOM identities, expansion behavior.
5. `CCQB...OU2HG`: historical 9,995 XLM fixture from ledger 2,996,166.
6. `CABYF6JZTII7M2T2XBRI232WFSY5CHLGKQZX3VPDF57QN7RPUS2URX7Z`: 57-row market contract through the generic endpoint.
7. `CDCOGD7IKPJOTI4F6FGCLDTW26ST4VUPLXRLNYXMHQSDW5PAKFOBR4GK`: 53-row market and staking contract through the generic endpoint.
8. `CANG2CAQU5X62NBAYQBMAPDPMBXIK5FV53C66SSOR56HL3M2TL6H6PGQ`: mutable dense portfolio, verify invariants rather than an exact count.
9. `CBYX7HAGR7UKDJ5FVP4H27ANV4PNCU2KTNH3ZJO3PYPUHBEGDVL4BOUE`: 24 non-native assets and no native hero.
10. `CAUPXJIGAV4PLNWJIVYTDRMS7YCEQXSQ7GASJWSJT6EPIKZ3TPWUKXT5`: 200 response with no materialized rows.
11. Forced balance timeout while metadata and smart-account rules remain available.

Regression gates:

- `go test ./internal/gateway ./internal/handlers ./internal/templates/v2/pages`
- `go vet ./...`
- Templ generation and CSS build
- `git diff --check`
- Browser review at mobile and desktop widths

## Recommended pull request sequence

1. Generic-address and smart-wallet adapters, normalized portfolio model, formatter, and fixtures.
2. Smart-account page vertical slice using CC3L3 and CA4ID.
3. Regular contract page and dense-portfolio behavior.
4. Optional API provenance and history improvements without changing the shared page model.

This sequence delivers user-visible value early while keeping regular-contract and smart-account endpoint semantics explicit.
