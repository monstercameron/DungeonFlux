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
  const sceneBounds = meta.sceneBounds ?? meta.scene_bounds;
  if (max.some((item, index) => item <= min[index])) throw new Error("voxel grid bounds are empty");
  const resolution = Number(meta.voxelResolution ?? meta.voxel_resolution);
  if (!Number.isFinite(resolution) || resolution <= 0) throw new Error("voxel resolution is invalid");
  if (Number(meta.leafSize ?? meta.leaf_size ?? LEAF_SIZE) !== LEAF_SIZE) throw new Error("unsupported voxel leaf size");
  const treeDepth = integer(meta.treeDepth ?? meta.tree_depth, "tree depth", 1);
  if (treeDepth > 22) throw new Error("voxel tree depth is too large");
  const nodeCount = integer(meta.nodeCount ?? meta.node_count, "node count");
  const leafDataCount = integer(meta.leafDataCount ?? meta.leaf_data_count, "leaf data count");
  if (nodeCount > 16_777_216 || leafDataCount % 2 !== 0) throw new Error("voxel node counts are invalid");
  return { min, max, sceneMin: sceneBounds ? vector3(sceneBounds.min, "sceneBounds.min") : min, sceneMax: sceneBounds ? vector3(sceneBounds.max, "sceneBounds.max") : max, resolution, treeDepth, nodeCount, leafDataCount, version };
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
  const toWorld = (point) => {
    const engine = sourceFrame ? sourceToEngine(point) : point;
    const angle = -transform.rotY * Math.PI / 180;
    const c = Math.cos(angle); const s = Math.sin(angle);
    const x = engine[0] * c - engine[2] * s;
    const z = engine[0] * s + engine[2] * c;
    return [x * transform.scale + transform.translate[0], engine[1] * transform.scale + transform.translate[1] + transform.offsetY, z * transform.scale + transform.translate[2]];
  };
  const voxelAt = (point) => lookupVoxel(normalized, words, toLocal(point));
  const intersectsBox = (min, max) => {
    const corners = [];
    for (const x of [min[0], max[0]]) for (const y of [min[1], max[1]]) for (const z of [min[2], max[2]]) corners.push(toLocal([x, y, z]));
    const localMin = [0, 1, 2].map((axis) => Math.min(...corners.map((corner) => corner[axis])));
    const localMax = [0, 1, 2].map((axis) => Math.max(...corners.map((corner) => corner[axis])));
    return intersectsLocalBox(normalized, words, localMin, localMax);
  };
  const floorAt = (point, options = {}) => findFloor(normalized, toLocal, voxelAt, point, { ...options, worldResolution: normalized.resolution * transform.scale });
  // The box list and the filtered grid cost hundreds of milliseconds of main
  // thread on the TV; a collider is immutable, so each is computed once (the
  // TV prewarms them during the hook so combat entry does not stall).
  let boxes = null;
  const grids = new Map();
  const filtered = (grid, filterOptions = {}) => {
    const key = JSON.stringify([grid, filterOptions]);
    if (!grids.has(key)) grids.set(key, filterGrid(grid, intersectsBox, floorAt, boundsInWorld(normalized, toWorld), filterOptions));
    return structuredClone(grids.get(key));
  };
  return { meta: normalized, transform, voxelAt, intersectsBox, floorAt, occupiedBoxes: () => (boxes ??= occupiedBoxes(normalized, words, toWorld)), boundsInWorld: () => boundsInWorld(normalized, toWorld), sceneBoundsInWorld: () => boundsInWorld({ ...normalized, min: normalized.sceneMin, max: normalized.sceneMax }, toWorld), filterGrid: filtered };
}

function boundsInWorld(meta, toWorld) {
  const corners = [];
  for (const x of [meta.min[0], meta.max[0]]) for (const y of [meta.min[1], meta.max[1]]) for (const z of [meta.min[2], meta.max[2]]) corners.push(toWorld([x, y, z]));
  return { min: [0, 1, 2].map((axis) => Math.min(...corners.map((corner) => corner[axis]))), max: [0, 1, 2].map((axis) => Math.max(...corners.map((corner) => corner[axis]))) };
}

function occupiedBoxes(meta, words, toWorld) {
  const boxes = []; const pending = [[0, 0, [0, 0, 0]]];
  while (pending.length) {
    const [node, depth, origin] = pending.pop();
    const word = words[node];
    if (word === SOLID_LEAF) {
      const box = worldBox(meta, toWorld, origin, LEAF_SIZE * (2 ** (meta.treeDepth - depth)));
      if (box) boxes.push(box);
      continue;
    }
    if (depth >= meta.treeDepth) {
      const leafIndex = word & UINT24_MASK;
      const mixed = mixedLeafBounds(meta, words, leafIndex);
      if (mixed) {
        const box = worldBox(meta, toWorld, origin.map((value, axis) => value + mixed.min[axis]), mixed.size);
        if (box) boxes.push(box);
      }
      continue;
    }
    const mask = word >>> 24; const first = word & UINT24_MASK;
    const childSpan = LEAF_SIZE * (2 ** (meta.treeDepth - depth - 1));
    for (let octant = 7, offset = popcount(mask) - 1; octant >= 0; octant -= 1) if (mask & (1 << octant)) {
      pending.push([first + offset--, depth + 1, [origin[0] + (octant & 1 ? childSpan : 0), origin[1] + (octant & 2 ? childSpan : 0), origin[2] + (octant & 4 ? childSpan : 0)]]);
    }
  }
  return boxes;
}

function mixedLeafBounds(meta, words, leafIndex) {
  const base = meta.nodeCount + leafIndex * 2;
  const min = [LEAF_SIZE, LEAF_SIZE, LEAF_SIZE]; const max = [-1, -1, -1];
  for (let bit = 0; bit < 64; bit += 1) if (words[base + Math.floor(bit / 32)] & (1 << (bit % 32))) {
    const point = [bit & 3, (bit >> 2) & 3, (bit >> 4) & 3];
    for (let axis = 0; axis < 3; axis += 1) { min[axis] = Math.min(min[axis], point[axis]); max[axis] = Math.max(max[axis], point[axis]); }
  }
  return max[0] < 0 ? null : { min, size: max.map((value, axis) => value - min[axis] + 1) };
}

function worldBox(meta, toWorld, origin, size) {
  const extents = Array.isArray(size) ? size : [size, size, size];
  const rawMin = meta.min.map((value, axis) => value + origin[axis] * meta.resolution);
  const rawMax = rawMin.map((value, axis) => value + extents[axis] * meta.resolution);
  const localMin = rawMin.map((value, axis) => Math.max(meta.min[axis], value));
  const localMax = rawMax.map((value, axis) => Math.min(meta.max[axis], value));
  if (localMax.some((value, axis) => value <= localMin[axis])) return null;
  const corners = [];
  for (const x of [localMin[0], localMax[0]]) for (const y of [localMin[1], localMax[1]]) for (const z of [localMin[2], localMax[2]]) corners.push(toWorld([x, y, z]));
  return { min: [0, 1, 2].map((axis) => Math.min(...corners.map((corner) => corner[axis]))), max: [0, 1, 2].map((axis) => Math.max(...corners.map((corner) => corner[axis]))) };
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

function scanFloor(meta, toLocal, voxelAt, point, options) {
  const low = Number(options.floorSearchMin ?? options.floorY - 1.5);
  const high = Number(options.floorSearchMax ?? options.floorY + 1.5);
  const step = Math.max(Number(options.worldResolution ?? meta.resolution), 0.01);
  if (!Number.isFinite(low) || !Number.isFinite(high) || high <= low) return null;
  for (let y = high; y >= low; y -= step) {
    const local = toLocal([point[0], y, point[1]]);
    const above = toLocal([point[0], y + step, point[1]]);
    const inside = local.every((value, axis) => value >= meta.min[axis] && value < meta.max[axis]);
    const aboveInside = above.every((value, axis) => value >= meta.min[axis] && value < meta.max[axis]);
    if (inside && aboveInside && !voxelAt([point[0], y + step, point[1]]) && voxelAt([point[0], y, point[1]])) return y + step;
  }
  return null;
}

function findFloor(meta, toLocal, voxelAt, point, options) {
  const radius = Math.max(0, Number(options.supportRadius ?? 0));
  const offsets = [[0, 0], [radius, 0], [-radius, 0], [0, radius], [0, -radius]];
  const floors = offsets.map(([x, z]) => scanFloor(meta, toLocal, voxelAt, [point[0] + x, point[1] + z], options)).filter((floor) => floor !== null);
  return floors.length ? Math.max(...floors) : null;
}

function supportSamples(grid, index, floorAt, options) {
  const inset = Math.min(options.insetMargin, grid.cell_m / 2);
  const x = Number(grid.origin[0]) + (index % grid.cols) * grid.cell_m;
  const z = Number(grid.origin[1]) + Math.floor(index / grid.cols) * grid.cell_m;
  const low = Math.min(inset, grid.cell_m / 2);
  const high = grid.cell_m - low;
  return [[x + low, z + low], [x + high, z + low], [x + low, z + high], [x + high, z + high], [x + grid.cell_m / 2, z + grid.cell_m / 2]].map(([sx, sz]) => floorAt([sx, sz], options));
}

function adaptiveCell(grid, index, intersectsBox, floorAt, options) {
  const floors = supportSamples(grid, index, floorAt, options);
  if (floors.some((floor) => floor === null)) return { blocked: true, floor: null };
  const min = Math.min(...floors); const max = Math.max(...floors);
  if (max - min > options.maxFloorSlope) return { blocked: true, floor: null };
  const floor = max;
  return { blocked: intersectsBoxForFloor(grid, index, intersectsBox, options, floor), floor };
}

function intersectsBoxForFloor(grid, index, intersectsBox, options, floor) {
  const inset = Math.min(options.insetMargin, grid.cell_m / 2);
  const x = Number(grid.origin[0]) + (index % grid.cols) * grid.cell_m;
  const z = Number(grid.origin[1]) + Math.floor(index / grid.cols) * grid.cell_m;
  return intersectsBox([x + inset, floor + options.stepClearance, z + inset], [x + grid.cell_m - inset, floor + options.agentHeight, z + grid.cell_m - inset]);
}

function deriveGrid(grid, bounds, options) {
  // keepAuthoredGrid: the game's engine owns the grid (its cells are the
  // engine's cells), so the collider only measures floors and blocked cells
  // on it instead of re-gridding the whole scan.
  if (!options.allCandidates || options.keepAuthoredGrid) return grid;
  const cell = grid.cell_m;
  const origin = [Math.floor(bounds.min[0] / cell) * cell, Math.floor(bounds.min[2] / cell) * cell];
  const cols = Math.ceil((bounds.max[0] - origin[0]) / cell);
  const rows = Math.ceil((bounds.max[2] - origin[1]) / cell);
  return { ...grid, origin, cols, rows, walkable: Array.from({ length: cols * rows }, (_, index) => index) };
}

function cornerHeights(grid, floorAt, options, floorsByCell, playable) {
  const heights = [];
  for (let row = 0; row <= grid.rows; row += 1) for (let column = 0; column <= grid.cols; column += 1) {
    const x = Number(grid.origin[0]) + column * grid.cell_m;
    const z = Number(grid.origin[1]) + row * grid.cell_m;
    const direct = floorAt([x, z], options);
    const references = adjacentFloors(grid, column, row, floorsByCell, playable);
    const reference = references.length ? references.reduce((sum, value) => sum + value, 0) / references.length : null;
    heights.push(reference !== null && (direct === null || Math.abs(direct - reference) > options.maxFloorSlope) ? reference : direct ?? reference ?? options.floorY);
  }
  return heights;
}

function adjacentFloors(grid, column, row, floorsByCell, playable) {
  const values = [];
  for (const [dc, dr] of [[-1, -1], [0, -1], [-1, 0], [0, 0]]) {
    const c = column + dc; const r = row + dr;
    if (c < 0 || c >= grid.cols || r < 0 || r >= grid.rows) continue;
    const index = r * grid.cols + c;
    if (!playable.has(index)) continue;
    const floor = floorsByCell[index];
    if (floor !== null && Number.isFinite(floor)) values.push(floor);
  }
  return values;
}

function filterGrid(grid, intersectsBox, floorAt, bounds, rawOptions = {}) {
  if (!grid || !Array.isArray(grid.walkable) || !Number.isInteger(grid.cols) || !Number.isInteger(grid.rows) || !(grid.cell_m > 0)) throw new Error("voxel grid is invalid");
  if (grid.cols <= 0 || grid.rows <= 0 || !Array.isArray(grid.origin) || grid.origin.length < 2 || grid.origin.slice(0, 2).some((value) => !Number.isFinite(Number(value))) || grid.walkable.some((index) => !Number.isInteger(index) || index < 0 || index >= grid.cols * grid.rows) || new Set(grid.walkable).size !== grid.walkable.length) throw new Error("voxel grid cells are invalid");
  const agentHeight = Number(rawOptions.agentHeight ?? rawOptions.height ?? 1.8);
  const stepClearance = Number(rawOptions.stepClearance ?? rawOptions.step_height ?? 0.3);
  const insetMargin = Number(rawOptions.insetMargin ?? rawOptions.inset ?? 0.15);
  const floorY = Number(rawOptions.floorY ?? rawOptions.floor_y ?? 0);
  if (!(agentHeight > stepClearance) || stepClearance < 0 || insetMargin < 0 || insetMargin >= grid.cell_m / 2 || !Number.isFinite(agentHeight + stepClearance + insetMargin + floorY)) throw new Error("voxel collision dimensions are invalid");
  const options = { agentHeight, stepClearance, insetMargin, floorY, floorSearchMin: rawOptions.floorSearchMin ?? rawOptions.floor_search_min, floorSearchMax: rawOptions.floorSearchMax ?? rawOptions.floor_search_max, supportRadius: Number(rawOptions.supportRadius ?? rawOptions.support_radius ?? 0), maxFloorSlope: Number(rawOptions.maxFloorSlope ?? rawOptions.max_floor_slope ?? 0.45), allCandidates: Boolean(rawOptions.allCandidates ?? rawOptions.all_candidates), keepAuthoredGrid: Boolean(rawOptions.keepAuthoredGrid ?? rawOptions.keep_authored_grid) };
  if (!(options.supportRadius >= 0) || !Number.isFinite(options.supportRadius)) throw new Error("voxel support radius is invalid");
  if (!(options.maxFloorSlope >= 0) || !Number.isFinite(options.maxFloorSlope)) throw new Error("voxel floor slope is invalid");
  const activeGrid = deriveGrid(grid, bounds, options);
  const results = activeGrid.walkable.map((index) => options.allCandidates ? adaptiveCell(activeGrid, index, intersectsBox, floorAt, options) : { blocked: cellBlocked(activeGrid, index, intersectsBox, options), floor: null });
  const excluded = activeGrid.walkable.filter((_, index) => results[index].blocked);
  const unsupported = options.allCandidates ? activeGrid.walkable.filter((_, index) => results[index].floor === null) : [];
  // With the engine's grid kept, the engine decides walkability; the collider
  // only supplies floor heights, so no engine cell is removed here.
  const removed = new Set(options.keepAuthoredGrid ? [] : excluded);
  const floorsByCell = results.map((result) => result.floor);
  const playable = new Set(activeGrid.walkable.filter((index) => !removed.has(index)));
  return { ...activeGrid, walkable: activeGrid.walkable.filter((index) => !removed.has(index)), excluded, excludedUnsupported: unsupported, authoredWalkable: grid.walkable.length, voxel_filtered: true, floorYByCell: floorsByCell, corner_heights: options.allCandidates ? cornerHeights(activeGrid, floorAt, options, floorsByCell, playable) : activeGrid.corner_heights };
}

/** Parses a validated PlayCanvas voxel metadata object and little-endian binary payload. */
export function parseVoxelCollider(meta, bytes, options = {}) {
  return makeCollider(meta, wordsFromBytes(bytes), options);
}

const loadedColliders = new Map();

/**
 * Loads an explicit .voxel.json manifest and its adjacent .voxel.bin payload.
 * Without a custom fetcher the collider is cached per URL and transform, so a
 * battle scene reloaded on the TV (or prewarmed during the hook) reuses it.
 */
export function loadVoxelCollider(metaURL, options = {}) {
  if (options.fetcher) return fetchVoxelCollider(metaURL, options);
  const key = JSON.stringify([metaURL, options.binaryURL ?? "", normalizedTransform(options.transform ?? {})]);
  if (!loadedColliders.has(key)) {
    const pending = fetchVoxelCollider(metaURL, options);
    loadedColliders.set(key, pending);
    pending.catch(() => loadedColliders.delete(key));
  }
  return loadedColliders.get(key);
}

async function fetchVoxelCollider(metaURL, options) {
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
