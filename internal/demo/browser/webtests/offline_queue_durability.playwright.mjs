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

test("same-profile page refresh replays queued offline edits after restore", async (t) => {
  const port = 19080 + Math.floor(Math.random() * 500);
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
  await connectEditor(page, origin, "actor-queue-refresh", "doc-queue-refresh");
  await openDebugPanel(page);
  await transitionToOfflineQueueState(context, page, "ab");

  await context.setOffline(false);
  await page.reload();
  await waitForRestoredReplay(page, "actor-queue-refresh", "doc-queue-refresh", "ab");
  await openDebugPanel(page);
  await waitForObservabilityRestore(page, "actor-queue-refresh", "doc-queue-refresh");
});

test("same-profile browser restart replays queued offline edits after restore", async (t) => {
  const port = 19600 + Math.floor(Math.random() * 300);
  const dataDir = mkdtempSync(path.join(tmpdir(), "syncraft-server-data-"));
  const userDataDir = mkdtempSync(path.join(tmpdir(), "syncraft-browser-profile-"));
  const origin = `http://127.0.0.1:${port}`;

  const server = await startDemoServer(port, dataDir);
  t.after(async () => {
    await stopDemoServer(server);
  });

  let context = await chromium.launchPersistentContext(userDataDir);
  t.after(async () => {
    await context.close();
  });

  let page = await context.newPage();
  await connectEditor(page, origin, "actor-queue-restart", "doc-queue-restart");
  await openDebugPanel(page);
  await transitionToOfflineQueueState(context, page, "ab");
  await context.close();

  context = await chromium.launchPersistentContext(userDataDir);
  page = await context.newPage();
  await page.goto(origin);
  await page.getByLabel("Actor Instance").fill("actor-queue-restart");
  await page.getByLabel("Document").fill("doc-queue-restart");
  await page.getByRole("button", { name: "Connect", exact: true }).click();
  await waitForLiveText(page, "ab");
  await openDebugPanel(page);
  await waitForObservabilityRestore(page, "actor-queue-restart", "doc-queue-restart");
});

test("offline replay converges to the same visible text as the online flow", async (t) => {
  const port = 19950 + Math.floor(Math.random() * 200);
  const dataDir = mkdtempSync(path.join(tmpdir(), "syncraft-server-data-"));
  const userDataDir = mkdtempSync(path.join(tmpdir(), "syncraft-browser-profile-"));
  const origin = `http://127.0.0.1:${port}`;

  const server = await startDemoServer(port, dataDir);
  t.after(async () => {
    await stopDemoServer(server);
  });

  const browser = await chromium.launch();
  t.after(async () => {
    await browser.close();
  });

  const onlineContext = await browser.newContext();
  t.after(async () => {
    await onlineContext.close();
  });
  const onlinePage = await onlineContext.newPage();
  await connectEditor(onlinePage, origin, "actor-online-flow", "doc-online-flow");
  await onlinePage.locator("#editor").fill("ab");
  await onlinePage.waitForFunction(() => {
    const text = document.getElementById("editor")?.value ?? "";
    const status = document.getElementById("status")?.textContent?.trim();
    return text === "ab" && status === "live";
  }, null, { timeout: 5000 });

  const offlineContext = await chromium.launchPersistentContext(userDataDir);
  t.after(async () => {
    await offlineContext.close();
  });
  const offlinePage = await offlineContext.newPage();
  await connectEditor(offlinePage, origin, "actor-offline-flow", "doc-offline-flow");
  await transitionToOfflineQueueState(offlineContext, offlinePage, "ab");

  await offlineContext.setOffline(false);
  await waitForLiveText(offlinePage, "ab");

  const onlineText = await onlinePage.locator("#editor").inputValue();
  const offlineText = await offlinePage.locator("#editor").inputValue();
  assert.equal(offlineText, onlineText);
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

async function openDebugPanel(page) {
  if (await page.getByRole("button", { name: "Show Debug Panel", exact: true }).isVisible()) {
    await page.getByRole("button", { name: "Show Debug Panel", exact: true }).click();
  }
}

async function transitionToOfflineQueueState(context, page, queuedText) {
  await context.setOffline(true);
  await page.evaluate(() => {
    window.syncraftBrowserTestAPI.forceDisconnect();
  });
  await waitForStatus(page, "disconnected");
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

async function waitForStatus(page, wantStatus) {
  await page.waitForFunction((expected) => {
    const status = document.getElementById("status")?.textContent?.trim();
    return status === expected;
  }, wantStatus, { timeout: 10000 });
}

async function waitForLiveText(page, expectedText) {
  await page.waitForFunction((wantText) => {
    const status = document.getElementById("status")?.textContent?.trim();
    const queue = document.getElementById("queue-summary")?.textContent?.trim() ?? "";
    const error = document.getElementById("error")?.textContent?.trim() ?? "";
    const text = document.getElementById("editor")?.value ?? "";
    return text === wantText &&
      status === "live" &&
      queue === "no queued local operations" &&
      error === "none";
  }, expectedText, { timeout: 10000 });
}

async function waitForRestoredReplay(page, actorID, documentID, expectedText) {
  await page.waitForFunction(({ actorID: wantActor, documentID: wantDoc, expectedText: wantText }) => {
    const actor = document.getElementById("actor-id")?.value ?? "";
    const doc = document.getElementById("document-id")?.value ?? "";
    const status = document.getElementById("status")?.textContent?.trim();
    const queue = document.getElementById("queue-summary")?.textContent?.trim() ?? "";
    const error = document.getElementById("error")?.textContent?.trim() ?? "";
    const text = document.getElementById("editor")?.value ?? "";
    return actor === wantActor &&
      doc === wantDoc &&
      text === wantText &&
      status === "live" &&
      queue === "no queued local operations" &&
      error === "none";
  }, { actorID, documentID, expectedText }, { timeout: 10000 });
}

async function waitForObservabilityRestore(page, actorID, documentID) {
  await page.waitForFunction(({ expectedActorID, expectedDocumentID }) => {
    const actor = document.getElementById("debug-actor-instance")?.textContent?.trim() ?? "";
    const documentID = document.getElementById("debug-document-id")?.textContent?.trim() ?? "";
    const connection = document.getElementById("debug-connection-state")?.textContent?.trim() ?? "";
    const queueState = document.getElementById("debug-queue-state")?.textContent?.trim() ?? "";
    const pendingCount = document.getElementById("debug-pending-queue-count")?.textContent?.trim() ?? "";
    const eventTypes = Array.from(document.querySelectorAll(".debug-event-type")).map((node) => node.textContent?.trim());
    return actor === expectedActorID
      && documentID === expectedDocumentID
      && connection === "live"
      && queueState === "live"
      && pendingCount === "0"
      && eventTypes.includes("state_ready");
  }, { expectedActorID: actorID, expectedDocumentID: documentID }, { timeout: 10000 });
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
