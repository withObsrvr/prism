# NFT Gateway Endpoint Checklist

This document proposes a concrete NFT API checklist for Gateway to support Prism.

It is intentionally split into:
- **P0 / V1 required** endpoints to make Prism's NFT experience real
- **P1** enhancements that improve usability
- **P2** marketplace / advanced analytics features

---

# Design principles

Gateway responses should be:

- normalized for explorer use
- independent of any one NFT contract implementation where possible
- explicit about standard detection (`sep_50`, `oz_non_fungible`, `custom_nft`)
- explicit about optionality (`null` when unavailable)
- pagination-friendly
- compatible with Prism's humanized rendering

Recommended conventions:

- Base path: `/api/v1/nfts`
- Use cursor pagination where lists can grow large
- Include `contract_id`, `token_id`, `tx_hash`, `ledger_sequence`, and timestamps consistently
- Preserve normalized fields plus raw references where useful

---

# P0 / V1 required endpoints

## 1. Get NFT collection detail

**Endpoint**

`GET /api/v1/nfts/collections/{contract_id}`

**Purpose**

Returns normalized collection-level metadata and health.

**Gateway must provide**

- NFT classification
- collection metadata
- supply / holder summary
- TTL / health summary

**Suggested response**

```json
{
  "contract_id": "CA...",
  "is_nft": true,
  "standard": "sep_50",
  "implementation": "oz_non_fungible",
  "classification_confidence": "high",
  "classification_source": "spec_and_method_match",
  "verified": false,
  "name": "Stellar Geometries",
  "symbol": "SGEO",
  "description": "Generative on-chain geometric art...",
  "website": "https://example.com",
  "image_url": "ipfs://...",
  "banner_url": null,
  "metadata_storage": "ipfs",
  "admin": "G...",
  "deployed_at": "2026-01-02T03:04:05Z",
  "deployed_ledger": 123456,
  "minted_count": 256,
  "burned_count": 0,
  "active_count": 256,
  "holder_count": 142,
  "holder_pct": 55.47,
  "ttl": {
    "status": "healthy",
    "remaining_ledgers": 120000,
    "remaining_human": "120 days"
  }
}
```

**Notes**

- `holder_pct` can be omitted if Prism should compute it
- `verified` should be optional if no verification system exists yet

---

## 2. List tokens in a collection

**Endpoint**

`GET /api/v1/nfts/collections/{contract_id}/tokens?limit=24&cursor=...&sort=token_id&order=asc`

**Purpose**

Returns token cards for a collection page.

**Gateway must provide**

- token enumeration
- current owner
- metadata summary
- burned / active state
- pagination

**Suggested response**

```json
{
  "contract_id": "CA...",
  "count": 24,
  "next_cursor": "eyJ0b2tlbl9pZCI6IjI0In0=",
  "tokens": [
    {
      "token_id": "42",
      "name": "Convergence Point",
      "description": "A study in rotational symmetry...",
      "owner": "G...",
      "owner_label": "GABC...7X92",
      "image_url": "ipfs://...",
      "metadata_storage": "on_chain",
      "minted_at": "2026-08-12T10:00:00Z",
      "minted_ledger": 4102847,
      "last_transfer_at": "2026-10-18T09:00:00Z",
      "burned": false,
      "attributes": [
        {"trait_type": "Palette", "value": "Midnight Rose"}
      ]
    }
  ]
}
```

**Recommended query params**

- `limit`
- `cursor`
- `sort=token_id|minted_at|last_transfer_at`
- `order=asc|desc`
- `owner=G...` (optional filter)
- `burned=false` (optional filter)

---

## 3. Get token detail

**Endpoint**

`GET /api/v1/nfts/collections/{contract_id}/tokens/{token_id}`

**Purpose**

Returns a full NFT token detail record.

**Gateway must provide**

- token metadata
- current owner
- mint / transfer summary
- traits
- raw metadata references

**Suggested response**

```json
{
  "contract_id": "CA...",
  "token_id": "42",
  "name": "Convergence Point",
  "description": "A study in rotational symmetry and negative space.",
  "owner": "G...",
  "owner_label": "GABC...7X92",
  "approved": null,
  "burned": false,
  "image_url": "ipfs://...",
  "animation_url": null,
  "external_url": "https://example.com/token/42",
  "metadata_storage": "on_chain",
  "metadata_uri": "ipfs://.../42.json",
  "metadata_fetch_status": "ok",
  "attributes": [
    {"trait_type": "Palette", "value": "Midnight Rose", "rarity_pct": 12.0},
    {"trait_type": "Complexity", "value": "High", "rarity_pct": 8.0}
  ],
  "mint": {
    "tx_hash": "abc123...",
    "ledger_sequence": 4102847,
    "timestamp": "2026-08-12T10:00:00Z",
    "to": "G..."
  },
  "last_transfer": {
    "tx_hash": "def456...",
    "ledger_sequence": 4200000,
    "timestamp": "2026-10-18T09:00:00Z",
    "from": "G...",
    "to": "G..."
  }
}
```

---

## 4. Get collection activity

**Endpoint**

`GET /api/v1/nfts/collections/{contract_id}/activity?limit=50&cursor=...&type=transfer`

**Purpose**

Returns normalized collection activity for the Activity tab.

**Gateway must provide**

- mints
- transfers
- burns
- approvals
- operator approvals

**Suggested response**

```json
{
  "contract_id": "CA...",
  "count": 50,
  "next_cursor": "...",
  "events": [
    {
      "event_id": "evt_123",
      "type": "transfer",
      "token_id": "42",
      "from": "G...",
      "to": "G...",
      "operator": null,
      "approved": null,
      "tx_hash": "abc123...",
      "ledger_sequence": 4200000,
      "timestamp": "2026-10-18T09:00:00Z"
    },
    {
      "event_id": "evt_124",
      "type": "mint",
      "token_id": "43",
      "from": null,
      "to": "G...",
      "tx_hash": "def456...",
      "ledger_sequence": 4200001,
      "timestamp": "2026-10-18T09:00:05Z"
    }
  ]
}
```

**Recommended query params**

- `limit`
- `cursor`
- `type=mint|transfer|burn|approve|approve_for_all`
- `token_id`
- `account`
- `start_ledger`
- `end_ledger`

---

## 5. Get token provenance / history

**Endpoint**

`GET /api/v1/nfts/collections/{contract_id}/tokens/{token_id}/history`

**Purpose**

Returns full token history for a Provenance section.

**Suggested response**

```json
{
  "contract_id": "CA...",
  "token_id": "42",
  "events": [
    {
      "type": "mint",
      "from": null,
      "to": "G...",
      "tx_hash": "mint_tx...",
      "ledger_sequence": 4102847,
      "timestamp": "2026-08-12T10:00:00Z"
    },
    {
      "type": "transfer",
      "from": "G...",
      "to": "G...",
      "tx_hash": "sale_tx...",
      "ledger_sequence": 4200000,
      "timestamp": "2026-10-18T09:00:00Z"
    }
  ]
}
```

---

## 6. Get collection holders

**Endpoint**

`GET /api/v1/nfts/collections/{contract_id}/holders?limit=100&cursor=...`

**Purpose**

Returns holder list for the Holders tab.

**Suggested response**

```json
{
  "contract_id": "CA...",
  "count": 100,
  "next_cursor": "...",
  "holders": [
    {
      "address": "G...",
      "label": "GABC...7X92",
      "token_count": 4,
      "supply_pct": 1.56,
      "token_ids": ["42", "88", "107", "156"],
      "last_activity_at": "2026-10-18T09:00:00Z"
    }
  ]
}
```

---

## 7. Get NFTs owned by an account

**Endpoint**

`GET /api/v1/nfts/accounts/{address}/tokens?limit=24&cursor=...`

**Purpose**

Allows Prism account pages to show owned NFTs.

**Suggested response**

```json
{
  "address": "G...",
  "count": 24,
  "next_cursor": "...",
  "tokens": [
    {
      "contract_id": "CA...",
      "collection_name": "Stellar Geometries",
      "token_id": "42",
      "name": "Convergence Point",
      "image_url": "ipfs://...",
      "last_transfer_at": "2026-10-18T09:00:00Z"
    }
  ]
}
```

---

## 8. NFT-aware search

**Endpoint**

`GET /api/v1/nfts/search?q=stellar+geometries`

**Purpose**

Supports Prism discovery and global search integration.

**Suggested response**

```json
{
  "query": "stellar geometries",
  "collections": [
    {
      "contract_id": "CA...",
      "name": "Stellar Geometries",
      "symbol": "SGEO",
      "standard": "sep_50",
      "verified": false,
      "image_url": "ipfs://..."
    }
  ],
  "tokens": [],
  "accounts": []
}
```

---

# P1 recommended endpoints

## 9. List NFT collections

**Endpoint**

`GET /api/v1/nfts/collections?limit=24&cursor=...&sort=recent_activity`

**Purpose**

Supports a future `/nfts` index page.

**Suggested response**

```json
{
  "count": 24,
  "next_cursor": "...",
  "collections": [
    {
      "contract_id": "CA...",
      "name": "Stellar Geometries",
      "symbol": "SGEO",
      "verified": false,
      "standard": "sep_50",
      "image_url": "ipfs://...",
      "item_count": 256,
      "holder_count": 142,
      "recent_activity_at": "2026-10-18T09:00:00Z"
    }
  ]
}
```

**Useful query params**

- `verified=true`
- `standard=sep_50`
- `sort=recent_activity|holder_count|minted_count`

---

## 10. Get NFT contract facts

**Endpoint**

`GET /api/v1/nfts/collections/{contract_id}/contract`

**Purpose**

Supports the Contract tab with lower-level facts.

**Suggested response**

```json
{
  "contract_id": "CA...",
  "standard": "sep_50",
  "implementation": "oz_non_fungible",
  "admin": "G...",
  "code_hash": "...",
  "deployed_ledger": 123456,
  "deployed_tx_hash": "...",
  "methods": [
    "balance",
    "owner_of",
    "transfer",
    "transfer_from",
    "approve",
    "approve_for_all",
    "get_approved",
    "is_approved_for_all"
  ],
  "event_types": [
    "transfer",
    "approve",
    "approve_for_all"
  ],
  "ttl": {
    "status": "healthy",
    "remaining_ledgers": 120000
  }
}
```

---

## 11. Get account NFT activity

**Endpoint**

`GET /api/v1/nfts/accounts/{address}/activity?limit=50&cursor=...`

**Purpose**

Shows NFT actions involving a given account.

---

## 12. Trait summary for a collection

**Endpoint**

`GET /api/v1/nfts/collections/{contract_id}/traits`

**Purpose**

Supports trait filters and rarity UI.

**Suggested response**

```json
{
  "contract_id": "CA...",
  "traits": [
    {
      "trait_type": "Palette",
      "values": [
        {"value": "Midnight Rose", "count": 31, "pct": 12.11},
        {"value": "Solar Gold", "count": 18, "pct": 7.03}
      ]
    }
  ]
}
```

---

# P2 optional marketplace / advanced endpoints

These require more than SEP-50. They likely require marketplace-specific indexing.

## 13. Get collection market stats

**Endpoint**

`GET /api/v1/nfts/collections/{contract_id}/market`

**Purpose**

Returns collection-level market analytics.

**Suggested response**

```json
{
  "contract_id": "CA...",
  "floor_price": "1200",
  "floor_currency": "XLM",
  "listed_count": 12,
  "total_volume": "84200",
  "volume_currency": "XLM",
  "last_sale_price": "1800",
  "last_sale_at": "2026-10-18T09:00:00Z"
}
```

---

## 14. Get token market data

**Endpoint**

`GET /api/v1/nfts/collections/{contract_id}/tokens/{token_id}/market`

**Purpose**

Returns per-token listing / sale info.

---

## 15. Get collection sales history

**Endpoint**

`GET /api/v1/nfts/collections/{contract_id}/sales?limit=50&cursor=...`

**Purpose**

Returns normalized sales, distinct from plain transfers.

---

# Cross-cutting response requirements

Every NFT endpoint should follow these rules.

## A. Consistent identifiers

Include where relevant:

- `contract_id`
- `token_id`
- `tx_hash`
- `ledger_sequence`
- `timestamp`
- `event_id`

## B. Explicit nullability

If a field is unavailable, prefer `null` over inventing placeholder strings.

Examples:
- `approved: null`
- `banner_url: null`
- `animation_url: null`

## C. Pagination

List endpoints should include:

- `count`
- `next_cursor`

## D. Classification fields

Where useful, include:

- `is_nft`
- `standard`
- `implementation`
- `classification_confidence`
- `classification_source`

## E. Metadata status fields

Where metadata is fetched externally, include:

- `metadata_storage`
- `metadata_uri`
- `metadata_fetch_status`
- optionally `metadata_error`

## F. Health / TTL fields

Where a collection / contract is involved, include:

- `ttl.status`
- `ttl.remaining_ledgers`
- optionally `ttl.remaining_human`

---

# Minimum implementation order

If Gateway is implementing this incrementally, the recommended order is:

## Phase 1

1. `GET /api/v1/nfts/collections/{contract_id}`
2. `GET /api/v1/nfts/collections/{contract_id}/tokens`
3. `GET /api/v1/nfts/collections/{contract_id}/tokens/{token_id}`
4. `GET /api/v1/nfts/collections/{contract_id}/activity`
5. `GET /api/v1/nfts/collections/{contract_id}/holders`

## Phase 2

6. `GET /api/v1/nfts/accounts/{address}/tokens`
7. `GET /api/v1/nfts/accounts/{address}/activity`
8. `GET /api/v1/nfts/search`
9. `GET /api/v1/nfts/collections`
10. `GET /api/v1/nfts/collections/{contract_id}/contract`
11. `GET /api/v1/nfts/collections/{contract_id}/traits`
12. `GET /api/v1/nfts/collections/{contract_id}/tokens/{token_id}/history`

## Phase 3

13. `GET /api/v1/nfts/collections/{contract_id}/market`
14. `GET /api/v1/nfts/collections/{contract_id}/tokens/{token_id}/market`
15. `GET /api/v1/nfts/collections/{contract_id}/sales`

---

# Prism mapping

These endpoints map cleanly to Prism needs:

- **Collection page**
  - collection detail
  - collection tokens
  - collection activity
  - collection holders
  - contract facts

- **Token detail page**
  - token detail
  - token history
  - optional token market data

- **Account page**
  - account-owned NFTs
  - account NFT activity

- **Search / discovery**
  - NFT search
  - collections index

---

# Bottom line

If you only build the P0 endpoints, Prism can already ship a real NFT collection and token explorer.

If you add P1, the experience becomes much more complete.

If you add P2, you can support marketplace-style metrics like floor and volume.
