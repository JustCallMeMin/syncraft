# Post-v1 Offline Queueing Run Plan

## Purpose

This document defines the full next-run plan for Syncraft after the v1 demo baseline.
It converts the offline queueing re-entry rule into an execution sequence that is still
guarded by design approval, canonical updates, QA coverage, and reviewer-security sign-off.

## Run Position

- This run is post-v1 work.
- The run is hybrid: design and canonical updates come first, then implementation may start.
- The run follows a high-safety bar: no feature implementation begins until the design gate is
  accepted.
- The default target is an alpha browser-only queueing flow on one local browser profile or
  device, not broad offline-first support.

## Proposed Model

- Queue canonical CRDT operations locally, not higher-level text intents.
- Treat offline local edits as provisional local state until replay succeeds.
- Use the existing reconnect and `submit_operation` path after catch-up, rather than introducing
  a special server-side offline merge authority.
- Keep local durability browser-local only for the first alpha.

## Delivery Phases

### Phase 1: Design Gate

1. Approve offline queueing alpha scope and ADR.
2. Update canonical protocol and invariants for offline replay.
3. Update product scope language for the post-v1 alpha.

### Phase 2: Client Durability And Replay

1. Implement a browser-local queued-operation store.
2. Persist actor counter and queue metadata across browser restart.
3. Add explicit offline provisional state and queue-status UX.
4. Replay queued operations after normal reconnect catch-up.

### Phase 3: Validation And Sign-off

1. Add durability and replay test harness coverage.
2. Add blocked-queue and corruption scenarios.
3. Obtain final reviewer-security sign-off for replay and local durability boundaries.

## Task Graph

### Design Gate

- `@agent-founding-engine`: Approve offline queueing alpha scope and ADR
- `@agent-core-engine`: Update canonical protocol and invariants for offline queued operation replay
- `@agent-product-docs`: Update PRD and post-v1 scope language for offline queueing alpha

### Client Durability And Replay

- `@agent-client`: Implement browser-local queued operation store with IndexedDB
- `@agent-client`: Persist actor counter and offline queue metadata across browser restart
- `@agent-client`: Add offline provisional editor state and queued-operation UX
- `@agent-client`: Implement reconnect queue replay over existing submit_operation flow

### Validation And Sign-off

- `@agent-qa`: Add offline queue durability and replay test harness
- `@agent-qa`: Add blocked-queue and local-store-corruption scenarios
- `@agent-reviewer-security`: Final reviewer-security sign-off for offline queue durability and replay boundaries

## Acceptance Criteria

- queued offline operations survive tab refresh and browser restart on the same browser profile
- reconnect replays queued operations only after normal catch-up completes
- replayed queued operations converge to the same visible state as a continuously online flow for
  the same operation set
- duplicate replay remains harmless because canonical operation IDs still deduplicate correctly
- blocked or corrupted queue state is explicit and does not silently drop local work
- docs and task memory describe the feature as a post-v1 alpha, not a v1 commitment or broad
  offline-first promise

## Current Status

Planning approved.
Design gate is complete, and the first client implementation tranche is now in place:

- browser-local queued-operation durability exists via IndexedDB
- actor-counter and reconnect metadata survive same-profile restart
- provisional, replaying, and blocked queue states are explicit in the browser shell
- healthy disconnected browser sessions remain editable so offline queueing is usable in practice
- reconnect replay now submits queued canonical operations through the existing browser
  `submit_operation` path after normal catch-up completes

The next work in sequence is QA validation and reviewer-security sign-off.

## Related Docs

- [`post-v1-offline-queueing-reentry-rule.md`](./post-v1-offline-queueing-reentry-rule.md)
- [`use-cases/offline-queueing.md`](./use-cases/offline-queueing.md)
- [`feature-breakdown-milestone-backlog.md`](./feature-breakdown-milestone-backlog.md)
- [`../02-canonical/protocol-spec.md`](../02-canonical/protocol-spec.md)
- [`../02-canonical/canonical-invariants.md`](../02-canonical/canonical-invariants.md)
