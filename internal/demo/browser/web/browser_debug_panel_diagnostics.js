import { deriveQueueSurfaceState } from "./queue_ui_state.js";

/**
 * deriveDebugPanelDiagnostics normalizes browser-shell state into debug-panel-friendly values.
 */
export function deriveDebugPanelDiagnostics(input) {
  const sessionIntent = input?.sessionIntent || null;
  const queueMetadata = input?.queueMetadata || null;
  const pendingQueueRecords = Array.isArray(input?.pendingQueueRecords) ? input.pendingQueueRecords : [];
  const queueBlockedReason = normalizeText(input?.queueBlockedReason);
  const reconnectingWithQueue = Boolean(input?.reconnectingWithQueue);
  const replayInFlight = Boolean(input?.replayInFlight);
  const browserStatusState = input?.browserStatusState || null;
  const debugEvents = Array.isArray(input?.debugEvents) ? input.debugEvents : [];

  const connectionState = normalizeText(browserStatusState?.getConnectionState?.()) || "disconnected";
  const lastError = normalizeText(browserStatusState?.getErrorText?.()) || "none";
  const pendingQueueCount = Number.isInteger(queueMetadata?.pending_queue_count)
    ? queueMetadata.pending_queue_count
    : pendingQueueRecords.length;
  const queueSurface = deriveQueueSurfaceState({
    connectionState,
    pendingQueueCount,
    blockedReason: queueBlockedReason,
    reconnectingWithQueue,
  });

  return {
    session: {
      actorInstanceID: sessionIntent?.actorID || "none",
      documentID: sessionIntent?.documentID || "none",
      connectionState,
      lastError,
    },
    queue: {
      queueState: queueSurface.state,
      pendingQueueCount: String(pendingQueueCount),
      queueSummary: queueSurface.queueSummary,
      lastSnapshotID: normalizeText(queueMetadata?.last_snapshot_id) || "none",
      lastOperationID: normalizeText(queueMetadata?.last_operation_id) || "none",
    },
    replay: {
      replayState: replayInFlight
        ? "replaying"
        : reconnectingWithQueue && pendingQueueCount > 0
          ? "waiting_for_reconnect"
          : pendingQueueCount > 0
            ? "queued_idle"
            : "idle",
      replayInFlight: replayInFlight ? "yes" : "no",
      blockedReason: queueBlockedReason || "none",
    },
    recentEvents: debugEvents.map((event) => ({
      eventID: normalizeText(event?.event_id) || "none",
      eventType: normalizeText(event?.event_type) || "unknown_event",
      level: normalizeText(event?.level) || "info",
      occurredAt: normalizeText(event?.occurred_at) || "none",
      actorID: normalizeText(event?.actor_id) || "none",
      documentID: normalizeText(event?.document_id) || "none",
      message: normalizeText(event?.message) || "browser debug event",
    })),
  };
}

function normalizeText(value) {
  if (typeof value !== "string") {
    return "";
  }
  return value.trim();
}
