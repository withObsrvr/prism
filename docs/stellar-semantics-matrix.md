# Stellar semantics matrix

This document is the protocol-semantics boundary for Prism's ledger summary
system. It maps Stellar concepts to typed evidence and semantic bands. The
top-level bands are deliberately stable; protocol detail appears as sub-band
state rather than as an ever-growing taxonomy.

Implementation versions:

- bands: `ledger_bands_v2`
- baselines: `ledger_baselines_v1`
- thresholds: `ledger_thresholds_v1`
- interpretations: `ledger_interpretations_v2`
- controlled templates: `ledger_summary_templates_v1`
- package: `internal/summary`

## Rules

1. A band describes only evidence marked available for the requested ledger.
2. `not_applicable` is different from `unavailable`.
3. Missing numerators and missing denominators never become zero.
4. A baseline-dependent band is `not_evaluated` when no baseline exists.
5. Partial transaction evidence cannot produce a complete outcome claim.
6. Protocol-sensitive denominators must be effective at the entity ledger.
7. Correlation between pressure and fees is not a causal claim.

## Ledger matrix

| Stellar concept | Typed evidence | Prism state | Safe claims | Claims requiring more evidence |
|---|---|---|---|---|
| CPU instructions | used instructions and effective ledger limit | `capacity.cpu` | CPU retained headroom / approached its limit | CPU caused elevated fees |
| Ledger entry access | declared read/write footprint counts and effective caps | `capacity.read_entries`, `capacity.write_entries` | declared footprint pressure | observed changes equal declared footprint |
| Ledger I/O | read/write bytes and effective caps | `capacity.read_bytes`, `capacity.write_bytes` | byte-I/O pressure | entry-count pressure from bytes |
| Transaction bandwidth | comparable transaction-byte numerator and ledger cap | `capacity.transaction_bytes` | transaction-byte pressure | total envelope bytes against a Soroban-only cap |
| Events and return values | emitted byte total and effective cap | `capacity.event_return_bytes` | event/return pressure | event count as byte usage |
| Inclusion fee | effective charged inclusion fee and base fee | `fees.inclusion` | at base, elevated, or extreme relative to base | number or identity of excluded transactions |
| Resource fee | charged Soroban resource component | `fees.resource` availability | component was measured | resource pressure from fee amount alone |
| Rent | rent charged for TTL or size changes | `fees.rent` availability | rent was charged | archival pressure without TTL/state evidence |
| Dynamic storage write pricing | effective write fee or rate | `fees.storage_write` availability | storage-write pricing was measured | ledger-local write contention caused the rate |
| Transaction result | complete included transaction set and canonical result codes | `outcomes` | none, isolated, clustered, widespread | network failure from one ledger alone |
| Activity | transaction/operation counts and a versioned comparison population | `activity` | quiet, typical, or busy against that baseline | unusual activity without a baseline |
| Classic/Soroban mix | complete transaction counts by execution family | `composition` | classic-heavy, mixed, or Soroban-heavy | protocol intent or user intent |
| Entry changes | created, updated, deleted, restored counts | `state_change.volume` and component availability | measured change volume and kinds | declared write footprint from observed changes |
| Archival | persistent/instance eviction evidence | `state_change.archived` availability | entries were archived | temporary entries can be restored |
| TTL extension | explicit TTL extension evidence | `state_change.ttl_extended` availability | TTLs were extended | all updated TTL entries were user initiated |
| Evidence coverage | sources, completion ledger, sampling and caveats | `evidence` | complete, partial, stale, unavailable | positive claims hidden by missing evidence |

## Thresholds v1

Thresholds are product policy, not Stellar protocol constants:

- capacity: healthy below 50%, elevated from 50%, near limit from 80%,
  exceeded above 100%;
- inclusion fee: base through 1.1×, elevated above 1.1×, extreme above 10×;
- failures: widespread at 10% or more; otherwise a cause representing at
  least half of two or more failures is clustered;
- activity and state-change volume: quiet below 0.5× baseline, typical through
  1.5×, busy/heavy above 1.5×;
- composition: classic-heavy at 25% Soroban or less, Soroban-heavy at 60% or
  more, mixed between them.

Changing these values requires a new threshold version and fixture update.

## Interpretation registry v1

The first registry is declarative and validated at runtime. Each specification
contains a stable ID, registered claims, allowed band states, priority, and
template version. Eligibility never depends on free-form prose.

Initial IDs cover routine and active healthy ledgers, complete success,
isolated/clustered/widespread failures, capacity pressure, healthy capacity,
elevated fees, the healthy-capacity/elevated-fee contrast, Soroban-heavy
composition, heavy state change, and incomplete evidence.

Incomplete evidence has the lowest fallback priority. Independent verified
observations lead whenever one is available; for example, elevated fees or
widespread failures can lead while capacity evidence is unavailable. Missing
evidence remains explicit in the limitations block and provenance, and no
capacity claim becomes eligible without its required measurements. Evidence
incompleteness leads only when Prism has no substantive safe interpretation.

## Ledger v3 adapter

`internal/handlers/ledger_v3_summary.go` converts the evidence already loaded
for Ledger v3 into `LedgerFacts` and builds a `LedgerSummaryEnvelope`. The
envelope carries facts, metrics, bands, eligible candidates, deterministic
selection, and every behavior version. Ledger v3 consumes that envelope through
the controlled renderer and uses the result for its headline and opening
interpretation.

Adapter rules:

- bounded transaction or operation rows mark the envelope partial;
- failure-cause grouping is partial when transaction rows are partial;
- a ledger with no Soroban transactions marks Soroban capacity and fee
  components `not_applicable`;
- unsupported event/return-byte and comparable transaction-byte capacity keep
  overall Soroban capacity partial rather than silently disappearing;
- activity compares transaction and operation counts with the rounded median
  of up to 32 preceding ledgers and requires at least 16 valid samples;
- the baseline window excludes the ledger being interpreted, deduplicates
  sequences, and records its method, range, and sample count;
- a zero median cannot support a relative band and remains `not_evaluated`;
- state-change volume remains `not_evaluated` until Gateway supplies a bounded
  historical change window; operation counts are not used as a proxy;
- median charged fees become inclusion-fee evidence only for ledgers with no
  Soroban transactions; mixed ledgers require a separately served inclusion
  component;
- immutable ledger identity and aggregate counts come from the full ledger
  response, while optional endpoints retain independent availability.

## Controlled rendering

`internal/summary/render.go` is the only ledger-summary prose boundary. It
maps registered interpretation IDs to versioned templates and inserts exact
values calculated by Go. It returns separate headline, detail, supporting
readings, caveats, and next-action fields; templates render those fields as
escaped text.

Before rendering, Prism recomputes eligible candidates from the envelope's
semantic bands. The selected lead and every supporting interpretation must be
present in that derived set. This prevents a stale caller, cache entry, or
future selector from rendering a claim that the current evidence does not
admit. Unknown versions and unknown interpretation IDs fail closed.

Next actions link to the evidence sections visible in the ledger overview.
Missing or partial evidence produces explicit caveats, and the renderer never
turns absence into a measured zero. Jev is not in this path: a future Jev
integration may select among the same eligible IDs, but it will not supply
facts, arithmetic, links, or prose.

## Snapshot caching and selection trace

Closed-ledger summary envelopes use a bounded, five-minute in-process snapshot
cache. Its identity includes network, ledger sequence, and the band, baseline,
threshold, interpretation-registry, controlled-template, and selector versions.
Changing any behavior version therefore makes the previous snapshot
unreachable. The deliberately short lifetime avoids pinning an incomplete
Gateway projection indefinitely while still preventing repeated optional
evidence requests during normal navigation.

Every envelope also carries a machine-readable `selection_trace` containing
the surface, selector version, mode, ordered eligible IDs, deterministic lead,
applied lead/source, and cache status. Structured logs emit the same decision
fields. This is editorial-selection observability, not evidence confidence.
The controlled renderer rejects a trace whose eligible set or applied choice
does not match the current semantic state.

When `jev.enabled` is true, cache misses with at least two eligible readings run
the versioned `prism_ledger_summary_selector_v1` question. Jev receives only the
seven semantic band values and the closed eligible ID/description choices. It
does not receive raw ledger facts, measurements, links, or template prose.

Prism validates the selected ID, complete probability distribution, confidence,
registry version, and model identity. Success, disagreement, timeout, invalid
output, and skipped calls are recorded separately under `selection_trace.shadow`
and in structured logs. `applied_lead` remains deterministic in every case;
the provenance UI explicitly says the shadow result did not alter the page.

When `jev.shadow_log_path` is configured, Prism appends versioned evaluation
records as mode-0600 JSONL. Records contain public ledger identity, semantic
bands, eligible IDs, deterministic and shadow decisions, probabilities,
latency, tokens, and behavior versions. They contain no API keys, raw ledger
payloads, account data, or generated prose. `prism jev evaluate-ledgers`
deduplicates identical versioned ledger evaluations and reports agreement,
confidence bands, disagreement pairs, statuses, latency, token totals, human
label matches, and a prioritized review queue.

## Authoritative references

- Stellar ledger structure: <https://developers.stellar.org/docs/learn/fundamentals/stellar-data-structures/ledgers>
- Operations and transaction atomicity: <https://developers.stellar.org/docs/learn/fundamentals/transactions/operations-and-transactions>
- Fees, resource limits, and metering: <https://developers.stellar.org/docs/learn/fundamentals/fees-resource-limits-metering>
- Contract state archival and TTL: <https://developers.stellar.org/docs/learn/fundamentals/contract-development/storage/state-archival>
- Data API scope and retention: <https://developers.stellar.org/docs/data/apis>
