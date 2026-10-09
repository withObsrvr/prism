# Prism Interpretation and Summary System

Status: proposed implementation design  
Audience: Prism developers and data/API developers  
Last updated: 2026-10-09

## Purpose

Prism's existing summaries prove that raw Stellar and Soroban evidence can be turned into understandable statements. The next step is to make that capability systematic across network windows, ledgers, transactions, operations, invocations, events, state changes, accounts, contracts, and assets.

This document defines:

- the information Prism can present at each evidence level;
- the difference between facts, measurements, semantic bands, interpretations, and prose;
- a shared architecture for producing trustworthy summaries;
- the bounded role Jev can play in choosing what matters;
- controlled rendering, validation, provenance, caching, and fallback behavior;
- TypeSafe playground examples for evaluating the approach;
- a staged implementation plan.

This design complements `docs/jev-integration-design.md`. That document describes Jev-assisted Ask routing. This document describes evidence interpretation and summary selection.

The protocol-to-band semantics and versioned ledger thresholds are maintained in
`docs/stellar-semantics-matrix.md`.

## Product objective

Every Prism summary should help a reader answer four questions:

1. What happened?
2. What is important or unusual about it?
3. What evidence supports that interpretation?
4. What should I inspect next?

The summary must remain useful to a non-expert without hiding the data an expert needs to verify it.

The core product metaphor is defined in `PRODUCT.md`: Prism refracts dense blockchain evidence into semantic bands such as capacity, fees, outcomes, activity, composition, and state change. In this design, that metaphor is literal architecture rather than presentation copy alone.

## Current state

Prism already has several interpretation mechanisms:

- ledger v2 presents counts, close time, Soroban calls, events, and success/failure totals;
- ledger v3 derives a capacity headline and exposes fees, failures, state changes, transaction distribution, chain context, and provenance;
- `internal/humanize` builds transaction and contract narratives from typed semantic data;
- `internal/txoutcome` explains structured failure evidence and rollback effects;
- `internal/insight` validates evidence envelopes before rendering home-page interpretations;
- account, contract, asset, Explore, and home view models already expose many potential summary facts.

These mechanisms are useful but currently local to individual pages. They do not yet share one vocabulary for facts, semantic bands, claims, interpretation candidates, or confidence. As a result, adding a richer summary tends to mean adding another sentence directly in a handler.

The proposed system preserves the working mechanisms while giving them a common foundation.

## Core principles

### Facts before interpretation

Chain facts, decoded evidence, and serving metadata are authoritative inputs. An interpretation may prioritize or contextualize facts, but it may not change them.

### Arithmetic stays in code

Counts, percentages, ratios, time windows, rates, baselines, deltas, and thresholds are calculated and validated in Go or upstream serving projections. Jev is not a calculator.

### Semantic bands are deterministic

Prism converts measurements into versioned bands such as `low`, `typical`, `elevated`, `high`, or `unavailable`. Thresholds are explicit and testable.

### Interpretations are registered claims

An interpretation is not free-form prose. It is a versioned identifier with eligibility requirements, claims, evidence requirements, priority, template, and validator.

### Jev prioritizes; Prism asserts

Jev may select the most relevant interpretation from candidates Prism has already proven eligible. Jev does not calculate metrics, invent an interpretation, select an unchecked entity, or generate the final claim.

### Missing data is not zero

Every summary input carries availability and provenance. Missing denominators, incomplete history, partial decoding, stale windows, and pruned evidence must remain visible.

### Simple summaries stay simple

If only one honest interpretation exists, render it deterministically without calling Jev.

## The useful pattern

```text
Raw Gateway/chain evidence
        |
        v
1. Normalize facts and availability
        |
        v
2. Calculate and reconcile metrics
        |
        v
3. Assign deterministic semantic bands
        |
        v
4. Produce eligible interpretation candidates
        |
        v
5. Select the lead and supporting interpretations
   deterministic policy or optional Jev selector
        |
        v
6. Apply confidence, relevance, and safety policy
        |
        v
7. Render controlled templates with exact values
        |
        v
8. Validate every claim against facts and bands
        |
        v
9. Attach evidence, caveats, provenance, and next actions
        |
        v
10. Cache by immutable entity or completed window
```

The same pipeline applies at every level. Only the fact schema, band registry, and interpretation registry change.

## Interpretation layers

The system should make these layers explicit:

| Layer | Meaning | Example |
|---|---|---|
| Raw fact | Value supplied by an authoritative source | `failed_tx_count = 0` |
| Metric | Reconciled calculation | `failure_rate = 0 / 23` |
| Band | Deterministic semantic state | `failure_state = none` |
| Candidate | An eligible registered reading | `all_transactions_succeeded` |
| Selection | Editorial priority | lead with fee/capacity contrast |
| Template | Reviewed language | “All 23 transactions succeeded.” |
| Evidence | Inspectable support | ledger transactions and result codes |
| Caveat | Known limitation | contract function classification partial |

Do not collapse these layers into a single `Summary string` too early. Keeping them separate allows Prism to validate, rerender, localize, test, and explain its conclusions.

## Evidence hierarchy and information inventory

The following inventory describes what Prism can potentially summarize. Availability varies by endpoint and network; each implementation must verify its source rather than assume every field is present.

### 1. Network and rolling-window level

Examples: home page, network health, recent-activity window, insight cards.

Available or derivable information:

- latest complete ledger and freshness;
- transaction and operation counts over a completed window;
- successful and failed activity;
- classic versus Soroban composition;
- operation-family composition;
- contract calls and distinct contracts;
- event counts and event-family composition;
- deployments and newly active contracts;
- activity leaders by calls and callers;
- failure leaders and failure concentration;
- contract TTL attention and shortest runway;
- network utilization against protocol limits;
- fee levels and fee movement;
- comparisons with previous windows or historical baselines;
- completeness, staleness, delivery mode, and provenance.

Useful deterministic interpretations:

- activity is within its typical range;
- transaction activity is elevated or depressed;
- contract calls form a majority/minority of classified activity;
- failures are elevated, concentrated, recovering, or unavailable;
- deployments are elevated;
- a contract is newly seeing verified adoption;
- one or more contracts need TTL attention;
- a resource limit is under sustained pressure;
- evidence is partial, stale, or unavailable.

Useful Jev role:

- select which of several already-qualified insights should lead the home page;
- decide which secondary dimensions deserve mention;
- choose between equally valid activity, risk, adoption, and capacity narratives;
- never decide whether a numerical detector threshold crossed.

### 2. Ledger-window level

Examples: 32-ledger home spectrogram and comparisons between adjacent ledger groups.

Available or derivable information:

- ledger range and completed time span;
- total and per-ledger transactions/operations;
- activity distribution across ledgers;
- success/failure totals and concentration;
- classic/Soroban/operation-family mix;
- number of quiet, typical, and busy ledgers;
- bursts, gaps, or sustained activity;
- contracts, functions, accounts, or assets contributing to the window;
- window completeness and exact included ledgers.

Useful interpretations:

- activity is steady, bursty, rising, falling, or mixed;
- contract calls form a majority of classified operations;
- failures are isolated to one ledger or spread across the window;
- one contract or operation family dominates the window;
- the latest ledger differs materially from its neighbors.

Jev should receive precomputed shape bands such as `steady`, `bursty`, `latest_spike`, or `insufficient_history`, not a raw series it must count or calculate.

### 3. Ledger level

Existing v3 fields demonstrate the breadth available here.

Header and identity:

- sequence and hash;
- previous ledger hash;
- close time and relative age;
- protocol version;
- transaction and operation counts.

Outcome and composition:

- successful and failed transaction counts;
- failure rate and result-code groups;
- operation-family distribution;
- Soroban transaction share;
- contracts invoked;
- distinct source accounts;
- transaction ordering and operation counts;
- event count and decoded event families.

Capacity and resource use:

- CPU instructions used and ledger cap;
- read/write footprint entries and caps;
- transaction/envelope size where comparable to an honest cap;
- maximum-utilized resource;
- per-resource headroom;
- whether any limit was meaningfully pressured;
- source and availability of every numerator and denominator.

Fees:

- base fee;
- clearing/inclusion fee where available;
- multiple of base fee;
- total collected;
- bids and recent fee history where supported;
- explicit limitations on excluded resource fees.

State and chain context:

- created, updated, removed, and restored entries;
- ledger-entry type distribution;
- state-change count;
- neighboring ledger activity and pressure;
- evidence gaps and provenance.

Useful interpretations:

- routine ledger;
- active but healthy ledger;
- ample capacity;
- one resource approaching or reaching pressure;
- elevated fees despite ample capacity;
- failure-free ledger;
- isolated failure versus clustered failures;
- Soroban-heavy or classic-heavy ledger;
- event-heavy ledger;
- state-write-heavy ledger;
- unusual change volume;
- evidence incomplete, so no capacity claim is safe.

Jev can choose the lead among several eligible readings. It must not infer that high fees were caused by CPU pressure unless the evidence contract explicitly supports that causal relationship.

### 4. Transaction level

Identity and envelope:

- hash, ledger, close time, source account, account sequence;
- fee charged and maximum fee;
- operation count and final success state.

Semantic activity:

- transaction type and subtype;
- actors and roles;
- value transfers, minting, burning, swaps, and asset identities;
- contracts and functions involved;
- smart-wallet and protocol interactions;
- operation sequence;
- call graph and invocation references where available;
- events emitted;
- effects, balance changes, and state changes.

Soroban resources:

- instructions;
- read/write bytes;
- resource relationship to transaction or ledger limits when the correct denominator exists.

Failure evidence:

- authoritative transaction result;
- normalized and raw result codes;
- failure phase and scope;
- failed operation and function;
- decoded arguments;
- executed-but-rolled-back operations;
- operations not executed;
- diagnostic availability and caveats;
- whether effects were applied to the ledger.

Useful interpretations:

- transferred, swapped, minted, burned, funded, deployed, upgraded, restored, or invoked;
- smart-wallet policy change or multisig approval;
- multi-operation workflow with one dominant effect;
- successful execution with material state changes;
- failure before execution, during contract execution, or during validation;
- partial execution rolled back because the transaction failed;
- exact cause known, broad category known, or exact cause unavailable;
- unusually resource-heavy transaction relative to a valid comparison population.

Jev may help select the dominant story when a transaction has multiple successful effects. Deterministic transaction outcome interpretation remains authoritative for success and failure.

### 5. Operation level

Available or derivable information:

- operation index and type;
- source account;
- transaction and ledger context;
- success inherited from the enclosing transaction plus operation result where served;
- payment destination, amount, and asset;
- Soroban contract and function;
- deployment, restore, extend-TTL, account, trustline, offer, and other classic-operation fields when decoded;
- emitted events, effects, balance changes, and state changes attributable to the operation;
- whether successful execution was later rolled back by transaction atomicity.

Useful interpretations:

- sent an asset;
- invoked a contract function;
- created or modified an account/trustline/offer;
- deployed or restored contract state;
- succeeded and applied;
- executed but was rolled back;
- failed with a known result code;
- did not execute because an earlier operation failed.

Most operation summaries should be deterministic because the operation type strongly constrains the language. Jev is useful primarily for ambiguous contract function semantics or choosing a dominant effect within a complex operation.

### 6. Invocation and call-frame level

Available or potential information:

- calling and called contracts;
- function name and decoded arguments;
- execution order and depth;
- success state;
- verified or inferred contract identity;
- nested invocation relationships;
- events and state changes associated with the call;
- diagnostic frames for failures.

Useful interpretations:

- direct user invocation versus nested protocol call;
- router-to-pool, wallet-to-protocol, factory-to-component, or token interaction;
- deepest failing frame;
- repeated calls to the same contract/function;
- cross-contract workflow;
- incomplete call graph.

Jev may classify unfamiliar function names into bounded behavioral families, but the UI must mark that classification as inferred. Call order, depth, success, and identities remain deterministic.

### 7. Event level

Available information:

- event type and index;
- classic or Soroban source;
- transaction, operation, ledger, and timestamp;
- from/to actors;
- amount, asset, issuer, and token metadata;
- emitting contract;
- raw topics/data when exposed;
- decoded transfer, mint, burn, approval, swap, or protocol-specific semantics.

Useful interpretations:

- transferred, minted, burned, approved, swapped, deposited, withdrew, voted, or emitted an unknown event;
- value-flow event versus state/administrative event;
- known token event versus custom contract event;
- event corroborates a transaction narrative;
- event decoding is partial or unavailable.

Jev can help classify unknown but decoded event text into a closed family. It should not invent missing amounts, participants, or asset identities.

### 8. State-change level

Available information:

- create, update, remove, and restore action;
- ledger-entry type;
- key;
- before and after values;
- owning account or contract where resolvable;
- transaction, operation, and invocation context;
- balance delta where the entry represents value;
- storage durability and TTL metadata when available.

Useful interpretations:

- balance increased or decreased;
- contract data created, updated, removed, or restored;
- authorization, signer, trustline, offer, or policy state changed;
- persistent or temporary storage affected;
- change is material to the transaction's main story;
- raw change is available but semantic meaning is unknown.

Arithmetic differences and before/after reconciliation stay in Go. Jev may help rank which of many state changes best explains the transaction.

### 9. Account and smart-account level

Available information:

- native and asset balances;
- recent transactions, operations, and transfers;
- sequence and last-modified ledger;
- signers and thresholds;
- subentry count;
- assets sent and received;
- counterparties;
- classic account versus contract-backed smart account;
- smart-wallet policies, allowed contracts, session keys, approval model, and signer configuration;
- activity and failure patterns over defined windows where served.

Useful interpretations:

- recently active or inactive;
- primarily receives, sends, trades, or invokes contracts;
- balance composition and concentration;
- signer/threshold configuration;
- smart-account authorization model;
- policy or signer changes;
- recurring counterparties or protocol interactions;
- evidence is only recent and must not be presented as lifetime behavior.

Behavior labels require explicit windows and minimum evidence. Avoid identity, ownership, intent, risk, or maliciousness claims that chain evidence cannot establish.

### 10. Contract level

Available information:

- contract ID, display identity, creator, deployment, and last activity;
- verified project association where available;
- exported and observed functions;
- calls, callers, success/failure counts, and top functions;
- recent invocations and events;
- state size, storage entries, durability, and TTL runway;
- linked assets or token identity;
- WASM/interface artifacts where available;
- behavioral signals already derived in `internal/humanize/contract.go`.

Useful interpretations:

- token, managed token, DEX, oracle, factory, wallet, administrative, or generic behavior;
- active function families;
- recent activity and caller breadth;
- concentrated versus broad usage;
- elevated failures or recovery;
- deployment-to-adoption progression;
- storage/archival attention;
- upgrade or administrative activity;
- project association verified, inferred, or unknown.

Jev can improve semantic classification of unfamiliar function sets and choose which verified behavior should lead. Deterministic rules should continue handling well-known interfaces and projects.

### 11. Asset level

Available information:

- asset code, issuer, and type;
- issuer authorization flags;
- linked token contract and classic/Soroban relationship;
- holder/trustline counts;
- circulating supply where supported;
- transfer count, volume, and unique accounts over a defined window;
- top-holder concentration;
- recent transfers;
- pairs, pools, liquidity, and volume where served;
- provenance and completeness.

Useful interpretations:

- active or quiet over a stated window;
- broad or concentrated holder distribution;
- transfer activity dominated by a few actors;
- classic asset linked to a Soroban token representation;
- issuer controls such as authorization, revocability, or clawback are enabled;
- evidence unavailable for supply, volume, or holders.

Prism must avoid price, value, liquidity-quality, legitimacy, or investment claims unless authoritative evidence for those concepts is explicitly available.

### 12. Search and Explore result-set level

Available information:

- applied filters and query scope;
- matched count, cap, cursor, and window;
- ledger range and evidence freshness;
- event/operation distribution;
- status, protocol, function, asset, and actor facets;
- representative rows.

Useful interpretations:

- what the result set contains;
- which filter or dimension most narrows it;
- dominant result family;
- whether results are capped or partial;
- no matching evidence versus unavailable evidence.

Jev may choose a concise description of a complex result set, but matching, counts, filters, and completeness remain deterministic.

## Shared domain model

Introduce a presentation-neutral summary package rather than embedding interpretation logic in handlers or templates.

Suggested package layout:

```text
internal/
  summary/
    types.go          # facts, bands, candidates, selections, claims
    registry.go       # interpretation registries
    eligibility.go    # deterministic candidate production
    selector.go       # selector interface and fallback policy
    render.go         # controlled templates
    validate.go       # claim/evidence validation
    ledger.go         # ledger fact and metric builder
    transaction.go    # transaction fact and metric builder
    operation.go
    contract.go
    account.go
    asset.go
    window.go
  jev/
    client.go
    types.go
    questions.go
    validate.go
```

Core types:

```go
type EntityLevel string

const (
    LevelNetwork     EntityLevel = "network"
    LevelWindow      EntityLevel = "window"
    LevelLedger      EntityLevel = "ledger"
    LevelTransaction EntityLevel = "transaction"
    LevelOperation   EntityLevel = "operation"
    LevelInvocation  EntityLevel = "invocation"
    LevelEvent       EntityLevel = "event"
    LevelStateChange EntityLevel = "state_change"
    LevelAccount     EntityLevel = "account"
    LevelContract    EntityLevel = "contract"
    LevelAsset       EntityLevel = "asset"
    LevelResultSet   EntityLevel = "result_set"
)

type Availability string

const (
    Available   Availability = "available"
    Partial     Availability = "partial"
    Stale       Availability = "stale"
    Unavailable Availability = "unavailable"
)

type Provenance struct {
    Status                Availability
    Sources               []string
    CompleteThroughLedger int64
    WindowStart           time.Time
    WindowEnd             time.Time
    UpdatedAt             time.Time
    Caveats               []Caveat
}

type SemanticState struct {
    Level       EntityLevel
    EntityID    string
    Bands       map[string]string
    Labels      map[string]string
    Provenance  Provenance
}

type ClaimID string
type InterpretationID string

type InterpretationSpec struct {
    ID               InterpretationID
    Level            EntityLevel
    Description      string
    Claims           []ClaimID
    RequiredBands    map[string][]string
    RequiredEvidence []string
    Priority         int
    TemplateVersion  string
}

type Candidate struct {
    ID          InterpretationID
    Description string
    Claims      []ClaimID
}

type Selection struct {
    Lead            InterpretationID
    Supporting      []InterpretationID
    Source          string // deterministic or jev
    Model           string
    Confidence      float64
    Probabilities   map[InterpretationID]float64
    RegistryVersion string
}

type RenderedSummary struct {
    Headline       string
    Detail         string
    Metrics        []Metric
    Evidence       []EvidenceLink
    Caveats        []string
    NextActions    []ActionLink
    Interpretation Selection
    Provenance     Provenance
}
```

Entity-specific facts and metrics should remain typed structs. Do not reduce numerical evidence to `map[string]any`; only the semantic state passed to a generic selector needs a common representation.

## Interpretation registry

Each interpretation is a small contract. Example:

```go
var LedgerInterpretationsV2 = []InterpretationSpec{
    {
        ID:          "healthy_capacity_with_elevated_fees",
        Level:       LevelLedger,
        Description: "Execution resources retained substantial headroom while fees were unusually elevated.",
        Claims: []ClaimID{
            "ledger_capacity_available",
            "ledger_fees_elevated",
        },
        RequiredBands: map[string][]string{
            "capacity_pressure": {"low"},
            "fee_pressure":      {"very_elevated"},
        },
        RequiredEvidence: []string{
            "capacity_numerators",
            "capacity_denominators_at_ledger",
            "fee_charged",
            "base_fee",
        },
        Priority:        70,
        TemplateVersion: "ledger_summary_templates_v1",
    },
}
```

Eligibility is evaluated in Go. If the required bands or evidence are absent, the option is never sent to Jev and can never be rendered.

## Candidate selection

Define one interface:

```go
type Selector interface {
    Select(
        ctx context.Context,
        state SemanticState,
        candidates []Candidate,
    ) (Selection, error)
}
```

Implementations:

- `DeterministicSelector`: selects by registered priority and stable tie-breaking;
- `JevSelector`: asks Jev to choose among eligible candidates;
- `FallbackSelector`: tries Jev and falls back deterministically on error or low confidence.

Skip Jev when:

- zero candidates exist;
- one candidate exists;
- the summary is a direct factual sentence rather than an editorial choice;
- required evidence is partial or unavailable and only a limitation message is safe;
- the result is already cached for an immutable entity and registry version.

## Worked ledger example

Input facts:

```text
Ledger                 5,104,847
Transactions           23
Operations             23
Successful             23
Failed                 0
Soroban calls          19
Events                 22
CPU utilization        26%
Fee multiple           172x base
Close time             5.0s
```

Calculated metrics:

```text
success_rate           100%
failure_rate           0%
soroban_share          19 / 23
events_per_call        22 / 19
maximum_utilization    26%
all_measured_limits_have_headroom = true
```

Bands assigned by Go:

```json
{
  "capacity_pressure": "low",
  "fee_pressure": "very_elevated",
  "failure_state": "none",
  "soroban_activity": "high",
  "event_activity": "active",
  "close_performance": "normal"
}
```

Eligible candidates:

```text
healthy_capacity
elevated_fees
contract_heavy
all_transactions_succeeded
healthy_capacity_with_elevated_fees
```

Possible Jev selection:

```text
lead = healthy_capacity_with_elevated_fees
supporting = all_transactions_succeeded, contract_heavy
```

Controlled rendering:

```text
This ledger had plenty of execution capacity, although fees reached 172x the base fee.

All 23 transactions succeeded. 19 Soroban calls emitted 22 events.
```

The exact numbers come from the typed fact struct. Jev never writes or edits them.

## Worked transaction example

Suppose one successful transaction:

- routes a swap through a wallet contract;
- transfers the input token;
- invokes a protocol router and pool;
- transfers the output token;
- changes several contract-data entries;
- emits swap and transfer events.

Possible deterministic bands:

```json
{
  "outcome": "succeeded",
  "value_flow": "swap",
  "actor_model": "wallet_mediated",
  "protocol_identity": "verified",
  "state_change_materiality": "supporting",
  "resource_pressure": "normal",
  "semantic_completeness": "complete"
}
```

Eligible interpretations:

```text
wallet_mediated_swap
protocol_swap
multi_contract_value_flow
successful_stateful_contract_interaction
```

If Jev chooses `wallet_mediated_swap`, Prism can render a reviewed transaction template populated from exact actors, amounts, assets, and contracts. Supporting state changes and call frames remain visible below it.

If the transaction failed, `internal/txoutcome` remains the lead authority. Jev must not override the normalized result, applied-to-ledger state, rollback explanation, or evidence caveats.

## Rendering and validation

Templates are functions over typed facts, not format strings populated with model text:

```go
type LedgerRenderer func(
    facts LedgerFacts,
    metrics LedgerMetrics,
) RenderedSummary
```

Every registered claim needs a validator:

```go
func ValidateClaim(
    claim ClaimID,
    facts LedgerFacts,
    metrics LedgerMetrics,
    bands LedgerBands,
) error
```

Examples:

- `all_transactions_succeeded` requires total transactions greater than zero and failures equal to zero;
- `capacity_available` requires every referenced meter to have a valid historical denominator and remain below the registered pressure threshold;
- `fees_elevated` requires a valid comparison basis and the corresponding fee band;
- `contract_majority` requires a complete classified denominator and a share above the registered threshold;
- `failure_recovery` requires both a prior failing condition and a current non-failing condition under the same versioned rule.

Validate before and after rendering. Before rendering, validate claim eligibility. After rendering, validate that every metric referenced by the template is available and correctly formatted.

## Confidence model

Keep four concepts separate:

1. **Evidence availability:** whether the source data is complete and current.
2. **Classification confidence:** confidence in semantic decoding, such as contract behavior.
3. **Selection confidence:** Jev's confidence that one eligible interpretation deserves priority.
4. **Claim validity:** deterministic pass/fail validation that the claim agrees with evidence.

Selection confidence is not confidence that the underlying fact is true. A headline can have low editorial-selection confidence while every displayed metric is fully authoritative.

Do not derive one blended confidence percentage. Present the relevant concepts independently.

## TypeSafe playground: ledger prioritization

### State

```json
{
  "surface": "ledger_summary",
  "bands": {
    "capacity_pressure": "low",
    "fee_pressure": "very_elevated",
    "failure_state": "none",
    "soroban_activity": "high",
    "event_activity": "active",
    "close_performance": "normal"
  },
  "availability": {
    "capacity": "complete",
    "fees": "complete",
    "outcomes": "complete",
    "soroban": "complete"
  }
}
```

### Questions

```json
{
  "lead_interpretation": {
    "type": "choice",
    "instructions": "Which eligible interpretation should lead a concise Prism ledger summary? Prefer operationally important and unusual conditions in `bands`. Do not perform arithmetic, infer a cause, or use information outside the supplied state.",
    "criteria": {
      "healthy_capacity": "The main story is that every measured execution resource retained substantial headroom.",
      "elevated_fees": "The main story is that transaction fees were unusually elevated.",
      "contract_heavy": "The main story is substantial Soroban contract activity.",
      "all_transactions_succeeded": "The main story is that all included transactions succeeded.",
      "healthy_capacity_with_elevated_fees": "The most informative story is the contrast between ample execution capacity and unusually elevated fees. This does not assert that either condition caused the other."
    }
  },
  "mention_capacity": {
    "type": "noul",
    "instructions": "Is execution-capacity headroom important enough to mention in a short summary of this `surface`, given `bands` and `availability`?"
  },
  "mention_fees": {
    "type": "noul",
    "instructions": "Is fee pressure important enough to mention in a short summary of this `surface`, given `bands` and `availability`?"
  },
  "mention_outcomes": {
    "type": "noul",
    "instructions": "Are transaction outcomes important enough to mention in a short summary of this `surface`, given `bands` and `availability`?"
  },
  "mention_soroban": {
    "type": "noul",
    "instructions": "Is Soroban activity important enough to mention in a short summary of this `surface`, given `bands` and `availability`?"
  }
}
```

Expected behavior: `healthy_capacity_with_elevated_fees` should be competitive or lead; capacity and fees should both receive high mention probabilities. Outcomes and Soroban activity may be supporting details.

## TypeSafe playground: home-window interpretation

### State

```json
{
  "surface": "home_activity_summary",
  "window": {
    "kind": "completed_ledgers",
    "ledger_count": "32",
    "completeness": "complete"
  },
  "bands": {
    "contract_call_share": "majority",
    "activity_level": "typical",
    "failure_state": "normal",
    "change_from_baseline": "not_evaluated"
  }
}
```

### Questions

```json
{
  "lead_interpretation": {
    "type": "choice",
    "instructions": "Which eligible interpretation best describes `bands` over `window`? Do not claim growth, decline, or unusual activity when `change_from_baseline` is not_evaluated.",
    "criteria": {
      "contract_majority": "Contract calls form a clear but not overwhelming majority of classified activity.",
      "balanced_activity": "Contract and non-contract activity have similar shares.",
      "elevated_activity": "Overall activity is elevated relative to a defined baseline.",
      "elevated_failures": "Failures are elevated relative to a defined baseline.",
      "routine_window": "No supplied dimension is notably different from its usual state."
    }
  },
  "supports_change_claim": {
    "type": "noul",
    "instructions": "Does the supplied state support saying that contract activity increased, decreased, or changed?"
  }
}
```

Expected behavior: `contract_majority` should lead and `supports_change_claim` should be near zero. A 60% share describes composition, not growth.

## TypeSafe playground: transaction emphasis

### State

```json
{
  "surface": "transaction_summary",
  "bands": {
    "outcome": "succeeded",
    "value_flow": "swap",
    "actor_model": "wallet_mediated",
    "protocol_identity": "verified",
    "state_change_materiality": "supporting",
    "resource_pressure": "normal",
    "semantic_completeness": "complete"
  }
}
```

### Questions

```json
{
  "lead_interpretation": {
    "type": "choice",
    "instructions": "Which eligible interpretation should lead the transaction summary? Select the interpretation that best explains the transaction's user-visible effect. Do not invent actors, amounts, assets, or causality.",
    "criteria": {
      "wallet_mediated_swap": "A wallet contract acted for the user to perform a swap through a verified protocol.",
      "protocol_swap": "The primary effect is a swap through a verified protocol.",
      "multi_contract_value_flow": "The primary story is value moving through several contracts.",
      "successful_stateful_contract_interaction": "The primary story is a successful contract interaction that changed state."
    }
  },
  "mention_state_changes": {
    "type": "noul",
    "instructions": "Are state changes important enough to mention in the short transaction summary given `bands`?"
  },
  "mention_resources": {
    "type": "noul",
    "instructions": "Is resource use important enough to mention in the short transaction summary given `bands`?"
  }
}
```

Expected behavior: `wallet_mediated_swap` should lead. State changes may be a supporting detail; normal resource use should generally not displace the user-visible effect.

## Playground evaluation matrix

Test semantic states rather than changing raw numbers and expecting Jev to calculate bands.

| Scenario | Expected lead behavior |
|---|---|
| Low capacity pressure, normal fees, no failures | healthy/routine ledger |
| Low capacity pressure, very elevated fees | capacity/fee contrast or elevated fees |
| High capacity pressure, normal fees | capacity pressure |
| Failures elevated and clustered | failure pressure |
| One isolated failure in a large ledger | isolated failure should not become a network-pattern claim |
| High Soroban share, otherwise normal | contract-heavy ledger |
| Capacity unavailable, fees elevated | never select a capacity interpretation |
| Contract share majority, no baseline | composition claim only; no growth claim |
| Failure condition previously high and now normal | recovery only if both windows use the same rule |
| Transaction swap plus supporting state changes | value flow should normally lead |
| Failed transaction with partial diagnostics | deterministic failure interpretation and caveat should lead |

For every run, save:

- model version;
- question-registry version;
- full probability distribution;
- confidence;
- selected option;
- expected acceptable options;
- whether the selection passed Prism's eligibility policy.

## Deterministic fallback

Every registry needs a stable fallback priority. A ledger example:

```text
authoritative failure concern
    > critical capacity pressure
    > high capacity pressure
    > unusual fee state
    > verified recovery or adoption insight
    > unusual composition
    > healthy/routine state
```

This is not a universal severity ordering. Each surface defines its own policy. A transaction should normally lead with its user-visible effect, while a failed transaction should lead with the authoritative outcome.

Fallback is used when:

- Jev is disabled;
- the call fails or times out;
- response validation fails;
- selection confidence is below the surface threshold;
- Jev selects an ineligible option;
- the model or question registry is not approved;
- cached selection is missing or stale.

## Provenance and caching

Immutable entities such as closed ledgers and finalized transactions should be interpreted once per registry version and cached.

Ledger v3 implements the first application cache as a bounded five-minute
snapshot. The key includes the network and ledger plus every behavior version:
bands, baselines, thresholds, interpretation registry, templates, and selector.
The finite lifetime is intentional while optional Gateway projections may
arrive after ledger close. Each envelope records `hit`, `miss`, or `bypass` in
its selection trace, and the same fields are emitted as structured logs.

```go
type InterpretationEnvelope struct {
    Level               EntityLevel
    EntityID            string
    InterpretationID    InterpretationID
    SupportingIDs       []InterpretationID
    SelectionSource     string
    SelectionConfidence float64
    Model                string
    RegistryVersion      string
    BandVersion          string
    TemplateVersion      string
    EvidenceVersion      string
    CompleteThrough      int64
    EvaluatedAt          time.Time
}
```

Rolling surfaces such as home summaries are cached by completed window identity. Do not recompute because a page was requested; recompute because the completed evidence window or an interpretation version changed.

Preferred flow:

```text
Evidence projection completes
    -> fact/metric/band builder runs
    -> candidates produced
    -> optional Jev selection runs once
    -> validated envelope stored
    -> web handlers render from the envelope
```

An initial implementation may run in the Go application with a short timeout and cache, but page availability must not depend on Jev.

## UI presentation

Recommended summary structure:

```text
[Interpretive headline]
[One or two exact supporting sentences]

[Key metrics]
[Evidence links]
[Caveats and freshness]
[What to inspect next]
```

Example:

```text
This ledger had plenty of execution capacity, although fees were unusually elevated.

All 23 transactions succeeded. 19 Soroban calls emitted 22 events.

CPU       26% full
Fees      172x base
Failed    0

Evidence: capacity limits, transaction results, fee data
```

The UI should not label all such content as “AI.” If Jev participated, precise disclosure is better:

```text
Interpretation priority selected by Jev; facts and wording produced by Prism.
```

The evidence panel should show data availability, completed-through ledger, source, and caveats independently of the selection mechanism.

## Safety boundaries

The summary system must not infer:

- malicious intent, fraud, or legitimacy;
- ownership or real-world identity;
- investment quality or future price;
- causality from correlation;
- protocol identity from an unverified name alone;
- lifetime behavior from a recent window;
- success of an operation whose transaction rolled back;
- zero use from missing data;
- “network congestion” solely from one high fee or one busy ledger;
- growth from a composition percentage without a baseline;
- exact failure cause when diagnostics provide only a broad result category.

Use language calibrated to evidence:

- authoritative: “The transaction failed.”
- measured: “Failures were above the registered baseline.”
- classified: “This appears to be a token-style contract.”
- limited: “The available evidence does not identify the exact cause.”

## Testing strategy

### Fact and metric tests

- source fields map correctly into typed facts;
- totals reconcile with component counts;
- rates handle zero denominators;
- windows contain only complete ledgers;
- historical caps are resolved at the entity ledger, not from current config;
- missing data remains unavailable rather than becoming zero.

### Band tests

- every threshold boundary has below/equal/above cases;
- band versions are explicit;
- unavailable inputs produce unavailable bands;
- changes in threshold policy require version changes.

### Eligibility tests

- candidates appear only when every required claim can be proven;
- partial evidence removes unsafe candidates;
- incompatible candidates cannot be selected together;
- unknown interpretation IDs fail closed.

### Rendering tests

- templates use exact typed values;
- singular/plural and zero cases are correct;
- no unavailable value is interpolated;
- HTML escaping and links are safe;
- templates do not introduce stronger claims than their registered claim set.

### Selector tests

- no Jev call for zero or one candidate;
- validated Jev choices are accepted above the configured threshold;
- low confidence and errors use deterministic fallback;
- ineligible and unknown choices are rejected;
- full probability distributions and versions are observable;
- real TypeSafe calls are never made in unit tests.

### Evaluation corpus

Create versioned fixtures for each surface:

```text
internal/summary/testdata/
  ledger-interpretations-v1.json
  transaction-interpretations-v1.json
  contract-interpretations-v1.json
  home-window-interpretations-v1.json
```

Fixtures should specify facts, expected bands, eligible candidates, acceptable lead selections, forbidden claims, and expected fallback.

## Implementation sequence

### Phase 1: vocabulary and inventory

- Adopt the evidence hierarchy in this document.
- List the exact Gateway fields and availability for each current page.
- Define a first claim registry for ledger and transaction surfaces.
- Define versioned thresholds for capacity, fees, failures, activity, and composition.
- Mark every current sentence as fact, measurement, interpretation, or caveat.

### Phase 2: shared deterministic pipeline

- Add `internal/summary` core types and validators.
- Implement ledger facts, metrics, bands, candidates, deterministic selection, and templates.
- Move v3 headline eligibility and contradiction checks behind the shared layer.
- Preserve the current UI while changing its source to the new summary envelope.

### Phase 3: transaction integration

- Adapt `internal/humanize` and `internal/txoutcome` to produce registered claims and candidates.
- Keep authoritative failure handling ahead of semantic story selection.
- Add operation, event, value-flow, state-change, and resource supporting summaries.

### Phase 4: Jev shadow selection

- Reuse the optional Jev client described in `docs/jev-integration-design.md`.
- Send only semantic bands and eligible candidates.
- Record what Jev would lead with without changing rendered summaries.
- Build and label disagreement cases.
- Calibrate surface-specific selection thresholds.

Ledger v3 now implements the shadow mechanics above with
`prism_ledger_summary_selector_v1`. Shadow identity participates in the cache
key, and success/failure/skipped/rejected outcomes are observable. Threshold
calibration and evaluation-corpus review remain before guarded prioritization.

The optional JSONL evaluation recorder and `prism jev evaluate-ledgers` command
provide that corpus and first report. The report removes duplicate evaluations
of the same network, ledger, shadow identity, and behavior versions; otherwise
navigation frequency would overweight popular ledgers. Human review labels are
kept as `preferred_lead` and `reviewer_note` fields in a copied corpus.

### Phase 5: guarded Jev prioritization

- Enable Jev selection for immutable ledger summaries with deterministic fallback.
- Cache by ledger, model, registry, band, and template version.
- Add precise UI disclosure and selection observability.
- Expand to complex successful transactions after ledger evaluation succeeds.

### Phase 6: wider surface coverage

- Add rolling-window/home summaries.
- Add contract behavior and activity summaries.
- Add account and asset summaries only after their observation windows and evidence contracts are explicit.
- Add operation/event/invocation summaries where deterministic decoding leaves meaningful ambiguity.

## First recommended implementation slice

Start with ledger v3 because it already has:

- a broad fact model;
- capacity, fee, failure, state-change, and composition sections;
- explicit served/derived/gap provenance;
- an interpretive headline;
- code that already prevents the headline from contradicting capacity meters.

The first slice should deliver:

1. `LedgerFacts`, `LedgerMetrics`, and `LedgerBands`;
2. five to eight ledger interpretation specifications;
3. eligibility and claim validators;
4. deterministic selector and templates;
5. a summary envelope consumed by the existing v3 view model;
6. an offline playground/evaluation fixture;
7. optional Jev shadow selection;
8. caching by ledger and registry version.

Recommended initial interpretation IDs:

```text
routine_ledger
active_healthy_ledger
all_transactions_succeeded
isolated_transaction_failure
clustered_transaction_failures
capacity_pressure
healthy_capacity
elevated_fees
healthy_capacity_with_elevated_fees
soroban_heavy
state_change_heavy
evidence_incomplete
```

After this slice proves the shared system, transaction integration can reuse the same selection, validation, provenance, and rendering contracts.

## Definition of done

The interpretation system is ready for its first production surface when:

- facts, metrics, bands, candidates, selection, and rendering are separate typed stages;
- every interpretation has explicit claims and evidence requirements;
- every numerical statement is produced by code;
- missing and partial evidence cannot produce positive claims;
- deterministic fallback works without Jev;
- Jev sees only semantic state and eligible closed choices;
- unknown or ineligible model output fails closed;
- summaries carry registry, band, template, and evidence versions;
- immutable summaries are cached;
- routing/selection confidence is not presented as factual confidence;
- unit tests cover reconciliation, thresholds, eligibility, templates, and fallback;
- the UI exposes evidence, caveats, and next actions;
- the resulting summary is materially more useful than a list of counts while remaining auditable.

## References

- Jev integration design: `docs/jev-integration-design.md`
- Product principles: `PRODUCT.md`
- Ledger v3 view model: `internal/templates/v2/viewmodel/ledger_v3.go`
- Ledger v3 live overlays: `internal/handlers/ledger_v3_live.go`
- Transaction humanization: `internal/humanize/tx.go`
- Contract humanization: `internal/humanize/contract.go`
- Transaction outcome interpretation: `internal/txoutcome/interpret.go`
- Home insight interpretation: `internal/insight`
- Gateway evidence types: `internal/gateway/types.go`
- TypeSafe primitives: https://docs.typesafe.ai/primitives
- TypeSafe state: https://docs.typesafe.ai/concepts/state
- TypeSafe API: https://docs.typesafe.ai/api
- Jev model limitations: https://docs.typesafe.ai/model-jaggedness/jev-1.13
