import * as pc from "../vendor/playcanvas.mjs";
import { createBattleGrid, createSplatEntity, loadSplatBundle } from "./battle_scene.mjs";
import { createCinematicEffects } from "./cinematic_effects.mjs";

const VERSION = 1;
const WORLD_LAYER = pc.LAYERID_WORLD;
const DEFAULT_CAMERA = {
  position: [0, 3, 6],
  target: [0, 1, 0],
  fov: 35,
};

const listeners = new Set();
let runtime = null;

function emit(message) {
  const payload = { v: VERSION, ...message };
  for (const listener of listeners) {
    try {
      listener(payload);
    } catch (error) {
      // A client listener must not prevent status delivery to the other bridge listeners.
      queueMicrotask(() => {
        throw error;
      });
    }
  }
}

function errorMessage(code, detail) {
  emit({ type: "error", code, detail: String(detail ?? code) });
}

function asVector(value, fallback) {
  return Array.isArray(value) && value.length >= 3 ? value : fallback;
}

function resolveCanvas(canvasID) {
  const canvas = document.getElementById(canvasID);
  if (!(canvas instanceof HTMLCanvasElement)) {
    throw new Error(`canvas not found: ${canvasID}`);
  }
  return canvas;
}

function configureCanvas(canvas) {
  canvas.style.opacity = "0";
  canvas.style.display = "block";
  canvas.style.position = "fixed";
  canvas.style.inset = "0";
  canvas.style.width = "100%";
  canvas.style.height = "100%";
  canvas.style.zIndex = "0";
  canvas.style.pointerEvents = "none";
}

function createApplication(canvas) {
  const app = new pc.Application(canvas, {
    graphicsDeviceOptions: {
      deviceTypes: ["webgl2"],
      antialias: true,
      alpha: true,
    },
  });
  app.setCanvasFillMode(pc.FILLMODE_FILL_WINDOW);
  app.setCanvasResolution(pc.RESOLUTION_AUTO);
  app.scene.gsplat.renderer = pc.GSPLAT_RENDERER_RASTER_CPU_SORT;
  app.scene.gsplat.lodMode = pc.GSPLAT_LODMODE_DISTANCE;
  app.scene.layers.getLayerById(WORLD_LAYER).enabled = true;
  const resize = () => app.resizeCanvas?.(window.innerWidth ?? canvas.clientWidth, window.innerHeight ?? canvas.clientHeight);
  window.addEventListener?.("resize", resize);
  resize();
  console.info(`[splat runtime] grid MSAA: ${app.graphicsDevice?.backBuffer?.samples ?? "unknown"}`);
  app.on("destroy", () => {
    window.removeEventListener?.("resize", resize);
    if (runtime?.app === app) runtime = null;
  });
  return app;
}

function createCamera(app, definition) {
  const camera = new pc.Entity("df-splat-camera");
  camera.addComponent("camera", {
    clearColor: new pc.Color(0, 0, 0, 0),
    fov: Number(definition.fov ?? DEFAULT_CAMERA.fov),
    farClip: Number(definition.far ?? 1000),
    layers: [WORLD_LAYER],
  });
  app.root.addChild(camera);
  return camera;
}

function applyTransform(entity, transform = {}) {
  const scale = Number(transform.scale ?? 1);
  const offsetY = Number(transform.offset_y ?? 0);
  const translate = asVector(transform.translate, [0, 0, 0]);
  const rotation = Number(transform.rot_y_deg ?? 0);
  entity.setLocalScale(scale, scale, scale);
  entity.setLocalPosition(
    Number(translate[0]),
    Number(translate[1]) + offsetY,
    Number(translate[2]),
  );
  const current = entity.getLocalEulerAngles?.() ?? { z: 0 };
  entity.setLocalEulerAngles(0, rotation, Number(current.z ?? 0));
}

function applyCameraDefinition(camera, definition = DEFAULT_CAMERA) {
  const position = asVector(definition.position, DEFAULT_CAMERA.position);
  const target = asVector(definition.target ?? definition.look_at, DEFAULT_CAMERA.target);
  camera.setPosition(Number(position[0]), Number(position[1]), Number(position[2]));
  camera.lookAt(Number(target[0]), Number(target[1]), Number(target[2]));
  if (camera.camera) {
    camera.camera.fov = Number(definition.fov ?? DEFAULT_CAMERA.fov);
  }
}

function getCameraDefinition(cameras, preset) {
  return cameras?.[preset] ?? cameras?.TACTICAL ?? DEFAULT_CAMERA;
}

function splatCount(asset) {
  const data = asset?.resource?.gsplatData ?? asset?.resource;
  return Number(data?.numSplats ?? data?.numPoints ?? data?.count ?? 0);
}

function releaseAsset(app, asset) {
  if (!asset) return;
  try { app?.assets?.remove?.(asset); } finally { asset.unload?.(); }
}

function percentile(values, fraction) {
  if (values.length === 0) return 0;
  const sorted = [...values].sort((left, right) => left - right);
  return sorted[Math.max(0, Math.floor((sorted.length - 1) * fraction))];
}

function recordFrame(runtimeState) {
  const now = performance.now();
  if (document.hidden) {
    runtimeState.lastFrameAt = now;
    runtimeState.lastStatsAt = now;
    runtimeState.fpsSamples = [];
    runtimeState.lowFpsSince = 0;
    return;
  }
  if (runtimeState.lastFrameAt > 0) {
    const elapsed = now - runtimeState.lastFrameAt;
    if (elapsed > 0) runtimeState.fpsSamples.push(1000 / elapsed);
  }
  runtimeState.lastFrameAt = now;
  if (now - runtimeState.lastStatsAt < 2000) return;
  const fpsP5 = percentile(runtimeState.fpsSamples, 0.05);
  runtimeState.fpsSamples = [];
  runtimeState.lastStatsAt = now;
  emit({ type: "stats", fps_p5: Math.round(fpsP5 * 10) / 10 });
  if (fpsP5 < 30) {
    runtimeState.lowFpsSince ||= now;
    if (now - runtimeState.lowFpsSince >= 3000) void downgradeForFps(runtimeState);
  } else {
    runtimeState.lowFpsSince = 0;
  }
}

function setVisible(canvas, visible) {
  canvas.style.opacity = visible ? "1" : "0";
  canvas.style.transition = visible ? "opacity 600ms ease" : "none";
}

function applyCamera(runtimeState, command) {
  if (!command || !runtimeState.camera) return;
  const sequence = Number(command.seq ?? 0);
  if (!Number.isSafeInteger(sequence) || sequence < 0) return;
  if (sequence > 0 && sequence <= runtimeState.cameraSequence) return;
  if (sequence === 0 && (command.preset ?? "TACTICAL") === runtimeState.cameraPreset) return;
  if (sequence > 0) runtimeState.cameraSequence = sequence;
  if (runtimeState.effects?.camera(command)) { runtimeState.cameraPreset = command.preset; return; }
  const preset = command.preset ?? "TACTICAL";
  const definition = getCameraDefinition(runtimeState.cameras, preset);
  applyCameraDefinition(runtimeState.camera, definition);
  runtimeState.cameraPreset = preset;
}

function acceptScene(runtimeState, scene) {
  const sequence = Number(scene.seq ?? 0);
  if (sequence <= runtimeState.sceneSequence) return;
  runtimeState.sceneSequence = sequence;
  applyCamera(runtimeState, scene.camera);
  setVisible(runtimeState.canvas, Boolean(scene.visible));
}

function dispose() {
  if (!runtime) return;
  const old = runtime;
  runtime = null;
  old.destroyed = true;
  old.canvas.style.opacity = "0";
  old.canvas.replaceWith(old.canvas.cloneNode(true));
  old.app.destroy();
  emit({ type: "disposed" });
}

function createRuntimeState(app, camera, message) {
  return {
    app, camera, canvas: null, cameras: message.cameras ?? {},
    sceneSequence: 0, cameraSequence: 0, effects: null, paused: false, cameraPreset: "TACTICAL", liteURL: message.lite_url,
    liteTried: false, lowFpsReported: false, replacing: false, transform: message.transform,
    lastFrameAt: 0, lastStatsAt: performance.now(), lowFpsSince: 0, fpsSamples: [], destroyed: false,
  };
}

function attachScene(state, message, bundle) {
  const { app, camera } = state;
  const entity = createSplatEntity(pc, app, bundle, {
    layers: [WORLD_LAYER], lodRangeMin: Number(message.lod_range_min ?? 0),
    lodRangeMax: Number(message.lod_range_max ?? 99), name: "df-splat-scene",
  });
  applyTransform(entity, message.transform);
  let grid = null;
  const activeGrid = bundle.grid ?? message.grid;
  if (activeGrid) {
    grid = createBattleGrid(pc, app, activeGrid, {
      name: "df-battle-grid", lineWidth: 0.06, opacity: 0.95, collider: bundle.collider,
    });
    if (grid.layer?.id !== undefined && !camera.camera.layers.includes(grid.layer.id)) {
      camera.camera.layers = [...camera.camera.layers, grid.layer.id];
    }
    if (grid.depthLayer?.id !== undefined && !camera.camera.layers.includes(grid.depthLayer.id)) {
      camera.camera.layers = [...camera.camera.layers, grid.depthLayer.id];
    }
  }
  state.splat = { asset: bundle.asset, entity };
  state.grid = grid;
  state.streaming = bundle.streaming;
  applyCamera(state, { preset: "TACTICAL" });
}

async function initialize(message) {
  dispose();
  if (message.device && message.device !== "webgl2") {
    errorMessage("WEBGL_UNAVAILABLE", `unsupported device: ${message.device}`);
    return;
  }
  let canvas;
  let state = null;
  try {
    canvas = resolveCanvas(message.canvas_id);
    configureCanvas(canvas);
    const app = createApplication(canvas);
    const camera = createCamera(app, getCameraDefinition(message.cameras, "TACTICAL"));
    state = createRuntimeState(app, camera, message);
    state.canvas = canvas;
    state.effects = createCinematicEffects({ pc, app, camera, canvas, cameras: message.cameras ?? {}, reducedMotion: Boolean(message.reduced_motion || window.matchMedia?.("(prefers-reduced-motion: reduce")?.matches), onPanComplete: (target) => state.orbitTarget = target });
    state.effects.setPose(getCameraDefinition(message.cameras, "TACTICAL"));
    runtime = state;
    app.on("error", (detail) => errorMessage("CONTEXT_LOST", detail));
    app.on("postrender", () => {
      if (runtime?.app === app) recordFrame(runtime);
    });
    app.start();
    const bundle = await loadSplatBundle(pc, app, message.scene_url, {
      lodMetaURL: message.lod_meta_url,
      metaURL: message.meta_url,
      grid: message.grid,
      voxelURL: message.voxel_collider_url ?? message.voxel_collider?.url,
      transform: message.transform,
      voxelOptions: message.voxel_collider_options ?? message.voxel_collider,
    });
    if (runtime !== state) {
      if (!state.destroyed) {
        state.destroyed = true;
        app.destroy();
      }
      return;
    }
    attachScene(state, message, bundle);
    emit({
      type: "ready",
      fps: 0,
      gaussians: splatCount(bundle.asset),
      playable: bundle.grid?.walkable?.length ?? message.grid?.walkable?.length ?? 0,
      terrain_excluded: bundle.grid?.excluded?.length ?? 0,
      antialias_samples: app.graphicsDevice?.backBuffer?.samples ?? 0,
      device: "webgl2",
      streaming: bundle.streaming,
      status: bundle.streaming ? "manifest loaded; chunks stream on demand" : "scene loaded",
    });
  } catch (error) {
    const active = state && runtime === state;
    if (active) {
      runtime.app.destroy();
      runtime = null;
    } else if (state?.app && !state.destroyed) {
      state.destroyed = true;
      state.app.destroy();
    }
    if (active || !state) errorMessage(canvas ? "LOAD_FAILED" : "WEBGL_UNAVAILABLE", error);
  }
}

async function downgradeForFps(runtimeState) {
  if (runtimeState.replacing || runtimeState.lowFpsReported) return;
  if (runtimeState.liteTried || !runtimeState.liteURL) {
    runtimeState.lowFpsReported = true;
    errorMessage("LOW_FPS", "5th-percentile FPS stayed below 30");
    return;
  }
  runtimeState.replacing = true;
  runtimeState.liteTried = true;
  try {
    const bundle = await loadSplatBundle(pc, runtimeState.app, runtimeState.liteURL);
    if (runtime !== runtimeState) {
      releaseAsset(runtimeState.app, bundle.asset);
      return;
    }
    const replacementEntity = createSplatEntity(pc, runtimeState.app, bundle, {
      layers: [WORLD_LAYER],
      name: "df-splat-scene-lite",
    });
    applyTransform(replacementEntity, runtimeState.transform);
    const replacement = { asset: bundle.asset, entity: replacementEntity };
    runtimeState.splat.entity.destroy();
    releaseAsset(runtimeState.app, runtimeState.splat.asset);
    runtimeState.splat = replacement;
    runtimeState.lowFpsSince = 0;
    runtimeState.fpsSamples = [];
    emit({
      type: "ready",
      fps: 0,
      gaussians: splatCount(replacement.asset),
      device: "webgl2",
    });
  } catch (error) {
    if (runtime === runtimeState) errorMessage("LOAD_FAILED", error);
  } finally {
    runtimeState.replacing = false;
  }
}

function send(raw) {
  let message;
  try {
    message = typeof raw === "string" ? JSON.parse(raw) : raw;
  } catch (error) {
    errorMessage("INVALID_MESSAGE", error);
    return;
  }
  if (!message || message.v !== VERSION || typeof message.type !== "string") {
    errorMessage("INVALID_MESSAGE", "expected v:1 and a message type");
    return;
  }
  switch (message.type) {
    case "init":
      void initialize(message);
      break;
    case "scene":
      if (runtime) acceptScene(runtime, message);
      break;
    case "pause":
      if (runtime) {
        runtime.paused = Boolean(message.on);
        runtime.effects?.pause(runtime.paused);
        runtime.app.autoRender = !runtime.paused;
        if (!runtime.paused) runtime.app.renderNextFrame = true;
      }
      break;
    case "effects":
      runtime?.effects?.send(message);
      break;
    case "dispose":
      dispose();
      break;
    default:
      errorMessage("INVALID_MESSAGE", `unknown type: ${message.type}`);
  }
}

function onEvent(listener) {
  if (typeof listener !== "function") throw new TypeError("listener must be a function");
  listeners.add(listener);
  return () => listeners.delete(listener);
}

window.dfSplat = Object.freeze({ send, onEvent });
