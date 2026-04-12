# ADR: Post-v1 Offline Queueing Alpha Scope

## Status

Accepted

## Decision Date

2026-04-12

## Context

Syncraft v1 was intentionally shipped without offline queueing. The project now needs an explicit
post-v1 direction for the next run, but that direction must not weaken the current guarantees:

- deterministic convergence
- snapshot-plus-delta reconnect behavior
- server as relay plus persistence rather than merge authority
- narrow product claims

The project also already has a planning gate in
[`post-v1-offline-queueing-reentry-rule.md`](./post-v1-offline-queueing-reentry-rule.md) that
blocks feature implementation until the shape of the offline queueing model is explicit.

## Decision

Syncraft will pursue offline queueing as a **post-v1 alpha** with the following boundaries:

1. The first implementation target is **browser-only**.
2. The accepted durability boundary is **one local browser profile or device only**.
3. Offline local edits are represented as **canonical CRDT operations**, not higher-level text
   intent objects.
4. Offline local edits are treated as **provisional local state** until replay succeeds after
   reconnect.
5. Reconnect must continue to use the existing flow:
   - perform normal catch-up first
   - replay queued local operations only after catch-up completes
   - submit replay through the existing `submit_operation` path
6. The server does **not** gain a special offline merge role for this alpha.
7. The alpha does **not** imply broad offline-first support, multi-device portability, or cross-tab
   coordination guarantees.

## Consequences

### Positive

- The next run stays aligned with the current CRDT and reconnect model.
- Replay semantics remain close to the already tested online path.
- Product language stays narrow and defensible.
- Reviewer-security and QA can reason about a smaller durability and failure surface.

### Negative

- The first alpha will not satisfy broader offline-first expectations.
- Browser storage corruption and blocked replay still need explicit UX and validation work.
- Same-profile durability is useful but intentionally limited.

### Follow-up Requirements

- update canonical protocol and invariants before implementation starts
- update PRD and planning language so the alpha boundary is explicit
- implement queue durability and replay in the browser client only
- add QA coverage for refresh, restart, replay equivalence, and corruption handling
- obtain reviewer-security sign-off before the alpha is considered ready

## Rejected Alternatives

### Broad Offline-First Scope Immediately

Rejected because it would over-expand the next run before protocol, invariants, and durability
boundaries are updated.

### Queue Text Intents Instead Of Canonical Operations

Rejected because it would introduce a second reconciliation model and make replay semantics less
traceable to the current CRDT engine.

### Add Special Server-Side Offline Merge Semantics

Rejected because it would erode the existing architectural boundary that keeps the server out of
merge-truth decisions.

## Related Docs

- [`post-v1-offline-queueing-run-plan.md`](./post-v1-offline-queueing-run-plan.md)
- [`post-v1-offline-queueing-reentry-rule.md`](./post-v1-offline-queueing-reentry-rule.md)
- [`use-cases/offline-queueing.md`](./use-cases/offline-queueing.md)
- [`../02-canonical/protocol-spec.md`](../02-canonical/protocol-spec.md)
- [`../02-canonical/canonical-invariants.md`](../02-canonical/canonical-invariants.md)
