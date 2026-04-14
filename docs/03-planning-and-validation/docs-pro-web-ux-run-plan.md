# Docs Pro Web UX Run Plan

## Purpose

This run plan defines the first post-MVP web tranche after `docs-core web UX`. The goal
is to make Syncraft feel materially closer to a practical collaborative writing product
while preserving plain-text CRDT boundaries.

## Product Direction

This tranche is `Docs Pro`, not workspace-home and not review-first.

The browser surface should evolve toward:

- stronger title and document chrome
- clearer save and sync language
- outline-based navigation for longer documents
- comments-lite for coordination around plain-text ranges
- lightweight checkpoints as confidence and recovery landmarks
- richer collaborator presence and activity signals

## Approved Additions

- browser-local outline derived from heading syntax (`#`, `##`, `###`)
- comments-lite anchored to element-id backed range positions
- checkpoint metadata stored separately from operation and snapshot truth
- richer collaborator activity states such as `typing`, `viewing`, and `idle`
- deeper accessibility and viewport-state regression coverage

## Preserved Boundaries

- plain text only
- no rich-text formatting model
- no suggestion mode
- no permission or admin layer
- no workspace-management-first pivot
- no CRDT schema expansion for comments or checkpoints

## Delivery Shape

The run should be organized into these phases:

1. Accept `Docs Pro` ADR and align PRD and demo guidance.
2. Update canonical protocol and invariant wording for comments-lite and checkpoints.
3. Implement outline model and navigation.
4. Implement comments metadata flow and browser comments rail.
5. Implement checkpoint metadata flow and user-visible checkpoint list.
6. Refine collaborator activity UX and keep diagnostics payload-safe.
7. Close with browser-first QA and reviewer-security sign-off.

## Expected Evidence

The run is not complete until these are all true:

- outline updates live after local and remote edits
- comments converge across tabs and survive refresh or restart
- checkpoints bootstrap correctly and do not alter replay semantics
- debug panel and logs do not leak raw document text via comments or checkpoints
- browser regressions cover accessibility and narrow-width layouts
