# Syncraft Canonical Invariants

## Purpose

This document defines the official invariants that Syncraft v1 must preserve, plus the accepted
replay constraints for the post-v1 offline queueing alpha.

## Official Invariants

- every `operation_id` is globally unique
- every inserted `element_id` is globally unique
- insert application is idempotent
- delete application is idempotent
- the same semantic operation set converges to the same visible text across replicas
- replay from snapshot plus later operations yields the same logical state as replay from full history
- duplicate delivery does not alter final visible state
- out-of-order delivery does not alter final visible state
- restart recovery reconstructs the same logical document state from persisted data
- queued offline operations, once replayed after reconnect, must converge to the same visible text
  as the equivalent continuously online operation set
- duplicate replay of already accepted queued operations must not alter final visible state
- presence traffic, title updates, and browser diagnostics must not alter CRDT convergence for
  document text

## Interpretation Rules

- server receive order is not merge truth
- deterministic tie-breaking must come from CRDT semantics
- snapshots are derived state and cannot change semantic meaning
- provisional offline local state must not be described or treated as accepted shared state before
  replay succeeds
- offline queue replay must happen after normal catch-up, not before
- document title metadata must remain separate from CRDT text semantics
- presence, cursor, and selection traffic must remain ephemeral and must not be replayed as
  document truth

## Validation Rule

If a change weakens, bypasses, or redefines any invariant above, treat it as an
architectural change and record it explicitly.

## Related Docs

- [`canonical-domain-model.md`](./canonical-domain-model.md)
- [`protocol-spec.md`](./protocol-spec.md)
- [`../03-planning-and-validation/use-case-catalog.md`](../03-planning-and-validation/use-case-catalog.md)
- [`../03-planning-and-validation/offline-queueing-alpha-adr.md`](../03-planning-and-validation/offline-queueing-alpha-adr.md)
- [`../01-product/mvp-scope-release-criteria.md`](../01-product/mvp-scope-release-criteria.md)
