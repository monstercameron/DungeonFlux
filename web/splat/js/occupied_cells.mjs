const walkableCache = new WeakMap();
export function tokenCell(raw) {
  return Array.isArray(raw) ? { c: Number(raw[0]), r: Number(raw[1]) }
    : { c: Number(raw?.c), r: Number(raw?.r) };
}
export function validTokenCell(grid, raw) {
  const cell = tokenCell(raw);
  if (!grid || !Number.isInteger(cell.c) || !Number.isInteger(cell.r)) return false;
  if (cell.c < 0 || cell.r < 0 || cell.c >= grid.cols || cell.r >= grid.rows) return false;
  if (!walkableCache.has(grid)) walkableCache.set(grid, new Set(grid.walkable ?? []));
  return walkableCache.get(grid).has(cell.r * grid.cols + cell.c);
}
export function adjacentCells(a, b) {
  return Math.max(Math.abs(a.c - b.c), Math.abs(a.r - b.r)) === 1;
}
export function terrainHeight(grid, c, r) {
  const values = grid.corner_heights;
  if (!Array.isArray(values) || values.length !== (grid.cols + 1) * (grid.rows + 1)) return 0;
  const x = Math.max(0, Math.min(grid.cols - 1, Math.floor(c)));
  const z = Math.max(0, Math.min(grid.rows - 1, Math.floor(r)));
  const tx = Math.max(0, Math.min(1, c - x));
  const tz = Math.max(0, Math.min(1, r - z));
  const at = (a, b) => Number.isFinite(values[b * (grid.cols + 1) + a]) ? values[b * (grid.cols + 1) + a] : 0;
  return at(x,z)*(1-tx)*(1-tz) + at(x+1,z)*tx*(1-tz) + at(x,z+1)*(1-tx)*tz + at(x+1,z+1)*tx*tz;
}
export function cellPosition(grid, raw) {
  const cell = tokenCell(raw);
  const size = Number(grid.cell_m ?? 1.524), origin = grid.origin ?? [0,0];
  return [origin[0] + (cell.c + .5)*size, terrainHeight(grid, cell.c+.5, cell.r+.5), origin[1] + (cell.r+.5)*size];
}
export function positionCell(grid, position) {
  const size = Number(grid.cell_m ?? 1.524), origin = grid.origin ?? [0,0];
  return { c: Math.floor((position[0]-origin[0])/size), r: Math.floor((position[2]-origin[1])/size) };
}
export function normalizeTokenPath(grid, token) {
  if (Array.isArray(token.path) && token.path.length > 256) return null;
  const destination = tokenCell(token.cell);
  const cells = Array.isArray(token.path) ? token.path.map(tokenCell) : [];
  if (!cells.length || cells.at(-1).c !== destination.c || cells.at(-1).r !== destination.r) cells.push(destination);
  if (cells.some(cell => !validTokenCell(grid,cell))) return null;
  for (let i=1;i<cells.length;i++) if (!adjacentCells(cells[i-1],cells[i])) return null;
  return cells;
}
export function createOccupiedCellSet(grid, tokens = []) {
  const occupied = new Map();
  for (const token of tokens) {
    const cell = tokenCell(token?.cell), key = `${cell.c},${cell.r}`;
    if (validTokenCell(grid,cell) && !occupied.has(key)) occupied.set(key,token.id ?? `token-${occupied.size}`);
  }
  return occupied;
}
