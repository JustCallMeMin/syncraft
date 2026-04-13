import test from "node:test";
import assert from "node:assert/strict";

import { createBrowserStatusState } from "../web/browser_status_state.js";

test("browser status state clears sticky error across clean transitions", () => {
  const state = createBrowserStatusState();

  state.set("error", "websocket connection failed");
  const next = state.set("connecting", "");

  assert.deepEqual(next, {
    connectionState: "connecting",
    errorText: "none",
  });
});

test("browser status state resets to disconnected with no error", () => {
  const state = createBrowserStatusState();
  state.set("error", "temporary");

  assert.deepEqual(state.reset(), {
    connectionState: "disconnected",
    errorText: "none",
  });
});
