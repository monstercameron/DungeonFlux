const DEFAULT_CLEARANCE = 0.04;
const DEFAULT_FLOOR_STEP = 0.2;
const DEFAULT_VOXEL_SIZE = 0.2;

function finite(value, fallback) {
  const result = Number(value);
  return Number.isFinite(result) ? result : fallback;
}

function point(value) {
  return Array.isArray(value) && value.length >= 3
    ? value.slice(0, 3).map((item) => finite(item, 0))
    : null;
}

function boxFrom(value, voxelSize) {
  const min = point(value?.min ?? value?.bounds?.min);
  const max = point(value?.max ?? value?.bounds?.max);
  if (min && max && max.every((item, index) => item >= min[index])) return { min, max };
  const position = point(value?.position ?? value?.origin);
  const size = point(value?.size) ?? [voxelSize, voxelSize, voxelSize];
  if (!position) return null;
  return { min: position, max: position.map((item, index) => item + Math.max(0, size[index])) };
}

function collectBoxes(value, output, options, seen) {
  if (!value || seen.has(value)) return;
  if (typeof value === "object") seen.add(value);
  if (value.occupied === false || value.solid === false) return;
  const box = boxFrom(value, options.voxelSize);
  if (box) output.push(box);
  const children = Array.isArray(value) ? value : value.children ?? value.nodes ?? value.boxes ?? value.occupied;
  if (Array.isArray(children)) for (const child of children) collectBoxes(child, output, options, seen);
}

function gridFloors(box, grid) {
  const floors = grid?.floorYByCell;
  if (!Array.isArray(floors)) return [];
  const origin = Array.isArray(grid.origin) ? grid.origin : [0, 0];
  const cellM = finite(grid.cell_m, 1.524);
  const cols = Math.max(0, Number(grid.cols) || 0);
  const rows = Math.max(0, Number(grid.rows) || 0);
  const firstColumn = Math.max(0, Math.floor((box.min[0] - origin[0]) / cellM));
  const lastColumn = Math.min(cols - 1, Math.ceil((box.max[0] - origin[0]) / cellM) - 1);
  const firstRow = Math.max(0, Math.floor((box.min[2] - origin[1]) / cellM));
  const lastRow = Math.min(rows - 1, Math.ceil((box.max[2] - origin[1]) / cellM) - 1);
  const result = [];
  for (let row = firstRow; row <= lastRow; row += 1) {
    for (let column = firstColumn; column <= lastColumn; column += 1) {
      const rawFloor = floors[row * cols + column];
      const floor = rawFloor === null || rawFloor === undefined ? NaN : finite(rawFloor, NaN);
      if (Number.isFinite(floor)) result.push(floor);
    }
  }
  return result;
}

function isFloorVolume(box, grid, options) {
  const floors = gridFloors(box, grid);
  if (floors.length === 0) return false;
  const nearestFloor = Math.max(...floors);
  return box.max[1] <= nearestFloor + options.clearance + options.floorStep;
}

/** Extracts bounded world-space obstruction boxes from an exposed collider tree. */
export function extractColliderBoxes(source, options = {}) {
  const settings = {
    clearance: finite(options.clearance, DEFAULT_CLEARANCE),
    floorStep: finite(options.floorStep, DEFAULT_FLOOR_STEP),
    maxBoxes: Math.max(1, Math.floor(finite(options.maxBoxes, 4096))),
    voxelSize: finite(options.voxelSize, DEFAULT_VOXEL_SIZE),
  };
  const boxes = [];
  collectBoxes(source, boxes, settings, new Set());
  const grid = options.grid;
  const filtered = grid?.floorYByCell
    ? boxes.filter((box) => !isFloorVolume(box, grid, settings))
    : boxes;
  if (filtered.length <= settings.maxBoxes) return filtered;
  options.onLimit?.({ available: filtered.length, retained: settings.maxBoxes });
  const step = filtered.length / settings.maxBoxes;
  return Array.from({ length: settings.maxBoxes }, (_value, index) => filtered[Math.floor((index + 0.5) * step)]);
}

function addFace(positions, indices, corners) {
  const base = positions.length / 3;
  for (const corner of corners) positions.push(...corner);
  indices.push(base, base + 1, base + 2, base, base + 2, base + 3);
}

/** Builds six-sided depth proxy geometry for obstruction boxes. */
export function occluderGeometry(boxes) {
  const positions = [];
  const indices = [];
  for (const box of boxes) {
    const [x0, y0, z0] = box.min;
    const [x1, y1, z1] = box.max;
    const corners = [[x0, y0, z0], [x1, y0, z0], [x1, y0, z1], [x0, y0, z1],
      [x0, y1, z0], [x1, y1, z0], [x1, y1, z1], [x0, y1, z1]];
    addFace(positions, indices, [corners[0], corners[1], corners[5], corners[4]]);
    addFace(positions, indices, [corners[1], corners[2], corners[6], corners[5]]);
    addFace(positions, indices, [corners[2], corners[3], corners[7], corners[6]]);
    addFace(positions, indices, [corners[3], corners[0], corners[4], corners[7]]);
    addFace(positions, indices, [corners[4], corners[5], corners[6], corners[7]]);
    addFace(positions, indices, [corners[3], corners[2], corners[1], corners[0]]);
  }
  return { positions, indices };
}

function depthMaterial(pc, options) {
  const material = new pc.StandardMaterial();
  material.depthTest = true;
  material.depthWrite = true;
  material.redWrite = false;
  material.greenWrite = false;
  material.blueWrite = false;
  material.alphaWrite = false;
  material.blendType = pc.BLEND_NONE;
  material.cull = options.cull ?? pc.CULLFACE_NONE;
  material.update();
  return material;
}

/** Creates a color-free depth-only PlayCanvas proxy and exposes cleanup. */
export function createVoxelDepthOccluder(pc, app, source, grid, options = {}) {
  if (!pc?.Mesh || !pc?.StandardMaterial || !app?.graphicsDevice) {
    throw new TypeError("PlayCanvas and an application are required");
  }
  const boxes = extractColliderBoxes(source, { ...options, grid });
  const geometry = occluderGeometry(boxes);
  const mesh = new pc.Mesh(app.graphicsDevice);
  mesh.setPositions(geometry.positions);
  mesh.setIndices(geometry.indices);
  mesh.update();
  const material = depthMaterial(pc, options);
  const instance = new pc.MeshInstance(mesh, material);
  const entity = new pc.Entity(options.name ?? "df-voxel-depth-occluder");
  entity.addComponent("render", { meshInstances: [instance], layers: options.layers });
  app.root.addChild(entity);
  return Object.freeze({ boxes, geometry, entity, mesh, material, destroy: () => {
    entity.destroy();
    mesh.destroy?.();
    material.destroy?.();
  } });
}
