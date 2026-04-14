/**
 * deriveSaveState maps browser collaboration state into docs-core save/sync chrome.
 */
export function deriveSaveState(input) {
  const connectionState = normalizeText(input?.connectionState) || "disconnected";
  const pendingQueueCount = Number.isInteger(input?.pendingQueueCount) ? input.pendingQueueCount : 0;
  const blockedReason = normalizeText(input?.blockedReason);
  const replayInFlight = Boolean(input?.replayInFlight);
  const liveSubmitInFlight = Boolean(input?.liveSubmitInFlight);
  const titleUpdateInFlight = Boolean(input?.titleUpdateInFlight);

  if (blockedReason) {
    return { label: "Blocked", tone: "blocked" };
  }
  if (connectionState === "connecting" || connectionState === "catching_up") {
    return { label: "Connecting", tone: "connecting" };
  }
  if (replayInFlight) {
    return { label: "Replaying queued changes", tone: "replaying" };
  }
  if (pendingQueueCount > 0 || connectionState === "offline_provisional" || connectionState === "disconnected") {
    return { label: "Offline (provisional)", tone: "offline" };
  }
  if (connectionState === "error") {
    return { label: "Blocked", tone: "error" };
  }
  if (liveSubmitInFlight || titleUpdateInFlight) {
    return { label: "Saving", tone: "saving" };
  }
  if (connectionState === "live") {
    return { label: "Saved", tone: "saved" };
  }
  return { label: "Connecting", tone: "connecting" };
}

function normalizeText(value) {
  return typeof value === "string" ? value.trim() : "";
}
