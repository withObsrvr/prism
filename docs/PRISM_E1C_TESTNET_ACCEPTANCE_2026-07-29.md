# Prism E1C Testnet Acceptance

Date: 2026-07-29
Status: Accepted on testnet; mainnet pending

## Accepted boundary

Prism now consumes the frozen `explorer_events_v1` contract through the authenticated testnet Gateway. Live Explore results, matched counts, coverage, and pagination all come from the same serving-only filter set.

Implemented behavior:

- typed decoding and validation for evidence version, packet status, coverage, provenance, count caps, warnings, normalized functions, normalized assets, normalized actors, sender, recipient, and cursor;
- exact server-side event type, function, asset, actor, outcome, time-window, ledger, contract, transaction, and contract-name filters;
- no live `topic_match` use, client-side row filtering, cold lookup, token-transfer fallback, or generic-event fallback;
- distinct ready, authoritative empty, partial, unavailable, and invalid-query presentation;
- visible serving coverage and serving-only provenance;
- explicit success or failure text derived from transaction-scoped outcome evidence;
- issuer-safe asset filtering: exact `XLM`, `CODE:G…`, and `C…` identifiers are accepted, while a bare code requests more detail;
- ledger-bounded queries default to the API's retained serving coverage unless the user explicitly selects a close-time window;
- active exact filters preserved in stable-cursor pagination URLs;
- mobile wrapping for long hashes, assets, actors, query sentences, and filter chips.

## Live browser corpus

The browser report is stored at:

`/tmp/prism-e1c-browser-acceptance/report.json`

Screenshots are stored at:

- `/tmp/prism-e1c-browser-acceptance/exact-desktop.png`
- `/tmp/prism-e1c-browser-acceptance/exact-mobile.png`
- `/tmp/prism-e1c-browser-acceptance/invalid-desktop.png`

All 16 assertions passed with no console errors, page errors, or failed local HTTP requests.

### Combined exact fixture

The accepted API fixture was exercised through the running Prism UI using:

- transaction `a62200c58d597786923f447ee9b9096a958fde4955a5c1cb848732fd03f08961`;
- event type and normalized function `transfer`;
- asset `USDC:GCKIUOTK3NWD33ONH7TQERCSLECXLWQMA377HSJR4E2MV7KPQFAQLOLN`;
- actor `CBHHWFAYLB3SXJCE232DC6WNSK74IBEOROAGCI2AFBA2H5NQOH2KYKNN`;
- successful outcome;
- exact ledger range `3,852,615` to `3,852,615`;
- a 24-hour close-time window.

Prism rendered two events, `2 matched`, two explicit `Success` statuses, ledger `3,852,615` on both rows, current serving coverage, and serving-only provenance. It did not render fixture or unavailable copy.

### Exact failure filter

The corpus also queried failed transaction `833fdce52b326521a048f93f1556ead1bbc7b5f9967fb60c9f1a1f142cdf1ba3` with transfer, failed-outcome, and exact ledger `3,866,012` filters. The only visible row was explicitly `Failed` and carried the requested ledger.

### State distinctions

- a nonexistent contract-name query rendered `No matches in retained coverage` as an authoritative empty result;
- bare `asset=USDC` rendered `Query needs more detail` and `Query not run`, with no broad substitute query or rows;
- fixture tests cover typed partial and unavailable packets without turning either into an authoritative empty result;
- the exact mobile fixture rendered two rows with `scrollWidth` equal to the 390-pixel viewport.

## Regression acceptance

- the focused E1A failure-explanation browser corpus passes 20/20, including `General failure category` for partial `invoke_host_function_trapped` evidence;
- the updated E1B browser corpus passes fully with both live SAC mappings and no API findings;
- all Go packages pass with the race detector when `CGO_ENABLED=1`;
- the production build passes;
- `git diff --check` passes;
- `go vet` passes for the changed Gateway, handlers, intent, and transaction-outcome packages.

The repository-wide `make vet` command still reports pre-existing unkeyed `LedgerSpotlightCard` literals in generated `ledger_detail_templ.go`. E1C does not modify that template or struct.

## Remaining work

- deploy and accept E1A, E1B, and E1C on mainnet;
- replay partial and unavailable E1C packets through a controlled live environment when one can be induced safely;
- continue with Prism Slice 4 consumption of the structured insight preview and detail APIs.
