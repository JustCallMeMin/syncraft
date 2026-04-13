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

test("fresh connect accepts immediate unicode input and continued typing without sticky error state", async (t) => {
  const port = 18080 + Math.floor(Math.random() * 1000);
  const dataDir = mkdtempSync(path.join(tmpdir(), "syncraft-demo-"));
  const server = spawn("go", ["run", "./cmd/demo-server", "-listen", `:${port}`, "-data-dir", dataDir], {
    cwd: repoRoot,
    stdio: "pipe",
  });
  let serverExited = false;

  t.after(async () => {
    if (!server.killed) {
      server.kill("SIGKILL");
    }
    server.stdout?.destroy();
    server.stderr?.destroy();
    await Promise.race([
      once(server, "close"),
      delay(1000),
    ]);
  });

  server.once("exit", () => {
    serverExited = true;
  });

  await waitForServer(`http://127.0.0.1:${port}`, () => serverExited);

  const browser = await chromium.launch();
  t.after(async () => {
    await browser.close();
  });

  const page = await browser.newPage();
  await page.goto(`http://127.0.0.1:${port}`);
  await page.getByLabel("Actor").fill("actor-unicode-e2e");
  await page.getByLabel("Document").fill("unicode-e2e-doc");
  await page.getByRole("button", { name: "Connect", exact: true }).click();
  await page.locator("#editor").fill("âââ");

  await page.waitForFunction(() => {
    const status = document.getElementById("status")?.textContent?.trim();
    const error = document.getElementById("error")?.textContent?.trim();
    const text = document.getElementById("editor")?.value ?? "";
    return text === "âââ" && status !== "connecting" && error === "none";
  }, null, { timeout: 5000 });

  await page.locator("#editor").fill("âââb");
  await page.waitForFunction(() => {
    const status = document.getElementById("status")?.textContent?.trim();
    const error = document.getElementById("error")?.textContent?.trim();
    const text = document.getElementById("editor")?.value ?? "";
    return text === "âââb" && status === "live" && error === "none";
  }, null, { timeout: 5000 });

  const status = await page.locator("#status").textContent();
  const error = await page.locator("#error").textContent();
  const text = await page.locator("#editor").inputValue();

  assert.equal(text, "âââb");
  assert.equal(status?.trim(), "live");
  assert.equal(error?.trim(), "none");
});

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
