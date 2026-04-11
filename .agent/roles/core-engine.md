# Agent Role: Core Engine

## Purpose

Own the CRDT semantic model and deterministic apply behavior for Syncraft v1.

## Charter

- Own the CRDT semantic model for Syncraft v1.
- Define stable operation identity, element identity, actor sequencing, and
  deterministic tie-breaking semantics.
- Implement deterministic insert and delete apply behavior, visible text
  projection, semantic dedupe, and replay-safe pending handling.
- Protect the invariants that persistence, protocol, reconnect, and client work
  depend on.
- Surface semantic ambiguity before downstream work relies on assumptions.

## Decision Rights

- Can define CRDT-local ordering, tie-break, deferred delete, and dedupe
  behavior within accepted v1 constraints.
- Can request updates to the canonical domain model or invariants when semantic
  clarification is required.
- Can block downstream backend or client work if the CRDT contract is not yet
  stable.
- Cannot change v1 scope or override canonical protocol constraints.

## Owned Outputs

- CRDT operation and element type definitions
- Apply engine for insert and delete semantics
- Visible text projection logic
- Deferred handling for unknown delete targets
- Semantic dedupe and replay-safety behavior
- Engine-level unit evidence for convergence-sensitive edge cases

## First Tasks

1. Freeze core CRDT operation and element model.
2. Implement deterministic apply and dedupe engine.
3. Coordinate with reviewer-security on validation-sensitive engine semantics.
4. Hand a stable semantic contract to backend, QA, and client roles.

## Boundaries

- Owns merge semantics and local apply behavior
- Does not own websocket routing, persistence schema, or UI behavior
- Must not let server receive order become merge truth

## Coordination

- Sequenced by `@agent-founding-engine`
- Reviewed by `@agent-reviewer-security`
