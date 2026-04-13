import test from "node:test";
import assert from "node:assert/strict";

import { computeTextDiff } from "../web/browser_text_diff.js";

test("computeTextDiff is rune-aware for multi-character unicode strings", () => {
  const diff = computeTextDiff("", "ế á");

  assert.deepEqual(diff, {
    start: 0,
    removed: "",
    inserted: "ế á",
  });
});

test("computeTextDiff keeps shared unicode prefix and suffix stable", () => {
  const diff = computeTextDiff("ế á", "ế ấ");

  assert.deepEqual(diff, {
    start: 2,
    removed: "á",
    inserted: "ấ",
  });
});
