# Prism E1B Testnet Acceptance

Date: 2026-07-28
Status: Prism consumer accepted on testnet; full cross-system corpus blocked by two API evidence regressions

## Scope

This acceptance exercised Prism's `entity_search_v1` consumer through the running local UI at `http://localhost:3002/v2/home` with the testnet Gateway configuration.

The browser report is stored at:

`/tmp/prism-e1b-browser-acceptance/report.json`

Screenshots are stored at:

- `/tmp/prism-e1b-browser-acceptance/desktop-usdc.png`
- `/tmp/prism-e1b-browser-acceptance/desktop-fuzzy.png`
- `/tmp/prism-e1b-browser-acceptance/mobile-usdc.png`

## Prism acceptance

All 18 Prism-owned browser assertions passed. The run recorded no console errors, page errors, or failed local HTTP requests.

Accepted behavior:

- ambiguous `USDC` preserved distinct issuer-specific assets and omitted the issuer-free shortcut;
- `has_more`, exact symbol matching, verification status, identity source, and serving watermark were visible;
- suggest and submit both resolved ambiguous `USDC` to `/v2/explore?asset=USDC`;
- `USDCC` used asset and SAC type filters, returned asset identities instead of pool-name noise, and required an explicit fuzzy-result click;
- pool prefix `001041ac` rendered a unique prefix match with an explicit `Explore` action;
- the native SAC resolved to canonical `XLM`;
- an exact classic-asset slug outranked its embedded issuer account;
- an exact account used typed account identity evidence;
- authoritative empty was distinct from unavailable;
- the primary action was keyboard reachable;
- the suggestion surface had no horizontal overflow at a 390px viewport;
- desktop and mobile rendering stayed within Prism's current restrained palette and product component vocabulary.

## API evidence findings

Two cross-system assertions remain open. Prism's fixture and mapping tests pass for both, but the live testnet `entity_search_v1` packets did not supply the required evidence.

### 1. Known SAC contract did not resolve as a SAC

Query:

`CAW2SVC7HTEFP64JVQSHIZNOYCOKPE54IPCSAD3AKG2ZYMUWQFQB7KVH`

Expected live evidence:

- `entity_kind: sac`;
- canonical classic-asset slug;
- verified token-registry identity.

Observed through Prism:

- no exact E1B entity result was returned;
- Prism retained its syntax-safe fallback and offered `Open contract` with rule `entity.contract`;
- Prism did not invent a SAC mapping.

### 2. Known classic asset omitted reverse SAC evidence

Query:

`USDC:GCYG5OOZY4O2EZOY7OPT4FYY2XWZQ3WCX6M24CVWWHTV67ATKAVK77QC`

Expected live evidence:

- exact `classic_asset` result;
- `details.sac_contract_id` populated with the mapped SAC contract.

Observed through Prism:

- the exact classic asset resolved correctly;
- the result did not include a usable `sac_contract_id`, so Prism could not disclose the reverse mapping.

## API follow-up

The API owner should rerun the accepted E1B SAC forward/reverse corpus against the currently deployed testnet projection and Gateway, then verify:

1. the SAC map, canonical identity table, and denormalized search table share a complete watermark;
2. the known SAC row remains present in `serving.sv_entity_search_current` with `entity_kind = 'sac'` and the canonical asset slug;
3. the corresponding classic-asset search metadata still contains `sac_contract_id`;
4. repeated and comma-separated `contract,sac,protocol_contract` and `asset,sac` filters preserve those rows;
5. a later projector publish did not replace the previously accepted SAC-enriched snapshot with a search snapshot missing the join.

After those two packets are corrected, rerun the existing browser script. No Prism code change should be necessary because fixture tests already cover both mappings.
