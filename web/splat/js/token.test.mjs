import test from "node:test";
import assert from "node:assert/strict";
import { animationRestarted } from "./token.mjs";

test("animationRestarted restarts an attack at the same cell and sequence", () => {
  const idle = { anim: "idle" };
  const attack = { anim: "attack" };
  assert.equal(animationRestarted(idle, attack, 4, 4), true);
  assert.equal(animationRestarted(attack, attack, 4, 4), false);
});

test("animationRestarted accepts a newer movement sequence", () => {
  assert.equal(animationRestarted({ anim: "walk" }, { anim: "walk" }, 4, 5), true);
});

test("animationRestarted ignores stale snapshots", () => {
  assert.equal(animationRestarted({ anim: "attack" }, { anim: "idle" }, 5, 4), false);
});
