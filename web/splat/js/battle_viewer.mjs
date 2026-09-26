import { createBattleRuntime } from "./battle_runtime.mjs";
import { createBattleControls } from "./battle_controls.mjs";

/** createBattleViewer mounts a runtime with optional controls; no demo or game state is invented. */
export function createBattleViewer({ canvas, controls = true, controlsContainer, cameraControls = true, onMove, ...options } = {}) {
  const runtime = createBattleRuntime({ canvas, cameraControls, ...options });
  let panel = null;
  const setControlsVisible = (visible) => {
    if (runtime.getState().disposed) throw new Error("battle viewer disposed");
    if (visible && !panel) panel = createBattleControls({ runtime, canvas, container: controlsContainer ?? canvas.parentElement, cameraControls, onMove });
    panel?.setVisible(Boolean(visible));
  };
  try { if (controls) setControlsVisible(true); } catch (error) { runtime.dispose(); throw error; }
  const unsubscribe = runtime.onEvent(event => { if (event.type === "disposed") { panel?.dispose(); panel = null; } });
  return Object.freeze({ ...runtime, setControlsVisible,
    dispose() { panel?.dispose(); panel = null; unsubscribe(); runtime.dispose(); } });
}
