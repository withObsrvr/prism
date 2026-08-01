# Prism Home Insight Availability API Follow-up

Date: 2026-07-31  
Owner: Stellar Query API / insight projection  
Consumer: Prism `/v2/home`

## Status update: testnet API and Prism consumer implemented

As of 2026-07-31, the Query API has implemented the evaluation envelope, recent insight
index, delivery state, registry-aware selection, and the three positive evidence-v2
detectors. Registry v1 is live on testnet with complete 168-hour baselines. Registries v2
through v4 remain disabled while the API completes its ten-hour observation gate and
staged promotion sequence. Mainnet has not received this insight-availability rollout.

Prism now consumes and validates all of these contracts. A complete current evaluation
can substantiate a compact no-threshold result; retained comparisons are labeled delayed;
unavailable or contradictory delivery fails closed; and recent detections remain clearly
historical. Prism also supports deterministic detail interpretation for all three frozen
evidence-v2 positive types.

Remaining work belongs to rollout acceptance rather than contract implementation:

1. Complete the remaining reconciled v1 observation hours on testnet.
2. Run Prism browser acceptance after each staged v2, v3, and v4 registry promotion.
3. Calibrate and deploy the registry sequence to mainnet.
4. Repeat populated, quiet, partial, last-good, and unavailable browser acceptance on mainnet.

## Original problem

`What changed` is only populated when a detector crosses its anomaly threshold. That is
correct for qualifying insights, but `/home/summary` currently gives Prism too little
evidence to make the other hours useful.

Live diagnosis on 2026-07-31 established two different states:

- mainnet returned a `partial` insight component with four valid retained insight rows in
  about 322 ms; Prism rendered the first three;
- testnet returned a `stale` insight component at ledger `3,881,598` with zero retained
  rows while the live homepage was around ledger `3,897,970`, a lag of roughly 16,372
  ledgers at the time of the check.

The request was not failing Prism's three-second deadline. The missing testnet content was
an upstream freshness and retention problem. Prism correctly refused to label snapshot
metrics as a change.

## Original API work (implemented on testnet)

### P0: Restore and monitor the testnet insight projection

1. Advance the testnet insight projector to the serving watermark.
2. Reconcile its completed-hour windows before changing the component from `stale`.
3. Alert on projection ledger lag, completed-hour lag, zero-row runs, and projector
   restart/failure state.
4. Preserve exact `as_of_ledger`, `complete_through_ledger`, last evaluated window, warning
   code, and retryability in `/home/summary`.

A stale component must not be marked ready merely because the query itself succeeds.

### P1: Publish every detector evaluation, not only threshold crossings

Extend `/api/v1/home/summary` with a bounded, versioned evaluation packet for the last
completed hour. This should be part of the existing summary response so Prism does not add
another synchronous homepage dependency.

Suggested shape:

```json
{
  "insight_evaluation": {
    "evidence_version": "home_insight_evaluation_v1",
    "status": "ready",
    "window_start": "2026-07-31T12:00:00Z",
    "window_end": "2026-07-31T13:00:00Z",
    "comparison_method": "rolling_7d_median_prior_complete_hour",
    "complete_through_ledger": 63735168,
    "rules": [
      {
        "type": "failure_spike",
        "observed_value": 8,
        "baseline_value": 11,
        "ratio": 0.7273,
        "minimum_observed": 3,
        "minimum_ratio": 3,
        "threshold_crossed": false,
        "status": "ready"
      }
    ],
    "caveats": [],
    "provenance": {}
  }
}
```

Requirements:

- include exactly one result for each deployed detector: `failure_spike`,
  `contract_deployments_spike`, and `transaction_activity_spike`;
- supply observed, baseline, ratio, detector thresholds, exact hour boundaries, status,
  source ledger, and provenance;
- distinguish a valid zero from missing evidence;
- keep partial rule evaluations typed and scoped rather than downgrading the entire packet;
- make the packet internally reconcilable with any qualifying row in `insights`.

This lets Prism substantiate an authoritative no-change result and, after a later consumer
slice, show a small comparison such as `Failures 0.7× · Deployments 0.5× · Activity 1.1×`
without inventing or recomputing anomaly facts.

### P1: Retain a bounded recent-change index

Add a bounded `recent_insights` collection to `/home/summary`, or a list endpoint that the
summary projection can consume, containing the newest valid qualifying insight per type
from a documented lookback such as 24 hours.

Each row must retain the existing stable `insight_id`, type, evidence version, exact
observed window, subject, severity, observed/baseline/ratio, status, caveats, and evidence
provenance. Do not copy an old row into the current `insights` array: current evaluations
and historical detections must remain distinct.

This allows Prism to say `Last flagged 3 hours ago` and link the retained detail packet
when the current completed hour is healthy but quiet. It must never make an old event look
current.

### P1: Last-good semantics for stale projections

When refresh fails after a valid insight projection was previously served, retain the last
usable rows for a bounded, documented interval and return them with:

- component status `stale`;
- their original observed window;
- the retained source ledger and update time;
- a warning code identifying last-good delivery.

If no valid packet has ever been retained, return `stale` with zero rows and the exact lag
diagnostic. Prism will continue to render a compact delayed state in that case.

## Prism behavior after the API lands

1. Keep qualifying current insights as the primary `What changed` content.
2. When the current evaluation is complete and no threshold crossed, show the compact
   negative result plus supplied detector comparisons.
3. Optionally add one subordinate `Last flagged` link from `recent_insights`.
4. Preserve the current compact `Hourly comparison delayed` state when evaluation evidence
   is stale, partial without usable rules, or unavailable.
5. Never infer change from current leaders, TTL, utilization, or spectrogram data.

## Acceptance gates

- Testnet and mainnet return a reconciled evaluation for all three detector families for
  ten consecutive completed hours.
- A qualifying evaluation has a matching current insight row and stable detail ID.
- A non-qualifying complete hour is authoritative `empty`, not `unavailable`.
- Stale acceptance proves both retained-row and never-retained cases.
- Direct API and Gateway payloads match aside from request-specific timestamps.
- Prism browser acceptance covers current populated, current no-change, partial, stale with
  retained rows, stale without retained rows, and unavailable states.
