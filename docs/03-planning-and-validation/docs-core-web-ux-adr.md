# ADR: Adopt Docs-Core Web UX Tranche With Title Metadata And Ephemeral Presence

## Status

Accepted

## Context

Syncraft `main` now includes:

- the v1 plain-text collaboration baseline
- the browser-only offline queueing alpha
- the minimal observability UX debug panel

The current browser surface proves correctness and recovery, but it is still a narrow demo
shell. The next post-v1 tranche needs to improve practical daily-use collaboration without
weakening CRDT convergence, replay, persistence, or server-boundary guarantees.

## Decision

Syncraft adopts a post-v1 `docs-core web UX` tranche with these boundaries:

- keep the document model plain text only
- keep CRDT correctness and replay guarantees unchanged
- keep the server as relay plus persistence, not merge authority
- add document title UX as persistent document metadata
- add live collaborator presence, cursor, and basic selection as ephemeral protocol traffic
- add a deeper browser QA matrix, including Playwright-driven end-to-end coverage

This tranche explicitly allows:

- document title chrome and inline title editing
- save and sync state UX
- live collaborator presence strip
- remote caret and basic remote selection visualization
- operator-facing diagnostics for these states through the existing debug panel

This tranche explicitly does not add:

- rich text formatting
- comments, suggestion mode, or review workflows
- multi-document workspace management
- enterprise permissions
- dashboard-style observability surfaces
- server-side merge authority for presence or title changes

## Architecture Consequences

### Title Metadata

- document title is stored as lightweight document metadata
- title is not represented as CRDT text content
- title is persisted separately from the operation log used for CRDT replay
- title updates may be validated and fanned out by the server

### Presence

- presence state is ephemeral and must not be stored in operation-log persistence
- presence includes collaborator identity, cursor anchor/focus, selection collapse state,
  and last-seen timing
- presence must not affect CRDT ordering or convergence semantics
- stale presence may expire by timeout

### Browser UX

- the browser app shifts from a minimal demo form to a docs-core writing surface
- connection and save state must be legible without opening the debug panel
- the debug panel remains collapsed by default and continues to redact document payloads

## Testing Consequences

This tranche requires broader browser-focused QA:

- unit coverage for title state, save-state derivation, presence normalization, and cursor
  mapping
- integration coverage for title persistence and presence relay
- Playwright end-to-end coverage for multi-tab editing, title propagation, presence,
  reconnect, offline queue replay, blocked queue, and refresh or restart continuity
- accessibility and visual-state checks for top bar, title field, presence strip, status
  chips, and debug panel

## Related Docs

- [`docs-core-web-ux-run-plan.md`](./docs-core-web-ux-run-plan.md)
- [`../01-product/prd-business-spec.md`](../01-product/prd-business-spec.md)
- [`../01-product/mvp-scope-release-criteria.md`](../01-product/mvp-scope-release-criteria.md)
- [`../01-product/user-journeys-demo-script.md`](../01-product/user-journeys-demo-script.md)
- [`../02-canonical/protocol-spec.md`](../02-canonical/protocol-spec.md)
- [`../02-canonical/canonical-invariants.md`](../02-canonical/canonical-invariants.md)
- [`../02-canonical/canonical-constraints-and-non-goals.md`](../02-canonical/canonical-constraints-and-non-goals.md)
