import test from "node:test";
import assert from "node:assert/strict";

import { createDebugPanelState } from "../web/browser_debug_panel_state.js";

test("createDebugPanelState starts collapsed by default", () => {
  const state = createDebugPanelState();
  assert.equal(state.isExpanded(), false);
});

test("createDebugPanelState toggles and forces explicit panel state", () => {
  const state = createDebugPanelState(false);
  assert.equal(state.toggle(), true);
  assert.equal(state.isExpanded(), true);
  assert.equal(state.setExpanded(false), false);
  assert.equal(state.isExpanded(), false);
});
