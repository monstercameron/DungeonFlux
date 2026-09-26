const GRAY = Math.round(0.47 * 255);

function sceneBounds(collider) {
  const bounds = collider?.sceneBoundsInWorld?.();
  if (!Array.isArray(bounds?.min) || !Array.isArray(bounds?.max)) return null;
  const min = bounds.min.slice(0, 3).map(Number);
  const max = bounds.max.slice(0, 3).map(Number);
  if (min.length !== 3 || max.length !== 3 || !min.every(Number.isFinite) || !max.every(Number.isFinite)) return null;
  return min.every((value, axis) => value < max[axis]) ? { min, max } : null;
}

function capturedSkyGLSL(bounds, padding, floorY) {
  const ceiling = (bounds.max[1] + padding).toFixed(6);
  return `
void modifySplatCenter(inout vec3 center) {
}
void modifySplatRotationScale(vec3 originalCenter, vec3 modifiedCenter, inout vec4 rotation, inout vec3 scale) {
}
void modifySplatColor(vec3 center, inout vec4 color) {
    bool distant = center.x < ${(bounds.min[0] - padding).toFixed(6)} || center.x > ${(bounds.max[0] + padding).toFixed(6)}
        || center.z < ${(bounds.min[2] - padding).toFixed(6)} || center.z > ${(bounds.max[2] + padding).toFixed(6)};
    // Nearby sky fragments can survive the spatial envelope after reconstruction.
    // Only blue-dominant fragments well above gameplay height qualify.
    bool capturedBlue = center.y > ${(floorY + (bounds.max[1] - floorY) * 0.6).toFixed(6)}
        && color.b > color.r * 1.2 && color.b > color.g * 1.08;
    if (center.y > ${ceiling} || (distant && center.y > ${floorY.toFixed(6)}) || capturedBlue) color.a = 0.0;
}
`;
}

/** Excludes elevated outliers beyond the physical scene while preserving terrain below the gameplay floor. */
export function installCapturedSkyExclusion(entity, collider, options = {}) {
  const bounds = sceneBounds(collider);
  if (!entity?.gsplat?.setWorkBufferModifier || !bounds) return null;
  const padding = Number(options.padding ?? 1);
  if (!Number.isFinite(padding) || padding < 0) throw new Error("captured sky padding must be finite and nonnegative");
  const floorY = Number(options.floorY ?? 0);
  if (!Number.isFinite(floorY)) throw new Error("captured sky floor must be finite");
  const modifier = { glsl: capturedSkyGLSL(bounds, padding, floorY) };
  entity.gsplat.setWorkBufferModifier(modifier);
  return Object.freeze({ bounds, padding, floorY, modifier });
}

function createTexture(pc, app) {
  if (!pc?.Texture || !app?.graphicsDevice) throw new TypeError("PlayCanvas graphics device is required");
  const texture = new pc.Texture(app.graphicsDevice, {
    width: 1,
    height: 1,
    cubemap: true,
    mipmaps: false,
    format: pc.PIXELFORMAT_RGBA8,
    levels: [Array.from({ length: 6 }, () => new Uint8Array([GRAY, GRAY, GRAY, 255]))],
  });
  try { texture.upload?.(); return texture; }
  catch (error) { texture.destroy?.(); throw error; }
}

/** Installs and owns a neutral gray six-face cubemap skybox. */
export function installGraySkybox(pc, app, camera) {
  if (!app?.scene || !("skybox" in app.scene)) throw new TypeError("PlayCanvas scene skybox API is required");
  const texture = createTexture(pc, app);
  const scene = app.scene;
  const previous = { skybox: scene.skybox, intensity: scene.skyboxIntensity };
  scene.skybox = texture;
  scene.skyboxIntensity = 1;
  let addedLayer = false;
  if (camera?.camera && pc.LAYERID_SKYBOX !== undefined) {
    const layers = camera.camera.layers ?? [];
    if (!layers.includes(pc.LAYERID_SKYBOX)) { camera.camera.layers = [...layers, pc.LAYERID_SKYBOX]; addedLayer = true; }
  }
  let destroyed = false;
  return Object.freeze({ texture, destroy() {
    if (destroyed) return;
    destroyed = true;
    if (scene.skybox === texture) {
      scene.skybox = previous.skybox ?? null;
      if (previous.intensity !== undefined) scene.skyboxIntensity = previous.intensity;
    }
    if (addedLayer && camera?.camera) camera.camera.layers = (camera.camera.layers ?? []).filter((id) => id !== pc.LAYERID_SKYBOX);
    texture.destroy?.();
  } });
}
