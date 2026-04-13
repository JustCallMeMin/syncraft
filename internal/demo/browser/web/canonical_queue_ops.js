/**
 * createInsertOperations builds canonical insert operations from one visible-text insertion.
 */
export function createInsertOperations(input) {
  const documentID = validateIdentifier("documentID", input?.documentID);
  const actorID = validateIdentifier("actorID", input?.actorID);
  const value = typeof input?.value === "string" ? input.value : "";
  const startCounter = validatePositiveInteger("startCounter", input?.startCounter);
  const visibleElements = cloneVisibleElements(input?.visibleElements || []);
  const initialIndex = validateInsertIndex(input?.index, visibleElements.length);

  if (value === "") {
    return {
      operations: [],
      visibleElements,
      nextCounter: startCounter,
    };
  }

  let nextCounter = startCounter;
  let nextVisibleElements = visibleElements;
  const operations = [];
  let nextIndex = initialIndex;
  for (const rune of Array.from(value)) {
    const left = nextIndex > 0 ? nextVisibleElements[nextIndex - 1].id : null;
    const right = nextIndex < nextVisibleElements.length ? nextVisibleElements[nextIndex].id : null;
    const operation = {
      document_id: documentID,
      operation_id: `op_${actorID}_${nextCounter}`,
      actor_id: actorID,
      actor_counter: nextCounter,
      type: "insert",
      insert_payload: {
        element_id: `elem_${actorID}_${nextCounter}`,
        value: rune,
        left_origin_id: left,
        right_origin_id: right,
      },
    };
    operations.push(operation);
    nextVisibleElements = applyOperationToVisibleElements(nextVisibleElements, operation);
    nextCounter += 1;
    nextIndex += 1;
  }

  return {
    operations,
    visibleElements: nextVisibleElements,
    nextCounter,
  };
}

/**
 * createDeleteOperations builds canonical delete operations from one visible-text removal span.
 */
export function createDeleteOperations(input) {
  const documentID = validateIdentifier("documentID", input?.documentID);
  const actorID = validateIdentifier("actorID", input?.actorID);
  const count = validateNonNegativeInteger("count", input?.count);
  const startCounter = validatePositiveInteger("startCounter", input?.startCounter);
  const visibleElements = cloneVisibleElements(input?.visibleElements || []);
  const index = validateDeleteIndex(input?.index, visibleElements.length, count);

  let nextCounter = startCounter;
  let nextVisibleElements = visibleElements;
  const operations = [];
  for (let offset = 0; offset < count; offset += 1) {
    const target = nextVisibleElements[index];
    if (!target) {
      throw new Error(`delete target at visible index ${index} was not found`);
    }
    const operation = {
      document_id: documentID,
      operation_id: `op_${actorID}_${nextCounter}`,
      actor_id: actorID,
      actor_counter: nextCounter,
      type: "delete",
      delete_payload: {
        target_element_id: target.id,
      },
    };
    operations.push(operation);
    nextVisibleElements = applyOperationToVisibleElements(nextVisibleElements, operation);
    nextCounter += 1;
  }

  return {
    operations,
    visibleElements: nextVisibleElements,
    nextCounter,
  };
}

/**
 * applyOperationToVisibleElements applies one canonical operation to the visible element list.
 */
export function applyOperationToVisibleElements(visibleElements, operation) {
  const nextVisibleElements = cloneVisibleElements(visibleElements);
  validateOperation(operation);

  if (operation.type === "insert") {
    const nextElement = {
      id: operation.insert_payload.element_id,
      value: operation.insert_payload.value,
    };
    const insertIndex = resolveInsertIndex(nextVisibleElements, operation.insert_payload);
    nextVisibleElements.splice(insertIndex, 0, nextElement);
    return nextVisibleElements;
  }

  const targetIndex = nextVisibleElements.findIndex((element) => element.id === operation.delete_payload.target_element_id);
  if (targetIndex === -1) {
    throw new Error(`delete target ${operation.delete_payload.target_element_id} is not visible locally`);
  }
  nextVisibleElements.splice(targetIndex, 1);
  return nextVisibleElements;
}

function resolveInsertIndex(visibleElements, payload) {
  const leftID = payload.left_origin_id || null;
  const rightID = payload.right_origin_id || null;

  if (leftID === null && rightID === null) {
    return 0;
  }
  if (leftID !== null) {
    const leftIndex = visibleElements.findIndex((element) => element.id === leftID);
    if (leftIndex === -1) {
      throw new Error(`left origin ${leftID} is not visible locally`);
    }
    return leftIndex + 1;
  }
  const rightIndex = visibleElements.findIndex((element) => element.id === rightID);
  if (rightIndex === -1) {
    throw new Error(`right origin ${rightID} is not visible locally`);
  }
  return rightIndex;
}

function validateOperation(operation) {
  if (!operation || typeof operation !== "object") {
    throw new Error("operation must be an object");
  }
  validateIdentifier("document_id", operation.document_id);
  validateIdentifier("operation_id", operation.operation_id);
  validateIdentifier("actor_id", operation.actor_id);
  validatePositiveInteger("actor_counter", operation.actor_counter);
  if (operation.type === "insert") {
    validateInsertPayload(operation.insert_payload);
    return;
  }
  if (operation.type === "delete") {
    validateDeletePayload(operation.delete_payload);
    return;
  }
  throw new Error(`unsupported operation type ${String(operation.type)}`);
}

function validateInsertPayload(payload) {
  if (!payload || typeof payload !== "object") {
    throw new Error("insert_payload must be an object");
  }
  validateIdentifier("insert_payload.element_id", payload.element_id);
  if (typeof payload.value !== "string" || Array.from(payload.value).length !== 1) {
    throw new Error("insert_payload.value must contain exactly one rune");
  }
  if (payload.left_origin_id !== null && payload.left_origin_id !== undefined) {
    validateIdentifier("insert_payload.left_origin_id", payload.left_origin_id);
  }
  if (payload.right_origin_id !== null && payload.right_origin_id !== undefined) {
    validateIdentifier("insert_payload.right_origin_id", payload.right_origin_id);
  }
}

function validateDeletePayload(payload) {
  if (!payload || typeof payload !== "object") {
    throw new Error("delete_payload must be an object");
  }
  validateIdentifier("delete_payload.target_element_id", payload.target_element_id);
}

function validateIdentifier(name, value) {
  if (typeof value !== "string" || value.trim() === "") {
    throw new Error(`${name} must be a non-empty string`);
  }
  return value.trim();
}

function validatePositiveInteger(name, value) {
  if (!Number.isInteger(value) || value <= 0) {
    throw new Error(`${name} must be a positive integer`);
  }
  return value;
}

function validateNonNegativeInteger(name, value) {
  if (!Number.isInteger(value) || value < 0) {
    throw new Error(`${name} must be a non-negative integer`);
  }
  return value;
}

function validateInsertIndex(value, length) {
  if (!Number.isInteger(value) || value < 0 || value > length) {
    throw new Error(`insert index ${value} is outside visible text bounds`);
  }
  return value;
}

function validateDeleteIndex(value, length, count) {
  if (!Number.isInteger(value) || value < 0 || value + count > length) {
    throw new Error(`delete range ${value}:${count} is outside visible text bounds`);
  }
  return value;
}

function cloneVisibleElements(visibleElements) {
  return visibleElements.map((element) => ({
    id: validateIdentifier("visible element id", element?.id),
    value: typeof element?.value === "string" ? element.value : "",
  }));
}
