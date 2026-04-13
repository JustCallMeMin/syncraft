import test from "node:test";
import assert from "node:assert/strict";

import {
  applyOperationToVisibleElements,
  createDeleteOperations,
  createInsertOperations,
} from "../web/canonical_queue_ops.js";

test("createInsertOperations builds canonical insert payloads from visible text context", () => {
  const result = createInsertOperations({
    documentID: "doc-a",
    actorID: "actor-a",
    startCounter: 3,
    index: 1,
    value: "xy",
    visibleElements: [
      { id: "elem-left", value: "a" },
      { id: "elem-right", value: "b" },
    ],
  });

  assert.equal(result.operations.length, 2);
  assert.deepEqual(result.operations.map((operation) => operation.operation_id), ["op_actor-a_3", "op_actor-a_4"]);
  assert.equal(result.operations[0].insert_payload.left_origin_id, "elem-left");
  assert.equal(result.operations[0].insert_payload.right_origin_id, "elem-right");
  assert.equal(result.operations[1].insert_payload.left_origin_id, "elem_actor-a_3");
  assert.equal(result.nextCounter, 5);
  assert.deepEqual(result.visibleElements.map((element) => element.value).join(""), "axyb");
});

test("createDeleteOperations targets visible element ids in order", () => {
  const result = createDeleteOperations({
    documentID: "doc-a",
    actorID: "actor-a",
    startCounter: 7,
    index: 1,
    count: 2,
    visibleElements: [
      { id: "elem-1", value: "a" },
      { id: "elem-2", value: "b" },
      { id: "elem-3", value: "c" },
    ],
  });

  assert.equal(result.operations.length, 2);
  assert.equal(result.operations[0].delete_payload.target_element_id, "elem-2");
  assert.equal(result.operations[1].delete_payload.target_element_id, "elem-3");
  assert.equal(result.nextCounter, 9);
  assert.deepEqual(result.visibleElements.map((element) => element.value).join(""), "a");
});

test("applyOperationToVisibleElements rejects invalid local origin context", () => {
  assert.throws(
    () => applyOperationToVisibleElements([], {
      document_id: "doc-a",
      operation_id: "op-a",
      actor_id: "actor-a",
      actor_counter: 1,
      type: "insert",
      insert_payload: {
        element_id: "elem-a",
        value: "a",
        left_origin_id: "missing",
      },
    }),
    /left origin missing is not visible locally/,
  );
});
