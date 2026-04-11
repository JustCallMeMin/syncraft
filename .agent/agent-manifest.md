# Syncraft Agent Manifest

## Purpose

This file is the workspace-local manifest for agent roles that have been
approved through Syncraft Tasks in Notion.

## Source Policy

- Source of approval: Notion Tasks database
- Local execution surface: `.agent/roles/`
- Canonical behavior references:
  - `docs/02-canonical/canonical-architecture.md`
  - `docs/02-canonical/protocol-spec.md`
  - `docs/02-canonical/canonical-invariants.md`
  - `docs/02-canonical/canonical-constraints-and-non-goals.md`
  - `docs/03-planning-and-validation/feature-breakdown-milestone-backlog.md`

## Approved Roles

### `@agent-ceo`

- Status: active in governance
- Local file: [`roles/ceo.md`](./roles/ceo.md)
- Purpose: v1 execution governance, role coverage, and hiring sequence

### `@agent-founding-engine`

- Status: active in governance
- Local file: [`roles/founding-engineer.md`](./roles/founding-engineer.md)
- Purpose: technical decomposition and execution sequencing

### `@agent-reviewer-security`

- Status: active in governance
- Local file: [`roles/reviewer-security.md`](./roles/reviewer-security.md)
- Purpose: independent review for validation, logging, least privilege, and
  governance-sensitive work

### `@agent-core-engine`

- Status: active in governance
- Local file: [`roles/core-engine.md`](./roles/core-engine.md)
- Purpose: CRDT semantics, deterministic apply behavior, replay correctness,
  and semantic dedupe

### `@agent-backend`

- Status: active in governance
- Local file: [`roles/backend.md`](./roles/backend.md)
- Purpose: protocol handling, persistence, snapshots, rebuild, and catch-up

### `@agent-qa`

- Status: active in governance
- Local file: [`roles/qa.md`](./roles/qa.md)
- Purpose: invariant harnesses, permutation coverage, reconnect, and recovery
  validation

### `@agent-client`

- Status: active in governance
- Local file: [`roles/client.md`](./roles/client.md)
- Purpose: minimal plain-text editor, optimistic apply, remote apply, and
  reconnect UX

### `@agent-product-docs`

- Status: active in governance
- Local file: [`roles/product-docs.md`](./roles/product-docs.md)
- Purpose: Notion-to-local doc alignment, demo alignment, and documentation
  governance

## Approved Creation Sequence

1. `@agent-ceo`
2. `@agent-founding-engine`
3. `@agent-reviewer-security`
4. `@agent-core-engine`

## Current Materialization State

The currently approved planning roles have been materialized in the workspace.
