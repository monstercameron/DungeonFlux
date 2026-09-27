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
  set src(value) { this.attrs.src = value; }
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
    destroy() {}
  }
  class Mesh { setPositions() {} setNormals() {} setUvs() {} setIndices() {} update() {} destroy() {} }
  class Material { setParameter() {} update() {} destroy() {} }
  class Texture { setSource() {} upload() {} destroy() {} }
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

test("clip sprite shows fallback before decode and switches only after loadeddata", () => {
  const video = new FakeVideo();
  const { pc, app, entities } = fakePlayCanvas(video);
  const sprite = createTokenSprite({ pc, app, token: { id: "hero", kind: "player", anim: "idle", clips: { idle: "idle.mp4" } } });
  const fallback = entities.find((entity) => entity.name?.startsWith("df-token-sprite"));
  const clip = entities.find((entity) => entity.name?.startsWith("df-token-clip"));
  assert.equal(fallback.enabled, true);
  assert.equal(clip.enabled, false);
  video.readyState = 2; video.dispatch("loadeddata");
  assert.equal(fallback.enabled, false);
  assert.equal(clip.enabled, true);
  sprite.setClips({ idle: "replacement.mp4" });
  assert.equal(fallback.enabled, true);
  assert.equal(clip.enabled, false);
  video.dispatch("error");
  assert.equal(fallback.enabled, true);
  sprite.setClips({ idle: "recovery.mp4" });
  video.readyState = 2; video.dispatch("loadeddata");
  assert.equal(fallback.enabled, false);
  assert.equal(clip.enabled, true);
  sprite.destroy();
});
