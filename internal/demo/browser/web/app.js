import { createOfflineQueueStore } from "/offline_queue_store.js";

const form = document.getElementById("connect-form");
const actorInput = document.getElementById("actor-id");
const documentInput = document.getElementById("document-id");
const reconnectButton = document.getElementById("reconnect-button");
const editor = document.getElementById("editor");
const statusNode = document.getElementById("status");
const errorNode = document.getElementById("error");

let socket = null;
let applyingRemoteState = false;
let lastText = "";
let reconnectTimer = null;
let sessionIntent = null;
let queueStore = null;
let pendingQueueRecords = [];

actorInput.value = actorInput.value || `actor-${Math.random().toString(36).slice(2, 8)}`;

initializeQueueStore();

form.addEventListener("submit", async (event) => {
  event.preventDefault();
  await connect({
    actorID: actorInput.value.trim(),
    documentID: documentInput.value.trim(),
    retry: true,
  });
});

reconnectButton.addEventListener("click", async () => {
  if (!sessionIntent) {
    setStatus("error", "connect once before trying to reconnect");
    return;
  }
  await connect(sessionIntent);
});

editor.addEventListener("input", () => {
  if (applyingRemoteState || !socket || socket.readyState !== WebSocket.OPEN) {
    return;
  }

  const nextText = editor.value;
  const diff = computeDiff(lastText, nextText);
  if (!diff) {
    lastText = nextText;
    return;
  }

  const removedCount = diff.removed.length;
  for (let index = 0; index < removedCount; index += 1) {
    socket.send(JSON.stringify({
      type: "delete_at",
      index: diff.start,
    }));
  }

  if (diff.inserted.length > 0) {
    socket.send(JSON.stringify({
      type: "insert_text",
      index: diff.start,
      value: diff.inserted,
    }));
  }
});

async function connect(intent) {
  sessionIntent = intent;
  window.clearTimeout(reconnectTimer);

  if (!(await loadPendingQueue(intent))) {
    return;
  }

  if (socket && socket.readyState === WebSocket.OPEN) {
    socket.close();
  }

  const protocol = window.location.protocol === "https:" ? "wss" : "ws";
  socket = new WebSocket(`${protocol}://${window.location.host}/ws`);
  setStatus("connecting", "");
  editor.disabled = true;
  reconnectButton.disabled = true;

  socket.addEventListener("open", () => {
    socket.send(JSON.stringify({
      type: "init",
      actor_id: intent.actorID,
      document_id: intent.documentID,
    }));
  });

  socket.addEventListener("message", (event) => {
    const message = JSON.parse(event.data);
    if (message.type !== "state") {
      return;
    }
    applyingRemoteState = true;
    editor.value = message.text || "";
    lastText = editor.value;
    applyingRemoteState = false;
    setStatus(message.connection_state || "live", message.last_error || "");
    editor.disabled = message.connection_state === "error" || message.connection_state === "connecting";
    reconnectButton.disabled = false;
  });

  socket.addEventListener("close", () => {
    setStatus("disconnected", "");
    editor.disabled = true;
    reconnectButton.disabled = false;
    if (intent.retry) {
      reconnectTimer = window.setTimeout(() => {
        connect(intent).catch((error) => {
          setStatus("error", error.message);
        });
      }, 750);
    }
  });

  socket.addEventListener("error", () => {
    setStatus("error", "websocket connection failed");
    editor.disabled = true;
    reconnectButton.disabled = false;
  });
}

async function initializeQueueStore() {
  try {
    queueStore = createOfflineQueueStore();
    await queueStore.open();
    window.syncraftOfflineQueueStore = queueStore;
  } catch (error) {
    queueStore = null;
    setStatus("error", `offline queue store init failed: ${error.message}`);
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
    return true;
  } catch (error) {
    setStatus("error", `offline queue load failed: ${error.message}`);
    editor.disabled = true;
    reconnectButton.disabled = false;
    return false;
  }
}

function setStatus(state, errorText) {
  statusNode.textContent = state;
  statusNode.className = state === "error" ? "status-error" : state === "live" ? "status-live" : "";
  errorNode.textContent = errorText || "none";
}

function computeDiff(previous, next) {
  if (previous === next) {
    return null;
  }

  let start = 0;
  while (start < previous.length && start < next.length && previous[start] === next[start]) {
    start += 1;
  }

  let previousEnd = previous.length;
  let nextEnd = next.length;
  while (
    previousEnd > start &&
    nextEnd > start &&
    previous[previousEnd - 1] === next[nextEnd - 1]
  ) {
    previousEnd -= 1;
    nextEnd -= 1;
  }

  return {
    start,
    removed: previous.slice(start, previousEnd),
    inserted: next.slice(start, nextEnd),
  };
}
