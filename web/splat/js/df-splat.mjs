import { createBattleViewer } from "./battle_viewer.mjs";

const listeners = new Set();
let runtime = null;
function emit(event) { if (!["ready", "error", "stats", "pick", "disposed"].includes(event.type)) return; for (const listener of listeners) { try { const { state, ...payload } = event; listener({ v: 1, ...payload }); } catch (error) { queueMicrotask(() => { throw error; }); } } }
function bridgeRuntime(next) { runtime = next; runtime.onEvent(event => emit(event)); return runtime; }

/** Sends a legacy v1 message to the fullscreen bridge runtime. */
function send(raw) {
  let message;
  try { message = typeof raw === "string" ? JSON.parse(raw) : raw; } catch (error) { emit({ type: "error", code: "INVALID_MESSAGE", detail: String(error) }); return; }
  if (!message || message.v !== 1 || typeof message.type !== "string") { emit({ type: "error", code: "INVALID_MESSAGE", detail: "expected v:1 and a message type" }); return; }
  if (message.type === "init") {
    if (message.device && message.device !== "webgl2") { emit({ type: "error", code: "WEBGL_UNAVAILABLE", detail: `unsupported device: ${message.device}` }); return; }
    try {
      const canvas = document.getElementById(message.canvas_id); if (!canvas) throw new Error(`canvas not found: ${message.canvas_id}`);
      runtime?.dispose(); runtime = null; bridgeRuntime(createBattleViewer({ canvas, controls: false, cameraControls: false, layout: "fullscreen", reducedMotion: message.reduced_motion }));
      void runtime.load(message.profile ?? message.scene ?? message).catch(() => {});
    } catch (error) { emit({ type: "error", code: "WEBGL_UNAVAILABLE", detail: String(error?.message ?? error) }); }
    return;
  }
  if (message.type === "dispose") { runtime?.dispose(); runtime = null; return; }
  if (!["scene", "pause", "effects"].includes(message.type)) { emit({ type: "error", code: "INVALID_MESSAGE", detail: `unknown type: ${message.type}` }); return; }
  runtime?.send(message);
}

/** Registers a bridge event listener and returns its unsubscribe function. */
function onEvent(listener) { if (typeof listener !== "function") throw new TypeError("listener must be a function"); listeners.add(listener); return () => listeners.delete(listener); }
if (typeof window !== "undefined") window.dfSplat = Object.freeze({ send, onEvent });
export { send, onEvent };
