# Syncraft Local Docs

## Purpose

This directory is the local, repo-friendly documentation mirror for Syncraft.
It is meant for employees and contributors who need to read, review, or update
project documentation without depending on Notion for every step.

Notion remains the primary shared memory and governance source. Local docs are
the working mirror used for:

- onboarding and day-to-day reading
- implementation-facing specs
- doc contributions in pull requests
- keeping repository docs aligned with actual code

## Start Here

- Product intent: [`01-product/prd-business-spec.md`](./01-product/prd-business-spec.md)
- Release gate: [`01-product/mvp-scope-release-criteria.md`](./01-product/mvp-scope-release-criteria.md)
- Browser demo entry: `go run ./cmd/demo-server`
- Canonical domain and protocol: [`02-canonical/`](./02-canonical/README.md)
- Planning and validation: [`03-planning-and-validation/`](./03-planning-and-validation/README.md)
- Documentation workflow: [`04-governance-and-rag/`](./04-governance-and-rag/README.md)

## Directory Map

- `01-product/`
  Product intent, scope, demo framing, and release criteria.
- `02-canonical/`
  Canonical architecture, domain model, protocol, invariants, glossary, and constraints.
- `03-planning-and-validation/`
  Milestone backlog, use case catalog, and validation-oriented docs.
- `04-governance-and-rag/`
  Local documentation workflow, RAG memory mapping, and doc-writing guidance.

## Source Policy

- Notion is the primary shared memory and approval surface for Syncraft governance.
- Local docs are the primary in-repo reading surface for contributors.
- When canonical behavior changes, update both Notion and the corresponding local doc
  in the same work session.
- Do not leave the repository with a local doc that contradicts the current accepted
  Notion policy.

## Contribution Rules

1. Edit the nearest canonical local document first when you are changing behavior,
   terminology, scope, or validation expectations.
2. Keep links relative inside `docs/` so the repo remains portable.
3. Prefer updating an existing canonical doc over creating a new overlapping note.
4. If a topic is unresolved, capture it in Notion first and avoid presenting it locally
   as settled policy.
5. If a document is temporary, exploratory, or draft-only, say so explicitly in the file.

## Maintenance Notes

- Use short, explicit file names.
- Keep one concept in one canonical file whenever possible.
- Add links forward and backward so readers can move through product, canonical spec,
  planning, and validation layers without search.
- Use `python scripts/notion_sync_docs.py check` to validate the Notion mirror map
  and `python scripts/notion_sync_docs.py sync` to refresh mapped local docs.
