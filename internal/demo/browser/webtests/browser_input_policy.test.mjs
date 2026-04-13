import test from "node:test";
import assert from "node:assert/strict";

import { resolveEditorInputMode } from "../web/browser_input_policy.js";

test("resolveEditorInputMode buffers early input while first session state is still pending", () => {
  assert.equal(resolveEditorInputMode({
    applyingRemoteState: false,
    isComposingText: false,
    hasSessionIntent: true,
    isSocketOpen: true,
    sessionReady: false,
    connectionState: "connecting",
  }), "buffer");
});

test("resolveEditorInputMode uses live mode only after session is ready and live", () => {
  assert.equal(resolveEditorInputMode({
    applyingRemoteState: false,
    isComposingText: false,
    hasSessionIntent: true,
    isSocketOpen: true,
    sessionReady: true,
    connectionState: "live",
  }), "live");
});

test("resolveEditorInputMode falls back to offline mode when disconnected", () => {
  assert.equal(resolveEditorInputMode({
    applyingRemoteState: false,
    isComposingText: false,
    hasSessionIntent: true,
    isSocketOpen: false,
    sessionReady: false,
    connectionState: "connecting",
  }), "offline");
});
