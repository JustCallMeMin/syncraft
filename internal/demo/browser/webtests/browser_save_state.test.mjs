import test from "node:test";
import assert from "node:assert/strict";

import { deriveSaveState } from "../web/browser_save_state.js";

test("deriveSaveState reports saved when live and idle", () => {
  assert.deepEqual(
    deriveSaveState({
      connectionState: "live",
      pendingQueueCount: 0,
      blockedReason: "",
      replayInFlight: false,
      liveSubmitInFlight: false,
      titleUpdateInFlight: false,
    }),
    { label: "Saved", tone: "saved" },
  );
});

test("deriveSaveState reports offline provisional with queued edits", () => {
  assert.deepEqual(
    deriveSaveState({
      connectionState: "disconnected",
      pendingQueueCount: 2,
      blockedReason: "",
      replayInFlight: false,
      liveSubmitInFlight: false,
      titleUpdateInFlight: false,
    }),
    { label: "Offline (provisional)", tone: "offline" },
  );
});
