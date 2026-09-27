import test from "node:test";
import assert from "node:assert/strict";
import { createVideoGate } from "./token_sprite.mjs";

test("video gate keeps fallback visible until a decoded frame", () => {
  const events = [];
  const gate = createVideoGate({ onClip: () => events.push("clip"), onFallback: () => events.push("fallback") });
  assert.equal(gate.visible(), false);
  gate.decoded();
  assert.equal(gate.visible(), true);
  assert.deepEqual(events, ["clip"]);
});

test("video gate returns to fallback on decode error", () => {
  const gate = createVideoGate();
  gate.decoded();
  gate.failed();
  assert.equal(gate.visible(), false);
});

test("video gate recovers after a new clip resets the failed state", () => {
  const events = [];
  const gate = createVideoGate({ onClip: () => events.push("clip"), onFallback: () => events.push("fallback") });
  gate.decoded(); gate.failed(); gate.reset(); gate.decoded();
  assert.equal(gate.visible(), true);
  assert.deepEqual(events, ["clip", "fallback", "fallback", "clip"]);
});
