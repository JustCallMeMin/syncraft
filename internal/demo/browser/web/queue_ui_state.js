/**
 * deriveQueueSurfaceState resolves the browser-facing queue state from connection and local queue facts.
 */
export function deriveQueueSurfaceState(input) {
  const connectionState = input?.connectionState || "disconnected";
  const pendingQueueCount = normalizePendingCount(input?.pendingQueueCount);
  const blockedReason = normalizeText(input?.blockedReason);
  const reconnectingWithQueue = Boolean(input?.reconnectingWithQueue);

  if (blockedReason) {
    return {
      state: "queue_blocked",
      queueSummary: blockedReason,
    };
  }

  if (reconnectingWithQueue && pendingQueueCount > 0) {
    return {
      state: "replaying_queue",
      queueSummary: `${pendingQueueCount} queued operation${pendingQueueCount === 1 ? "" : "s"} waiting for replay after reconnect`,
    };
  }

  if (connectionState === "disconnected" && pendingQueueCount > 0) {
    return {
      state: "offline_provisional",
      queueSummary: `${pendingQueueCount} queued operation${pendingQueueCount === 1 ? "" : "s"} remain provisional until replay succeeds`,
    };
  }

  if (pendingQueueCount > 0) {
    return {
      state: connectionState,
      queueSummary: `${pendingQueueCount} queued operation${pendingQueueCount === 1 ? "" : "s"} stored locally`,
    };
  }

  return {
    state: connectionState,
    queueSummary: "no queued local operations",
  };
}

function normalizePendingCount(value) {
  if (!Number.isInteger(value) || value < 0) {
    return 0;
  }
  return value;
}

function normalizeText(value) {
  if (typeof value !== "string") {
    return "";
  }
  return value.trim();
}
