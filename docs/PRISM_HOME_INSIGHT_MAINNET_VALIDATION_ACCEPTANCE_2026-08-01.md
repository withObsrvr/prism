# Prism Home Insight Mainnet Validation Acceptance

Date: 2026-08-01

Scope: Prism's live mainnet homepage evaluation validation and browser rendering

Deployment: local Prism build against the live mainnet Query API through Gateway

## Outcome

Prism now accepts the API's documented `qualifying_evidence_partial` state. In this
state, the detector completed its numerical evaluation and crossed its threshold, while
only the optional qualifying-subject evidence is incomplete.

Before this repair, Prism incorrectly required every `partial` rule to use
`evaluation_outcome: source_partial`. It consequently rendered valid current insight
cards alongside the false diagnostic `The detector evaluation packet failed Prism's
evidence checks.`

The validator now distinguishes:

- a partial source evaluation (`source_partial`);
- a completed, threshold-crossing evaluation whose qualifying evidence is partial
  (`evaluated` plus `qualifying_evidence_partial`);
- an unavailable source (`source_unavailable`).

The exception is deliberately narrow. A partial/evaluated rule is accepted only when it
crossed its threshold and carries the exact qualifying-evidence caveat. Ready source
failures, arbitrary partial caveats, and unavailable evaluated results remain rejected.

## Live mainnet acceptance

The live mainnet summary was fresh and returned the documented partial evaluation shape.
At the time of browser acceptance, Prism rendered two current failure insight cards and
two retained-evidence links.

Desktop (1440 pixels) and mobile (390 pixels) assertions passed:

- the homepage returned HTTP `200`;
- every current insight had a retained-evidence link;
- measured `Last hour`, `Typical hour`, and `Change` values remained visible;
- the legitimate `Limited` status and per-card coverage notes remained visible;
- the false detector-evaluation rejection was absent;
- both retained insight detail routes returned HTTP `200`;
- no horizontal overflow, console errors, or page errors occurred.

Artifacts:

- `/tmp/prism-mainnet-insight-check-20260801/report.json`
- `/tmp/prism-mainnet-insight-check-20260801/home-1440.png`
- `/tmp/prism-mainnet-insight-check-20260801/home-390.png`

## Automated verification

Passed:

```text
go test ./...
CGO_ENABLED=1 go test -race ./...
go build ./...
git diff --check
```

`go vet ./...` remains blocked by six pre-existing unkeyed `LedgerSpotlightCard`
literals in generated `internal/templates/v2/pages/ledger_detail_templ.go`. This repair
adds no vet finding.

