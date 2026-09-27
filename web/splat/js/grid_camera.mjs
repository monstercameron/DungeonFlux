const PRESETS = Object.freeze({
  COMBAT_EST: Object.freeze({ position: [0, 8, 12], target: [0, 0, 0], fov: 35, arcSeconds: 3 }),
  TACTICAL: Object.freeze({ position: [0, 7, 10], target: [0, 0, 0], fov: 35, pitch: 38 }),
  TURN_FOCUS: Object.freeze({ position: [0, 6.3, 9], target: [0, 0, 0], fov: 35, dolly: 1.1 }),
  IMPACT: Object.freeze({ position: [0, 7, 10], target: [0, 0, 0], fov: 35, shakePx: 8, durationMs: 200 }),
  KO: Object.freeze({ position: [0, 8.05, 11], target: [0, 0, 0], fov: 35, rise: 0.15, pullBack: 0.1 }),
  VICTORY: Object.freeze({ position: [0, 8, 12], target: [0, 0, 0], fov: 35, arcSeconds: 5 }),
});

/** Returns an immutable copy of the named demo camera preset. */
export function cameraPreset(name) {
  return PRESETS[name] ?? PRESETS.TACTICAL;
}

/** Returns the stable camera names used by the combat battlefield. */
export function cameraPresetNames() {
  return Object.keys(PRESETS);
}

/** Applies a preset to a PlayCanvas camera, optionally focusing its active token. */
export function applyCameraPreset(camera, name, focus = null) {
  if (!camera) throw new TypeError("a camera is required");
  const preset = cameraPreset(name);
  const focusPoint = Array.isArray(focus) && focus.length >= 3 ? focus : preset.target;
  const scale = preset.dolly ? 1 / preset.dolly : 1;
  const position = preset.position.map((value, index) => focusPoint[index] + (value - preset.target[index]) * scale);
  camera.setPosition(...position);
  camera.lookAt(...focusPoint);
  if (camera.camera) camera.camera.fov = preset.fov;
  return preset;
}

