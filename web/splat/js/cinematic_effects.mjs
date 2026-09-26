import { createCameraMotion } from "./camera_motion.mjs";
import { createTiltShift } from "./tilt_shift.mjs";

function validSeq(value) { return Number.isSafeInteger(value) && value > 0; }
function seconds(dt) { return Number.isFinite(dt) ? Math.min(1, Math.max(0, dt)) : 0; }
function duration(value) {
  const number = Number(value ?? 0);
  return Number.isFinite(number) ? Math.min(5000, Math.max(0, number)) : 0;
}

class CinematicEffects {
  constructor({ pc, app, camera, canvas, cameras = {}, reducedMotion = false, onPanComplete }) {
    if (!camera) throw new TypeError("camera is required");
    this.app = app;
    this.cameras = cameras;
    this.reducedMotion = Boolean(reducedMotion);
    this.enabled = true;
    this.motion = createCameraMotion({ camera, canvas, reducedMotion });
    this.tilt = createTiltShift(pc, camera);
    this.onPanComplete = onPanComplete;
    this.cameraSeq = 0;
    this.effectsSeq = 0;
    this.legacyPreset = null;
    this.remaining = 0;
    this.target = null;
    this.paused = false;
    this.destroyed = false;
    this.onUpdate = (dt) => this.update(dt);
    this.onDestroy = () => this.destroy();
    app?.on?.("update", this.onUpdate);
    app?.on?.("destroy", this.onDestroy);
  }

  hasPreset(preset) { return Object.hasOwn(this.cameras, preset) && this.cameras[preset] != null; }

  setPose(definition) {
    this.remaining = 0;
    this.target = null;
    return this.motion.setPose(definition);
  }

  track(point, dt) {
    if (this.destroyed || this.paused || !this.enabled || !Array.isArray(point) || point.length !== 3 || !point.every(Number.isFinite) || !Number.isFinite(dt)) return false;
    this.remaining = 0;
    this.target = null;
    return this.motion.track(point, dt);
  }

  move(preset, durationMs) {
    const definition = this.cameras[preset];
    const time = duration(durationMs);
    this.remaining = this.enabled && !this.reducedMotion ? time / 1000 : 0;
    this.target = definition.target ?? definition.look_at ?? null;
    this.motion.pan(definition, time);
    if (!this.remaining && this.target) {
      this.onPanComplete?.(this.target);
      this.target = null;
    }
  }

  camera(command = {}) {
    if (this.destroyed || !this.hasPreset(command.preset)) return false;
    const seq = command.seq ?? 0;
    if (!Number.isSafeInteger(seq) || seq < 0) return false;
    if (seq > 0 && seq <= this.cameraSeq) return false;
    if (seq === 0 && this.legacyPreset === command.preset) return false;
    if (seq > 0) this.cameraSeq = seq;
    this.legacyPreset = command.preset;
    this.move(command.preset, command.duration_ms ?? command.durationMs);
    return true;
  }

  send(effect = {}) {
    if (this.destroyed || !effect || !validSeq(effect.seq) || effect.seq <= this.effectsSeq) return false;
    if (effect.pan && !this.hasPreset(effect.pan.preset)) return false;
    this.effectsSeq = effect.seq;
    if (effect.reduced_motion !== undefined) {
      this.reducedMotion = Boolean(effect.reduced_motion);
      this.motion.setReducedMotion(this.reducedMotion);
      if (this.reducedMotion) this.stop();
    }
    if (effect.enabled !== undefined) {
      this.enabled = Boolean(effect.enabled);
      this.motion.setEnabled(this.enabled);
      if (!this.enabled) { this.stop(); this.tilt.configure({ enabled: false }); }
    }
    if (effect.stop) this.stop();
    if (effect.tilt_shift && this.enabled) this.tilt.configure(effect.tilt_shift);
    if (effect.pan) this.move(effect.pan.preset, effect.pan.duration_ms);
    if (effect.shake && this.enabled) this.motion.shake(effect.shake);
    return true;
  }

  update(dt) {
    if (this.destroyed || this.paused) return;
    this.motion.update(dt);
    if (this.remaining <= 0) return;
    this.remaining = Math.max(0, this.remaining - seconds(dt));
    if (this.remaining === 0 && this.target) {
      this.onPanComplete?.(this.target);
      this.target = null;
    }
  }

  pause(on) { this.paused = Boolean(on); this.motion.setPaused(this.paused); }
  stop() { this.remaining = 0; this.target = null; return this.motion.stop(); }

  destroy() {
    if (this.destroyed) return;
    this.destroyed = true;
    this.app?.off?.("update", this.onUpdate);
    this.app?.off?.("destroy", this.onDestroy);
    this.tilt.destroy();
    this.motion.destroy();
  }
}

/** Creates game-triggerable post processing, eased pans, and impact shakes. */
export function createCinematicEffects(options = {}) {
  const state = new CinematicEffects(options);
  return Object.freeze({
    setPose: (definition) => state.setPose(definition),
    camera: (command) => state.camera(command),
    send: (effect) => state.send(effect),
    track: (point, dt) => state.track(point, dt),
    pause: (on) => state.pause(on),
    stop: () => state.stop(),
    destroy: () => state.destroy(),
    update: (dt) => state.update(dt),
    getPose: state.motion.getPose,
    getTrackTarget: state.motion.getTrackTarget,
  });
}
