import { createBattleViewer } from "./battle_viewer.mjs";

const listeners = new Set();
let runtime = null;
// The TV's battle stage can remount (and re-subscribe) after the scene is
// already ready; it then never saw "ready" and kept the canvas hidden. The
// last lifecycle event of the live runtime is replayed to new listeners, and
// an init for the scene that is already loaded is not reloaded.
let lifecycle = null;
let loadedSceneURL = "";
let runtimeCanvas = null;
let lastScene = null;
function emit(event) { if (!["ready", "error", "stats", "pick", "disposed"].includes(event.type)) return; if (["ready", "error", "disposed"].includes(event.type)) lifecycle = event; for (const listener of listeners) { try { const { state, ...payload } = event; listener({ v: 1, ...payload }); } catch (error) { queueMicrotask(() => { throw error; }); } } }
function bridgeRuntime(next) { runtime = next; runtime.onEvent(event => emit(event)); return runtime; }

/**
 * gameProfile fetches a scene profile and pins it to the engine's grid: the
 * init message's grid (the engine's cells) replaces the profile grid and the
 * collider keeps it (keep_authored_grid) instead of re-gridding the whole
 * scan, so a token's engine cell is the cell it is drawn on. Relative asset
 * URLs are resolved against the profile's URL.
 */
async function gameProfile(profileURL, message) {
  const url = new URL(profileURL, globalThis.location?.href).href;
  const response = await fetch(url);
  if (!response.ok) throw new Error(`battle profile HTTP ${response.status}`);
  const profile = await response.json();
  const resolve = value => (typeof value === "string" && value ? new URL(value, url).href : value);
  for (const key of ["scene_url", "lite_url", "meta_url", "lod_meta_url", "voxel_collider_url"]) if (profile[key]) profile[key] = resolve(profile[key]);
  if (profile.voxel_collider && typeof profile.voxel_collider === "object") profile.voxel_collider = { ...profile.voxel_collider, url: resolve(profile.voxel_collider.url), keep_authored_grid: true };
  const grid = message.grid;
  if (grid && Number(grid.cols) > 0 && Number(grid.rows) > 0 && Array.isArray(grid.walkable)) profile.grid = { ...profile.grid, ...grid };
  return profile;
}

/** Sends a legacy v1 message to the fullscreen bridge runtime. */
function send(raw) {
  let message;
  try { message = typeof raw === "string" ? JSON.parse(raw) : raw; } catch (error) { emit({ type: "error", code: "INVALID_MESSAGE", detail: String(error) }); return; }
  if (!message || message.v !== 1 || typeof message.type !== "string") { emit({ type: "error", code: "INVALID_MESSAGE", detail: "expected v:1 and a message type" }); return; }
  if (message.type === "init") {
    if (message.device && message.device !== "webgl2") { emit({ type: "error", code: "WEBGL_UNAVAILABLE", detail: `unsupported device: ${message.device}` }); return; }
    try {
      const canvas = document.getElementById(message.canvas_id); if (!canvas) throw new Error(`canvas not found: ${message.canvas_id}`);
      if (runtime && runtimeCanvas === canvas && canvas.isConnected && loadedSceneURL === String(message.scene_url ?? "")) { if (lifecycle) emit(lifecycle); return; }
      runtime?.dispose(); runtime = null; lifecycle = null; runtimeCanvas = canvas; loadedSceneURL = String(message.scene_url ?? ""); bridgeRuntime(createBattleViewer({ canvas, controls: false, cameraControls: false, layout: "fullscreen", reducedMotion: message.reduced_motion }));
      // A scene_url that names a profile (web/splat/scenes/<id>.json) is fetched as
      // the profile, which carries the splat, collider, grid, and cameras. Sending
      // the init object itself made the loader treat the profile JSON as the splat.
      const profileURL = typeof message.scene_url === "string" && /\/scenes\/[^/]+\.json$/.test(message.scene_url) ? message.scene_url : null;
      const source = message.profile ?? message.scene ?? (profileURL ? gameProfile(profileURL, message) : message);
      void Promise.resolve(source).then(value => runtime?.load(value)).catch(() => {});
    } catch (error) { emit({ type: "error", code: "WEBGL_UNAVAILABLE", detail: String(error?.message ?? error) }); }
    return;
  }
  if (message.type === "dispose") { runtime?.dispose(); runtime = null; lifecycle = null; loadedSceneURL = ""; return; }
  if (!["scene", "pause", "effects"].includes(message.type)) { emit({ type: "error", code: "INVALID_MESSAGE", detail: `unknown type: ${message.type}` }); return; }
  if (message.type === "scene") lastScene = message;
  runtime?.send(message);
}

/** Registers a bridge event listener and returns its unsubscribe function. */
function onEvent(listener) {
  if (typeof listener !== "function") throw new TypeError("listener must be a function");
  listeners.add(listener);
  if (runtime && lifecycle) { const { state, ...payload } = lifecycle; queueMicrotask(() => { if (listeners.has(listener)) listener({ v: 1, ...payload }); }); }
  return () => listeners.delete(listener);
}
/** Returns the live runtime's state snapshot (tokens, camera, grid) for debugging, or null. */
function state() { return { runtime: runtime?.getState?.() ?? null, lastScene }; }
if (typeof window !== "undefined") window.dfSplat = Object.freeze({ send, onEvent, state });
export { send, onEvent };
