const MAX_TOKENS = 64;
const LIFT = 0.025;
const INSET = 0.08;
const PLAYER_COLOR = [0.04, 0.86, 1.0];
const VILLAIN_COLOR = [1.0, 0.09, 0.16];

const VERTEX_GLSL = `
attribute vec3 aPosition;
attribute vec2 aUv0;
uniform mat4 matrix_model;
uniform mat4 matrix_viewProjection;
varying vec2 vUv;
void main(void) {
  vUv = aUv0;
  gl_Position = matrix_viewProjection * matrix_model * vec4(aPosition, 1.0);
}`;

const FRAGMENT_GLSL = `
precision highp float;
uniform vec3 uColor;
uniform float uOpacity;
varying vec2 vUv;
void main(void) {
  vec2 edge = min(vUv, 1.0 - vUv);
  float aa = max(fwidth(edge.x), fwidth(edge.y));
  float border = 1.0 - smoothstep(0.035 - aa, 0.035 + aa, min(edge.x, edge.y));
  float glow = 1.0 - smoothstep(0.16, 0.48, min(edge.x, edge.y));
  float alpha = clamp((0.48 + 0.46 * border + 0.10 * glow) * uOpacity, 0.0, 1.0);
  vec3 color = mix(uColor, min(vec3(1.0), uColor + vec3(0.35)), border);
  gl_FragColor = vec4(color, alpha);
}`;

function finite(value, fallback) {
  const result = Number(value);
  return Number.isFinite(result) ? result : fallback;
}

function dimensions(grid) {
  const cols = Math.floor(finite(grid?.cols, 0));
  const rows = Math.floor(finite(grid?.rows, 0));
  return { cols: Math.max(0, cols), rows: Math.max(0, rows) };
}

function cell(raw) {
  if (Array.isArray(raw)) return { c: Number(raw[0]), r: Number(raw[1]) };
  return { c: Number(raw?.c), r: Number(raw?.r) };
}

function cellKey(c, r) { return `${c},${r}`; }

function walkableSet(grid) {
  const values = Array.isArray(grid?.walkable) ? grid.walkable : [];
  return new Set(values.map(Number).filter(Number.isInteger));
}

function validCell(grid, candidate, walkable, dims) {
  const value = cell(candidate);
  if (!Number.isInteger(value.c) || !Number.isInteger(value.r)) return null;
  if (value.c < 0 || value.r < 0 || value.c >= dims.cols || value.r >= dims.rows) return null;
  if (!walkable.has(value.r * dims.cols + value.c)) return null;
  return value;
}

function tokenColor(kind) {
  const value = String(kind ?? "").trim().toLowerCase();
  if (["villain", "enemy", "thrall", "monster"].includes(value)) return "villain";
  if (["pc", "player", "hero"].includes(value)) return "player";
  return null;
}

function occupied(grid, tokens) {
  const dims = dimensions(grid);
  const walkable = walkableSet(grid);
  const cells = new Map();
  for (const token of Array.isArray(tokens) ? tokens.slice(0, MAX_TOKENS) : []) {
    const value = validCell(grid, token?.cell, walkable, dims);
    const color = tokenColor(token?.kind);
    if (!value || !color) continue;
    const key = cellKey(value.c, value.r);
    const previous = cells.get(key);
    // A villain wins a mixed cell deterministically, avoiding order-dependent flicker.
    if (!previous || color === "villain") cells.set(key, { ...value, color });
  }
  return [...cells.values()].sort((left, right) => (left.r - right.r) || (left.c - right.c));
}

function cornerHeight(grid, c, r) {
  const { cols } = dimensions(grid);
  const heights = grid?.corner_heights;
  if (!Array.isArray(heights) || heights.length !== (cols + 1) * (dimensions(grid).rows + 1)) return 0;
  return finite(heights[r * (cols + 1) + c], 0);
}

function heightAt(grid, c, r) {
  const x = Math.floor(c);
  const z = Math.floor(r);
  const tx = c - x;
  const tz = r - z;
  return cornerHeight(grid, x, z) * (1 - tx) * (1 - tz)
    + cornerHeight(grid, x + 1, z) * tx * (1 - tz)
    + cornerHeight(grid, x, z + 1) * (1 - tx) * tz
    + cornerHeight(grid, x + 1, z + 1) * tx * tz;
}

/** Builds inset, terrain-conforming quads for occupied cells. */
export function occupiedCellGeometry(grid, cells, options = {}) {
  const dims = dimensions(grid);
  const size = Math.max(0.01, finite(grid?.cell_m, 1.524));
  const origin = Array.isArray(grid?.origin) ? grid.origin : [0, 0];
  const inset = Math.min(size * 0.45, Math.max(0, finite(options.inset, INSET)));
  const lift = finite(options.lift, LIFT);
  const positions = [];
  const uvs = [];
  const indices = [];
  for (const value of Array.isArray(cells) ? cells : []) {
    if (value.c < 0 || value.r < 0 || value.c >= dims.cols || value.r >= dims.rows) continue;
    const x0 = finite(origin[0], 0) + value.c * size + inset;
    const x1 = finite(origin[0], 0) + (value.c + 1) * size - inset;
    const z0 = finite(origin[1], 0) + value.r * size + inset;
    const z1 = finite(origin[1], 0) + (value.r + 1) * size - inset;
    const corners = [[x0, heightAt(grid, value.c + inset / size, value.r + inset / size) + lift, z0],
      [x1, heightAt(grid, value.c + 1 - inset / size, value.r + inset / size) + lift, z0],
      [x1, heightAt(grid, value.c + 1 - inset / size, value.r + 1 - inset / size) + lift, z1],
      [x0, heightAt(grid, value.c + inset / size, value.r + 1 - inset / size) + lift, z1]];
    const base = positions.length / 3;
    for (const point of corners) positions.push(...point);
    uvs.push(0, 0, 1, 0, 1, 1, 0, 1);
    indices.push(base, base + 1, base + 2, base, base + 2, base + 3);
  }
  return { positions, uvs, indices };
}

function material(pc, color, suffix) {
  const result = new pc.ShaderMaterial({
    uniqueName: `df-occupied-cell-${suffix}`,
    vertexGLSL: VERTEX_GLSL,
    fragmentGLSL: FRAGMENT_GLSL,
    attributes: { aPosition: pc.SEMANTIC_POSITION, aUv0: pc.SEMANTIC_TEXCOORD0 },
  });
  result.name = `df-occupied-cell-${suffix}`;
  result.setParameter("uColor", color);
  result.setParameter("uOpacity", 0.9);
  result.blendType = pc.BLEND_NORMAL;
  result.depthTest = true;
  result.depthWrite = false;
  result.cull = pc.CULLFACE_NONE;
  result.update();
  return result;
}

function makeMesh(pc, app, geometry) {
  const mesh = new pc.Mesh(app.graphicsDevice);
  mesh.setPositions(geometry.positions);
  mesh.setUvs(0, geometry.uvs);
  mesh.setIndices(geometry.indices);
  mesh.update();
  return mesh;
}

/** Creates the bounded player/villain occupied-cell renderer. */
export function createOccupiedCells({ pc, app, grid, layer } = {}) {
  if (!pc?.Mesh || !pc?.ShaderMaterial || !pc?.MeshInstance || !pc?.Entity || !app?.graphicsDevice) {
    throw new TypeError("PlayCanvas and an application are required");
  }
  const playerMaterial = material(pc, PLAYER_COLOR, "player");
  const villainMaterial = material(pc, VILLAIN_COLOR, "villain");
  const entity = new pc.Entity("df-occupied-cells");
  const layerId = layer?.id ?? layer;
  entity.addComponent("render", { meshInstances: [], layers: layerId === undefined ? undefined : [layerId] });
  app.root.addChild(entity);
  let meshes = [];
  let signature = "";
  let destroyed = false;
  const rebuild = (cells) => {
    for (const mesh of meshes) mesh.destroy?.();
    meshes = [];
    const groups = { player: [], villain: [] };
    for (const value of cells) groups[value.color].push(value);
    const instances = [];
    for (const kind of ["player", "villain"]) {
      if (!groups[kind].length) continue;
      const mesh = makeMesh(pc, app, occupiedCellGeometry(grid, groups[kind]));
      meshes.push(mesh);
      instances.push(new pc.MeshInstance(mesh, kind === "player" ? playerMaterial : villainMaterial));
    }
    if (entity.render) entity.render.meshInstances = instances;
  };
  const update = (tokens) => {
    if (destroyed) return;
    const cells = occupied(grid, tokens);
    const next = cells.map((value) => `${value.c},${value.r}:${value.color}`).join("|");
    if (next === signature) return;
    signature = next;
    rebuild(cells);
  };
  const setEnabled = (enabled) => { if (!destroyed) entity.enabled = Boolean(enabled); };
  const destroy = () => {
    if (destroyed) return;
    destroyed = true;
    for (const mesh of meshes) mesh.destroy?.();
    meshes = [];
    entity.destroy?.();
    playerMaterial.destroy?.();
    villainMaterial.destroy?.();
  };
  entity.once?.("destroy", () => { if (!destroyed) destroy(); });
  return Object.freeze({ update, setEnabled, destroy, entity, playerMaterial, villainMaterial });
}

export const OCCUPIED_CELL_SHADER = Object.freeze({ vertex: VERTEX_GLSL, fragment: FRAGMENT_GLSL });
