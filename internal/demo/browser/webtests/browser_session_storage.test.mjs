import test from "node:test";
import assert from "node:assert/strict";

import { loadSessionIntent, saveSessionIntent } from "../web/browser_session_storage.js";

test("loadSessionIntent returns null for missing or invalid storage values", () => {
  const storage = createStorage();
  assert.equal(loadSessionIntent(storage), null);
  storage.setItem("syncraft:last-session-intent", "{bad json");
  assert.equal(loadSessionIntent(storage), null);
});

test("saveSessionIntent and loadSessionIntent round-trip actor and document ids", () => {
  const storage = createStorage();
  saveSessionIntent({
    actorID: "actor-a",
    documentID: "doc-a",
    retry: true,
  }, storage);

  assert.deepEqual(loadSessionIntent(storage), {
    actorID: "actor-a",
    documentID: "doc-a",
    retry: true,
  });
});

function createStorage() {
  const values = new Map();
  return {
    getItem(key) {
      return values.has(key) ? values.get(key) : null;
    },
    setItem(key, value) {
      values.set(key, value);
    },
  };
}
