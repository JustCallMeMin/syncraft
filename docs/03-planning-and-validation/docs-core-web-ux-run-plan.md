# Docs-Core Web UX Run Plan

## Purpose

This document defines the next post-v1 tranche after observability UX. Its goal is to turn
the current browser shell into a more practical docs-core collaborative plain-text surface
while preserving Syncraft's existing correctness, replay, and persistence boundaries.

## Run Position

- this run starts after the observability UX tranche is complete on `main`
- this run includes both planning and implementation work
- the accepted scope is a plain-text, Docs-like collaboration surface rather than a full
  document suite
- protocol and backend changes are allowed because document title metadata and live presence
  require explicit support

## Accepted Boundary

- plain text remains the only document content model
- document title is persistent metadata, not CRDT text content
- collaborator presence, caret, and selection are ephemeral
- server remains relay plus persistence, not merge authority
- comments, suggestion mode, rich text, permissions, and workspace management remain out of
  scope
- logs remain the technical audit source and the in-app debug panel remains an operator aid

The accepted ADR for this tranche is
[`docs-core-web-ux-adr.md`](./docs-core-web-ux-adr.md).

## Product Shape

The browser shell moves to a three-zone layout:

- top app bar with product mark, document title, save-state chip, collaborator strip, and
  debug toggle
- main editor canvas centered in a larger writing surface
- secondary utility rail for reconnect, queue, replay, and concise diagnostics

The tranche introduces:

- document title editing and propagation
- save and sync state derived from live, offline, replaying, and blocked conditions
- collaborator presence strip
- remote caret and basic selection rendering
- deeper operator diagnostics for those states through the existing debug panel

## Task Graph

1. `@agent-founding-engine`: Approve docs-core web UX scope and ADR
2. `@agent-core-engine`: Update canonical protocol and invariants for title metadata and
   presence semantics
3. `@agent-product-docs`: Update PRD, MVP scope, and demo guidance for the docs-core web UX
   tranche
4. `@agent-backend`: Implement document title metadata persistence and state bootstrap
5. `@agent-backend`: Implement presence protocol, relay, and stale-session expiry
6. `@agent-client`: Redesign the browser shell into a docs-core layout with title chrome and
   save-state UX
7. `@agent-client`: Implement collaborator strip, remote caret, and remote selection rendering
8. `@agent-qa`: Add unit, integration, Playwright, accessibility, and state-matrix coverage
9. `@agent-reviewer-security`: Review presence boundary, title metadata persistence, and
   event-redaction policy

## Acceptance Criteria

- title changes converge across connected browser tabs without entering the CRDT text model
- presence traffic does not affect CRDT correctness or replay semantics
- remote caret and basic selection rendering are stable enough for demo and operator use
- save and sync state is legible in the normal UI without opening the debug panel
- the browser shell remains usable through live, offline, replaying, and blocked states
- tests cover multi-tab collaboration, title propagation, reconnect, offline queue replay,
  presence expiry, and blocked queue diagnostics

## Related Docs

- [`docs-core-web-ux-adr.md`](./docs-core-web-ux-adr.md)
- [`feature-breakdown-milestone-backlog.md`](./feature-breakdown-milestone-backlog.md)
- [`observability-ux-run-plan.md`](./observability-ux-run-plan.md)
- [`../01-product/prd-business-spec.md`](../01-product/prd-business-spec.md)
- [`../01-product/mvp-scope-release-criteria.md`](../01-product/mvp-scope-release-criteria.md)
- [`../01-product/user-journeys-demo-script.md`](../01-product/user-journeys-demo-script.md)
- [`../02-canonical/protocol-spec.md`](../02-canonical/protocol-spec.md)
- [`../02-canonical/canonical-invariants.md`](../02-canonical/canonical-invariants.md)
