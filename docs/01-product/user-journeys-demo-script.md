# Syncraft User Journeys / Demo Script

## Purpose

This document turns the Syncraft PRD into concrete user-facing journeys and a repeatable
demo sequence. It exists to make the product value legible during implementation reviews,
stakeholder walkthroughs, and final demos.

The focus is not feature breadth. The focus is demonstrating that Syncraft v1 solves the
right problem: correct and recoverable real-time plain-text collaboration.

## Demo Principles

- Show correctness before polish.
- Use plain-text scenarios that are easy to understand visually.
- Keep each demo step tied to one product claim.
- Prefer observable recovery and convergence behavior over UI ornament.
- Avoid adding flows that imply rich-text, workflow, or enterprise scope.

## Primary Audience

- reviewers evaluating technical credibility
- instructors or judges watching a milestone demo
- teammates validating whether implementation still matches the product promise

## Core Product Claims To Demonstrate

- multiple users can edit the same plain-text document concurrently
- concurrent changes converge to one stable visible result
- reconnect restores correct state through snapshot plus delta
- duplicate and out-of-order delivery do not corrupt the final text
- persisted data supports recovery after server restart

## Journey 1: First-Time Shared Editing

### User Goal

Two collaborators want to edit the same shared plain-text document at the same time.

### Setup

- open the same document in two browser sessions
- ensure both sessions are connected to the same backend
- start with a short readable text sample

### Flow

1. User A opens the document and types a short line.
2. User B opens the same document and confirms the text appears.
3. User A and User B both insert text at nearby positions.
4. Both sessions show the same final visible text.

### Product Value Shown

- real-time collaboration works
- Syncraft is not single-writer
- the system converges under concurrent editing

### Demo Notes

- use a sentence short enough that the audience can track the change visually
- call out that correctness does not depend on one client being authoritative

## Journey 2: Concurrent Insert At The Same Logical Position

### User Goal

Two collaborators make conflicting inserts at the same location without losing data.

### Setup

- start with a known base string such as `hello`
- place both users at the same insertion point

### Flow

1. User A inserts one token at the shared position.
2. User B inserts a different token at the same position at nearly the same time.
3. Both clients receive remote operations.
4. The final text stabilizes to the same ordering on both replicas.

### Product Value Shown

- Syncraft handles the core CRDT conflict scenario directly
- deterministic ordering rules produce one converged result

### Demo Notes

- explain that the exact ordering is less important than consistency across replicas
- keep the inserted tokens visually distinct

## Journey 3: Reconnect And Catch Up

### User Goal

A user temporarily loses connection and returns to the correct document state without
manual repair.

### Setup

- start with two connected clients
- disconnect one client or suspend its network access

### Flow

1. User B disconnects.
2. User A continues editing the document.
3. User B reconnects.
4. The backend serves the current state through snapshot plus remaining operations.
5. User B catches up and sees the same final text as User A.

### Product Value Shown

- reconnect behavior is safe and understandable
- recovery is a first-class feature, not an afterthought

### Demo Notes

- narrate the reconnect steps in plain language
- avoid deep protocol detail unless the audience is technical

## Journey 4: Delivery Irregularities Do Not Corrupt State

### User Goal

The system remains correct even when the network behaves badly.

### Setup

- use a test harness, debug mode, or prepared environment that can simulate duplicate or
  delayed operations

### Flow

1. Start from a clean shared document.
2. Trigger a small concurrent edit scenario.
3. Introduce duplicate or delayed delivery.
4. Show that the visible final state still converges.

### Product Value Shown

- Syncraft is resilient under realistic delivery failures
- the product claim is stronger than “works when the network is perfect”

### Demo Notes

- this can be shown through logs, a test harness view, or a controlled debug panel
- keep the scenario compact so the audience can still track the text outcome

## Journey 5: Server Restart Recovery

### User Goal

Work survives backend restart and rebuild.

### Setup

- use a document with a short but non-trivial edit history
- persist operation log and snapshot data before restart

### Flow

1. Users create and edit a shared document.
2. Stop the backend.
3. Restart the backend.
4. Reopen or reconnect clients.
5. The rebuilt document state matches the state before restart.

### Product Value Shown

- persistence is not cosmetic
- Syncraft can recover from operational disruption without losing document correctness

### Demo Notes

- explicitly state that recovery comes from persisted state, not hidden client caching

## Recommended Demo Order

Use this order for the primary milestone demo:

1. First-time shared editing
2. Concurrent insert at the same logical position
3. Reconnect and catch up
4. Server restart recovery
5. Delivery irregularities do not corrupt state

This order moves from easiest-to-understand value to strongest technical proof.

## Demo Script Outline

### Opening

“Syncraft is a real-time plain-text collaboration system. The v1 goal is not rich text or
workflow features. The goal is to prove correct convergence and reliable recovery when
multiple users edit the same document.”

### Mid-Demo Explanation

“What matters here is that each client reaches the same final visible text, even when edits
happen concurrently or one client temporarily falls behind.”

### Closing

“This demo shows that Syncraft v1 delivers the core product promise: collaborative text
editing that is correct, deterministic to recover, and resilient to operational disruption.”

## Pass Conditions For A Live Demo

- both clients show the same final visible text after concurrent editing
- reconnect restores the same visible state as a continuously connected client
- restart recovery preserves the document state
- no demo step depends on manually editing the database or forcing a hidden reset

## Failure Signals

- two replicas show different final text
- reconnect requires a manual refresh that bypasses the intended recovery flow
- restart recovery produces missing or duplicated text
- the demo can only succeed under ideal in-order single-client conditions
