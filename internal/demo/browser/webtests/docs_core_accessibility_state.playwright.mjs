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

test("docs-core shell exposes accessible names and stays usable across viewport sizes", async (t) => {
  const port = 21400 + Math.floor(Math.random() * 200);
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

  for (const width of [375, 768, 1024, 1440]) {
    const page = await browser.newPage({ viewport: { width, height: 900 } });
    await page.goto(origin);

    await assertAccessibleShell(page);
    await connectEditor(page, origin, `actor-${width}`, `doc-${width}`);

    const editorBox = await page.locator("#editor").boundingBox();
    const titleBox = await page.locator("#document-title").boundingBox();
    const railBox = await page.locator(".utility-rail").boundingBox();
    assert.ok(editorBox && editorBox.width > 180, `editor width should remain usable at ${width}`);
    assert.ok(titleBox && titleBox.width > 100, `title field should remain visible at ${width}`);
    assert.ok(railBox && railBox.height > 100, `utility rail should remain visible at ${width}`);

    await page.close();
  }
});

async function assertAccessibleShell(page) {
  await assert.doesNotReject(async () => page.getByLabel("Title").isVisible());
  await assert.doesNotReject(async () => page.getByLabel("Actor Instance").isVisible());
  await assert.doesNotReject(async () => page.getByLabel("Document").isVisible());
  assert.equal(await page.locator("#collaborator-strip").getAttribute("aria-label"), "Active collaborators");
  assert.equal(await page.locator("#debug-panel-toggle").getAttribute("aria-controls"), "debug-panel");
}

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
