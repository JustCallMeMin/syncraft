# Syncraft Protocol Spec

## Purpose

This document defines the v1 synchronization protocol for Syncraft. It specifies the
message envelope, operation payloads, replay rules, reconnect flow, validation rules, and
failure handling required to preserve deterministic convergence.

This document is intentionally narrow. It covers only the protocol and state transfer
required for plain-text collaboration in v1.

Terminology in this document follows `./domain-glossary.md`. In particular, `operation`
is the canonical term for one CRDT change unit, while `event` is reserved for lifecycle and
observability contexts.

## Scope

This protocol spec covers:

- operation envelope
- insert and delete message payloads
- snapshot metadata
- client session and subscription flow
- reconnect and delta synchronization
- duplicate and out-of-order handling
- validation and error messages
- protocol versioning constraints for v1

This protocol spec does not cover:

- authentication design
- rich-text or block document semantics
- presence features beyond minimal session connection semantics
- post-v1 compaction or tombstone garbage collection

## Protocol Goals

- preserve convergence across replicas
- make replay deterministic
- allow the server to validate and persist operations without becoming a conflict resolver
- support reconnect through snapshot plus delta
- tolerate duplicate and out-of-order delivery

## Transport Model

Syncraft v1 uses WebSocket transport for live session communication.

The transport is used for:

- client hello and document subscription
- operation submission
- operation broadcast
- reconnect bootstrap coordination
- error reporting

The protocol must not depend on server-side total ordering for merge correctness.

## Protocol Version

All protocol messages must include a version field:

- `protocol_version`: string

For v1, the required value is:

- `syncraft.v1`

Messages with an unsupported protocol version must be rejected.

## Common Envelope

All protocol messages share this envelope:

```json
{
  "protocol_version": "syncraft.v1",
  "message_type": "string",
  "document_id": "string",
  "session_id": "string",
  "message_id": "string"
}
```

### Envelope Fields

- `protocol_version`: protocol compatibility marker
- `message_type`: identifies the payload type
- `document_id`: target document identifier
- `session_id`: client session identifier for observability and request correlation
- `message_id`: unique per-message transport identifier for tracing and diagnostics

### Rules

- `message_id` is for transport observability and must not be used for CRDT merge semantics.
- `document_id` must be present on all document-scoped messages.
- `session_id` must be stable for the lifetime of one connected client session.

## Operation Identity Model

Each CRDT operation must include stable semantic identifiers:

- `operation_id`
- `actor_id`
- `actor_counter`

### Rules

- `operation_id` must be globally unique within the system.
- `actor_id` identifies the logical editor instance or actor that issued the operation.
- `actor_counter` must increase monotonically per actor.
- Merge correctness must rely on operation content and CRDT rules, not on server receive time.
- Creation timestamps may be recorded for observability, but they must not affect merge order.

## Element Identity Model

Each inserted text element must have a stable `element_id`.

### Rules

- `element_id` must be globally unique.
- Deletes always target `element_id`, never an absolute index.
- Snapshot state must preserve enough identity information to replay and continue applying
  later operations correctly.

## Message Types

The v1 protocol uses these message types:

- `client_hello`
- `subscribe_document`
- `subscribe_ack`
- `submit_operation`
- `broadcast_operation`
- `request_catchup`
- `catchup_snapshot`
- `catchup_operations`
- `catchup_complete`
- `error`

## Client Hello

Sent by the client when establishing a session.

```json
{
  "protocol_version": "syncraft.v1",
  "message_type": "client_hello",
  "session_id": "sess_123",
  "message_id": "msg_123",
  "actor_id": "actor_a"
}
```

### Rules

- the client must declare `actor_id`
- the server may reject malformed or missing actor identifiers
- the hello does not subscribe the client to a document on its own

## Subscribe Document

Sent by the client to join a document stream.

```json
{
  "protocol_version": "syncraft.v1",
  "message_type": "subscribe_document",
  "document_id": "doc_123",
  "session_id": "sess_123",
  "message_id": "msg_124",
  "known_snapshot_id": "snap_10",
  "known_last_operation_id": "op_900"
}
```

### Rules

- `known_snapshot_id` and `known_last_operation_id` are optional on a first-time subscribe
- on reconnect they should reflect the client’s latest trusted local state
- the server uses them only to decide what state transfer is needed

## Subscribe Ack

Returned by the server after a successful subscription decision.

```json
{
  "protocol_version": "syncraft.v1",
  "message_type": "subscribe_ack",
  "document_id": "doc_123",
  "session_id": "sess_123",
  "message_id": "msg_125",
  "subscription_mode": "live_only"
}
```

### Allowed `subscription_mode` values

- `live_only`
- `snapshot_then_delta`

### Rules

- `live_only` means the client is already up to date enough to continue from live traffic
- `snapshot_then_delta` means catch-up transfer must happen before the subscription is fully current

## Submit Operation

Sent by the client to submit a local CRDT operation.

```json
{
  "protocol_version": "syncraft.v1",
  "message_type": "submit_operation",
  "document_id": "doc_123",
  "session_id": "sess_123",
  "message_id": "msg_126",
  "operation": {
    "operation_id": "op_901",
    "actor_id": "actor_a",
    "actor_counter": 44,
    "type": "insert",
    "payload": {}
  }
}
```

### Rules

- the client may optimistically apply the operation locally before server acknowledgment
- the server must validate schema, document scope, and required identifiers before persisting
- the server must not rewrite operation semantics in a way that changes CRDT meaning

## Broadcast Operation

Sent by the server to subscribed clients after accepting an operation.

```json
{
  "protocol_version": "syncraft.v1",
  "message_type": "broadcast_operation",
  "document_id": "doc_123",
  "session_id": "server",
  "message_id": "msg_127",
  "operation": {
    "operation_id": "op_901",
    "actor_id": "actor_a",
    "actor_counter": 44,
    "type": "insert",
    "payload": {}
  }
}
```

### Rules

- clients must be able to receive their own operations back without corruption
- duplicate broadcast of the same operation must not change final state
- clients must ignore or safely deduplicate already applied operations

## Operation Payloads

For v1, the insert payload unit is one rune.

### Rationale

- one rune preserves a simpler CRDT editing model than text chunks
- one rune avoids treating multi-byte Unicode content as transport-level byte fragments
- one rune keeps merge, replay, delete targeting, and visible text projection easier to reason about
  than chunk-level semantics
- the increased operation count is accepted as a v1 trade-off in exchange for correctness clarity

### Insert Operation

Insert operations must contain:

```json
{
  "operation_id": "op_901",
  "actor_id": "actor_a",
  "actor_counter": 44,
  "type": "insert",
  "payload": {
    "element_id": "elem_901",
    "value": "x",
    "left_origin_id": "elem_100",
    "right_origin_id": "elem_200"
  }
}
```

### Insert Rules

- `value` must be plain text content valid under v1 assumptions
- `value` must contain exactly one rune
- `left_origin_id` and `right_origin_id` identify the stable neighboring context known at insert time
- one of the origins may be null at document boundaries
- concurrent ordering among inserts in the same region must be resolved deterministically using
  CRDT rules derived from actor identity and logical counters

### Delete Operation

Delete operations must contain:

```json
{
  "operation_id": "op_950",
  "actor_id": "actor_b",
  "actor_counter": 12,
  "type": "delete",
  "payload": {
    "target_element_id": "elem_901"
  }
}
```

### Delete Rules

- deletes target stable element identifiers only
- deleting an already deleted element must be a no-op
- deleting an unknown element must be deferred, not rejected
- deferred deletes must be staged in a deterministic pending-operation mechanism until the target
  element materializes or the replay context proves the delete can never apply
- duplicate deferred deletes for the same target must remain idempotent

## Snapshot Metadata

Snapshots are derived state used for faster bootstrap.

The server returns snapshot metadata through `catchup_snapshot`:

```json
{
  "protocol_version": "syncraft.v1",
  "message_type": "catchup_snapshot",
  "document_id": "doc_123",
  "session_id": "server",
  "message_id": "msg_200",
  "snapshot": {
    "snapshot_id": "snap_10",
    "last_included_operation_id": "op_900",
    "state": {}
  }
}
```

### Snapshot Rules

- snapshots must contain enough CRDT state to preserve stable identities and ordering semantics
- `last_included_operation_id` marks the replay watermark for delta fetch
- replay from snapshot plus later operations must produce the same state as replay from full log

## Catch-Up Flow

Clients that are behind must use snapshot plus delta.

### Request Catchup

```json
{
  "protocol_version": "syncraft.v1",
  "message_type": "request_catchup",
  "document_id": "doc_123",
  "session_id": "sess_123",
  "message_id": "msg_201",
  "known_snapshot_id": "snap_7",
  "known_last_operation_id": "op_700"
}
```

### Catchup Operations

```json
{
  "protocol_version": "syncraft.v1",
  "message_type": "catchup_operations",
  "document_id": "doc_123",
  "session_id": "server",
  "message_id": "msg_202",
  "catchup_id": "catchup_1",
  "batch_index": 1,
  "has_more": true,
  "last_operation_id_in_batch": "op_920",
  "operations": []
}
```

### Catchup Complete

```json
{
  "protocol_version": "syncraft.v1",
  "message_type": "catchup_complete",
  "document_id": "doc_123",
  "session_id": "server",
  "message_id": "msg_203",
  "last_operation_id": "op_950"
}
```

### Catch-Up Rules

1. client subscribes with known local state
2. server decides whether live-only or snapshot-plus-delta is required
3. if catch-up is required, server sends snapshot metadata and state
4. server sends operations after the snapshot watermark
5. server confirms catch-up completion
6. client continues on the live stream

The client must not assume that reconnect means replay from scratch unless the server explicitly
chooses to provide a fresh snapshot baseline.

### Catch-Up Metadata Rules

- `catchup_id` identifies one reconnect transfer session
- `batch_index` increases monotonically within one `catchup_id`
- `has_more` tells the client whether another delta batch should be expected
- `last_operation_id_in_batch` marks the highest operation included in that batch for tracing and
  retry diagnostics
- v1 does not require total batch count or checksum metadata yet

## Ordering And Replay Rules

- all replicas must reach the same visible text from the same operation set
- operation application must be idempotent
- replay order may vary at delivery time, but final convergence must not
- server persistence order must not be treated as the source of merge truth
- deterministic tie-breaking must be defined by CRDT semantics, not websocket arrival order

## Duplicate Handling

- duplicate `submit_operation` requests with the same `operation_id` must not create duplicate
  semantic operations
- duplicate `broadcast_operation` messages must not alter final state
- duplicate deletes must remain harmless no-ops after the first successful application

## Out-Of-Order Handling

- clients and server-side apply logic must tolerate operations arriving in different valid orders
- if an operation depends on referenced context not yet materialized locally, the implementation
  may stage it temporarily, but final behavior must remain deterministic
- out-of-order delivery must not require manual operator repair

## Validation Rules

The server must validate at least:

- required envelope fields are present
- `protocol_version` is supported
- `message_type` is recognized
- `document_id` is valid for document-scoped messages
- `operation_id`, `actor_id`, and `actor_counter` are present on operations
- operation payload shape matches its type
- text payloads respect plain-text-only v1 constraints

Validation must reject malformed messages before persistence.

## Error Message

Errors use this shape:

```json
{
  "protocol_version": "syncraft.v1",
  "message_type": "error",
  "document_id": "doc_123",
  "session_id": "server",
  "message_id": "msg_500",
  "error_code": "invalid_operation",
  "error_message": "Operation payload is missing target_element_id."
}
```

### Error Principles

- `error_message` must be human-readable
- errors must not leak sensitive internal details
- malformed input must produce actionable diagnostics for development and testing

## Minimal Error Codes

- `unsupported_protocol_version`
- `invalid_message_type`
- `invalid_document_id`
- `invalid_operation`
- `invalid_snapshot_reference`
- `unauthorized`
- `internal_error`

## Logging And Observability

The system must log enough metadata to trace:

- session start and subscription events
- operation receipt, validation, persistence, and broadcast
- snapshot selection and catch-up lifecycle
- errors and rejection reasons

Logs must exclude sensitive values where not required and must not treat logs as merge inputs.

## Versioning Constraints For v1

- v1 supports one active protocol version: `syncraft.v1`
- breaking changes to message shape require a new protocol version
- additive optional fields are allowed only if older clients can safely ignore them

## Decisions Locked For V1

- insert payloads represent exactly one rune
- unknown delete targets are deferred through a deterministic pending mechanism rather than rejected
- catch-up batching metadata includes `catchup_id`, `batch_index`, `has_more`, and
  `last_operation_id_in_batch`

## Related Records

- [`domain-glossary.md`](./domain-glossary.md)
- [`canonical-domain-model.md`](./canonical-domain-model.md)
- [`../01-product/prd-business-spec.md`](../01-product/prd-business-spec.md)
- [`../01-product/user-journeys-demo-script.md`](../01-product/user-journeys-demo-script.md)
- [`../01-product/mvp-scope-release-criteria.md`](../01-product/mvp-scope-release-criteria.md)
- ADR-001
- ADR-002
- ADR-003
