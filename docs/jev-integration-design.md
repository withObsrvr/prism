# Jev Integration for Prism

Status: proposed implementation design  
Audience: Prism developers  
Last updated: 2026-10-09

## Purpose

This document explains where TypeSafe AI's Jev model fits in Prism, how to integrate it into the existing Go application, and how to evaluate the proposed primitives in the TypeSafe playground before enabling them in production.

The recommended first integration is semantic routing for Ask Prism. Jev identifies the shape of a user's request; Prism code resolves entities, retrieves blockchain evidence, performs calculations, and renders the answer. Jev must not be treated as a source of ledger facts or as a replacement for Prism's deterministic evidence handlers.

## Executive summary

Prism currently recognizes a closed set of question intents with regex and string matching in `internal/intent`. That approach is fast and auditable, but it misses reasonable paraphrases and becomes difficult to extend as query language grows.

Jev is useful here because it accepts structured state plus typed questions and returns closed, machine-readable decisions:

- `Choice`: one option from a fixed set, with probabilities and confidence;
- `Noul`: the probability that a yes/no statement is true;
- `Score`: a position on a developer-defined ordered rubric.

The proposed request flow is:

```text
User query
    |
    v
Exact identifier/search parser
    | not an address, hash, ledger, or other direct navigation
    v
Existing deterministic intent matcher
    | no sufficiently strong match
    v
Jev semantic analysis
    | typed routing dimensions
    v
Prism routing policy and entity resolution
    | supported and sufficiently confident
    v
Existing deterministic intent handler
    |
    v
Gateway evidence -> calculations in Go -> evidence-backed answer
```

Jev is therefore a semantic router and evidence planner. It does not answer whether Soroswap is failing, whether USDC is busy, or whether a contract is close to archival. Prism answers those questions from Gateway data.

## Why the primitive is decomposed

An initial experiment used one `Choice` with options such as `protocol_busy` and `asset_activity`. For the query "Is USDC busy today?", Jev selected `protocol_busy` with 74% probability and `asset_activity` with 25%. The primitive mixed two independent judgments:

1. What subject is the user asking about?
2. What does the user want to know about it?

Separating those judgments produced the desired result:

```text
subject_kind = asset (100%)
request_kind = activity (100%)
time_window = one_day (93%)
```

The same decomposition correctly interpreted "Are Soroswap swaps still failing?" as:

```text
subject_kind   = protocol
request_kind   = recent_failures
evidence_scope = aggregate_pattern
trend_kind     = continuing
time_window    = current_snapshot
```

Independent Nouls are also needed for facets that can coexist. The question "Is Soroswap failing or just quiet?" asks about both failures and activity. A single `Choice` was forced to prefer one, while two Nouls correctly returned approximately 95% yes for each facet.

The design rule is:

- use `Choice` for mutually exclusive classifications;
- use independent Nouls for concerns that may coexist;
- compose the answers with deterministic Go code;
- ignore answers for dimensions that are irrelevant or insufficiently confident.

## Playground setup

Open the TypeSafe playground, select `jev-latest`, put the query object in the state field, and put the payload below in the questions field.

### State

```json
{
  "query": "Is Soroswap failing or just quiet?"
}
```

### Questions

```json
{
  "asks_about_failures": {
    "type": "noul",
    "instructions": "Does `query` ask whether failures are occurring?",
    "criteria": {
      "true": "The user asks whether one or more transactions, swaps, invocations, or operations are failing.",
      "false": "The user does not ask whether failures are occurring."
    }
  },
  "asks_about_activity": {
    "type": "noul",
    "instructions": "Does `query` ask about activity volume, busyness, or quietness?",
    "criteria": {
      "true": "The user asks whether something is busy, active, quiet, frequently used, or has a particular activity level.",
      "false": "The user does not ask about activity volume."
    }
  },
  "subject_kind": {
    "type": "choice",
    "instructions": "What kind of subject is the user asking about in `query`? Classify named assets such as XLM, USDC, EURC, AQUA, and asset codes as asset—not protocol.",
    "criteria": {
      "transaction": "One specific transaction or transaction hash.",
      "asset": "A Stellar asset, token, or asset code such as XLM, USDC, EURC, or AQUA.",
      "protocol": "A named application or protocol such as Soroswap, Blend, Phoenix, or Aquarius.",
      "contract": "A specific Soroban contract or contract address.",
      "network": "Stellar or Soroban activity generally, without a narrower subject.",
      "unknown": "The subject cannot be determined."
    }
  },
  "request_kind": {
    "type": "choice",
    "instructions": "What does the user want to determine from `query`?",
    "criteria": {
      "activity": "How busy, active, quiet, popular, or frequently used the subject is.",
      "failure_reason": "Why one particular transaction or invocation failed.",
      "recent_failures": "Whether multiple recent transactions, swaps, operations, or contract invocations are failing.",
      "expiration_risk": "Whether contract state has limited TTL, may expire, or may be archived.",
      "general_information": "General information that does not match the other request types."
    }
  },
  "evidence_scope": {
    "type": "choice",
    "instructions": "Does `query` concern one specific occurrence or an aggregate pattern across multiple occurrences?",
    "criteria": {
      "single_occurrence": "One particular transaction, swap, invocation, or failure.",
      "aggregate_pattern": "A trend, rate, recent set, or repeated behavior across multiple occurrences.",
      "unclear": "The scope cannot be determined."
    }
  },
  "trend_kind": {
    "type": "choice",
    "instructions": "What kind of temporal comparison does the user request in `query`?",
    "criteria": {
      "current_level": "The user asks only about the current level or state.",
      "continuing": "The user asks whether an earlier condition is still happening.",
      "recovery": "The user asks whether an earlier negative condition has stopped or returned to normal.",
      "increase": "The user asks whether activity or failures have increased.",
      "decrease": "The user asks whether activity or failures have decreased.",
      "none": "No temporal comparison is requested."
    }
  },
  "time_window": {
    "type": "choice",
    "instructions": "What explicit time period does the user request in `query`?",
    "criteria": {
      "one_hour": "The user explicitly requests the current or previous hour.",
      "one_day": "The user explicitly requests today, the last 24 hours, or the past day.",
      "one_week": "The user explicitly requests this week or the last seven days.",
      "one_month": "The user explicitly requests this month or the last 30 days.",
      "current_snapshot": "The user asks about the state right now without specifying a historical duration.",
      "unspecified": "No explicit time period is expressed."
    }
  }
}
```

Question IDs are application identifiers; Jev does not use them during inference. Each instruction must therefore be complete without relying on its ID.

### Playground test matrix

Run at least the following examples. Record the full probability distribution and confidence, not only the winning option.

| Query | Expected semantic result |
|---|---|
| `Is USDC busy today?` | asset + activity + one_day; activity Noul high |
| `Are Soroswap swaps failing today?` | protocol + recent_failures + aggregate_pattern + one_day; failure Noul high |
| `Are Soroswap swaps still failing?` | protocol + recent_failures + aggregate_pattern + continuing + current_snapshot |
| `Are Soroswap swaps not failing anymore?` | protocol + recent_failures + aggregate_pattern + recovery; current_snapshot or unspecified is acceptable |
| `Why did my Soroswap swap fail?` | protocol or transaction + failure_reason + single_occurrence |
| `Is Soroswap failing or just quiet?` | both Nouls high; the single request Choice may be uncertain and must not suppress either facet |
| `Which contracts might expire this week?` | contract + expiration_risk + aggregate_pattern + one_week |
| `Are transactions failing?` | network + recent_failures + aggregate_pattern + current_snapshot |
| `What has Blend been doing recently?` | protocol + activity; time may be unspecified |
| `Write a poem about Stellar` | general_information; both Nouls low |
| `Show transaction abc123` | transaction + general_information; direct Prism search should handle this before Jev in production |

Also test misspellings, punctuation, capitalization, negation, unsupported questions, and prompt-like text embedded in the query. A model result is not ready for production merely because the happy-path examples work.

## Target application types

Create a focused package such as `internal/jev`. Do not mix the external API schema into `internal/intent`.

Suggested public types:

```go
package jev

type Config struct {
    Enabled        bool
    BaseURL        string
    APIKey         string
    Model          string
    Timeout        time.Duration
    MinChoiceConfidence float64
    MinNoulProbability  float64
}

type Analysis struct {
    Model             string
    SubjectKind       ChoiceAnswer
    RequestKind       ChoiceAnswer
    EvidenceScope     ChoiceAnswer
    TrendKind         ChoiceAnswer
    TimeWindow        ChoiceAnswer
    AsksAboutFailures float64
    AsksAboutActivity float64
    Usage             Usage
}

type ChoiceAnswer struct {
    Choice        string
    Probabilities map[string]float64
    Confidence    float64
}

type Usage struct {
    InputTokens  int
    OutputTokens int
}

type Analyzer interface {
    AnalyzeQuery(ctx context.Context, query string) (Analysis, error)
}
```

Use an interface so handlers can receive a fake analyzer in unit tests. The production `Client` implements the interface with `net/http`.

## TypeSafe API client

The client sends:

```http
POST https://api.typesafe.ai/v1/systemone
Authorization: Bearer <API_KEY>
Content-Type: application/json
```

Top-level request shape:

```json
{
  "state": {
    "query": "Is USDC busy today?"
  },
  "model": "jev-latest",
  "questions": {}
}
```

Keep the question registry in Go as a versioned, immutable value. For example, record `prism_ask_router_v1` alongside each analysis. A prompt or criteria change is a behavioral release and must increment the registry version.

Client requirements:

- use a dedicated `http.Client` with a short timeout;
- propagate the inbound request context;
- set the bearer token without logging it;
- reject unknown answer types and missing required answers;
- validate probabilities and confidence as finite values in `[0,1]`;
- validate every `Choice` result against its local allowed-option registry;
- cap response body size before decoding JSON;
- retry `429` and `529` only, with bounded exponential backoff and jitter;
- do not retry context cancellation, validation errors, or most other 4xx responses;
- expose model name and token usage for metrics;
- never log the API key or the complete user query by default.

There is no need to add a Python or JavaScript service. Prism can call the documented HTTP endpoint directly from Go.

## Configuration and dependency injection

Add optional configuration alongside the existing Gateway configuration:

```yaml
jev:
  enabled: false
  base_url: "https://api.typesafe.ai"
  api_key: ""
  model: "jev-latest"
  timeout: 2s
  min_choice_confidence: 0.80
  min_noul_probability: 0.80
```

Environment overrides follow the existing Viper convention:

```text
PRISM_JEV_ENABLED=true
PRISM_JEV_API_KEY=...
PRISM_JEV_MODEL=jev-latest
PRISM_JEV_TIMEOUT=2s
```

Required wiring changes:

1. Add `Jev jev.Config` to `server.Config`.
2. Construct the optional Jev client in `server.New` only when enabled and an API key is present.
3. Add an `intent.Analyzer` or `jev.Analyzer` dependency to `Application` and `handlers.Handlers`.
4. Pass it through `server.Routes()` using the existing constructor-based dependency injection pattern.
5. Keep Jev disabled by default so missing credentials never prevent Prism from starting.

If `enabled=true` but the API key is absent, startup should fail with a clear configuration error. If Jev is disabled, the analyzer should be nil or a deliberately named no-op implementation.

## Routing policy

Do not replace exact parsing or the deterministic matcher. The initial production policy should be:

1. Resolve addresses, hashes, ledger numbers, and other navigational input with existing search code.
2. Run the existing deterministic intent registry.
3. If it has a sufficiently strong match, execute the existing handler.
4. Otherwise call Jev.
5. Validate and compose Jev's dimensions in Go.
6. Resolve the actual entity from Prism's registry or search candidates.
7. Execute one or more deterministic evidence handlers.
8. If Jev fails, times out, returns unsupported values, or is too uncertain, use the existing unsupported flow.

Do not calculate one misleading "combined confidence." Preserve each dimension and use only the dimensions required for the selected route.

Illustrative composition logic:

```go
func BuildPlan(a jev.Analysis) (Plan, bool) {
    if a.SubjectKind.Confidence < 0.80 || a.SubjectKind.Choice == "unknown" {
        return Plan{}, false
    }

    plan := Plan{SubjectKind: a.SubjectKind.Choice}

    if a.AsksAboutFailures >= 0.80 {
        plan.IncludeFailureMetrics = true
    }
    if a.AsksAboutActivity >= 0.80 {
        plan.IncludeActivityMetrics = true
    }

    if !plan.IncludeFailureMetrics && !plan.IncludeActivityMetrics {
        if a.RequestKind.Confidence < 0.80 {
            return Plan{}, false
        }
        plan.PrimaryRequest = a.RequestKind.Choice
    }

    if a.TrendKind.Confidence >= 0.70 && a.TrendKind.Choice != "none" {
        plan.Trend = a.TrendKind.Choice
    }
    if a.TimeWindow.Confidence >= 0.70 {
        plan.RequestedWindow = a.TimeWindow.Choice
    }

    return plan, true
}
```

The numbers above are pilot defaults, not proven thresholds. Calibrate them against a labeled Prism query set before production rollout. Different dimensions may require different thresholds.

### Example compositions

```text
asset + activity
    -> asset_activity handler

protocol + activity
    -> protocol_busy/contract_activity handler

transaction + failure_reason + single_occurrence
    -> transaction_failure handler

recent_failures + aggregate_pattern
    -> recent_failures handler

expiration_risk
    -> expiring_contracts handler

failure Noul high + activity Noul high
    -> fetch both evidence sets and compare the hypotheses
```

For "Is Soroswap failing or just quiet?", deterministic evidence composition can produce:

```text
low activity + normal failure rate
    -> Soroswap appears quiet, not unusually failure-prone.

normal activity + elevated failure rate
    -> Soroswap is active, but failures are elevated.

low activity + elevated failure rate
    -> Soroswap is quiet, and its limited recent activity has an elevated failure rate.

normal activity + normal failure rate
    -> Soroswap is neither unusually quiet nor experiencing elevated failures.
```

Those conclusions come from Gateway measurements and Prism thresholds, not from Jev.

## Entity resolution

Jev's primitives return closed choices, scores, and probabilities. They do not provide arbitrary string extraction. `subject_kind = protocol` does not extract `Soroswap`.

Keep entity resolution deterministic:

- known protocol names and aliases: use `intent.Registry.Protocols`;
- asset codes and issuers: use Prism search/Gateway candidates;
- contract, account, and transaction identifiers: use the existing search classifier and parser;
- ambiguous names: retrieve a bounded candidate list, then optionally ask a second Jev `Choice` whose options are those exact candidates;
- no entity found: request clarification or return unsupported rather than inventing one.

When a second request is used for candidate selection, the options must come from Prism. Never allow model output to become an unchecked address, hash, URL, or query parameter.

## User interface changes

The current Ask page says "deterministic answers" and "This route was selected without an LLM." Those statements must become conditional when Jev participates.

Suggested wording:

```text
Semantically routed by Jev; answered from Prism evidence.
```

Show interpretation separately from factual evidence:

```text
Understood as
  Subject: protocol (100%)
  Request: recent failures (100%)
  Scope: aggregate pattern (100%)
  Trend: continuing (95%)

Evidence
  Gateway sources, completed-through ledger, measured values, thresholds, and caveats
```

Do not display a Jev routing probability as confidence that the blockchain claim is true. "Routing confidence" and "evidence status" are separate concepts.

## Failure and fallback behavior

Jev is optional infrastructure. It must not make core explorer navigation or deterministic Ask routes unavailable.

| Condition | Behavior |
|---|---|
| Jev disabled | Current deterministic behavior |
| Jev timeout/network error | Log a sanitized event; use unsupported fallback |
| `429` or `529` after bounded retries | Use unsupported fallback |
| Invalid or unknown response option | Reject the analysis; use unsupported fallback |
| Low required confidence | Do not route; request clarification or use unsupported fallback |
| Low confidence on an irrelevant dimension | Ignore that dimension; continue if the required plan is safe |
| Gateway evidence unavailable | State that evidence is unavailable; do not generate a conclusion |
| Entity cannot be resolved | Ask for a specific asset/protocol/hash or use unsupported fallback |

Jev should fail closed: uncertainty broadens neither data access nor the claims Prism makes.

## Security, privacy, and cost controls

- Send only the user's query and the minimum semantic context needed for routing.
- Do not send API credentials, cookies, IP addresses, account session data, or unrelated ledger payloads.
- Treat the query as untrusted content, not instructions to Prism or permission to call arbitrary tools.
- Keep all options closed and validate returned values.
- Limit query length before sending it externally.
- Apply per-client and global rate limits to Jev-assisted Ask requests.
- Consider hashing or redacting queries in logs; addresses and transaction hashes are public-chain data but may still reveal user interest.
- Record model, registry version, latency, status, token counts, chosen values, and confidence for evaluation.
- Pin a model version for controlled production releases if repeatability is more important than automatically receiving `jev-latest` updates.

## Testing strategy

### Unit tests

Add tests for:

- request JSON and authorization headers;
- decoding all answer types;
- missing answers, wrong types, unknown choices, NaN/out-of-range values, and oversized bodies;
- retry behavior for `429` and `529`;
- no retry for invalid requests and context cancellation;
- routing composition for single and multi-facet questions;
- per-dimension confidence gating;
- fallback behavior when the analyzer is nil or returns an error;
- entity resolution independent of Jev;
- no model call for exact identifiers or strong deterministic matches.

Use `httptest.Server` for the client and a fake `Analyzer` for handlers. Tests must not call the real TypeSafe API.

### Offline evaluation set

Create a versioned fixture, for example `internal/jev/testdata/ask-routing-v1.json`, containing:

```json
{
  "query": "Is USDC busy today?",
  "expected": {
    "subject_kind": "asset",
    "request_kind": "activity",
    "asks_about_activity": true,
    "time_window": "one_day"
  }
}
```

Include clear examples, paraphrases, ambiguous questions, negations, multi-intent questions, misspellings, unsupported questions, direct identifiers, and adversarial text. Evaluate exact classification, false-positive routing, false-negative routing, confidence calibration, and behavior by query family.

### Shadow mode

The safest rollout is shadow mode:

1. Keep the current routing result user-visible.
2. Run Jev only for eligible Ask queries.
3. Store sanitized analysis telemetry without changing behavior.
4. Review disagreements and label a representative sample.
5. Tune questions, criteria, and thresholds; increment the registry version after changes.
6. Enable Jev routing for a small percentage of traffic.
7. Expand only after latency, cost, fallback rate, and routing quality meet targets.

Suggested metrics:

- requests eligible for Jev;
- successful, timed-out, rate-limited, and invalid calls;
- p50/p95 latency;
- tokens and estimated cost per analysis;
- route distribution and unsupported rate;
- confidence distribution per primitive;
- deterministic/Jev disagreement rate;
- clarification and evidence-unavailable rates;
- manually labeled accuracy and false-positive rate.

## Implementation sequence

### Phase 1: client and evaluation harness

- Add `internal/jev` request/response types, validation, client, and tests.
- Store the v1 question registry in code.
- Add a development command or test utility that runs fixture queries when an API key is explicitly provided.
- Build the labeled evaluation fixture from the playground cases in this document.

### Phase 2: optional application wiring

- Add Viper configuration and environment variables.
- Wire the analyzer through `server.Application` and `handlers.Handlers`.
- Keep the integration disabled by default.
- Add metrics and sanitized structured logs.

### Phase 3: shadow routing

- Invoke Jev only after direct search classification and deterministic intent matching fail.
- Record the analysis and the route Prism would have chosen.
- Do not change the user-visible route.
- Calibrate per-dimension thresholds from observed results.

### Phase 4: guarded production routing

- Enable high-confidence supported routes.
- Use existing deterministic handlers and evidence rendering.
- Add UI disclosure distinguishing semantic routing from evidence confidence.
- Retain unsupported and evidence-unavailable fallbacks.

### Phase 5: multi-evidence answers

- Use the failure and activity Nouls to build multi-handler evidence plans.
- Compose conclusions only from validated Gateway measurements.
- Consider additional Nouls only when a demonstrated multi-intent requirement exists; avoid accumulating speculative primitives without evaluation data.

## Non-goals for the first integration

Jev must not initially:

- generate prose answers;
- calculate counts, rates, ratios, balances, TTL, time windows, or thresholds;
- replace Prism's insight detector registry;
- infer arbitrary entity identifiers;
- label transactions as malicious or fraudulent;
- operate on large raw ledger or event payloads;
- become a dependency for explorer pages unrelated to Ask;
- replace visible evidence, provenance, or caveats.

These boundaries follow Jev's documented strengths: fast, typed semantic judgments. TypeSafe explicitly recommends keeping arithmetic, counting, and date comparisons in application code and filtering large state before evaluation.

## Definition of done for the pilot

The pilot is ready for guarded traffic when:

- the client and routing policy are covered by deterministic tests;
- Jev is optional and fails closed;
- exact identifiers and strong deterministic routes bypass Jev;
- returned options are validated against the local registry;
- a labeled evaluation set demonstrates acceptable accuracy and unsupported rejection;
- multi-intent queries preserve all high-probability facets;
- latency and cost budgets are documented and met;
- the UI clearly separates routing confidence from evidence confidence;
- model and question-registry versions are observable;
- no blockchain conclusion is produced without Gateway evidence.

## Ledger-summary shadow playground fixture

Ledger v3 uses `prism_ledger_summary_selector_v1` only after Go has calculated
facts, metrics, bands, and eligible interpretations. To exercise the same
decision in the TypeSafe playground, use this state:

```json
{
  "surface": "ledger_summary",
  "bands": {
    "evidence": "complete",
    "capacity.overall": "healthy",
    "fees.overall": "elevated",
    "outcomes": "none",
    "activity": "busy",
    "composition": "soroban_heavy",
    "state_change.volume": "typical"
  },
  "eligible_ids": [
    "healthy_capacity_with_elevated_fees",
    "elevated_fees",
    "soroban_heavy",
    "all_transactions_succeeded",
    "healthy_capacity"
  ]
}
```

Use this questions payload:

```json
{
  "lead_interpretation": {
    "type": "choice",
    "instructions": "Which eligible interpretation should lead a concise Prism ledger summary? Prefer operationally important and unusual conditions represented in `bands`. Choose only from the supplied criteria. Do not perform arithmetic, infer a cause, or use information outside this state.",
    "criteria": {
      "healthy_capacity_with_elevated_fees": "Measured resources retained headroom while inclusion fees were elevated.",
      "elevated_fees": "The measured inclusion fee was elevated relative to the ledger base fee.",
      "soroban_heavy": "Soroban transactions formed a large share of included transactions.",
      "all_transactions_succeeded": "Every included transaction in the complete evidence set succeeded.",
      "healthy_capacity": "Every applicable measured resource retained substantial headroom."
    }
  }
}
```

The playground choice is an editorial preference only. Prism independently
checks that the choice and complete probability distribution contain exactly
these eligible IDs, and shadow mode never applies the result to the page.

## References

- TypeSafe introduction: https://docs.typesafe.ai/introduction
- Primitives: https://docs.typesafe.ai/primitives
- State: https://docs.typesafe.ai/concepts/state
- Confidence: https://docs.typesafe.ai/confidence
- Intent routing pattern: https://docs.typesafe.ai/patterns/intent-routing
- HTTP API: https://docs.typesafe.ai/api
- Jev model limitations: https://docs.typesafe.ai/model-jaggedness/jev-1.13
- Current Prism intents: `internal/intent/intent.go`
- Current Ask handler: `internal/handlers/ask_v2.go`
- Prism product principles: `PRODUCT.md`
