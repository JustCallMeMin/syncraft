# ADR: Adopt Docs Pro Plain-Text Web UX Tranche

## Status

Accepted

## Context

The `docs-core web UX` tranche upgraded Syncraft from a minimal browser demo into a
credible plain-text collaboration surface with:

- document title metadata
- save-state chrome
- collaborator presence
- remote caret and selection overlays
- deeper browser diagnostics

That tranche is still intentionally narrow. It improves operator confidence and writing
quality, but it does not yet feel like a daily-use collaborative writing product.

The next tranche must move the product beyond MVP positioning without violating current
Syncraft boundaries:

- plain text only
- CRDT correctness first
- server as relay plus persistence, not merge authority
- title, comments, and checkpoints as metadata side channels rather than CRDT text changes

## Decision

Syncraft will adopt a post-v1 `Docs Pro` web UX tranche with these approved additions:

- richer document chrome and metadata presentation
- browser-local outline derived from plain-text heading syntax
- comments-lite anchored to text ranges via element-id anchored positions
- lightweight checkpoint metadata with user-visible recovery landmarks
- richer collaborator presence and activity affordances

The following remain out of scope for this tranche:

- rich-text formatting
- suggestion mode
- permissions or admin controls
- workspace or folder management as the primary product focus
- publishing or sharing workflows

## Consequences

The implementation must preserve current correctness boundaries:

- comments and checkpoints persist separately from CRDT operations and snapshots
- presence and activity remain ephemeral
- outline remains derived from visible text projection, not a separate document model
- save-state UX remains a browser-facing product state, not a second backend source of truth

The tranche must be tested browser-first with deep coverage for:

- outline updates and navigation
- multi-tab comments and title convergence
- checkpoint bootstrap after refresh or restart
- reconnect, replay, blocked-queue, and presence expiry scenarios
- accessibility and viewport-state behavior
