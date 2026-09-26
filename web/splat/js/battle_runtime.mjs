import * as pc from "../vendor/playcanvas.mjs";
import { createBattleGrid, createSplatEntity, loadSplatBundle } from "./battle_scene.mjs";
import { createCinematicEffects } from "./cinematic_effects.mjs";
import { installCapturedSkyExclusion, installGraySkybox } from "./gray_skybox.mjs";
import { applyColorGrade, isKnownTheme } from "./color_grade.mjs";
import { createCanvasSurface } from "./canvas_surface.mjs";
import { createRuntimeStats } from "./runtime_stats.mjs";
import { acceptScene, attachTokens, applyCamera, tokenSnapshot } from "./token_scene.mjs";
import { installOrbitControls } from "./camera_controls.mjs";

const WORLD_LAYER = pc.LAYERID_WORLD;
const DEFAULT_CAMERA = { position: [0, 3, 6], target: [0, 1, 0], fov: 35 };
const OWNED_CANVASES = new WeakMap();
const ASSET_URL = /(?:\.(?:ply|sog)|\/(?:meta|lod-meta)\.json)(?:$|[?#])/i;
function abortError() { const error = new Error("battle load cancelled or superseded"); error.name = "AbortError"; return error; }
function copy(value) { return value == null ? value : JSON.parse(JSON.stringify(value)); }
function definition(cameras, preset) { return cameras?.[preset] ?? cameras?.TACTICAL ?? DEFAULT_CAMERA; }
function releaseAsset(app, asset) { if (asset) { app.assets?.remove?.(asset); asset.unload?.(); } }
function count(asset) { const data = asset?.resource?.gsplatData ?? asset?.resource; return Number(data?.numSplats ?? data?.numPoints ?? data?.count ?? 0); }
function abortable(promise, signal, onLate = () => {}) {
  return new Promise((resolve, reject) => {
    const abort = () => reject(abortError());
    if (signal.aborted) abort(); else signal.addEventListener("abort", abort, { once: true });
    Promise.resolve(promise).then(value => {
      signal.removeEventListener("abort", abort);
      if (signal.aborted) onLate(value); else resolve(value);
    }, error => { signal.removeEventListener("abort", abort); if (!signal.aborted) reject(error); });
  });
}
function resolveProfile(source, base) {
  if (!source || typeof source !== "object" || Array.isArray(source)) throw new TypeError("battle profile must be an object");
  const profile = copy(source);
  for (const key of ["scene_url", "lite_url", "meta_url", "lod_meta_url", "voxel_collider_url"]) {
    if (profile[key]) profile[key] = new URL(profile[key], base).href;
  }
  if (profile.voxel_collider?.url) profile.voxel_collider.url = new URL(profile.voxel_collider.url, base).href;
  if (typeof profile.scene_url !== "string" || !ASSET_URL.test(profile.scene_url)) throw new Error("scene_url must be a PLY, SOG or splat manifest");
  if (profile.device && profile.device !== "webgl2") throw new Error(`unsupported device: ${profile.device}`);
  return profile;
}
async function readProfile(source, signal, base) {
  if (source && typeof source === "object") return resolveProfile(source, base);
  if (typeof source !== "string" || !source.trim()) throw new TypeError("battle profile is required");
  if (source.trim().startsWith("{")) return resolveProfile(JSON.parse(source), base);
  const url = new URL(source, base).href;
  if (ASSET_URL.test(url)) return { scene_url: url };
  const response = await fetch(url, { signal });
  if (!response.ok) throw new Error(`battle profile HTTP ${response.status}`);
  return resolveProfile(await response.json(), response.url || url);
}
function createApplication(canvas) {
  const app = new pc.Application(canvas, { graphicsDeviceOptions: { deviceTypes: ["webgl2"], antialias: true, alpha: true } });
  if (app.graphicsDevice) app.graphicsDevice.maxPixelRatio = Math.min(2, globalThis.devicePixelRatio || 1);
  app.setCanvasResolution?.(pc.RESOLUTION_AUTO);
  app.scene.gsplat.renderer = pc.GSPLAT_RENDERER_RASTER_CPU_SORT;
  app.scene.gsplat.lodMode = pc.GSPLAT_LODMODE_DISTANCE;
  app.scene.layers.getLayerById(WORLD_LAYER).enabled = true;
  return app;
}
function createCamera(app) {
  const camera = new pc.Entity("df-splat-camera", app);
  camera.addComponent("camera", { clearColor: new pc.Color(.47, .47, .47, 1), fov: 35, farClip: 1000,
    layers: [WORLD_LAYER, ...(pc.LAYERID_SKYBOX === undefined ? [] : [pc.LAYERID_SKYBOX])] });
  app.root.addChild(camera);
  return camera;
}
function transform(entity, value = {}) {
  const scale = Number(value.scale ?? 1), at = value.translate ?? [0, 0, 0];
  entity.setLocalScale(scale, scale, scale);
  entity.setLocalPosition(Number(at[0]), Number(at[1]) + Number(value.offset_y ?? 0), Number(at[2]));
  entity.setLocalEulerAngles(0, Number(value.rot_y_deg ?? 0), Number(entity.getLocalEulerAngles?.().z ?? 0));
}

/** createBattleRuntime mounts an independent renderer without replacing the caller's canvas. */
export function createBattleRuntime({ canvas, layout = "embedded", cameraControls = false, reducedMotion, baseURL } = {}) {
  if (!canvas || typeof canvas.getContext !== "function" || typeof canvas.addEventListener !== "function") throw new TypeError("an HTML canvas is required");
  if (OWNED_CANVASES.has(canvas)) throw new Error("canvas already has a battle runtime");
  const base = baseURL ?? canvas.ownerDocument?.baseURI ?? globalThis.document?.baseURI ?? globalThis.location?.href;
  const surface = createCanvasSurface({ canvas, layout });
  let app;
  try { app = createApplication(canvas); } catch (error) { surface.dispose(); throw error; }
  OWNED_CANVASES.set(canvas, app);
  const listeners = new Set();
  const state = { app, canvas, cameras: {}, camera: null, cameraPreset: "TACTICAL", sceneSequence: 0, cameraSequence: 0,
    effectSequence: 0, dispatchEffectSequence: 0, pendingScene: null, pendingCamera: null, tokens: null, grid: null, activeGrid: null,
    splat: null, collider: null, effects: null, orbit: null, stats: null, skybox: null, profile: null,
    loadAbort: null, loadGeneration: 0, ready: false, disposed: false, paused: false, visible: layout !== "fullscreen",
    gridVisible: true, charactersVisible: true, effectsEnabled: true, cameraControls: Boolean(cameraControls),
    colorGrade: null, gradeOverride: false, lod: "auto", lodOverride: false, lodLevelCount: 1,
    reducedMotion: reducedMotion ?? Boolean(globalThis.matchMedia?.("(prefers-reduced-motion: reduce)")?.matches), tokenSignature: "" };
  // Entity's default app is global in PlayCanvas; async loads need an explicit owner.
  const engine = { ...pc, app, Entity: class OwnedEntity extends pc.Entity { constructor(name) { super(name, app); } } };
  const context = { app, engine, canvas, surface, base, listeners, state };
  for (const [name, operation] of Object.entries(RUNTIME_OPERATIONS)) context[name] = operation.bind(null, context);
  initialize(context);
  const { load, send, setCamera, setTokens, follow, setVisible, setGridVisible, setCharactersVisible,
    setCameraControlsEnabled, setEffectsEnabled, setColorGrade, effects, pause, setLOD, dispose } = context;
  return Object.freeze({ load, send, onEvent: context.onEvent, getState: () => context.snapshot(true),
    setCamera, setTokens, follow, setVisible, setGridVisible, setCharactersVisible, setCameraControlsEnabled,
    setEffectsEnabled, setColorGrade, effects, pause, setLOD, resize: surface.resize, dispose });
}
function initialize(context) {
  const { app, surface, state, error } = context;
  surface.setResizeHandler(size => { if (size.width > 0 && size.height > 0) app.graphicsDevice?.resizeCanvas?.(size.width, size.height); });
  surface.resize();
  app.on("update", dt => { if (!state.disposed) state.tokens?.update(dt); });
  app.on("postrender", () => state.stats?.tick(undefined, Boolean(globalThis.document?.hidden)));
  app.on("error", detail => error("CONTEXT_LOST", detail)); app.start();
}
function onEvent(context, listener) {
  const { requireLive, listeners } = context;
  requireLive();
  if (typeof listener !== "function") throw new TypeError("listener must be a function");
  listeners.add(listener);
  return () => listeners.delete(listener);
}

function snapshot(context, includeGrid = false) {
  const { state } = context;
  return { ready: state.ready, disposed: state.disposed, paused: state.paused, visible: state.visible,
    cameraControls: state.cameraControls, gridVisible: state.gridVisible, charactersVisible: state.charactersVisible,
    effectsEnabled: state.effectsEnabled, reducedMotion: state.reducedMotion, cameraPreset: state.cameraPreset,
    tokens: copy(tokenSnapshot(state)), cameras: copy(state.cameras), colorGrade: copy(state.colorGrade),
    source: copy(state.profile?.source ?? null), title: state.profile?.title ?? state.profile?.source?.title ?? "",
    tiltShift: copy(state.tiltShift ?? { enabled: false }), lod: state.lod, lodLevelCount: state.lodLevelCount, ...(includeGrid ? { grid: copy(state.activeGrid) } : {}) };
}

function emit(context, type, detail = {}) {
  const { app, state, listeners, snapshot } = context;
  if (state.ready) app.renderNextFrame = true;
  const event = { type, ...detail, state: snapshot(type === "ready") };
  for (const listener of listeners) { try { listener(event); } catch (error) { queueMicrotask(() => { throw error; }); } }
}

function requireLive(context) {
  const { state } = context; if (state.disposed) throw new Error("battle runtime disposed"); }

function error(context, code, detail) {
  const { emit } = context; emit("error", { code, detail: String(detail?.message ?? detail) }); }

function unbindInput(context) {
  const { canvas, state, takeover } = context;
  state.orbit?.dispose(); state.orbit = null;
  canvas.removeEventListener("pointerdown", takeover, true); canvas.removeEventListener("wheel", takeover, true);
}

function takeover(context) {
  const { state, emit } = context;
  state.tokens?.clearFollow(); const pose = state.effects?.stop();
  if (pose?.target) state.orbit?.setTarget(pose.target); emit("state");
}

function bindInput(context) {
  const { canvas, state, unbindInput, takeover } = context;
  unbindInput();
  if (!state.cameraControls || !state.ready) return;
  canvas.addEventListener("pointerdown", takeover, true);
  canvas.addEventListener("wheel", takeover, { passive: true, capture: true });
  state.orbit = installOrbitControls({ canvas, camera: state.camera, target: state.effects.getPose()?.target ?? definition(state.cameras).target ?? [0, 0, 0] });
}

function disposeScene(context) {
  const { app, state, unbindInput } = context;
  unbindInput(); state.stats?.dispose(); state.stats = null;
  state.tokens?.destroy(); state.tokens = null; state.effects?.destroy(); state.effects = null;
  state.grid?.entity?.destroy?.(); state.grid?.depthProxy?.destroy?.();
  for (const layer of [state.grid?.layer, state.grid?.depthLayer]) if (layer) app.scene.layers?.remove?.(layer);
  state.grid = null; state.activeGrid = null;
  state.splat?.entity?.destroy?.(); releaseAsset(app, state.splat?.asset); state.splat = null;
  state.skybox?.destroy(); state.skybox = null;
  if (state.camera?.camera) state.camera.camera.layers = [WORLD_LAYER, ...(pc.LAYERID_SKYBOX === undefined ? [] : [pc.LAYERID_SKYBOX])];
}

function tokenChanged(context, value) {
  const { state, emit } = context;
  const signature = JSON.stringify([value.count, value.moving, value.followId, value.tokens.map(token => [token.id, token.name, token.cell])]);
  if (signature !== state.tokenSignature) { state.tokenSignature = signature; emit("tokens"); }
}

function readyDetail(context, bundle) {
  const { app, state } = context;
  return { fps: 0, gaussians: count(bundle.asset), device: "webgl2", streaming: Boolean(bundle.streaming),
    playable: state.activeGrid?.walkable?.length ?? 0, terrain_excluded: state.activeGrid?.excluded?.length ?? 0,
    antialias_samples: app.graphicsDevice?.backBuffer?.samples ?? 0, status: bundle.streaming ? "manifest loaded; chunks stream on demand" : "scene loaded" };
}

function attachBundle(context, profile, bundle) {
  const { app, engine, state, emit, bindInput, tokenChanged, readyDetail, downgradeForFps, applyLOD } = context;
  const entity = createSplatEntity(engine, app, bundle, { layers: [WORLD_LAYER], name: "df-splat-scene" });
  state.collider = bundle.collider; state.splat = { entity, asset: bundle.asset };
  installCapturedSkyExclusion(entity, bundle.collider, { floorY: Number(profile.voxel_collider_options?.floor_y ?? profile.voxel_collider?.floor_y ?? 0) });
  applyColorGrade(entity, state.colorGrade, state.effectsEnabled); transform(entity, profile.transform);
  state.activeGrid = bundle.grid ?? profile.grid ?? null;
  if (state.activeGrid) {
    state.grid = createBattleGrid(engine, app, state.activeGrid, { name: "df-battle-grid", lineWidth: .06, opacity: .95, collider: bundle.collider });
    for (const layer of [state.grid.layer, state.grid.depthLayer]) if (layer?.id !== undefined) state.camera.camera.layers = [...new Set([...state.camera.camera.layers, layer.id])];
    state.onTokens = tokenChanged; attachTokens(engine, state, state.activeGrid, state.grid.layer?.id, profile);
    state.grid.entity.enabled = state.gridVisible; state.tokens.setEnabled(state.charactersVisible); state.tokens.setGridVisible(state.gridVisible);
  }
  state.lodLevelCount = Math.max(1, Number(bundle.asset.resource?.octree?.lodLevels ?? 1));
  if (!state.lodOverride) state.lod = profile.lod ?? "auto";
  applyLOD(); state.ready = true; app.renderNextFrame = true;
  if (state.pendingCamera) applyCamera(state, state.pendingCamera);
  state.effects.pause(state.paused); bindInput();
  state.stats = createRuntimeStats({ hasLite: Boolean(profile.lite_url), downgrade: downgradeForFps, emit: message => emit(message.type, message) });
  emit("ready", readyDetail(bundle));
}

async function load(context, source) {
  const { app, engine, canvas, base, state, snapshot, emit, requireLive, error, disposeScene, attachBundle } = context;
  requireLive(); state.loadAbort?.abort(); const controller = new AbortController(); state.loadAbort = controller;
  const generation = ++state.loadGeneration;
  state.ready = false; disposeScene();
  if (generation > 1) { state.sceneSequence = 0; state.cameraSequence = 0; state.pendingScene = null; state.pendingCamera = null; }
  state.effectSequence = 0; state.dispatchEffectSequence = 0; state.tiltShift = { enabled: false }; state.tokenSignature = ""; emit("loading");
  try {
    const profile = await abortable(readProfile(source, controller.signal, base), controller.signal);
    state.profile = profile; state.cameras = copy(profile.cameras ?? { TACTICAL: DEFAULT_CAMERA });
    state.cameraPreset = "TACTICAL"; if (!state.gradeOverride) state.colorGrade = copy(profile.color_grade ?? null);
    state.camera ??= createCamera(app); state.skybox = installGraySkybox(engine, app, state.camera);
    const pose = definition(state.cameras); state.camera.camera.farClip = Number(pose.far ?? 1000);
    state.effects = createCinematicEffects({ pc: engine, app, camera: state.camera, canvas, cameras: state.cameras, reducedMotion: state.reducedMotion,
      onPanComplete: target => state.orbit?.setTarget(target) });
    state.effects.setPose(pose); state.effects.send({ seq: ++state.dispatchEffectSequence, enabled: state.effectsEnabled });
    const bundle = await abortable(loadSplatBundle(pc, app, profile, { grid: profile.grid, lodMetaURL: profile.lod_meta_url,
      metaURL: profile.meta_url, voxelURL: profile.voxel_collider_url, transform: profile.transform,
      voxelOptions: profile.voxel_collider_options }), controller.signal, value => releaseAsset(app, value.asset));
    if (generation !== state.loadGeneration || state.disposed) { releaseAsset(app, bundle.asset); throw abortError(); }
    attachBundle(profile, bundle); return snapshot(true);
  } catch (failure) {
    if (controller.signal.aborted || generation !== state.loadGeneration || state.disposed) throw abortError();
    disposeScene(); error("LOAD_FAILED", failure); throw failure;
  }
}

async function downgradeForFps(context) {
  const { app, engine, state, emit, readyDetail } = context;
  const generation = state.loadGeneration, controller = state.loadAbort;
  const bundle = await abortable(loadSplatBundle(pc, app, state.profile.lite_url), controller.signal, value => releaseAsset(app, value.asset));
  if (state.disposed || generation !== state.loadGeneration) { releaseAsset(app, bundle.asset); return; }
  const entity = createSplatEntity(engine, app, bundle, { layers: [WORLD_LAYER], name: "df-splat-scene-lite" });
  installCapturedSkyExclusion(entity, bundle.collider ?? state.collider, { floorY: Number(state.profile.voxel_collider?.floor_y ?? 0) });
  applyColorGrade(entity, state.colorGrade, state.effectsEnabled); transform(entity, state.profile.transform);
  state.splat.entity.destroy(); releaseAsset(app, state.splat.asset); state.splat = { entity, asset: bundle.asset };
  emit("ready", readyDetail(bundle));
}

function setCamera(context, command) {
  const { state, emit, requireLive } = context;
  requireLive(); const value = typeof command === "string" ? { preset: command } : { ...command };
  if (value.preset && state.ready && !Object.hasOwn(state.cameras, value.preset)) return false;
  const previous = Math.max(state.cameraSequence, !state.ready ? Number(state.pendingCamera?.seq ?? 0) : 0);
  if (value.seq === undefined) value.seq = previous + 1;
  if (!Number.isSafeInteger(value.seq) || value.seq < 0) throw new TypeError("camera sequence must be a nonnegative safe integer");
  if (value.seq > 0 && value.seq <= previous) return false;
  if (!value.preset && value.follow === undefined && !value.focus_token_id) return false;
  if (state.ready && value.focus_token_id && !tokenSnapshot(state).tokens.some(token => token.id === value.focus_token_id)) return false;
  if (!state.ready) state.pendingCamera = value; else applyCamera(state, value);
  emit("state"); return true;
}

function setTokens(context, tokens, seq) {
  const { state, emit, requireLive } = context;
  requireLive(); if (!Array.isArray(tokens)) throw new TypeError("tokens must be an array");
  const next = seq ?? state.sceneSequence + 1;
  if (!Number.isSafeInteger(next) || next < 0) throw new TypeError("token sequence must be a nonnegative safe integer");
  if (next <= state.sceneSequence) return false;
  acceptScene(state, { seq: next, tokens: copy(tokens) }); emit("state"); return true;
}

function follow(context, id) {
  const { setCamera } = context; return setCamera(id == null ? { follow: false } : { follow: true, focus_token_id: id }); }

function setVisible(context, on) {
  const { canvas, state, emit, requireLive } = context; requireLive(); state.visible = Boolean(on); canvas.style.opacity = state.visible ? "1" : "0"; canvas.style.transition = state.visible ? "opacity 600ms ease" : "none"; emit("state"); }

function setGridVisible(context, on) {
  const { state, emit, requireLive } = context; requireLive(); state.gridVisible = Boolean(on); if (state.grid?.entity) state.grid.entity.enabled = state.gridVisible; state.tokens?.setGridVisible(state.gridVisible); emit("state"); }

function setCharactersVisible(context, on) {
  const { state, emit, requireLive } = context; requireLive(); state.charactersVisible = Boolean(on); state.tokens?.setEnabled(state.charactersVisible); emit("state"); }

function setCameraControlsEnabled(context, on) {
  const { state, emit, requireLive, bindInput } = context; requireLive(); state.cameraControls = Boolean(on); bindInput(); emit("state"); }

function effects(context, command = {}) {
  const { state, emit, requireLive } = context;
  requireLive(); if (command.color_grade?.theme && !isKnownTheme(command.color_grade.theme)) return false;
  const value = { ...command, seq: command.seq ?? state.effectSequence + 1 };
  if (!Number.isSafeInteger(value.seq) || value.seq <= state.effectSequence) return false;
  if (!state.effects || !state.effects.send({ ...value, seq: state.dispatchEffectSequence + 1 })) return false;
  state.effectSequence = value.seq; state.dispatchEffectSequence += 1;
  if (value.pan || value.stop) state.tokens?.clearFollow();
  if (value.reduced_motion !== undefined) { state.reducedMotion = Boolean(value.reduced_motion); state.tokens?.setReducedMotion(state.reducedMotion); }
  if (value.enabled !== undefined) state.effectsEnabled = Boolean(value.enabled);
  if (value.color_grade) { state.colorGrade = copy(value.color_grade); state.gradeOverride = true; }
  if (value.tilt_shift) state.tiltShift = copy(value.tilt_shift);
  if (value.enabled === false) state.tiltShift = { enabled: false };
  applyColorGrade(state.splat?.entity, state.colorGrade, state.effectsEnabled); emit("state"); return true;
}

function setEffectsEnabled(context, on) {
  const { state, emit, requireLive, effects } = context;
  requireLive(); state.effectsEnabled = Boolean(on);
  if (state.effects) effects({ enabled: state.effectsEnabled });
  else emit("state");
}

function setColorGrade(context, config) {
  const { state, emit, requireLive } = context; requireLive(); if (config?.theme && !isKnownTheme(config.theme)) return false; state.colorGrade = copy(config ?? null); state.gradeOverride = true; applyColorGrade(state.splat?.entity, state.colorGrade, state.effectsEnabled); emit("state"); return true; }

function pause(context, on) {
  const { app, state, emit, requireLive } = context; requireLive(); state.paused = Boolean(on); state.effects?.pause(state.paused); state.tokens?.pause(state.paused); state.stats?.reset(); app.autoRender = !state.paused; if (!state.paused) app.renderNextFrame = true; emit("state"); }

function applyLOD(context) {
  const { state } = context;
  const component = state.splat?.entity?.gsplat; if (!component) return;
  const level = state.lod === "auto" ? "auto" : Math.max(0, Math.min(state.lodLevelCount - 1, Number(state.lod)));
  state.lod = level; component.lodRangeMin = level === "auto" ? 0 : level; component.lodRangeMax = level === "auto" ? state.lodLevelCount - 1 : level;
}

function setLOD(context, value) {
  const { state, emit, requireLive, applyLOD } = context; requireLive(); if (value !== "auto" && (!Number.isInteger(value) || value < 0)) throw new TypeError("LOD must be auto or a nonnegative integer"); state.lod = value; state.lodOverride = true; applyLOD(); emit("state"); }

function send(context, raw) {
  const { state, emit, requireLive, error, load, effects, pause, dispose } = context;
  let message; try { message = typeof raw === "string" ? JSON.parse(raw) : raw; } catch (failure) { error("INVALID_MESSAGE", failure); return; }
  if (!message || message.v !== 1 || typeof message.type !== "string") { error("INVALID_MESSAGE", "expected v:1 and a message type"); return; }
  if (message.type === "dispose") { dispose(); return; }
  requireLive();
  switch (message.type) {
    case "init": void load(message.profile ?? message).catch(() => {}); break;
    case "scene": acceptScene(state, copy(message)); emit("scene"); break;
    case "pause": pause(message.on); break;
    case "effects": effects(message); break;
    default: error("INVALID_MESSAGE", `unknown type: ${message.type}`);
  }
}

function dispose(context) {
  const { app, canvas, surface, listeners, state, emit, disposeScene } = context;
  if (state.disposed) return;
  state.disposed = true; state.ready = false; state.loadAbort?.abort(); disposeScene();
  emit("disposed"); listeners.clear(); app.destroy(); surface.dispose(); OWNED_CANVASES.delete(canvas);
}

const RUNTIME_OPERATIONS = { snapshot, emit, requireLive, error, unbindInput, takeover, bindInput, disposeScene, tokenChanged, readyDetail, attachBundle, load, downgradeForFps, setCamera, setTokens, follow, setVisible, setGridVisible, setCharactersVisible, setCameraControlsEnabled, effects, setEffectsEnabled, setColorGrade, pause, applyLOD, setLOD, send, dispose, onEvent };
