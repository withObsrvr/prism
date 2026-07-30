# Prism Contract-State Archival Experience Plan

Date: 2026-07-29  
Status: Proposed design brief, awaiting confirmation  
Owner: Prism  
API dependency: `PRISM_CONTRACT_STATE_ARCHIVAL_EVIDENCE_API_PLAN_2026-07-29.md`  
Related plan: `prism-home-live-data-implementation-plan-2026-07-25.md`

## 1. Feature summary

Replace the homepage's contract-level TTL countdown with a truthful contract-state
archival explanation. The default view must tell a person which contract is affected,
which state is affected, what loss of availability means, and when it may happen.
Exact ledger evidence remains visible without requiring Stellar storage expertise.

The homepage stays compact. Entry-level proof, deadline distributions, restoration
history, and provenance belong on the contract page.

## 2. Primary user understanding

After scanning one row, a user should understand:

> Some state associated with this contract may require restoration soon. The contract
> itself is not necessarily expiring, and this is the exact final live ledger.

The interface must never imply that the whole contract expires when the evidence only
describes one or more storage entries.

## 3. Design direction

- **Register:** product.
- **Color strategy:** Restrained. Neutral table structure carries the layout. Amber
  marks approaching archival, and red is reserved for confirmed unavailability or
  permanent deletion. Status always has explicit text.
- **Physical scene:** a developer or investigator uses Prism on a normal work display
  to assess a live network condition quickly, then opens a contract to verify the
  exact state evidence.
- **Anchors:** Prism's current homepage mock for information hierarchy, Orb Markets
  for compact comparison density, and Stripe-style evidence tables for predictable
  scanning.
- **Typography:** Instrument Sans for explanations. JetBrains Mono with tabular
  numerals for ledger sequences, entry counts, and contract addresses.

This is a focused refinement of an existing surface, so it does not need a new visual
direction probe.

## 4. Scope

- **Fidelity:** production-ready design and implementation plan.
- **Breadth:** homepage archival section, contract storage detail, and the shared
  presentation logic that turns archival evidence into user-facing states.
- **Interactivity:** live ledger-relative status on the homepage, navigable rows, and
  entry-level inspection on the contract page.
- **Responsive behavior:** desktop comparison table, structurally reduced mobile
  rows, no horizontal overflow.
- **Accessibility:** WCAG AA, keyboard-reachable rows, visible focus, non-color status,
  and screen-reader labels that state affected scope and availability.

Out of scope for this slice:

- TTL extension transactions or wallet signing;
- notification subscriptions and watchlists;
- temporary storage on the homepage;
- LLM-generated archival explanations;
- semantic labels not supported by API evidence.

## 5. Information architecture

### 5.1 Homepage

Rename the section:

`Contract state nearing archival`

Use one compact table with these columns:

| Contract | Affected state | Availability |
|---|---|---|
| Resolved name plus short contract address | Exact count and proven state kind | Human consequence and time, followed by exact ledger evidence |

Illustrative row:

```text
Soroswap Router          3 persistent data entries      May need restoration in about 10 sec
CCESWLP4...EIU7QO                                        2 ledgers, final live ledger 3,871,198
```

If the API supplies a proven semantic classification, the affected-state cell can
say `3 balance entries` or `12 liquidity positions`. Otherwise it must say
`persistent data entries`. Prism must not infer meaning from an opaque storage key.

The whole row is an anchor to the contract page. Do not add a repeated `Inspect`
instruction or a separate card CTA.

### 5.2 Contract page

Place an archival summary at the top of the existing Storage tab. Do not add a new
dashboard card grid.

The summary answers:

1. Is relevant state live, on its final live ledger, archived, or unknown?
2. Does the evidence concern persistent data, the contract instance, contract code,
   or a mixture?
3. What is the user-facing consequence?
4. What is the earliest final live ledger?
5. How many entries share that deadline?

Below the summary, add an evidence table sorted by earliest deadline:

| State | Kind | Availability | Final live ledger | Evidence |
|---|---|---|---|---|
| Proven semantic label or safe fallback | Persistent, instance, or code | Live, final ledger, archived, restored, or unknown | Exact sequence | Safe key fingerprint and source state |

The table supports bounded pagination. It must not imply that a returned page is the
complete set unless the API proves completeness.

## 6. Presentation rules

### 6.1 Meaning by state kind

| API state kind | Homepage wording |
|---|---|
| `persistent_data` | `Stored data may need restoration` |
| `contract_instance` | `Contract calls may require restoration` |
| `contract_code` | `Contract code may require restoration` |
| multiple restorable kinds | `Multiple kinds of contract state may need restoration` |
| unknown restorable kind | `Contract state may need restoration` |
| `temporary_data` | Never eligible for the homepage; detail may say `Temporary data will be permanently deleted` |

### 6.2 Ledger-relative availability

Prism derives the live status from the page heartbeat ledger and the absolute final
live ledger. It does not continue displaying a snapshot-relative countdown after the
network advances.

| Condition | Primary wording | Supporting evidence |
|---|---|---|
| More than the imminent threshold remains | `May need restoration in about 4 hours` | `2,880 ledgers, final live ledger 3,874,000` |
| More than one ledger remains and no trustworthy time estimate exists | `May need restoration soon` | `12 ledgers, final live ledger 3,871,210` |
| One ledger remains | `Final live ledger next` | `Final live ledger 3,871,198` |
| Current ledger equals final live ledger | `On its final live ledger` | `Final live ledger 3,871,198` |
| Current ledger passed the deadline but archival evidence has not caught up | `May already require restoration` | `Archival evidence is 2 ledgers behind` |
| API confirms archival | `Restoration required` | Exact archived-at evidence |
| API confirms extension | Remove the warning on refresh | New final live ledger remains on contract detail |
| Required evidence is missing | `Availability unknown` | Scoped caveat, never a synthetic zero |

Approximate wall-clock time is shown only when a cadence value and its provenance are
present. Always retain the exact final live ledger.

### 6.3 Counts

Homepage counts refer only to entries sharing the earliest displayed deadline:

- `1 persistent data entry`;
- `3 persistent data entries`;
- `Contract instance and code`.

Do not use the current attention-window count as if all entries expire at the nearest
deadline. Total tracked entries and total entries inside the attention window remain
contract-detail evidence.

### 6.4 Color and emphasis

- Neutral: availability is not imminent or exact state is unknown.
- Amber: state is approaching archival.
- Red: archival is confirmed, access is unavailable, or temporary data deletion is
  confirmed.
- Emerald: a previously at-risk entry is confirmed extended or restored.

Color supplements the state label. It never replaces it.

## 7. Interaction model

1. The homepage heartbeat continues to update from the recent-ledger fragment.
2. Every archival row carries machine-readable `as_of_ledger`,
   `final_live_ledger`, state kind, and count-at-deadline values.
3. A small homepage controller recomputes only presentation state whenever the
   heartbeat ledger changes. It does not invent new source evidence.
4. When the heartbeat passes the evidence snapshot, the controller can state
   `May already require restoration`, but not `Archived`, until the archival source
   confirms it.
5. The archival fragment uses adaptive refresh: normal cadence outside the imminent
   window, faster bounded refresh while any row is imminent.
6. Activating a row opens the contract page's Storage tab with the archival summary
   in view.
7. Hover and focus use the same information. No essential explanation exists only in
   a tooltip.

## 8. Key states

### Ready

Render up to four ranked contracts from one coherent component snapshot. Show an
exact affected-state count and final live ledger.

### Authoritative empty

`No restorable contract state is inside the current attention window.`

This state is valid only when the API reports complete coverage through the component
snapshot.

### Partial

Keep valid rows visible. Replace any over-precise live countdown with snapshot-safe
wording and disclose the scoped limitation.

### Stale

Keep absolute ledger evidence visible. Use `May already require restoration` when
the current heartbeat has passed a displayed deadline. Never continue a stale
positive countdown.

### Unavailable

`Contract-state availability is temporarily unavailable.` Keep the rest of the
homepage usable.

### Unknown classification

Use `Contract state` or `persistent data entries`. Do not guess a balance, position,
allowance, or configuration label.

### Extended or restored between snapshots

The next confirmed packet replaces the old state. Prism must not animate or preserve
an obsolete warning as historical truth on the homepage.

## 9. Prism implementation slices

### TTL-P0: Contract consumption and pure presentation model

1. Add versioned Gateway types for the enriched homepage preview and archival detail.
2. Preserve nullable zero-ledger semantics. Zero is a valid final-live-ledger state,
   not missing evidence.
3. Add a pure resolver that accepts component snapshot, heartbeat ledger, absolute
   deadline, affected kinds, and archival status, then returns presentation copy and
   tone.
4. Add fixtures for persistent data, instance, code, mixed state, semantic unknown,
   final live ledger, confirmed archived, partial, stale, and unavailable packets.

Exit gate: every user-facing claim is reproducible from a fixture and no copy treats
an attention-window count as count-at-deadline.

### TTL-P1: Homepage row redesign and live reconciliation

1. Replace `Contract data expiring soon` and `Expires in` with the information
   architecture in this brief.
2. Add the affected-state column and consequence-first availability copy.
3. Add the exact final live ledger as persistent evidence.
4. Recompute ledger-relative presentation from the live heartbeat.
5. Add adaptive refresh for imminent rows and safe stale transitions.
6. Keep desktop rows aligned with the busiest-contract table; structurally collapse
   affected state into the primary column on narrow screens.

Exit gate: a browser test advances the heartbeat across an entry deadline without
ever displaying a positive remaining count after that deadline.

### TTL-P2: Contract-detail archival evidence

1. Add one typed Gateway client call for the archival-detail endpoint.
2. Integrate the summary and evidence table into the existing Storage tab.
3. Render exact entry classification, deadline, status, safe key evidence, and
   provenance.
4. Add bounded pagination and authoritative partial-result treatment.
5. Preserve the current storage sample when archival detail is unavailable; one
   failed component must not erase unrelated contract detail.

Exit gate: every homepage row has a proof path that explains the affected entries
without requiring raw XDR interpretation.

### TTL-P3: Acceptance and rollout

1. Unit-test the presentation state machine and copy matrix.
2. Add handler and template tests for all component states.
3. Run browser acceptance while testnet ledgers advance through at least one displayed
   deadline.
4. Verify desktop, compact desktop, and mobile geometry.
5. Verify keyboard and screen-reader labels.
6. Accept direct API and Gateway parity fixtures before mainnet promotion.
7. Record testnet evidence, then repeat acceptance on mainnet after the API rollout.

## 10. Expected Prism files

- `internal/gateway/types.go`
- `internal/gateway/client.go`
- `internal/gateway/client_test.go`
- `internal/handlers/home_v2_evidence.go`
- `internal/handlers/home_v2_evidence_test.go`
- `internal/templates/v2/viewmodel/types.go`
- `internal/templates/v2/pages/home_evidence.templ`
- `internal/templates/v2/pages/home_test.go`
- `internal/templates/v2/pages/contract_detail.templ`
- contract-detail handler and viewmodel files selected by the existing route
- `web/static/css/v2-unified.css`
- focused browser-acceptance script and frozen acceptance report

## 11. Acceptance assertions

1. Prism never says a contract expires when only associated entries are at risk.
2. The homepage count is the exact number of entries at the displayed nearest
   deadline.
3. `current_ledger > final_live_ledger` never renders a positive countdown.
4. `current_ledger == final_live_ledger` renders `On its final live ledger`.
5. Lagging evidence cannot become a confirmed archived claim.
6. Confirmed archived, extended, restored, temporary-deleted, and unknown states are
   distinct.
7. Semantic storage labels appear only with evidence source and confidence.
8. Exact ledger evidence is visible without opening a tooltip.
9. Homepage rows use one bounded summary response and never fan out per contract.
10. The contract page distinguishes returned-page completeness from complete
    projection coverage.
11. Every state has explicit text and passes color-blind inspection.
12. No homepage row exceeds the established compact evidence-row height at supported
    breakpoints.

## 12. Recommended implementation references

- Impeccable product guidance for predictable table and state behavior.
- Prism `PRODUCT.md` for meaning-first, evidence-second narration.
- Prism `DESIGN.md` for restrained color, mono evidence, and compact table structure.
- Stellar state archival semantics for the final-live-ledger boundary and restoration
  behavior.

## 13. Confirmation gate

Confirm this brief before TTL-P0 begins. The API plan can proceed through contract
freeze and fixtures in parallel, but Prism implementation should consume the frozen
packet rather than an illustrative shape.
