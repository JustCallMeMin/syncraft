import test from "node:test";
import assert from "node:assert/strict";

import { deriveQueueSurfaceState } from "../web/queue_ui_state.js";

test("deriveQueueSurfaceState reports offline provisional when disconnected with pending queue", () => {
  const result = deriveQueueSurfaceState({
    connectionState: "disconnected",
    pendingQueueCount: 2,
  });
  assert.equal(result.state, "offline_provisional");
  assert.match(result.queueSummary, /2 queued operations remain provisional/);
});

test("deriveQueueSurfaceState reports replaying queue during reconnect with pending queue", () => {
  const result = deriveQueueSurfaceState({
    connectionState: "connecting",
    pendingQueueCount: 3,
    reconnectingWithQueue: true,
  });
  assert.equal(result.state, "replaying_queue");
  assert.match(result.queueSummary, /3 queued operations waiting for replay/);
});

test("deriveQueueSurfaceState reports queue blocked when blocked reason exists", () => {
  const result = deriveQueueSurfaceState({
    connectionState: "live",
    pendingQueueCount: 1,
    blockedReason: "queue metadata is corrupted",
  });
  assert.equal(result.state, "queue_blocked");
  assert.equal(result.queueSummary, "queue metadata is corrupted");
});

test("deriveQueueSurfaceState reports no queued local operations by default", () => {
  const result = deriveQueueSurfaceState({
    connectionState: "live",
    pendingQueueCount: 0,
  });
  assert.equal(result.state, "live");
  assert.equal(result.queueSummary, "no queued local operations");
});
