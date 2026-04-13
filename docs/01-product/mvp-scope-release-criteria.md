# Syncraft MVP Scope / Release Criteria

## Purpose

This document defines the exact v1 scope boundary for Syncraft and the criteria required to
declare the MVP complete. It exists to protect the project from scope creep and to give the
team a shared definition of done.

This is a product execution document. It translates the PRD, ADRs, and roadmap into
delivery gates.

## Scope Principle

Syncraft v1 succeeds by being narrow and correct.

The MVP is complete when it can prove real-time plain-text collaboration with deterministic
recovery and convergence under the conditions already defined as critical in project memory.

The MVP is not complete merely because there is a usable editor UI.

## In-Scope Product Capabilities

The following capabilities are required for Syncraft v1:

- create or open a plain-text shared document
- connect multiple users to the same document session
- apply local edits and propagate remote edits in real time
- converge under concurrent insert and delete operations
- persist operations durably
- rebuild state from snapshot plus operation log
- recover correctly after reconnect
- recover correctly after backend restart
- expose enough observability to diagnose failures in development and demo environments

The current repo delivery path for these capabilities is the minimal browser demo shell
served by `go run ./cmd/demo-server`.

## In-Scope Technical Foundations

The following foundations are required because the product claims depend on them:

- a concrete sequence CRDT choice for text ordering
- stable operation identifiers
- idempotent insert and delete apply behavior
- deterministic replay rules
- snapshot plus delta bootstrap flow
- validation for inbound operations
- automated tests for duplicate and out-of-order delivery scenarios

## Explicitly Out Of Scope

The following items must not block v1 release:

- rich text formatting
- headings, block trees, and structured documents beyond plain text
- comments, suggestions, or review mode
- distributed undo and redo
- advanced presence features beyond minimal session awareness
- multi-document workspace management
- enterprise-grade permissions
- long-history tombstone garbage collection
- mobile-native clients
- offline-first marketing claims beyond what the tested reconnect flow actually supports

## Must-Have Release Outcomes

The MVP may be called complete only if all of the following are true:

- at least two clients can edit the same plain-text document concurrently
- the same operation set converges to the same visible text across replicas
- duplicate delivery does not alter final visible state
- out-of-order delivery does not alter final visible state
- reconnect from snapshot plus delta restores the correct state
- backend restart recovery restores the correct state from persistence
- the team can run a repeatable demo without manual data repair

## Quality Gates

### Functional Gates

- local insert and delete behavior works on a single replica
- remote operations apply cleanly to another replica
- concurrent editing produces one converged visible result
- reconnect path is exercised in an end-to-end scenario
- restart recovery is exercised in an end-to-end scenario

### Correctness Gates

- insert is idempotent
- delete is idempotent
- replay from full log matches replay from snapshot plus log
- element and operation identities remain stable and unique
- tie-breaking is deterministic for concurrent inserts

### Testing Gates

- unit tests cover core apply logic
- integration tests cover concurrent editing
- integration tests cover reconnect
- integration tests cover restart recovery
- permutation or equivalent tests cover delivery reordering and duplication

### Documentation Gates

- PRD is current
- ADRs reflect the accepted technical direction
- roadmap and local docs do not contradict release scope
- demo script matches the actual shipped product behavior

## Nice-To-Have But Not Release Blocking

These items are valuable, but they must not delay v1 if the must-have outcomes are already
met:

- more polished editor visuals
- presence indicators beyond the essentials
- improved onboarding copy
- broader observability dashboards
- performance improvements beyond baseline acceptable operation

## Current Implementation Note

The shipped browser surface is intentionally narrow:

- one shared plain-text editor view
- visible connection and error state
- reconnect and restart recovery through the same demo path

This is sufficient for v1 release evidence if the correctness and recovery gates are met.
It should not be described as a broad end-user editing product yet.

## Release Blockers

The MVP must not be declared complete if any of the following remain true:

- replicas diverge under tested concurrent scenarios
- reconnect can lose, duplicate, or corrupt visible text
- restart recovery depends on manual intervention
- operation persistence is incomplete or unverified
- the system only works under ideal in-order delivery assumptions
- v1 demo success depends on unsupported or hidden operator steps

## Definition Of Done

Syncraft v1 is done when:

- the required product capabilities are present
- correctness gates have been met
- release blockers have been cleared
- the team can execute the agreed demo script end to end
- the result still respects the narrow v1 scope described in the PRD

The current checklist evidence for these claims is consolidated in
[`../03-planning-and-validation/v1-demo-release-evidence.md`](../03-planning-and-validation/v1-demo-release-evidence.md).

## Exit Checklist

- [x] Shared plain-text editing works with at least two clients
- [x] Concurrent edits converge consistently
- [x] Reconnect flow is verified
- [x] Restart recovery is verified
- [x] Persistence behavior is verified
- [x] Duplicate delivery tests pass
- [x] Out-of-order delivery tests pass
- [x] Demo script is repeatable
- [x] Scope has not expanded beyond v1 commitments
- [x] Product and technical docs are aligned

These checklist items are satisfied by the current repo state as of 2026-04-12. Final overall
demo-readiness sign-off is supported by the current local release evidence, including the
practical v1 performance baseline documented in `v1-nfr-baseline.md`.

## Deferred Post-v1 Candidates

These are reasonable future candidates once v1 is stable:

- richer presence and collaboration signals
- richer document semantics
- compaction or garbage collection for long-lived histories
- more ambitious offline workflows
- broader permission models
- UI and workflow expansion for non-technical audiences

## Post-v1 Scope Note

The next approved post-v1 run is a constrained offline queueing alpha.

Its currently accepted boundary is:

- browser-only
- one local browser profile or device
- canonical CRDT operation queueing
- provisional local state until replay succeeds

This note does not expand the v1 scope.
It exists so product and planning docs do not accidentally describe the next run as broad
offline-first support.

The next planning tranche after that alpha is observability UX for the shipped browser demo
shell. Its currently accepted direction is a minimal in-app debug panel for demo and operator
diagnosis while logs remain the primary technical audit source. Broader dashboards remain out
of scope.
