import * as pc from "../vendor/playcanvas.mjs";
import { createGridOverlay } from "./grid_overlay.mjs";
import { applyCameraPreset, cameraPresetNames } from "./grid_camera.mjs";
import { cellsJSON, installDebugPickMode } from "./debug_pick.mjs";

const GRID = Object.freeze({
  origin: [-6.096, -4.572],
  cell_m: 1.524,
  cols: 8,
  rows: 6,
  walkable: Array.from({ length: 48 }, (_, index) => index),
});
const WORLD_LAYER = pc.LAYERID_WORLD;
const canvas = document.querySelector("#df-splat-viewer");
const statusNode = document.querySelector("#status");
const presetsNode = document.querySelector("#presets");
const gridButton = document.querySelector("#grid-toggle");
const exportButton = document.querySelector("#export-picks");

let app;
let camera;
let gridEntity;
let pickMode;
let gridVisible = true;
function setStatus(message, kind = "info") {
  statusNode.textContent = message;
  statusNode.dataset.kind = kind;
}

function querySource() {
  const source = new URLSearchParams(window.location.search).get("src");
  if (!source) return "";
  try {
    return new URL(source, window.location.href).href;
  } catch {
    return "";
  }
}

function configureApp() {
  app = new pc.Application(canvas, {
    graphicsDeviceOptions: { deviceTypes: ["webgl2"], antialias: false, alpha: false },
  });
  app.setCanvasFillMode(pc.FILLMODE_FILL_WINDOW);
  app.setCanvasResolution(pc.RESOLUTION_AUTO);
  app.scene.gsplat.renderer = pc.GSPLAT_RENDERER_RASTER_CPU_SORT;
  app.scene.gsplat.lodMode = pc.GSPLAT_LODMODE_DISTANCE;
  app.scene.layers.getLayerById(WORLD_LAYER).enabled = true;
  app.scene.ambientLight = new pc.Color(0.28, 0.28, 0.28);
  return app;
}

function createViewerCamera() {
  camera = new pc.Entity("df-viewer-camera");
  camera.addComponent("camera", {
    clearColor: new pc.Color(0.025, 0.04, 0.065, 1),
    fov: 35,
    farClip: 1000,
    layers: [WORLD_LAYER],
  });
  app.root.addChild(camera);
  applyCameraPreset(camera, "TACTICAL");
}

function addSplat(source) {
  return new Promise((resolve, reject) => {
    const asset = new pc.Asset("viewer-splat", "gsplat", { url: source });
    asset.once("load", () => resolve(asset));
    asset.once("error", (error) => reject(error instanceof Error ? error : new Error(String(error))));
    app.assets.add(asset);
    app.assets.load(asset);
  });
}

function addPresetButtons() {
  for (const name of cameraPresetNames()) {
    const button = document.createElement("button");
    button.type = "button";
    button.textContent = name;
    button.dataset.preset = name;
    button.addEventListener("click", () => {
      applyCameraPreset(camera, name);
      for (const item of presetsNode.children) item.setAttribute("aria-pressed", String(item === button));
    });
    presetsNode.append(button);
  }
  presetsNode.querySelector('[data-preset="TACTICAL"]')?.setAttribute("aria-pressed", "true");
}

/** Creates a timestamp-driven p5 FPS sampler and logger. */
export function createFPSReporter({ intervalMs = 2000, logger = () => {} } = {}) {
  let lastFrameAt = null;
  let lastReportAt = null;
  let samples = [];
  return (now) => {
    if (!Number.isFinite(now)) return null;
    if (lastFrameAt === null) {
      lastFrameAt = now;
      lastReportAt = now;
      return null;
    }
    const elapsed = now - lastFrameAt;
    lastFrameAt = now;
    if (elapsed <= 0) return null;
    samples.push(1000 / elapsed);
    if (now - lastReportAt < intervalMs) return null;
    const sorted = [...samples].sort((left, right) => left - right);
    const p5 = sorted[Math.floor(Math.max(0, sorted.length - 1) * 0.05)] ?? 0;
    samples = [];
    lastReportAt = now;
    logger(p5);
    return p5;
  };
}

const fpsReporter = createFPSReporter({
  logger: (p5) => {
    console.log(`[splat viewer] p5 fps: ${p5.toFixed(1)}`);
    if (statusNode.dataset.kind !== "error") {
      statusNode.textContent = `${statusNode.textContent.split("\n")[0]}\np5 fps: ${p5.toFixed(1)}`;
    }
  },
});

function recordFPS() {
  fpsReporter(performance.now());
}

function installControls() {
  gridButton.addEventListener("click", () => {
    if (!gridEntity) return;
    gridVisible = !gridVisible;
    gridEntity.enabled = gridVisible;
    gridButton.setAttribute("aria-pressed", String(gridVisible));
  });
  exportButton.addEventListener("click", () => pickMode?.download());
}

async function start() {
  const source = querySource();
  if (!source) {
    setStatus("Missing source. Open with ?src=scene.ply&debug.", "error");
    return;
  }
  if (!/\.(?:ply|sog)(?:$|[?#])/i.test(source)) {
    setStatus("Source must be a .ply or .sog asset.", "error");
    return;
  }
  setStatus(`Loading ${source}\n8×6 grid ready; debug picks: ${new URLSearchParams(window.location.search).has("debug") ? "on" : "off"}`);
  try {
    configureApp();
    createViewerCamera();
    addPresetButtons();
    installControls();
    app.on("postrender", recordFPS);
    app.start();
    const asset = await addSplat(source);
    const scene = new pc.Entity("df-viewer-splat");
    scene.addComponent("gsplat", { asset, layers: [WORLD_LAYER] });
    app.root.addChild(scene);
    gridEntity = createGridOverlay(pc, app, GRID, { layers: [WORLD_LAYER], name: "df-viewer-grid" });
    pickMode = installDebugPickMode({
      camera,
      canvas,
      grid: GRID,
      onPick: ({ c, r, selected, walkable }) => {
        setStatus(`Loaded ${source}\n8×6 grid · picked (${c}, ${r}) ${selected ? "on" : "off"}\n${cellsJSON(walkable)}`);
      },
    });
    setStatus(`Loaded ${source}\n8×6 grid · ${pickMode.enabled ? "debug picking enabled" : "add &debug to pick cells"}`);
  } catch (error) {
    setStatus(`Splat load failed: ${error?.message ?? error}`, "error");
    console.error("[splat viewer] load failed", error);
  }
}

start();
