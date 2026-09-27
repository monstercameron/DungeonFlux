import test from "node:test";
import assert from "node:assert/strict";
import { createTokenSprite, createVideoGate } from "./token_sprite.mjs";

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

class FakeVideo {
  constructor() { this.readyState = 0; this.listeners = {}; this.attrs = {}; this.paused = true; }
  addEventListener(name, handler) { (this.listeners[name] ??= []).push(handler); }
  dispatch(name) { for (const handler of this.listeners[name] ?? []) handler(); }
  set src(value) { this.attrs.src = value; this.readyState = 0; }
  getAttribute(name) { return this.attrs[name] ?? null; }
  removeAttribute(name) { delete this.attrs[name]; }
  play() { this.paused = false; return Promise.resolve(); }
  pause() { this.paused = true; }
  load() {}
}

function fakePlayCanvas(video) {
  const entities = [];
  class Entity {
    constructor(name) { this.name = name; this.enabled = true; this.position = { x: 0, y: 0, z: 0 }; entities.push(this); }
    addComponent() {}
    setLocalScale() {}
    setLocalPosition(x, y, z) { this.setPosition(x, y, z); }
    setPosition(x, y, z) { this.position = { x, y, z }; }
    getPosition() { return this.position; }
    setEulerAngles() {}
    destroy() { this.destroyed = true; this.enabled = false; }
  }
  class Mesh { setPositions() {} setNormals() {} setUvs() {} setIndices() {} update() {} destroy() {} }
  class Material { setParameter() {} update() {} destroy() {} }
  class Texture { constructor() { this.uploads = 0; } setSource() {} upload() { this.uploads++; } destroy() {} }
  class MeshInstance { constructor() {} }
  const pc = { Mesh, Entity, Texture, MeshInstance, StandardMaterial: Material, ShaderMaterial: Material, Color: class Color {},
    PIXELFORMAT_RGBA8: 1, FILTER_NEAREST: 1, FILTER_LINEAR: 1, ADDRESS_CLAMP_TO_EDGE: 1,
    CULLFACE_NONE: 0, BLEND_NONE: 0, BLEND_NORMAL: 1, SEMANTIC_POSITION: 0, SEMANTIC_TEXCOORD0: 1,
  };
  const app = { graphicsDevice: {}, root: { addChild() {} } };
  globalThis.document = { createElement(kind) {
    if (kind === "video") return video;
    return { width: 0, height: 0, getContext: () => ({ clearRect() {}, fillRect() {}, translate() {}, fillStyle: "", imageSmoothingEnabled: false }) };
  } };
  return { pc, app, entities };
}

test("clip sprite preserves visibility, feet and decoded frames through load, error and recovery", (t) => {
  const previousDocument = globalThis.document;
  t.after(() => { if (previousDocument === undefined) delete globalThis.document; else globalThis.document = previousDocument; });
  const video = new FakeVideo();
  const { pc, app, entities } = fakePlayCanvas(video);
  const sprite = createTokenSprite({ pc, app, token: { id: "hero", kind: "player", height_m: 1.8, anim: "idle", clips: { idle: "idle.mp4" } } });
  const fallback = entities.find((entity) => entity.name?.startsWith("df-token-sprite"));
  const clip = entities.find((entity) => entity.name?.startsWith("df-token-clip"));
  const bases = entities.filter((entity) => entity.name === "df-token-base");
  assert.equal(bases.length, 1, "one shared base rather than overlapping discs");
  assert.equal(fallback.enabled, true);
  assert.equal(clip.enabled, false);
  sprite.entity.setPosition(3, 2 + sprite.height / 2, 5);
  sprite.faceCamera(null);
  assert.equal(fallback.position.y - 1.8 / 2, 2, "fallback feet stay on the same ground");
  assert.equal(bases[0].position.y, 2.015);
  sprite.entity.enabled = false;
  video.readyState = 2; video.dispatch("loadeddata");
  assert.equal(sprite.entity.enabled, false);
  assert.equal(fallback.enabled, false);
  assert.equal(clip.enabled, false, "decode must not resurrect a hidden token");
  assert.equal(bases[0].enabled, false);
  assert.ok(sprite.texture.uploads > 0, "upload first decoded frame before exposing texture");
  sprite.entity.enabled = true;
  assert.equal(sprite.entity.enabled, true);
  assert.equal(fallback.enabled, false);
  assert.equal(clip.enabled, true);
  assert.equal(bases[0].enabled, true);
  sprite.setClips({ clips: { idle: "replacement.mp4" } });
  assert.equal(video.getAttribute("src"), "replacement.mp4");
  assert.equal(fallback.enabled, true);
  assert.equal(clip.enabled, false);
  video.dispatch("error");
  video.readyState = 2; video.dispatch("canplay");
  assert.equal(fallback.enabled, true, "failed clip cannot expose a late decoded event");
  assert.equal(clip.enabled, false);
  sprite.setClips({ clips: { idle: "recovery.mp4" } });
  assert.equal(video.getAttribute("src"), "recovery.mp4");
  video.readyState = 2; video.dispatch("canplay");
  assert.equal(fallback.enabled, false);
  assert.equal(clip.enabled, true, "a new decoded source recovers after failure");
  sprite.setPaused(true);
  assert.equal(video.paused, true);
  sprite.setPaused(false);
  assert.equal(video.paused, false);
  sprite.destroy();
  const uploads = sprite.texture.uploads;
  video.dispatch("loadeddata"); video.dispatch("ended"); sprite.play("idle");
  sprite.setClips({ clips: { idle: "late.mp4" } }); sprite.tick();
  assert.equal(sprite.texture.uploads, uploads, "destroyed sprite ignores late media events");
  assert.ok(entities.every((entity) => entity.destroyed), "both quads and shared base are released");
  assert.equal(video.getAttribute("src"), null);
});
