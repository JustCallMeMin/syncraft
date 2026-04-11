# Syncraft Use Case Catalog

## Purpose

This catalog indexes the core and adjacent use cases used to explain, implement,
and validate Syncraft behavior.

## Scope Rules

- A use case page in this catalog does not automatically mean the use case is committed for v1.
- Approved v1 commitments remain governed by product scope, release criteria, and
  canonical specs.
- Deferred or exploratory scenarios must say so explicitly.

## Current V1-Committed Use Cases

- [`use-cases/collaborative-edit.md`](./use-cases/collaborative-edit.md)
- [`use-cases/concurrent-insert.md`](./use-cases/concurrent-insert.md)
- [`use-cases/reconnect.md`](./use-cases/reconnect.md)
- [`use-cases/recovery-restart.md`](./use-cases/recovery-restart.md)
- [`use-cases/delivery-irregularities.md`](./use-cases/delivery-irregularities.md)

## Deferred Or Out-Of-Scope Use Cases

- [`use-cases/offline-queueing.md`](./use-cases/offline-queueing.md)

## Decision Anchors

- scope discipline: ADR-001
- CRDT model choice: ADR-003
- protocol behavior: [`../02-canonical/protocol-spec.md`](../02-canonical/protocol-spec.md)
- correctness: [`../02-canonical/canonical-invariants.md`](../02-canonical/canonical-invariants.md)
