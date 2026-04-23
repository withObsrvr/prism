# NFT Gateway Requirements

This document lists everything Gateway needs to do for NFT support in Prism.

## 1. Detect and classify NFT contracts

Gateway needs to:

- identify contracts that implement **SEP-50**
- identify contracts that are **NFT-like** even if not perfectly standard
- detect known implementations like **OpenZeppelin Non-Fungible**
- expose:
  - `is_nft`
  - `standard` (`SEP-50`, `custom`, etc.)
  - `implementation` (`OpenZeppelin`, `unknown`, etc.)
  - confidence / classification source if useful

This prevents Prism from doing contract introspection itself.

---

## 2. Extract and normalize collection-level metadata

Gateway needs to return normalized collection metadata such as:

- contract id
- collection name
- symbol
- description
- standard
- implementation
- website
- image / banner / icon
- creator / admin / owner if available
- verification status if supported
- metadata storage type:
  - on-chain
  - IPFS
  - HTTPS
  - Arweave
  - unknown

If metadata is spread across:
- contract methods
- contract storage
- metadata URIs
- external JSON

Gateway should unify that into one clean shape.

---

## 3. Enumerate all tokens in a collection

Gateway needs to support token discovery and listing:

- list all token IDs in a collection
- support pagination
- support sorting:
  - token id
  - recently minted
  - recently transferred
  - optionally rarity / price / last sale later
- include burned / active status
- include mint timestamp / ledger if known

This is core to making a collection page work.

---

## 4. Resolve current ownership

Gateway needs to know, per token:

- current owner
- whether token exists
- whether token is burned
- current approved address, if relevant
- operator approvals, if relevant

And collection-wide:

- holder count
- balances by holder
- token ownership distribution

Prism should not reconstruct ownership by replaying events.

---

## 5. Index NFT events

Gateway needs to index and normalize SEP-50 NFT events:

- `transfer`
- `approve`
- `approve_for_all`

And derive NFT lifecycle actions such as:

- mint
- transfer
- burn
- approval
- operator approval revoke/grant

For each event/activity row, Gateway should expose:

- event type
- contract id
- token id
- from
- to
- operator / approved account where relevant
- tx hash
- ledger
- timestamp
- raw event reference if needed

---

## 6. Decode NFT-related contract calls

Gateway needs to decode calls like:

- `transfer`
- `transfer_from`
- `approve`
- `approve_for_all`
- `owner_of`
- `balance`
- implementation-specific mint / burn methods

This is needed so Prism can humanize transactions and show meaningful activity.

Gateway should expose semantic call classification like:

- `nft_mint`
- `nft_transfer`
- `nft_burn`
- `nft_approve`
- `nft_approve_for_all`

---

## 7. Build per-token detail

Gateway needs token detail endpoints that return:

- token id
- contract id
- collection name
- token name
- token description
- token image / media URL
- animation URL if present
- metadata storage type
- owner
- approved address
- mint info:
  - tx hash
  - ledger
  - timestamp
- last transfer info
- traits / attributes
- raw metadata URI
- parsed metadata JSON if available
- burned status

This is what Prism would render on a token detail page.

---

## 8. Build collection activity feeds

Gateway needs collection-level activity feeds:

- recent mints
- recent transfers
- recent burns
- recent approvals
- recent operator approvals

This should be queryable by:
- collection
- token
- account / holder
- time range
- ledger range

Prism can then render tabs like Activity and Provenance.

---

## 9. Build per-token provenance/history

Gateway needs to provide complete token history:

- mint event
- all transfers
- burns
- relevant approvals if you want them shown

Each provenance row should include:

- action
- from / to
- date/time
- tx hash
- ledger
- detail fields for rendering

That powers the “Provenance” section in the mock.

---

## 10. Compute holder analytics

Gateway needs to compute holder stats such as:

- unique holder count
- percent unique ownership
- tokens per holder
- top holders
- holder concentration
- possibly distribution buckets

These are derived/indexed stats, not UI work.

---

## 11. Expose collection stats

Gateway should provide:

- total minted
- total burned
- active supply
- holder count
- collection size
- recent transfer count
- recent mint count

Optional later:
- daily active traders
- velocity / turnover
- rarity distribution

---

## 12. Resolve metadata from external sources

If NFT metadata is external, Gateway needs to fetch and normalize it.

That includes support for:
- IPFS
- HTTPS metadata JSON
- Arweave
- on-chain metadata blobs if present

Gateway should parse common NFT metadata fields like:

- name
- description
- image
- animation_url
- external_url
- attributes / traits
- background_color
- properties

And expose:
- normalized parsed fields
- raw URI/source
- fetch status / parse status

Prism should not be responsible for metadata crawling.

---

## 13. Handle broken or partial metadata

Gateway needs graceful behavior for:

- invalid metadata URI
- unreachable IPFS/HTTP asset
- malformed JSON
- missing image
- missing traits
- inconsistent token names/descriptions

It should still return a usable object with:

- status flags
- null/optional fields
- metadata error indicators if useful

---

## 14. Provide TTL / rent / health information

Since Prism cares about TTL, Gateway needs NFT collection contract health data:

- contract TTL remaining
- storage TTL remaining if relevant
- health status
- restore-needed risk
- extend-needed soon flags
- human-friendly ledger counts or raw remaining ledgers

Prism can convert this into labels like:
- Healthy
- Expiring soon
- At risk

But Gateway must provide the underlying values.

---

## 15. Support search/discovery for NFTs

Gateway needs NFT-aware search support:

- search NFT collections by name
- search by contract id
- search by symbol
- search by token id within collection
- optionally search holders / creators

This helps Prism integrate NFTs into global explorer search.

---

## 16. Support account-level NFT views

Gateway should let Prism ask:

- which NFTs does this account own?
- which NFT collections has this account interacted with?
- recent NFT transfers for this account
- approvals granted by this account
- approvals granted to this account

This is important for wallet/account pages later.

---

## 17. Support collection holder lists

Gateway should provide a holders endpoint per collection:

- holder address
- token count
- optionally token ids owned
- percent of supply
- last activity

This supports the “Holders” tab in the mock page.

---

## 18. Support contract tab data

For the “Contract” tab, Gateway should expose:

- contract id
- standard
- implementation
- admin / owner if known
- metadata source
- methods detected
- emitted event types detected
- code hash / wasm hash if useful
- deployment ledger / tx hash
- TTL / storage status

Prism can then render a clean contract facts panel.

---

## 19. Support semantic normalization for Prism

Gateway should normalize NFT concepts into stable explorer-facing types.

Examples:
- activity types:
  - `mint`
  - `transfer`
  - `burn`
  - `approve`
  - `approve_for_all`
- storage kinds:
  - `on_chain`
  - `ipfs`
  - `https`
  - `arweave`
- standards:
  - `sep_50`
  - `oz_non_fungible`
  - `custom_nft`

This keeps Prism from containing protocol-specific branching everywhere.

---

## 20. Optionally provide marketplace analytics

This is optional, but needed if you want the mock’s market fields.

Gateway would need to provide:
- floor price
- listed count
- total volume
- last sale
- collection sales history
- token last sale
- active listings

Important: SEP-50 alone does **not** give you this.
To support it, Gateway would also need to:
- index marketplace contracts
- detect listing events
- detect sale events
- map marketplace sales back to NFT collections/tokens

If you do not build marketplace indexing yet, these fields should be optional or absent.

---

## 21. Support verification / identity mapping

If Prism wants “Verified” badges, Gateway needs to provide:

- collection verification status
- verified creator / issuer identity
- protocol/project mapping
- known project names/logos

This likely comes from:
- curated registry
- semantic contract registry
- manual metadata

---

## 22. Support media delivery strategy

Gateway should decide how media URLs are exposed:

- raw IPFS/HTTP URLs only
- normalized gateway URLs
- cached/proxied image URLs
- thumbnail URLs if generated

This matters because Prism should ideally not need custom logic for every metadata storage backend.

---

## 23. Provide pagination/cursors everywhere needed

Gateway needs pagination for:

- collections index
- tokens in collection
- activity feed
- holders list
- account-owned NFTs

Cursor-based pagination is preferable.

---

## 24. Provide filtering and sorting support

Gateway should support server-side filtering/sorting for NFT data:

### Collections
- verified only
- standard
- recently active
- holder count
- volume/floor if supported

### Tokens
- owner
- burned / active
- trait filters
- recently minted
- recently transferred

### Activity
- type
- token id
- account
- time range

This avoids Prism doing everything client-side.

---

## 25. Provide stable API types tailored for Prism

Gateway should expose stable NFT API responses, not raw event blobs.

At minimum, Gateway needs endpoints for:

- collection detail
- collection tokens
- token detail
- collection activity
- token provenance
- collection holders
- account-owned NFTs
- NFT-aware search
- optional market stats

---

## 26. Handle non-standard but common NFT contracts

Because SEP-50 is still draft, Gateway should also:

- support OpenZeppelin contracts directly
- support custom contracts that are clearly NFT collections
- expose whether support is:
  - strict standard
  - heuristic
  - partial

This makes Prism useful before the whole ecosystem fully converges.

---

## 27. Preserve raw references for drill-down

Even after normalization, Gateway should preserve enough raw references for explorer drill-down:

- tx hash
- ledger sequence
- event id
- contract id
- token id
- metadata URI
- raw event topics/data if needed for debugging

Prism may need these for deeper technical pages.

---

## 28. Expose freshness / sync state

Gateway should ideally expose whether NFT data is:

- fully indexed
- partially indexed
- metadata still resolving
- stale / delayed

Especially for external metadata and derived analytics.

---

## 29. Support cross-linking into existing Prism concepts

Gateway should make NFT data easy to connect with existing Prism entities:

- accounts
- contracts
- ledgers
- transactions
- events

That means every NFT object should include IDs usable across the rest of the explorer.

---

## 30. Provide enough data for human-readable summaries

To support Prism’s “humanized first” experience, Gateway needs to include enough facts for sentence generation:

For example:
- token id
- collection name
- sender
- recipient
- mint/burn/transfer type
- approved account
- operator
- ledger/time
- tx hash

So Prism can say:
- “Minted Convergence Point (#42) to GABC…”
- “Transferred NFT #107 from GAAA… to GBBB…”
- “Approved GCCC… to transfer NFT #12”

---

## Minimum Gateway deliverables for V1

If you want the smallest useful NFT implementation, Gateway at minimum needs to do:

1. classify NFT contracts
2. return collection metadata
3. enumerate tokens
4. resolve current owner per token
5. return token detail
6. index mint/transfer/burn activity
7. return collection activity/provenance
8. compute holder count
9. provide TTL/rent status
10. normalize metadata from on-chain/IPFS/HTTP

Optional for later:
- floor price
- total volume
- listings
- sales
- verification
- rarity
