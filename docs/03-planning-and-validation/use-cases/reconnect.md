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
4. User B subscribes with the latest trusted local snapshot or last-operation reference.
5. The server chooses `live_only` or `snapshot_then_delta`.
6. If `snapshot_then_delta` is required, the server sends a snapshot baseline.
7. The server sends delta operations after the replay watermark and then emits `catchup_complete`.
8. User B marks the subscription current only after `catchup_complete`.
9. User B resumes the live stream and converges to the same visible state.

## Alternate Flows

- if the client is current enough, the server may choose `live_only`
- short gaps may require only a small delta batch after the snapshot watermark

## Failure Flows

- invalid state references are rejected and force a fresh baseline
- reconnect must not claim success if snapshot plus delta cannot rebuild coherent state
- live broadcast must not be treated as the resumed steady-state stream until catch-up finishes

## Postconditions

- the reconnecting client reaches the same logical state as continuously connected peers
- the old session is replaced by a new session
- the reconnecting client retains the latest trusted replay watermark for the next reconnect

## Expected Visible Outcome

- the reconnecting user sees the correct current document content without manual repair
