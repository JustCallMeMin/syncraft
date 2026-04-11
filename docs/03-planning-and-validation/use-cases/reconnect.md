# Use Case: Reconnect

## Preconditions

- one client disconnects from an active shared document
- the disconnected client retains a local replica and last trusted known state
- other clients may continue editing during the interruption
- the backend can provide a coherent catch-up decision and state transfer

## Trigger

A previously subscribed client loses its live session and later rejoins the same document.

## Main Flow

1. User B loses the active session.
2. Other collaborators continue producing accepted operations.
3. User B establishes a new connection and sends a new session handshake.
4. User B includes known local snapshot or last-operation state.
5. The server chooses `live_only` or `snapshot_then_delta`.
6. If needed, the server sends a snapshot baseline and later operations.
7. User B resumes the live stream and converges to the same visible state.

## Alternate Flows

- if the client is current enough, the server may choose `live_only`
- short gaps may require only a small catch-up set

## Failure Flows

- invalid state references are rejected and force a fresh baseline
- reconnect must not claim success if snapshot plus delta cannot rebuild coherent state

## Postconditions

- the reconnecting client reaches the same logical state as continuously connected peers
- the old session is replaced by a new session

## Expected Visible Outcome

- the reconnecting user sees the correct current document content without manual repair
