# Use Case: Offline Queueing

## Status

Deferred for v1. The current post-v1 alpha run is now implementing the browser-only queueing
path, but it still remains a constrained alpha rather than a broad offline-first commitment.

## Preconditions

- a client can accept local edits while offline
- queued operations remain durable enough to survive the offline interval
- reconnect logic can merge queued work with remote changes safely
- canonical protocol and invariants define the replay boundary before implementation starts

## Trigger

A disconnected client continues editing locally and later attempts to synchronize queued edits.

## Main Flow

1. The client loses connectivity.
2. The user continues editing locally.
3. The browser client builds canonical queued operations from local visible-element state and
   persists them in IndexedDB.
4. Connectivity returns and the client reconnects.
5. The client completes normal reconnect catch-up first.
6. The client replays queued canonical operations through the normal submission path.
7. The system accepts, deduplicates, or blocks replay without weakening canonical invariants.

## Alternate Flows

- offline edits may be shown as provisional local state
- future versions may require explicit conflict UX
- the first post-v1 alpha is browser-only and limited to one local browser profile or device
- the current browser shell uses visible-element state from the demo server to build local
  canonical operations before reconnect replay

## Failure Flows

- overlap with remote change may be unsafe if merge semantics are not yet canonical
- local process loss may destroy queued work if durability is insufficient
- corrupted local queue records may block replay and require explicit recovery guidance

## Postconditions

- no canonical v1 behavior is promised here yet
- the currently accepted post-v1 alpha boundary is browser-only with same-profile durability
- broader offline-first behavior still requires explicit promotion into product and technical specs
- any post-v1 implementation must first satisfy
  `../post-v1-offline-queueing-reentry-rule.md`

## Expected Visible Outcome

- no required v1 visible outcome exists because the scenario is deferred from v1
- in the post-v1 alpha, offline local state must be visibly provisional until replay succeeds
- when browser-local queue state is healthy, the editor remains writable while disconnected so
  provisional local edits can continue accumulating before replay
- when a fresh browser session is still waiting for its first ready state, user input is buffered
  until the active session becomes ready instead of being misclassified as an offline or error path

## Related Docs

- [`../post-v1-offline-queueing-reentry-rule.md`](../post-v1-offline-queueing-reentry-rule.md)
- [`../post-v1-offline-queueing-run-plan.md`](../post-v1-offline-queueing-run-plan.md)
- [`../offline-queueing-alpha-adr.md`](../offline-queueing-alpha-adr.md)
- [`../../02-canonical/protocol-spec.md`](../../02-canonical/protocol-spec.md)
- [`../../02-canonical/canonical-invariants.md`](../../02-canonical/canonical-invariants.md)
