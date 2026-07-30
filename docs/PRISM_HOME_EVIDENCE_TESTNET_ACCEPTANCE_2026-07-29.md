# Prism Homepage Evidence Testnet Acceptance

Date: 2026-07-29  
Environment: local Prism at `http://localhost:3002`, testnet Gateway  
Scope: Slice 4 preview plus the TTL, leaders, and utilization portions of Slice 6

## Outcome

The homepage evidence fragments are implemented and accepted against testnet. Browser acceptance passed 25 of 25 assertions with no console errors, page errors, or failed HTTP requests.

The live testnet state observed during acceptance was:

| Section | API state | Prism result |
|---|---|---|
| What changed | Partial with deployment evidence | One supported deployment signal rendered with a compact interpretation, three comparable facts, a proof link, and its coverage caveat available on demand |
| Contract data expiring soon | Partial with evidence | Four rows remained visible with plain-language ledger expiration and an on-demand health disclosure |
| Busiest contracts, 24h | Partial with evidence | Four ranked rows remained visible with aligned calls, callers, failure percentages, and top functions |
| Smart contract capacity | Partial with two usable metrics | Contract computation and contract state access rendered independently; the missing metric did not suppress them |

Live mode did not fall back to mock facts. Repeated component-snapshot labels are intentionally omitted from the homepage; the summary packet remains the source of truth, while item-level proof links and contract-detail destinations carry the deeper evidence path.

## Rich insight coverage

The populated testnet `home_insight_evidence_v1` path now supplies a live deployment signal. The explicitly labeled demo fixture continues to exercise the failure-spike path. Together they verify:

- deterministic failure-spike narration;
- observed and baseline values in the generated interpretation;
- contributor concentration supplied by the packet;
- typed evidence-link generation;
- identity, caveat, provenance, and window preservation in the view model;
- a distilled preview with `Last hour`, `Typical hour`, and `Change` facts;
- a single-result horizontal evidence row and a three-result wide-screen comparison composition;
- collapsed coverage notes and suppression of a redundant network subject;
- omission of implementation-rule and repeated provenance metadata;
- populated TTL, leader, and utilization presentation.

Unknown evidence versions fall back to measured facts rather than unsupported interpretation. Invalid or internally inconsistent packets are rejected from the ready state.

## Browser assertions

1. The ledger timeline rendered 60 columns.
2. The insight fragment rendered evidence or an authoritative empty state.
3. Resolved insights were not presented as an outage.
4. A populated insight used the compact, two-column evidence-row hierarchy.
5. Three simultaneous changes used the prototype’s wide-screen comparison composition without overflow.
6. Four expiration rows rendered with plain-language ledger evidence.
7. Four leader rows rendered comparable failure percentages.
8. Two independently available smart-contract capacity metrics rendered with user-facing labels.
9. The homepage did not repeat component-snapshot provenance.
10. Every archival and leader row linked directly to its contract-detail page.
11. Desktop evidence used the asymmetric 7/5 comparison layout.
12. Leader values had stable comparison headers.
13. The companion contract tables shared the same header and row tracks, including a deliberately long USDC-style identity with no cell collision.
14. Component-health details were available without interrupting the default scan path.
15. Opening a compact health disclosure revealed its evidence caveat.
16. Redundant category headings, repeated badges, and instructional link text were absent.
17. Live testnet rendered no synthetic data labels or mock facts.
18. The evidence pair stacked in reading and keyboard order at compact-desktop widths before the five-column leader table compressed.
19. The 390-pixel mobile viewport had no horizontal overflow.
20. Evidence remained visible on mobile.
21. Mobile search remained at or below 240 pixels high.
22. The rich demo applied the deterministic failure rule.
23. Every demo evidence section was populated.
24. Demo data was visibly labeled in the shell and evidence fragments.
25. The insight preview retained its proof path without exposing implementation metadata.

Artifacts:

- `/tmp/prism-home-evidence-browser-acceptance/report.json`
- `/tmp/prism-home-evidence-browser-acceptance/testnet-desktop.png`
- `/tmp/prism-home-evidence-browser-acceptance/testnet-mobile.png`
- `/tmp/prism-home-evidence-browser-acceptance/demo-rich-insight.png`

## Automated verification

The following gates pass:

- `CGO_ENABLED=1 go test ./... -race -count=1`;
- `make build`;
- scoped `go vet` for the gateway, handlers, insight, intent, and transaction-outcome packages;
- `git diff --check`;
- the 25-assertion homepage browser corpus;
- the previously accepted E1A, E1B, and E1C browser corpora.

Regression coverage includes all three deterministic insight rules plus ready, authoritative empty, partial, stale, unavailable, invalid, unknown-type, unknown-version, independent utilization metrics, cache sharing, and the prohibition on live mock fallback.

Repository-wide `make vet` still reports pre-existing unkeyed `LedgerSpotlightCard` literals generated from `internal/templates/v2/pages/ledger_detail.templ`; that source is unchanged by this implementation. The packages changed for homepage evidence pass vet.

## Compact-layout acceptance

The homepage density pass was compared in Chromium against the reference prototype at matching viewports:

| Viewport | Reference prototype | Prism before | Prism after |
|---|---:|---:|---:|
| 1440 px desktop | 2,219 px | 3,369 px | 1,511 px |
| 390 px mobile | 3,693 px with horizontal overflow | 4,451 px | 2,404 px with no horizontal overflow |

The reduction came from shorter section intervals, a shallower spectrogram, a 7/5 evidence composition, compact health disclosures, comparison-density contract rows, and compact utilization gauges. A single “What changed” item now uses a 186-pixel horizontal evidence row instead of a 276-pixel narrow article; two or three items use the prototype’s flat comparison columns above 1024 pixels. The leader module receives 723 pixels and the expiration module 517 pixels at the 1440-pixel test viewport instead of forcing both across 1280 pixels. The two contract modules share 29-pixel header tracks and 64-pixel row tracks; expiration values use a protected numeric column, while long contract identities truncate with their complete value retained as title text. At 1024 pixels and below, the comparison sections stack before their content compresses. Mobile search was reduced from 433 pixels to 227 pixels. The layout adopts the useful structural traits of the Orb reference: identity-led navigation, restrained first-screen facts, and detail-page disclosure, without importing Orb's palette or visual styling.

The homepage retains every evidence row, explicit failure label, caveat, expiration value, utilization value, and proof destination. It omits repeated category kickers, repeated snapshot/provenance labels, identity-source metadata, exact success/failure count pairs, ranking timestamps, duplicate generated identities, tracked-entry prose, utilization source prose, and standalone “inspect” instructions. Full identifiers remain available as title text, exact outcome counts are reserved for detail surfaces, and expiration and leader rows are whole-row links to the contract page.

## Remaining acceptance

- Repeat homepage evidence acceptance against mainnet after the API promotion.
- Exercise controlled live partial, stale, and unavailable packets when the API owner can replay or induce those states safely; fixture and handler coverage already pass.
- Complete the remaining product-evidence and rollout-cleanup items in Slice 6.

No Prism production or mainnet deployment was performed as part of this acceptance.
