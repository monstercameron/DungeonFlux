const FPS_LIMIT = 30;
const REPORT_INTERVAL = 2000;
const LOW_FPS_WINDOW = 3000;

function percentile(values, fraction) {
  if (!values.length) return 0;
  const sorted = [...values].sort((left, right) => left - right);
  return sorted[Math.max(0, Math.floor((sorted.length - 1) * fraction))];
}

/** Creates bounded frame statistics and one-shot low-FPS downgrade reporting. */
export function createRuntimeStats({ emit = () => {}, downgrade = null, hasLite = false, clock = () => globalThis.performance?.now?.() ?? Date.now() } = {}) {
  const state = { lastFrameAt: 0, lastStatsAt: clock(), lowFpsSince: 0, samples: [], replacing: false, lowFpsReported: false, liteTried: false, disposed: false };
  const reset = () => { state.lastFrameAt = 0; state.lastStatsAt = clock(); state.lowFpsSince = 0; state.samples = []; state.lowFpsReported = false; };
  const downgradeOnce = () => {
    if (state.replacing || state.lowFpsReported) return;
    if (!hasLite || state.liteTried || typeof downgrade !== "function") { state.lowFpsReported = true; emit({ type: "error", code: "LOW_FPS", detail: "5th-percentile FPS stayed below 30" }); return; }
    state.replacing = true; state.liteTried = true;
    Promise.resolve().then(downgrade).catch(error => { if (!state.disposed && error?.name !== "AbortError") emit({ type: "error", code: "LOAD_FAILED", detail: String(error?.message ?? error) }); }).finally(() => { state.replacing = false; state.lowFpsSince = 0; state.samples = []; });
  };
  const tick = (now = clock(), hidden = false) => {
    if (state.disposed) return;
    if (hidden) { state.lastFrameAt = now; state.lastStatsAt = now; state.lowFpsSince = 0; state.samples = []; return; }
    if (state.lastFrameAt > 0) { const elapsed = now - state.lastFrameAt; if (elapsed > 0) state.samples.push(1000 / elapsed); if (state.samples.length > 4096) state.samples.shift(); }
    state.lastFrameAt = now;
    if (now - state.lastStatsAt < REPORT_INTERVAL) return;
    const fpsP5 = percentile(state.samples, .05);
    state.samples = []; state.lastStatsAt = now; emit({ type: "stats", fps_p5: Math.round(fpsP5 * 10) / 10 });
    if (fpsP5 < FPS_LIMIT) { state.lowFpsSince ||= now; if (now - state.lowFpsSince >= LOW_FPS_WINDOW) downgradeOnce(); } else state.lowFpsSince = 0;
  };
  const dispose = () => { state.disposed = true; state.samples = []; state.lowFpsSince = 0; };
  return Object.freeze({ tick, reset, dispose });
}
