import { createChromaMaterial } from "./billboard.mjs";

const MIN_HEIGHT = 0.6;
const MAX_HEIGHT = 3;
const WIDTH_RATIO = 0.72;
const PIXEL_WIDTH = 64;
const PIXEL_HEIGHT = 80;

// Token base (DM-038 / R1-COMBAT): a soft ring underfoot so a pixel stand-in
// or a not-yet-arrived clip still reads as a placed combatant rather than a
// floating block. Gold for player and neutral tokens, blood for the enemy,
// matching the lamplight/blood palette (assets/concept, AGENTS.md).
const BASE_GOLD = [0.906, 0.761, 0.478];
const BASE_BLOOD = [0.702, 0.216, 0.184];
const BASE_VERTEX = `
attribute vec3 aPosition;
attribute vec2 aUv0;
uniform mat4 matrix_model;
uniform mat4 matrix_viewProjection;
varying vec2 vUv0;
void main(void) {
  vUv0 = aUv0;
  gl_Position = matrix_viewProjection * matrix_model * vec4(aPosition, 1.0);
}`;
const BASE_FRAGMENT = `
precision highp float;
uniform vec3 uColor;
varying vec2 vUv0;
void main(void) {
  float dist = length(vUv0 - vec2(0.5)) * 2.0;
  float ring = 1.0 - smoothstep(0.62, 0.82, dist);
  float ringEdge = smoothstep(0.5, 0.62, dist) * (1.0 - smoothstep(0.82, 0.92, dist));
  float fill = (1.0 - smoothstep(0.0, 0.6, dist)) * 0.22;
  float alpha = clamp(max(fill, ringEdge * 0.85), 0.0, 0.85) * ring + ringEdge * 0.85;
  if (dist > 0.92) discard;
  gl_FragColor = vec4(uColor, clamp(alpha, 0.0, 0.85) * 0.5);
}`;

function makeBaseMesh(pc, app) {
  const mesh = new pc.Mesh(app.graphicsDevice);
  mesh.setPositions([-0.5, 0, -0.5, 0.5, 0, -0.5, 0.5, 0, 0.5, -0.5, 0, 0.5]);
  mesh.setNormals([0, 1, 0, 0, 1, 0, 0, 1, 0, 0, 1, 0]);
  mesh.setUvs(0, [0, 0, 1, 0, 1, 1, 0, 1]);
  mesh.setIndices([0, 1, 2, 0, 2, 3]);
  mesh.update();
  return mesh;
}

const BASE_DIAMETER_RATIO = 1.15;
const BASE_LIFT_M = 0.015;

/**
 * createTokenBase makes a flat gold/blood disc that sits under a token's
 * feet. It is its own top-level entity (never a child of the sprite's own
 * non-uniformly scaled, yaw-billboarded entity, which would skew a naive
 * child disc): the sprite's faceCamera keeps it positioned at
 * (sprite x, ground y, sprite z) every time the sprite itself moves, and it
 * carries no rotation of its own, so it always reads as a flat ring on the
 * ground regardless of the sprite's facing.
 */
function createTokenBase(pc, app, { role, layer, diameter }) {
  const mesh = makeBaseMesh(pc, app);
  const material = new pc.ShaderMaterial({
    uniqueName: "df-token-base",
    vertexGLSL: BASE_VERTEX,
    fragmentGLSL: BASE_FRAGMENT,
    attributes: { aPosition: pc.SEMANTIC_POSITION, aUv0: pc.SEMANTIC_TEXCOORD0 },
  });
  material.setParameter("uColor", role === "enemy" ? BASE_BLOOD : BASE_GOLD);
  material.blendType = pc.BLEND_NORMAL;
  material.depthWrite = false;
  material.depthTest = true;
  material.cull = pc.CULLFACE_NONE;
  material.update();
  const instance = new pc.MeshInstance(mesh, material);
  const entity = new pc.Entity("df-token-base");
  entity.addComponent("render", { meshInstances: [instance], layers: [layer ?? pc.LAYERID_WORLD] });
  entity.setLocalScale(diameter, 1, diameter);
  app.root.addChild(entity);
  let destroyed = false;
  return {
    entity, mesh, material,
    /** placeUnder moves the disc to the ground point under a sprite whose feet sit at `groundY`. */
    placeUnder(x, groundY, z) { if (!destroyed) entity.setLocalPosition(x, groundY + BASE_LIFT_M, z); },
    destroy() { if (destroyed) return; destroyed = true; entity.destroy(); mesh.destroy(); material.destroy(); },
  };
}

const ROLE_ALIASES = Object.freeze({
  player: "player", pc: "player", hero: "player",
  enemy: "enemy", villain: "enemy", thrall: "enemy", monster: "enemy",
  npc: "npc", ally: "npc", neutral: "npc",
});

/** Returns the stable visual role for a token kind. */
export function roleForToken(token) {
  const kind = String(token?.kind ?? "").trim().toLowerCase();
  if (kind.startsWith("pc-") || kind.startsWith("player-") || kind.startsWith("hero-")) return "player";
  return ROLE_ALIASES[kind] ?? "npc";
}

function heightOf(token) {
  const value = Number(token?.height_m);
  return Math.max(MIN_HEIGHT, Math.min(MAX_HEIGHT, Number.isFinite(value) && value > 0 ? value : 1.8));
}

function canvasFor() {
  if (typeof document !== "undefined" && document.createElement) return document.createElement("canvas");
  if (typeof OffscreenCanvas !== "undefined") return new OffscreenCanvas(PIXEL_WIDTH, PIXEL_HEIGHT);
  throw new TypeError("a canvas implementation is required to draw token sprites");
}

function rect(ctx, color, x, y, width, height) {
  ctx.fillStyle = color; ctx.fillRect(x, y, width, height);
}

function drawPlayer(ctx) {
  rect(ctx, "#0a1728", 21, 12, 22, 18); rect(ctx, "#f6c58b", 25, 17, 14, 13);
  rect(ctx, "#08c9e8", 17, 29, 30, 30); rect(ctx, "#b9fbff", 24, 32, 16, 6);
  rect(ctx, "#05728d", 20, 43, 8, 13); rect(ctx, "#07516d", 36, 43, 8, 13);
  rect(ctx, "#d7f9ff", 8, 32, 11, 19); rect(ctx, "#6da3b1", 9, 35, 9, 4);
  rect(ctx, "#0a1728", 45, 18, 8, 53); rect(ctx, "#d7f9ff", 47, 19, 4, 36);
  rect(ctx, "#ffd34e", 43, 54, 12, 4); rect(ctx, "#875329", 47, 58, 4, 12);
  rect(ctx, "#0b243c", 20, 57, 10, 26); rect(ctx, "#0b243c", 34, 57, 10, 26);
  rect(ctx, "#49d8eb", 18, 82, 14, 6); rect(ctx, "#49d8eb", 32, 82, 14, 6);
  rect(ctx, "#ffffff", 28, 21, 3, 3); rect(ctx, "#ffffff", 35, 21, 3, 3);
}

function drawEnemy(ctx) {
  rect(ctx, "#270f1d", 18, 10, 10, 18); rect(ctx, "#270f1d", 36, 10, 10, 18);
  rect(ctx, "#e7c59b", 19, 10, 6, 10); rect(ctx, "#e7c59b", 39, 10, 6, 10);
  rect(ctx, "#9d1d2f", 20, 16, 24, 18); rect(ctx, "#f08b62", 25, 20, 14, 13);
  rect(ctx, "#5d1022", 14, 29, 36, 31); rect(ctx, "#c82d3f", 20, 34, 24, 13);
  rect(ctx, "#ffb22e", 26, 24, 4, 3); rect(ctx, "#ffb22e", 35, 24, 4, 3);
  rect(ctx, "#ed4550", 8, 36, 10, 17); rect(ctx, "#ed4550", 46, 36, 10, 17);
  rect(ctx, "#3b122b", 19, 57, 11, 27); rect(ctx, "#3b122b", 34, 57, 11, 27);
  rect(ctx, "#d83e49", 17, 82, 15, 6); rect(ctx, "#d83e49", 32, 82, 15, 6);
  rect(ctx, "#ff5964", 29, 42, 6, 4);
}

function drawNpc(ctx) {
  rect(ctx, "#5a3217", 22, 12, 20, 20); rect(ctx, "#f2b36d", 26, 18, 12, 13);
  rect(ctx, "#b96b22", 15, 28, 34, 39); rect(ctx, "#ffc04a", 21, 35, 22, 16);
  rect(ctx, "#f7dd8a", 9, 35, 8, 24); rect(ctx, "#f7dd8a", 47, 35, 8, 24);
  rect(ctx, "#613819", 19, 65, 12, 19); rect(ctx, "#613819", 33, 65, 12, 19);
  rect(ctx, "#f0b941", 17, 82, 15, 6); rect(ctx, "#f0b941", 32, 82, 15, 6);
  rect(ctx, "#6739b7", 47, 13, 4, 72); rect(ctx, "#e2a8ff", 43, 10, 12, 7);
  rect(ctx, "#3d2416", 29, 22, 3, 3); rect(ctx, "#3d2416", 35, 22, 3, 3);
}

function drawTexture(role) {
  const canvas = canvasFor(); canvas.width = PIXEL_WIDTH; canvas.height = PIXEL_HEIGHT;
  const ctx = canvas.getContext("2d");
  ctx.clearRect(0, 0, PIXEL_WIDTH, PIXEL_HEIGHT); ctx.imageSmoothingEnabled = false; ctx.translate(0, -8);
  if (role === "player") drawPlayer(ctx); else if (role === "enemy") drawEnemy(ctx); else drawNpc(ctx);
  return canvas;
}

function makeMesh(pc, app, flip) {
  const mesh = new pc.Mesh(app.graphicsDevice);
  mesh.setPositions([-0.5, -0.5, 0, 0.5, -0.5, 0, 0.5, 0.5, 0, -0.5, 0.5, 0]);
  mesh.setNormals([0,0,1,0,0,1,0,0,1,0,0,1]);
  mesh.setUvs(0, flip ? [1, 0, 0, 0, 0, 1, 1, 1] : [0, 0, 1, 0, 1, 1, 0, 1]); mesh.setIndices([0, 1, 2, 0, 2, 3]); mesh.update();
  return mesh;
}

function makeMaterial(pc, app, texture) {
  const material = new pc.StandardMaterial();
  material.useLighting = false;
  material.emissive = new pc.Color(1, 1, 1);
  material.emissiveMap = texture;
  material.opacityMap = texture;
  material.opacityMapChannel = "a";
  material.alphaTest = 0.5;
  material.blendType = pc.BLEND_NONE;
  material.cull = pc.CULLFACE_NONE;
  material.depthWrite = true;
  material.depthTest = true;
  material.update();
  return material;
}

/** createTokenSprite creates an upright sprite with its feet on the cell: the
 * token's green-screen loop when it has clips, else the pixel stand-in. */
export function createTokenSprite({ pc, app, token, layer } = {}) {
  if (!pc?.Mesh || !pc?.StandardMaterial || !pc?.MeshInstance || !pc?.Texture || !pc?.Entity || !app?.graphicsDevice) {
    throw new TypeError("PlayCanvas and an application are required");
  }
  if (hasClips(token) && typeof document !== "undefined" && document.createElement) return createClipSprite({ pc, app, token, layer });
  const role = roleForToken(token), height = heightOf(token), width = height * WIDTH_RATIO;
  const canvas = drawTexture(role);
  const texture = new pc.Texture(app.graphicsDevice, {
    format: pc.PIXELFORMAT_RGBA8, mipmaps: false, flipY: true,
    minFilter: pc.FILTER_NEAREST, magFilter: pc.FILTER_NEAREST,
    addressU: pc.ADDRESS_CLAMP_TO_EDGE, addressV: pc.ADDRESS_CLAMP_TO_EDGE,
  });
  texture.setSource(canvas);
  const material = makeMaterial(pc, app, texture);
  const mesh = makeMesh(pc, app, Boolean(token?.flip_u));
  const entity = new pc.Entity(`df-token-sprite-${token?.id ?? "token"}`);
  const instance = new pc.MeshInstance(mesh, material, entity);
  entity.addComponent("render", { meshInstances: [instance], layers: [layer ?? pc.LAYERID_WORLD] });
  entity.setLocalScale(width, height, 1);
  entity.setLocalPosition(0, height / 2, 0);
  app.root.addChild(entity);
  const base = createTokenBase(pc, app, { role, layer, diameter: width * BASE_DIAMETER_RATIO });
  base.placeUnder(0, 0, 0);
  let destroyed = false;
  const faceCamera = (camera) => {
    if (destroyed) return;
    const at = entity.getPosition?.();
    if (at) base.placeUnder(at.x, at.y - height / 2, at.z);
    if (!camera || !at) return;
    const eye = camera.getPosition();
    const dx = eye.x - at.x, dz = eye.z - at.z;
    if (Math.abs(dx) + Math.abs(dz) > 1e-5) entity.setEulerAngles(0, Math.atan2(dx, dz) * 180 / Math.PI, 0);
  };
  const destroy = () => {
    if (destroyed) return;
    destroyed = true;
    entity.destroy(); mesh.destroy(); texture.destroy(); material.destroy(); base.destroy();
  };
  return { entity, height, width, role, material, mesh, texture, canvas, faceCamera, destroy, video: false };
}

// Billboard loops are 9:16 (496 x 864) with the character's feet about 8%
// above the bottom edge and the head about 5% below the top.
const CLIP_ASPECT = 496 / 864;
const CLIP_FEET = 0.08;
const CLIP_BODY = 0.87;

/** hasClips reports whether a token carries at least one loop URL. */
export function hasClips(token) {
  return Boolean(token?.clips && Object.values(token.clips).some(value => typeof value === "string" && value));
}

/** clipFor picks the loop for an animation: walk loops while a token moves,
 * attack and hit are one-shots, and fall holds the last frame. */
export function clipFor(token, anim = token?.anim) {
  const clips = token?.clips ?? {};
  const name = String(anim || "idle");
  if (name === "fall") return { url: clips.fall || clips.hit || clips.idle || "", loop: false, hold: true };
  if (name === "walk") return { url: clips.walk || clips.idle || "", loop: true, hold: false };
  if (name === "idle" || name === "flee" || !clips[name]) return { url: clips.idle || "", loop: true, hold: false };
  return { url: clips[name], loop: false, hold: false };
}

function createClipVideo() {
  const video = document.createElement("video");
  video.muted = true; video.playsInline = true; video.preload = "auto"; video.crossOrigin = "anonymous";
  return video;
}

function createClipSprite({ pc, app, token, layer }) {
  const quadHeight = heightOf(token) / CLIP_BODY, width = quadHeight * CLIP_ASPECT;
  const video = createClipVideo();
  const texture = new pc.Texture(app.graphicsDevice, {
    format: pc.PIXELFORMAT_RGBA8, mipmaps: false, flipY: true,
    minFilter: pc.FILTER_LINEAR, magFilter: pc.FILTER_LINEAR,
    addressU: pc.ADDRESS_CLAMP_TO_EDGE, addressV: pc.ADDRESS_CLAMP_TO_EDGE,
  });
  texture.setSource(video);
  const material = createChromaMaterial(app, texture);
  const mesh = makeMesh(pc, app, Boolean(token?.flip_u));
  const entity = new pc.Entity(`df-token-clip-${token?.id ?? "token"}`);
  entity.addComponent("render", { meshInstances: [new pc.MeshInstance(mesh, material, entity)], layers: [layer ?? pc.LAYERID_WORLD] });
  entity.setLocalScale(width, quadHeight, 1);
  app.root.addChild(entity);
  // Reported height excludes the margin under the feet, so the controller's
  // "centre = ground + height / 2" placement puts the feet on the ground.
  const height = quadHeight * (1 - 2 * CLIP_FEET);
  const base = createTokenBase(pc, app, { role: roleForToken(token), layer, diameter: width * BASE_DIAMETER_RATIO });
  base.placeUnder(0, 0, 0);
  let destroyed = false, current = { url: "" }, clips = { ...(token?.clips ?? {}) }, paused = false;
  const play = () => { if (!paused && !destroyed) video.play()?.catch?.(() => {}); };
  const show = (anim) => {
    const next = clipFor({ clips }, anim);
    if (!next.url) return;
    current = { ...next, anim };
    video.loop = next.loop;
    if (video.getAttribute("src") !== next.url) video.src = next.url; else video.currentTime = 0;
    play();
  };
  video.addEventListener("ended", () => { if (!current.hold) show("idle"); });
  show(token?.anim);
  const faceCamera = (camera) => {
    if (destroyed) return;
    const at = entity.getPosition?.();
    if (at) base.placeUnder(at.x, at.y - height / 2, at.z);
    if (!camera || !at) return;
    const eye = camera.getPosition();
    const dx = eye.x - at.x, dz = eye.z - at.z;
    if (Math.abs(dx) + Math.abs(dz) > 1e-5) entity.setEulerAngles(0, Math.atan2(dx, dz) * 180 / Math.PI, 0);
  };
  return {
    entity, height, width, role: roleForToken(token), material, mesh, texture, faceCamera, video: true,
    /** play restarts the one-shot for a new anim_seq. */
    play(anim) { show(anim); },
    /** setClips adopts new loop URLs; the current animation switches only if its URL changed. */
    setClips(next) {
      clips = { ...(next?.clips ?? {}) };
      const wanted = clipFor({ clips }, current.anim ?? next?.anim);
      if (wanted.url && wanted.url !== current.url) show(current.anim ?? next?.anim);
    },
    setPaused(on) { paused = Boolean(on); if (paused) video.pause(); else play(); },
    tick() { if (!destroyed && video.readyState >= 2) texture.upload(); },
    destroy() {
      if (destroyed) return;
      destroyed = true;
      video.pause(); video.removeAttribute("src"); video.load();
      entity.destroy(); mesh.destroy(); texture.destroy(); material.destroy(); base.destroy();
    },
  };
}
