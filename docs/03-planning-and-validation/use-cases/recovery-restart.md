# Use Case: Recovery Restart

## Preconditions

- operation history and snapshots are persisted with sufficient replay metadata
- the backend restarts or rebuilds in-memory state
- at least one document has meaningful prior edit history

## Trigger

The backend loses in-memory state and must rebuild from persisted artifacts.

## Main Flow

1. Users create non-trivial document history.
2. The backend stops or loses active state.
3. The backend restarts and selects snapshot plus later operations.
4. The backend rebuilds document state from persisted artifacts.
5. Clients reconnect or reopen the document.
6. Reconnected clients receive rebuilt state through normal subscription and catch-up paths.

## Alternate Flows

- if no usable snapshot exists, the backend may replay from the operation log alone
- only affected documents may need rebuild

## Failure Flows

- incomplete or invalid persisted history must surface recovery failure
- a semantically different rebuilt state is a failed recovery even if the service restarts

## Postconditions

- rebuilt state matches the last correct persisted logical state
- reconnecting clients converge to the rebuilt state without corruption

## Expected Visible Outcome

- users reopening or reconnecting see the same logical content that existed before disruption
