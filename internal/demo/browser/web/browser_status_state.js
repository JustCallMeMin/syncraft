/**
 * createBrowserStatusState keeps browser-shell connection state outside the rendered DOM.
 */
export function createBrowserStatusState() {
  let connectionState = "disconnected";
  let errorText = "none";

  return {
    getConnectionState() {
      return connectionState;
    },
    getErrorText() {
      return errorText;
    },
    set(nextConnectionState, nextErrorText) {
      connectionState = normalizeConnectionState(nextConnectionState);
      errorText = normalizeErrorText(nextErrorText);
      return {
        connectionState,
        errorText,
      };
    },
    reset() {
      connectionState = "disconnected";
      errorText = "none";
      return {
        connectionState,
        errorText,
      };
    },
  };
}

function normalizeConnectionState(value) {
  if (typeof value !== "string" || value.trim() === "") {
    return "disconnected";
  }
  return value.trim();
}

function normalizeErrorText(value) {
  if (typeof value !== "string" || value.trim() === "") {
    return "none";
  }
  return value.trim();
}
