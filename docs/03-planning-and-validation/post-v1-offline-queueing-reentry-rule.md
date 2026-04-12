# Post-v1 Offline Queueing Re-entry Rule

## Purpose

This document defines the minimum planning gate that must be satisfied before Syncraft opens any
implementation task for offline queueing.

Offline queueing remains deferred. This page exists to stop accidental scope creep and to make the
re-entry decision explicit, reviewable, and traceable.

## Current Scope Position

- Offline queueing is not part of the approved v1 scope.
- No client may claim committed local offline edits as accepted shared state in the current product.
- No implementation task for offline queueing should be opened until the re-entry gate below is
  satisfied.

## Re-entry Gate

Offline queueing may move from deferred planning into implementation only if all of the following
are true:

1. A new ADR or equivalent accepted decision records the offline queueing model.
2. Product scope is updated so offline queueing is no longer treated as a deferred-only scenario.
3. Canonical protocol rules define how queued local operations are reintroduced after reconnect.
4. Canonical invariants define what must still hold when offline work overlaps with remote change.
5. Client durability expectations are explicit, including what survives browser refresh, process
   loss, or machine restart.
6. Reviewer-security signs off the local durability and replay boundary before implementation
   begins.
7. QA owns a scenario and test plan before the first implementation task is marked in progress.

If any item above is missing, offline queueing stays deferred.

## Required Design Constraints

Any future offline queueing design must satisfy these constraints before it can be treated as
accepted:

- Preserve canonical convergence.
  Queued local work must not weaken deterministic replay or convergence guarantees.
- Preserve server role boundaries.
  The server must remain relay plus persistence, not become a central merge truth because of
  offline replay.
- Make provisional state explicit.
  Offline local edits must be described as provisional local intent until reconciliation rules are
  complete.
- Define overlap semantics.
  The project must decide how queued local operations behave when their original visible context is
  no longer current after reconnect.
- Define local durability scope.
  The project must state whether queued operations survive tab refresh only, browser restart, or
  broader device-local restart scenarios.
- Define failure visibility.
  Users must be able to tell when queued work is pending, accepted, rejected, or needs manual
  intervention.
- Keep v1 semantics stable.
  Post-v1 offline work must not retroactively change the meaning of the current reconnect and
  snapshot-plus-delta flow for online replicas.

## Required Decision Inputs

Before opening implementation work, the project must explicitly answer:

- Is offline state represented as committed local document state or as staged intent?
- What storage mechanism is trusted for local queue durability?
- What replay order applies when queued local operations are resubmitted?
- When can queued work be auto-applied, and when must the user resolve ambiguity?
- What observability is required for queued, replayed, rejected, and dropped local operations?

## Required Follow-up Work After Re-entry

Once the gate is satisfied, implementation should start with planning and validation work before
feature code:

1. add or update ADRs and canonical protocol/invariant pages
2. define client-side local durability boundary
3. define replay and reconciliation semantics for overlapping remote change
4. define QA scenarios for provisional local state, reconnect replay, and queue-loss failure modes
5. open implementation tasks only after those items are accepted

## Non-Goals For This Re-entry Rule

This document does not approve offline queueing.
It does not define the final replay algorithm.
It does not promise browser-local persistence or multi-device offline sync.

## Related Docs

- [`use-cases/offline-queueing.md`](./use-cases/offline-queueing.md)
- [`../01-product/prd-business-spec.md`](../01-product/prd-business-spec.md)
- [`../01-product/mvp-scope-release-criteria.md`](../01-product/mvp-scope-release-criteria.md)
- [`../02-canonical/protocol-spec.md`](../02-canonical/protocol-spec.md)
- [`../02-canonical/canonical-constraints-and-non-goals.md`](../02-canonical/canonical-constraints-and-non-goals.md)
