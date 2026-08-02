# Prism Home Insight Evaluation Consumer Acceptance

Date: 2026-07-31

Scope: Prism homepage evaluation/history/delivery consumer and evidence-v2 positive insight detail

Deployment: local Prism build; live testnet registry-v1 API for homepage acceptance; frozen Query API fixtures for evidence-v2 detail acceptance

## Outcome

Prism now consumes the full insight availability contract instead of treating an empty
`insights` array as the only evidence about the completed hour.

The live testnet homepage rendered a complete registry-v1 quiet hour with three supplied
comparisons, one subordinate recent detection link, and the exact evaluated window. The
desktop and mobile browser suite passed every assertion with no horizontal overflow,
console errors, or page errors.

The three frozen evidence-v2 positive packets also passed desktop and mobile detail
acceptance. Prism rendered deterministic positive explanations only after validating the
rule-specific safeguards:

- successful activity growth requires the successful count and ratio threshold while the
  failure rate remains inside both supplied guardrails;
- failure recovery requires failures back inside the normal range while meaningful call
  activity continues;
- new contract adoption requires calls, distinct callers, success rate, and deployment age
  to satisfy the frozen thresholds.

## Implemented contract coverage

- Typed Gateway decoding for `insight_evaluation`, `recent_insights`, `insight_delivery`,
  `family`, `direction`, `severity`, `ratio_comparison`, and the three evidence-v2 fact
  unions.
- Registry-exact evaluation validation for detector registries v1 through v4.
- Reconciliation of evaluation windows, statuses, outcomes, thresholds, ratios, ledger
  bounds, qualifying counts, caveats, and provenance.
- Delivery validation for `current`, `last_good`, and `unavailable` without changing the
  original evaluated window.
- Compact homepage checks for a complete no-threshold hour and one historical `Last
  flagged` link that never masquerades as a current result.
- Evidence-v2 stable ID support in routes, Gateway caching, homepage links, detail
  validation, interpretation, and evidence navigation.
- Positive styling uses Prism's existing semantic green and preserves explicit text status.

## Browser assertions

Live testnet homepage, at 1440 and 390 pixels:

- HTTP `200`;
- zero current cards and exactly three registry-v1 detector checks for the sampled quiet hour;
- explicit `No unusual changes crossed Prism's thresholds` result;
- failures, deployments, and transactions each show current-to-typical evidence;
- exactly one subordinate retained insight link;
- retained detail returns `200` and preserves rule/evidence content;
- no horizontal overflow, console errors, or page errors.

Frozen evidence-v2 details, at 1440 and 390 pixels:

- all three routes return HTTP `200`;
- each page leads with the correct positive meaning and its evidence guard;
- each page exposes `Why Prism flagged this`, explicit evidence status, windows,
  provenance, and aggregate proof navigation;
- no horizontal overflow, console errors, or page errors.

Artifacts:

- `/tmp/prism-current-insight-consumer-acceptance/report.json`
- `/tmp/prism-current-insight-consumer-acceptance/home-1440.png`
- `/tmp/prism-current-insight-consumer-acceptance/home-390.png`
- `/tmp/prism-v2-insight-browser-acceptance/report.json`
- `/tmp/prism-v2-insight-browser-acceptance/`

## Automated verification

Passed:

```text
templ generate
go test ./...
CGO_ENABLED=1 go test -race ./...
go build ./...
```

`go vet ./...` remains blocked by six pre-existing unkeyed `LedgerSpotlightCard`
literals in generated `ledger_detail_templ.go`. This change adds no new vet finding.

## Remaining rollout gates

1. Complete the Query API's ten-hour testnet v1 observation gate.
2. Repeat live homepage and detail acceptance after staged registry v2, v3, and v4 promotion.
3. Calibrate and deploy the same contract to mainnet.
4. Run mainnet populated, quiet, partial, last-good, and unavailable browser acceptance.
