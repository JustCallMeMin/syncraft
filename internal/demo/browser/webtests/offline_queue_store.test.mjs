import test from "node:test";
import assert from "node:assert/strict";

import { OfflineQueueStore } from "../web/offline_queue_store.js";

test("append and loadPending persist canonical queued operations", async () => {
  const store = new OfflineQueueStore(createFakeIndexedDB());
  await store.open();

  await store.append(makeRecord({ operation_id: "op-2", actor_counter: 2 }));
  await store.append(makeRecord({ operation_id: "op-1", actor_counter: 1 }));
  await store.append(makeRecord({
    operation_id: "op-other",
    actor_counter: 1,
    document_id: "other-doc",
    operation: {
      operation_id: "op-other",
      actor_id: "actor-a",
      actor_counter: 1,
      type: "insert",
      payload: { value: "z" },
    },
  }));

  const pending = await store.loadPending("doc-a", "actor-a");
  assert.equal(pending.length, 2);
  assert.deepEqual(pending.map((record) => record.operation_id), ["op-1", "op-2"]);
});

test("markReplayed and clearReplayed update stored queue status", async () => {
  const store = new OfflineQueueStore(createFakeIndexedDB());
  await store.open();
  await store.append(makeRecord({ operation_id: "op-1", actor_counter: 1 }));

  const replayed = await store.markReplayed("doc-a", "actor-a", "op-1");
  assert.equal(replayed.status, "replayed");

  const pending = await store.loadPending("doc-a", "actor-a");
  assert.equal(pending.length, 0);

  const removed = await store.clearReplayed("doc-a", "actor-a");
  assert.equal(removed, 1);
});

test("markBlocked preserves reason for corrupted queue handling", async () => {
  const store = new OfflineQueueStore(createFakeIndexedDB());
  await store.open();
  await store.append(makeRecord({ operation_id: "op-1", actor_counter: 1 }));

  const blocked = await store.markBlocked("doc-a", "actor-a", "op-1", "payload checksum mismatch");
  assert.equal(blocked.status, "blocked");
  assert.equal(blocked.failure_reason, "payload checksum mismatch");
});

test("saveMetadata and loadMetadata persist actor counter continuity", async () => {
  const store = new OfflineQueueStore(createFakeIndexedDB());
  await store.open();

  await store.saveMetadata({
    document_id: "doc-a",
    actor_id: "actor-a",
    next_actor_counter: 14,
    pending_queue_count: 2,
    last_snapshot_id: "snap-1",
    last_operation_id: "op-13",
  });

  const metadata = await store.loadMetadata("doc-a", "actor-a");
  assert.equal(metadata.next_actor_counter, 14);
  assert.equal(metadata.pending_queue_count, 2);
  assert.equal(metadata.last_snapshot_id, "snap-1");
  assert.equal(metadata.last_operation_id, "op-13");
});

test("append rejects malformed canonical records", async () => {
  const store = new OfflineQueueStore(createFakeIndexedDB());
  await store.open();

  await assert.rejects(
    store.append({
      document_id: "doc-a",
      actor_id: "actor-a",
      operation_id: "op-1",
      actor_counter: 1,
      operation: {
        operation_id: "op-1",
        actor_id: "actor-a",
        actor_counter: 2,
      },
    }),
    /operation\.actor_counter must match actor_counter/,
  );
});

test("loadPending rejects corrupted queued operation records already in storage", async () => {
  const store = new OfflineQueueStore(createFakeIndexedDB());
  const db = await store.open();
  db.transaction(["queued_operations"], "readwrite").objectStore("queued_operations").put({
    record_key: "doc-a::actor-a::op-bad",
    document_id: "doc-a",
    actor_id: "actor-a",
    operation_id: "op-bad",
    actor_counter: 1,
    queued_at: "2026-04-13T00:00:00Z",
    status: "pending",
    document_actor_status: ["doc-a", "actor-a", "pending"],
    document_actor: ["doc-a", "actor-a"],
    operation: {
      operation_id: "op-bad",
      actor_id: "actor-a",
      actor_counter: 2,
      type: "insert",
      payload: { value: "x" },
    },
  });

  await assert.rejects(
    store.loadPending("doc-a", "actor-a"),
    /stored queued operation 1 is corrupted: operation\.actor_counter must match actor_counter/,
  );
});

test("loadMetadata rejects corrupted persisted metadata", async () => {
  const store = new OfflineQueueStore(createFakeIndexedDB());
  const db = await store.open();
  db.transaction(["queue_metadata"], "readwrite").objectStore("queue_metadata").put({
    metadata_key: "doc-a::actor-a",
    document_id: "doc-a",
    actor_id: "actor-a",
    next_actor_counter: 0,
    pending_queue_count: 1,
    last_snapshot_id: null,
    last_operation_id: null,
  });

  await assert.rejects(
    store.loadMetadata("doc-a", "actor-a"),
    /stored queue metadata is corrupted: next_actor_counter must be a positive integer/,
  );
});

function makeRecord(overrides = {}) {
  const operationID = overrides.operation_id || "op-1";
  const actorID = overrides.actor_id || "actor-a";
  const actorCounter = overrides.actor_counter || 1;
  return {
    document_id: overrides.document_id || "doc-a",
    actor_id: actorID,
    operation_id: operationID,
    actor_counter: actorCounter,
    queued_at: overrides.queued_at || "2026-04-13T00:00:00Z",
    status: overrides.status || "pending",
    operation: overrides.operation || {
      operation_id: operationID,
      actor_id: actorID,
      actor_counter: actorCounter,
      type: "insert",
      payload: { value: "x" },
    },
  };
}

function createFakeIndexedDB() {
  const databases = new Map();
  return {
    open(name) {
      const request = makeRequest();
      queueMicrotask(() => {
        let db = databases.get(name);
        const isNew = !db;
        if (!db) {
          db = createFakeDatabase();
          databases.set(name, db);
        }
        request.result = db;
        if (isNew && typeof request.onupgradeneeded === "function") {
          request.onupgradeneeded({ target: request });
        }
        if (typeof request.onsuccess === "function") {
          request.onsuccess({ target: request });
        }
      });
      return request;
    },
  };
}

function createFakeDatabase() {
  const stores = new Map();
  const objectStoreNames = {
    contains(name) {
      return stores.has(name);
    },
  };

  return {
    objectStoreNames,
    createObjectStore(name, options = {}) {
      const store = createFakeStore(options.keyPath);
      stores.set(name, store);
      return store;
    },
    transaction(names) {
      const requested = Array.isArray(names) ? names : [names];
      return createFakeTransaction(requested, stores);
    },
  };
}

function createFakeTransaction(names, stores) {
  return {
    error: null,
    onabort: null,
    onerror: null,
    objectStore(name) {
      return stores.get(name ?? names[0]);
    },
  };
}

function createFakeStore(keyPath) {
  const records = new Map();
  const indexes = new Map();
  return {
    createIndex(name, field) {
      indexes.set(name, field);
    },
    index(name) {
      const field = indexes.get(name);
      return {
        getAll(value) {
          const request = makeRequest();
          queueMicrotask(() => {
            request.result = Array.from(records.values()).filter((record) => compareIndexValue(record[field], value));
            request.onsuccess?.({ target: request });
          });
          return request;
        },
        openCursor(value) {
          const request = makeRequest();
          const matching = Array.from(records.values()).filter((record) => compareIndexValue(record[field], value));
          let index = 0;
          const advance = () => {
            if (index >= matching.length) {
              request.result = null;
              request.onsuccess?.({ target: request });
              return;
            }
            const current = matching[index];
            request.result = {
              value: current,
              continue() {
                index += 1;
                queueMicrotask(advance);
              },
              delete() {
                const deleteRequest = makeRequest();
                queueMicrotask(() => {
                  records.delete(current[keyPath]);
                  deleteRequest.onsuccess?.({ target: deleteRequest });
                });
                return deleteRequest;
              },
            };
            request.onsuccess?.({ target: request });
          };
          queueMicrotask(advance);
          return request;
        },
      };
    },
    get(key) {
      const request = makeRequest();
      queueMicrotask(() => {
        request.result = records.get(key);
        request.onsuccess?.({ target: request });
      });
      return request;
    },
    put(value) {
      const request = makeRequest();
      queueMicrotask(() => {
        records.set(value[keyPath], JSON.parse(JSON.stringify(value)));
        request.result = value[keyPath];
        request.onsuccess?.({ target: request });
      });
      return request;
    },
  };
}

function makeRequest() {
  return {
    result: undefined,
    error: null,
    onsuccess: null,
    onerror: null,
    onupgradeneeded: null,
  };
}

function compareIndexValue(left, right) {
  return JSON.stringify(left) === JSON.stringify(right);
}
