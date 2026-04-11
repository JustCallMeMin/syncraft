# Syncraft Agent Operating Rules

## Purpose

This file defines project-specific agent rules for Syncraft. All agents working in this repository must follow these rules in addition to higher-level system instructions.

## Primary Project Memory

Syncraft uses Notion as its primary shared RAG memory.

Knowledge Hub:

- `https://www.notion.so/33ce98673be48196a5e1eca86bcc259f`

Seed assessment document:

- `https://www.notion.so/33ce98673be481c0abd7fcbed0073f3a`

Core databases:

- Architecture Decisions: `https://www.notion.so/831d8b1d3f9340a3b75c99f35e5c5e69`
- Tasks: `https://www.notion.so/435b018569aa477baade8a0acb53e6fa`
- Risks: `https://www.notion.so/a8845f1c996b4058b9b23715ee6e6897`
- Learnings and Incidents: `https://www.notion.so/6f6077459090400c8b261ca7252d011a`

Canonical knowledge layers:

- System Overview: `https://www.notion.so/33de98673be48114a83ee2e91f569842`
- Canonical Architecture: `https://www.notion.so/33de98673be4813ca60beca82fcb8742`
- Canonical Protocol: `https://www.notion.so/33ce98673be48146ba55f1f938a9b635`
- Canonical Invariants: `https://www.notion.so/33de98673be481508444db2143e44c02`
- Canonical Constraints and Non-Goals: `https://www.notion.so/33de98673be481beb7cdc743a9d6e77c`

Open uncertainty tracking:

- Open Questions Register: `https://www.notion.so/33de98673be4815ea278eab391dcaf45`
- Decision Queue: `https://www.notion.so/33de98673be481679d04c2dba6a0fc0a`
- Assumptions Log: `https://www.notion.so/33de98673be48155ba5cf6296b27909c`

Operational memory:

- Change Log: `https://www.notion.so/33de98673be48125a3b5dd8f73dc9440`
- Operational Timeline: `https://www.notion.so/33de98673be481ae988fdc02de94001f`
- Known Issues and Known Limitations: `https://www.notion.so/33de98673be48107bbbfeff0010dc0f2`

Navigation and linking discipline:

- Navigation and Linking Discipline: `https://www.notion.so/33de98673be481ada75ee3f63b50a9db`

Review and freshness governance:

- Review and Freshness Governance: `https://www.notion.so/33de98673be4812ca0f9fed158b804d8`

RAG-specific support docs:

- FAQ: `https://www.notion.so/33de98673be4817c8a42c1ebd03c2575`
- Glossary: `https://www.notion.so/33ce98673be481559d46e77a68407d35`
- Scenario Library: `https://www.notion.so/33de98673be48115a8e7feb8cae0639d`
- Examples Library: `https://www.notion.so/33de98673be48134b071c03e4f69155e`

Traceability discipline:

- Traceability Discipline: `https://www.notion.so/33de98673be4815ca6d1c76cde719836`

## Memory Architecture

Use this map when deciding where project knowledge belongs.

- Knowledge Hub is the routing page and top-level source-of-truth map.
- Architecture Decisions is the canonical database for accepted architecture policy.
- Tasks is the canonical database for actionable work, ownership, status, and dependencies.
- Risks is the canonical database for active delivery and technical risk posture.
- Learnings and Incidents is the canonical database for reusable debugging knowledge and incident history.
- System Overview is the canonical executive summary of the system.
- Canonical Architecture is the canonical system structure and responsibility map.
- Canonical Protocol is the canonical sync protocol definition.
- Canonical Invariants is the canonical correctness invariant set.
- Canonical Constraints and Non-Goals is the canonical v1 boundary and exclusion page.
- Change Log is the canonical summary of important changes over time.
- Operational Timeline is the canonical index of incidents, debugging sessions, and operationally important events.
- Known Issues and Known Limitations is the canonical list of accepted limitations and current known issues.
- Navigation and Linking Discipline is the canonical rule set for backlinks, related-doc links, sibling links, and orphan prevention.
- Review and Freshness Governance is the canonical rule set for freshness ownership, review SLA, and stale or deprecated handling.
- FAQ is the canonical repeated-question page for high-frequency Syncraft answers.
- Domain Glossary is the canonical terminology page for Syncraft terms.
- Scenario Library is the canonical entry point for standard scenarios and expected outcomes.
- Examples Library is the canonical short-form example library for operation, reconnect, replay, and duplicate-delivery flows.
- Traceability Discipline is the canonical rule set for requirement, decision, task, risk, incident, and learning link chains.
- Open Questions Register is the canonical list of unresolved questions.
- Decision Queue is the canonical list of decisions that must be settled before affected work should proceed.
- Assumptions Log is the canonical list of temporary assumptions currently in use.

## Memory Update Checklist

Use this checklist after any substantial Syncraft task.

1. Identify which canonical source changed.
2. Update the relevant canonical knowledge-layer page if system summary, architecture, protocol, invariants, or constraints changed.
3. Update Tasks if scope, status, owner, reviewer, dependencies, or execution outcome changed.
4. Update Risks if a material technical or delivery risk was introduced, changed, mitigated, accepted, or closed.
5. Update Learnings and Incidents if the work produced reusable debugging knowledge or incident context.
6. Update Change Log, Operational Timeline, or Known Issues and Known Limitations if the work changed project history, operational understanding, or accepted limitations.
7. Update Open Questions Register, Decision Queue, or Assumptions Log if the work depends on unresolved answers, pending decisions, or temporary assumptions.
8. Apply Navigation and Linking Discipline if the work creates or substantially revises canonical pages, catalogs, scenario pages, or evidence landing pages.
9. Apply Review and Freshness Governance when page ownership, review state, or authoritative status changed.
10. Apply Traceability Discipline so requirement, decision, task, risk, incident, and learning links are explicit.
11. Update FAQ, Domain Glossary, Scenario Library, or Examples Library when repeated explanations, terminology, standard scenarios, or canonical examples changed.
12. Update local docs in `docs/` and keep them aligned with Notion when implemented behavior changed.
13. Ensure the final recorded state is explicit, traceable, and not left only in chat, terminal output, or blank task pages.

## Mandatory Notion Workflow

Before starting substantial work, agents must:

1. consult the Syncraft Notion Knowledge Hub and relevant databases
2. consult the canonical knowledge-layer pages relevant to the work, especially architecture, protocol, invariants, and constraints
3. consult the operational-memory pages when the work may change accepted limitations, recent project history, or incident/debug context
4. consult the uncertainty-tracking pages when the work touches unresolved questions, pre-code decisions, or temporary assumptions
5. consult the navigation and linking discipline when the work creates or substantially revises memory pages
6. consult review and freshness governance when relying on canonical pages whose recency or authority may be uncertain
7. consult traceability discipline when the work spans requirement, decision, implementation, risk, incident, or learning chains
8. consult FAQ, Domain Glossary, Scenario Library, and Examples Library when the work touches repeated explanations, terminology, standard scenarios, or common protocol flows
9. check for existing decisions, risks, tasks, and prior learnings related to the work
10. avoid introducing changes that conflict with accepted ADRs, canonical invariants, or tracked risks without explicitly recording the change

After completing substantial work, agents must:

1. update the relevant Notion memory
2. update the affected canonical knowledge-layer page when the work changes the executive system summary, architecture, protocol, invariants, or v1 constraints
3. update Change Log, Operational Timeline, or Known Issues and Known Limitations when the work changes project history, operational context, or accepted limitations
4. update Open Questions Register, Decision Queue, or Assumptions Log when the work reveals unresolved questions, pre-code decisions, or temporary assumptions
5. apply Navigation and Linking Discipline so new or revised pages are discoverable without search
6. apply Review and Freshness Governance so touched pages have current `Last Reviewed`, status, and freshness ownership
7. apply Traceability Discipline so tasks, risks, incidents, and learnings preserve explicit link chains
8. update FAQ, Domain Glossary, Scenario Library, or Examples Library when repeated answers, terminology, standard scenarios, or canonical examples changed
9. record new technical decisions in Architecture Decisions when the work changes system behavior, interfaces, protocol rules, persistence rules, or architectural direction
10. update or create Tasks entries when implementation scope, ownership, dependencies, or status changes
11. update or create Risks entries when new delivery or technical risks are discovered, mitigated, accepted, or closed
12. record debugging outcomes, incidents, and reusable findings in Learnings and Incidents when they are likely to help future work

## Definition of Substantial Work

Work is considered substantial if it includes any of the following:

- architecture or protocol changes
- new persistence or schema changes
- synchronization logic changes
- CRDT behavior changes
- security-sensitive changes
- completion or reassignment of implementation tasks
- discovery of blockers, incidents, or major technical uncertainty

## Decision Governance

Agents must treat accepted ADRs in Notion as current project policy unless they are explicitly superseded by a newer decision record.
Agents must also treat the canonical knowledge-layer pages as current project policy for their respective scope unless they are explicitly superseded.
Agents must treat the Open Questions Register, Decision Queue, and Assumptions Log as the current uncertainty ledger for Syncraft work.
Agents must treat the Change Log, Operational Timeline, and Known Issues and Known Limitations pages as the current operational-memory ledger for Syncraft work.

If a proposed implementation conflicts with an accepted ADR, the agent must:

1. stop treating the change as implicit
2. document the new proposal in Notion
3. record why the previous decision no longer fits
4. proceed only with explicit acknowledgment of the change in project memory

If a proposed implementation conflicts with the Canonical Architecture, Canonical Protocol, Canonical Invariants, or Canonical Constraints and Non-Goals, the agent must:

1. stop treating the change as implicit
2. update the relevant canonical page or record the proposed change alongside the affected canonical layer
3. document why the previous canonical statement no longer fits
4. proceed only with explicit acknowledgment of the change in project memory

If work proceeds using a temporary assumption or depends on an unresolved answer, the agent must:

1. record the unresolved item in the Open Questions Register, Decision Queue, or Assumptions Log as appropriate
2. make the assumption or pending decision visible before relying on it in substantial work
3. promote the item into an ADR or canonical page once it becomes accepted policy

## Task Governance

All implementation tasks should be represented in the Tasks database when they become actionable.

Tasks must include:

- a clear title
- current status
- responsible agent
- reviewer agent when required
- dependencies when relevant
- notes that explain the expected outcome

## Risk Governance

All material technical or delivery risks must be recorded in the Risks database.

Risk entries should include:

- severity
- likelihood
- owner
- trigger
- mitigation
- current status

## Learning Capture

Agents must prefer durable capture over ephemeral chat summaries.

If a debugging session, failed experiment, or production-style issue reveals something reusable, agents must add a concise learning or incident record to Notion instead of leaving the information only in terminal output or conversation history.

## Operational Memory

Agents must keep operational memory explicit and easy to retrieve.

- Use Change Log for important project-level changes over time.
- Use Operational Timeline as the index for incidents, debugging sessions, and operationally important events.
- Use Known Issues and Known Limitations for accepted limitations and current known issues.
- If an incident or debugging session is important enough to preserve, update both the detailed Learning or Incident record and the Operational Timeline.

## Navigation And Linking Discipline

Agents must keep important Notion pages discoverable without relying on search alone.

- Use the Navigation and Linking Discipline page for minimum backlink, sibling-link, and orphan-prevention rules.
- When creating a canonical page, update at least one upstream page so the new page has a discovery path.
- Keep `Related Docs` current on canonical pages, catalogs, scenario pages, and evidence landing pages.
- Do not leave important pages reachable from only one direction unless they are intentionally temporary or private.

## Review And Freshness Governance

Agents must keep authoritative memory current enough to trust.

- Use the Review and Freshness Governance page for freshness ownership, review SLA, and stale/deprecated handling.
- Every canonical page should have an explicit or inferable freshness owner.
- Update `Last Reviewed` when revalidating a page, even if the page content did not materially change.
- If a page is no longer confidently current, downgrade its status instead of leaving an unqualified canonical claim in place.
- Do not rely on stale or deprecated pages as current implementation policy without first revalidating or superseding them.

## RAG Support Docs

Agents must maintain the retrieval-oriented support layer, not just the canonical specs.

- Use FAQ for repeated questions that would otherwise be answered ad hoc across tasks or chat threads.
- Use Domain Glossary to keep terminology stable and reduce ambiguity across specs, code, tasks, and tests.
- Use Scenario Library as the entry point for standard scenarios and their expected outcomes.
- Use Examples Library for short concrete flows such as operation submission, reconnect, replay, duplicate delivery, and out-of-order handling.
- Do not let FAQ answers, glossary entries, scenarios, or examples drift away from the canonical architecture, protocol, invariants, or constraints pages.

## Traceability Discipline

Agents must preserve chain reasoning across Syncraft memory.

- Use the Traceability Discipline page when work spans requirement, decision, task, risk, incident, or learning records.
- Every substantial task should link to its source requirement or source spec, its related ADR or canonical decision page, and its related risk when one exists.
- Every substantial incident or learning record should link to the triggering task, the related decision or canonical page, and the resulting learning, mitigation, or follow-up task.
- Use Traceability Matrix for scenario-level and release-level mapping, and use page-local `Related Docs` plus template traceability sections for record-level chains.
- If a reader cannot move from requirement to decision to implementation to validation to incident history without search, the traceability is incomplete.

## Local Documentation Sync

Notion is the primary shared memory, but local repository documentation must remain consistent with implemented behavior.

### Local Docs Architecture

The local documentation mirror under `docs/` is organized as follows:

- `docs/README.md` is the local entry point for repo readers.
- `docs/01-product/` contains product intent, MVP scope, NFR baseline, and demo-facing docs.
- `docs/02-canonical/` contains canonical architecture, domain model, protocol, invariants, constraints, and glossary docs.
- `docs/03-planning-and-validation/` contains milestone backlog, use case catalog, and implementation-facing use case docs.
- `docs/04-governance-and-rag/` contains local documentation workflow and RAG memory mapping guidance for contributors.

The current canonical local files are:

- `docs/01-product/prd-business-spec.md`
- `docs/01-product/mvp-scope-release-criteria.md`
- `docs/01-product/v1-nfr-baseline.md`
- `docs/01-product/user-journeys-demo-script.md`
- `docs/02-canonical/canonical-architecture.md`
- `docs/02-canonical/canonical-domain-model.md`
- `docs/02-canonical/protocol-spec.md`
- `docs/02-canonical/canonical-invariants.md`
- `docs/02-canonical/canonical-constraints-and-non-goals.md`
- `docs/02-canonical/domain-glossary.md`
- `docs/03-planning-and-validation/feature-breakdown-milestone-backlog.md`
- `docs/03-planning-and-validation/use-case-catalog.md`

Agents must prefer updating these canonical local docs instead of creating ad hoc flat notes in `docs/`.

The local Notion mirror workflow is:

- validate mappings with `python scripts/notion_sync_docs.py check`
- refresh mapped docs with `python scripts/notion_sync_docs.py sync`
- maintain `docs/notion-sync-map.json` when adding or moving mirrored canonical files
- avoid editing generated mirror files and the Notion source in contradictory ways during the same task

When code changes modify architecture, protocol semantics, or operational assumptions, agents must:

- update the relevant local docs in `docs/`
- keep local docs and Notion records aligned
- avoid leaving contradictory versions of the same decision in the repo and in Notion

When updating local docs or AGENTS guidance, agents must keep the following canonical-layer meanings aligned:

- System Overview for executive system shape and value statement
- Canonical Architecture for component boundaries and responsibilities
- Canonical Protocol for current sync protocol behavior
- Canonical Invariants for official correctness invariants
- Canonical Constraints and Non-Goals for v1 boundaries and exclusions

## Scope Discipline

Agents must preserve the approved Syncraft v1 direction:

- plain text only
- CRDT correctness first
- server as relay plus persistence, not central conflict resolver
- deterministic replay and convergence as non-negotiable goals

Any proposed deviation from this direction must be captured in Notion before it is treated as accepted project direction.

## Uncertainty Discipline

Agents must not leave material uncertainty implicit.

- Use Open Questions Register for unresolved points that still need an answer.
- Use Decision Queue for decisions that must be settled before affected code should proceed or expand.
- Use Assumptions Log for temporary assumptions currently relied on by planning or implementation.
- If an assumption becomes risky or long-lived, escalate it into a decision or risk record.
