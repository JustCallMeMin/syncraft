import test from "node:test";
import assert from "node:assert/strict";

import { createBrowserStatusState } from "../web/browser_status_state.js";
import { deriveDebugPanelDiagnostics } from "../web/browser_debug_panel_diagnostics.js";

test("deriveDebugPanelDiagnostics exposes current session, queue, and replay values", () => {
  const browserStatusState = createBrowserStatusState();
  browserStatusState.set("live", "none");

  const diagnostics = deriveDebugPanelDiagnostics({
    sessionIntent: {
      actorID: "actor-observe",
      documentID: "doc-observe",
    },
    browserStatusState,
    queueMetadata: {
      pending_queue_count: 2,
      last_snapshot_id: "snap-9",
      last_operation_id: "op-11",
    },
    pendingQueueRecords: [{}, {}],
    replayInFlight: true,
    debugEvents: [{
      event_id: "browser-event-4",
      event_type: "replay_started",
      level: "warning",
      occurred_at: "2026-04-14T10:00:00.000Z",
      actor_id: "actor-observe",
      document_id: "doc-observe",
      message: "Replay started.",
    }],
  });

  assert.deepEqual(diagnostics, {
    session: {
      actorInstanceID: "actor-observe",
      documentID: "doc-observe",
      connectionState: "live",
      lastError: "none",
    },
    queue: {
      queueState: "live",
      pendingQueueCount: "2",
      queueSummary: "2 queued operations stored locally",
      lastSnapshotID: "snap-9",
      lastOperationID: "op-11",
    },
    replay: {
      replayState: "replaying",
      replayInFlight: "yes",
      blockedReason: "none",
    },
    recentEvents: [{
      eventID: "browser-event-4",
      eventType: "replay_started",
      level: "warning",
      occurredAt: "2026-04-14T10:00:00.000Z",
      actorID: "actor-observe",
      documentID: "doc-observe",
      message: "Replay started.",
    }],
  });
});

test("deriveDebugPanelDiagnostics redacts missing values into safe defaults", () => {
  const diagnostics = deriveDebugPanelDiagnostics({
    browserStatusState: createBrowserStatusState(),
    queueBlockedReason: "metadata load failed",
  });

  assert.equal(diagnostics.session.actorInstanceID, "none");
  assert.equal(diagnostics.session.documentID, "none");
  assert.equal(diagnostics.queue.queueState, "queue_blocked");
  assert.equal(diagnostics.queue.lastSnapshotID, "none");
  assert.equal(diagnostics.replay.blockedReason, "metadata load failed");
  assert.deepEqual(diagnostics.recentEvents, []);
});
