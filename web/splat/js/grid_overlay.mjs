const DEFAULT_LINE_WIDTH = 0.018;
const DEFAULT_HEIGHT = 0.02;

function number(value, fallback) {
  const result = Number(value);
  return Number.isFinite(result) ? result : fallback;
}

function walkableSet(grid) {
  return new Set(Array.isArray(grid?.walkable) ? grid.walkable : []);
}

function cellIndex(grid, column, row) {
  return row * Number(grid.cols) + column;
}

function addQuad(positions, indices, a, b, width) {
  const dx = b.x - a.x;
  const dz = b.z - a.z;
  const length = Math.hypot(dx, dz) || 1;
  const px = (-dz / length) * width / 2;
  const pz = (dx / length) * width / 2;
  const base = positions.length / 3;
  for (const point of [[a.x + px, a.y, a.z + pz], [b.x + px, b.y, b.z + pz],
    [b.x - px, b.y, b.z - pz], [a.x - px, a.y, a.z - pz]]) {
    positions.push(...point);
  }
  indices.push(base, base + 1, base + 2, base, base + 2, base + 3);
}

function edgeKey(a, b) {
  return a < b ? `${a}:${b}` : `${b}:${a}`;
}

function cornerHeight(grid, column, row, fallback) {
  const cols = Number(grid.cols) || 0;
  const heights = grid.corner_heights;
  if (!Array.isArray(heights) || heights.length !== (cols + 1) * (Number(grid.rows) + 1)) return fallback;
  return number(heights[row * (cols + 1) + column], fallback);
}

function appendCellEdges(positions, indices, seen, grid, column, row, width, lift) {
  const origin = Array.isArray(grid.origin) ? grid.origin : [0, 0];
  const cellM = number(grid.cell_m, 1.524);
  const x = number(origin[0], 0) + column * cellM;
  const z = number(origin[1], 0) + row * cellM;
  const corners = [
    { x, z, y: cornerHeight(grid, column, row, 0) + lift },
    { x: x + cellM, z, y: cornerHeight(grid, column + 1, row, 0) + lift },
    { x: x + cellM, z: z + cellM, y: cornerHeight(grid, column + 1, row + 1, 0) + lift },
    { x, z: z + cellM, y: cornerHeight(grid, column, row + 1, 0) + lift },
  ];
  for (let side = 0; side < 4; side += 1) {
    const next = (side + 1) % 4;
    const cornersBySide = [[column, row], [column + 1, row], [column + 1, row + 1], [column, row + 1]];
    const start = cornersBySide[side];
    const end = cornersBySide[next];
    const key = edgeKey(`${start[0]},${start[1]}`, `${end[0]},${end[1]}`);
    if (!seen.has(key)) {
      seen.add(key);
      addQuad(positions, indices, corners[side], corners[next], width);
    }
  }
}

/** Builds a transparent, depth-tested grid mesh over the authored floor plane. */
export function gridGeometry(grid, options = {}) {
  const positions = [];
  const indices = [];
  const seen = new Set();
  const walkable = walkableSet(grid);
  const width = number(options.lineWidth, DEFAULT_LINE_WIDTH);
  const height = number(options.height, DEFAULT_HEIGHT);
  const cols = Math.max(0, Number(grid?.cols) || 0);
  const rows = Math.max(0, Number(grid?.rows) || 0);
  for (let row = 0; row < rows; row += 1) {
    for (let column = 0; column < cols; column += 1) {
      if (walkable.has(cellIndex(grid, column, row))) {
        appendCellEdges(positions, indices, seen, grid, column, row, width, height);
      }
    }
  }
  return { positions, indices };
}

/** Creates or replaces the PlayCanvas grid entity and returns it. */
export function createGridOverlay(pc, app, grid, options = {}) {
  if (!app?.graphicsDevice) throw new TypeError("a PlayCanvas application is required");
  const geometry = gridGeometry(grid, options);
  const mesh = new pc.Mesh(app.graphicsDevice);
  mesh.setPositions(geometry.positions);
  mesh.setIndices(geometry.indices);
  mesh.update();
  const material = new pc.StandardMaterial();
  const color = options.color ?? [0.2, 0.85, 0.9];
  material.diffuse = new pc.Color(...color, number(options.opacity, 0.72));
  material.emissive = new pc.Color(...color, 1);
  material.opacity = number(options.opacity, 0.72);
  material.blendType = pc.BLEND_NORMAL;
  material.depthWrite = false;
  material.cull = pc.CULLFACE_NONE;
  material.update();
  const instance = new pc.MeshInstance(mesh, material);
  const entity = new pc.Entity(options.name ?? "df-splat-grid");
  entity.addComponent("render", { meshInstances: [instance], layers: options.layers });
  app.root.addChild(entity);
  return entity;
}
