import * as pc from "../vendor/playcanvas.mjs";

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
      antialias: false,
      alpha: true,
    },
  });
  app.setCanvasFillMode(pc.FILLMODE_FILL_WINDOW);
  app.setCanvasResolution(pc.RESOLUTION_AUTO);
  app.scene.gsplat.renderer = pc.GSPLAT_RENDERER_RASTER_CPU_SORT;
  app.scene.gsplat.lodMode = pc.GSPLAT_LODMODE_DISTANCE;
  app.scene.layers.getLayerById(WORLD_LAYER).enabled = true;
  app.on("destroy", () => {
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
  entity.setLocalEulerAngles(0, rotation, 0);
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

function addSplatAsset(app, url, name) {
  return new Promise((resolve, reject) => {
    const asset = new pc.Asset(name, "gsplat", { url });
    const fail = (error) => {
      app.assets.remove(asset);
      reject(error instanceof Error ? error : new Error(String(error ?? "load failed")));
    };
    asset.once("load", () => resolve(asset));
    asset.once("error", fail);
    app.assets.add(asset);
    app.assets.load(asset);
  });
}

function splatCount(asset) {
  const data = asset?.resource?.gsplatData ?? asset?.resource;
  return Number(data?.numSplats ?? data?.numPoints ?? data?.count ?? 0);
}

function percentile(values, fraction) {
  if (values.length === 0) return 0;
  const sorted = [...values].sort((left, right) => left - right);
  return sorted[Math.max(0, Math.floor((sorted.length - 1) * fraction))];
}

function recordFrame(runtimeState) {
  const now = performance.now();
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

async function loadSplat(app, url, transform) {
  const asset = await addSplatAsset(app, url, "df-splat-scene");
  const entity = new pc.Entity("df-splat-scene");
  entity.addComponent("gsplat", { asset, layers: [WORLD_LAYER] });
  applyTransform(entity, transform);
  app.root.addChild(entity);
  return { asset, entity };
}

function setVisible(canvas, visible) {
  canvas.style.opacity = visible ? "1" : "0";
  canvas.style.transition = visible ? "opacity 600ms ease" : "none";
}

function applyCamera(runtimeState, command) {
  if (!command || !runtimeState.camera) return;
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
  old.canvas.style.opacity = "0";
  old.canvas.replaceWith(old.canvas.cloneNode(true));
  old.app.destroy();
  emit({ type: "disposed" });
}

async function initialize(message) {
  dispose();
  if (message.device && message.device !== "webgl2") {
    errorMessage("WEBGL_UNAVAILABLE", `unsupported device: ${message.device}`);
    return;
  }
  let canvas;
  try {
    canvas = resolveCanvas(message.canvas_id);
    configureCanvas(canvas);
    const app = createApplication(canvas);
    const camera = createCamera(app, getCameraDefinition(message.cameras, "TACTICAL"));
    runtime = {
      app,
      canvas,
      camera,
      cameras: message.cameras ?? {},
      sceneSequence: 0,
      paused: false,
      cameraPreset: "TACTICAL",
      liteURL: message.lite_url,
      liteTried: false,
      lowFpsReported: false,
      replacing: false,
      transform: message.transform,
      lastFrameAt: 0,
      lastStatsAt: performance.now(),
      lowFpsSince: 0,
      fpsSamples: [],
    };
    app.on("error", (detail) => errorMessage("CONTEXT_LOST", detail));
    app.on("postrender", () => {
      if (runtime?.app === app) recordFrame(runtime);
    });
    app.start();
    const scene = await loadSplat(app, message.scene_url, message.transform);
    if (runtime?.app !== app) return;
    runtime.splat = scene;
    applyCamera(runtime, { preset: "TACTICAL" });
    emit({
      type: "ready",
      fps: 0,
      gaussians: splatCount(scene.asset),
      device: "webgl2",
    });
  } catch (error) {
    if (runtime?.app) runtime.app.destroy();
    runtime = null;
    errorMessage(canvas ? "LOAD_FAILED" : "WEBGL_UNAVAILABLE", error);
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
    const replacement = await loadSplat(
      runtimeState.app,
      runtimeState.liteURL,
      runtimeState.transform,
    );
    if (runtime !== runtimeState) return;
    runtimeState.splat.entity.destroy();
    runtimeState.app.assets.remove(runtimeState.splat.asset);
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
    errorMessage("LOAD_FAILED", error);
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
        runtime.app.autoRender = !runtime.paused;
        if (!runtime.paused) runtime.app.renderNextFrame = true;
      }
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
