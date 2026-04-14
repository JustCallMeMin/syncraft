# Planning And Validation

Use this folder for execution sequencing and proof-oriented documents.

- [`feature-breakdown-milestone-backlog.md`](./feature-breakdown-milestone-backlog.md)
  Milestone-level delivery breakdown.
- [`use-case-catalog.md`](./use-case-catalog.md)
  Entry point for implementation-facing use cases.
- [`v1-demo-release-evidence.md`](./v1-demo-release-evidence.md)
  Release-evidence landing page for the shipped browser demo path.
- [`post-v1-offline-queueing-reentry-rule.md`](./post-v1-offline-queueing-reentry-rule.md)
  Planning gate that must be satisfied before offline queueing leaves deferred status.
- [`post-v1-offline-queueing-run-plan.md`](./post-v1-offline-queueing-run-plan.md)
  Approved next-run sequence for the post-v1 offline queueing alpha.
- [`observability-ux-run-plan.md`](./observability-ux-run-plan.md)
  Approved planning tranche for the next post-v1 observability UX run.
- [`observability-ux-debug-panel-adr.md`](./observability-ux-debug-panel-adr.md)
  Accepted scope boundary for the minimal in-app observability debug panel.
- [`observability-debug-event-model.md`](./observability-debug-event-model.md)
  Client-side event taxonomy, redaction rules, and retention cap for the debug panel timeline.
- [`docs-core-web-ux-run-plan.md`](./docs-core-web-ux-run-plan.md)
  Approved post-v1 run plan for the docs-core browser collaboration tranche.
- [`docs-core-web-ux-adr.md`](./docs-core-web-ux-adr.md)
  Accepted scope boundary for document title metadata, save-state chrome, and ephemeral
  presence in the web UX upgrade.
- [`docs-pro-web-ux-run-plan.md`](./docs-pro-web-ux-run-plan.md)
  Approved next-run plan for the first post-MVP docs-pro collaboration tranche.
- [`docs-pro-web-ux-adr.md`](./docs-pro-web-ux-adr.md)
  Accepted scope boundary for outline, comments-lite, checkpoints, and richer activity UX.
- [`offline-queueing-alpha-adr.md`](./offline-queueing-alpha-adr.md)
  Accepted decision for the post-v1 offline queueing alpha scope and durability boundary.
- [`use-cases/`](./use-cases/)
  Individual use case specs used by engineering and QA.

This layer should stay tightly linked to:

- product scope in `../01-product/`
- canonical behavior in `../02-canonical/`
- test and traceability work in Notion
