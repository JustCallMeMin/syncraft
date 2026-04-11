# Syncraft Domain Glossary

## Purpose

This document standardizes the core domain terms used across Syncraft v1 so agents and
contributors do not use overlapping words for different concepts.

Use these definitions in code, documentation, tasks, ADRs, tests, and review comments.

## Canonical Terms

### Operation

`operation` is the canonical term for one CRDT change unit in Syncraft.

Use `operation` when referring to:

- one semantic insert or delete change in the CRDT model
- persisted entries in the append-only operation log
- payloads exchanged through `submit_operation`, `broadcast_operation`, and catch-up flows
- replay, deduplication, ordering, and convergence rules

Examples:

- `insert operation`
- `delete operation`
- `operation_id`
- `operation log`
- `operation payload`

### Event

`event` is reserved for transport, session, lifecycle, and observability signals.

Use `event` when referring to:

- session start or end
- subscription lifecycle
- logging and tracing records
- operational telemetry

Do not use `event` as a synonym for a CRDT change unit.

Examples:

- `session event`
- `subscription event`
- `log event`

### Delta

`delta` is reserved for catch-up transfer after a snapshot baseline.

Use `delta` when referring to:

- the set or batch of operations sent after a known snapshot watermark
- reconnect flow phrasing such as `snapshot plus delta`
- replay after `last_included_operation_id`

`delta` is not the primary name for a single CRDT change unit in Syncraft v1.

Examples:

- `snapshot plus delta`
- `delta batch`
- `delta replay`

### Mutation

`mutation` is not a canonical Syncraft v1 domain term.

Avoid `mutation` in protocol, CRDT, persistence, and recovery documentation unless another
tooling context requires it and the local meaning is explicitly defined.

If a document currently means CRDT change unit, use `operation` instead.

## Usage Rules

- Prefer `operation` for CRDT semantics, persistence, replay, and protocol payloads.
- Prefer `event` for observability and lifecycle signaling only.
- Prefer `delta` only for snapshot-following catch-up transfer.
- Avoid mixing `operation`, `event`, `mutation`, and `delta` in the same paragraph unless the
  distinction matters and is stated explicitly.
- When a new document introduces a term that could overlap with these definitions, define it
  inline before use.

## Scope Notes

- This glossary governs Syncraft v1 terminology only.
- It does not replace protocol or architecture specifications.
- If a future ADR changes the semantic model, this glossary must be updated in the same change.
