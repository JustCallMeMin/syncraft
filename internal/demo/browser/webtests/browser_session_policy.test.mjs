import test from "node:test";
import assert from "node:assert/strict";

import { isCurrentGeneration, shouldEnableEditor } from "../web/browser_session_policy.js";

test("isCurrentGeneration rejects stale socket generations", () => {
  assert.equal(isCurrentGeneration(4, 3), false);
  assert.equal(isCurrentGeneration(4, 4), true);
});

test("shouldEnableEditor keeps editor writable while disconnected with healthy queue state", () => {
  assert.equal(shouldEnableEditor({
    hasSessionIntent: true,
    replayInFlight: false,
    blockedReason: "",
    hasQueueStore: true,
    hasQueueMetadata: true,
    connectionState: "disconnected",
  }), true);
});

test("shouldEnableEditor disables editing for blocked or transient states", () => {
  assert.equal(shouldEnableEditor({
    hasSessionIntent: true,
    replayInFlight: false,
    blockedReason: "queue metadata is corrupted",
    hasQueueStore: true,
    hasQueueMetadata: true,
    connectionState: "disconnected",
  }), false);

  assert.equal(shouldEnableEditor({
    hasSessionIntent: true,
    replayInFlight: false,
    blockedReason: "",
    hasQueueStore: true,
    hasQueueMetadata: true,
    connectionState: "connecting",
  }), false);
});
