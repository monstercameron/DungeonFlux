import test from "node:test";
import assert from "node:assert/strict";
import { animationRestarted, createTokenController } from "./token.mjs";

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

function harness(options = {}) {
  const plays = [];
  const grid = { cols: 3, rows: 1, walkable: [0, 1, 2], cell_m: 1, origin: [0, 0] };
  const spriteFactory = () => ({
    entity: { enabled: true, setPosition() {} }, height: 1, material: {}, video: true,
    faceCamera() {}, setClips() {}, play(anim) { plays.push(anim); }, setPaused() {}, destroy() {},
  });
  const controller = createTokenController({ grid, spriteFactory, ...options,
    occupiedCellsFactory: () => ({ update() {}, setEnabled() {}, destroy() {} }),
  });
  return { controller, plays };
}

test("controller plays a same-cell attack once and does not replay identical snapshots", () => {
  const { controller, plays } = harness();
  const base = { id: "hero", kind: "player", cell: [0, 0], anim_seq: 2, anim: "idle", clips: { idle: "idle" } };
  controller.acceptToken(base);
  controller.acceptToken({ ...base, anim: "attack" });
  controller.acceptToken({ ...base, anim: "attack" });
  assert.deepEqual(plays, ["idle", "attack"]);
});

test("controller settles a reduced-motion walk on idle", () => {
  const { controller, plays } = harness({ reducedMotion: true });
  controller.acceptToken({ id: "hero", kind: "player", cell: [0, 0], anim_seq: 1, anim: "walk", clips: { idle: "idle", walk: "walk" } });
  assert.deepEqual(plays, ["walk", "idle"]);
});

test("controller restores idle when a walk route arrives", () => {
  const { controller, plays } = harness();
  controller.acceptToken({ id: "hero", kind: "player", cell: [0, 0], anim_seq: 1, anim: "idle", clips: { idle: "idle", walk: "walk" } });
  controller.acceptToken({ id: "hero", kind: "player", cell: [1, 0], path: [[1, 0]], step_ms: 100, anim_seq: 2, anim: "walk", clips: { idle: "idle", walk: "walk" } });
  controller.update(0.2);
  assert.equal(plays.at(-1), "idle");
});
