import * as pc from "../vendor/playcanvas.mjs";

const WORLD_LAYER = pc.LAYERID_WORLD;
const CHROMA = [0.0, 0.6941176, 0.2509804];
const VERTEX_SHADER = `
attribute vec3 aPosition;
attribute vec2 aUv0;
uniform mat4 matrix_model;
uniform mat4 matrix_viewProjection;
varying vec2 vUv0;
void main(void) {
  vUv0 = aUv0;
  gl_Position = matrix_viewProjection * matrix_model * vec4(aPosition, 1.0);
}`;
// Keys on green dominance (g - max(r, b)) rather than one exact colour: the
// video model paints its "#00B140" background anywhere from (0,177,64) to a
// lighter (113,204,103), with darker green where the floor shadow falls.
const FRAGMENT_SHADER = `
uniform sampler2D uVideo;
varying vec2 vUv0;
void main(void) {
  vec4 color = texture2D(uVideo, vUv0);
  float green = color.g - max(color.r, color.b);
  float keep = 1.0 - smoothstep(0.10, 0.20, green);
  if (keep < 0.5) discard;
  color.g = min(color.g, max(color.r, color.b) + 0.03);
  gl_FragColor = vec4(color.rgb, 1.0);
}`;

function number(value, fallback) {
  return Number.isFinite(Number(value)) ? Number(value) : fallback;
}

function vector(value, fallback) {
  return Array.isArray(value) && value.length >= 3 ? value : fallback;
}

function setPosition(entity, position) {
  const point = vector(position, [0, 0, 0]);
  entity.setPosition(number(point[0], 0), number(point[1], 0), number(point[2], 0));
}

function createVideo(url) {
  const video = document.createElement("video");
  video.muted = true;
  video.loop = true;
  video.playsInline = true;
  video.preload = "auto";
  video.crossOrigin = "anonymous";
  video.src = String(url);
  return video;
}

function createTexture(app, video) {
  const texture = new pc.Texture(app.graphicsDevice, {
    format: pc.PIXELFORMAT_RGBA8,
    mipmaps: false,
    minFilter: pc.FILTER_LINEAR,
    magFilter: pc.FILTER_LINEAR,
    addressU: pc.ADDRESS_CLAMP_TO_EDGE,
    addressV: pc.ADDRESS_CLAMP_TO_EDGE,
  });
  texture.setSource(video);
  return texture;
}

/** createChromaMaterial returns the depth-writing, green-keyed material for a video texture. */
export function createChromaMaterial(app, texture) {
  return createMaterial(app, texture);
}

// The keyed video drew nothing on the TV: a video texture uploads with alpha 0
// on this path, so the old alpha discard (and alphaTest) threw every fragment
// away; the shader also needs a uniqueName and no precision line under
// PlayCanvas 2.22's GLSL ES 3 wrapper. Keying is on green dominance alone.
function createMaterial(app, texture) {
  const material = new pc.ShaderMaterial({
    uniqueName: "df-chroma-billboard",
    name: "df-chroma-billboard",
    vertexGLSL: VERTEX_SHADER,
    fragmentGLSL: FRAGMENT_SHADER,
    attributes: {
      aPosition: pc.SEMANTIC_POSITION,
      aUv0: pc.SEMANTIC_TEXCOORD0,
    },
    uniforms: { uVideo: "texture_2d" },
  });
  material.setParameter("uVideo", texture);
  material.blendType = pc.BLEND_NONE;
  material.depthTest = true;
  material.depthWrite = true;
  material.cull = pc.CULLFACE_NONE;
  material.update();
  return material;
}

function createMeshInstance(app, material, width, height) {
  const mesh = pc.createPlane(app.graphicsDevice, {
    width,
    height,
    widthSegments: 1,
    lengthSegments: 1,
  });
  const instance = new pc.MeshInstance(mesh, material);
  instance.layer = WORLD_LAYER;
  instance.castShadow = false;
  instance.receiveShadow = false;
  return instance;
}

function faceCamera(entity, camera) {
  if (!camera) return;
  const position = entity.getPosition();
  const cameraPosition = camera.getPosition();
  const dx = cameraPosition.x - position.x;
  const dz = cameraPosition.z - position.z;
  if (Math.abs(dx) + Math.abs(dz) < 0.0001) return;
  entity.setEulerAngles(90, Math.atan2(dx, dz) * 180 / Math.PI, 0);
}

function configureEntity(entity, definition) {
  const scale = number(definition.scale, 1);
  const height = number(definition.height_m, 1.8) * scale;
  const width = number(definition.width_m, height * 9 / 16) * scale;
  entity.setLocalScale(1, 1, 1);
  entity.setLocalPosition(0, height / 2, 0);
  entity.setLocalScale(width, 1, height);
}

function updateVideoTexture(texture, video) {
  if (video.readyState >= 2) texture.upload();
}

class Billboard {
  constructor(controller, definition) {
    this.controller = controller;
    this.definition = { ...definition };
    this.video = createVideo(definition.url);
    this.texture = createTexture(controller.app, this.video);
    this.material = createMaterial(controller.app, this.texture);
    this.entity = new pc.Entity(`df-billboard-${definition.id}`);
    this.entity.addComponent("render", {
      meshInstances: [createMeshInstance(controller.app, this.material, 1, 1)],
      layers: [WORLD_LAYER],
    });
    configureEntity(this.entity, definition);
    setPosition(this.entity, definition.position);
    controller.app.root.addChild(this.entity);
    this.video.addEventListener("loadeddata", () => updateVideoTexture(this.texture, this.video));
  }

  async play() {
    try {
      await this.video.play();
    } catch (error) {
      this.controller.onError?.(this.definition.id, error);
    }
  }

  updateDefinition(definition) {
    this.definition = { ...this.definition, ...definition };
    configureEntity(this.entity, this.definition);
    setPosition(this.entity, this.definition.position);
  }

  pause() {
    this.video.pause();
  }

  update(camera) {
    faceCamera(this.entity, camera);
    updateVideoTexture(this.texture, this.video);
  }

  destroy() {
    this.video.pause();
    this.video.removeAttribute("src");
    this.video.load();
    this.entity.destroy();
    this.texture.destroy();
    this.material.destroy();
  }
}

/** Manages chroma-keyed, depth-writing billboard videos in the splat world. */
export class BillboardController {
  constructor(app, options = {}) {
    this.app = app;
    this.camera = options.camera ?? null;
    this.onError = options.onError;
    this.billboards = new Map();
    this.updateHandler = () => this.update();
    this.app.on("update", this.updateHandler);
  }

  /** Adds or replaces one billboard and starts its muted loop. */
  async add(definition) {
    if (!definition?.id || !definition?.url) throw new TypeError("billboard id and url are required");
    this.remove(definition.id);
    const billboard = new Billboard(this, definition);
    this.billboards.set(String(definition.id), billboard);
    await billboard.play();
    return billboard.entity;
  }

  /** Applies a full token snapshot, retaining only the listed billboards. */
  async setTokens(tokens = []) {
    const wanted = new Set(tokens.map((token) => String(token.id)));
    for (const id of this.billboards.keys()) {
      if (!wanted.has(id)) this.remove(id);
    }
    await Promise.all(tokens.map(async (token) => {
      const definition = {
        ...token,
        url: token.url ?? token.clips?.[token.anim ?? "idle"] ?? token.portrait,
        position: token.position ?? [0, 0, 0],
      };
      const current = this.billboards.get(String(token.id));
      if (current && current.definition.url === definition.url) {
        current.updateDefinition(definition);
        return;
      }
      await this.add(definition);
    }));
  }

  /** Pauses or resumes every billboard without changing its scene placement. */
  setPaused(paused) {
    for (const billboard of this.billboards.values()) {
      if (paused) billboard.pause();
      else void billboard.play();
    }
  }

  /** Removes one billboard and releases its video and GPU resources. */
  remove(id) {
    const billboard = this.billboards.get(String(id));
    if (!billboard) return;
    billboard.destroy();
    this.billboards.delete(String(id));
  }

  /** Updates camera-facing transforms and uploads current video frames. */
  update() {
    for (const billboard of this.billboards.values()) billboard.update(this.camera);
  }

  /** Removes every billboard and unregisters the update callback. */
  destroy() {
    for (const id of this.billboards.keys()) this.remove(id);
    this.app.off("update", this.updateHandler);
  }
}

export const BILLBOARD_CHROMA = Object.freeze([...CHROMA]);
