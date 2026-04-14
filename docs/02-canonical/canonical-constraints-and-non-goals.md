# Syncraft Canonical Constraints And Non-Goals

## Purpose

This document states the official constraints and non-goals that limit Syncraft v1 scope.

## Canonical Constraints

- plain text only
- CRDT correctness first
- server as relay plus persistence, not central conflict resolver
- deterministic replay and convergence are non-negotiable
- snapshots are derived acceleration artifacts, not independent truth sources

## Canonical Non-Goals For V1

- rich text formatting
- block-based document semantics
- comments, review mode, and workflow features
- distributed undo and redo
- enterprise-grade permissions and policy management
- long-history tombstone garbage collection
- mobile-native clients
- offline-first multi-device sync as a committed v1 feature

## Post-v1 Tranche Allowances

Post-v1 tranches may add product-surface behavior only when the change is recorded
explicitly in product, planning, and canonical docs first.

The currently accepted post-v1 allowances are:

- browser-only offline queueing alpha within its documented durability boundary
- minimal in-app observability UX debug panel
- docs-core web UX with:
  - title metadata separate from CRDT text
  - save and sync chrome
  - ephemeral collaborator presence, caret, and basic selection
- docs-pro web UX planning with:
  - browser-local outline derived from plain-text headings
  - comments-lite and checkpoints as metadata side channels
  - richer collaborator activity UX that remains ephemeral

These allowances do not weaken the canonical constraints above.

## Governance Rule

Any proposal that moves an item out of the non-goals list or weakens a canonical
constraint must be reflected in product, release, and technical docs before it is
treated as accepted scope.

## Related Docs

- [`../01-product/prd-business-spec.md`](../01-product/prd-business-spec.md)
- [`../01-product/mvp-scope-release-criteria.md`](../01-product/mvp-scope-release-criteria.md)
- [`canonical-domain-model.md`](./canonical-domain-model.md)
