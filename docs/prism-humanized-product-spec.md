# Prism Product Spec: Humanized Block Explorer

Related docs:
- [Technical Implementation Spec](./prism-humanized-technical-implementation-spec.md)

## 1. Product summary

Prism is a block explorer that turns raw blockchain data into understandable, trustworthy stories.

Traditional explorers optimize for:
- hashes
- ledgers
- raw operations
- protocol-native terminology

Prism optimizes for:
- what happened
- who was involved
- why it matters
- what looks unusual
- what to inspect next

### Core principle
**API returns facts. Prism tells the story.**

- **Obsrvr/Gateway**
  - Layer 1: deterministic classification
  - Layer 2: structured enrichment
- **Prism**
  - Layer 3: narration, framing, progressive disclosure

No passive LLM dependency is required.

---

# 2. Product goals

## Primary goals
1. Make blockchain activity understandable to non-experts.
2. Preserve trust by keeping all raw evidence accessible.
3. Turn transactions, contracts, and wallets into narrative objects, not just records.
4. Differentiate Prism through semantic understanding and actor-centric views.
5. Support both casual viewers and expert auditors on the same page.

## Non-goals
1. Replacing raw explorer functionality.
2. Generating freeform AI prose on every page load.
3. Hiding uncertainty or heuristics.
4. Embedding presentation-specific prose into API responses.

---

# 3. Target users

## Primary users
### A. Curious users
People who want to understand what happened without protocol expertise.

Needs:
- plain-language summaries
- recognizable labels
- low cognitive load

### B. Operators / builders
Founders, PMs, protocol teams, wallet builders.

Needs:
- protocol activity understanding
- contract behavior summaries
- deploy and usage context
- actor relationships

### C. Investigators / compliance / analysts
Need:
- anomaly clues
- behavior patterns
- actor roles
- evidence and auditability

### D. Power users
Need:
- raw operations
- events
- diffs
- storage
- exact transaction metadata

---

# 4. Product thesis

A humanized explorer should answer these questions in order:

1. **What happened?**
2. **Who was involved?**
3. **Why does Prism think that?**
4. **What stands out?**
5. **What should I inspect next?**
6. **What are the raw on-chain details?**

This ordering drives every page design.

---

# 5. Information architecture

## Primary surfaces
1. Home
2. Search
3. Transaction detail
4. Contract detail
5. Account detail
6. Smart wallet detail
7. Protocol / entity views later
8. Network / devtools remain secondary

## Humanization model by page
### Transaction page
Narrative-first event explanation.

### Contract page
Software profile + behavioral context.

### Account page
Behavior and relationship profile.

### Smart wallet page
Policy, signer, and automation understanding.

---

# 6. Product principles

## 6.1 Plain-language first
Lead with a sentence, not a hash.

## 6.2 Facts before flourish
Narration is deterministic template rendering from structured inputs.

## 6.3 Evidence always available
Every classification and signal needs drill-down evidence.

## 6.4 Confidence is visible
Heuristic vs classified vs inferred should be shown.

## 6.5 Progressive disclosure
Friendly summary first, raw technical details below.

## 6.6 Actor-centric design
People understand actors and roles better than operation taxonomies.

---

# 7. Functional architecture

## Layer ownership

### Layer 1 — Classifier
**Owner:** API

Produces:
- `tx_type`
- `subtype`
- `confidence`
- `wallet_involved`
- `effective_actor_type`
- `contract_type`
- `wallet_type`

### Layer 2 — Context enricher
**Owner:** API

Produces:
- actor identities and roles
- labels
- token metadata
- function names
- event summaries
- storage summary
- deploy context
- recent activity
- optional diffs/call graph
- anomaly flags over time

### Layer 3 — Narrator
**Owner:** Prism

Produces:
- title
- summary sentence
- subtitle
- warnings
- explanations
- recommendations
- ordering of content

---

# 8. Core UI components

## 8.1 NarrativeHeader
Purpose:
- show the primary human-readable story

Inputs:
- classification
- actors
- assets
- status
- time

Outputs:
- title
- one-sentence narrative
- subtitle/context line

Example:
- “Smart wallet policy update”
- “This wallet added a new signing key.”
- “Submitted 12 minutes ago by a heuristic smart wallet.”

---

## 8.2 ConfidenceBadge
Purpose:
- communicate certainty and provenance

Variants:
- Classified
- High confidence
- Medium confidence
- Heuristic
- Inferred

Tooltip content:
- confidence value
- rule source if available
- what confidence means

---

## 8.3 ActorChips
Purpose:
- present involved entities as meaningful participants

Chip fields:
- label
- role
- actor type
- optional secondary badge

Examples:
- `Aquabot #7` · effective actor
- `Soroswap Router` · protocol
- `USDC` · token contract

---

## 8.4 SignalCards
Purpose:
- summarize what stands out

Signal types:
- unusual time
- new signer
- first protocol interaction
- large amount deviation
- repeated failure
- state growth spike
- unusual storage profile
- first-seen contract behavior

Each card contains:
- title
- short explanation
- reason/evidence
- next suggested click

---

## 8.5 EvidencePanel
Purpose:
- explain why Prism classified this object the way it did

Shows:
- tx type
- subtype
- confidence
- operation types
- actor-type evidence
- matched heuristics
- source provenance

---

## 8.6 SemanticSection
Purpose:
- group structured facts into human-readable blocks

Examples:
- “What happened”
- “Who was involved”
- “Why this matters”
- “What changed”

---

## 8.7 RawDataTabs
Purpose:
- expose full explorer data

Tabs:
- Summary
- Actors
- Operations
- Events
- Diffs
- Storage
- Raw / XDR

---

## 8.8 SuggestedNextSteps
Purpose:
- guide exploration

Examples:
- View prior signer changes
- See this wallet’s recent activity
- Open contract transactions for `transfer`
- View all interactions with this protocol

---

# 9. Page specifications

# 9.1 Transaction Detail Page

## Page goal
Help a user understand what happened in a transaction, who was involved, why Prism classified it that way, and whether anything unusual occurred.

## Primary endpoint
- `GET /api/v1/silver/tx/{hash}/semantic`
- optional: `?include_deep=true`

## Page sections

### A. Hero / Narrative
Component:
- `NarrativeHeader`

Content:
- semantic title
- confidence badge
- status badge
- narrative sentence
- actor chips
- timestamp / ledger

Examples:
- “Smart wallet policy update”
- “Wallet X added a new signing key.”
- “2 actors · 1 contract · submitted 7m ago”

---

### B. What happened
Component:
- `WhatHappenedCard`

Description:
- 2–4 sentence structured explanation
- deterministic templates only

Examples:
- “This transaction appears to update the wallet’s signing policy.”
- “Prism detected a signer-management function and no transfer semantics.”
- “No asset movement was observed.”

---

### C. Who was involved
Component:
- `ActorList`

Rows:
- label
- address
- role(s)
- actor type
- wallet/protocol metadata if present

---

### D. What stands out
Component:
- `SignalCards`

Possible signals:
- off-hours activity
- unusually large amount
- new counterparty
- signer rotation
- policy change
- failure pattern
- unusual protocol route

---

### E. Why Prism thinks this
Component:
- `EvidencePanel`

Contents:
- `classification.tx_type`
- `classification.subtype`
- `classification.confidence`
- `classification.operation_types`
- `classification.wallet_involved`
- heuristic notes and rule matches

---

### F. Assets and effects
Component:
- `AssetFlowSection`

Shows:
- sent asset
- received asset
- fee
- token metadata
- path / protocol route if available

---

### G. Operations / Events / Diffs
Component:
- `RawDataTabs`

Contents:
- structured op list
- event list
- diffs if deep mode
- call graph if deep mode
- raw tx metadata

---

### H. Suggested next steps
Component:
- `SuggestedNextSteps`

Examples:
- View this wallet
- View contract
- See prior policy changes
- View all transactions by this contract

---

## Transaction page success metrics
- user can explain tx in one sentence within 5 seconds
- user can identify main actor in one glance
- user can inspect raw evidence in <2 clicks
- reduced bounce on tx page

---

# 9.2 Contract Detail Page

## Page goal
Present a contract as a living software artifact with identity, purpose, behavior, and state context.

## Primary endpoints
- `GET /silver/contracts/{id}/metadata`
- `GET /silver/contracts/{id}/analytics`
- `GET /silver/contracts/{id}/storage`
- `GET /api/v1/semantic/contracts`
- later: `GET /api/v1/silver/contracts/{id}/transactions`

## Page sections

### A. Hero
Component:
- `ContractHeader`

Shows:
- display name
- contract ID
- contract type
- tags
- creator
- deployed at
- function count
- state size

---

### B. What this contract is
Component:
- `ContractNarrativeCard`

Examples:
- “This appears to be a token contract.”
- “This contract exposes approve, transfer, and transfer_from functions.”
- “Current usage is low, suggesting recent deployment or limited adoption.”

Template inputs:
- `contract_type`
- exported functions
- activity metrics
- semantic registry info

---

### C. Contract profile
Component:
- `ContractProfileGrid`

Fields:
- creator
- deploy ledger
- wasm hash
- exported functions count
- storage entries
- state size
- monthly rent if available
- verification state if available

---

### D. Activity and health
Component:
- `ContractHealthSection`

Shows:
- total invocations
- last invocation
- activity trend chart
- avg / peak invocations
- success rate if available
- top callers later
- top functions

---

### E. Function surface
Component:
- `FunctionSurfaceTable`

Shows:
- function name
- usage counts
- time-window activity if available
- last-called if available

Narrative summary above table:
- “Most observed usage is concentrated in transfer-like functions.”

---

### F. Storage story
Component:
- `StorageStoryCard`
- `StoragePreviewTable`

Summary examples:
- “This contract currently has 3 persistent entries and about 164 KB of state.”
- “Storage appears lightweight for a token-style contract.”

Table fields:
- key preview
- type
- size
- TTL if available

---

### G. Recent invocations
Component:
- `InvocationFeed`

Humanized row style:
- short action summary
- caller chip
- status
- age

---

### H. Raw tabs
Tabs:
- Overview
- Functions
- Storage
- Transactions
- Code / Interface
- Raw

---

## Contract page success metrics
- user understands likely contract purpose quickly
- exported function surface is understandable
- state/storage is described in human terms
- deploy/creator context visible without scrolling deep

---

# 9.3 Account Detail Page

## Page goal
Explain what an account does, how it behaves, and who it interacts with.

## Primary endpoints
Current + future:
- account balances/activity endpoints
- semantic tx endpoint for tx drilldown
- later: actor behavior summaries from API

## Sections

### A. Hero
- account label / short address
- actor type if known
- first seen / last active
- recent behavior summary

### B. What this account does
Examples:
- “This account primarily receives payments.”
- “This account behaves like an operational treasury.”
- “This address frequently interacts with Soroban contracts.”

### C. Behavior profile
Shows:
- active hours
- typical tx volume
- top counterparties
- protocol usage
- activity trend

### D. What changed recently
- new counterparties
- increased tx activity
- protocol changes
- balance swings

### E. Recent activity
Humanized feed from semantic or enriched tx data.

### F. Raw account data
Balances, signers, thresholds, offers, etc.

---

# 9.4 Smart Wallet Detail Page

## Page goal
Make smart wallet behavior, signer policy, and governance comprehensible.

## Primary endpoints
- `/api/v1/silver/smart-wallets`
- `/api/v1/semantic/contracts?wallet_type=...`
- `/api/v1/silver/tx/{hash}/semantic`

## Sections

### A. Wallet identity
- wallet family
- source provenance: classified / heuristic
- implementation hints
- signer count

### B. What this wallet is used for
Examples:
- “This wallet appears to manage signers and route protocol interactions.”
- “Observed activity suggests automation or agent-like behavior.”

### C. Policy timeline
Feed of:
- signer add/remove
- threshold changes
- policy updates
- approvals

### D. Behavior profile
- active windows
- common protocols
- tx cadence
- unusual behavior

### E. Evidence panel
- why it is considered a smart wallet
- matched functions
- heuristic provenance

---

# 10. Narrative system specification

## 10.1 Philosophy
Narratives are generated from deterministic templates and structured semantic facts.

## 10.2 Template architecture
Prism maintains a registry keyed by semantic classification.

Example classes:
- `account_funding`
- `smart_wallet_policy_update`
- `smart_wallet_multisig_approval`
- `smart_wallet_transfer`
- `smart_wallet_swap`
- `wallet_mediated_protocol_interaction`
- `contract_call`
- generic fallback

## 10.3 Template interface
Conceptual interface:

```go
type NarrativeTemplate interface {
    Matches(tx SemanticTx) bool
    Title(tx SemanticTx) string
    Narrative(tx SemanticTx) string
    Subtitle(tx SemanticTx) string
    Signals(tx SemanticTx) []HumanSignal
    Evidence(tx SemanticTx) []HumanEvidence
}
```

## 10.4 Fallback behavior
If tx type is unknown:
- use generic semantic title
- describe actor + operation count + success/failure
- never hallucinate intent

Example fallback:
- “This transaction called a contract and produced 3 events.”

---

# 11. Signals framework specification

## Purpose
Highlight meaning, not just data.

## Signal model
Each signal has:
- `id`
- `title`
- `severity`
- `summary`
- `reason`
- `recommended_action`
- `evidence`

## Initial signal set
### Transaction signals
- Off-hours activity
- New counterparty
- Large transfer vs baseline
- Repeated failure
- Policy change
- Signer change
- First protocol interaction

### Contract signals
- Sudden activity spike
- State size spike
- First recent invocation after long inactivity
- Token-like contract with low observed transfer usage

### Wallet signals
- Signer rotation burst
- New protocol usage
- Behavior outside normal hours
- First large outflow

## Ownership
- signal detection logic may eventually move into API
- signal framing and copy live in Prism

---

# 12. Search and discovery spec

## Current vision
Search should accept:
- hashes
- account IDs
- contract IDs
- asset IDs
- plain-language assisted intents later

## Humanized search result cards
Each result should include:
- object type
- display label
- one-line description
- why it matched if useful

Examples:
- “Contract · Token contract · Active 2h ago”
- “Wallet · Heuristic smart wallet · Signer-management activity observed”

## Future conversational mode
Not phase 1.
Potential explicit-action feature.

---

# 13. Data contracts and endpoint mapping

## Transaction detail
### Endpoint
`GET /api/v1/silver/tx/{hash}/semantic`

### Components mapped
- `NarrativeHeader` ← `classification`, `actors`, `assets`, `transaction`
- `ActorList` ← `actors`
- `AssetFlowSection` ← `assets`, `operations`, `events`
- `EvidencePanel` ← `classification`
- `RawDataTabs` ← `operations`, `events`, `diffs`, `call_graph`, `transaction`

---

## Contract detail
### Endpoints
- `GET /silver/contracts/{id}/metadata`
- `GET /silver/contracts/{id}/analytics`
- `GET /silver/contracts/{id}/storage`
- `GET /api/v1/semantic/contracts`

### Components mapped
- `ContractHeader` ← metadata + semantic contract
- `ContractNarrativeCard` ← semantic contract + metadata + analytics
- `ContractProfileGrid` ← metadata
- `ContractHealthSection` ← analytics
- `FunctionSurfaceTable` ← metadata.exported_functions + analytics.top_functions
- `StorageStoryCard` / `StoragePreviewTable` ← storage + metadata storage summary

---

## Smart wallet discovery
### Endpoints
- `GET /api/v1/silver/smart-wallets`
- `GET /api/v1/semantic/contracts`
- `GET /api/v1/silver/contracts/{id}/transactions`

### Components mapped
- wallet family badges
- smart wallet lists
- policy timelines
- heuristic/classified provenance UI

---

# 14. Voice and content guidelines

## Tone
- calm
- precise
- confident but transparent
- non-hype
- editorial but factual

## Rules
1. Prefer intent over protocol jargon.
2. Never overstate certainty.
3. Always disclose heuristic classification.
4. Keep summary sentences short.
5. Raw technical terms should appear in tooltips or deep tabs, not lead copy.

## Good copy
- “This wallet added a signer.”
- “This transaction routed through a protocol contract.”
- “This contract looks token-like based on its function surface.”

## Bad copy
- “Invoke host function succeeded.”
- “OperationTypeInvokeHostFunction.”
- “AI thinks this may be suspicious.”

---

# 15. Trust and auditability requirements

## Must-have
- all humanized claims map to structured evidence
- confidence visible
- heuristics labeled
- raw underlying data accessible
- no unexplained narrative jumps

## UX requirement
Every narrative card should support:
- “Why Prism thinks this”
- “View raw details”

---

# 16. Performance requirements

## Transaction page
- lean semantic endpoint is default
- deep enrichment only on explicit interaction or delayed panel load

## Contract page
- summary shell loads first
- storage and deeper tables can be fragment-loaded

## General
- humanized content must not significantly degrade current page responsiveness

---

# 17. Phased implementation plan

## Phase 1 — Humanized transaction page
### Scope
- build semantic tx page experience
- narrative header
- actor panel
- evidence panel
- signal placeholders/basic rule-based signals
- raw tabs preserved

### Why first
- highest user value
- strongest semantic endpoint exists
- clearest differentiation

### Deliverables
- tx semantic view model
- template registry
- evidence panel
- actor chips
- fallback templates

---

## Phase 2 — Humanized contract page
### Scope
- use live metadata/analytics/storage
- add “what this contract is”
- function surface summary
- storage story
- creator/deploy framing

### Deliverables
- contract narrative card
- software profile layout
- storage story section
- activity/health section upgrades

---

## Phase 3 — Smart wallet experience
### Scope
- classified vs heuristic badges
- policy/signer timelines
- wallet behavior summaries

### Deliverables
- wallet family views
- signer/policy event presentation
- smart wallet discovery UI

---

## Phase 4 — Account behavior views
### Scope
- humanized account pages
- relationship and behavior summaries
- top counterparties and change detection

---

## Phase 5 — Advanced signals and guided investigation
### Scope
- richer anomaly signals
- “follow the story”
- suggested investigations
- optional on-demand LLM explainers

---

# 18. Optional LLM policy

## Default
No LLM on passive page load.

## Allowed explicit-use cases
- “Explain this unusual transaction”
- memo interpretation
- natural-language search
- summarizing complex multi-op txs on demand

## Constraints
- user action required
- cheap model only
- aggressive caching
- never replace deterministic core summary

---

# 19. Success metrics

## Product metrics
- reduced bounce rate on tx pages
- increased depth of navigation from tx → actor/contract
- more clicks on evidence and actor panels
- increased contract page dwell time

## UX metrics
- users can answer “what happened?” faster
- fewer confusing support questions
- higher confidence in smart wallet / contract interpretation

## Trust metrics
- evidence panel engagement
- low rate of user-reported misleading summaries

---

# 20. Immediate implementation backlog

## P0
- define Prism semantic tx view model
- build tx narrative template registry
- implement `NarrativeHeader`
- implement `EvidencePanel`
- implement `ActorChips`
- map `/silver/tx/{hash}/semantic` into tx page

## P1
- add contract narrative card
- add contract profile grid refinements
- add storage story card
- add function surface summary

## P2
- build signal framework
- add smart wallet provenance badges
- add suggested next steps

## P3
- on-demand explain mode
- conversational search
- behavior baselines and anomaly scoring

---

# 21. Final product statement

Prism should feel like:

- a block explorer for humans first
- an investigation tool second
- a raw blockchain terminal always available underneath

In one sentence:

**Prism turns structured blockchain facts into readable stories, while keeping every claim auditable.**
