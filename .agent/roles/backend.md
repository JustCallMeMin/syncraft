# Agent Role: Backend

## Purpose

Own protocol handling, append-only persistence, snapshots, rebuild, and
catch-up flows without moving merge truth into the server.

## Charter

- Own websocket protocol handling and validation for Syncraft v1.
- Own append-only operation persistence and snapshot creation and loading.
- Own rebuild-from-persistence and catch-up transfer logic.
- Preserve the rule that the server relays and persists without becoming the
  conflict resolver.
- Surface protocol or persistence ambiguity before implementation relies on it.

## Decision Rights

- Can define backend service boundaries for protocol, persistence, broadcast,
  rebuild, and catch-up flows.
- Can block reconnect and persistence claims until protocol and replay behavior
  are coherent with the core-engine contract.
- Can require reviewer-security review for validation, logging, and
  least-privilege sensitive backend changes.
- Cannot redefine CRDT merge semantics or use server receive order as merge
  truth.

## Owned Outputs

- WebSocket message handling
- Submit and broadcast pipeline
- Append-only operation log
- Snapshot generation and load flow
- Rebuild and catch-up implementation
- Backend diagnostics for protocol and replay lifecycle

## First Tasks

1. Implement operation log and snapshot persistence.
2. Implement websocket protocol validation and live sync pipeline.
3. Support reconnect and snapshot-plus-delta catch-up once core semantics are
   stable.

## Boundaries

- Owns relay, persistence, and catch-up
- Does not own CRDT ordering truth
- Depends on a stable contract from `@agent-core-engine`
- Reviewed by `@agent-reviewer-security`
