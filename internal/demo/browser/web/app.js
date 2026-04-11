const form = document.getElementById("connect-form");
const actorInput = document.getElementById("actor-id");
const documentInput = document.getElementById("document-id");
const editor = document.getElementById("editor");
const statusNode = document.getElementById("status");
const errorNode = document.getElementById("error");

let socket = null;
let applyingRemoteState = false;
let lastText = "";

actorInput.value = actorInput.value || `actor-${Math.random().toString(36).slice(2, 8)}`;

form.addEventListener("submit", (event) => {
  event.preventDefault();
  connect();
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

function connect() {
  if (socket && socket.readyState === WebSocket.OPEN) {
    socket.close();
  }

  const protocol = window.location.protocol === "https:" ? "wss" : "ws";
  socket = new WebSocket(`${protocol}://${window.location.host}/ws`);
  setStatus("connecting", "");
  editor.disabled = true;

  socket.addEventListener("open", () => {
    socket.send(JSON.stringify({
      type: "init",
      actor_id: actorInput.value.trim(),
      document_id: documentInput.value.trim(),
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
  });

  socket.addEventListener("close", () => {
    setStatus("disconnected", "");
    editor.disabled = true;
  });

  socket.addEventListener("error", () => {
    setStatus("error", "websocket connection failed");
    editor.disabled = true;
  });
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
