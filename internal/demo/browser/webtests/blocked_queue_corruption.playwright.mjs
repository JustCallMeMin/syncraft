import test from "node:test";
import assert from "node:assert/strict";
import { spawn } from "node:child_process";
import { once } from "node:events";
import { setTimeout as delay } from "node:timers/promises";
import { tmpdir } from "node:os";
import { mkdtempSync } from "node:fs";
import path from "node:path";

import { chromium } from "playwright";

const repoRoot = path.resolve(import.meta.dirname, "..", "..", "..", "..");

test("corrupted queued operation record surfaces queue_blocked on reconnect", async (t) => {
  const port = 20150 + Math.floor(Math.random() * 200);
  const dataDir = mkdtempSync(path.join(tmpdir(), "syncraft-server-data-"));
  const userDataDir = mkdtempSync(path.join(tmpdir(), "syncraft-browser-profile-"));
  const origin = `http://127.0.0.1:${port}`;

  const server = await startDemoServer(port, dataDir);
  t.after(async () => {
    await stopDemoServer(server);
  });

  const context = await chromium.launchPersistentContext(userDataDir);
  t.after(async () => {
    await context.close();
  });

  const page = await context.newPage();
  await connectEditor(page, origin, "actor-corrupt-record", "doc-corrupt-record");
  await transitionToOfflineQueueState(context, page, "ab");
  await corruptPendingQueuedOperation(page, "doc-corrupt-record", "actor-corrupt-record");

  await context.setOffline(false);
  await page.reload();
  await waitForBlockedQueue(page, /stored queued operation 1 is corrupted/i);
  assert.equal(await page.locator("#editor").isDisabled(), true);
});

test("corrupted queue metadata surfaces queue_blocked before reconnect", async (t) => {
  const port = 20450 + Math.floor(Math.random() * 200);
  const dataDir = mkdtempSync(path.join(tmpdir(), "syncraft-server-data-"));
  const userDataDir = mkdtempSync(path.join(tmpdir(), "syncraft-browser-profile-"));
  const origin = `http://127.0.0.1:${port}`;

  const server = await startDemoServer(port, dataDir);
  t.after(async () => {
    await stopDemoServer(server);
  });

  const context = await chromium.launchPersistentContext(userDataDir);
  t.after(async () => {
    await context.close();
  });

  const page = await context.newPage();
  await connectEditor(page, origin, "actor-corrupt-meta", "doc-corrupt-meta");
  await corruptQueueMetadata(page, "doc-corrupt-meta", "actor-corrupt-meta");
  await page.reload();
  await waitForBlockedQueue(page, /stored queue metadata is corrupted/i);
  assert.equal(await page.locator("#editor").isDisabled(), true);
});

async function connectEditor(page, origin, actorID, documentID) {
  await page.goto(origin);
  await page.getByLabel("Actor Instance").fill(actorID);
  await page.getByLabel("Document").fill(documentID);
  await page.getByRole("button", { name: "Connect", exact: true }).click();
  await page.waitForFunction(() => {
    const status = document.getElementById("status")?.textContent?.trim();
    const error = document.getElementById("error")?.textContent?.trim();
    return status === "live" && error === "none";
  }, null, { timeout: 10000 });
}

async function transitionToOfflineQueueState(context, page, queuedText) {
  await context.setOffline(true);
  await page.evaluate(() => {
    window.syncraftBrowserTestAPI.forceDisconnect();
  });
  await page.waitForFunction(() => {
    const status = document.getElementById("status")?.textContent?.trim();
    return status === "disconnected";
  }, null, { timeout: 10000 });
  await page.locator("#editor").fill(queuedText);
  await page.waitForFunction((expectedText) => {
    const status = document.getElementById("status")?.textContent?.trim();
    const queue = document.getElementById("queue-summary")?.textContent ?? "";
    const text = document.getElementById("editor")?.value ?? "";
    return text === expectedText &&
      status === "offline_provisional" &&
      queue.includes("queued operation");
  }, queuedText, { timeout: 5000 });
}

async function corruptPendingQueuedOperation(page, documentID, actorID) {
  await page.evaluate(async ({ documentID, actorID }) => {
    const db = await new Promise((resolve, reject) => {
      const request = window.indexedDB.open("syncraft-offline-queue", 1);
      request.onsuccess = () => resolve(request.result);
      request.onerror = () => reject(request.error ?? new Error("open offline queue database failed"));
    });
    const queueStore = db.transaction(["queued_operations"], "readwrite").objectStore("queued_operations");
    const request = queueStore.getAll();
    const records = await new Promise((resolve, reject) => {
      request.onsuccess = () => resolve(request.result ?? []);
      request.onerror = () => reject(request.error ?? new Error("load queued operations failed"));
    });
    const record = records.find((entry) => entry.document_id === documentID && entry.actor_id === actorID);
    if (!record) {
      throw new Error("pending queue record was not found for corruption test");
    }
    record.operation.actor_counter += 1;
    await new Promise((resolve, reject) => {
      const putRequest = queueStore.put(record);
      putRequest.onsuccess = () => resolve();
      putRequest.onerror = () => reject(putRequest.error ?? new Error("write corrupted queue record failed"));
    });
  }, { documentID, actorID });
}

async function corruptQueueMetadata(page, documentID, actorID) {
  await page.evaluate(async ({ documentID, actorID }) => {
    const db = await new Promise((resolve, reject) => {
      const request = window.indexedDB.open("syncraft-offline-queue", 1);
      request.onsuccess = () => resolve(request.result);
      request.onerror = () => reject(request.error ?? new Error("open offline queue database failed"));
    });
    const metadataStore = db.transaction(["queue_metadata"], "readwrite").objectStore("queue_metadata");
    await new Promise((resolve, reject) => {
      const putRequest = metadataStore.put({
        metadata_key: `${documentID}::${actorID}`,
        document_id: documentID,
        actor_id: actorID,
        next_actor_counter: 0,
        pending_queue_count: 0,
        last_snapshot_id: null,
        last_operation_id: null,
      });
      putRequest.onsuccess = () => resolve();
      putRequest.onerror = () => reject(putRequest.error ?? new Error("write corrupted queue metadata failed"));
    });
  }, { documentID, actorID });
}

async function waitForBlockedQueue(page, errorPattern) {
  await page.waitForFunction((patternSource) => {
    const status = document.getElementById("status")?.textContent?.trim() ?? "";
    const queue = document.getElementById("queue-summary")?.textContent?.trim() ?? "";
    return status === "queue_blocked" && new RegExp(patternSource, "i").test(queue);
  }, errorPattern.source, { timeout: 10000 });
}

async function startDemoServer(port, dataDir) {
  const server = spawn("go", ["run", "./cmd/demo-server", "-listen", `:${port}`, "-data-dir", dataDir], {
    cwd: repoRoot,
    stdio: "pipe",
  });
  let exited = false;
  server.once("exit", () => {
    exited = true;
  });
  await waitForServer(`http://127.0.0.1:${port}`, () => exited);
  return server;
}

async function stopDemoServer(server) {
  if (!server || server.killed) {
    return;
  }
  server.kill("SIGKILL");
  server.stdout?.destroy();
  server.stderr?.destroy();
  await Promise.race([
    once(server, "close"),
    delay(1000),
  ]);
}

async function waitForServer(url, didServerExit) {
  for (let attempt = 0; attempt < 50; attempt += 1) {
    if (typeof didServerExit === "function" && didServerExit()) {
      throw new Error(`server ${url} exited before becoming ready`);
    }
    try {
      const response = await fetch(url);
      if (response.ok) {
        return;
      }
    } catch {
      // Retry while server starts.
    }
    await delay(200);
  }
  throw new Error(`server ${url} did not become ready`);
}
