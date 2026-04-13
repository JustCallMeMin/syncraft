/**
 * resolveEditorInputMode decides whether browser input should be ignored, buffered, submitted live, or queued offline.
 */
export function resolveEditorInputMode(input) {
  if (input?.applyingRemoteState || input?.isComposingText) {
    return "ignore";
  }
  if (!input?.hasSessionIntent) {
    return "offline";
  }
  if (input?.isSocketOpen && !input?.sessionReady) {
    return "buffer";
  }
  if (input?.isSocketOpen && input?.sessionReady && input?.connectionState === "live") {
    return "live";
  }
  return "offline";
}
