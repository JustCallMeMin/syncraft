# Observability Debug Event Model

## Purpose

This document defines the client-side debug event model and bounded event-buffer policy for the
post-v1 observability UX tranche. It exists so the in-app debug panel can be implemented without
inventing its own event taxonomy, retention behavior, or redaction rules.

## Scope

This model applies only to the browser demo shell debug panel.

It does not define:

- backend protocol messages
- metrics pipelines
- analytics exports
- long-term storage
- production telemetry

## Event Record Shape

Each browser debug event must be represented as one client-local record with this shape:

- `event_id`: browser-local unique identifier
- `event_type`: one allowed event class from the list below
- `level`: `info`, `warning`, or `error`
- `occurred_at`: ISO-8601 timestamp
- `document_id`: current document id when available
- `actor_id`: current actor instance id when available
- `connection_state`: current browser connection state when relevant
- `queue_state`: current derived queue surface state when relevant
- `pending_queue_count`: integer when relevant
- `snapshot_id`: last known snapshot id when relevant
- `last_operation_id`: last known operation id when relevant
- `message`: short human-readable summary
- `details`: optional structured metadata object containing ids, counts, or mode flags only

## Allowed Event Classes

The first tranche may emit only these event classes:

- `connect_requested`
- `socket_opened`
- `state_ready`
- `disconnect_detected`
- `reconnect_scheduled`
- `reconnect_requested`
- `queue_store_init_failed`
- `queue_load_failed`
- `queue_metadata_load_failed`
- `offline_queue_appended`
- `queue_blocked`
- `replay_started`
- `replay_progress`
- `replay_completed`
- `replay_failed`

The implementation must not invent additional event classes without updating this policy first.

## Redaction Rules

The debug event buffer must not store:

- raw editor text
- inserted rune values
- deleted text content
- serialized CRDT payload bodies
- arbitrary websocket payload dumps

Allowed details include:

- operation counts
- queue counts
- document id
- actor instance id
- snapshot id
- operation id
- event mode flags such as `preserve_session_state`

If an event would be more useful with raw text, the implementation must still redact it and use a
count or identifier instead.

## Retention Policy

The browser event timeline must use a bounded in-memory ring buffer.

The first implementation must use:

- maximum size: `50` events
- append-only timeline semantics
- oldest events dropped first when the buffer is full

The first tranche must not persist the event buffer across page reload or browser restart.

## Emission Rules

- Emit at most one event per meaningful browser-shell transition.
- Do not emit duplicate timeline entries for the same transition from stale socket generations.
- `replay_progress` should be coarse and keyed to replayed queue count rather than one noisy
  entry per low-value internal step.
- `queue_blocked` and failure events must include a short remediation-oriented message when
  possible.

## UI Mapping Defaults

The debug panel should map event levels as follows:

- `info`: ordinary lifecycle and replay progress
- `warning`: disconnected, reconnect scheduled, or provisional queue conditions
- `error`: blocked queue, queue store failure, metadata failure, replay failure

The `Recent Events` section should show newest first.

## Related Docs

- [`observability-ux-run-plan.md`](./observability-ux-run-plan.md)
- [`observability-ux-debug-panel-adr.md`](./observability-ux-debug-panel-adr.md)
- [`../02-canonical/protocol-spec.md`](../02-canonical/protocol-spec.md)
