function validGrid(grid) {
  return Number(grid?.cols) > 0 && Number(grid?.rows) > 0 && Number(grid?.cell_m) > 0;
}

function cellAt(grid, x, z) {
  const origin = Array.isArray(grid.origin) ? grid.origin : [0, 0];
  return {
    c: Math.floor((x - Number(origin[0])) / Number(grid.cell_m)),
    r: Math.floor((z - Number(origin[1])) / Number(grid.cell_m)),
  };
}

function inBounds(grid, cell) {
  return cell.c >= 0 && cell.c < Number(grid.cols) && cell.r >= 0 && cell.r < Number(grid.rows);
}

function pickWorld(camera, canvas, clientX, clientY, height) {
  const rect = canvas.getBoundingClientRect();
  const x = clientX - rect.left;
  const y = clientY - rect.top;
  const near = camera.screenToWorld(x, y, 0);
  const far = camera.screenToWorld(x, y, 1);
  const directionY = far.y - near.y;
  if (Math.abs(directionY) < 1e-9) return null;
  const factor = (height - near.y) / directionY;
  return { x: near.x + (far.x - near.x) * factor, z: near.z + (far.z - near.z) * factor };
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
    const world = pickWorld(camera, canvas, event.clientX, event.clientY, 0.02);
    if (!world) return;
    const cell = cellAt(grid, world.x, world.z);
    if (!inBounds(grid, cell)) return;
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
