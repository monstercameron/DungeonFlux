import { createGridOverlay } from "./grid_overlay.mjs";
import { loadVoxelCollider } from "./voxel_collider.mjs";
import { createVoxelDepthOccluder } from "./voxel_occlusion.mjs";

const SCENE_EXTENSIONS = /(?:\.(?:ply|sog)|\/(?:meta|lod-meta)\.json)(?:$|[?#])/i;

function addAsset(app, pc, name, url, type) {
  return new Promise((resolve, reject) => {
    const asset = new pc.Asset(name, type, { url });
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

/** Loads a PlayCanvas splat asset or its streaming octree manifest. */
export async function loadSplatBundle(pc, app, source, options = {}) {
  const sceneURL = typeof source === "string" ? source : source?.scene_url;
  if (!SCENE_EXTENSIONS.test(sceneURL ?? "")) {
    throw new Error("Source must be a .ply or .sog asset.");
  }
  const lodMetaURL = options.lodMetaURL ?? source?.lod_meta_url;
  const metaURL = options.metaURL ?? source?.meta_url;
  const assetURLValue = lodMetaURL || sceneURL;
  const asset = await addAsset(app, pc, "df-splat-scene", assetURLValue, "gsplat");
  try {
    let metadata = null;
    if (metaURL) metadata = (await addAsset(app, pc, "df-splat-meta", metaURL, "json")).resource;
    const voxelConfig = source?.voxel_collider;
    const voxelURL = options.voxelURL || source?.voxel_collider_url || source?.voxelColliderURL || voxelConfig?.url;
    const colliderDeclared = Boolean(voxelConfig || source?.voxel_collider_url || source?.voxelColliderURL || options.voxelURL);
    if (colliderDeclared && (!voxelURL || (voxelConfig !== undefined && (typeof voxelConfig !== "object" || Array.isArray(voxelConfig))))) throw new Error("voxel collider declaration must include a .voxel.json URL");
    const collider = voxelURL ? await loadVoxelCollider(voxelURL, { ...(voxelConfig ?? {}), ...options.voxelOptions, transform: options.transform ?? voxelConfig?.transform ?? source?.transform }) : null;
    const filterOptions = { ...(voxelConfig ?? {}), ...(options.voxelOptions ?? {}), ...(options.colliderOptions ?? {}) };
    const grid = collider && options.grid ? collider.filterGrid(options.grid, filterOptions) : options.grid ?? null;
    return { asset, sceneURL, metaURL, lodMetaURL: lodMetaURL ?? "", metadata, collider, grid, streaming: Boolean(lodMetaURL || /lod-meta\.json(?:$|[?#])/i.test(assetURLValue)) };
  } catch (error) {
    app?.assets?.remove?.(asset);
    asset.unload?.();
    throw error;
  }
}

/** Adds a unified GSplat component with optional manifest LOD bounds. */
export function createSplatEntity(pc, app, bundle, options = {}) {
  const entity = new pc.Entity(options.name ?? "df-splat-scene");
  const componentOptions = {
    asset: bundle.asset,
    layers: options.layers ?? [pc.LAYERID_WORLD],
    unified: true,
  };
  entity.addComponent("gsplat", componentOptions);
  if (entity.gsplat) {
    if (Number.isFinite(options.lodRangeMin)) entity.gsplat.lodRangeMin = options.lodRangeMin;
    if (Number.isFinite(options.lodRangeMax)) entity.gsplat.lodRangeMax = options.lodRangeMax;
  }
  if (bundle.streaming) entity.setLocalEulerAngles?.(0, 0, 180);
  app.root.addChild(entity);
  return entity;
}

/** Applies the battlefield transform to a scene or overlay entity. */
export function applyBattleTransform(entity, transform = {}) {
  const scale = Number(transform.scale ?? 1);
  const offsetY = Number(transform.offset_y ?? 0);
  const translate = Array.isArray(transform.translate) ? transform.translate : [0, 0, 0];
  entity.setLocalScale(scale, scale, scale);
  entity.setLocalPosition(Number(translate[0]), Number(translate[1]) + offsetY, Number(translate[2]));
  const current = entity.getLocalEulerAngles?.() ?? { z: 0 };
  entity.setLocalEulerAngles(0, Number(transform.rot_y_deg ?? 0), Number(current.z ?? 0));
  return entity;
}

/** Creates a grid layer after World and places the walkable grid on it. */
export function createBattleGrid(pc, app, grid, options = {}) {
  const world = app.scene.layers.getLayerById(pc.LAYERID_WORLD);
  const layers = app.scene.layers;
  let layer = world;
  let depthLayer = null;
  let depthProxy = null;
  if (pc.Layer && layers?.insert && Array.isArray(layers.layerList)) {
    const index = Math.max(0, layers.layerList.lastIndexOf(world));
    if (options.collider?.occupiedBoxes) {
      depthLayer = new pc.Layer({ name: options.depthLayerName ?? "df-voxel-depth", opaqueSortMode: pc.SORTMODE_NONE, transparentSortMode: pc.SORTMODE_NONE, clearDepthBuffer: false });
      layers.insert(depthLayer, index + 1);
      depthProxy = createVoxelDepthOccluder(pc, app, options.collider.occupiedBoxes(), grid, { layers: [depthLayer.id], maxBoxes: 65536, name: options.depthName ?? "df-voxel-depth-occluder" });
      app.on?.("destroy", () => depthProxy?.destroy());
    }
    layer = new pc.Layer({
      name: options.layerName ?? "df-battle-grid",
      opaqueSortMode: pc.SORTMODE_NONE,
      transparentSortMode: pc.SORTMODE_NONE,
      clearDepthBuffer: false,
    });
    layers.insert(layer, index + (depthLayer ? 2 : 1));
  }
  const entity = createGridOverlay(pc, app, grid, {
    ...options,
    name: options.name ?? "df-battle-grid",
    layers: [layer.id],
  });
  return { entity, layer, depthLayer, depthProxy };
}

/** Returns the profile encoded in ?battle=, or null for the legacy source mode. */
export function battleProfile(search, baseURL = window.location.href) {
  const encoded = new URLSearchParams(search).get("battle");
  if (!encoded) return null;
  try {
    const profile = JSON.parse(encoded);
    if (!profile || typeof profile !== "object" || Array.isArray(profile)) throw new Error("battle profile must be an object");
    const resolve = (url) => url ? new URL(url, baseURL).href : "";
    const voxel = profile.voxel_collider && typeof profile.voxel_collider === "object" ? { ...profile.voxel_collider, url: resolve(profile.voxel_collider.url) } : profile.voxel_collider;
    return {
      ...profile,
      scene_url: resolve(profile.scene_url),
      meta_url: resolve(profile.meta_url),
      lod_meta_url: resolve(profile.lod_meta_url),
      voxel_collider_url: resolve(profile.voxel_collider_url),
      voxel_collider: voxel,
    };
  } catch (error) {
    throw new Error(`battle profile JSON failed: ${error?.message ?? error}`);
  }
}

/** Loads a battle profile from an inline JSON value or a relative profile URL. */
export async function loadBattleProfile(search, baseURL = window.location.href) {
  const encoded = new URLSearchParams(search).get("battle");
  if (!encoded) return null;
  try {
    if (encoded.trim().startsWith("{")) return battleProfile(search, baseURL);
    const profileURL = new URL(encoded, baseURL).href;
    const response = await fetch(profileURL);
    if (!response.ok) throw new Error(`battle profile HTTP ${response.status}`);
    const profile = await response.json();
    if (!profile || typeof profile !== "object" || Array.isArray(profile)) {
      throw new Error("battle profile must be an object");
    }
    const resolve = (url) => url ? new URL(url, profileURL).href : "";
    const voxel = profile.voxel_collider && typeof profile.voxel_collider === "object" ? { ...profile.voxel_collider, url: resolve(profile.voxel_collider.url) } : profile.voxel_collider;
    return {
      ...profile,
      scene_url: resolve(profile.scene_url),
      meta_url: resolve(profile.meta_url),
      lod_meta_url: resolve(profile.lod_meta_url),
      voxel_collider_url: resolve(profile.voxel_collider_url),
      voxel_collider: voxel,
    };
  } catch (error) {
    throw new Error(`battle profile failed: ${error?.message ?? error}`);
  }
}
