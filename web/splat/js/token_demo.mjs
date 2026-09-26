export { createTokenController } from "./token.mjs";

function validCell(grid, index) { return (grid.walkable ?? []).includes(index); }

/** Selects three nearby cells for the player and villain demo snapshot. */
export function selectDemoCells(grid, target = [0, 0, 0]) {
  const cols = Number(grid.cols); const size = Number(grid.cell_m ?? 1.524); const origin = Array.isArray(grid.origin) ? grid.origin : [0, 0];
  const targetC = (Number(target[0]) - Number(origin[0])) / size; const targetR = (Number(target[2] ?? target[1]) - Number(origin[1])) / size;
  return (grid.walkable ?? []).filter((index) => validCell(grid, index)).map((index) => ({ c: index % cols, r: Math.floor(index / cols) })).sort((a, b) => ((a.c - targetC) ** 2 + (a.r - targetR) ** 2) - ((b.c - targetC) ** 2 + (b.r - targetR) ** 2)).slice(0, 24);
}

/** Builds the initial or next legal demo snapshot. */
export function createDemoSnapshot(cells, seq = 1) {
  const [player, villain, spare] = cells;
  return { seq, tokens: [{ id: "player", name: "Player", kind: "player", cell: player, path: [], anim_seq: seq }, { id: "villain", name: "Villain", kind: "villain", cell: villain ?? player, path: [], anim_seq: seq }], spare };
}

/** Finds a legal 8-connected route, omitting the starting cell. */
export function findDemoPath(grid, start, destination) {
  const key = (cell) => `${cell.c},${cell.r}`;
  const open = [{ ...start }]; const previous = new Map([[key(start), null]]);
  while (open.length) {
    const current = open.shift();
    if (key(current) === key(destination)) break;
    for (let dc = -1; dc <= 1; dc += 1) for (let dr = -1; dr <= 1; dr += 1) {
      if (!dc && !dr) continue;
      const next = { c: current.c + dc, r: current.r + dr }; const index = next.r * Number(grid.cols) + next.c;
      if (next.c < 0 || next.r < 0 || next.c >= grid.cols || next.r >= grid.rows || !(grid.walkable ?? []).includes(index) || previous.has(key(next))) continue;
      previous.set(key(next), current); open.push(next);
    }
  }
  if (!previous.has(key(destination))) return [];
  const path = []; let current = { ...destination };
  while (previous.get(key(current))) { path.unshift(current); current = previous.get(key(current)); }
  return path;
}
