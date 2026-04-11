# Syncraft Project Assessment and Roadmap

## Purpose

This document evaluates the proposed Syncraft v1 architecture, identifies the main delivery risks, and defines an implementation roadmap that is realistic for a first production-grade milestone.

## Scope

This assessment covers:

- v1 architecture choices
- engineering risks and mitigations
- testing and validation strategy
- a six-to-eight week implementation roadmap
- recommended team responsibilities

## v1 Summary

Syncraft v1 is a CRDT-based real-time plain text collaboration platform with the following scope:

- plain text editing only
- multiple users editing the same document concurrently
- CRDT-based synchronization
- WebSocket-based real-time transport
- PostgreSQL-backed operation log and snapshots

## Assessment

### Overall Evaluation

The project is technically strong and suitable for a serious capstone, thesis, or systems-oriented product prototype. It solves a real distributed systems problem rather than only a UI synchronization problem.

The proposal is viable, but only if the team treats the CRDT engine as the core product and keeps the rest of v1 intentionally narrow. The highest risk is not WebSocket transport, PostgreSQL persistence, or the editor UI. The highest risk is correctness of the text CRDT under concurrent and unreliable delivery conditions.

### Strengths

- The problem statement is real and technically meaningful.
- The v1 scope is constrained enough to be achievable if the team avoids feature creep.
- The proposal already identifies key invariants such as convergence, idempotency, and deterministic replay.
- The architecture supports a strong demo story: concurrent insert, reconnect, duplicate delivery tolerance, and crash recovery.

### Main Risks

- The text CRDT is underspecified. "Each character has a globally unique ID" is not enough to guarantee correct ordering semantics.
- Offline and reconnect behavior are harder than they appear because they require deduplication, causal tracking, and replay rules.
- Tombstone accumulation can degrade memory use and rebuild performance even in plain text.
- It is easy to build a demo that appears to work while still violating convergence under reordering or duplicate delivery.

## Recommended Architecture Decisions

### CRDT Choice

For v1, choose one concrete sequence CRDT and implement it faithfully instead of designing a custom hybrid.

Recommended direction:

- use an RGA-style sequence CRDT or another simple linked-position sequence CRDT
- represent inserts relative to stable neighbors
- use deterministic tie-breaking based on actor ID and logical counter

Avoid for v1:

- rich text semantics
- block nesting
- custom garbage collection
- distributed undo and redo

### Server Role

The server should remain simple:

- authenticate and authorize requests if enabled
- validate operation schema and document identity
- persist operations durably
- broadcast operations to subscribers
- serve snapshot plus delta on reconnect

The server should not become the conflict resolver. If correctness depends on server ordering, the CRDT design is incomplete.

### Persistence Model

Use two persistence layers:

1. Operation log
   - append-only
   - source of truth for replay and recovery
2. Snapshot
   - periodic compact state for faster bootstrap and recovery

Minimal tables for v1:

- `documents`
- `document_operations`
- `document_snapshots`
- optional `client_sessions`

### Sync Protocol

Define the protocol early and keep it stable. Each operation should include at least:

- `document_id`
- `operation_id`
- `actor_id`
- `actor_counter`
- `type`
- operation payload
- creation timestamp for observability only, not merge semantics

Reconnect flow should be:

1. client requests latest snapshot metadata
2. server returns snapshot state and last applied operation watermark
3. client requests operations after that watermark
4. client replays remaining operations
5. client resubscribes to live stream

## Non-Negotiable Invariants

The implementation should be considered incorrect if any of the following fail:

- each `element_id` is globally unique
- insert is idempotent
- delete is idempotent
- applying the same operation set in different arrival orders converges to the same visible text
- replay from snapshot plus log yields the same state as replay from full log
- duplicate delivery does not alter final state
- out-of-order delivery does not alter final state

## Risk Register

| ID | Risk | Impact | Likelihood | Mitigation | Owner |
|---|---|---|---|---|---|
| R1 | Wrong sequence CRDT design causes divergence under concurrent insert | High | High | Select one known CRDT design early and validate with invariant tests before UI work | Core engine owner |
| R2 | Reconnect protocol replays operations incorrectly | High | Medium | Define watermark semantics, idempotent operation IDs, and replay tests before full integration | Sync protocol owner |
| R3 | Duplicate or out-of-order delivery breaks state | High | High | Build deterministic apply logic and automated message permutation tests | Core engine owner |
| R4 | Offline edits cannot merge safely after reconnect | High | Medium | Support offline mode only after local queueing and replay rules are proven in tests | Client sync owner |
| R5 | Snapshot format drifts from operation semantics | Medium | Medium | Treat snapshot as derived state and verify rebuild equivalence in tests | Persistence owner |
| R6 | Tombstone growth degrades performance | Medium | Medium | Explicitly accept as v1 limitation and measure cost on representative document sizes | Performance owner |
| R7 | Team spends too much time on editor UX instead of correctness | Medium | High | Keep editor minimal and prioritize engine plus protocol milestones | Project lead |
| R8 | Missing property-based or permutation tests hides correctness bugs | High | High | Make invariant testing a required milestone before multi-user demo claims | QA owner |
| R9 | Security and input validation are deferred | Medium | Medium | Validate all inbound operations and review WebSocket and persistence boundaries | Security reviewer |
| R10 | No clear observability makes failure diagnosis slow | Medium | Medium | Add structured logs and operation trace identifiers from day one | Backend owner |

## Delivery Strategy

### What Must Be True for v1 Success

v1 is successful if it can reliably demonstrate:

- two or more clients editing the same plain text document in real time
- deterministic convergence under concurrent insert and delete
- correct recovery after duplicate and out-of-order operation delivery
- reconnect from snapshot plus delta without manual repair
- server restart recovery from persisted state

### What Is Explicitly Out of Scope

These items should not enter v1 unless the core engine is already stable:

- rich text
- comments or presence indicators beyond basic cursor metadata
- permissions beyond simple document access
- distributed undo and redo
- CRDT garbage collection
- multi-device offline sync with conflict visualization

## Testing Strategy

### Unit Tests

Required unit test areas:

- operation validation
- insert semantics
- delete semantics
- tie-breaking behavior
- idempotent apply behavior
- replay determinism

### Integration Tests

Required integration scenarios:

- two clients concurrent insert at same logical position
- duplicate operation delivery
- out-of-order operation delivery
- reconnect after missing operations
- snapshot creation and reload
- server restart and rebuild from persistence

### Property-Style Correctness Tests

At least one automated test suite should generate permutations of operation arrival order and verify that all replicas converge to the same visible text and internal stable state assumptions.

### Non-Functional Tests

Minimum non-functional checks for v1:

- document bootstrap time from snapshot plus delta
- operation throughput under modest concurrent load
- memory growth with increasing tombstones
- WebSocket reconnect latency

## Suggested Team Responsibilities

If the project has multiple contributors, assign clear ownership:

- `@agent-core-engine`: CRDT data model, apply logic, invariants, engine tests
- `@agent-backend`: WebSocket gateway, persistence, snapshots, recovery
- `@agent-client`: local state integration, optimistic apply, reconnect flow
- `@agent-qa`: scenario matrix, deterministic replay tests, regression suite
- `@agent-reviewer-security`: input validation, persistence boundaries, logging review

Security-related changes should be reviewed by `@agent-reviewer-security` before being considered complete.

## Six-to-Eight Week Roadmap

### Week 1: Architecture Freeze

Deliverables:

- choose the sequence CRDT design
- define operation schema
- define snapshot schema
- define reconnect protocol
- document invariants and failure cases

Exit criteria:

- architecture review approved
- no unresolved ambiguity in operation semantics

### Week 2: Core Engine Prototype

Deliverables:

- implement in-memory document model
- implement insert and delete
- implement deterministic ordering rules
- implement visible text projection

Exit criteria:

- unit tests cover core apply logic
- duplicate and repeated apply are idempotent

### Week 3: Correctness Test Harness

Deliverables:

- message permutation test suite
- replay determinism tests
- concurrent insert and delete scenario tests

Exit criteria:

- same operation set converges under varied arrival orders
- failing cases produce readable diagnostics

### Week 4: Persistence and Recovery

Deliverables:

- PostgreSQL schema
- append-only operation storage
- snapshot creation and loading
- rebuild from snapshot plus operation log

Exit criteria:

- recovery tests pass
- restart produces same state as uninterrupted runtime

### Week 5: Real-Time Transport

Deliverables:

- WebSocket session handling
- operation validation
- broadcast pipeline
- subscription and document session lifecycle

Exit criteria:

- two clients can edit concurrently against the same backend
- logs trace each operation end to end

### Week 6: Client Integration

Deliverables:

- minimal editor UI
- optimistic local apply
- remote operation apply
- reconnect bootstrap flow

Exit criteria:

- user-visible real-time collaboration works in browser
- reconnect rehydrates correctly from server state

### Week 7: Hardening

Deliverables:

- duplicate and out-of-order simulation in integration environment
- basic performance measurement
- error reporting and structured logs
- docs cleanup

Exit criteria:

- core regression suite passes
- known v1 limitations documented clearly

### Week 8: Demo and Stabilization

Deliverables:

- final demo script
- architecture and API docs
- deployment checklist
- rollback and recovery notes

Exit criteria:

- stable demo with repeatable scenarios
- no unresolved high-severity correctness defects

## Minimal Initial Backlog

Start with these implementation tasks only:

1. Define CRDT element and operation model.
2. Implement deterministic insert and delete apply logic.
3. Build invariant-focused tests before network transport.
4. Add persistence for operation log and snapshots.
5. Add WebSocket synchronization.
6. Add minimal browser editor integration.

## Performance Assessment

Expected v1 performance characteristics:

- acceptable for small to medium plain text documents
- acceptable for low to moderate concurrent session counts
- not optimized for long-lived documents with heavy edit history

This is acceptable for v1, but the limitation should be explicit in all architecture and demo material.

## Rollback and Recovery Approach

For major releases and demos, the rollback strategy should be:

- preserve append-only operation log as immutable source of truth
- treat snapshots as replaceable derived artifacts
- rebuild state from operation log if snapshot corruption or schema drift is detected
- gate protocol changes behind explicit version fields

## Recommendation

Proceed with the project. The idea is strong and defensible, but success depends on disciplined execution:

- keep v1 narrow
- treat CRDT correctness as the primary deliverable
- lock architecture decisions early
- invest heavily in invariant-based testing before UI expansion

If this discipline is maintained, Syncraft can become a compelling distributed systems project rather than only a collaborative text editor demo.
