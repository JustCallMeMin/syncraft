# Local Documentation Workflow

## Purpose

This guide explains how employees should work with Syncraft documentation locally
in the repository.

## Read Order

1. Start in [`../README.md`](../README.md).
2. Read the product layer in [`../01-product/`](../01-product/README.md).
3. Read canonical specs in [`../02-canonical/`](../02-canonical/README.md).
4. Use planning docs in [`../03-planning-and-validation/`](../03-planning-and-validation/README.md)
   when implementing or validating work.

## When To Edit Local Docs

Update local docs whenever:

- code changes behavior, protocol, persistence, or recovery semantics
- scope or release criteria change
- terminology changes
- a use case or validation rule changes
- a document would otherwise become stale or misleading for a repo reader

## How To Add A New Doc

1. Put the file in the nearest existing folder.
2. Add it to that folder's `README.md`.
3. Add backlinks to the most relevant neighboring docs.
4. Avoid creating a duplicate of an existing canonical concept.

## Canonical Rule

If a topic already has a canonical file in `docs/`, update that file instead of
creating a new note.

## Notion Sync Rule

Notion remains the primary shared memory and approval surface.
When a local canonical doc changes meaningfully, update the matching Notion record
in the same work session.
