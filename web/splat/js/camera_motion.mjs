const MAX_PAN_MS = 5000;
const MAX_SHAKE_MS = 200;
const MAX_SHAKE_PX = 32;

function finite(value, fallback) {
  const number = Number(value);
  return Number.isFinite(number) ? number : fallback;
}

function vector(value, fallback) {
  if (Array.isArray(value) && value.length >= 3) return value.slice(0, 3).map((item, i) => finite(item, fallback[i]));
  if (value && typeof value === "object") return [finite(value.x, fallback[0]), finite(value.y, fallback[1]), finite(value.z, fallback[2])];
  return fallback.slice();
}

function poseFrom(definition, fallback) {
  const source = definition && typeof definition === "object" ? definition : {};
  return { position: vector(source.position, fallback.position), target: vector(source.target ?? source.look_at, fallback.target), fov: Math.max(1, Math.min(179, finite(source.fov, fallback.fov))) };
}

function clonePose(pose) {
  return { position: pose.position.slice(), target: pose.target.slice(), fov: pose.fov };
}

function easeInOut(value) {
  const t = Math.max(0, Math.min(1, value));
  return t * t * (3 - 2 * t);
}

function applyPose(camera, pose) {
  if (typeof camera?.setPosition === "function") camera.setPosition(...pose.position);
  else if (camera?.position) Object.assign(camera.position, { x: pose.position[0], y: pose.position[1], z: pose.position[2] });
  if (typeof camera?.lookAt === "function") camera.lookAt(...pose.target);
  else if (camera?.target) Object.assign(camera.target, { x: pose.target[0], y: pose.target[1], z: pose.target[2] });
  if (camera?.camera && "fov" in camera.camera) camera.camera.fov = pose.fov;
}

function currentCameraPose(camera, fallback) {
  const position = typeof camera?.getPosition === "function" ? camera.getPosition() : camera?.position;
  const next = clonePose(fallback);
  if (position) next.position = vector(position, next.position);
  if (camera?.target) next.target = vector(camera.target, next.target);
  else if (camera?.forward) {
    const forward = vector(camera.forward, [0, 0, -1]);
    const distance = Math.max(0.001, Math.hypot(...next.position.map((value, i) => value - fallback.target[i])));
    next.target = next.position.map((value, i) => value + forward[i] * distance);
  }
  if (camera?.camera && "fov" in camera.camera) next.fov = Math.max(1, Math.min(179, finite(camera.camera.fov, next.fov)));
  return next;
}

function applyShake(state) {
  if (!state.canvas?.style || !state.shakeState) return;
  const shake = state.shakeState;
  const phase = shake.elapsed * 77;
  const fade = Math.max(0, 1 - shake.elapsed / shake.duration);
  const x = Math.sin(phase) * shake.amplitude * fade;
  const y = Math.cos(phase * 1.37 + 0.7) * shake.amplitude * 0.72 * fade;
  state.canvas.style.transform = `${state.baseTransform} translate3d(${x.toFixed(3)}px, ${y.toFixed(3)}px, 0)`;
}

function clearShake(state) {
  state.shakeState = null;
  if (state.canvas?.style) state.canvas.style.transform = state.baseTransform;
}

function setPose(state, definition) {
  if (state.destroyed) return clonePose(state.pose);
  state.pose = poseFrom(definition, state.pose);
  state.panState = null;
  state.trackTarget = null;
  clearShake(state);
  applyPose(state.camera, state.pose);
  return clonePose(state.pose);
}

function pan(state, definition, durationMs = 1200) {
  if (state.destroyed) return clonePose(state.pose);
  state.pose = currentCameraPose(state.camera, state.pose);
  const target = poseFrom(definition, state.pose);
  const duration = Math.max(0, Math.min(MAX_PAN_MS, finite(durationMs, 1200)));
  clearShake(state);
  state.trackTarget = null;
  if (!state.enabled || duration === 0) return setPose(state, target);
  state.panState = { from: clonePose(state.pose), to: target, elapsed: 0, duration: duration / 1000 };
  return clonePose(target);
}

function validPoint(point) {
  return Array.isArray(point) && point.length === 3 && point.every((value) => Number.isFinite(value));
}

function track(state, point, dtSeconds) {
  if (state.destroyed || state.paused || (!state.enabled && !state.reducedMotion) || !validPoint(point)) return false;
  const dt = Number(dtSeconds);
  if (!Number.isFinite(dt)) return false;
  state.pose = currentCameraPose(state.camera, state.pose);
  state.trackTarget = null;
  state.panState = null;
  state.trackTarget = point.slice();
  const amount = state.reducedMotion ? 1 : 1 - Math.exp(-12 * Math.max(0, Math.min(1, dt)));
  const offset = state.pose.position.map((value, i) => value - state.pose.target[i]);
  const target = state.pose.target.map((value, i) => value + (point[i] - value) * amount);
  state.pose = { position: target.map((value, i) => value + offset[i]), target, fov: state.pose.fov };
  applyPose(state.camera, state.pose);
  return true;
}

function shake(state, { amplitude_px = 8, duration_ms = 200 } = {}) {
  if (!state.enabled || state.destroyed) return;
  const amplitude = Math.max(0, Math.min(MAX_SHAKE_PX, finite(amplitude_px, 8)));
  const duration = Math.max(0, Math.min(MAX_SHAKE_MS, finite(duration_ms, 200))) / 1000;
  if (duration === 0 || amplitude === 0) return clearShake(state);
  state.shakeState = { elapsed: 0, duration, amplitude };
}

function updatePan(state, dt) {
  if (!state.panState) return;
  const motion = state.panState;
  motion.elapsed = Math.min(motion.duration, motion.elapsed + dt);
  const t = easeInOut(motion.elapsed / motion.duration);
  state.pose = { position: motion.from.position.map((value, i) => value + (motion.to.position[i] - value) * t), target: motion.from.target.map((value, i) => value + (motion.to.target[i] - value) * t), fov: motion.from.fov + (motion.to.fov - motion.from.fov) * t };
  applyPose(state.camera, state.pose);
  if (motion.elapsed >= motion.duration) state.panState = null;
}

function updateShake(state, dt) {
  if (!state.shakeState) return;
  state.shakeState.elapsed = Math.min(state.shakeState.duration, state.shakeState.elapsed + dt);
  applyShake(state);
  if (state.shakeState.elapsed >= state.shakeState.duration) clearShake(state);
}

function update(state, dtSeconds) {
  if (state.destroyed || state.paused) return;
  const dt = Math.max(0, Math.min(1, finite(dtSeconds, 0)));
  updatePan(state, dt);
  updateShake(state, dt);
}

function stop(state) {
  state.pose = currentCameraPose(state.camera, state.pose);
  state.panState = null;
  state.trackTarget = null;
  clearShake(state);
  return clonePose(state.pose);
}

function setEnabled(state, value) {
  state.enabled = Boolean(value) && !state.reducedMotion;
  if (!state.enabled) {
    state.panState = null;
    state.trackTarget = null;
    clearShake(state);
  }
}

function destroy(state) {
  if (state.destroyed) return;
  state.pose = currentCameraPose(state.camera, state.pose);
  state.destroyed = true;
  state.panState = null;
  state.trackTarget = null;
  clearShake(state);
}

/** Creates deterministic camera pan and impact shake motion without timers. */
export function createCameraMotion({ camera, canvas, reducedMotion = false } = {}) {
  const state = { camera, canvas, baseTransform: canvas?.style?.transform ?? "", pose: poseFrom(null, { position: [0, 0, 0], target: [0, 0, -1], fov: 45 }), panState: null, trackTarget: null, shakeState: null, reducedMotion: Boolean(reducedMotion), enabled: !reducedMotion, paused: false, destroyed: false };
  return {
    setPose: (definition) => setPose(state, definition),
    pan: (definition, durationMs) => pan(state, definition, durationMs),
    track: (point, dtSeconds) => track(state, point, dtSeconds),
    shake: (definition) => shake(state, definition),
    stop: () => stop(state),
    setEnabled: (value) => setEnabled(state, value),
    setReducedMotion: (value) => { state.reducedMotion = Boolean(value); if (state.reducedMotion) setEnabled(state, false); },
    setPaused: (value) => { state.paused = Boolean(value); },
    update: (dtSeconds) => update(state, dtSeconds),
    destroy: () => destroy(state),
    getPose: () => clonePose(state.pose),
    getTrackTarget: () => state.trackTarget?.slice() ?? null,
    get trackTarget() { return state.trackTarget?.slice() ?? null; },
  };
}
