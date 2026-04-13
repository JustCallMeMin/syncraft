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

test("tabs keep independent actor/document intent and refresh does not cross-overwrite", async (t) => {
  const port = 20100 + Math.floor(Math.random() * 200);
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

  const context = await browser.newContext();
  const pageA = await context.newPage();
  const pageB = await context.newPage();

  await pageA.goto(origin);
  await pageA.getByLabel("Actor Instance").fill("actor-tab-a");
  await pageA.getByLabel("Document").fill("doc-tab-a");
  await pageA.getByRole("button", { name: "Connect", exact: true }).click();
  await waitForLive(pageA);

  await pageB.goto(origin);
  const actorBInitial = await pageB.getByLabel("Actor Instance").inputValue();
  const documentBInitial = await pageB.getByLabel("Document").inputValue();
  assert.notEqual(actorBInitial, "actor-tab-a");
  assert.notEqual(documentBInitial, "doc-tab-a");

  await pageB.getByLabel("Actor Instance").fill("actor-tab-b");
  await pageB.getByLabel("Document").fill("doc-tab-b");
  await pageB.getByRole("button", { name: "Connect", exact: true }).click();
  await waitForLive(pageB);

  await pageA.reload();
  await pageB.reload();

  assert.equal(await pageA.getByLabel("Actor Instance").inputValue(), "actor-tab-a");
  assert.equal(await pageA.getByLabel("Document").inputValue(), "doc-tab-a");
  assert.equal(await pageB.getByLabel("Actor Instance").inputValue(), "actor-tab-b");
  assert.equal(await pageB.getByLabel("Document").inputValue(), "doc-tab-b");
});

async function waitForLive(page) {
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
