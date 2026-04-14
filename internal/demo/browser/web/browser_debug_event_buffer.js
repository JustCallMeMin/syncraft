/**
 * createDebugEventBuffer stores bounded browser-local diagnostics for the in-app debug panel.
 */
export function createDebugEventBuffer(maxEvents = 50) {
  const capacity = Number.isInteger(maxEvents) && maxEvents > 0 ? maxEvents : 50;
  let nextID = 1;
  let events = [];

  return {
    /**
     * append stores one normalized event and returns the stored record.
     */
    append(event) {
      const storedEvent = normalizeEvent(event, nextID);
      nextID += 1;
      events = [storedEvent, ...events].slice(0, capacity);
      return storedEvent;
    },

    /**
     * list returns newest-first timeline entries for rendering.
     */
    list() {
      return events.map((event) => ({ ...event, details: { ...event.details } }));
    },

    /**
     * size returns the current number of retained timeline entries.
     */
    size() {
      return events.length;
    },
  };
}

function normalizeEvent(event, nextID) {
  const normalizedDetails = normalizeDetails(event?.details);
  return {
    event_id: `browser-event-${nextID}`,
    event_type: normalizeEventType(event?.event_type),
    level: normalizeLevel(event?.level),
    occurred_at: normalizeTimestamp(event?.occurred_at),
    document_id: normalizeText(event?.document_id) || "none",
    actor_id: normalizeText(event?.actor_id) || "none",
    connection_state: normalizeText(event?.connection_state) || "disconnected",
    queue_state: normalizeText(event?.queue_state) || "disconnected",
    pending_queue_count: Number.isInteger(event?.pending_queue_count) && event.pending_queue_count >= 0
      ? event.pending_queue_count
      : 0,
    snapshot_id: normalizeText(event?.snapshot_id) || "none",
    last_operation_id: normalizeText(event?.last_operation_id) || "none",
    message: normalizeText(event?.message) || "browser debug event",
    details: normalizedDetails,
  };
}

function normalizeEventType(value) {
  const normalized = normalizeText(value);
  return normalized || "unknown_event";
}

function normalizeLevel(value) {
  const normalized = normalizeText(value);
  if (normalized === "warning" || normalized === "error") {
    return normalized;
  }
  return "info";
}

function normalizeTimestamp(value) {
  const normalized = normalizeText(value);
  return normalized || new Date().toISOString();
}

function normalizeDetails(value) {
  if (!value || typeof value !== "object" || Array.isArray(value)) {
    return {};
  }
  const safeDetails = {};
  for (const [key, rawValue] of Object.entries(value)) {
    if (typeof rawValue === "string") {
      safeDetails[key] = rawValue.trim();
      continue;
    }
    if (typeof rawValue === "number" || typeof rawValue === "boolean") {
      safeDetails[key] = rawValue;
    }
  }
  return safeDetails;
}

function normalizeText(value) {
  if (typeof value !== "string") {
    return "";
  }
  return value.trim();
}
