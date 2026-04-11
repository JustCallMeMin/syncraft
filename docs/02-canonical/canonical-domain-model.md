# Syncraft Canonical Domain Model

## Purpose

This document defines the official domain model for Syncraft v1.
It removes ambiguity between product language, protocol fields, persistence records,
and implementation code.

## Canonical Entities

### Document

A document is the shared plain-text collaboration target.

- identified by `document_id`
- has one logical CRDT state at any point in time
- may have many concurrent sessions connected to it
- may have many replicas across clients and server reconstruction flows

### Actor

An actor is the logical operation producer used for CRDT identity.

- identified by `actor_id`
- is not the end-user identity
- represents one logical editing identity for one client instance working on one
  document lineage
- scopes monotonic `actor_counter`

### Client Instance

A client instance is one concrete running editor process or browser tab that
maintains local document state.

- one client instance owns exactly one active `actor_id` per opened document in v1
- same user in two tabs means two client instances and two actors

### Replica

A replica is one materialized copy of a document's CRDT state.

- every connected client instance maintains a local replica
- server bootstrap and recovery may materialize a replica transiently
- replicas must converge from the same semantic operation set

### Session

A session is one live protocol conversation between one client instance and the sync server.

- identified by `session_id`
- connection-scoped, not user-scoped and not operation-scoped
- begins at `client_hello`
- ends when the websocket closes or the server invalidates it
- reconnect creates a new session even if actor lineage is preserved

### Operation

An operation is one semantic CRDT mutation submitted against a document.

- identified by `operation_id`
- produced by exactly one `actor_id`
- `actor_counter` increases monotonically within one actor
- semantic dedupe uses `operation_id`, not transport identifiers

### Snapshot

A snapshot is a derived persisted representation of document state used to accelerate
bootstrap and recovery.

- identified by `snapshot_id`
- not the semantic source of truth
- must preserve enough identity and ordering state for deterministic replay

### Connection

A connection is one transport-level websocket link between client and server.

- in v1, one live connection maps to one live session
- dropped connection does not destroy local actor or replica state
- reconnect replaces the connection and session but may continue the same actor lineage

## Identity Decisions Locked For V1

### `actor_id`

`actor_id` identifies a logical editing actor tied to one client instance's editing
lineage for one document. It is not the user account identifier.

### `session_id`

`session_id` identifies one live connection-scoped protocol session.
Reconnect always creates a new `session_id`.

### `message_id`

`message_id` is a transport and diagnostics identifier only.
It must never be used as the semantic dedupe key.

### `operation_id`

`operation_id` is the canonical semantic dedupe key for operations.

## Relationship Model

- one document has many operations
- one document has many sessions over time and may have many concurrent sessions
- one document has many replicas
- one actor produces many operations
- one session carries many protocol messages
- one snapshot covers one document state at one replay watermark

## Implementation Implications

- code should model `actor_id`, `session_id`, and `operation_id` as distinct identities
- reconnect logic should preserve actor lineage when restoring existing local editor state
- persistence should treat snapshots as accelerators and operations as semantic history
- tests should treat same-user multi-tab behavior as distinct actors when exercised

## Related Docs

- [`canonical-architecture.md`](./canonical-architecture.md)
- [`protocol-spec.md`](./protocol-spec.md)
- [`canonical-invariants.md`](./canonical-invariants.md)
- [`domain-glossary.md`](./domain-glossary.md)
