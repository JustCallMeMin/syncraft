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

test("title propagation and presence rendering work across two tabs", async (t) => {
  const port = 21100 + Math.floor(Math.random() * 200);
  const dataDir = mkdtempSync(path.join(tmpdir(), "syncraft-server-data-"));
  const origin = `http://127.0.0.1:${port}`;

  const server = await startDemoServer(port, dataDir);
  t.after(async () => {
    await stopDemoServer(server);
  });

  const browser = await chromium.launch();
  t.after(async () => {
    await browser.close();
  });

  const pageA = await browser.newPage();
  const pageB = await browser.newPage();

  await connectEditor(pageA, origin, "actor-a", "doc-shared");
  await connectEditor(pageB, origin, "actor-b", "doc-shared");

  await pageA.getByLabel("Title").fill("Shared Sprint Notes");
  await pageA.getByLabel("Title").blur();
  await pageB.waitForFunction(() => {
    return document.getElementById("document-title")?.value === "Shared Sprint Notes";
  }, null, { timeout: 5000 });

  await pageB.locator("#editor").fill("hello world");
  await pageB.locator("#editor").evaluate((node) => {
    node.focus();
    node.setSelectionRange(0, 5);
    node.dispatchEvent(new Event("selectionchange", { bubbles: true }));
  });

  await pageA.waitForFunction(() => {
    const collaborators = document.querySelectorAll(".collaborator-chip").length;
    const caret = document.querySelectorAll(".remote-caret").length;
    const selection = document.querySelectorAll(".remote-selection").length;
    return collaborators >= 1 && caret >= 1 && selection >= 1;
  }, null, { timeout: 5000 });

  assert.equal(await pageA.locator("#document-title").inputValue(), "Shared Sprint Notes");
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
