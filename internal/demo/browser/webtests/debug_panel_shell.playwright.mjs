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

test("debug panel shell is collapsed by default and toggleable without affecting the editor", async (t) => {
  const port = 20750 + Math.floor(Math.random() * 200);
  const dataDir = mkdtempSync(path.join(tmpdir(), "syncraft-server-data-"));
  const origin = `http://127.0.0.1:${port}`;

  let server = await startDemoServer(port, dataDir);
  t.after(async () => {
    await stopDemoServer(server);
  });

  const browser = await chromium.launch();
  t.after(async () => {
    await browser.close();
  });

  const page = await browser.newPage();
  await page.goto(origin);

  assert.equal(await page.locator("#debug-panel").isHidden(), true);
  await expectToggleState(page, false);

  await page.getByRole("button", { name: "Show Debug Panel", exact: true }).click();
  assert.equal(await page.locator("#debug-panel").isVisible(), true);
  await expectToggleState(page, true);

  await page.getByRole("button", { name: "Hide Debug Panel", exact: true }).click();
  assert.equal(await page.locator("#debug-panel").isHidden(), true);
  await expectToggleState(page, false);

  await page.getByLabel("Actor Instance").fill("actor-debug-shell");
  await page.getByLabel("Document").fill("doc-debug-shell");
  await page.getByRole("button", { name: "Connect", exact: true }).click();
  await page.waitForFunction(() => {
    const status = document.getElementById("status")?.textContent?.trim();
    const error = document.getElementById("error")?.textContent?.trim();
    return status === "live" && error === "none";
  }, null, { timeout: 10000 });

  await page.getByRole("button", { name: "Show Debug Panel", exact: true }).click();
  await page.waitForFunction(() => {
    const actor = document.getElementById("debug-actor-instance")?.textContent?.trim();
    const documentID = document.getElementById("debug-document-id")?.textContent?.trim();
    const connection = document.getElementById("debug-connection-state")?.textContent?.trim();
    const pendingCount = document.getElementById("debug-pending-queue-count")?.textContent?.trim();
    const queueState = document.getElementById("debug-queue-state")?.textContent?.trim();
    const eventTypes = Array.from(document.querySelectorAll(".debug-event-type")).map((node) => node.textContent?.trim());
    return actor === "actor-debug-shell"
      && documentID === "doc-debug-shell"
      && connection === "live"
      && queueState === "live"
      && pendingCount === "0"
      && eventTypes.includes("state_ready")
      && eventTypes.includes("connect_requested");
  }, null, { timeout: 5000 });
  await page.locator("#editor").fill("debug panel shell");
  await page.waitForFunction(() => {
    const text = document.getElementById("editor")?.value ?? "";
    const status = document.getElementById("status")?.textContent?.trim();
    return text === "debug panel shell" && status === "live";
  }, null, { timeout: 5000 });

  await stopDemoServer(server);
  await page.evaluate(() => window.syncraftBrowserTestAPI.forceDisconnect());
  await page.waitForFunction(() => {
    const connection = document.getElementById("debug-connection-state")?.textContent?.trim();
    return connection === "disconnected";
  }, null, { timeout: 5000 });

  await page.locator("#editor").fill("debug panel shell offline");
  await page.waitForFunction(() => {
    const queueState = document.getElementById("debug-queue-state")?.textContent?.trim();
    const pendingCount = document.getElementById("debug-pending-queue-count")?.textContent?.trim();
    const replayState = document.getElementById("debug-replay-state")?.textContent?.trim();
    const eventTypes = Array.from(document.querySelectorAll(".debug-event-type")).map((node) => node.textContent?.trim());
    const eventText = document.getElementById("debug-events-list")?.textContent ?? "";
    return queueState === "offline_provisional"
      && pendingCount !== "0"
      && replayState === "queued_idle"
      && eventTypes.includes("offline_queue_appended")
      && eventTypes.includes("disconnect_detected")
      && !eventText.includes("debug panel shell offline");
  }, null, { timeout: 5000 });
});

async function expectToggleState(page, expanded) {
  const button = page.locator("#debug-panel-toggle");
  assert.equal(await button.getAttribute("aria-expanded"), String(expanded));
  assert.equal(await page.evaluate(() => window.syncraftBrowserTestAPI.isDebugPanelExpanded()), expanded);
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
