# Prism Mainnet Evidence Browser Acceptance

Date: 2026-07-31
Status: Accepted locally against live mainnet; Prism production deployment not performed
Scope: Mainnet homepage evidence plus Emergency E1A, E1B, and E1C Prism consumption

## Result

The focused browser corpus passed 24 of 24 assertions against a fresh local Prism build configured for the live mainnet Gateway. There were no browser console errors, page errors, failed same-origin HTTP responses, or mobile horizontal-overflow failures.

The homepage's independently loaded evidence fragments all resolved in 1.373 seconds during the final cold browser pass. The accepted mainnet snapshot rendered:

- an authoritative-empty `What changed` state, with no invented insight;
- four contract archival-attention rows;
- four busiest-contract rows;
- one independently usable smart-contract capacity metric from the API's typed partial component;
- no demo or synthetic fixture evidence.

One capacity metric is the truthful mainnet state for this acceptance. Prism preserves the available measurement instead of suppressing the section when another ledger-specific source limit is unavailable.

## Emergency evidence acceptance

### E1A transaction outcome

Pinned transaction:

`134647cd39de32778a66066552ac6cd058af8148de94ce01f9638c126806530e`

Prism rendered explicit `Failed` text, a non-green failed operation state, the broad general-failure category, and the default-view diagnostics limitation. It did not infer an exact Soroban cause from incomplete diagnostics.

### E1B identity and SAC mapping

Pinned mapping:

- SAC contract: `CA26IEX25MPYPDRQHKMUYWEZ33IG6KWOFUBWOMKZB3O2X67KSEN2WXTR`
- classic asset: `PP5968G:GBNGLBXTVIAPESU2KEUDU5DE3UYI5AUQX33WPKHPO6KKDKAR42CFKUTA`

Forward lookup opened the linked classic asset. Reverse lookup disclosed the matching SAC. Both paths exposed verified token-registry identity and the serving watermark.

### E1C exact Explorer filtering

Pinned mainnet query:

- ledger: `63716372`;
- transaction: `439aad9654c2f2a0b42d142f90259a79c0c5bd51ee7b8f3f834af3b9b3cff654`;
- contract: `CCW67TSZV3SSS2HXMBQ5JFGCKJNXKZM7UQUWUZPUTHXSTZLEO7SJMI75`;
- function: `transfer`;
- asset: `USDC:GA5ZSEJYB37JRC5AVCIA5MOP4RHTM335X2KGX3IHOJAPP5RE34K4KZVN`;
- outcome: `success`.

The combined exact query returned two rows. Both visibly reported `Success`, both belonged to ledger 63,716,372, and the page disclosed serving-only evidence and coverage. The same result set remained inside a 390-pixel viewport.

## Prism resilience added during acceptance

The mainnet serving repair moved normal `/home/summary` latency safely below one second, but a transient refresh failure could still blank all four summary-backed fragments. Prism now:

1. gives the API's 2.5-second fail-closed query budget a 3-second fragment deadline;
2. retains a successful home-summary packet for at most two minutes, separately from the 15-second fresh cache and 10-second negative cache;
3. marks retained packets with local-only delivery metadata that is never serialized as Query API evidence;
4. renders retained populated sections as `Delayed` with an explicit last-snapshot warning;
5. refuses to present a retained empty result as a current authoritative empty state;
6. suppresses relative TTL countdowns and urgency color on retained packets while preserving the absolute live-until ledger.

Focused Gateway and handler regression tests cover successful fallback, negative-cache reuse, the no-last-good failure path, delayed section state, retained-empty truthfulness, and stale TTL handling.

## Verification

- `CGO_ENABLED=1 make test`: passed, including the race detector.
- standalone `CGO_ENABLED=0 go build`: passed.
- focused mainnet browser corpus: 24/24 passed.
- `go vet ./...`: remains blocked by six pre-existing unkeyed `LedgerSpotlightCard` literals in generated `internal/templates/v2/pages/ledger_detail_templ.go`; the homepage and Gateway changes introduced no new vet finding.

Temporary browser artifacts:

- `/tmp/prism-mainnet-evidence-browser-acceptance/report.json`
- `/tmp/prism-mainnet-evidence-browser-acceptance/home-mainnet-desktop.png`
- `/tmp/prism-mainnet-evidence-browser-acceptance/home-mainnet-mobile.png`
- `/tmp/prism-mainnet-evidence-browser-acceptance/e1a-mainnet-failure.png`
- `/tmp/prism-mainnet-evidence-browser-acceptance/e1b-mainnet-reverse-sac.png`
- `/tmp/prism-mainnet-evidence-browser-acceptance/e1c-mainnet-exact.png`

## Remaining gates

- Do not implement the affected-state archival redesign until the API owner freezes and ships the versioned archival-evidence packet and fixtures.
- Exercise the last-good browser presentation against a controlled live failure or replay endpoint; deterministic Gateway and handler coverage already passes.
- Complete the remaining Slice 5 insight-detail work and Slice 6 product-evidence/obsolete-feed cleanup.
- Repair the existing generated ledger-detail vet findings.
- Deploy Prism only after the current shared worktree changes are intentionally separated and reviewed.
