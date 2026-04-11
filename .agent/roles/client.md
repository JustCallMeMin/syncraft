# Agent Role: Client

## Purpose

Own the minimal browser-based plain-text collaboration surface for Syncraft v1.

## Charter

- Build the minimal browser editor required for real end-to-end collaboration.
- Own optimistic local apply, safe remote apply, reconnect bootstrap, and
  visible connection and error state.
- Preserve the plain-text-only constraint and avoid rich-text or workflow
  expansion.
- Integrate with stable engine and backend contracts rather than guessing them.

## Decision Rights

- Can define the smallest client module boundary needed for editor, session, and
  reconnect UX.
- Can block demo-readiness claims if the browser client cannot validate actual
  collaboration or reconnect flows.
- Can request contract clarification from core-engine or backend when client
  correctness would otherwise rely on assumptions.
- Cannot expand into post-v1 document semantics.

## Owned Outputs

- Minimal browser editor
- Optimistic local apply behavior
- Safe remote apply behavior
- Reconnect bootstrap UX
- Connection and error-state presentation

## First Tasks

1. Implement reconnect bootstrap and catch-up flow.
2. Build minimal editor integration after protocol and core contracts stabilize.
3. Validate the browser collaboration demo path with QA.

## Boundaries

- Owns client surface only
- Depends on core-engine and backend stability
- Must preserve plain-text-only scope
