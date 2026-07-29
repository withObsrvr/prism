# Prism Homepage Evidence Testnet Acceptance

Date: 2026-07-29  
Environment: local Prism at `http://localhost:3002`, testnet Gateway  
Scope: Slice 4 preview plus the TTL, leaders, and utilization portions of Slice 6

## Outcome

The homepage evidence fragments are implemented and accepted against testnet. Browser acceptance passed 14 of 14 assertions with no console errors, page errors, or failed HTTP requests.

The live testnet state observed during acceptance was:

| Section | API state | Prism result |
|---|---|---|
| What changed | Authoritative empty | Intentional “No material changes” state; not presented as an outage |
| Nearing archival | Partial with evidence | Four rows remained visible with absolute expiration ledgers and a scoped warning |
| Most called, 24 hours | Partial with evidence | Four ranked rows remained visible with calls, callers, and explicit success/failure counts |
| Network utilization | Partial with two usable metrics | Both usable metrics rendered independently; the missing metric did not suppress them |

All four sections disclosed the shared summary snapshot ledger. Live mode did not fall back to mock facts.

## Rich insight coverage

Because the current testnet summary is authoritatively empty, the populated `home_insight_evidence_v1` path was accepted with an explicitly labeled mock fixture. It verifies:

- deterministic failure-spike narration;
- observed and baseline values;
- contributor concentration supplied by the packet;
- interpretation rule disclosure;
- typed evidence-link generation;
- identity, caveat, provenance, and window rendering;
- populated TTL, leader, and utilization presentation.

Unknown evidence versions fall back to measured facts rather than unsupported interpretation. Invalid or internally inconsistent packets are rejected from the ready state.

## Browser assertions

1. The ledger timeline rendered 60 columns.
2. The insight fragment rendered evidence or an authoritative empty state.
3. The empty insight set was not presented as an outage.
4. Four TTL rows rendered with absolute expiration ledgers.
5. Four leader rows rendered with explicit outcome text.
6. Two independently available utilization metrics rendered.
7. All four evidence sections disclosed their snapshot ledger.
8. Live testnet rendered no synthetic data labels or mock facts.
9. The 390-pixel mobile viewport had no horizontal overflow.
10. Evidence remained visible on mobile.
11. The rich demo applied the deterministic failure rule.
12. Every demo evidence section was populated.
13. Demo data was visibly labeled in the shell and evidence fragments.
14. The interpretation rule and evidence link were visible.

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
- the 14-assertion homepage browser corpus;
- the previously accepted E1A, E1B, and E1C browser corpora.

Regression coverage includes all three deterministic insight rules plus ready, authoritative empty, partial, stale, unavailable, invalid, unknown-type, unknown-version, independent utilization metrics, cache sharing, and the prohibition on live mock fallback.

Repository-wide `make vet` still reports pre-existing unkeyed `LedgerSpotlightCard` literals generated from `internal/templates/v2/pages/ledger_detail.templ`; that source is unchanged by this implementation. The packages changed for homepage evidence pass vet.

## Remaining acceptance

- Accept a populated live insight packet after the insight projection is producing rows in the promoted environment.
- Repeat homepage evidence acceptance against mainnet after the API promotion.
- Exercise controlled live partial, stale, and unavailable packets when the API owner can replay or induce those states safely; fixture and handler coverage already pass.
- Complete the remaining product-evidence and rollout-cleanup items in Slice 6.

No Prism production or mainnet deployment was performed as part of this acceptance.
