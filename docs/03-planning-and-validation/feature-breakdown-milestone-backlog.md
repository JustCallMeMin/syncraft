# Syncraft Feature Breakdown / Milestone Backlog

## Purpose

This document translates the PRD, MVP scope, and protocol specification into an execution
backlog organized by milestone. It is intended to make implementation sequencing explicit
and to reduce ambiguity in task ownership and dependency order.

## Planning Principles

- correctness before UI breadth
- protocol and data model before transport polish
- persistence before recovery claims
- invariant-driven testing before demo confidence
- narrow milestones with explicit exit criteria

## Milestone 1: Core Model Freeze

### Goal

Lock the CRDT data model and operation semantics so the rest of the system can build on a
stable contract.

### Tasks

- `@agent-core-engine`: Define CRDT element and operation model
- `@agent-core-engine`: Implement one-rune insert semantics in the CRDT model
- `@agent-core-engine`: Implement deterministic deferred handling for unknown delete targets
- `@agent-reviewer-security`: Review operation schema and validation boundaries

### Dependencies

- ADR-003 accepted
- Syncraft Protocol Spec approved for v1 use

### Exit Criteria

- operation identifiers and element identifiers are fixed
- insert and delete payload semantics are unambiguous
- tie-breaking behavior is documented
- open protocol questions affecting implementation are resolved

## Milestone 2: Engine Apply Logic

### Goal

Build the in-memory CRDT engine that can apply insert and delete operations deterministically.

### Tasks

- `@agent-core-engine`: Implement deterministic insert apply logic
- `@agent-core-engine`: Implement deterministic delete apply logic
- `@agent-core-engine`: Implement visible text projection
- `@agent-core-engine`: Implement operation deduplication behavior

### Dependencies

- Milestone 1 complete

### Exit Criteria

- single-replica apply behavior is correct
- duplicate operations are harmless
- internal ordering is deterministic

## Milestone 3: Correctness Test Harness

### Goal

Prove convergence and replay invariants before broader integration.

### Tasks

- `@agent-qa`: Build invariant-focused test harness
- `@agent-qa`: Add permutation tests for delivery reordering
- `@agent-qa`: Add duplicate delivery tests
- `@agent-core-engine`: Add replay determinism tests
- `@agent-reviewer-security`: Review error reporting and validation behavior

### Dependencies

- Milestone 2 complete

### Exit Criteria

- convergence holds across tested operation order permutations
- replay from different delivery orders converges to the same visible state
- duplicate delivery does not alter final state

## Milestone 4: Persistence Layer

### Goal

Persist operations and snapshots so Syncraft can recover from restart and support catch-up.

### Tasks

- `@agent-backend`: Design PostgreSQL schema for operation log and snapshots
- `@agent-backend`: Implement append-only operation persistence
- `@agent-backend`: Implement snapshot creation and load flow
- `@agent-backend`: Add rebuild-from-persistence path

### Dependencies

- Milestone 1 complete
- Milestone 2 behavior stable enough for serialization

### Exit Criteria

- accepted operations are persisted durably
- snapshots preserve sufficient CRDT state
- rebuild from persistence matches expected document state

## Milestone 5: Protocol And Transport Integration

### Goal

Implement the v1 websocket protocol and live operation flow defined in the protocol spec.

### Tasks

- `@agent-backend`: Implement `client_hello` and document subscription handling
- `@agent-backend`: Implement `submit_operation` validation and persistence pipeline
- `@agent-backend`: Implement `broadcast_operation` fanout
- `@agent-backend`: Implement protocol error responses
- `@agent-reviewer-security`: Review websocket validation and least-privilege boundaries

### Dependencies

- Milestone 2 complete
- Milestone 4 complete

### Exit Criteria

- clients can subscribe and exchange operations through the defined protocol
- malformed messages are rejected cleanly
- logs trace protocol flow end to end

## Milestone 6: Reconnect And Catch-Up

### Goal

Implement snapshot-plus-delta recovery and prove reconnect correctness.

### Tasks

- `@agent-backend`: Implement catch-up decision logic
- `@agent-backend`: Implement snapshot metadata response
- `@agent-backend`: Implement delta operation replay after snapshot watermark
- `@agent-client`: Implement reconnect bootstrap flow
- `@agent-client`: Resubscribe and continue live stream after catch-up

### Dependencies

- Milestone 4 complete
- Milestone 5 complete

### Exit Criteria

- reconnect restores the same visible state as a continuously connected replica
- catch-up does not require manual intervention
- replay watermark semantics behave as documented

## Milestone 7: Client Collaboration Surface

### Goal

Expose the engine and protocol through a minimal browser-based plain-text editor.

### Tasks

- `@agent-client`: Build minimal editor integration
- `@agent-client`: Apply local optimistic operations
- `@agent-client`: Apply remote operations safely
- `@agent-client`: Surface basic connection and error state

### Dependencies

- Milestone 5 complete
- reconnect flow sufficiently stable from Milestone 6

### Exit Criteria

- two users can edit the same document in a browser
- UI is minimal but reliable
- product demo no longer depends on internal tooling only

## Milestone 8: Demo And Release Hardening

### Goal

Turn the implementation into a repeatable v1 demo that satisfies MVP release criteria.

### Tasks

- `@agent-qa`: Run the agreed demo script end to end
- `@agent-qa`: Validate restart recovery scenario
- `@agent-qa`: Validate reconnect and delivery-irregularity scenarios
- `@agent-product-docs`: Ensure PRD, MVP scope, protocol spec, and demo docs are aligned
- `@agent-reviewer-security`: Final review of validation, logging, and persistence boundaries

### Dependencies

- Milestones 1 through 7 complete

### Exit Criteria

- demo pass conditions are satisfied
- release blockers are cleared
- docs and implementation tell the same story

## Backlog Summary By Area

### Core Engine

- data model freeze
- deterministic insert and delete apply
- visible text projection
- replay and dedup semantics

### Backend

- operation persistence
- snapshot creation and load
- websocket protocol handling
- catch-up and reconnect support

### Client

- minimal editor surface
- optimistic local apply
- remote apply
- reconnect bootstrap and resubscription

### Testing

- invariant tests
- permutation tests
- duplicate delivery tests
- reconnect and restart recovery tests

### Docs

- PRD
- user journeys and demo script
- MVP scope and release criteria
- protocol spec
- milestone backlog

## Milestone 9: Offline Queueing Alpha

### Goal

Introduce a browser-local offline queueing alpha without weakening current convergence,
replay, and server-boundary guarantees.

### Tasks

- `@agent-founding-engine`: Approve offline queueing alpha scope and ADR
- `@agent-core-engine`: Update canonical protocol and invariants for offline queued operation replay
- `@agent-product-docs`: Update PRD and post-v1 scope language for offline queueing alpha
- `@agent-client`: Implement browser-local queued operation store with IndexedDB
- `@agent-client`: Persist actor counter and offline queue metadata across browser restart
- `@agent-client`: Add offline provisional editor state and queued-operation UX
- `@agent-client`: Implement reconnect queue replay over existing `submit_operation` flow
- `@agent-qa`: Add offline queue durability and replay test harness
- `@agent-qa`: Add blocked-queue and local-store-corruption scenarios
- `@agent-reviewer-security`: Final reviewer-security sign-off for offline queue durability and replay boundaries

### Dependencies

- Milestone 8 complete
- [`post-v1-offline-queueing-reentry-rule.md`](./post-v1-offline-queueing-reentry-rule.md) accepted as the planning gate

### Exit Criteria

- offline queueing scope is explicitly limited to a post-v1 alpha and no broader product claim
- canonical protocol and invariants describe replay expectations before implementation proceeds
- browser-local queue durability survives refresh and restart within the accepted boundary
- replay after reconnect converges to the same visible state as a continuously online flow
- blocked-queue behavior is explicit and reviewer-security sign-off is recorded

## Milestone 10: Observability UX

### Goal

Introduce a minimal in-app debug panel for demo and operator-facing diagnosis without
expanding Syncraft into a dashboard-oriented product.

### Tasks

- `@agent-founding-engine`: Approve observability UX scope and debug-panel ADR
- `@agent-product-docs`: Update PRD, MVP scope, and demo script for the observability UX decision
- `@agent-client`: Define browser debug event model and bounded event buffer policy
- `@agent-client`: Implement collapsible in-app debug panel shell
- `@agent-client`: Surface session, queue, and replay diagnostics in the debug panel
- `@agent-client`: Add recent event timeline without payload leakage
- `@agent-qa`: Add regression coverage for observability panel state and blocked-queue diagnostics
- `@agent-reviewer-security`: Review observability UX boundaries and event redaction policy

### Dependencies

- Milestone 9 complete and merged into `main`
- the observability decision is resolved in favor of a minimal in-app debug panel

### Exit Criteria

- logs remain the canonical audit trail
- the browser demo exposes enough observability for operator-facing diagnosis without opening terminal logs
- the panel remains collapsed by default and does not widen product scope
- blocked queue, reconnect, and replay transitions are legible without exposing raw text payloads

## Milestone 11: Docs-Core Web UX

### Goal

Upgrade the browser shell into a more practical Docs-like plain-text collaboration surface
with title metadata, save-state chrome, collaborator presence, and remote caret or selection
visualization.

### Tasks

- `@agent-founding-engine`: Approve docs-core web UX scope and ADR
- `@agent-core-engine`: Update canonical protocol and invariants for title metadata and
  presence semantics
- `@agent-product-docs`: Update PRD, MVP scope, and demo guidance for the docs-core web UX
  tranche
- `@agent-backend`: Implement document title metadata persistence and state bootstrap
- `@agent-backend`: Implement presence protocol, relay, and stale-session expiry
- `@agent-client`: Redesign the browser shell into a docs-core layout with title chrome and
  save-state UX
- `@agent-client`: Implement collaborator strip, remote caret, and remote selection rendering
- `@agent-qa`: Add unit, integration, Playwright, accessibility, and state-matrix coverage
- `@agent-reviewer-security`: Review presence boundary, title metadata persistence, and
  event-redaction policy

### Dependencies

- Milestone 10 complete and merged into `main`
- the docs-core web UX decision is accepted in
  [`docs-core-web-ux-adr.md`](./docs-core-web-ux-adr.md)
- canonical protocol and product docs are updated before backend or client implementation
  expands

### Exit Criteria

- document title is visible, editable, persisted, and propagated across connected clients
- save-state UX distinguishes connecting, live, saving, saved, offline provisional,
  replaying, and blocked states
- collaborator presence strip and remote caret or selection rendering are available in the
  browser shell
- presence traffic remains ephemeral and does not alter CRDT convergence or replay semantics
- browser regressions cover multi-tab editing, title propagation, reconnect, offline queue
  replay, presence expiry, and blocked queue diagnostics

## Immediate Next Tasks

The offline queueing alpha and observability UX tranches are now merged into `main`.
The next approved run is the docs-core web UX tranche documented in
[`docs-core-web-ux-run-plan.md`](./docs-core-web-ux-run-plan.md).
