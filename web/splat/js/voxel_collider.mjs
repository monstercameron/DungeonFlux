const UINT24_MASK = 0x00ffffff;
const SOLID_LEAF = 0xff000000;
const LEAF_SIZE = 4;

function vector3(value, name) {
  if (!Array.isArray(value) || value.length < 3 || value.some((item) => !Number.isFinite(Number(item)))) {
    throw new Error(`voxel ${name} must be a finite vec3`);
  }
  return value.slice(0, 3).map(Number);
}

function integer(value, name, minimum = 0) {
  const number = Number(value);
  if (!Number.isInteger(number) || number < minimum) throw new Error(`voxel ${name} is invalid`);
  return number;
}

function normalizedMeta(meta) {
  if (!meta || typeof meta !== "object" || Array.isArray(meta)) throw new Error("voxel metadata must be an object");
  const version = String(meta.version ?? "");
  const major = Number(version.split(".")[0]);
  if (!/^1(?:\.\d+)?$/.test(version) || major !== 1) throw new Error(`unsupported voxel version: ${meta.version ?? "missing"}`);
  const bounds = meta.gridBounds ?? meta.grid_bounds;
  const min = vector3(bounds?.min, "gridBounds.min");
  const max = vector3(bounds?.max, "gridBounds.max");
  if (max.some((item, index) => item <= min[index])) throw new Error("voxel grid bounds are empty");
  const resolution = Number(meta.voxelResolution ?? meta.voxel_resolution);
  if (!Number.isFinite(resolution) || resolution <= 0) throw new Error("voxel resolution is invalid");
  if (Number(meta.leafSize ?? meta.leaf_size ?? LEAF_SIZE) !== LEAF_SIZE) throw new Error("unsupported voxel leaf size");
  const treeDepth = integer(meta.treeDepth ?? meta.tree_depth, "tree depth", 1);
  if (treeDepth > 22) throw new Error("voxel tree depth is too large");
  const nodeCount = integer(meta.nodeCount ?? meta.node_count, "node count");
  const leafDataCount = integer(meta.leafDataCount ?? meta.leaf_data_count, "leaf data count");
  if (nodeCount > 16_777_216 || leafDataCount % 2 !== 0) throw new Error("voxel node counts are invalid");
  return { min, max, resolution, treeDepth, nodeCount, leafDataCount, version };
}

function wordsFromBytes(bytes) {
  const source = bytes instanceof ArrayBuffer ? new Uint8Array(bytes) : bytes;
  if (!(source instanceof Uint8Array) || source.byteLength % 4 !== 0) throw new Error("voxel binary is not aligned");
  const words = new Uint32Array(source.byteLength / 4);
  const view = new DataView(source.buffer, source.byteOffset, source.byteLength);
  for (let index = 0; index < words.length; index += 1) words[index] = view.getUint32(index * 4, true);
  return words;
}

function normalizedTransform(transform = {}) {
  const scale = Number(transform.scale ?? 1);
  const translate = Array.isArray(transform.translate) ? transform.translate : [0, 0, 0];
  if (!Number.isFinite(scale) || scale <= 0 || translate.length < 3 || translate.some((item) => !Number.isFinite(Number(item)))) {
    throw new Error("voxel world transform is invalid");
  }
  const offsetY = Number(transform.offset_y ?? 0);
  const rotY = Number(transform.rot_y_deg ?? 0);
  if (!Number.isFinite(offsetY) || !Number.isFinite(rotY)) throw new Error("voxel world transform rotation is invalid");
  return { scale, translate: translate.slice(0, 3).map(Number), offsetY, rotY };
}

function inverseWorldTransform(point, transform) {
  const x = point[0] - transform.translate[0];
  const y = point[1] - transform.translate[1] - transform.offsetY;
  const z = point[2] - transform.translate[2];
  const angle = transform.rotY * Math.PI / 180;
  const c = Math.cos(angle); const s = Math.sin(angle);
  return [(x * c - z * s) / transform.scale, y / transform.scale, (x * s + z * c) / transform.scale];
}

function sourceToEngine(point) {
  return [-point[0], -point[1], point[2]];
}

function makeCollider(meta, words, options) {
  const normalized = normalizedMeta(meta);
  const expected = normalized.nodeCount + normalized.leafDataCount;
  if (words.length !== expected) throw new Error(`voxel binary length mismatch: expected ${expected} words, got ${words.length}`);
  validateTree(normalized, words);
  const transform = normalizedTransform(options.transform ?? {});
  const sourceFrame = normalized.version.startsWith("1.0") || meta.coordinateFrame === "source" || meta.coordinate_frame === "source";
  const toLocal = (point) => {
    const local = inverseWorldTransform(point, transform);
    return sourceFrame ? sourceToEngine(local) : local;
  };
  const voxelAt = (point) => lookupVoxel(normalized, words, toLocal(point));
  const intersectsBox = (min, max) => {
    const corners = [];
    for (const x of [min[0], max[0]]) for (const y of [min[1], max[1]]) for (const z of [min[2], max[2]]) corners.push(toLocal([x, y, z]));
    const localMin = [0, 1, 2].map((axis) => Math.min(...corners.map((corner) => corner[axis])));
    const localMax = [0, 1, 2].map((axis) => Math.max(...corners.map((corner) => corner[axis])));
    return intersectsLocalBox(normalized, words, localMin, localMax);
  };
  return { meta: normalized, transform, voxelAt, intersectsBox, filterGrid: (grid, filterOptions = {}) => filterGrid(grid, intersectsBox, filterOptions) };
}

function validateTree(meta, words) {
  const pending = [[0, 0]];
  const seen = new Set();
  while (pending.length) {
    const [node, depth] = pending.pop();
    if (node >= meta.nodeCount) throw new Error("voxel child pointer is invalid");
    if (seen.has(node)) throw new Error("voxel tree contains a repeated node");
    seen.add(node);
    const word = words[node];
    if (word === SOLID_LEAF) continue;
    if (depth >= meta.treeDepth) {
      if (word >>> 24) throw new Error("voxel leaf word is invalid");
      const leafIndex = word & UINT24_MASK;
      if ((leafIndex * 2) + 1 >= meta.leafDataCount) throw new Error("voxel mixed leaf index is invalid");
      continue;
    }
    const mask = word >>> 24;
    if (!mask) throw new Error("voxel interior node has no children");
    const first = word & UINT24_MASK;
    const children = popcount(mask);
    if (first + children > meta.nodeCount) throw new Error("voxel child pointer is invalid");
    for (let octant = 0, offset = 0; octant < 8; octant += 1) {
      if (mask & (1 << octant)) pending.push([first + offset++, depth + 1]);
    }
  }
}

function lookupVoxel(meta, words, point) {
  if (point.some((value, index) => value < meta.min[index] || value >= meta.max[index])) return true;
  const voxel = point.map((value, index) => Math.floor((value - meta.min[index]) / meta.resolution));
  const side = LEAF_SIZE * (2 ** meta.treeDepth);
  if (voxel.some((value) => value < 0 || value >= side)) return true;
  let node = 0;
  for (let depth = 0; depth < meta.treeDepth; depth += 1) {
    const word = words[node];
    if (word === SOLID_LEAF) return true;
    const mask = word >>> 24;
    const shift = meta.treeDepth - depth - 1;
    const octant = ((voxel[0] >> (shift + 2)) & 1) | (((voxel[1] >> (shift + 2)) & 1) << 1) | (((voxel[2] >> (shift + 2)) & 1) << 2);
    if (!(mask & (1 << octant))) return false;
    const before = mask & ((1 << octant) - 1);
    node = (word & UINT24_MASK) + popcount(before);
  }
  const leaf = words[node];
  if (leaf === SOLID_LEAF) return true;
  if (leaf >>> 24) throw new Error("voxel leaf word is invalid");
  const leafIndex = leaf & UINT24_MASK;
  const local = voxel.map((value) => value % LEAF_SIZE);
  const bit = local[0] + (local[1] << 2) + (local[2] << 4);
  const dataWord = words[meta.nodeCount + (leafIndex * 2) + Math.floor(bit / 32)];
  if (!Number.isFinite(dataWord)) throw new Error("voxel mixed leaf index is invalid");
  return Boolean(dataWord & (1 << (bit % 32)));
}

function intersectsLocalBox(meta, words, min, max) {
  for (let axis = 0; axis < 3; axis += 1) {
    if (min[axis] < meta.min[axis] || max[axis] > meta.max[axis] || max[axis] <= meta.min[axis] || min[axis] >= meta.max[axis]) return true;
  }
  const lower = min.map((value, axis) => Math.max(0, Math.floor((value - meta.min[axis]) / meta.resolution)));
  const upper = max.map((value, axis) => Math.min(Math.ceil((meta.max[axis] - meta.min[axis]) / meta.resolution) - 1, Math.floor((value - meta.min[axis] - Number.EPSILON) / meta.resolution)));
  if (upper.some((value, axis) => value < lower[axis])) return true;
  for (let x = lower[0]; x <= upper[0]; x += 1) for (let y = lower[1]; y <= upper[1]; y += 1) for (let z = lower[2]; z <= upper[2]; z += 1) {
    const point = meta.min.map((value, axis) => value + (axis === 0 ? x : axis === 1 ? y : z) * meta.resolution + meta.resolution / 2);
    if (lookupVoxel(meta, words, point)) return true;
  }
  return false;
}

function popcount(value) {
  let count = 0;
  for (let bits = value >>> 0; bits; bits >>>= 1) count += bits & 1;
  return count;
}

function cellBlocked(grid, index, intersectsBox, options) {
  const inset = Math.min(options.insetMargin, grid.cell_m / 2);
  const x = Number(grid.origin[0]) + (index % grid.cols) * grid.cell_m;
  const z = Number(grid.origin[1]) + Math.floor(index / grid.cols) * grid.cell_m;
  return intersectsBox([x + inset, options.floorY + options.stepClearance, z + inset], [x + grid.cell_m - inset, options.floorY + options.agentHeight, z + grid.cell_m - inset]);
}

function filterGrid(grid, voxelAt, rawOptions = {}) {
  if (!grid || !Array.isArray(grid.walkable) || !Number.isInteger(grid.cols) || !Number.isInteger(grid.rows) || !(grid.cell_m > 0)) throw new Error("voxel grid is invalid");
  if (grid.cols <= 0 || grid.rows <= 0 || !Array.isArray(grid.origin) || grid.origin.length < 2 || grid.origin.slice(0, 2).some((value) => !Number.isFinite(Number(value))) || grid.walkable.some((index) => !Number.isInteger(index) || index < 0 || index >= grid.cols * grid.rows) || new Set(grid.walkable).size !== grid.walkable.length) throw new Error("voxel grid cells are invalid");
  const agentHeight = Number(rawOptions.agentHeight ?? rawOptions.height ?? 1.8);
  const stepClearance = Number(rawOptions.stepClearance ?? rawOptions.step_height ?? 0.3);
  const insetMargin = Number(rawOptions.insetMargin ?? rawOptions.inset ?? 0.15);
  const floorY = Number(rawOptions.floorY ?? rawOptions.floor_y ?? 0);
  if (!(agentHeight > stepClearance) || stepClearance < 0 || insetMargin < 0 || insetMargin >= grid.cell_m / 2 || !Number.isFinite(agentHeight + stepClearance + insetMargin + floorY)) throw new Error("voxel collision dimensions are invalid");
  const options = { agentHeight, stepClearance, insetMargin, floorY };
  const excluded = grid.walkable.filter((index) => cellBlocked(grid, index, voxelAt, options));
  const removed = new Set(excluded);
  return { ...grid, walkable: grid.walkable.filter((index) => !removed.has(index)), excluded, authoredWalkable: grid.walkable.length };
}

/** Parses a validated PlayCanvas voxel metadata object and little-endian binary payload. */
export function parseVoxelCollider(meta, bytes, options = {}) {
  return makeCollider(meta, wordsFromBytes(bytes), options);
}

/** Loads an explicit .voxel.json manifest and its adjacent .voxel.bin payload. */
export async function loadVoxelCollider(metaURL, options = {}) {
  if (!metaURL || !/\.voxel\.json(?:$|[?#])/i.test(metaURL)) throw new Error("voxel collider URL must end in .voxel.json");
  const fetcher = options.fetcher ?? fetch;
  const response = await fetcher(metaURL);
  if (!response.ok) throw new Error(`voxel metadata HTTP ${response.status}`);
  const meta = await response.json();
  const binaryURL = options.binaryURL ?? meta.binaryURL ?? meta.binary_url ?? metaURL.replace(/\.voxel\.json(?=([?#]|$))/i, ".voxel.bin");
  const binary = await fetcher(new URL(binaryURL, metaURL).href);
  if (!binary.ok) throw new Error(`voxel binary HTTP ${binary.status}`);
  return parseVoxelCollider(meta, await binary.arrayBuffer(), options);
}
