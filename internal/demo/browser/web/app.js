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
import { createDebugPanelState } from "/browser_debug_panel_state.js";
import { deriveDebugPanelDiagnostics } from "/browser_debug_panel_diagnostics.js";
import { createDebugEventBuffer } from "/browser_debug_event_buffer.js";
import { deriveSaveState } from "/browser_save_state.js";
import { colorTokenForActor } from "/browser_collaborator_palette.js";
import {
  anchorForRuneIndex,
  codeUnitIndexFromRuneIndex,
  runeIndexFromAnchor,
  runeIndexFromCodeUnitIndex,
} from "/browser_presence_mapping.js";

const form = document.getElementById("connect-form");
const actorInput = document.getElementById("actor-id");
const documentInput = document.getElementById("document-id");
const titleInput = document.getElementById("document-title");
const saveStateChipNode = document.getElementById("save-state-chip");
const collaboratorStripNode = document.getElementById("collaborator-strip");
const reconnectButton = document.getElementById("reconnect-button");
const debugPanelToggleButton = document.getElementById("debug-panel-toggle");
const debugPanel = document.getElementById("debug-panel");
const presenceLayer = document.getElementById("presence-layer");
const debugActorInstanceNode = document.getElementById("debug-actor-instance");
const debugDocumentIDNode = document.getElementById("debug-document-id");
const debugConnectionStateNode = document.getElementById("debug-connection-state");
const debugLastErrorNode = document.getElementById("debug-last-error");
const debugQueueStateNode = document.getElementById("debug-queue-state");
const debugPendingQueueCountNode = document.getElementById("debug-pending-queue-count");
const debugQueueSummaryNode = document.getElementById("debug-queue-summary");
const debugLastSnapshotIDNode = document.getElementById("debug-last-snapshot-id");
const debugLastOperationIDNode = document.getElementById("debug-last-operation-id");
const debugReplayStateNode = document.getElementById("debug-replay-state");
const debugReplayInFlightNode = document.getElementById("debug-replay-in-flight");
const debugBlockedReasonNode = document.getElementById("debug-blocked-reason");
const debugEventsListNode = document.getElementById("debug-events-list");
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
let titleUpdateInFlight = false;
let titleDirty = false;
let titleSyncTimer = null;
let persistedDocumentTitle = "Untitled document";
let collaborators = [];
let presenceUpdateTimer = null;
const browserStatusState = createBrowserStatusState();
const debugPanelState = createDebugPanelState(false);
const debugEventBuffer = createDebugEventBuffer(50);

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
  emitDebugEvent("connect_requested", "info", "Connect requested from the browser shell.", {
    retry: intent.retry,
  });
  await connect(intent, {
    preserveSessionState: false,
  });
});

reconnectButton.addEventListener("click", async () => {
  if (!sessionIntent) {
    setStatus("error", "connect once before trying to reconnect");
    return;
  }
  emitDebugEvent("reconnect_requested", "warning", "Manual reconnect requested from the browser shell.", {
    retry: sessionIntent.retry,
  });
  await connect(sessionIntent, {
    preserveSessionState: false,
  });
});

debugPanelToggleButton.addEventListener("click", () => {
  renderDebugPanel(debugPanelState.toggle());
});

titleInput.addEventListener("input", () => {
  titleDirty = true;
  updateSaveStateChip();
  scheduleTitleSync(350);
});

titleInput.addEventListener("blur", async () => {
  await flushTitleSync();
});

editor.addEventListener("input", async () => {
  await processEditorChange();
  schedulePresenceUpdate(90);
});

editor.addEventListener("compositionstart", () => {
  isComposingText = true;
});

editor.addEventListener("compositionend", async () => {
  isComposingText = false;
  await processEditorChange();
  schedulePresenceUpdate(90);
});

editor.addEventListener("click", () => {
  schedulePresenceUpdate(50);
});

editor.addEventListener("keyup", () => {
  schedulePresenceUpdate(50);
});

editor.addEventListener("focus", () => {
  schedulePresenceUpdate(20);
});

document.addEventListener("selectionchange", () => {
  if (document.activeElement !== editor) {
    return;
  }
  schedulePresenceUpdate(50);
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
    collaborators = [];
    renderCollaborators();
    renderPresenceLayer();
  } else {
    updateEditorDisabled(browserStatusState.getConnectionState());
  }
  reconnectButton.disabled = true;

  socket.addEventListener("open", () => {
    if (!isCurrentGeneration(connectionGeneration, eventGeneration)) {
      return;
    }
    emitDebugEvent("socket_opened", "info", "Socket opened for the active actor instance.", {
      preserve_session_state: preserveSessionState,
    });
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
    emitDebugEvent("state_ready", "info", "State ready received for the active session.", {
      next_actor_counter: message.next_actor_counter || queueMetadata?.next_actor_counter || 1,
    });
    sessionReady = true;
    resolveStateWaiters(message);
    applyingRemoteState = true;
    editor.value = message.text || "";
    lastText = editor.value;
    localVisibleElements = Array.isArray(message.visible_elements) ? message.visible_elements : [];
    collaborators = Array.isArray(message.collaborators) ? message.collaborators : [];
    if (typeof message.title === "string" && message.title.trim() !== "") {
      persistedDocumentTitle = message.title.trim();
      if (!titleDirty || titleInput.value.trim() === persistedDocumentTitle) {
        titleInput.value = persistedDocumentTitle;
        titleDirty = false;
      }
      titleUpdateInFlight = false;
    }
    applyingRemoteState = false;
    setStatus(message.connection_state || "live", message.last_error || "");
    updateEditorDisabled(message.connection_state || "live");
    reconnectButton.disabled = false;
    renderCollaborators();
    renderPresenceLayer();
    updateMetadataFromState(message).catch((error) => {
      setStatus("error", `offline metadata save failed: ${error.message}`);
      updateEditorDisabled("error");
    });
    schedulePresenceUpdate(40);
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
    emitDebugEvent("disconnect_detected", "warning", "Socket closed for the active session.", {
      retry: intent.retry,
    });
    setStatus("disconnected", "");
    updateEditorDisabled("disconnected");
    reconnectButton.disabled = false;
    renderCollaborators();
    renderPresenceLayer();
    if (intent.retry) {
      emitDebugEvent("reconnect_scheduled", "warning", "Reconnect scheduled after disconnect.", {
        retry_delay_ms: 750,
      });
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
      renderCollaborators();
      renderPresenceLayer();
      return;
    }
    setStatus("error", "websocket connection failed");
    updateEditorDisabled("error");
    reconnectButton.disabled = false;
  });
}

function scheduleTitleSync(delayMs) {
  window.clearTimeout(titleSyncTimer);
  titleSyncTimer = window.setTimeout(() => {
    flushTitleSync().catch((error) => {
      setStatus("error", error.message);
      updateEditorDisabled("error");
    });
  }, delayMs);
}

async function flushTitleSync() {
  if (!sessionIntent || !socket || socket.readyState !== WebSocket.OPEN || !sessionReady) {
    return;
  }
  const nextTitle = normalizeTitle(titleInput.value);
  if (nextTitle === persistedDocumentTitle) {
    titleDirty = false;
    updateSaveStateChip();
    return;
  }
  titleUpdateInFlight = true;
  updateSaveStateChip();
  socket.send(JSON.stringify({
    type: "update_title",
    title: nextTitle,
  }));
}

function schedulePresenceUpdate(delayMs) {
  if (!sessionIntent || !socket || socket.readyState !== WebSocket.OPEN || !sessionReady) {
    return;
  }
  window.clearTimeout(presenceUpdateTimer);
  presenceUpdateTimer = window.setTimeout(() => {
    sendPresenceUpdate();
  }, delayMs);
}

function sendPresenceUpdate() {
  if (!sessionIntent || !socket || socket.readyState !== WebSocket.OPEN || !sessionReady) {
    return;
  }
  const selectionStart = typeof editor.selectionStart === "number" ? editor.selectionStart : 0;
  const selectionEnd = typeof editor.selectionEnd === "number" ? editor.selectionEnd : selectionStart;
  const anchorRuneIndex = runeIndexFromCodeUnitIndex(editor.value, selectionStart);
  const focusRuneIndex = runeIndexFromCodeUnitIndex(editor.value, selectionEnd);
  socket.send(JSON.stringify({
    type: "presence_update",
    presence: {
      display_name: actorInput.value.trim(),
      cursor_anchor: anchorForRuneIndex(anchorRuneIndex, localVisibleElements),
      cursor_focus: anchorForRuneIndex(focusRuneIndex, localVisibleElements),
      selection_direction: selectionStart <= selectionEnd ? "forward" : "backward",
      is_collapsed: selectionStart === selectionEnd,
    },
  }));
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
    emitDebugEvent("queue_store_init_failed", "error", "Offline queue store initialization failed.", {});
    setStatus("error", queueBlockedReason);
    updateEditorDisabled("error");
  }
}

function updateSaveStateChip() {
  const saveState = deriveSaveState({
    connectionState: browserStatusState.getConnectionState(),
    pendingQueueCount: queueMetadata?.pending_queue_count ?? pendingQueueRecords.length,
    blockedReason: queueBlockedReason,
    replayInFlight,
    liveSubmitInFlight,
    titleUpdateInFlight,
  });
  saveStateChipNode.textContent = saveState.label;
  saveStateChipNode.className = `save-state-chip save-state-${saveState.tone}`;
}

function renderCollaborators() {
  const remoteCollaborators = collaborators.filter((collaborator) => !collaborator.is_self);
  if (remoteCollaborators.length === 0) {
    collaboratorStripNode.innerHTML = '<span class="collaborator-empty">No collaborators yet</span>';
    return;
  }
  collaboratorStripNode.innerHTML = remoteCollaborators.map((collaborator) => {
    const color = colorTokenForActor(collaborator.actor_id);
    const label = escapeHTML(collaborator.display_name || collaborator.actor_id || "Collaborator");
    return `
      <span class="collaborator-chip">
        <span class="collaborator-dot" style="background:${color}"></span>
        <span>${label}</span>
      </span>
    `;
  }).join("");
}

function renderPresenceLayer() {
  if (!presenceLayer) {
    return;
  }
  const text = editor.value;
  const remoteCollaborators = collaborators.filter((collaborator) => !collaborator.is_self);
  if (remoteCollaborators.length === 0 || text === "") {
    presenceLayer.innerHTML = "";
    return;
  }
  const nodes = [];
  for (const collaborator of remoteCollaborators) {
    const color = colorTokenForActor(collaborator.actor_id);
    const anchorRuneIndex = runeIndexFromAnchor(collaborator.cursor_anchor, localVisibleElements);
    const focusRuneIndex = runeIndexFromAnchor(collaborator.cursor_focus, localVisibleElements);
    const startRuneIndex = Math.min(anchorRuneIndex, focusRuneIndex);
    const endRuneIndex = Math.max(anchorRuneIndex, focusRuneIndex);
    const rect = measureTextRange(editor, text, startRuneIndex, Math.max(startRuneIndex, endRuneIndex));
    if (!rect) {
      continue;
    }
    const displayName = escapeHTML(collaborator.display_name || collaborator.actor_id || "Collaborator");
    if (!collaborator.is_collapsed && rect.width > 0 && rect.height > 0) {
      nodes.push(`
        <div class="remote-selection" style="left:${rect.left}px;top:${rect.top}px;width:${rect.width}px;height:${rect.height}px;background:${color};"></div>
      `);
    }
    const caretRect = measureTextCaret(editor, text, focusRuneIndex);
    if (!caretRect) {
      continue;
    }
    nodes.push(`
      <div class="remote-caret" style="left:${caretRect.left}px;top:${caretRect.top}px;height:${caretRect.height}px;background:${color};">
        <span class="remote-caret-label" style="background:${color};">${displayName}</span>
      </div>
    `);
  }
  presenceLayer.innerHTML = nodes.join("");
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
    emitDebugEvent("queue_load_failed", "error", "Pending queue records could not be loaded.", {});
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
    emitDebugEvent("queue_metadata_load_failed", "error", "Offline queue metadata could not be loaded.", {});
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
    emitDebugEvent("offline_queue_appended", "warning", "Offline operations were appended to the local queue.", {
      appended_count: queuedOperations.length,
      pending_queue_count: pendingQueueRecords.length,
    });
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
  emitDebugEvent("replay_started", "warning", "Queued operations started replay after reconnect.", {
    pending_queue_count: pendingQueueRecords.length,
  });
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
      emitDebugEvent("replay_progress", "info", "Queued operation replay progressed.", {
        pending_queue_count: pendingQueueRecords.length,
      });
    }

    await queueStore.clearReplayed(sessionIntent.documentID, sessionIntent.actorID);
    reconnectingWithQueue = false;
    queueBlockedReason = "";
    emitDebugEvent("replay_completed", "info", "Queued operation replay completed.", {
      pending_queue_count: pendingQueueRecords.length,
    });
    setStatus("live", "");
    updateEditorDisabled("live");
  } catch (error) {
    if (pendingQueueRecords.length > 0) {
      const record = pendingQueueRecords[0];
      await queueStore.markBlocked(record.document_id, record.actor_id, record.operation_id, error.message);
    }
    emitDebugEvent("replay_failed", "error", "Queued operation replay failed.", {
      pending_queue_count: pendingQueueRecords.length,
    });
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
  emitDebugEvent("queue_blocked", "error", "Queue entered a blocked state and editing is read-only.", {
    pending_queue_count: pendingQueueRecords.length,
  });
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
  collaborators = [];
  sessionReady = false;
  pendingEditorText = null;
  stateWaiters = [];
  lastText = "";
  titleUpdateInFlight = false;
  titleDirty = false;
  persistedDocumentTitle = "Untitled document";
  titleInput.value = persistedDocumentTitle;
  window.syncraftPendingQueueRecords = pendingQueueRecords;
  window.syncraftQueueMetadata = queueMetadata;
  const nextStatus = browserStatusState.reset();
  editor.value = "";
  statusNode.textContent = nextStatus.connectionState;
  statusNode.className = `status-${nextStatus.connectionState}`;
  queueSummaryNode.textContent = "no queued local operations";
  errorNode.textContent = nextStatus.errorText;
  if (intent) {
    actorInput.value = intent.actorID;
    documentInput.value = intent.documentID;
  }
  renderCollaborators();
  renderPresenceLayer();
  updateSaveStateChip();
  renderDebugDiagnostics();
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
  titleInput.disabled = !sessionIntent || replayInFlight || Boolean(queueBlockedReason) || !socket || socket.readyState !== WebSocket.OPEN || !sessionReady;
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
  updateSaveStateChip();
  renderDebugDiagnostics();
}

function normalizeTitle(value) {
  const normalized = typeof value === "string" ? value.trim() : "";
  return normalized === "" ? "Untitled document" : normalized;
}

function measureTextCaret(textarea, text, runeIndex) {
  return measureTextRange(textarea, text, runeIndex, runeIndex);
}

function measureTextRange(textarea, text, startRuneIndex, endRuneIndex) {
  if (!(textarea instanceof HTMLTextAreaElement)) {
    return null;
  }
  const startCodeUnitIndex = codeUnitIndexFromRuneIndex(text, startRuneIndex);
  const endCodeUnitIndex = codeUnitIndexFromRuneIndex(text, endRuneIndex);
  const selectionText = text.slice(startCodeUnitIndex, endCodeUnitIndex);
  const styles = window.getComputedStyle(textarea);
  const mirror = document.createElement("div");
  mirror.style.position = "absolute";
  mirror.style.visibility = "hidden";
  mirror.style.whiteSpace = "pre-wrap";
  mirror.style.wordWrap = "break-word";
  mirror.style.overflow = "hidden";
  mirror.style.width = `${textarea.clientWidth}px`;
  mirror.style.font = styles.font;
  mirror.style.fontFamily = styles.fontFamily;
  mirror.style.fontSize = styles.fontSize;
  mirror.style.fontWeight = styles.fontWeight;
  mirror.style.lineHeight = styles.lineHeight;
  mirror.style.letterSpacing = styles.letterSpacing;
  mirror.style.padding = styles.padding;
  mirror.style.border = styles.border;
  mirror.style.top = "-9999px";
  mirror.style.left = "0";
  mirror.style.tabSize = styles.tabSize;

  const before = document.createTextNode(text.slice(0, startCodeUnitIndex));
  const marker = document.createElement("span");
  marker.textContent = selectionText || "\u200b";
  const after = document.createTextNode(text.slice(endCodeUnitIndex) || "\u200b");
  mirror.append(before, marker, after);
  document.body.appendChild(mirror);

  const markerRect = marker.getBoundingClientRect();
  const mirrorRect = mirror.getBoundingClientRect();
  const result = {
    left: Math.max(0, markerRect.left - mirrorRect.left - textarea.scrollLeft),
    top: Math.max(0, markerRect.top - mirrorRect.top - textarea.scrollTop),
    width: Math.max(selectionText ? markerRect.width : 2, 2),
    height: Math.max(markerRect.height, parseFloat(styles.lineHeight) || 20),
  };
  mirror.remove();
  return result;
}

window.syncraftBrowserTestAPI = {
  forceDisconnect() {
    if (socket) {
      socket.close();
    }
  },
  isDebugPanelExpanded() {
    return debugPanelState.isExpanded();
  },
  getDebugPanelDiagnostics() {
    return deriveDebugPanelDiagnostics({
      sessionIntent,
      browserStatusState,
      queueMetadata,
      pendingQueueRecords,
      queueBlockedReason,
      reconnectingWithQueue,
      replayInFlight,
      debugEvents: debugEventBuffer.list(),
    });
  },
  getCollaborators() {
    return collaborators;
  },
  getDocumentTitle() {
    return titleInput.value;
  },
};

/**
 * renderDebugPanel syncs the current debug-panel expansion state into the DOM shell.
 */
function renderDebugPanel(expanded) {
  debugPanel.hidden = !expanded;
  debugPanelToggleButton.setAttribute("aria-expanded", String(expanded));
  debugPanelToggleButton.textContent = expanded ? "Hide Debug Panel" : "Show Debug Panel";
}

/**
 * renderDebugDiagnostics projects the current browser-shell state into the operator-facing panel.
 */
function renderDebugDiagnostics() {
  const diagnostics = deriveDebugPanelDiagnostics({
    sessionIntent,
    browserStatusState,
    queueMetadata,
    pendingQueueRecords,
    queueBlockedReason,
    reconnectingWithQueue,
    replayInFlight,
    debugEvents: debugEventBuffer.list(),
  });
  debugActorInstanceNode.textContent = diagnostics.session.actorInstanceID;
  debugDocumentIDNode.textContent = diagnostics.session.documentID;
  debugConnectionStateNode.textContent = diagnostics.session.connectionState;
  debugLastErrorNode.textContent = diagnostics.session.lastError;
  debugQueueStateNode.textContent = diagnostics.queue.queueState;
  debugPendingQueueCountNode.textContent = diagnostics.queue.pendingQueueCount;
  debugQueueSummaryNode.textContent = diagnostics.queue.queueSummary;
  debugLastSnapshotIDNode.textContent = diagnostics.queue.lastSnapshotID;
  debugLastOperationIDNode.textContent = diagnostics.queue.lastOperationID;
  debugReplayStateNode.textContent = diagnostics.replay.replayState;
  debugReplayInFlightNode.textContent = diagnostics.replay.replayInFlight;
  debugBlockedReasonNode.textContent = diagnostics.replay.blockedReason;
  renderDebugEvents(diagnostics.recentEvents);
}

/**
 * emitDebugEvent records one payload-safe browser event and refreshes the diagnostics panel.
 */
function emitDebugEvent(eventType, level, message, details = {}) {
  const diagnostics = deriveDebugPanelDiagnostics({
    sessionIntent,
    browserStatusState,
    queueMetadata,
    pendingQueueRecords,
    queueBlockedReason,
    reconnectingWithQueue,
    replayInFlight,
  });
  debugEventBuffer.append({
    event_type: eventType,
    level,
    document_id: diagnostics.session.documentID,
    actor_id: diagnostics.session.actorInstanceID,
    connection_state: diagnostics.session.connectionState,
    queue_state: diagnostics.queue.queueState,
    pending_queue_count: Number.parseInt(diagnostics.queue.pendingQueueCount, 10) || 0,
    snapshot_id: diagnostics.queue.lastSnapshotID,
    last_operation_id: diagnostics.queue.lastOperationID,
    message,
    details,
  });
  renderDebugDiagnostics();
}

/**
 * renderDebugEvents paints the bounded event timeline into the debug panel.
 */
function renderDebugEvents(events) {
  if (!Array.isArray(events) || events.length === 0) {
    debugEventsListNode.innerHTML = '<li class="debug-event-empty">No recent events yet.</li>';
    return;
  }
  debugEventsListNode.innerHTML = events.map((event) => `
      <li class="debug-event debug-event-${escapeHTML(event.level)}">
        <div class="debug-event-head">
          <span class="debug-event-type">${escapeHTML(event.eventType)}</span>
          <span class="debug-event-time">${escapeHTML(formatOccurredAt(event.occurredAt))}</span>
        </div>
        <p class="debug-event-message">${escapeHTML(event.message)}</p>
        <p class="debug-event-meta">Actor ${escapeHTML(event.actorID || sessionIntent?.actorID || "none")} · Document ${escapeHTML(event.documentID || sessionIntent?.documentID || "none")}</p>
      </li>
    `).join("");
}

function formatOccurredAt(value) {
  if (typeof value !== "string" || value.trim() === "") {
    return "unknown time";
  }
  const parsed = new Date(value);
  if (Number.isNaN(parsed.getTime())) {
    return value;
  }
  return parsed.toLocaleTimeString([], {
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
  });
}

function escapeHTML(value) {
  return String(value)
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll("\"", "&quot;")
    .replaceAll("'", "&#39;");
}

renderDebugDiagnostics();
renderDebugPanel(debugPanelState.isExpanded());
renderCollaborators();
renderPresenceLayer();
updateSaveStateChip();

