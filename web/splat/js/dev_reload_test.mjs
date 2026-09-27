import test from "node:test";
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { createContext, runInContext } from "node:vm";

const source = readFileSync(new URL("./dev_reload.mjs", import.meta.url), "utf8");
const flush = () => new Promise(resolve => setImmediate(resolve));

async function client(recovering = false) {
  const events = new Map();
  const state = { version: "build-1", ok: true, reloads: 0, activeTimers: 0, calls: 0, fail: false };
  const document = {
    documentElement: { hasAttribute: name => recovering && name === "data-dev-recover" },
    visibilityState: "visible",
    addEventListener: (name, fn) => events.set(name, fn),
  };
  const context = createContext({
    document,
    window: { location: { reload: () => state.reloads++ }, addEventListener: (name, fn) => events.set(name, fn) },
    AbortSignal: { timeout: ms => { assert.equal(ms, 3000); return "bounded"; } },
    setInterval: (_fn, ms) => { assert.equal(ms, 2000); state.activeTimers++; return 1; },
    clearInterval: () => state.activeTimers--,
    fetch: async (url, options) => {
      state.calls++;
      assert.equal(url, "/__dev/version");
      assert.equal(options.cache, "no-store");
      assert.equal(options.signal, "bounded");
      if (state.fail) throw new Error("offline");
      return { ok: state.ok, text: async () => state.version };
    },
  });
  runInContext(source, context);
  await flush();
  return { state, events, document, check: () => runInContext("checkBuild()", context) };
}

test("loaded page refreshes once a different healthy version is available", async () => {
  const { state, check } = await client();
  assert.equal(state.reloads, 0);
  state.version = "build-2";
  state.ok = false;
  await check();
  assert.equal(state.reloads, 0);
  state.ok = true;
  await check();
  assert.equal(state.reloads, 1);
  await check();
  assert.equal(state.reloads, 1);
});

test("startup page reloads on its first healthy version", async () => {
  const { state } = await client(true);
  assert.equal(state.reloads, 1);
});

test("failed request releases the polling guard for retry", async () => {
  const { state, check } = await client();
  state.fail = true;
  await check();
  assert.equal(state.reloads, 0);
  state.fail = false;
  state.version = "build-2";
  await check();
  assert.equal(state.reloads, 1);
});

test("returning from page cache restarts polling without duplicate intervals", async () => {
  const { state, events } = await client();
  events.get("pagehide")();
  assert.equal(state.activeTimers, 0);
  state.version = "build-2";
  events.get("pageshow")();
  await flush();
  assert.equal(state.activeTimers, 1);
  assert.equal(state.reloads, 1);
  events.get("pageshow")();
  await flush();
  assert.equal(state.activeTimers, 1);
});

test("returning to a background tab checks the current build immediately", async () => {
  const { state, events, document } = await client();
  document.visibilityState = "hidden";
  state.version = "build-2";
  events.get("visibilitychange")();
  await flush();
  assert.equal(state.reloads, 0);
  document.visibilityState = "visible";
  events.get("visibilitychange")();
  await flush();
  assert.equal(state.reloads, 1);
});
