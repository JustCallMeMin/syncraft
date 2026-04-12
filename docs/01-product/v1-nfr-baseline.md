# Syncraft V1 NFR Baseline

## Purpose

This document defines the minimum non-functional baseline for Syncraft v1 so
implementation and demo-readiness are judged against explicit thresholds instead
of implicit expectations.

## Baseline Principle

Syncraft v1 is correctness-first, but correctness claims still need a minimum
operational baseline so validation runs are meaningful.

## Canonical V1 Baseline

### Concurrent Users

- target: `2` to `5` concurrent active editors on the same document
- non-goal: broad scalability claims beyond small-team collaboration

### Document Size

- target: plain-text documents in the low tens of kilobytes
- enough history to exercise reconnect and restart replay without turning
  large-document optimization into a v1 promise

### Bootstrap Time

- target: initial open or catch-up should feel near-immediate in local development
  and demo conditions
- practical threshold: bootstrap from snapshot plus delta should normally complete
  within `5s` at the v1 target scale of one shared document in the low tens of
  kilobytes

### Reconnect Latency

- target: reconnect and catch-up should restore a correct visible document state
  quickly enough for live demos
- practical threshold: reconnect catch-up should normally complete within `5s`
  at the same v1 target scale

### Durability Expectation

- accepted operations must survive normal server restart when persistence is healthy
- restart recovery must rebuild the same logical document state from persisted
  snapshot plus operation history

### Correctness Priority

- correctness regressions are blockers even if latency is acceptable
- small latency variation is acceptable if convergence and replay remain correct

## Demo-Readiness Rule

A v1 demo is operationally acceptable only if:

- at least two concurrent editors can collaborate without divergence
- reconnect completes without manual repair
- restart recovery reproduces the same logical state from persistence
- no known correctness blocker remains open for the demo path

## Current Local Evidence

The current repo evidence for this baseline is:

- `internal/qa/performance/performance_test.go`:
  `TestV1TargetScaleBootstrapAndReconnect`
- target scale under test:
  - one plain-text document of `12 KiB`
  - snapshot baseline at `8 KiB`
  - reconnect delta tail for the remaining document history
- pass rule:
  - rebuild from persistence completes within `5s`
  - reconnect catch-up completes within `5s`

This is a practical v1 local-demo baseline, not a production SLO.

## Explicit Non-Claims

This baseline does not claim:

- internet-scale concurrency
- large-document optimization
- production-grade SLOs
- capacity guarantees beyond the v1 target range

## Related Docs

- [`prd-business-spec.md`](./prd-business-spec.md)
- [`mvp-scope-release-criteria.md`](./mvp-scope-release-criteria.md)
- [`../03-planning-and-validation/use-case-catalog.md`](../03-planning-and-validation/use-case-catalog.md)
- [`../02-canonical/protocol-spec.md`](../02-canonical/protocol-spec.md)
