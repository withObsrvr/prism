# Prism Insight Detail Browser Acceptance

Date: 2026-07-31

Scope: Prism Slice 5 retained insight detail consumer

Networks: testnet live-state check; frozen testnet contract fixture for populated-state rendering

Deployment: local Prism build only; no Prism testnet or mainnet deployment

## Outcome

The implemented insight detail surface passed all 12 focused browser assertions at
desktop and mobile widths. Invalid, missing, unavailable, and unverifiable evidence
remain distinct and do not fall back to homepage text or synthetic facts.

The deployed testnet homepage insight fragment returned HTTP `200` in `0.347 s`, but
the API currently describes the insights projection as stale and supplies no current
insight card. Prism rendered the unavailable stale-projection state and did not expose
a detail link. The previously frozen failure fixture ID returned `404`, which Prism
rendered as retained evidence not found rather than as an outage. A live populated
testnet detail pass must be repeated the next time the API emits and retains a current
insight.

## Accepted implementation

- Typed Gateway consumption of `GET /home/insights/{insight_id}` with status-sensitive
  positive and negative caching.
- Strict `hiev1_` route validation, network parity, and response-ID parity.
- Stable ID recomputation from network, type, subject, observed window end, and rule
  version before interpretation.
- Common evidence reconciliation plus bounded contributor and representative-sample
  validation.
- Deterministic detail support for failure, contract-deployment, and transaction-
  activity spikes.
- A meaning-first hero, exact rule and threshold explanation, ranked contributors,
  representative transaction pointers, proof links, visible caveats, evidence windows,
  watermarks, and provenance.
- Human-readable result labels such as `Contract stopped unexpectedly`, with the raw
  protocol code retained underneath as technical evidence.
- Network-preserving links to transaction, contract, ledger, and filtered Explore
  surfaces.
- Truthful HTTP and page states: `400` invalid ID, `404` retained packet missing, and
  `503` unavailable or failed evidence verification.

## Browser assertions

The populated page was exercised through the real Prism binary against the Query API
repository's frozen `failure_spike.json` contract fixture served from a local
Gateway-compatible stub. This isolates UI acceptance from the deployed stale
projection without weakening Prism validation.

| Assertion | Result |
|---|---|
| Valid retained evidence returns `200` | Pass |
| Meaning and measured magnitude lead | Pass |
| Rule, baseline, threshold, and match are explained | Pass |
| Contributors show human-readable category, count, and share | Pass |
| Transaction samples are explicitly bounded pointers | Pass |
| Transaction links preserve `network=testnet` | Pass |
| Aggregate proof link preserves exact filters and network | Pass |
| Window, watermark, and provenance remain visible | Pass |
| Desktop uses a compact two-column narrative and evidence rail | Pass |
| Mobile uses one content column, two-up metrics, and no overflow | Pass |
| Invalid IDs fail closed without a meaningless retry | Pass |
| Missing retained evidence is distinct from an outage | Pass |

Observed desktop geometry: two layout columns, aligned narrative and rail, `375 px`
hero height, and no horizontal overflow. Observed mobile geometry at `390 px`: one
layout column, two metric columns, and no horizontal overflow. Playwright reported no
unexpected console or page errors.

## Automated verification

Passed:

```text
templ generate
go test ./internal/gateway ./internal/insight ./internal/handlers ./internal/templates/v2/pages
CGO_ENABLED=1 go test ./... -race -count=1
go build -o /tmp/prism-insight-acceptance .
```

`go vet ./...` remains blocked by six pre-existing unkeyed
`LedgerSpotlightCard` literals in generated `ledger_detail_templ.go`. No new vet
finding points to the insight-detail implementation.

## Remaining acceptance

1. Repeat the populated flow using a current retained insight from the deployed
   testnet API, beginning at the homepage link.
2. Repeat populated and empty behavior on mainnet when a retained mainnet insight is
   available.
3. Exercise deployed partial and stale detail packets when the API can safely replay
   those states.
4. Add historical resource-threshold interpretation only after the API freezes and
   deploys a corresponding evidence type.
