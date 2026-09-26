import * as pc from "../vendor/playcanvas.mjs";
import { applyBattleTransform, createBattleGrid, createSplatEntity, loadBattleProfile, loadSplatBundle } from "./battle_scene.mjs";
import { createCinematicEffects } from "./cinematic_effects.mjs";
import { installCapturedSkyExclusion, installGraySkybox } from "./gray_skybox.mjs";
import { applyColorGrade as applySplatColorGrade } from "./color_grade.mjs";
import { THEMES } from "./theme_grades.mjs";
import { installOrbitControls } from "./camera_controls.mjs";
import { applyCameraPreset, cameraPreset, cameraPresetNames } from "./grid_camera.mjs";
import { cellsJSON, installDebugPickMode } from "./debug_pick.mjs";
import { createTokenController, createDemoSnapshot, findDemoPath, selectDemoCells } from "./token_demo.mjs";
const GRID = Object.freeze({
  origin: [-6.096, -4.572],
  cell_m: 1.524,
  cols: 8,
  rows: 6,
  walkable: Array.from({ length: 48 }, (_, index) => index),
});
const WORLD_LAYER = pc.LAYERID_WORLD;
const PRESET_LABELS = Object.freeze({ COMBAT_EST: "Battle", TACTICAL: "Overview", TURN_FOCUS: "Focus", IMPACT: "Impact", KO: "KO", VICTORY: "Victory", SOURCE: "Scan", SURVEY: "Wide" });
const canvas = document.querySelector("#df-splat-viewer");
const statusNode = document.querySelector("#status");
const presetsNode = document.querySelector("#presets");
const gridButton = document.querySelector("#grid-toggle");
const charactersButton = document.querySelector("#characters-toggle");
const exportButton = document.querySelector("#export-picks");
const lodNode = document.querySelector("#lod-selector");
const sourceInfoNode = document.querySelector("#source-info");
const tiltButton = document.querySelector("#tilt-toggle");
const shakeButton = document.querySelector("#shake-effect");
const panButton = document.querySelector("#pan-effect");
const stopButton = document.querySelector("#stop-effects");
const pauseButton = document.querySelector("#pause-toggle");
const followNode = document.querySelector("#follow-selector");
const moveButton = document.querySelector("#move-demo");
const tokenStatusNode = document.querySelector("#token-status");
const strengthNode = document.querySelector("#effect-strength");
const themeNode = document.querySelector("#theme-grade");
const gradeStrengthNode = document.querySelector("#grade-strength");
let activeGrade = { theme: "neutral", strength: 0 };
let tokenController;

let app;
let camera;
let gridEntity;
let pickMode;
let gridVisible = true;
let baseStatus = "";
let orbitControls;
let cinematic;
let effectSeq = 0;
let demoSeq = 1;
let demoCells = [];
let demoState = null;
function setStatus(message, kind = "info") {
  baseStatus = message;
  statusNode.textContent = message;
  statusNode.dataset.kind = kind;
}
function gridLabel(grid) {
  const playable = Array.isArray(grid?.walkable) ? grid.walkable.length : 0;
  const excluded = Array.isArray(grid?.excluded) ? grid.excluded.length : 0;
  const feet = Math.abs(Number(grid?.cell_m) - 1.524) < 0.01 ? " · 5 ft cells" : "";
  return `${Number(grid?.cols) || 0}×${Number(grid?.rows) || 0} grid · ${playable} playable${excluded ? ` · ${excluded} terrain excluded` : ""}${feet}`;
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
    graphicsDeviceOptions: { deviceTypes: ["webgl2"], antialias: true, alpha: false },
  });
  app.setCanvasFillMode(pc.FILLMODE_FILL_WINDOW);
  app.setCanvasResolution(pc.RESOLUTION_AUTO);
  const resize = () => app.resizeCanvas?.(window.innerWidth ?? canvas.clientWidth, window.innerHeight ?? canvas.clientHeight);
  window.addEventListener?.("resize", resize);
  resize();
  app.on?.("destroy", () => window.removeEventListener?.("resize", resize));
  app.scene.gsplat.renderer = pc.GSPLAT_RENDERER_RASTER_CPU_SORT;
  app.scene.gsplat.lodMode = pc.GSPLAT_LODMODE_DISTANCE;
  app.scene.layers.getLayerById(WORLD_LAYER).enabled = true;
  app.scene.ambientLight = new pc.Color(0.28, 0.28, 0.28);
  console.info(`[splat viewer] grid MSAA: ${app.graphicsDevice?.backBuffer?.samples ?? "unknown"}`);
  return app;
}
function createViewerCamera() {
  camera = new pc.Entity("df-viewer-camera");
  camera.addComponent("camera", {
    clearColor: new pc.Color(0.47, 0.47, 0.47, 1),
    fov: 35,
    farClip: 1000,
    layers: [WORLD_LAYER, ...(pc.LAYERID_SKYBOX === undefined ? [] : [pc.LAYERID_SKYBOX])],
  });
  app.root.addChild(camera);
  const skybox = installGraySkybox(pc, app, camera);
  app.on?.("destroy", () => skybox.destroy());
  applyCameraPreset(camera, "TACTICAL");
}
function addPresetButtons(profile = null) {
  const names = [...new Set([...cameraPresetNames(), ...Object.keys(profile?.cameras ?? {})])];
  for (const name of names) {
    const button = document.createElement("button");
    button.type = "button";
    button.textContent = PRESET_LABELS[name] ?? name;
    button.dataset.preset = name;
    button.addEventListener("click", () => {
      tokenController?.clearFollow();
      if (followNode) followNode.value = "off";
      const definition = profile?.cameras?.[name];
      cinematic?.stop();
      if (definition) {
        const position = definition.position ?? [0, 7, 10];
        const target = definition.target ?? definition.look_at ?? [0, 0, 0];
        camera.setPosition(...position);
        camera.lookAt(...target);
        if (camera.camera) camera.camera.fov = Number(definition.fov ?? 35);
        orbitControls?.setTarget(target);
      } else applyCameraPreset(camera, name);
      if (!definition) orbitControls?.setTarget(cameraPreset(name).target);
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
  const report = (now) => {
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
  report.reset = () => { lastFrameAt = null; lastReportAt = null; samples = []; };
  return report;
}
const fpsReporter = createFPSReporter({
  logger: (p5) => {
    console.log(`[splat viewer] p5 fps: ${p5.toFixed(1)}`);
    if (statusNode.dataset.kind !== "error") {
      statusNode.textContent = `${baseStatus}\np5 fps: ${p5.toFixed(1)}`;
    }
  },
});
document.addEventListener?.("visibilitychange", () => { if (document.hidden) fpsReporter.reset(); });

function recordFPS() {
  if (document.hidden) { fpsReporter.reset(); return; }
  fpsReporter(performance.now());
}
function installMotionPreference() {
  const button = document.querySelector("#motion-toggle");
  if (!button) return;
  const initial = !window.matchMedia?.("(prefers-reduced-motion: reduce")?.matches;
  const display = (enabled) => {
    button.setAttribute("aria-pressed", String(enabled));
    if (shakeButton) shakeButton.disabled = !enabled;
    if (panButton) panButton.disabled = !enabled;
  };
  display(initial);
  button.addEventListener("click", () => {
    const enabled = button.getAttribute("aria-pressed") !== "true";
    cinematic?.send({ seq: ++effectSeq, reduced_motion: !enabled, enabled: true });
    tokenController?.setReducedMotion(!enabled);
    display(enabled);
  });
}
function installControls() {
  installMotionPreference();
  gridButton.addEventListener("click", () => {
    if (!gridEntity) return;
    gridVisible = !gridVisible;
    gridEntity.enabled = gridVisible;
    tokenController?.setGridVisible(gridVisible);
    gridButton.setAttribute("aria-pressed", String(gridVisible));
  });
  charactersButton?.addEventListener("click", () => { if (!tokenController) return; const enabled = charactersButton.getAttribute("aria-pressed") !== "true"; tokenController.setEnabled(enabled); charactersButton.setAttribute("aria-pressed", String(enabled)); });
  followNode?.addEventListener("change", () => { const id = followNode.value; if (id === "off") tokenController?.clearFollow(); else tokenController?.follow(id); });
  moveButton?.addEventListener("click", () => moveDemoToken());
  pauseButton?.addEventListener("click", () => {
    const paused = pauseButton.getAttribute("aria-pressed") !== "true";
    pauseButton.setAttribute("aria-pressed", String(paused));
    pauseButton.textContent = paused ? "Resume" : "Pause";
    tokenController?.pause(paused);
    cinematic?.pause(paused);
  });
  exportButton.addEventListener("click", () => pickMode?.download());
  tiltButton?.addEventListener("click", () => {
    const enabled = tiltButton.getAttribute("aria-pressed") !== "true";
    tiltButton.setAttribute("aria-pressed", String(enabled));
    const strength = Number(strengthNode?.value ?? 1);
    cinematic?.send({ seq: ++effectSeq, enabled: true, tilt_shift: { enabled, center: 0.5, band: 0.3, falloff: 0.35, blur_px: enabled ? 2 * strength : 0 } });
  });
  gradeStrengthNode?.addEventListener("input", () => {
    applyViewerGrade({ theme: themeNode?.value ?? activeGrade.theme, strength: Number(gradeStrengthNode.value), enabled: (themeNode?.value ?? activeGrade.theme) !== "neutral" });
  });
  strengthNode?.addEventListener("input", () => {
    if (tiltButton?.getAttribute("aria-pressed") === "true") cinematic?.send({ seq: ++effectSeq, enabled: true, tilt_shift: { enabled: true, center: 0.5, band: 0.3, falloff: 0.35, blur_px: 2 * Number(strengthNode.value) } });
  });
  themeNode?.addEventListener("change", () => applyViewerGrade({ theme: themeNode.value, strength: THEMES[themeNode.value].strength, enabled: themeNode.value !== "neutral" }));
  shakeButton?.addEventListener("click", () => cinematic?.send({ seq: ++effectSeq, enabled: true, shake: { amplitude_px: 8, duration_ms: 200 } }));
  panButton?.addEventListener("click", () => {
    tokenController?.clearFollow(); if (followNode) followNode.value = "off";
    const wide = panButton.dataset.wide !== "true";
    panButton.dataset.wide = String(wide);
    panButton.textContent = wide ? "Pan back" : "Pan wide";
    cinematic?.send({ seq: ++effectSeq, enabled: true, pan: { preset: wide ? "SURVEY" : "TACTICAL", duration_ms: 2500 } });
  });
  stopButton?.addEventListener("click", () => { tokenController?.clearFollow(); if (followNode) followNode.value = "off"; const pose = cinematic?.stop(); if (pose?.target) orbitControls?.setTarget(pose.target); });
}
function moveDemoToken() {
  if (!tokenController || !demoState) return;
  const id = followNode?.value === "villain" ? "villain" : "player";
  const entry = tokenController.getState().tokens.find((token) => token.id === id);
  if (!entry || entry.moving) return;
  const target = demoCells[(demoSeq * 7 + (id === "villain" ? 1 : 0)) % demoCells.length];
  const path = findDemoPath(demoState.grid, { c: entry.cell[0], r: entry.cell[1] }, target);
  if (!path.length) return;
  demoSeq += 1;
  const tokens = demoState.tokens.map((token) => token.id === id ? { ...token, cell: target, path, anim_seq: demoSeq } : { ...token, path: [] });
  demoState = { ...demoState, seq: demoSeq, tokens };
  tokenController.apply(demoState);
}
function applyViewerGrade(config = {}) {
  const theme = typeof config.theme === "string" && Object.hasOwn(THEMES, config.theme) ? config.theme : "neutral";
  activeGrade = { theme, strength: Number(config.strength ?? THEMES[theme].strength ?? 0) };
  if (gridEntity?.splatEntity) applySplatColorGrade(gridEntity.splatEntity, { theme, strength: activeGrade.strength }, config.enabled !== false);
  if (themeNode) themeNode.value = theme;
  if (gradeStrengthNode) gradeStrengthNode.value = String(activeGrade.strength);
}
function addLODOptions(profile, levelCount) {
  if (!lodNode) return;
  lodNode.replaceChildren();
  const automatic = document.createElement("option");
  automatic.value = "auto";
  automatic.textContent = "Auto (stream)";
  lodNode.append(automatic);
  const last = Math.max(0, levelCount - 1);
  const min = Math.max(0, Math.min(last, Number(profile?.lod_min ?? profile?.lod?.min ?? 0)));
  const max = Math.max(min, Math.min(last, Number(profile?.lod_max ?? profile?.lod?.max ?? last)));
  for (let level = min; level <= max; level += 1) {
    const option = document.createElement("option");
    option.value = String(level);
    option.textContent = `LOD ${level}`;
    lodNode.append(option);
  }
  lodNode.disabled = max === min;
  if (profile?.lod !== undefined) lodNode.value = String(Math.max(min, Math.min(max, Number(profile.lod))));
}
function applyLOD(level) {
  if (!gridEntity?.splatEntity?.gsplat) return;
  const profile = gridEntity.profile;
  const min = Number(profile?.lod_min ?? profile?.lod?.min ?? 0);
  const manifestMax = Math.max(0, Number(gridEntity.lodLevels || 1) - 1);
  const max = Math.max(0, Number(profile?.lod_max ?? profile?.lod?.max ?? manifestMax));
  const component = gridEntity.splatEntity.gsplat;
  component.lodRangeMin = level === "auto" ? min : Number(level);
  component.lodRangeMax = level === "auto" ? max : Number(level);
  setStatus(`${gridEntity.sourceLabel}\n${gridLabel(profile?.grid ?? GRID)} · LOD ${level === "auto" ? "Auto" : level} · chunks stream on demand (resident chunks vary with the camera).`);
}
function setupViewer(profile) {
  configureApp();
  createViewerCamera();
  addPresetButtons(profile);
  const initialCamera = profile?.cameras?.TACTICAL;
  if (initialCamera) {
    camera.setPosition(...(initialCamera.position ?? [0, 7, 10]));
    camera.lookAt(...(initialCamera.target ?? initialCamera.look_at ?? [0, 0, 0]));
    if (camera.camera) camera.camera.fov = Number(initialCamera.fov ?? 35);
  }
  installControls();
  orbitControls = installOrbitControls({ canvas, camera, target: initialCamera?.target ?? initialCamera?.look_at ?? [0, 0, 0] });
  cinematic = createCinematicEffects({ pc, app, camera, canvas, cameras: profile?.cameras ?? {}, reducedMotion: Boolean(window.matchMedia?.("(prefers-reduced-motion: reduce")?.matches), onPanComplete: (target) => orbitControls?.setTarget(target) });
  cinematic.setPose(initialCamera ?? { position: [0, 7, 10], target: [0, 0, 0], fov: 35 });
  const stopForGesture = () => { tokenController?.clearFollow(); if (followNode) followNode.value = "off"; const pose = cinematic?.stop(); if (pose?.target) orbitControls?.setTarget(pose.target); };
  canvas.addEventListener("pointerdown", stopForGesture, true);
  canvas.addEventListener("wheel", stopForGesture, { passive: true, capture: true });
  app.on("destroy", () => { canvas.removeEventListener("pointerdown", stopForGesture, true); canvas.removeEventListener("wheel", stopForGesture, true); orbitControls?.dispose(); });
  lodNode?.addEventListener("change", () => applyLOD(lodNode.value));
  app.on("postrender", recordFPS);
  app.start();
  if (themeNode) {
    themeNode.replaceChildren(...Object.entries(THEMES).map(([value, grade]) => { const option = document.createElement("option"); option.value = value; option.textContent = grade.label; return option; }));
  }
}
async function attachViewer(profile, source, bundle) {
    const levels = Math.max(1, Number(bundle.asset.resource?.octree?.lodLevels ?? 1));
    const activeGrid = bundle.grid ?? profile?.grid ?? GRID;
    addLODOptions(profile, levels);
    const defaultLOD = profile?.lod === undefined ? 0 : Math.max(0, Math.min(levels - 1, Number(profile.lod)));
    const scene = createSplatEntity(pc, app, bundle, { layers: [WORLD_LAYER], lodRangeMin: defaultLOD, lodRangeMax: defaultLOD, name: "df-viewer-splat" });
    installCapturedSkyExclusion(scene, bundle.collider, { floorY: profile?.voxel_collider?.floor_y ?? 0 });
    const battleGrid = createBattleGrid(pc, app, activeGrid, { name: "df-viewer-grid", lineWidth: 0.06, opacity: 0.95, collider: bundle.collider });
    gridEntity = battleGrid.entity;
    gridEntity.lodLevels = Number(bundle.asset.resource?.octree?.lodLevels ?? 0);
    if (profile?.transform) {
      applyBattleTransform(scene, profile.transform);
    }
    if (battleGrid.layer?.id !== undefined && !camera.camera.layers.includes(battleGrid.layer.id)) camera.camera.layers = [...camera.camera.layers, battleGrid.layer.id];
    if (battleGrid.depthLayer?.id !== undefined && !camera.camera.layers.includes(battleGrid.depthLayer.id)) camera.camera.layers = [...camera.camera.layers, battleGrid.depthLayer.id];
    gridEntity.splatEntity = scene;
    {
      const target = profile?.cameras?.TACTICAL?.target ?? [0, 0, 0];
      demoCells = selectDemoCells(activeGrid, target);
      tokenController = createTokenController({ pc, app, grid: activeGrid, layer: battleGrid.layer?.id, effects: cinematic, reducedMotion: Boolean(window.matchMedia?.("(prefers-reduced-motion: reduce")?.matches), onState: (value) => {
        if (followNode) followNode.value = value.followId ?? "off";
        if (tokenStatusNode) tokenStatusNode.textContent = `${value.count} combatants · ${value.moving ? "moving" : "ready"}${value.followId ? ` · following ${value.followId}` : ""}`;
      } });
      demoState = { ...createDemoSnapshot(demoCells, demoSeq), grid: activeGrid };
      tokenController.apply(demoState);
      const demoEnabled = new URLSearchParams(window.location.search).get("tokens") !== "off";
      tokenController.setEnabled(demoEnabled);
      charactersButton?.setAttribute("aria-pressed", String(demoEnabled));
      followNode?.replaceChildren(new Option("Off", "off"), new Option("Player", "player"), new Option("Villain", "villain"));
      app.on("update", (dt) => tokenController?.update(dt));
      app.on("destroy", () => tokenController?.destroy());
    }
    gridEntity.profile = { ...(profile ?? {}), grid: activeGrid };
    applyViewerGrade(profile?.color_grade ?? activeGrade);
    gridEntity.sourceLabel = [profile?.title ?? profile?.source?.title, profile?.source?.author].filter(Boolean).join(" · ") || "Loaded scene";
    if (sourceInfoNode) {
      const info = profile?.source ?? {};
      sourceInfoNode.textContent = [
        `URL: ${source}`,
        profile?.title ? `Profile: ${profile.title}` : "",
        info.title ? `Title: ${info.title}` : "",
        info.author ? `Author: ${info.author}` : "",
        info.credits ? `Credits: ${info.credits}` : "",
        info.license ? `License: ${info.license}` : "",
        info.registration_note ? `Registration: ${info.registration_note}` : "",
        profile?.registration_note ? `Registration: ${profile.registration_note}` : "",
      ].filter(Boolean).join("\n");
    }
    pickMode = installDebugPickMode({
      camera: camera.camera,
      canvas,
      grid: activeGrid,
      onPick: ({ c, r, selected, walkable }) => {
        setStatus(`${gridEntity.sourceLabel}\n${gridLabel(activeGrid)} · picked (${c}, ${r}) ${selected ? "on" : "off"}\n${cellsJSON(walkable)}`);
      },
    });
    exportButton.style.display = pickMode.enabled ? "inline-block" : "none";
    const debugLabel = pickMode.enabled ? "debug picking enabled" : "";
    const streamLabel = bundle.streaming ? "Scene ready · streaming detail" : "Scene ready";
    setStatus(`${gridEntity.sourceLabel}\n${gridLabel(activeGrid)} · LOD ${defaultLOD} · ${debugLabel}${debugLabel ? " · " : ""}${streamLabel}`);
}
async function start() {
  let profile;
  try { profile = await loadBattleProfile(window.location.search, window.location.href); }
  catch (error) { setStatus(error?.message ?? String(error), "error"); return; }
  const source = profile?.scene_url ?? querySource();
  if (!source) { setStatus("Missing source. Open with ?src=scene.ply or ?battle={...}.", "error"); return; }
  if (!/(?:\.(?:ply|sog)|\/(?:meta|lod-meta)\.json)(?:$|[?#])/i.test(source)) {
    setStatus("Source must be a .ply or .sog asset (or an LOD manifest).", "error"); return;
  }
  setStatus(`Loading ${source}\n${gridLabel(profile?.grid ?? GRID)} ready; debug picks: ${new URLSearchParams(window.location.search).has("debug") ? "on" : "off"}`);
  try {
    setupViewer(profile);
    const bundle = await loadSplatBundle(pc, app, profile ?? source, {
      lodMetaURL: profile?.lod_meta_url,
      metaURL: profile?.meta_url,
      grid: profile?.grid,
      voxelURL: profile?.voxel_collider_url,
      transform: profile?.transform,
      voxelOptions: profile?.voxel_collider_options,
    });
    await attachViewer(profile, source, bundle);
  } catch (error) {
    setStatus(`Splat load failed: ${error?.message ?? error}`, "error");
    console.error("[splat viewer] load failed", error);
  }
}

start();
