# Syncraft Canonical Architecture

## Purpose

This document defines the canonical architectural structure for Syncraft v1 in one place.

## Architecture Summary

Syncraft v1 uses a client-server architecture where CRDT correctness lives in
operation semantics and deterministic apply behavior, not in server-side ordering.

## Core Components

### Client

- maintains local document state
- creates CRDT operations with stable identity
- applies local optimistic operations and remote operations
- performs reconnect bootstrap and catch-up replay

### Sync Server

- accepts websocket sessions
- validates message shape and document scope
- persists accepted operations
- broadcasts operations to subscribed clients
- serves snapshot plus delta for reconnect and recovery

### Persistence Layer

- append-only operation log as replay source of truth
- snapshots as derived acceleration artifacts for bootstrap and recovery

## Responsibility Boundaries

- clients and shared CRDT semantics determine merge correctness
- the server is relay plus persistence, not central conflict resolver
- snapshots must preserve enough identity and ordering state for deterministic replay

## Recovery Model

- reconnect and restart recovery use snapshot plus later operations
- replay from persisted state must converge to the same logical document state as
  uninterrupted execution

## Security And Validation Boundary

- all inbound protocol messages are validated before persistence
- sensitive operations are logged without leaking sensitive values

## Related Docs

- [`canonical-domain-model.md`](./canonical-domain-model.md)
- [`protocol-spec.md`](./protocol-spec.md)
- [`canonical-invariants.md`](./canonical-invariants.md)
- [`../01-product/prd-business-spec.md`](../01-product/prd-business-spec.md)
