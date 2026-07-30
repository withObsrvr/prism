# Prism transaction failure explanation: API follow-up

Date: 2026-07-28

## Purpose

Prism must distinguish an exact failure cause from a broad protocol result. `INVOKE_HOST_FUNCTION_TRAPPED` proves that contract execution stopped, but it does not identify the underlying contract error. The exact cause may require diagnostic host-error evidence.

Protocol reference: [Stellar Docs, Debugging Contract Errors](https://developers.stellar.org/docs/learn/fundamentals/contract-development/errors-and-debugging/debugging-errors).

Prism now handles this defensively by presenting `invoke_host_function_trapped` as a general failure category, exposing incomplete diagnostics, and keeping the protocol code in technical details. The API should make this distinction explicit rather than requiring each consumer to infer it from component state.

## Immediate `transaction_outcome_v1` correction

For the existing frozen packet, correct status and caveat propagation without changing the response shape:

1. When `components.diagnostics.status` is `partial`, `stale`, or `unavailable` and diagnostics are needed to resolve the cause, set the packet `status` to the corresponding non-ready state.
2. Emit a top-level caveat whose `affects` includes `failure` and `diagnostics`.
3. Do not return `status: ready` with a partial diagnostic component and no caveat.
4. Keep the enclosing transaction result authoritative even when diagnostic enrichment is incomplete.

The testnet packet for transaction `23ecd6a52e382822c2cecb7c60e00b54522c38f72d12a6dca012dd248dc8d9fc` currently violates item 3: the packet is `ready`, diagnostics are `partial`, and `caveats` is empty.

## Versioned diagnostic evidence contract

Add a new evidence version when structured diagnostic cause evidence is available. Do not overload `failure.status: ready`; that field currently means the broad result was resolved, not that the root cause was identified.

Recommended additive model for `transaction_outcome_v2`:

```json
{
  "failure": {
    "status": "ready",
    "normalized_code": "invoke_host_function_trapped",
    "cause_resolution": "category_only",
    "diagnostic": {
      "status": "partial",
      "host_error_type": "",
      "host_error_code": "",
      "message": "",
      "event_ids": [],
      "source": "transaction_meta_diagnostic_events"
    }
  }
}
```

`cause_resolution` should be one of:

- `exact`: evidence identifies the direct failure cause;
- `category_only`: evidence identifies a broad result such as a trapped invocation, but not its cause;
- `unresolved`: the available result does not identify even a useful category.

The API should provide evidence, not final user-facing prose. Prism remains responsible for deterministic explanations and audience-appropriate wording.

## Diagnostic normalization

When supported by transaction metadata, expose bounded structured values for:

- host error type, such as `Budget`, `Storage`, `WasmVm`, or `Auth`;
- host error code, such as `LimitExceeded` or `InvalidAction`;
- decoded diagnostic message when present;
- diagnostic event identifiers and operation index;
- completeness, source, source ledger, and materialization watermark.

Preserve the difference between confirmed evidence and common causes. For example, `WasmVm/InvalidAction` may commonly represent a contract panic, but the API must not claim a panic unless the evidence establishes one.

## Invariants

1. `cause_resolution: exact` requires a structured diagnostic or an operation/transaction code that is itself an exact cause.
2. `invoke_host_function_trapped` without a diagnostic subtype must be `category_only`.
3. A non-ready diagnostic component that affects cause resolution must produce a top-level caveat.
4. An empty diagnostic result is authoritative only when coverage for the source transaction and ledger is authoritative.
5. Every structured diagnostic must have a source locator or event identifier.
6. Diagnostic messages must be bounded and safe to display as untrusted evidence.
7. Direct and Gateway responses must remain semantically equivalent.

## Acceptance corpus

Add fixtures and direct plus Gateway acceptance for:

- trapped invocation with missing diagnostics;
- `Storage/LimitExceeded` footprint access;
- `WasmVm/InvalidAction`;
- `Auth/InvalidAction` missing authorization;
- other `Auth` errors such as expired or invalid signatures;
- budget/resource exhaustion;
- arithmetic, argument-type, or other runtime errors;
- a broad trapped result whose exact cause remains unavailable.

For each case, assert outcome, ledger application, cause resolution, diagnostic completeness, caveats, locators, and provenance independently.
