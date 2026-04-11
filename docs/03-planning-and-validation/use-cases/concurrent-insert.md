# Use Case: Concurrent Insert

## Preconditions

- a shared document has a known base string
- two actors target the same logical insertion region
- origin references and actor counters are available

## Trigger

Two insert operations are created concurrently against the same logical position.

## Main Flow

1. User A inserts a rune at the target position.
2. User B inserts a different rune at the same position before seeing A's insert.
3. Both operations are accepted and propagated.
4. CRDT tie-breaking resolves final ordering deterministically.
5. All replicas converge to the same final string containing both inserts.

## Alternate Flows

- different replicas may receive A and B in different orders
- one client may optimistically render local insert before reconciliation

## Failure Flows

- invalid origin references are rejected
- irregular delivery must still preserve deterministic final ordering

## Postconditions

- both inserted values remain present
- final ordering is stable across replicas

## Expected Visible Outcome

- every replica shows both inserted values in one identical order
