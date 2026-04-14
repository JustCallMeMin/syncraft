import test from "node:test";
import assert from "node:assert/strict";

import {
  anchorForRuneIndex,
  runeIndexFromAnchor,
  runeIndexFromCodeUnitIndex,
} from "../web/browser_presence_mapping.js";

test("presence mapping keeps unicode indices rune-safe", () => {
  const text = "ếa";
  assert.equal(runeIndexFromCodeUnitIndex(text, 0), 0);
  assert.equal(runeIndexFromCodeUnitIndex(text, 1), 1);
  assert.equal(runeIndexFromCodeUnitIndex("🙂a", 2), 1);
});

test("anchorForRuneIndex and runeIndexFromAnchor round-trip visible elements", () => {
  const visibleElements = [
    { id: "elem_1", value: "ế" },
    { id: "elem_2", value: "b" },
  ];
  const anchor = anchorForRuneIndex(1, visibleElements);
  assert.equal(anchor.element_id, "elem_1");
  assert.equal(anchor.offset, 1);
  assert.equal(runeIndexFromAnchor(anchor, visibleElements), 1);
});
