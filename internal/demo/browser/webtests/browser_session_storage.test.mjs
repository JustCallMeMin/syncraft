import test from "node:test";
import assert from "node:assert/strict";

import { loadSessionIntent, saveSessionIntent } from "../web/browser_session_storage.js";

test("loadSessionIntent returns null when url query does not define actor/document", () => {
  assert.equal(loadSessionIntent({ search: "" }), null);
  assert.equal(loadSessionIntent({ search: "?actor=actor-a" }), null);
});

test("saveSessionIntent and loadSessionIntent round-trip actor and document ids through the tab url", () => {
  const locationLike = {
    href: "http://localhost:8080/",
    search: "",
  };
  const historyCalls = [];
  const historyLike = {
    replaceState(_state, _title, nextURL) {
      historyCalls.push(nextURL);
      locationLike.href = `http://localhost:8080${nextURL}`;
      locationLike.search = new URL(locationLike.href).search;
    },
  };

  saveSessionIntent({
    actorID: "actor-a",
    documentID: "doc-a",
    retry: true,
  }, locationLike, historyLike);

  assert.equal(historyCalls.length, 1);
  assert.equal(historyCalls[0], "/?actor=actor-a&document=doc-a");
  assert.deepEqual(loadSessionIntent(locationLike), {
    actorID: "actor-a",
    documentID: "doc-a",
    retry: true,
  });
});
