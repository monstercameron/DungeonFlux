import { selectDemoCells, createDemoSnapshot, nextDemoMove } from "./token_demo.mjs";

/** createBattleDemo supplies stand-ins and legal moves only to explicitly local previews. */
export function createBattleDemo(runtime) {
  let seq = 0, cells = [], grid = null, tokens = [], disposed = false;
  const seed = (state) => {
    grid = state.grid; if (!grid?.walkable?.length) return;
    cells = selectDemoCells(grid, state.cameras?.TACTICAL?.target ?? [0,0,0]);
    tokens = createDemoSnapshot(cells, ++seq).tokens;
    runtime.setTokens(tokens, seq);
  };
  const unsubscribe = runtime.onEvent(event => {
    if (event.type === "loading") { grid = null; tokens = []; seq = 0; }
    if (event.type === "ready" && !tokens.length) seed(event.state);
  });
  if (runtime.getState().ready) seed(runtime.getState());
  const move = () => {
    const state = runtime.getState(); if (disposed || state.paused || !grid) return false;
    const id = state.tokens.followId ?? "player";
    const entry = state.tokens.tokens.find(token => token.id === id);
    if (!entry || entry.moving) return false;
    const route = nextDemoMove(grid, { c: entry.cell[0], r: entry.cell[1] }, cells, seq * 7);
    if (!route) return false;
    tokens = tokens.map(token => token.id === id ? { ...token, cell: route.target, path: route.path, anim_seq: ++seq } : { ...token, path: [] });
    runtime.setTokens(tokens, seq); return true;
  };
  return Object.freeze({ move, dispose() { if (disposed) return; disposed = true; unsubscribe(); } });
}
