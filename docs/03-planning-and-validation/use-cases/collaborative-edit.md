# Use Case: Collaborative Edit

## Preconditions

- at least two client instances subscribe to the same `document_id`
- each client has a valid `actor_id` and active `session_id`
- the document starts from a known visible state

## Trigger

A second or later collaborator joins the same document and begins submitting edits.

## Main Flow

1. User A opens a shared document and reaches a stable subscribed state.
2. User B opens the same document and reaches the same logical baseline.
3. Each client turns plain-text edits into CRDT operations and applies them optimistically locally.
4. Both users submit text operations.
5. The server validates, persists, and broadcasts accepted operations.
6. Each client applies remote operations safely, including its own operations replayed back from the server.
6. Replicas converge to the same visible result.

## Alternate Flows

- a later joiner may need catch-up before continuing on live traffic
- a client may receive its own operation back and handle it idempotently
- a browser-facing editor layer may surface `connecting`, `catching_up`, or `live` state while collaboration continues

## Failure Flows

- malformed operations are rejected instead of silently corrupting replica state
- disconnect leaves this baseline path and becomes reconnect handling

## Postconditions

- all accepted operations are reflected in the final converged state
- replicas hold equivalent visible text after delivery settles

## Expected Visible Outcome

- both users see the same final plain-text content
- one client does not silently override another
