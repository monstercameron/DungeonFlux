const MIN_HEIGHT = 0.6;
const MAX_HEIGHT = 3;
const WIDTH_RATIO = 0.72;
const PIXEL_WIDTH = 64;
const PIXEL_HEIGHT = 80;

const ROLE_ALIASES = Object.freeze({
  player: "player", pc: "player", hero: "player",
  enemy: "enemy", villain: "enemy", thrall: "enemy", monster: "enemy",
  npc: "npc", ally: "npc", neutral: "npc",
});

/** Returns the stable visual role for a token kind. */
export function roleForToken(token) {
  return ROLE_ALIASES[String(token?.kind ?? "").trim().toLowerCase()] ?? "npc";
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

/** createTokenSprite creates an upright alpha-cutout stand-in with its feet on the cell. */
export function createTokenSprite({ pc, app, token, layer } = {}) {
  if (!pc?.Mesh || !pc?.StandardMaterial || !pc?.MeshInstance || !pc?.Texture || !pc?.Entity || !app?.graphicsDevice) {
    throw new TypeError("PlayCanvas and an application are required");
  }
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
  let destroyed = false;
  const faceCamera = (camera) => {
    if (destroyed || !camera) return;
    const at = entity.getPosition(), eye = camera.getPosition();
    const dx = eye.x - at.x, dz = eye.z - at.z;
    if (Math.abs(dx) + Math.abs(dz) > 1e-5) entity.setEulerAngles(0, Math.atan2(dx, dz) * 180 / Math.PI, 0);
  };
  const destroy = () => {
    if (destroyed) return;
    destroyed = true;
    entity.destroy(); mesh.destroy(); texture.destroy(); material.destroy();
  };
  return { entity, height, width, role, material, mesh, texture, canvas, faceCamera, destroy };
}
