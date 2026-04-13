/**
 * isCurrentGeneration reports whether one socket event still belongs to the active browser session.
 */
export function isCurrentGeneration(currentGeneration, eventGeneration) {
  return Number.isInteger(currentGeneration) &&
    Number.isInteger(eventGeneration) &&
    currentGeneration > 0 &&
    currentGeneration === eventGeneration;
}

/**
 * shouldEnableEditor reports whether the browser editor should remain writable.
 */
export function shouldEnableEditor(input) {
  if (!input?.hasSessionIntent) {
    return false;
  }
  if (input?.replayInFlight) {
    return false;
  }
  if (typeof input?.blockedReason === "string" && input.blockedReason.trim() !== "") {
    return false;
  }
  if (!input?.hasQueueStore || !input?.hasQueueMetadata) {
    return false;
  }

  const connectionState = normalizeConnectionState(input?.connectionState);
  if (connectionState === "error" || connectionState === "connecting" || connectionState === "catching_up") {
    return false;
  }
  return true;
}

function normalizeConnectionState(value) {
  if (typeof value !== "string" || value.trim() === "") {
    return "disconnected";
  }
  return value.trim();
}
