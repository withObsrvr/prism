# Prism Humanized Explorer: Technical Implementation Spec

Related docs:
- [Product Spec](./prism-humanized-product-spec.md)

## 1. Purpose

This document translates the humanized Prism product spec into an implementation plan for the current Prism codebase.

It defines:
- backend/frontend ownership boundaries
- internal view models
- template and narrator architecture
- endpoint-to-component mapping
- phased implementation plan
- acceptance criteria

This spec assumes the architectural rule established in the product spec:

- **Gateway/API owns Layer 1 classification and Layer 2 structured enrichment**
- **Prism owns Layer 3 narration and presentation**

---

## 2. Current codebase baseline

Relevant existing areas in Prism:

- `internal/handlers/`
  - route handlers
  - page data assembly
  - live-data fallback behavior
- `internal/gateway/`
  - API client methods
  - gateway response types
  - shared format helpers
- `internal/templates/pages/`
  - page templates
- `internal/templates/fragments/`
  - HTMX fragments
- `internal/templates/components/`
  - shared UI components

Current relevant pages/routes:
- `/tx/{hash}`
- `/contracts/{id}`
- `/account/{id}`
- `/account/{id}/smart`

Current relevant live integrations already present:
- contract metadata / analytics / storage
- semantic contracts registry
- smart wallet detection
- existing transaction/detail endpoints

Missing key live integration for the humanized transaction page:
- `/api/v1/silver/tx/{hash}/semantic`

---

## 3. Implementation goals

### Primary technical goals
1. Introduce a deterministic narration layer in Prism.
2. Keep raw explorer data accessible and unchanged underneath.
3. Avoid any passive LLM dependency.
4. Make semantic page rendering composable and testable.
5. Allow gradual migration page-by-page without full rewrites.

### Non-goals
1. Rewriting the entire routing system.
2. Embedding prose generation in the gateway.
3. Removing existing raw tables or explorer affordances.
4. Building a generic rule engine in Prism for Layer 1 classification.

---

## 4. Ownership boundaries

## 4.1 Gateway/API responsibilities
Gateway should return structured facts only.

This includes:
- transaction classification
- subtype and confidence
- actor identities and roles
- asset metadata
- operation/event summaries
- smart-wallet provenance
- contract metadata
- contract activity analytics
- storage summaries
- optional deep diffs/call graph

Gateway should **not** return:
- UI-specific narrative strings
- presentational titles
- warning copy
- recommended user next steps

## 4.2 Prism responsibilities
Prism should:
- normalize gateway responses into page-specific view models
- select deterministic templates
- render titles, summaries, signal descriptions, and evidence blocks
- control progressive disclosure
- preserve raw evidence beneath the narrative layer

---

## 5. Proposed implementation architecture

Use a 4-step Prism-side pipeline for humanized pages:

1. **Fetch** structured semantic/enriched API payloads
2. **Normalize** them into a Prism-owned internal model
3. **Narrate** using a deterministic template registry
4. **Render** into page templates/components

### Core principle
Prism should not narrate directly from raw JSON in page templates.

Instead:
- handlers fetch API data
- a mapper converts API shapes into a stable internal model
- a narrator package produces human-ready fields
- templates stay dumb and declarative

---

## 6. New package/module structure

Recommended additions:

```text
internal/
  humanize/
    tx.go
    contract.go
    signals.go
    evidence.go
    templates/
      tx_account_funding.go
      tx_contract_call.go
      tx_smart_wallet_policy_update.go
      tx_smart_wallet_transfer.go
      tx_smart_wallet_swap.go
      tx_generic.go
```

### Responsibilities
- `internal/humanize/tx.go`
  - tx semantic normalization
  - narrator entrypoints
- `internal/humanize/contract.go`
  - contract metadata/analytics/storage normalization
  - contract narrative helpers
- `internal/humanize/signals.go`
  - UI signal derivation from structured fields
- `internal/humanize/evidence.go`
  - explanation/evidence formatting helpers
- `internal/humanize/templates/*`
  - deterministic narrative templates per semantic type

Alternative if you want smaller footprint initially:
- start with a single `internal/humanize/tx.go`
- split once transaction page stabilizes

---

## 7. New gateway client methods and types

## 7.1 Required additions
Add a dedicated semantic tx client method.

### New method
```go
GetSemanticTransaction(ctx context.Context, network, hash string, includeDeep bool) (*SemanticTransactionResponse, error)
```

### Endpoint
- `GET /api/v1/silver/tx/{hash}/semantic`
- optional query param: `include_deep=true`

## 7.2 New gateway types
Add new types in `internal/gateway/types.go` for the semantic tx payload.

Minimum initial response model:

```go
type SemanticTransactionResponse struct {
    Transaction   SemanticTransactionMeta   `json:"transaction"`
    Classification SemanticClassification   `json:"classification"`
    Actors        []SemanticActor           `json:"actors"`
    Assets        []SemanticAsset           `json:"assets"`
    Operations    []SemanticOperation       `json:"operations"`
    Events        []SemanticEvent           `json:"events"`
    Diffs         []SemanticDiff            `json:"diffs,omitempty"`
    CallGraph     *SemanticCallGraph        `json:"call_graph,omitempty"`
    LegacySummary string                    `json:"legacy_summary,omitempty"`
}
```

Add concrete structs matching actual gateway payloads once confirmed.

### Important rule
Do not overfit types to presentation.
These types should mirror API structure closely.

---

## 8. Internal Prism view models

Prism should own a stable set of humanized view models that templates consume.

## 8.1 Transaction page view model

```go
type HumanTxPageData struct {
    Hash            string
    Title           string
    Subtitle        string
    Narrative       string
    Status          string
    StatusColor     string
    ConfidenceLabel string
    ConfidenceValue string
    TxType          string
    Subtype         string
    Timestamp       string
    Ledger          string

    Actors          []HumanActor
    Assets          []HumanAssetFlow
    Signals         []HumanSignal
    Evidence        []HumanEvidence
    OperationGroups []HumanOperationGroup
    Events          []HumanEvent

    Raw             HumanTxRawData
}
```

Supporting types:

```go
type HumanActor struct {
    Label      string
    Address    string
    ShortAddr  string
    Roles      []string
    ActorType  string
    Badge      string
    Href       string
}

type HumanAssetFlow struct {
    Direction string
    Amount    string
    Asset     string
    Counterparty string
    Note      string
}

type HumanSignal struct {
    ID          string
    Title       string
    Severity    string
    Summary     string
    Reason      string
    ActionLabel string
    ActionHref  string
}

type HumanEvidence struct {
    Label string
    Value string
    Note  string
}
```

## 8.2 Contract page view model

Current `pages.ContractDetailData` can be extended or wrapped.

Preferred long-term direction:

```go
type HumanContractPageData struct {
    Header          HumanContractHeader
    Narrative       string
    Context         string
    Signals         []HumanSignal
    Profile         []HumanKeyValue
    FunctionSummary string
    Functions       []HumanContractFunction
    StorageSummary  string
    StorageItems    []HumanStorageItem
    Invocations     []HumanInvocation
    Raw             HumanContractRawData
}
```

Short-term pragmatic direction:
- keep `pages.ContractDetailData`
- add narrative-specific fields to it only where necessary
- avoid giant rewrites during Phase 2

---

## 9. Narration engine design

## 9.1 Transaction narrator interface

```go
type TxNarrator interface {
    Matches(tx SemanticTxContext) bool
    Build(tx SemanticTxContext) HumanTxNarrative
}

type HumanTxNarrative struct {
    Title     string
    Subtitle  string
    Narrative string
    Signals   []HumanSignal
    Evidence  []HumanEvidence
}
```

## 9.2 SemanticTxContext
Prism normalization layer should convert gateway response into a simpler semantic context before template matching.

```go
type SemanticTxContext struct {
    Hash              string
    TxType            string
    Subtype           string
    Confidence        float64
    Status            string
    Ledger            int64
    Timestamp         string
    WalletInvolved    bool
    EffectiveActorType string

    Actors            []HumanActor
    Assets            []HumanAssetFlow
    Operations        []SemanticOperation
    Events            []SemanticEvent
    LegacySummary     string
}
```

## 9.3 Template registry
Use ordered matching.

```go
var txNarrators = []TxNarrator{
    SmartWalletPolicyUpdateNarrator{},
    SmartWalletSwapNarrator{},
    SmartWalletTransferNarrator{},
    AccountFundingNarrator{},
    ContractCallNarrator{},
    GenericTxNarrator{},
}
```

Matching strategy:
- first explicit semantic tx type match
- then subtype-aware match
- then fallback

## 9.4 Initial template set
Implement these first:
- `smart_wallet_policy_update`
- `smart_wallet_multisig_approval`
- `smart_wallet_transfer`
- `smart_wallet_swap`
- `wallet_mediated_protocol_interaction`
- `account_funding`
- `contract_call`
- generic fallback

---

## 10. Signal framework

## 10.1 Philosophy
Signals in Prism are deterministic UI insights derived from existing structured fields.

They are not ML scores.
They should be explainable in one sentence.

## 10.2 Initial implementation approach
Start with lightweight Prism-side signal builders.

### Transaction signals
- signer/policy change
- smart-wallet activity
- transfer-like movement
- swap-like activity
- failed tx
- first-seen contract call if enough context exists

### Contract signals
- token-like function surface
- low usage / recent deployment
- storage-light / storage-heavy heuristics based on current totals

### Future migration path
If gateway later exposes anomaly flags, Prism should consume them and only render framing copy.

## 10.3 API for signals
```go
func BuildTxSignals(ctx SemanticTxContext) []HumanSignal
func BuildContractSignals(ctx HumanizedContractContext) []HumanSignal
```

---

## 11. Evidence framework

Every narrative should support an evidence panel.

## 11.1 Evidence source mapping
Transaction evidence should be assembled from:
- `classification.tx_type`
- `classification.subtype`
- `classification.confidence`
- `classification.operation_types`
- `classification.wallet_involved`
- actor roles/types
- observed assets/events if relevant

## 11.2 Evidence builder
```go
func BuildTxEvidence(ctx SemanticTxContext) []HumanEvidence
```

Example outputs:
- `Type` → `smart_wallet_policy_update`
- `Subtype` → `allow_signing_key`
- `Confidence` → `0.91`
- `Wallet involved` → `Yes`
- `Effective actor type` → `smart_wallet`

---

## 12. UI component implementation plan

Add new reusable components in `internal/templates/components/`.

## 12.1 Components to add
- `narrative_header.templ`
- `confidence_badge.templ`
- `actor_chips.templ`
- `signal_cards.templ`
- `evidence_panel.templ`
- `suggested_next_steps.templ`

## 12.2 Component responsibilities

### `NarrativeHeader`
Inputs:
- title
- subtitle
- narrative
- status
- confidence badge
- actor chips

### `ConfidenceBadge`
Inputs:
- label
- tooltip text
- strength/color

### `ActorChips`
Inputs:
- `[]HumanActor`
Output:
- compact inline cast list for hero/header

### `SignalCards`
Inputs:
- `[]HumanSignal`
Output:
- stacked or grid cards with reason and CTA

### `EvidencePanel`
Inputs:
- `[]HumanEvidence`
Output:
- key-value evidence list

---

## 13. Transaction page implementation details

## 13.1 Handler changes
Likely file:
- `internal/handlers/explorer.go`
- or wherever `/tx/{hash}` is currently assembled

### New helper
```go
func (h *Handlers) buildHumanTransactionData(r *http.Request, network, hash string) (pages.TransactionHumanizedData, error)
```

Suggested pipeline inside handler:
1. fetch semantic tx from gateway
2. normalize to `SemanticTxContext`
3. run narrator registry
4. build signals and evidence
5. map into page data struct
6. render page with humanized sections above raw sections

## 13.2 Page template strategy
Prefer incremental change over full replacement.

### Recommended approach
- keep existing transaction receipt page
- add a new semantic/humanized block at top
- leave raw receipt below it

This reduces risk and preserves current functionality.

## 13.3 New page data struct
If current tx page data is too raw-oriented, add a wrapper:

```go
type TransactionHumanizedData struct {
    Human *HumanTxPageData
    Raw   ExistingTransactionReceiptData
}
```

---

## 14. Contract page implementation details

## 14.1 Current state
Contract page already has live integration for:
- metadata
- analytics
- storage preview
- recent calls

## 14.2 Next step
Add humanized sections on top of existing contract page.

### New fields to add to `pages.ContractDetailData`
Only if using incremental path:
- `Narrative string`
- `Context string`
- `Signals []HumanSignal` (or page-specific simpler type)
- `FunctionSummary string`
- `StorageSummary string`

Alternative:
- create `HumanContractPageData`
- migrate template in one pass later

## 14.3 Contract narrative builder
Add:
```go
func BuildHumanContractNarrative(meta *gateway.ContractMetadata, analytics *gateway.ContractAnalytics) (narrative string, context string)
```

Initial heuristics:
- token-like if exported functions include combinations such as `approve`, `transfer`, `transfer_from`
- recent deployment if created recently and activity low
- low usage if invocation count low
- active contract if recent call count and recent activity present

## 14.4 Storage story builder
Add:
```go
func BuildContractStorageSummary(meta *gateway.ContractMetadata, storage *gateway.ContractStorageResponse) string
```

Examples:
- “This contract currently has 3 storage entries using about 164 KB of state.”
- “Most visible storage is persistent.”

---

## 15. Smart wallet page implementation details

## 15.1 Existing opportunity
Prism already has smart-wallet detection hooks and a dormant smart account route.

## 15.2 Required work
- gateway client types for `/api/v1/silver/smart-wallets` if not present
- page data model for wallet family, provenance, signer behavior
- policy timeline assembled from semantic tx classifications

## 15.3 Suggested order
Do not start here first.
Build only after semantic tx narration is stable.

---

## 16. Routing and feature flag strategy

## 16.1 Default strategy
Release incrementally behind existing live-data gating.

## 16.2 Optional feature flag
Add a Prism-side feature flag if needed:
- `PRISM_ENABLE_HUMANIZED_TX=true`
- `PRISM_ENABLE_HUMANIZED_CONTRACT=true`

This is optional but useful during rollout.

## 16.3 Fallback behavior
If semantic endpoint fails:
- fall back to current raw tx page behavior
- log warning
- never break basic explorer functionality

---

## 17. Caching and performance

## 17.1 Gateway client caching
Use the existing gateway cache mechanism.

Recommended TTLs:
- semantic tx, lean mode: `TTLRecentList` for very recent txs, immutable TTL for older txs if desired later
- contract metadata: `TTLContracts`
- deep semantic tx: slightly longer cache is acceptable

## 17.2 HTMX / progressive loading
Recommended:
- tx hero + evidence render on initial page load
- optional deep diffs/call graph fetched lazily
- contract storage/details remain fragment-loadable

## 17.3 Performance constraint
Humanized layer must add only lightweight string construction and list mapping.
No expensive recomputation inside templates.

---

## 18. Testing strategy

## 18.1 Unit tests
Add tests for:
- semantic tx normalization
- narrator matching order
- template rendering for each supported tx type
- signal generation
- evidence generation
- contract narrative heuristics

Recommended package:
- `internal/humanize/...`

## 18.2 Golden tests
Use golden tests for narrative outputs when practical.

Example:
- input semantic tx JSON fixture
- expected title/narrative/subtitle/signals/evidence

## 18.3 Handler tests
Add handler tests for:
- semantic page render success
- gateway failure fallback
- partial data fallback

## 18.4 Manual QA cases
Minimum manual set:
- smart wallet policy update tx
- classic account funding tx
- smart wallet transfer tx
- generic contract call tx
- token-like contract page
- newly deployed low-usage contract page

---

## 19. Acceptance criteria by phase

## Phase 1: Humanized transaction page
Done when:
- semantic tx client method exists
- tx page shows title, narrative, actor chips, evidence panel
- raw transaction details still visible
- unsupported semantic types fall back cleanly
- no LLM dependency introduced

## Phase 2: Humanized contract page
Done when:
- contract page shows “what this contract is” section
- function surface summary exists
- storage summary is human-readable
- raw/storage/function details still available

## Phase 3: Smart wallet UX
Done when:
- smart wallet provenance visible
- signer/policy-related txs are humanized consistently
- smart account page no longer requires mock-first behavior

---

## 20. Suggested phased engineering plan

## Sprint 1
- add semantic tx gateway client/types
- add `internal/humanize/tx.go`
- add first narrator templates
- add unit tests for normalization + narration

## Sprint 2
- add transaction hero/evidence components
- integrate humanized tx section into existing tx page
- preserve raw tabs/content
- add fallbacks and logging

## Sprint 3
- add contract narrative builder
- add contract narrative/storage summary components
- wire into current contract page

## Sprint 4
- add signals framework
- add suggested next steps
- start smart wallet page upgrade path

---

## 21. Concrete first files to touch

### Gateway
- `internal/gateway/types.go`
- `internal/gateway/client.go`

### Humanization layer
- `internal/humanize/tx.go`
- `internal/humanize/contract.go`
- `internal/humanize/signals.go`
- `internal/humanize/evidence.go`

### Handlers
- existing tx handler file
- `internal/handlers/contracts.go`

### Templates
- `internal/templates/components/narrative_header.templ`
- `internal/templates/components/confidence_badge.templ`
- `internal/templates/components/actor_chips.templ`
- `internal/templates/components/signal_cards.templ`
- `internal/templates/components/evidence_panel.templ`
- existing tx page template
- existing contract page template

---

## 22. Risks and mitigations

## Risk: semantic endpoint shape evolves
Mitigation:
- isolate API types from Prism internal normalized models
- avoid using raw API structs directly in templates

## Risk: narrative logic sprawls into handlers
Mitigation:
- centralize in `internal/humanize`
- keep handlers orchestration-only

## Risk: page templates become too stateful
Mitigation:
- precompute narrative, signals, evidence, labels in Go
- keep templates mostly declarative

## Risk: low-confidence outputs feel misleading
Mitigation:
- always show confidence/provenance
- include evidence panel
- use conservative fallback wording

## Risk: rollout destabilizes current explorer pages
Mitigation:
- layer humanized UI on top of existing raw views
- retain current fallback behavior
- gate if necessary

---

## 23. Recommended first implementation decision

For the current Prism codebase, the best immediate path is:

1. **Do transaction humanization first**
2. **Implement a small Prism-owned narrator package**
3. **Integrate it into the existing tx page without removing raw sections**
4. **Then apply the same pattern to contracts**

This yields maximum product differentiation with minimum architectural risk.

---

## 24. Final engineering principle

Prism should treat semantic API responses as a structured intermediate language.

The implementation pattern should be:

**gateway JSON → Prism normalization → deterministic narration → presentational components**

That keeps the system:
- testable
- explainable
- cheap to operate
- resilient to API evolution
- aligned with the humanized product vision
