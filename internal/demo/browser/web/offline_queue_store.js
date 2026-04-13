const DB_NAME = "syncraft-offline-queue";
const DB_VERSION = 1;
const STORE_NAME = "queued_operations";
const METADATA_STORE_NAME = "queue_metadata";
const STATUS_PENDING = "pending";
const STATUS_REPLAYED = "replayed";
const STATUS_BLOCKED = "blocked";
const VALID_STATUSES = new Set([STATUS_PENDING, STATUS_REPLAYED, STATUS_BLOCKED]);

/**
 * OfflineQueueStore persists canonical CRDT operations in browser-local IndexedDB storage.
 */
export class OfflineQueueStore {
  constructor(indexedDBFactory = globalThis.indexedDB) {
    if (!indexedDBFactory || typeof indexedDBFactory.open !== "function") {
      throw new Error("indexedDB is not available in this environment");
    }
    this.indexedDBFactory = indexedDBFactory;
    this.dbPromise = null;
  }

  /**
   * open ensures the IndexedDB database and object store exist.
   */
  async open() {
    if (!this.dbPromise) {
      this.dbPromise = this.#openDatabase();
    }
    return this.dbPromise;
  }

  /**
   * loadPending returns all pending queued operations for one document and actor.
   */
  async loadPending(documentID, actorID) {
    validateIdentifier("document_id", documentID);
    validateIdentifier("actor_id", actorID);
    const db = await this.open();
    return this.#runReadonly(db, (transaction, store, resolve, reject) => {
      const request = store.index("document_actor_status").getAll([documentID, actorID, STATUS_PENDING]);
      request.onsuccess = () => {
        const records = (request.result || []).map(cloneRecord).sort(byActorCounterThenOperationID);
        resolve(records);
      };
      request.onerror = () => {
        reject(normalizeRequestError("load pending queue records", request.error));
      };
    });
  }

  /**
   * loadMetadata returns persisted actor-counter and queue metadata for one document and actor pair.
   */
  async loadMetadata(documentID, actorID) {
    validateIdentifier("document_id", documentID);
    validateIdentifier("actor_id", actorID);
    const db = await this.open();
    return this.#runReadonly(db, (transaction, store, resolve, reject) => {
      const metadataStore = transaction.objectStore(METADATA_STORE_NAME);
      const request = metadataStore.get(buildMetadataKey(documentID, actorID));
      request.onsuccess = () => {
        resolve(request.result ? cloneRecord(request.result) : defaultMetadataRecord(documentID, actorID));
      };
      request.onerror = () => reject(normalizeRequestError("load queue metadata", request.error));
    });
  }

  /**
   * saveMetadata validates and persists browser-local actor-counter and queue metadata.
   */
  async saveMetadata(metadata) {
    const normalized = normalizeMetadata(metadata);
    const db = await this.open();
    return this.#runReadwrite(db, (transaction, store, resolve, reject) => {
      const metadataStore = transaction.objectStore(METADATA_STORE_NAME);
      const request = metadataStore.put(normalized);
      request.onsuccess = () => resolve(cloneRecord(normalized));
      request.onerror = () => reject(normalizeRequestError("save queue metadata", request.error));
    });
  }

  /**
   * append validates and persists one queued canonical operation record.
   */
  async append(record) {
    const normalized = normalizeRecord(record);
    const db = await this.open();
    return this.#runReadwrite(db, (transaction, store, resolve, reject) => {
      const request = store.put(normalized);
      request.onsuccess = () => resolve(cloneRecord(normalized));
      request.onerror = () => reject(normalizeRequestError("append queued operation", request.error));
    });
  }

  /**
   * markReplayed marks one queued operation as replayed after successful submission.
   */
  async markReplayed(documentID, actorID, operationID) {
    return this.#updateStatus(documentID, actorID, operationID, STATUS_REPLAYED);
  }

  /**
   * markBlocked marks one queued operation as blocked with a human-readable reason.
   */
  async markBlocked(documentID, actorID, operationID, reason) {
    if (typeof reason !== "string" || reason.trim() === "") {
      throw new Error("blocked queue records require a non-empty reason");
    }
    return this.#updateStatus(documentID, actorID, operationID, STATUS_BLOCKED, reason.trim());
  }

  /**
   * clearReplayed removes replayed records for one document and actor pair.
   */
  async clearReplayed(documentID, actorID) {
    validateIdentifier("document_id", documentID);
    validateIdentifier("actor_id", actorID);
    const db = await this.open();
    return this.#runReadwrite(db, (transaction, store, resolve, reject) => {
      const request = store.index("document_actor_status").openCursor([documentID, actorID, STATUS_REPLAYED]);
      let removed = 0;
      request.onsuccess = () => {
        const cursor = request.result;
        if (!cursor) {
          resolve(removed);
          return;
        }
        const deleteRequest = cursor.delete();
        deleteRequest.onsuccess = () => {
          removed += 1;
          cursor.continue();
        };
        deleteRequest.onerror = () => reject(normalizeRequestError("clear replayed queue records", deleteRequest.error));
      };
      request.onerror = () => reject(normalizeRequestError("scan replayed queue records", request.error));
    });
  }

  async #updateStatus(documentID, actorID, operationID, status, failureReason = "") {
    validateIdentifier("document_id", documentID);
    validateIdentifier("actor_id", actorID);
    validateIdentifier("operation_id", operationID);
    const db = await this.open();
    return this.#runReadwrite(db, (transaction, store, resolve, reject) => {
      const key = buildRecordKey(documentID, actorID, operationID);
      const request = store.get(key);
      request.onsuccess = () => {
        const record = request.result;
        if (!record) {
          reject(new Error(`queued operation ${operationID} was not found`));
          return;
        }
        const nextRecord = {
          ...record,
          status,
          failure_reason: failureReason,
          document_actor_status: [record.document_id, record.actor_id, status],
        };
        const putRequest = store.put(nextRecord);
        putRequest.onsuccess = () => resolve(cloneRecord(nextRecord));
        putRequest.onerror = () => reject(normalizeRequestError("update queued operation status", putRequest.error));
      };
      request.onerror = () => reject(normalizeRequestError("read queued operation", request.error));
    });
  }

  #runReadonly(db, executor) {
    return new Promise((resolve, reject) => {
      const transaction = db.transaction([STORE_NAME, METADATA_STORE_NAME], "readonly");
      transaction.onabort = () => reject(normalizeTransactionError("queue readonly transaction", transaction.error));
      transaction.onerror = () => reject(normalizeTransactionError("queue readonly transaction", transaction.error));
      executor(transaction, transaction.objectStore(STORE_NAME), resolve, reject);
    });
  }

  #runReadwrite(db, executor) {
    return new Promise((resolve, reject) => {
      const transaction = db.transaction([STORE_NAME, METADATA_STORE_NAME], "readwrite");
      transaction.onabort = () => reject(normalizeTransactionError("queue readwrite transaction", transaction.error));
      transaction.onerror = () => reject(normalizeTransactionError("queue readwrite transaction", transaction.error));
      executor(transaction, transaction.objectStore(STORE_NAME), resolve, reject);
    });
  }

  #openDatabase() {
    return new Promise((resolve, reject) => {
      const request = this.indexedDBFactory.open(DB_NAME, DB_VERSION);
      request.onupgradeneeded = () => {
        const db = request.result;
        if (db.objectStoreNames.contains(STORE_NAME)) {
          return;
        }
        const store = db.createObjectStore(STORE_NAME, { keyPath: "record_key" });
        store.createIndex("document_actor_status", "document_actor_status", { unique: false });
        store.createIndex("document_actor", "document_actor", { unique: false });
        db.createObjectStore(METADATA_STORE_NAME, { keyPath: "metadata_key" });
      };
      request.onsuccess = () => resolve(request.result);
      request.onerror = () => reject(normalizeRequestError("open offline queue database", request.error));
    });
  }
}

/**
 * createOfflineQueueStore builds one browser-local IndexedDB queue store.
 */
export function createOfflineQueueStore(indexedDBFactory = globalThis.indexedDB) {
  return new OfflineQueueStore(indexedDBFactory);
}

function normalizeRecord(record) {
  if (!record || typeof record !== "object") {
    throw new Error("queued operation record must be an object");
  }
  const documentID = validateIdentifier("document_id", record.document_id);
  const actorID = validateIdentifier("actor_id", record.actor_id);
  const operationID = validateIdentifier("operation_id", record.operation_id);
  if (!Number.isInteger(record.actor_counter) || record.actor_counter <= 0) {
    throw new Error("actor_counter must be a positive integer");
  }
  if (!record.operation || typeof record.operation !== "object") {
    throw new Error("queued operation record must include an operation object");
  }
  if (record.operation.operation_id !== operationID) {
    throw new Error("operation.operation_id must match operation_id");
  }
  if (record.operation.actor_id !== actorID) {
    throw new Error("operation.actor_id must match actor_id");
  }
  if (record.operation.actor_counter !== record.actor_counter) {
    throw new Error("operation.actor_counter must match actor_counter");
  }
  const status = record.status || STATUS_PENDING;
  if (!VALID_STATUSES.has(status)) {
    throw new Error(`unsupported queued operation status ${String(status)}`);
  }
  const queuedAt = typeof record.queued_at === "string" && record.queued_at.trim() !== ""
    ? record.queued_at
    : new Date().toISOString();
  const failureReason = typeof record.failure_reason === "string" ? record.failure_reason : "";
  return {
    record_key: buildRecordKey(documentID, actorID, operationID),
    document_id: documentID,
    actor_id: actorID,
    operation_id: operationID,
    actor_counter: record.actor_counter,
    operation: cloneRecord(record.operation),
    queued_at: queuedAt,
    status,
    failure_reason: failureReason,
    document_actor_status: [documentID, actorID, status],
    document_actor: [documentID, actorID],
  };
}

function validateIdentifier(name, value) {
  if (typeof value !== "string" || value.trim() === "") {
    throw new Error(`${name} must be a non-empty string`);
  }
  return value.trim();
}

function buildRecordKey(documentID, actorID, operationID) {
  return `${documentID}::${actorID}::${operationID}`;
}

function buildMetadataKey(documentID, actorID) {
  return `${documentID}::${actorID}`;
}

function defaultMetadataRecord(documentID, actorID) {
  return {
    metadata_key: buildMetadataKey(documentID, actorID),
    document_id: documentID,
    actor_id: actorID,
    next_actor_counter: 1,
    pending_queue_count: 0,
    last_snapshot_id: null,
    last_operation_id: null,
  };
}

function normalizeMetadata(metadata) {
  if (!metadata || typeof metadata !== "object") {
    throw new Error("queue metadata must be an object");
  }
  const documentID = validateIdentifier("document_id", metadata.document_id);
  const actorID = validateIdentifier("actor_id", metadata.actor_id);
  if (!Number.isInteger(metadata.next_actor_counter) || metadata.next_actor_counter <= 0) {
    throw new Error("next_actor_counter must be a positive integer");
  }
  const pendingQueueCount = metadata.pending_queue_count ?? 0;
  if (!Number.isInteger(pendingQueueCount) || pendingQueueCount < 0) {
    throw new Error("pending_queue_count must be a non-negative integer");
  }
  return {
    metadata_key: buildMetadataKey(documentID, actorID),
    document_id: documentID,
    actor_id: actorID,
    next_actor_counter: metadata.next_actor_counter,
    pending_queue_count: pendingQueueCount,
    last_snapshot_id: metadata.last_snapshot_id || null,
    last_operation_id: metadata.last_operation_id || null,
  };
}

function byActorCounterThenOperationID(left, right) {
  if (left.actor_counter !== right.actor_counter) {
    return left.actor_counter - right.actor_counter;
  }
  return left.operation_id.localeCompare(right.operation_id);
}

function cloneRecord(value) {
  return JSON.parse(JSON.stringify(value));
}

function normalizeRequestError(action, error) {
  return new Error(`${action}: ${error?.message || "unknown IndexedDB request failure"}`);
}

function normalizeTransactionError(action, error) {
  return new Error(`${action}: ${error?.message || "unknown IndexedDB transaction failure"}`);
}
