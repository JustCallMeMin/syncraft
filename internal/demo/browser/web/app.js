import { createOfflineQueueStore } from "/offline_queue_store.js";
import {
  isCurrentGeneration,
  shouldEnableEditor,
} from "/browser_session_policy.js";
import {
  loadSessionIntent,
  saveSessionIntent,
} from "/browser_session_storage.js";
import { deriveQueueSurfaceState } from "/queue_ui_state.js";
import {
  createDeleteOperations,
  createInsertOperations,
} from "/canonical_queue_ops.js";
import { computeTextDiff } from "/browser_text_diff.js";
import { resolveEditorInputMode } from "/browser_input_policy.js";
import { createBrowserStatusState } from "/browser_status_state.js";

const form = document.getElementById("connect-form");
const actorInput = document.getElementById("actor-id");
const documentInput = document.getElementById("document-id");
const reconnectButton = document.getElementById("reconnect-button");
const editor = document.getElementById("editor");
const statusNode = document.getElementById("status");
const queueSummaryNode = document.getElementById("queue-summary");
const errorNode = document.getElementById("error");

let socket = null;
let applyingRemoteState = false;
let lastText = "";
let reconnectTimer = null;
let sessionIntent = null;
let queueStore = null;
let pendingQueueRecords = [];
let queueMetadata = null;
let queueBlockedReason = "";
let reconnectingWithQueue = false;
let localVisibleElements = [];
let replayInFlight = false;
let stateWaiters = [];
let connectionGeneration = 0;
let isComposingText = false;
let liveSubmitInFlight = false;
let pendingEditorText = null;
let sessionReady = false;
const browserStatusState = createBrowserStatusState();

const restoredSessionIntent = loadSessionIntent();
if (restoredSessionIntent) {
  actorInput.value = restoredSessionIntent.actorID;
  documentInput.value = restoredSessionIntent.documentID;
} else {
  actorInput.value = actorInput.value || `actor-${Math.random().toString(36).slice(2, 8)}`;
}

initializeQueueStore();
if (restoredSessionIntent) {
  connect(restoredSessionIntent, {
    preserveSessionState: false,
  }).catch((error) => {
    setStatus("error", error.message);
    updateEditorDisabled("error");
  });
}

form.addEventListener("submit", async (event) => {
  event.preventDefault();
  const intent = {
    actorID: actorInput.value.trim(),
    documentID: documentInput.value.trim(),
    retry: true,
  };
  saveSessionIntent(intent);
  await connect(intent, {
    preserveSessionState: false,
  });
});

reconnectButton.addEventListener("click", async () => {
  if (!sessionIntent) {
    setStatus("error", "connect once before trying to reconnect");
    return;
  }
  await connect(sessionIntent, {
    preserveSessionState: false,
  });
});

editor.addEventListener("input", async () => {
  await processEditorChange();
});

editor.addEventListener("compositionstart", () => {
  isComposingText = true;
});

editor.addEventListener("compositionend", async () => {
  isComposingText = false;
  await processEditorChange();
});

async function processEditorChange() {
  const nextText = editor.value;
  const mode = resolveEditorInputMode({
    applyingRemoteState,
    isComposingText,
    hasSessionIntent: Boolean(sessionIntent),
    isSocketOpen: Boolean(socket && socket.readyState === WebSocket.OPEN),
    sessionReady,
    connectionState: browserStatusState.getConnectionState(),
  });
  if (mode === "ignore") {
    return;
  }
  if (mode === "buffer") {
    pendingEditorText = nextText;
    return;
  }

  const diff = computeTextDiff(lastText, nextText);
  if (!diff) {
    lastText = nextText;
    return;
  }

  if (mode === "live") {
    if (liveSubmitInFlight) {
      pendingEditorText = nextText;
      return;
    }
    await submitLiveEditorText(nextText);
    return;
  }

  await queueOfflineDiff(diff, nextText);
}

async function connect(intent, options = {}) {
  connectionGeneration += 1;
  const eventGeneration = connectionGeneration;
  const preserveSessionState = Boolean(options.preserveSessionState);
  sessionIntent = intent;
  window.clearTimeout(reconnectTimer);
  if (!preserveSessionState) {
    resetSessionState(intent);
  }

  if (!(await loadPendingQueue(intent))) {
    return;
  }
  if (!(await loadQueueMetadata(intent))) {
    return;
  }
  reconnectingWithQueue = pendingQueueRecords.length > 0;

  if (socket && socket.readyState === WebSocket.OPEN) {
    socket.close();
  }

  const protocol = window.location.protocol === "https:" ? "wss" : "ws";
  socket = new WebSocket(`${protocol}://${window.location.host}/ws`);
  if (!preserveSessionState) {
    setStatus("connecting", "");
    editor.disabled = true;
    sessionReady = false;
  } else {
    updateEditorDisabled(browserStatusState.getConnectionState());
  }
  reconnectButton.disabled = true;

  socket.addEventListener("open", () => {
    if (!isCurrentGeneration(connectionGeneration, eventGeneration)) {
      return;
    }
    socket.send(JSON.stringify({
      type: "init",
      actor_id: intent.actorID,
      document_id: intent.documentID,
      next_actor_counter: queueMetadata?.next_actor_counter || 1,
    }));
  });

  socket.addEventListener("message", (event) => {
    if (!isCurrentGeneration(connectionGeneration, eventGeneration)) {
      return;
    }
    const message = JSON.parse(event.data);
    if (message.type !== "state") {
      return;
    }
    if (message.actor_id && message.actor_id !== sessionIntent?.actorID) {
      return;
    }
    if (message.document_id && message.document_id !== sessionIntent?.documentID) {
      return;
    }
    sessionReady = true;
    resolveStateWaiters(message);
    applyingRemoteState = true;
    editor.value = message.text || "";
    lastText = editor.value;
    localVisibleElements = Array.isArray(message.visible_elements) ? message.visible_elements : [];
    applyingRemoteState = false;
    setStatus(message.connection_state || "live", message.last_error || "");
    updateEditorDisabled(message.connection_state || "live");
    reconnectButton.disabled = false;
    updateMetadataFromState(message).catch((error) => {
      setStatus("error", `offline metadata save failed: ${error.message}`);
      updateEditorDisabled("error");
    });
    if (message.connection_state === "live" && reconnectingWithQueue && pendingQueueRecords.length > 0 && !replayInFlight) {
      replayPendingQueue().catch((error) => {
        setQueueBlocked(`offline replay failed: ${error.message}`);
      });
      return;
    }
    if (!replayInFlight && pendingEditorText !== null && pendingEditorText !== editor.value) {
      const nextPendingEditorText = pendingEditorText;
      pendingEditorText = null;
      applyingRemoteState = true;
      editor.value = nextPendingEditorText;
      applyingRemoteState = false;
      processEditorChange().catch((error) => {
        setStatus("error", error.message);
        updateEditorDisabled("error");
      });
      return;
    }
    if (!replayInFlight && pendingQueueRecords.length === 0) {
      reconnectingWithQueue = false;
    }
  });

  socket.addEventListener("close", () => {
    if (!isCurrentGeneration(connectionGeneration, eventGeneration)) {
      return;
    }
    setStatus("disconnected", "");
    updateEditorDisabled("disconnected");
    reconnectButton.disabled = false;
    if (intent.retry) {
      reconnectTimer = window.setTimeout(() => {
        if (!isCurrentGeneration(connectionGeneration, eventGeneration)) {
          return;
        }
        connect(sessionIntent, {
          preserveSessionState: true,
        }).catch((error) => {
          setStatus("disconnected", error.message);
          updateEditorDisabled("disconnected");
        });
      }, 750);
    }
  });

  socket.addEventListener("error", () => {
    if (!isCurrentGeneration(connectionGeneration, eventGeneration)) {
      return;
    }
    if (preserveSessionState) {
      setStatus("disconnected", "");
      updateEditorDisabled("disconnected");
      reconnectButton.disabled = false;
      return;
    }
    setStatus("error", "websocket connection failed");
    updateEditorDisabled("error");
    reconnectButton.disabled = false;
  });
}

async function initializeQueueStore() {
  try {
    queueStore = createOfflineQueueStore();
    await queueStore.open();
    window.syncraftOfflineQueueStore = queueStore;
    queueBlockedReason = "";
    setStatus(browserStatusState.getConnectionState(), browserStatusState.getErrorText());
    updateEditorDisabled(browserStatusState.getConnectionState());
  } catch (error) {
    queueStore = null;
    queueBlockedReason = `offline queue store init failed: ${error.message}`;
    setStatus("error", queueBlockedReason);
    updateEditorDisabled("error");
  }
}

async function loadPendingQueue(intent) {
  if (!queueStore) {
    await initializeQueueStore();
  }
  if (!queueStore) {
    return false;
  }
  try {
    pendingQueueRecords = await queueStore.loadPending(intent.documentID, intent.actorID);
    window.syncraftPendingQueueRecords = pendingQueueRecords;
    queueBlockedReason = "";
    setStatus(browserStatusState.getConnectionState(), browserStatusState.getErrorText());
    updateEditorDisabled(browserStatusState.getConnectionState());
    return true;
  } catch (error) {
    queueBlockedReason = `offline queue load failed: ${error.message}`;
    setStatus("error", queueBlockedReason);
    updateEditorDisabled("error");
    reconnectButton.disabled = false;
    return false;
  }
}

async function loadQueueMetadata(intent) {
  if (!queueStore) {
    return false;
  }
  try {
    queueMetadata = await queueStore.loadMetadata(intent.documentID, intent.actorID);
    queueMetadata.pending_queue_count = pendingQueueRecords.length;
    queueMetadata = await queueStore.saveMetadata(queueMetadata);
    window.syncraftQueueMetadata = queueMetadata;
    queueBlockedReason = "";
    setStatus(browserStatusState.getConnectionState(), browserStatusState.getErrorText());
    updateEditorDisabled(browserStatusState.getConnectionState());
    return true;
  } catch (error) {
    queueBlockedReason = `offline metadata load failed: ${error.message}`;
    setStatus("error", queueBlockedReason);
    updateEditorDisabled("error");
    reconnectButton.disabled = false;
    return false;
  }
}

async function allocateActorCounter() {
  return allocateActorCounters(1);
}

async function allocateActorCounters(count) {
  if (!queueMetadata || !queueStore) {
    queueBlockedReason = "offline metadata is not initialized";
    setStatus("error", queueBlockedReason);
    return null;
  }
  const startCounter = queueMetadata.next_actor_counter;
  queueMetadata = {
    ...queueMetadata,
    next_actor_counter: startCounter + count,
  };
  queueMetadata = await queueStore.saveMetadata(queueMetadata);
  window.syncraftQueueMetadata = queueMetadata;
  queueBlockedReason = "";
  setStatus(browserStatusState.getConnectionState(), browserStatusState.getErrorText());
  return startCounter;
}

async function submitLiveDiff(diff) {
  if (!sessionIntent || !socket || socket.readyState !== WebSocket.OPEN) {
    revertEditorToLastText();
    return false;
  }

  let workingVisibleElements = localVisibleElements;
  const queuedOperations = [];
  const removedCount = Array.from(diff.removed).length;

  if (removedCount > 0) {
    const actorCounterStart = await allocateActorCounters(removedCount);
    if (actorCounterStart === null) {
      revertEditorToLastText();
      return false;
    }
    const deleteResult = createDeleteOperations({
      documentID: sessionIntent.documentID,
      actorID: sessionIntent.actorID,
      startCounter: actorCounterStart,
      index: diff.start,
      count: removedCount,
      visibleElements: workingVisibleElements,
    });
    queuedOperations.push(...deleteResult.operations);
    workingVisibleElements = deleteResult.visibleElements;
  }

  if (diff.inserted.length > 0) {
    const actorCounterStart = await allocateActorCounters(Array.from(diff.inserted).length);
    if (actorCounterStart === null) {
      revertEditorToLastText();
      return false;
    }
    const insertResult = createInsertOperations({
      documentID: sessionIntent.documentID,
      actorID: sessionIntent.actorID,
      startCounter: actorCounterStart,
      index: diff.start,
      value: diff.inserted,
      visibleElements: workingVisibleElements,
    });
    queuedOperations.push(...insertResult.operations);
    workingVisibleElements = insertResult.visibleElements;
  }

  localVisibleElements = workingVisibleElements;
  for (const operation of queuedOperations) {
    socket.send(JSON.stringify({
      type: "submit_operation",
      operation,
    }));
  }
  return true;
}

/**
 * submitLiveEditorText serializes live browser input so unicode strings cannot race local state.
 */
async function submitLiveEditorText(nextText) {
  liveSubmitInFlight = true;
  try {
    const diff = computeTextDiff(lastText, nextText);
    if (!diff) {
      lastText = nextText;
      return;
    }

    const submitted = await submitLiveDiff(diff);
    if (!submitted) {
      return;
    }
    lastText = nextText;
  } finally {
    liveSubmitInFlight = false;
  }

  if (pendingEditorText === null) {
    return;
  }

  const pendingText = pendingEditorText;
  pendingEditorText = null;
  if (pendingText !== editor.value) {
    pendingEditorText = editor.value;
  }
  await processEditorChange();
}

async function queueOfflineDiff(diff, nextText) {
  if (!sessionIntent) {
    revertEditorToLastText();
    setStatus("error", "connect once before queueing offline edits");
    return;
  }
  if (!queueStore || !queueMetadata) {
    revertEditorToLastText();
    setQueueBlocked("offline queue metadata is not initialized");
    return;
  }

  try {
    let workingVisibleElements = localVisibleElements;
    let nextCounter = queueMetadata.next_actor_counter;
    const queuedOperations = [];

    if (diff.removed.length > 0) {
      const deleteResult = createDeleteOperations({
        documentID: sessionIntent.documentID,
        actorID: sessionIntent.actorID,
        startCounter: nextCounter,
        index: diff.start,
        count: Array.from(diff.removed).length,
        visibleElements: workingVisibleElements,
      });
      queuedOperations.push(...deleteResult.operations);
      workingVisibleElements = deleteResult.visibleElements;
      nextCounter = deleteResult.nextCounter;
    }

    if (diff.inserted.length > 0) {
      const insertResult = createInsertOperations({
        documentID: sessionIntent.documentID,
        actorID: sessionIntent.actorID,
        startCounter: nextCounter,
        index: diff.start,
        value: diff.inserted,
        visibleElements: workingVisibleElements,
      });
      queuedOperations.push(...insertResult.operations);
      workingVisibleElements = insertResult.visibleElements;
      nextCounter = insertResult.nextCounter;
    }

    for (const operation of queuedOperations) {
      await queueStore.append({
        document_id: operation.document_id,
        actor_id: operation.actor_id,
        operation_id: operation.operation_id,
        actor_counter: operation.actor_counter,
        operation,
      });
    }

    pendingQueueRecords = await queueStore.loadPending(sessionIntent.documentID, sessionIntent.actorID);
    window.syncraftPendingQueueRecords = pendingQueueRecords;
    localVisibleElements = workingVisibleElements;
    queueMetadata = await queueStore.saveMetadata({
      ...queueMetadata,
      document_id: sessionIntent.documentID,
      actor_id: sessionIntent.actorID,
      next_actor_counter: nextCounter,
      pending_queue_count: pendingQueueRecords.length,
    });
    window.syncraftQueueMetadata = queueMetadata;
    lastText = nextText;
    reconnectingWithQueue = false;
    queueBlockedReason = "";
    setStatus("disconnected", "");
    updateEditorDisabled("disconnected");
  } catch (error) {
    revertEditorToLastText();
    setQueueBlocked(`offline queue append failed: ${error.message}`);
  }
}

async function replayPendingQueue() {
  if (!socket || socket.readyState !== WebSocket.OPEN) {
    return;
  }
  if (!sessionIntent || !queueStore || pendingQueueRecords.length === 0) {
    reconnectingWithQueue = false;
    return;
  }

  replayInFlight = true;
  updateEditorDisabled("replaying_queue");
  reconnectButton.disabled = true;
  setStatus("live", "");

  try {
    for (const record of [...pendingQueueRecords]) {
      socket.send(JSON.stringify({
        type: "submit_operation",
        operation: record.operation,
      }));
      await waitForNextState(2000);
      await queueStore.markReplayed(record.document_id, record.actor_id, record.operation_id);
      pendingQueueRecords = pendingQueueRecords.filter((entry) => entry.operation_id !== record.operation_id);
      window.syncraftPendingQueueRecords = pendingQueueRecords;
      queueMetadata = await queueStore.saveMetadata({
        ...queueMetadata,
        document_id: sessionIntent.documentID,
        actor_id: sessionIntent.actorID,
        pending_queue_count: pendingQueueRecords.length,
      });
      window.syncraftQueueMetadata = queueMetadata;
    }

    await queueStore.clearReplayed(sessionIntent.documentID, sessionIntent.actorID);
    reconnectingWithQueue = false;
    queueBlockedReason = "";
    setStatus("live", "");
    updateEditorDisabled("live");
  } catch (error) {
    if (pendingQueueRecords.length > 0) {
      const record = pendingQueueRecords[0];
      await queueStore.markBlocked(record.document_id, record.actor_id, record.operation_id, error.message);
    }
    setQueueBlocked(error.message);
  } finally {
    replayInFlight = false;
    reconnectButton.disabled = false;
    updateEditorDisabled(browserStatusState.getConnectionState());
  }
}

async function updateMetadataFromState(message) {
  if (!queueStore || !queueMetadata || !sessionIntent) {
    return;
  }
  queueMetadata = await queueStore.saveMetadata({
    document_id: sessionIntent.documentID,
    actor_id: sessionIntent.actorID,
    next_actor_counter: message.next_actor_counter || queueMetadata.next_actor_counter || 1,
    pending_queue_count: Array.isArray(window.syncraftPendingQueueRecords) ? window.syncraftPendingQueueRecords.length : pendingQueueRecords.length,
    last_snapshot_id: message.last_snapshot_id || null,
    last_operation_id: message.last_operation_id || null,
  });
  window.syncraftQueueMetadata = queueMetadata;
  setStatus(message.connection_state || "live", message.last_error || "");
  updateEditorDisabled(message.connection_state || "live");
}

function waitForNextState(timeoutMs) {
  return new Promise((resolve, reject) => {
    const timer = window.setTimeout(() => {
      stateWaiters = stateWaiters.filter((waiter) => waiter.reject !== reject);
      reject(new Error(`timed out waiting for replay state after ${timeoutMs}ms`));
    }, timeoutMs);
    stateWaiters.push({
      resolve(message) {
        window.clearTimeout(timer);
        resolve(message);
      },
      reject,
    });
  });
}

function resolveStateWaiters(message) {
  if (stateWaiters.length === 0) {
    return;
  }
  const waiters = stateWaiters;
  stateWaiters = [];
  for (const waiter of waiters) {
    waiter.resolve(message);
  }
}

function revertEditorToLastText() {
  applyingRemoteState = true;
  editor.value = lastText;
  applyingRemoteState = false;
}

function setQueueBlocked(reason) {
  queueBlockedReason = reason;
  reconnectingWithQueue = false;
  setStatus("error", reason);
  updateEditorDisabled("error");
  reconnectButton.disabled = false;
}

function resetSessionState(intent) {
  pendingQueueRecords = [];
  queueMetadata = null;
  queueBlockedReason = "";
  reconnectingWithQueue = false;
  replayInFlight = false;
  localVisibleElements = [];
  sessionReady = false;
  pendingEditorText = null;
  stateWaiters = [];
  lastText = "";
  window.syncraftPendingQueueRecords = pendingQueueRecords;
  window.syncraftQueueMetadata = queueMetadata;
  browserStatusState.reset();
  editor.value = "";
  statusNode.textContent = "disconnected";
  queueSummaryNode.textContent = "no queued local operations";
  errorNode.textContent = "none";
  if (intent) {
    actorInput.value = intent.actorID;
    documentInput.value = intent.documentID;
  }
}

function updateEditorDisabled(connectionState) {
  editor.disabled = !shouldEnableEditor({
    hasSessionIntent: Boolean(sessionIntent),
    replayInFlight,
    blockedReason: queueBlockedReason,
    hasQueueStore: Boolean(queueStore),
    hasQueueMetadata: Boolean(queueMetadata),
    connectionState,
  });
}

function setStatus(state, errorText) {
  const nextStatus = browserStatusState.set(state, errorText);
  const surface = deriveQueueSurfaceState({
    connectionState: nextStatus.connectionState,
    pendingQueueCount: queueMetadata?.pending_queue_count ?? pendingQueueRecords.length,
    blockedReason: queueBlockedReason,
    reconnectingWithQueue,
  });
  statusNode.textContent = surface.state;
  statusNode.className = `status-${surface.state}`;
  queueSummaryNode.textContent = surface.queueSummary;
  errorNode.textContent = nextStatus.errorText;
}

