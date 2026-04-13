# Syncraft PRD / Business Spec

## Purpose

This document defines the product intent for Syncraft v1 at the business and user level.
It describes who the product is for, what problems it solves, which use cases matter in v1,
what is explicitly out of scope, and how success should be measured.

This document is not an architecture decision record and does not replace technical design
documents. It exists to keep product scope aligned with the narrow v1 direction already
adopted for Syncraft.

## Product Summary

Syncraft is a real-time plain-text collaboration product built to make concurrent editing
correct, predictable, and recoverable under unreliable delivery conditions.

The v1 product is intentionally narrow. It is not trying to compete with full document
suites. It is trying to prove that teams can collaborate on the same plain-text document
with strong convergence guarantees, deterministic replay, and reliable recovery after
disconnects or server restarts.

## Problem Statement

Most collaborative editors optimize for a polished editing experience, but the underlying
merge and recovery semantics are often opaque. That makes them hard to trust for systems
demonstrations, research-oriented work, or engineering teams that need confidence in how
concurrent edits are applied and recovered.

Syncraft addresses a narrower problem: provide a collaborative plain-text editor where the
core value is not formatting or workspace breadth, but correctness under concurrency and
network disruption.

## Product Goals

- Deliver real-time collaboration for shared plain-text documents.
- Preserve convergence when multiple users edit concurrently.
- Recover safely from reconnects, duplicate delivery, out-of-order delivery, and server restart.
- Keep the user experience simple enough that the product can demonstrate correctness clearly.
- Provide a credible foundation for future expansion after v1 correctness is proven.

## Target Personas

### Persona 1: Systems Student or Researcher

This user cares about distributed systems behavior, not document styling.
They need a concrete product that demonstrates CRDT-based collaboration with observable,
defensible semantics.

Primary needs:

- understand what happens during concurrent editing
- trust that replicas converge
- inspect or explain recovery behavior

### Persona 2: Engineering Team Prototyping Collaborative Workflows

This user wants a lightweight collaborative text surface for shared notes, drafts, or
protocol content where plain text is enough and reliability matters more than formatting.

Primary needs:

- edit the same text with teammates in real time
- avoid silent corruption after reconnects
- trust recovery after transient backend or network failure

### Persona 3: Reviewer, Instructor, or Evaluator

This user is evaluating the product as a technical system.
They care that scope is disciplined, trade-offs are explicit, and success criteria are
clear and testable.

Primary needs:

- see that the v1 scope is intentionally constrained
- verify that the system solves a meaningful distributed collaboration problem
- evaluate success using concrete criteria instead of vague UX claims

## Core Use Cases

### Use Case 1: Concurrent Plain-Text Editing

Two or more users edit the same plain-text document at the same time and all replicas
converge to the same visible result.

### Use Case 2: Recovery After Temporary Disconnect

A user disconnects, reconnects, and completes a snapshot-plus-delta catch-up flow so the
local view returns to the correct document state without manual repair.

### Use Case 3: Tolerance to Delivery Irregularities

The system continues to produce the correct final document state even when operations are
delivered late, duplicated, or received in a different order.

### Use Case 4: Recovery After Server Restart

The backend rebuilds document state from persisted operations and snapshots so active
documents remain consistent after restart.

### Use Case 5: Minimal Multi-User Demo

A project team can run a clear demonstration showing multiple editors, concurrent changes,
reconnect behavior, and persistence-backed recovery without needing rich-text features.

In the current repo, this path is materialized through the shipped browser demo shell
served by `go run ./cmd/demo-server`.

## User Stories

- As a collaborator, I want to edit the same plain-text document as another user in real time so we can work together without overwriting each other.
- As a reconnecting user, I want the document to rebuild correctly after interruption so I do not need to refresh blindly or fix corruption manually.
- As a technical evaluator, I want success criteria tied to convergence and recovery so I can judge the product on system behavior instead of presentation polish.
- As a project team, I want the scope to stay narrow so we can finish a credible v1 instead of an incomplete all-purpose editor.

## Non-Goals

The following are explicitly out of scope for Syncraft v1:

- rich text formatting
- block-based document structure
- comments and review workflows
- advanced presence, avatars, or social collaboration features
- distributed undo and redo
- CRDT garbage collection and long-history compaction beyond basic snapshots
- enterprise permissions and policy management
- offline-first multi-device sync as a headline feature
- attempting to match the feature breadth of Google Docs, Notion, or similar suites

Post-v1 note:

- offline queueing may be explored as a narrow browser-only alpha after v1, but that does not
  change the current v1 non-goal boundary

## Key Product Assumptions

- Plain text is sufficient for the first milestone.
- Users and evaluators will value correctness and recovery over styling depth in v1.
- A narrower product with strong technical guarantees is more credible than a broader but
  weakly defined collaborative editor.
- Future expansion should happen only after core concurrency and replay behavior is trusted.

## Success Metrics

### Primary Success Metrics

- A live demo with at least two concurrent users editing the same document succeeds without divergence.
- Reconnect from snapshot plus delta restores the correct visible document state in repeated tests.
- Duplicate and out-of-order operation delivery do not change the final converged text.
- Server restart recovery reproduces the same document state from persisted data.

### Secondary Success Metrics

- A new user can understand the product value within a short demo focused on correctness and recovery.
- Core v1 scope remains intact without rich-text or workflow expansion creeping into the milestone.
- Engineering documentation and ADRs remain aligned with the implemented product direction.

## Constraints

- v1 is plain text only.
- CRDT correctness takes priority over feature breadth.
- The server acts as relay plus persistence, not as a central conflict resolver.
- Deterministic replay and convergence are non-negotiable.

## Current Delivery Shape

The currently shipped in-repo product surface is a minimal browser collaboration shell.
It exists to demonstrate the v1 claims through a real shared editor path, not to present
a polished standalone frontend product.

What this means in practice:

- the repo includes a browser-accessible plain-text editor for two-user collaboration demos
- reconnect and restart recovery can be shown through that browser path
- visible status and error state are part of the current operator-facing story
- UI polish, onboarding depth, and workspace-style product breadth are still intentionally limited

## Business Value of v1

Syncraft v1 creates value by proving a hard technical capability in a focused, demonstrable
form. The immediate value is not broad office productivity. The immediate value is:

- a trustworthy demonstration of collaborative editing correctness
- a foundation for future product or research expansion
- a technically defensible milestone for capstone, thesis, or prototype evaluation

## Post-v1 Direction

The next approved run after the v1 baseline is a narrow offline queueing alpha.

That direction is intentionally limited:

- browser-only
- one local browser profile or device
- provisional local offline state until replay succeeds
- replay through the existing reconnect and `submit_operation` path

This is not a broad offline-first product promise.
It is a constrained post-v1 experiment that must preserve the existing convergence and replay
story.

## Open Questions

- Which initial demo scenario best communicates value to non-technical reviewers?
- How much observability should be exposed in the product UI versus only in logs and test tooling?
