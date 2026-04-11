# Use Case: Delivery Irregularities

## Preconditions

- a shared document and known operation set exist
- the network or harness can introduce duplicate, delayed, or reordered delivery
- semantic identifiers such as `operation_id` and `element_id` are stable

## Trigger

Previously accepted operations are delivered in an irregular order or more than once.

## Main Flow

1. Clients produce a compact set of concurrent operations.
2. The network or harness introduces duplicate, delayed, or reordered delivery.
3. Replicas apply or stage operations using CRDT semantics.
4. Duplicate operations are ignored or reapplied idempotently.
5. Dependent operations are deferred deterministically until they can resolve.
6. Replicas settle to the same final visible state.

## Alternate Flows

- a delete may arrive before its target insert and remain deferred
- a client may receive its own already-applied operation again

## Failure Flows

- duplicates creating visible changes means semantic dedupe failed
- deferred operations resolving differently across replicas means deterministic apply failed

## Postconditions

- duplicate delivery is harmless
- out-of-order delivery does not require manual repair

## Expected Visible Outcome

- after delivery settles, every replica shows the same final plain-text content
