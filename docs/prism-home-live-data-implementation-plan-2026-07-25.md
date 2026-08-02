# Prism Home Live Data and Insight Implementation Plan

Date: 2026-07-25
Updated: 2026-07-31
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
4. **Contract data expiring soon.** Surface contracts whose persistent state needs attention in plain language.
5. **Busiest contracts, 24h.** Rank contracts by calls, with identity and comparable failure evidence.
6. **Smart contract capacity.** Explain current Soroban resource use without implying unsupported causality or presenting it as whole-network traffic.
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
| `GET /v2/home/leaders` | `/home/summary` | 60 seconds |
| `GET /v2/home/utilization` | `/home/summary` | 30 seconds |

Each fragment gets a bounded request context. Expected upstream failures render the relevant section state. Internal rendering failures remain HTTP 500 responses.

The summary is fetched through one short-lived server-side cache and projected into the insights, TTL, leaders, and utilization fragments. This keeps their snapshot ledger coherent while preserving independent component states. Do not make the browser call the Gateway directly.

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

> Transaction, account, contract, asset, or ledger

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

The section title is `What changed`. Its compact scope label is:

> Compared with a typical hour

The deterministic narration defines the typical hour as the prior seven-day hourly median. The shorter heading keeps the comparison visible without repeating the full method above every result.

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

- the signal type, deterministic title, summary, and supported evidence detail;
- `Last hour`, `Typical hour`, and `Change` as comparable facts;
- a linked contract subject when the subject is a specific contract, but not a redundant network subject;
- component state in the section scan path;
- coverage caveats behind an on-demand disclosure;
- a type-specific `View …` evidence destination.

The validated view model preserves the exact observed window, comparison method, evidence count, identity source and verification, source ledger, update time, rule version, and caveats. The homepage does not repeat all of that metadata in the default scan path; it belongs in progressive evidence disclosure and the planned insight-detail surface.

The evidence destination is generated in Prism from typed API evidence locators, not from an API-authored frontend URL. A detailed insight page may progressively disclose contributors, first and last affected ledger, functions, result codes, callers, transactions, and representative samples as the evidence API gains them.

Claims such as `one contract is most of it`, `every call followed the same swap path`, `two new contracts are already busy`, or `first threshold crossing in twelve days` are prohibited until the API supplies the corresponding contribution, path, activity, or historical-crossing facts.

### Empty and degraded states

- `empty`: `No significant changes in the last completed hour.`
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

## Contract data expiring soon

Use `contracts_needing_attention` and the typed `ttl_attention` component state from `/home/summary`.

- include restorable persistent data, contract-instance, and contract-code evidence when the API proves the exact state kind; exclude temporary data from homepage eligibility;
- display absolute final-live-ledger evidence and reconcile its presentation against the live homepage heartbeat;
- convert to approximate time only when ledger cadence evidence is present;
- present Contract, Affected state, and Availability as one compact comparison row;
- use the exact count and state kinds at the nearest deadline, not the total count inside the attention window;
- explain the consequence first, such as persistent data that may need restoration, then retain the exact final live ledger as supporting evidence;
- render the final-live-ledger, may-already-require-restoration, confirmed-archived, extended, partial, and unavailable states explicitly;
- reserve total tracked entries, total entries inside the attention window, entry keys, lifecycle history, and full provenance for contract detail;
- do not describe a partial or unavailable result as zero contracts at risk;
- do not run normal per-contract `validate_ttl=true` fan-out.

The focused UX, API dependency, interaction model, and acceptance matrix are frozen in
`prism-contract-state-archival-experience-plan-2026-07-29.md`. Its corrected
contract-state wording supersedes the earlier `Expires in` presentation while keeping
the one-summary-read architecture.

## Busiest contracts, 24h

Use the `leaders` component from `/home/summary` for:

- identity, identity source, and verification status;
- contract ID and kind;
- calls and unique callers;
- success and failure counts;
- failure rate;
- top function;
- last activity;
- source ledger and update time.

Render this as a comparison table with stable Contract, Calls, Failed, and Callers / top function columns. The homepage shows the failure percentage beside total calls; exact success and failure counts are reserved for detail. Rates remain visibly tied to call volume so a high rate over two calls does not imply the same weight as a high rate over thousands. Suppress a second contract-ID line when the display name is only another shortening of the same identifier.

Identity enrichment must be a bounded set lookup. Prism must not call contract analytics once per row.

## Smart contract capacity

Render contract computation, contract state access, and transaction-envelope size independently. The UI labels translate Soroban instruction metering and combined ledger read/write bytes into user-facing language; the typed Gateway fields retain the protocol terminology.

- preserve reported percentages above 100;
- clamp only visual width to 100 percent;
- require used and ledger-specific limit values before explanatory copy;
- preserve source ledger and limit source in the view model and detail evidence without repeating them in every homepage metric;
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

Status: Implemented and browser-accepted on testnet and mainnet. The focused mainnet transaction fixture passed on 2026-07-31.

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

The mainnet pass confirmed the Gateway route, explicit failed state, non-green
operation result, broad general-failure explanation, and visible incomplete-diagnostics
caveat. Prism does not claim an exact contract cause for the accepted partial packet.

#### Slice 3B: E1B entity-search consumption

Status: Implemented and accepted on testnet and mainnet. The full testnet corpus and pinned bidirectional mainnet mapping pass.

This is the Prism consumer for the frozen `entity_search_v1` contract from `GET /silver/search`. It replaces the previous compatibility-only `query/results` decoder and prevents local convenience shortcuts from overriding authoritative live identity evidence.

Implemented behavior:

1. Decode the complete versioned packet: status, bounds, type filters, truncation, warnings, provenance, canonical entity kind, canonical slug, match field and type, identity source, verification state, and typed source facts.
2. Reject unknown evidence versions and response states.
3. Cache `ready` identity snapshots briefly and retry `partial` or `unavailable` packets sooner.
4. Decode typed HTTP `503 unavailable` packets as evidence so Prism can distinguish an index outage from an authoritative empty search.
5. Route accounts, classic assets, contracts, SACs, liquidity pools, protocols, protocol contracts, transactions, and ledgers without relying on the compatibility `type` alone.
6. Route SAC identities to their canonical classic-asset or XLM slug, preserving the API's reverse mapping rather than presenting the contract as an unrelated generic contract.
7. Preserve ambiguous asset codes such as `USDC`; list distinct issuers, disclose `has_more`, and never select an issuer implicitly.
8. Open a unique exact match, allow a unique bounded prefix match, and require an explicit result click for fuzzy matches.
9. Render match quality, verification status, friendly identity source, serving-only provenance, and complete-through ledger in the suggestion surface.
10. Render distinct authoritative-empty, partial, unavailable, and capped-result states. If the live index is incomplete or unavailable, Prism does not fall back to a built-in asset identity guess.
11. Keep suggest and submit on the same resolver so the previewed action and submitted destination remain consistent.

Current route limitation:

- Prism does not yet have dedicated liquidity-pool or protocol detail pages. Those canonical entity kinds route into a scoped Explore query. The search result remains explicitly labeled `Explore`, not `Open`.

Remaining acceptance work:

- exercise typed partial and unavailable packets against a controlled live testnet state when the API owner can safely induce or replay them; fixture and handler regression coverage already passes;
- add dedicated pool and protocol destinations when those Prism detail surfaces exist.

#### Slice 3C: E1C exact Explorer filtering

Status: Implemented and browser-accepted on testnet and mainnet. The mainnet pinned combined-filter query passed on 2026-07-31.

1. Decode and validate the versioned `explorer_events_v1` packet, including status, coverage, provenance, applied-filter evidence, count caps, warnings, normalized function, asset, actors, sender, recipient, and stable cursor.
2. Translate Prism filters into exact server-side `type`, function, asset, actor, outcome, time, ledger, contract, and transaction parameters.
3. Remove live row filtering and the token-transfer or generic-event fallback. An authoritative empty E1C packet remains empty and never triggers a broader query.
4. Render ready, authoritative empty, partial, unavailable, and invalid-query states distinctly. Partial results disclose their warning and never claim completeness.
5. Require an exact asset identity (`XLM`, `CODE:G…`, or a `C…` token contract). A bare code such as `USDC` requests issuer selection instead of silently broadening the query.
6. Use normalized actor and function fields for the default row instead of parsing decoder JSON.
7. Preserve the full active filter set during stable cursor pagination.
8. Disclose serving-only provenance, complete ledger coverage, capped counts, and explicit success or failure text.
9. Keep long hashes, assets, actors, and filter chips within the mobile viewport.
10. Treat a ledger-bounded query without an explicit close-time selection as a retained-coverage query, so historical ledgers are not accidentally constrained by the default one-hour window.

Testnet acceptance covers the known two-event exact fixture, an exact failed transaction, authoritative empty behavior, ambiguous-asset validation, and desktop/mobile rendering. The report is frozen in `PRISM_E1C_TESTNET_ACCEPTANCE_2026-07-29.md`.

### Slice 4: What changed v1

Status: Prism implementation and populated testnet acceptance completed. Mainnet
authoritative-empty behavior passed earlier on 2026-07-31, and a later live snapshot
the same day supplied three valid failure-spike packets. Testnet currently reports a
stale insight projection with no retained rows.

1. Add typed Gateway insight and evidence models.
2. Add the `internal/insight` rule registry.
3. Implement deterministic templates for the three deployed types.
4. Add the insights fragment and state handling.
5. Generate evidence links from typed subject and window filters.
6. Preserve identity source, comparison rule, exact window, ledger, and updated time for progressive evidence disclosure while keeping the homepage preview distilled.
7. Add fixture tests for ready, authoritative empty, partial, stale, unavailable, unknown type, and unknown evidence version.
8. Accept populated and authoritative-empty testnet behavior, then repeat against mainnet.

Prism now validates `home_insight_evidence_v1` packets, applies deterministic rules for all three deployed insight types, generates evidence destinations from typed locators, and renders ready, partial, stale, authoritative-empty, unavailable, unknown-type, and unknown-version states without synthesizing facts. Testnet now supplies a live deployment insight, accepted as a compact horizontal evidence row. Prism renders zero to three truthful results: one result fills the available width; two or three use flat comparison columns above 1024 pixels; narrower layouts stack them. A labeled demo fixture continues to exercise the rich failure-insight path without acting as a live fallback.

The first accepted 2026-07-31 mainnet snapshot was authoritatively empty. Prism rendered
the empty state rather than turning missing insight rows into an outage or a synthetic
claim. A later live snapshot rendered three validated failure-spike packets and exposed
their retained detail routes.

#### Slice 4A: Homepage insight availability and density refinement

Status: Prism-side evaluation consumption is implemented and browser-checked against the
live testnet v1 registry on 2026-07-31. The Query API evaluation, recent index, and
delivery contracts are live on testnet. Its observation gate and staged v2 through v4
registry rollout remain API-owned gates documented in
`PRISM_HOME_INSIGHT_AVAILABILITY_API_FOLLOWUP_2026-07-31.md`.

The populated homepage preview no longer repeats observed, baseline, and ratio values in
both prose and the comparison strip. Supported packets now render one deterministic
headline, one supplied contributor sentence, one three-value comparison strip, a compact
subject identity, and one primary explanation link. The raw Explore destination remains
available from the insight-detail page and remains the homepage fallback only for packets
without a retained detail route.

Zero-card behavior is now proportional to what Prism knows:

- authoritative `empty` remains `What changed`, but presents a compact negative result:
  no unusual change crossed Prism's rules in the last completed hour, and it names the
  three evaluated detector families;
- stale, partial, invalid, or unavailable evidence with no usable packet is titled
  `Hourly comparison`, not `What changed`; it uses one compact delayed row, preserves a
  retry control and on-demand diagnostic detail, and directs attention to the current
  evidence sections below;
- Prism does not derive a change claim from leaders, TTL, the spectrogram, or utilization.

Prism now decodes and strictly validates `insight_evaluation`, `recent_insights`, and
`insight_delivery`. A complete quiet hour shows the exact supplied detector comparisons
under the negative result, and one recent detection remains subordinate as `Last flagged`
with a retained detail link. `last_good` is labeled delayed, `unavailable` fails closed,
and partial evaluations show only their usable checks without claiming a quiet hour.

Live testnet browser acceptance passed at 1440 and 390 pixels with three registry-v1
checks, one subordinate recent link, no horizontal overflow, and no browser errors.
Acceptance evidence is recorded in
`PRISM_HOME_INSIGHT_EVALUATION_CONSUMER_ACCEPTANCE_2026-07-31.md`.

### Slice 5: Evidence-rich interpretation

Status: The retained insight detail route, deterministic evidence interpretation,
and frozen-contract browser acceptance are complete as of 2026-07-31. Live populated
testnet and mainnet acceptance remain data-dependent; testnet currently reports the
insights projection as stale and therefore exposes no retained detail link.

This slice begins after the API evidence phase is deployed.

1. Consume stable insight IDs and versioned evidence packets.
2. Add an insight detail route, provisionally `GET /v2/insight/{id}`.
3. Add type-specific interpretation rules for contributor concentration, dominant function or failure reason, deployment adoption, and historical resource-threshold crossings.
4. Show why each rule matched, including threshold and caveats.
5. Link representative transactions, contracts, and ledger ranges.
6. Add contract tests ensuring Prism narration uses only supplied facts.

Prism now consumes `GET /home/insights/{insight_id}` through a typed, bounded Gateway
client and exposes `GET /v2/insight/{id}`. Before narration, it verifies the evidence
version, recomputes the stable insight ID, reconciles common and type-specific facts,
enforces contributor and sample caps, validates sample selection and ledger bounds,
and rejects mismatched network or route identity. Valid pages lead with meaning and
magnitude, explain the exact deterministic rule and threshold, rank supplied
contributors, label representative transactions as non-complete pointers, preserve
network context in every proof link, and keep caveats, windows, watermarks, and source
tables in a subordinate evidence rail.

The three deployed evidence-v1 insight types have deterministic detail coverage: failure
concentration, deployment adoption, and transaction activity. Evidence-v2 consumer
coverage is also complete for `successful_activity_growth`, `failure_recovery`, and
`new_contract_adoption`. Those positive rules validate the API classification, thresholds,
ratio direction, typed facts, quality guards, locator bounds, stable IDs, and provenance
before Prism uses positive language. Frozen-fixture browser acceptance passed for all
three types at 1440 and 390 pixels with no overflow or browser errors. Known protocol result
codes lead with user-facing language such as `Contract stopped unexpectedly`; the raw
code remains supporting evidence. Invalid IDs return `400`, missing retained packets
return `404`, and unavailable or unverifiable evidence returns `503` without a fixture
or reconstructed narrative. Browser acceptance against the API's frozen failure-spike
fixture passed 12/12 at desktop and mobile, including no horizontal overflow. Full
evidence is recorded in `PRISM_INSIGHT_DETAIL_BROWSER_ACCEPTANCE_2026-07-31.md`.

Historical resource-threshold crossings remain outside the three frozen v1 insight
types. Prism must not infer that history from current utilization; add that narrative
only after the API publishes a versioned threshold-crossing evidence packet.

### Slice 6: Remaining sections and rollout

Status: Current TTL, leaders, utilization, responsive behavior, explicit outcome labels, and reduced-motion foundations are browser-accepted on testnet and mainnet. Product evidence, archival TTL-P0, and rollout cleanup remain pending.

The initial TTL table acceptance exposed a semantic gap: its relative countdown is
snapshot-bound, its refresh cadence can trail the live ledger heartbeat, and the
aggregate does not prove which entries share the nearest deadline. Complete focused
Slices TTL-P0 through TTL-P3 in
`prism-contract-state-archival-experience-plan-2026-07-29.md` before treating the TTL
portion of Slice 6 as production-ready or promoting it to mainnet.

1. Add TTL, leaders, utilization, and API-backed product-evidence fragments.
2. Complete responsive behavior for desktop, tablet, and mobile.
3. Verify status never relies on color alone.
4. Verify reduced-motion behavior.
5. Test mainnet, testnet, mock mode, partial responses, stale responses, and total failures.
6. Roll out behind a home-layout feature flag or internal preview.
7. Accept testnet first, then mainnet.
8. Remove `/v2/home/feed` and obsolete home-only rendering code after fragment acceptance.

The completed fragments consume the same cached `/home/summary` snapshot. Partial and stale packets keep valid rows visible with scoped warnings; unavailable packets never become zero or empty states. Utilization metrics fail independently, so a missing transaction-size value does not suppress instructions or read/write bytes. Testnet acceptance is recorded in `PRISM_HOME_EVIDENCE_TESTNET_ACCEPTANCE_2026-07-29.md`.

Mainnet acceptance and the consumer-resilience follow-up are recorded in
`PRISM_MAINNET_EVIDENCE_BROWSER_ACCEPTANCE_2026-07-31.md`. Prism now keeps a
successful summary packet for a bounded two-minute stale-if-error window. Retained
facts are labeled delayed; retained empty results are not presented as current; and
retained TTL rows preserve their absolute live-until ledger without continuing a
snapshot-relative countdown.

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
- `internal/insight/detail.go`
- `internal/gateway/types.go`
- `internal/gateway/client.go`
- `internal/gateway/home_insight.go`
- `internal/handlers/insight_v2.go`
- `internal/templates/v2/viewmodel/types.go`
- `internal/templates/v2/pages/home.templ`
- `internal/templates/v2/pages/insight.templ`
- `internal/templates/v2/fragments/home_sections.templ`
- `web/static/css/v2-unified.css`
- `web/static/css/v2-insight.css`

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
- Quiet-hour claims require a complete, registry-valid evaluation and matching current delivery state.
- Recent insights remain historical and never enter the current insight card list.
- Positive activity growth must pass its failure-rate guard; recovery must prove continued activity; adoption must pass call, caller, success-rate, and age thresholds.
- Evidence destinations preserve subject, ledger range, status, and activity filters.
- Insight detail IDs are recomputed from immutable evidence identity before narration.
- Contributor and sample bounds fail closed rather than producing partial invented copy.
- Representative transactions are labeled as pointers, never aggregate proof.

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
3. Resolved on 2026-07-31: the retained insight detail route ships as the evidence-rich follow-up, linked only from validated v1 homepage packets.
