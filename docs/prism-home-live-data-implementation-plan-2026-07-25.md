# Prism Home Live Data and Insight Implementation Plan

Date: 2026-07-25
Updated: 2026-07-28
Status: Implementation in progress
Project: Prism
Route: `GET /v2/home`
API foundation: `/home/tillman/Documents/ttp-processor-demo/obsrvr-lake/stellar-query-api/docs/PRISM_HOME_API_IMPLEMENTATION_PLAN_2026-07-25.md`
API evidence phase: `/home/tillman/Documents/ttp-processor-demo/obsrvr-lake/stellar-query-api/docs/PRISM_INSIGHT_EVIDENCE_API_IMPLEMENTATION_PLAN_2026-07-28.md`

## Outcome

The v2 homepage will port the information architecture and ledger spectrogram from `/home/tillman/Downloads/prism-home.html` into Prism's Go, Templ, htmx, and existing design-system architecture.

The prototype is a product concept, not a factual fixture or a palette source. Prism will retain its current restrained neutral palette, emerald interaction color, violet Soroban meaning, cyan transfer meaning, amber caution, and red failure semantics.

Prism owns insight and interpretation. The Query API owns measured facts, detection inputs, evidence, and provenance. The initial release will use deterministic interpretation rather than an LLM. Every sentence that interprets activity must be reproducible from a versioned rule and link to supporting evidence.

Live and automatic data modes must start from an empty, explicitly unavailable presentation model. They must never seed plausible mock blockchain facts. Mock fixtures remain available only when:

- Prism is configured with `PRISM_DATA_SOURCE=mock`, or
- the request explicitly includes `?mock=true`.

Mock mode must be visibly labeled as demo data.

## Product boundary

Prism is not a thin rendering client. Its role is to help a user understand:

1. what happened;
2. what changed;
3. who or what contributed;
4. why Prism reached that interpretation;
5. how certain and current the interpretation is;
6. where to inspect the raw evidence.

The API must not return editorial prose. Prism must not scan seven days of raw activity, guess unavailable values, or invent causal explanations.

| Concern | Owner |
|---|---|
| Bounded aggregates and comparisons | Query API serving projections |
| Detection eligibility and source completeness | Query API |
| Facts, identities, time windows, ledger bounds, and provenance | Query API |
| Query routing and supported-language interpretation | Prism |
| Human-readable narration and severity language | Prism |
| Evidence links and progressive disclosure | Prism |
| LLM phrasing, if introduced later | Optional Prism renderer behind validation |

## Verified current implementation

Prism already has:

- direct lookup classification for transaction hashes, accounts, muxed accounts, contracts, ledgers, and federation addresses;
- smart redirects for recognized assets, token contracts, and smart accounts;
- a deterministic activity parser for scopes, topics, functions, assets, status, and common time windows;
- a static entity registry for selected assets, functions, topics, and protocols;
- deterministic Ask handlers for contracts near TTL expiration and known-protocol activity;
- typed Gateway clients for home summary, recent ledgers, ledger summary, top contracts, semantic contract identity, and explorer events;
- an existing homepage template and live JSON feed.

The current homepage still violates the target contract:

- `HomeV2` starts with `mockHomeV2Data` even in live mode;
- `buildHomeV2Data` also starts from mock values;
- the template substitutes factual fallback copy and numbers;
- the browser rebuilds ledger rows from JSON instead of receiving server-rendered fragments;
- the live feed may fan out to per-ledger summaries;
- the current layout does not implement the prototype's spectrogram or `What changed` hierarchy;
- Gateway home types do not yet consume the deployed structured insight and component-state fields;
- unsupported prose silently falls through to a broad Explore text query.

## Goals

1. Port the prototype's information hierarchy and spectrogram concept.
2. Adapt the visual treatment to Prism's existing palette and component vocabulary.
3. Make search the homepage's primary navigation surface without claiming general natural-language intelligence.
4. Turn structured API insight facts into clear, deterministic interpretations.
5. Give every interpretation a visible rule, time window, provenance state, and path to proof.
6. Keep search and the shell usable when one or more data components fail.
7. Distinguish loading, ready, empty, partial, stale, and unavailable states.
8. Avoid per-ledger and per-contract request fan-out.
9. Preserve explicit mock mode for development, screenshots, and tests.

## Non-goals

This implementation will not:

- add an LLM to the homepage request path;
- claim unrestricted conversational or semantic search;
- calculate seven-day anomaly baselines from raw history in Prism;
- infer a cause that is not represented by API evidence;
- treat missing TTL rows as a healthy zero;
- create watchlist persistence, authentication, or notification delivery;
- expose Gateway credentials to browser JavaScript;
- use decorative crypto-neon color, gradient text, glass panels, or equal-weight metric grids.

## Information architecture

The page order is intentional:

1. **Network heartbeat and spectrogram.** Show the most recent 60 ledgers as a compact classified signal.
2. **Unified search.** Let users open known entities or describe supported recent activity.
3. **What changed.** Interpret material deviations from a documented baseline.
4. **Nearing archival.** Surface contracts whose persistent state needs attention.
5. **Most called, 24 hours.** Rank contracts by calls, with identity and failure evidence.
6. **Network utilization.** Explain current resource use without implying unsupported causality.
7. **Product evidence.** Render only API-backed metadata plus the static open-source link.

The prototype's watchlist pane is omitted until persistence exists. Do not render an `Add a contract` control that cannot save anything.

## Visual direction

Physical scene: a developer or investigator uses Prism on a normal work display to understand a live network signal, then moves from an interpretation to auditable evidence.

Use the current Prism design system:

- page and panels: Paper Surface, Card Surface, Subtle Surface, and Evidence Border;
- primary interaction and linked blockchain evidence: Signal Emerald;
- Soroban calls and contract values: Soroban Violet;
- payments and transfers: Transfer Cyan;
- offers and AMM activity: Signal Emerald where it represents a category, not health;
- deployments and caution thresholds: Caution Amber;
- failures: Failure Red;
- everything else: neutral gray.

Color in the spectrogram is semantic data encoding. It must not become decorative page wash. State is always reinforced with text, labels, icons, or pattern, never color alone.

Typography remains Instrument Sans for product language and JetBrains Mono for values users copy, compare, or audit. The page uses varied spacing and table structure rather than nested card grids. The spectrogram and search surface may receive the stronger focus treatment because they are the two primary workflow objects.

## Presentation-state contract

Add a home-specific state model under the v2 viewmodel package:

```go
type HomeSectionState string

const (
	HomeSectionReady       HomeSectionState = "ready"
	HomeSectionPartial     HomeSectionState = "partial"
	HomeSectionStale       HomeSectionState = "stale"
	HomeSectionEmpty       HomeSectionState = "empty"
	HomeSectionUnavailable HomeSectionState = "unavailable"
)

type HomeSectionStatus struct {
	State      HomeSectionState
	Message    string
	AsOfLedger int64
	AsOfTime   time.Time
	Warnings   []string
	Retryable  bool
}
```

Loading is the initial Templ skeleton before a fragment completes.

| State | Meaning | Presentation |
|---|---|---|
| `ready` | Authoritative data is present and fresh | Render normally |
| `partial` | Useful evidence is present, but the API reports omissions | Render evidence plus a scoped warning |
| `stale` | Useful evidence is present, but its projection is too old | Keep it visible and label it delayed |
| `empty` | The source completed and authoritatively found no rows | Render an intentional empty state |
| `unavailable` | The source failed, timed out, or cannot prove zero | Render an inline unavailable state and retry affordance |

Zero is valid only when the endpoint proves it. Missing fields must not be formatted as zero.

## Page and fragment architecture

`GET /v2/home` renders a fast shell with:

- existing global navigation and network selection;
- search form and supported examples;
- static section headings and explanatory copy;
- accessible loading skeletons;
- a visible demo-data indicator in mock mode.

Data sections load independently:

| Prism fragment | Gateway source | Initial refresh |
|---|---|---|
| `GET /v2/home/timeline` | `/silver/ledgers/recent?limit=60` | 5 seconds |
| `GET /v2/home/insights` | `/home/summary` | 60 seconds |
| `GET /v2/home/ttl` | `/home/summary` | 60 seconds |
| `GET /v2/home/leaders` | `/silver/contracts/top?limit=5` | 60 seconds |
| `GET /v2/home/utilization` | `/home/summary` | 30 seconds |

Each fragment gets a bounded request context. Expected upstream failures render the relevant section state. Internal rendering failures remain HTTP 500 responses.

The summary may be fetched once through a short-lived server-side cache and projected into several fragments. Independent presentation states must still be preserved. Do not make the browser call the Gateway directly.

## Ledger heartbeat and spectrogram

The spectrogram ports the prototype concept, not its generated values.

### Data encoding

- one vertical column per returned ledger;
- column height represents included operations relative to the returned window;
- stacked segments represent non-overlapping activity categories;
- a separate failure row represents failed operations so failures are not double-counted as a category;
- the legend percentages use the same returned rows and denominator as the columns;
- the heading derives the actual row count and close-time span.

Category mapping:

| Visual category | Recent-ledger evidence |
|---|---|
| Payments | `operations.categories.payments` |
| Offers and AMM | `operations.categories.offers_and_amms` |
| Contract calls | `operations.soroban_detail.contract_calls` |
| Deployments | `operations.soroban_detail.contract_deployments` |
| Other Soroban | `operations.soroban_detail.other` |
| Everything else | Sum of remaining non-Soroban categories |
| Failed | `operations.failed` |

If Soroban detail is absent, merge it into a single `Soroban` segment. Never infer deployments from a compatibility total.

### Interaction

Every column is an anchor to `/v2/ledger/{sequence}` and is keyboard reachable. Hover and focus expose:

- ledger sequence and close time;
- transaction count;
- included, successful, and failed operations;
- category counts;
- validator introduction identity when available;
- freshness and provenance.

The component is server-rendered Templ updated through htmx. New-ledger motion uses transform or opacity only, lasts 150 to 250 milliseconds, and is disabled by reduced motion. Prism does not increment sequence or age locally as though a new ledger arrived.

The timeline must not call one summary endpoint per ledger.

## Search experience

### Honest promise

Use this homepage language:

> **Find anything on Stellar**
>
> Paste a transaction, account, contract, asset, or describe recent activity.

Recommended placeholder:

> Transaction, account, contract, asset, ledger, or recent activity

Do not use `Look up anything on Stellar, in plain language` until Prism supports open-ended semantic questions and a clear unsupported-query state.

### Supported today

The first release can truthfully support:

- exact transaction, account, muxed-account, contract, ledger, and federation identifiers;
- smart accounts and token or asset contract redirects discovered through the Gateway;
- known assets, functions, topics, and protocols;
- bounded activity phrases such as `USDC swaps today`, `failed transfers last hour`, and `Soroban calls this week`;
- deterministic TTL-expiration and known-protocol activity questions.

Suggestions label the resolved action before submission:

- `Open transaction`
- `Open account`
- `Open contract`
- `Explore USDC swaps today`
- `Answer contract TTL question`

### Query-router machinery

Refactor search into an explicit routing pipeline:

```text
normalize
  -> extract embedded identifiers
  -> exact entity classification
  -> deterministic answer intent
  -> structured activity query
  -> registry/entity suggestion
  -> supported-query fallback
```

Return a typed internal resolution instead of relying on confidence alone:

```go
type SearchResolutionKind string

const (
	SearchOpen        SearchResolutionKind = "open"
	SearchExplore     SearchResolutionKind = "explore"
	SearchAnswer      SearchResolutionKind = "answer"
	SearchUnsupported SearchResolutionKind = "unsupported"
)
```

The resolution records parsed slots, rule ID, confidence, destination, and a user-visible interpretation summary. The same result drives suggestions and submit routing so the two paths cannot disagree.

### Near-term deterministic improvements

1. Extract a hash or StrKey from surrounding prose, including `why did <hash> fail?`.
2. Add a transaction-failure intent that requires a transaction hash and renders decoded failure evidence.
3. Add explicit recent-failure, contract-activity, and asset-activity answer families.
4. Expand aliases and replace the static-only registry with bounded Gateway entity suggestions.
5. Make time-window semantics exact. Do not turn `this week` into the API's fixed 100,000-ledger TTL window.
6. Render an unsupported state that explains what Prism understood and offers valid examples.
7. Remove mock search fallbacks from live and automatic modes.

An unsupported question must never appear to have been semantically answered by a loose free-text Explore query.

## What changed and interpretation machinery

### Initial product contract

The section title is `What changed`. Its scope label is:

> Latest completed hour compared with the prior seven-day hourly median

Do not say `today`, `compared 12 minutes ago`, or imply that an hourly insight refreshes every ledger.

The deployed API currently detects:

- `failure_spike`;
- `contract_deployments_spike`;
- `transaction_activity_spike`.

Prism renders zero to three ranked insights. It does not force three cards.

### Interpretation domain model

Add a dedicated package, provisionally `internal/insight`, with four responsibilities:

1. validate the API fact packet;
2. map an insight type and evidence version to a Prism interpretation rule;
3. produce display copy, severity, caveats, and typed evidence destinations;
4. fall back safely when the type or evidence version is unknown.

```go
type Interpretation struct {
	RuleID       string
	RuleVersion  string
	Title        string
	Summary      string
	Detail       string
	Severity     string
	Confidence   string
	Metrics      []Metric
	Evidence     []EvidenceLink
	Caveats      []string
	Provenance   Provenance
}
```

Rules are pure functions over typed input. They do not make network calls and they never insert numbers, IDs, time windows, names, or causal claims that are absent from the evidence packet.

### V1 deterministic templates

`failure_spike`:

> Contract invocation failures were {ratio} times the seven-day median in the last completed hour: {observed} failures versus a median of {baseline}.

`contract_deployments_spike`:

> {observed} contracts were deployed in the last completed hour, {ratio} times the seven-day median of {baseline}.

`transaction_activity_spike`:

> Transaction activity reached {observed} in the last completed hour, {ratio} times the seven-day median of {baseline}.

Use locale-aware formatting and round only for presentation. Keep raw values in accessible detail. If the subject is a contract, resolve identity with its source and verification status, link the contract ID, and never present an inferred name as verified.

### Evidence and explanation

Each insight preview shows:

- observed and baseline values;
- ratio and comparison method;
- exact observed window;
- subject and identity status;
- evidence count;
- source ledger and update time;
- component state and caveats;
- `Inspect evidence` destination.

The evidence destination is generated in Prism from typed API evidence locators, not from an API-authored frontend URL. A detailed insight page may progressively disclose contributors, first and last affected ledger, functions, result codes, callers, transactions, and representative samples as the evidence API gains them.

Claims such as `one contract is most of it`, `every call followed the same swap path`, `two new contracts are already busy`, or `first threshold crossing in twelve days` are prohibited until the API supplies the corresponding contribution, path, activity, or historical-crossing facts.

### Empty and degraded states

- `empty`: `No material changes were detected in the last completed hour.`
- `partial`: render supported facts and identify which evidence is incomplete.
- `stale`: keep the last insight visible with its actual window and a delayed label.
- `unavailable`: `The seven-day comparison is temporarily unavailable.`
- unknown type or version: show measured values in a generic evidence row, not guessed prose.

### LLM policy

No LLM is required for v1.

If an LLM renderer is added later, it must be:

- optional and outside the synchronous homepage dependency chain;
- given only a validated, versioned evidence packet;
- prohibited from adding metrics, identities, time windows, causal claims, or certainty not present in that packet;
- validated so every number, identifier, and evidence reference in its output exists in the packet;
- cached by evidence hash and prompt version;
- accompanied by the deterministic interpretation as a fallback;
- labeled as generated phrasing when that distinction matters.

The LLM must never detect anomalies or decide whether evidence is complete.

## Nearing archival

Use `contracts_needing_attention` and the typed `ttl_attention` component state from `/home/summary`.

- include persistent and contract-instance state only;
- display absolute `live_until` evidence and derive remaining ledgers against the component snapshot;
- convert to approximate time only when ledger cadence evidence is present;
- show tracked and expiring entry counts plus durability classes in detail;
- do not describe a partial or unavailable result as zero contracts at risk;
- do not run normal per-contract `validate_ttl=true` fan-out.

## Most called contracts

Use `/silver/contracts/top?limit=5&period=24h` for:

- identity, identity source, and verification status;
- contract ID and kind;
- calls and unique callers;
- success and failure counts;
- failure rate;
- top function;
- last activity;
- source ledger and update time.

Render this as a table, matching the prototype's information hierarchy and Prism's table system. Rates remain visibly tied to counts so a high rate over two calls does not imply the same weight as a high rate over thousands.

Identity enrichment must be a bounded set lookup. Prism must not call contract analytics once per row.

## Network utilization

Render instructions, read and write bytes, and transaction-envelope size independently.

- preserve reported percentages above 100;
- clamp only visual width to 100 percent;
- require used and ledger-specific limit values before explanatory copy;
- show source ledger and limit source;
- treat a missing metric independently;
- do not claim that fees rose because of a value unless the API supplies fee evidence for the same window.

## Implementation slices

### Slice 1: Truthful-state foundation

1. Add `HomeSectionState` and `HomeSectionStatus`.
2. Create `emptyHomeV2Data(network)`.
3. Pass the configured data-source mode into `Handlers`.
4. Make `HomeV2` choose explicit mock data or an empty live shell.
5. Remove handler-side mock seeding from `buildHomeV2Data`.
6. Remove factual template fallbacks such as default ledger numbers, alerts, utilization, and footer claims.
7. Move mock fixtures into `home_v2_mock.go`.
8. Upgrade Gateway home types for freshness, component states, warnings, recent-ledger envelope, and insights.
9. Add tests proving live mode cannot render known fixture values.

This slice can begin immediately against frozen API fixtures.

### Slice 2: Shell, palette, and spectrogram

1. Port the target information architecture into `home.templ`.
2. Implement the Prism palette mapping and responsive layout.
3. Add a server-rendered timeline fragment and htmx polling.
4. Map enriched recent-ledger rows directly into stacked columns and a separate failure row.
5. Derive the header heartbeat from the same source.
6. Remove duplicated browser-side ledger rendering and per-ledger summary fan-out.
7. Add ready, stale, empty, partial, and unavailable spectrogram states.
8. Add keyboard and accessible tooltip behavior.

### Slice 3: Search contract and truthful routing

1. Introduce `SearchResolution` and one shared resolver for suggest and submit.
2. Add embedded identifier extraction.
3. Label every suggestion as Open, Explore, Answer, or Unsupported.
4. Add the transaction-failure intent using existing decoded and semantic transaction evidence where sufficient.
5. Correct TTL question wording and time semantics.
6. Add deterministic contract, asset, and recent-failure answer families.
7. Add bounded dynamic entity suggestions.
8. Remove live mock fallback behavior.
9. Add an educational unsupported-query state.

#### Slice 3A: E1A transaction-outcome consumption

Status: Implemented in Prism on 2026-07-28; testnet API acceptance is available, while mainnet API rollout remains pending.

This is the Prism consumer for API Emergency Slice E1A. It is additive to the remaining Slice 3 search families and does not depend on E1B.

1. Add a typed Gateway client for `transaction_outcome_v1` from `GET /silver/tx/{hash}/failure-evidence`.
2. Reject unknown evidence versions and use short-lived caching for partial or improvable packets.
3. Add one shared deterministic interpretation layer for search answers and transaction detail pages.
4. Make the enclosing transaction result authoritative over receipt colors and operation summaries.
5. Render execution separately from ledger application: `Success`, `Executed, not applied`, `Failed`, `Not executed`, and `Unknown`.
6. Put the resolved failure reason, phase, operation number, function, bounded arguments, evidence state, and caveats in the default failure view.
7. Explain multi-operation rollback and operations that never executed without describing them as successful.
8. Preserve a visible receipt-only fallback while E1A is unavailable on a network; the fallback must state that it cannot identify the underlying cause.
9. Cover classic failure, Soroban host failure, rollback, unresolved evidence, unavailable evidence, unknown versions, and caching with regression tests.

The 2026-07-28 human-readable failure refinement now distinguishes exact causes from broad protocol categories, propagates diagnostic component limitations defensively, condenses duplicate receipt facts, and keeps raw result codes behind technical disclosure. The API status/caveat correction and versioned diagnostic-evidence recommendation are specified in `PRISM_TRANSACTION_FAILURE_EXPLANATION_API_FOLLOWUP_2026-07-28.md`.

Remaining before this sub-slice is deployable across both networks:

- deploy and accept E1A on mainnet;
- run browser acceptance against populated testnet and mainnet transactions;
- confirm the Gateway base-path configuration exposes the endpoint in each environment.

### Slice 4: What changed v1

1. Add typed Gateway insight and evidence models.
2. Add the `internal/insight` rule registry.
3. Implement deterministic templates for the three deployed types.
4. Add the insights fragment and state handling.
5. Generate evidence links from typed subject and window filters.
6. Add identity source, comparison rule, exact window, ledger, and updated-time disclosure.
7. Add fixture tests for ready, authoritative empty, partial, stale, unavailable, unknown type, and unknown evidence version.
8. Accept populated mainnet and authoritative-empty testnet behavior.

### Slice 5: Evidence-rich interpretation

This slice begins after the API evidence phase is deployed.

1. Consume stable insight IDs and versioned evidence packets.
2. Add an insight detail route, provisionally `GET /v2/insight/{id}`.
3. Add type-specific interpretation rules for contributor concentration, dominant function or failure reason, deployment adoption, and historical resource-threshold crossings.
4. Show why each rule matched, including threshold and caveats.
5. Link representative transactions, contracts, and ledger ranges.
6. Add contract tests ensuring Prism narration uses only supplied facts.

### Slice 6: Remaining sections and rollout

1. Add TTL, leaders, utilization, and API-backed product-evidence fragments.
2. Complete responsive behavior for desktop, tablet, and mobile.
3. Verify status never relies on color alone.
4. Verify reduced-motion behavior.
5. Test mainnet, testnet, mock mode, partial responses, stale responses, and total failures.
6. Roll out behind a home-layout feature flag or internal preview.
7. Accept testnet first, then mainnet.
8. Remove `/v2/home/feed` and obsolete home-only rendering code after fragment acceptance.

## Expected file changes

Primary Prism files:

- `internal/server/application.go`
- `internal/server/routes.go`
- `internal/handlers/handlers.go`
- `internal/handlers/home_v2.go`
- `internal/handlers/home_v2_mock.go`
- `internal/handlers/home_v2_fragments.go`
- `internal/handlers/search_v2.go`
- `internal/handlers/ask_v2.go`
- `internal/search/classify.go`
- `internal/search/parser.go`
- `internal/search/resolver.go`
- `internal/intent/intent.go`
- `internal/insight/types.go`
- `internal/insight/registry.go`
- `internal/insight/rules.go`
- `internal/gateway/types.go`
- `internal/gateway/client.go`
- `internal/templates/v2/viewmodel/types.go`
- `internal/templates/v2/pages/home.templ`
- `internal/templates/v2/pages/insight.templ`
- `internal/templates/v2/fragments/home_sections.templ`
- `web/static/css/v2-unified.css`

Tests:

- `internal/gateway/client_test.go`
- `internal/handlers/home_v2_test.go`
- `internal/handlers/search_v2_test.go`
- `internal/search/resolver_test.go`
- `internal/intent/intent_test.go`
- `internal/insight/registry_test.go`
- `internal/templates/v2/pages/home_test.go`
- fragment rendering and accessibility tests

## Test matrix

### Truthfulness

- Live mode without a Gateway contains no fixture ledger, contract, TTL, utilization, insight, or product-metadata values.
- A failed source never repopulates data from mock fixtures.
- Missing numeric fields render unavailable, not zero.
- Every narrated number and ID exists in its evidence packet.
- Unsupported insight types do not produce invented narrative.

### Spectrogram

- Requesting 60 labels the actual count returned.
- Time span is derived from close timestamps.
- Category totals equal each ledger's included-operation total under the documented compatibility rules.
- Failed operations are not double-counted.
- No per-ledger summary requests occur.
- Stale data changes `Live` to `Data delayed`.
- Keyboard focus exposes the same facts as hover.

### Search

- Exact identifiers route directly.
- Identifiers embedded in prose are extracted without changing their value.
- Suggest and submit resolve the same query identically.
- Supported activity phrases produce exact filters and windows.
- Unsupported prose reaches the unsupported state, not a false answer.
- Search unavailability never produces mock results in live mode.

### Insights

- Ratios and values are presented from API fields without recomputation drift.
- Exact completed-hour boundaries are shown.
- Authoritative empty is distinct from unavailable.
- Partial and stale evidence preserves facts and adds caveats.
- Identity inference is not labeled verified.
- Rich claims remain suppressed when optional evidence is absent.
- Evidence destinations preserve subject, ledger range, status, and activity filters.

### Utilization

- Values above 100 percent remain visible while visual width is clamped.
- Missing transaction-size evidence does not suppress other gauges.
- Each limit shows its ledger-specific source when available.

## Verification commands

```text
templ generate
go test ./...
go test -race ./...
go vet ./...
make build
git diff --check
```

## Completion gate

The implementation is complete only when:

1. Live and automatic modes cannot render mock blockchain facts.
2. The prototype hierarchy and spectrogram are implemented with Prism's palette and components.
3. The spectrogram uses one enriched collection request and labels actual rows and time span.
4. Search makes an accurate capability promise and exposes how each query resolved.
5. `What changed` renders deterministic interpretations from structured API evidence.
6. Every interpretation exposes its rule, time window, freshness, provenance, and proof path.
7. Partial, stale, empty, and unavailable states remain distinct end to end.
8. No LLM is required for correctness or availability.
9. Automated tests, race tests, vet, build, and diff hygiene pass.
10. Testnet and mainnet acceptance cover live, populated, empty, partial, stale, and unavailable behavior.

## Deferred decisions

1. Watchlist storage, identity, notification channels, and delivery semantics.
2. Whether an evidence-constrained LLM renderer adds enough value after deterministic v1 ships.
3. Whether the insight detail route belongs in the first homepage rollout or the evidence-rich follow-up.
