# ADR: Post-v1 Observability UX Debug Panel Scope

## Status

Accepted

## Decision Date

2026-04-13

## Context

Syncraft now has a shipped browser demo shell on `main` plus a merged post-v1 offline queueing
alpha. The remaining observability decision was whether demo and operator-facing diagnosis should
continue to rely on logs only, or whether the browser shell should expose a narrow diagnostic
surface.

That decision needs to preserve the current project boundaries:

- logs remain the canonical technical audit trail
- Syncraft does not become a dashboard or analytics project
- no new protocol messages are added by default
- no raw text payloads are exposed as browser diagnostics

## Decision

Syncraft will pursue observability UX as a **post-v1 planning and implementation tranche** with
the following boundaries:

1. Logs remain the primary technical audit source.
2. The browser demo shell gains a **minimal in-app debug panel** for demo and operator-facing
   diagnosis.
3. The panel is **collapsed by default** and must not widen the product claim beyond a narrow
   diagnostic aid.
4. The panel may surface only state that already exists or can be derived locally, including:
   - document id
   - actor instance id
   - connection state
   - queue state and pending queue count
   - replay-in-flight state
   - last error text
   - last snapshot id and last operation id when available
   - a bounded recent-event timeline for connect, reconnect, replay, and blocked-queue transitions
5. The event timeline must **not** expose queued text payloads or raw document contents.
6. Broader dashboards, metrics backends, production analytics, and user-identity modeling remain
   out of scope for this tranche.
7. No new backend protocol or persistence contracts are introduced by default unless later work
   proves an existing browser-visible state field is insufficient.

## Consequences

### Positive

- Demo operators can diagnose reconnect, replay, and blocked-queue flows without opening terminal
  logs.
- The browser shell becomes more legible for evaluations without weakening the narrow product
  scope.
- The implementation can stay client-local and reuse current state fields.

### Negative

- The debug panel adds surface area to the browser shell that still needs explicit regression
  coverage.
- Logs remain necessary for full audit detail, so the panel does not replace terminal-side
  diagnosis.
- Event redaction and retention policy must be kept explicit to prevent accidental payload
  leakage.

### Follow-up Requirements

- update PRD, MVP scope, and demo script language so the logs-versus-panel boundary is explicit
- define a browser debug event model and bounded event buffer before UI implementation starts
- add QA coverage for panel rendering, blocked-queue diagnostics, reconnect, refresh, and replay
- obtain reviewer-security sign-off for event redaction and observability boundaries

## Rejected Alternatives

### Logs Only

Rejected because the browser demo now carries enough recovery and blocked-state complexity that
operator-facing diagnosis should not depend entirely on terminal access.

### Full Dashboard Or Metrics Surface

Rejected because it would widen Syncraft into a dashboard-style project and create scope not
required for the current demo and operator-facing needs.

### New Protocol Messages For Debug Data

Rejected because the current tranche should first exploit existing browser-visible state and
client-local derivations before expanding the wire protocol.

## Related Docs

- [`observability-ux-run-plan.md`](./observability-ux-run-plan.md)
- [`feature-breakdown-milestone-backlog.md`](./feature-breakdown-milestone-backlog.md)
- [`../01-product/prd-business-spec.md`](../01-product/prd-business-spec.md)
- [`../01-product/mvp-scope-release-criteria.md`](../01-product/mvp-scope-release-criteria.md)
- [`../01-product/user-journeys-demo-script.md`](../01-product/user-journeys-demo-script.md)
