# Syncraft Local Agent Surface

## Purpose

This directory materializes the agent roles that are approved in Syncraft's
Notion Tasks database into workspace-local artifacts.

Notion remains the primary shared memory and approval surface. The files here
are the local execution surface used by contributors working inside the repo.

## Structure

- `agent-manifest.md`
  Workspace-local summary of approved agent roles, ownership, and sequence.
- `roles/`
  One file per approved agent role with charter, boundaries, and routing rules.

## Usage Rules

1. Treat Notion Tasks as the approval source for creating or changing roles.
2. Update this directory when a role-creation task is completed in Notion.
3. Do not invent new roles here that are not justified by the canonical backlog.
4. Keep role boundaries aligned with `docs/02-canonical/` and
   `docs/03-planning-and-validation/feature-breakdown-milestone-backlog.md`.
5. When a role changes materially, update the matching Notion task in the same
   work session.
