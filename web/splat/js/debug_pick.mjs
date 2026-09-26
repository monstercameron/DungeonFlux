function validGrid(grid) {
  return Number(grid?.cols) > 0 && Number(grid?.rows) > 0 && Number(grid?.cell_m) > 0;
}

function inBounds(grid, cell) {
  return cell.c >= 0 && cell.c < Number(grid.cols) && cell.r >= 0 && cell.r < Number(grid.rows);
}

function heightAt(grid, column, row) {
  const cols = Number(grid.cols) || 0;
  const heights = grid.corner_heights;
  if (!Array.isArray(heights) || heights.length !== (cols + 1) * (Number(grid.rows) + 1)) return 0.02;
  const value = Number(heights[row * (cols + 1) + column]);
  return Number.isFinite(value) ? value : 0.02;
}

function cellCorners(grid, column, row) {
  const origin = Array.isArray(grid.origin) ? grid.origin : [0, 0];
  const cellM = Number(grid.cell_m);
  const x = Number(origin[0]) + column * cellM;
  const z = Number(origin[1]) + row * cellM;
  return [
    { x, y: heightAt(grid, column, row), z },
    { x: x + cellM, y: heightAt(grid, column + 1, row), z },
    { x: x + cellM, y: heightAt(grid, column + 1, row + 1), z: z + cellM },
    { x, y: heightAt(grid, column, row + 1), z: z + cellM },
  ];
}

function subtract(a, b) {
  return { x: a.x - b.x, y: a.y - b.y, z: a.z - b.z };
}

function cross(a, b) {
  return { x: a.y * b.z - a.z * b.y, y: a.z * b.x - a.x * b.z, z: a.x * b.y - a.y * b.x };
}

function dot(a, b) {
  return a.x * b.x + a.y * b.y + a.z * b.z;
}

function rayTriangle(origin, direction, a, b, c) {
  const edgeA = subtract(b, a);
  const edgeB = subtract(c, a);
  const perpendicular = cross(direction, edgeB);
  const determinant = dot(edgeA, perpendicular);
  if (Math.abs(determinant) < 1e-9) return null;
  const inverse = 1 / determinant;
  const offset = subtract(origin, a);
  const u = dot(offset, perpendicular) * inverse;
  if (u < 0 || u > 1) return null;
  const crossOffset = cross(offset, edgeA);
  const v = dot(direction, crossOffset) * inverse;
  if (v < 0 || u + v > 1) return null;
  const distance = dot(edgeB, crossOffset) * inverse;
  return distance >= 0 ? distance : null;
}

function occupiedSet(grid) {
  const values = grid.occupied ?? grid.blocked ?? grid.obstacles;
  return new Set(Array.isArray(values) ? values.map(String) : []);
}

function isPickable(grid, cell, walkable, occupied) {
  if (!grid.voxel_filtered) return true;
  const key = `${cell.c},${cell.r}`;
  const index = cell.r * Number(grid.cols) + cell.c;
  return walkable.has(index) && !occupied.has(key) && !occupied.has(String(index));
}

function pickCell(camera, canvas, grid, clientX, clientY) {
  const rect = canvas.getBoundingClientRect();
  const x = clientX - rect.left;
  const y = clientY - rect.top;
  const near = camera.screenToWorld(x, y, 0);
  const far = camera.screenToWorld(x, y, 1);
  const direction = subtract(far, near);
  const walkable = new Set(Array.isArray(grid.walkable) ? grid.walkable.map(Number) : []);
  const occupied = occupiedSet(grid);
  let best = null;
  for (let row = 0; row < Number(grid.rows); row += 1) {
    for (let column = 0; column < Number(grid.cols); column += 1) {
      const cell = { c: column, r: row };
      if (!isPickable(grid, cell, walkable, occupied)) continue;
      const corners = cellCorners(grid, column, row);
      const first = rayTriangle(near, direction, corners[0], corners[1], corners[2]);
      const second = rayTriangle(near, direction, corners[0], corners[2], corners[3]);
      const distance = first === null ? second : second === null ? first : Math.min(first, second);
      if (distance !== null && (best === null || distance < best.distance)) {
        best = { cell, distance };
      }
    }
  }
  return best?.cell ?? null;
}

function writeJSON(cells, filename) {
  const blob = new Blob([cellsJSON(cells)], { type: "application/json" });
  const link = document.createElement("a");
  link.href = URL.createObjectURL(blob);
  link.download = filename;
  link.click();
  URL.revokeObjectURL(link.href);
}

/** Serializes the currently selected cells in the nav-layer shape. */
export function cellsJSON(cells) {
  return JSON.stringify({ walkable: cells }, null, 2);
}

/** Installs query-string gated floor picking and returns a controller for authored cells. */
export function installDebugPickMode({ camera, canvas, grid, onPick, filename = "battlefield-picks.json" }) {
  if (!camera || !canvas || !validGrid(grid)) throw new TypeError("camera, canvas, and a valid grid are required");
  const enabled = new URLSearchParams(window.location.search).has("debug");
  const cells = [];
  const handler = (event) => {
    if (!enabled) return;
    const cell = pickCell(camera, canvas, grid, event.clientX, event.clientY);
    if (!cell || !inBounds(grid, cell)) return;
    const key = `${cell.c},${cell.r}`;
    const index = cells.findIndex((value) => `${value.c},${value.r}` === key);
    if (index >= 0) cells.splice(index, 1);
    else cells.push(cell);
    cells.sort((left, right) => left.r - right.r || left.c - right.c);
    const payload = { c: cell.c, r: cell.r, selected: index < 0, walkable: cells, json: cellsJSON(cells) };
    onPick?.(payload);
    if (event.shiftKey) writeJSON(cells, filename);
  };
  canvas.addEventListener("click", handler);
  return Object.freeze({ enabled, cells, download: () => writeJSON(cells, filename), dispose: () => canvas.removeEventListener("click", handler) });
}
