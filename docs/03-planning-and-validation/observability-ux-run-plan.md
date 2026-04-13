# Observability UX Run Plan

## Purpose

This document defines the next post-v1 planning tranche after the offline queueing alpha.
Its goal is to resolve the remaining observability UX decision and turn it into an
implementation-ready task graph without expanding Syncraft into a dashboard product.

## Run Position

- This run starts after the offline queueing alpha has been merged into `main`.
- The run is planning-first before implementation starts.
- The chosen surface is a minimal in-app debug panel inside the existing browser demo shell.
- The panel is for demo and operator-facing diagnosis, not for general end-user workflow.

## Accepted Decision

- logs remain required as the primary technical audit source
- a minimal in-app debug panel is also required for demo and operator-facing diagnosis
- broader observability dashboards remain out of scope
- no new protocol messages or metrics backend are part of this tranche by default

## Planned Product Shape

Add a collapsible debug panel to the current browser demo shell.

The panel should surface only diagnostic state that already exists or can be derived locally:

- current document id
- current actor instance id
- connection state
- queue state
- pending queue count
- replay-in-flight state
- last error text
- last snapshot id and last operation id when available
- a short recent event timeline for connect, reconnect, replay, and blocked-queue transitions

The panel must not expose queued text payloads or raw document contents as diagnostic events.

## Implementation Defaults

- Use a bounded browser-local event buffer.
- Keep the panel collapsed by default.
- Keep the browser shell itself as the only observability surface for this tranche.
- Treat the panel as a debug aid, not a product-scope expansion.

## Task Graph

1. `@agent-founding-engine`: Approve observability UX scope and debug-panel ADR
2. `@agent-product-docs`: Update PRD, MVP scope, and demo script for the observability UX decision
3. `@agent-client`: Define browser debug event model and bounded event buffer policy
4. `@agent-client`: Implement collapsible in-app debug panel shell
5. `@agent-client`: Surface session, queue, and replay diagnostics in the debug panel
6. `@agent-client`: Add recent event timeline without payload leakage
7. `@agent-qa`: Add regression coverage for observability panel state and blocked-queue diagnostics
8. `@agent-reviewer-security`: Review observability UX boundaries and event redaction policy

## Acceptance Criteria

- logs remain the canonical audit trail
- the browser demo exposes enough observability for operator-facing diagnosis without opening terminal logs
- no raw text payloads are logged into the browser event timeline
- the panel stays consistent across reconnect, refresh, replay, and blocked-queue states
- no backend protocol expansion is required for the first observability UX tranche

## Related Docs

- [`feature-breakdown-milestone-backlog.md`](./feature-breakdown-milestone-backlog.md)
- [`post-v1-offline-queueing-run-plan.md`](./post-v1-offline-queueing-run-plan.md)
- [`../01-product/prd-business-spec.md`](../01-product/prd-business-spec.md)
- [`../01-product/mvp-scope-release-criteria.md`](../01-product/mvp-scope-release-criteria.md)
- [`../01-product/user-journeys-demo-script.md`](../01-product/user-journeys-demo-script.md)
