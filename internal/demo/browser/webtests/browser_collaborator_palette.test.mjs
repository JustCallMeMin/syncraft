import test from "node:test";
import assert from "node:assert/strict";

import { colorTokenForActor } from "../web/browser_collaborator_palette.js";

test("colorTokenForActor is stable for one actor id", () => {
  const first = colorTokenForActor("actor-1");
  const second = colorTokenForActor("actor-1");
  assert.equal(first, second);
});
