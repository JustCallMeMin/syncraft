import test from "node:test";
import assert from "node:assert/strict";

import { createDebugEventBuffer } from "../web/browser_debug_event_buffer.js";

test("createDebugEventBuffer keeps newest events first and bounded", () => {
  const buffer = createDebugEventBuffer(2);
  buffer.append({
    event_type: "connect_requested",
    message: "connect one",
    actor_id: "actor-a",
    document_id: "doc-a",
  });
  buffer.append({
    event_type: "state_ready",
    message: "state two",
    actor_id: "actor-a",
    document_id: "doc-a",
  });
  buffer.append({
    event_type: "disconnect_detected",
    message: "disconnect three",
    actor_id: "actor-a",
    document_id: "doc-a",
  });

  const events = buffer.list();
  assert.equal(buffer.size(), 2);
  assert.equal(events[0].event_type, "disconnect_detected");
  assert.equal(events[1].event_type, "state_ready");
});

test("createDebugEventBuffer normalizes unsafe values into payload-safe defaults", () => {
  const buffer = createDebugEventBuffer();
  const stored = buffer.append({
    event_type: "",
    level: "unknown",
    occurred_at: "",
    message: "",
    details: {
      pending_queue_count: 3,
      retry: true,
      raw_text: undefined,
    },
  });

  assert.equal(stored.event_type, "unknown_event");
  assert.equal(stored.level, "info");
  assert.equal(stored.message, "browser debug event");
  assert.deepEqual(stored.details, {
    pending_queue_count: 3,
    retry: true,
  });
});
