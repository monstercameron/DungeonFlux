const DEFAULTS = Object.freeze({ enabled: false, center: 0.5, band: 0.35, falloff: 0.25, blur_px: 3 });

const VERTEX_SHADER = `
attribute vec2 aPosition;
varying vec2 vUv0;
void main(void) {
  gl_Position = vec4(aPosition, 0.0, 1.0);
  vUv0 = (aPosition + 1.0) * 0.5;
}`;

const BLUR_FRAGMENT_SHADER = `
precision highp float;
uniform sampler2D uSourceBuffer;
uniform vec2 uTexelSize;
uniform vec2 uDirection;
uniform float uBlurPx;
varying vec2 vUv0;
void main(void) {
  vec2 offset = uTexelSize * uDirection * (uBlurPx / 3.230769);
  vec4 blurred = texture2D(uSourceBuffer, vUv0) * 0.227027;
  blurred += texture2D(uSourceBuffer, vUv0 + 1.384615 * offset) * 0.316216;
  blurred += texture2D(uSourceBuffer, vUv0 - 1.384615 * offset) * 0.316216;
  blurred += texture2D(uSourceBuffer, vUv0 + 3.230769 * offset) * 0.070270;
  blurred += texture2D(uSourceBuffer, vUv0 - 3.230769 * offset) * 0.070270;
  gl_FragColor = vec4(blurred.rgb, 1.0);
}`;

const COMPOSE_FRAGMENT_SHADER = `
precision highp float;
uniform sampler2D uSourceBuffer;
uniform sampler2D uOriginalBuffer;
uniform vec2 uTexelSize;
uniform float uCenter;
uniform float uBand;
uniform float uFalloff;
uniform float uBlurPx;
varying vec2 vUv0;
void main(void) {
  vec4 source = texture2D(uOriginalBuffer, vUv0);
  float distanceFromCenter = abs(vUv0.y - uCenter);
  float edge = max(0.0, uBand * 0.5);
  float transition = max(uFalloff * 0.5, 0.0001);
  float blurMix = smoothstep(edge, edge + transition, distanceFromCenter);
  vec2 offset = uTexelSize * vec2(0.0, uBlurPx / 3.230769);
  vec4 blurred = texture2D(uSourceBuffer, vUv0) * 0.227027;
  blurred += texture2D(uSourceBuffer, vUv0 + 1.384615 * offset) * 0.316216;
  blurred += texture2D(uSourceBuffer, vUv0 - 1.384615 * offset) * 0.316216;
  blurred += texture2D(uSourceBuffer, vUv0 + 3.230769 * offset) * 0.070270;
  blurred += texture2D(uSourceBuffer, vUv0 - 3.230769 * offset) * 0.070270;
  gl_FragColor = vec4(mix(source.rgb, blurred.rgb, blurMix), 1.0);
}`;

function finite(value, fallback) {
  const result = Number(value);
  return Number.isFinite(result) ? result : fallback;
}

function clamp(value, min, max) {
  return Math.min(max, Math.max(min, value));
}

function normalize(input) {
  const value = input ?? {};
  return {
    enabled: Boolean(value.enabled),
    center: clamp(finite(value.center, DEFAULTS.center), 0, 1),
    band: clamp(finite(value.band, DEFAULTS.band), 0, 1),
    falloff: clamp(finite(value.falloff, DEFAULTS.falloff), 0, 1),
    blur_px: clamp(finite(value.blur_px, DEFAULTS.blur_px), 0, 16),
  };
}

function cameraComponent(camera) {
  return camera?.camera ?? camera;
}

class TiltShiftRenderer {
    constructor(pc, device, settings, effect) {
      this.pc = pc;
      this.device = device;
      this.effect = effect;
      this.settings = settings;
      this.blurShader = new pc.Shader(device, {
        attributes: { aPosition: pc.SEMANTIC_POSITION },
        vshader: VERTEX_SHADER,
        fshader: BLUR_FRAGMENT_SHADER,
      });
      this.composeShader = new pc.Shader(device, {
        attributes: { aPosition: pc.SEMANTIC_POSITION },
        vshader: VERTEX_SHADER,
        fshader: COMPOSE_FRAGMENT_SHADER,
      });
      this.intermediate = null;
      this.targetWidth = 0;
      this.targetHeight = 0;
    }

    render(input, output) {
      const { device } = this;
      const scope = device.scope;
      const texture = input?.colorBuffer;
      const width = texture?.width || device.width || 1;
      const height = texture?.height || device.height || 1;
      scope.resolve("uTexelSize").setValue([1 / width, 1 / height]);
      scope.resolve("uBlurPx").setValue(this.settings.blur_px);
      this.ensureTarget(width, height);
      scope.resolve("uSourceBuffer").setValue(texture);
      scope.resolve("uDirection").setValue([1, 0]);
      this.effect.drawQuad(this.intermediate, this.blurShader);
      scope.resolve("uSourceBuffer").setValue(this.intermediate.colorBuffer);
      scope.resolve("uOriginalBuffer").setValue(texture);
      scope.resolve("uCenter").setValue(this.settings.center);
      scope.resolve("uBand").setValue(this.settings.band);
      scope.resolve("uFalloff").setValue(this.settings.falloff);
      this.effect.drawQuad(output, this.composeShader);
    }

    ensureTarget(width, height) {
      const { pc, device } = this;
      if (this.intermediate && this.targetWidth === width && this.targetHeight === height) return;
      if (this.intermediate) this.destroyTarget();
      const colorBuffer = new pc.Texture(device, {
        width, height, format: pc.PIXELFORMAT_RGBA8, mipmaps: false,
        minFilter: pc.FILTER_LINEAR, magFilter: pc.FILTER_LINEAR,
        addressU: pc.ADDRESS_CLAMP_TO_EDGE, addressV: pc.ADDRESS_CLAMP_TO_EDGE,
      });
      this.intermediate = new pc.RenderTarget({ colorBuffer, depth: false, samples: 1 });
      this.targetWidth = width;
      this.targetHeight = height;
    }

    destroyTarget() {
      const target = this.intermediate;
      this.intermediate = null;
      if (!target) return;
      const colorBuffer = target.colorBuffer;
      target.destroy();
      colorBuffer?.destroy();
    }

    destroy() {
      this.destroyTarget();
      this.blurShader?.destroy();
      this.composeShader?.destroy();
      this.blurShader = null;
      this.composeShader = null;
    }
}

function makeTiltEffect(pc, device, settings) {
  const effect = new pc.PostEffect(device);
  const renderer = new TiltShiftRenderer(pc, device, settings, effect);
  effect.render = (input, output) => renderer.render(input, output);
  effect.destroy = () => renderer.destroy();
  return effect;
}

/** Installs an adjustable vertical tilt shift post effect on a camera. */
export function createTiltShift(pc, camera, options = {}) {
  if (!pc?.PostEffect || !pc?.Shader) throw new TypeError("PlayCanvas post effect API is required");
  const component = cameraComponent(camera);
  const queue = component?.postEffects;
  if (!queue?.addEffect || !queue?.removeEffect) throw new TypeError("Camera postEffects queue is required");
  const settings = normalize({ ...DEFAULTS, ...options });
  const device = pc.app?.graphicsDevice ?? component.device ?? component.system?.app?.graphicsDevice ?? camera?.app?.graphicsDevice;
  if (!device) throw new TypeError("Graphics device is required");
  const effect = makeTiltEffect(pc, device, settings);
  let attached = false;
  let destroyed = false;
  const attach = () => { if (!attached && settings.enabled) { queue.addEffect(effect); attached = true; } };
  const detach = () => { if (attached) { queue.removeEffect(effect); attached = false; } };
  const api = {
    configure(next = {}) {
      if (destroyed) return api;
      Object.assign(settings, normalize({ ...settings, ...next }));
      effect.settings = settings;
      if (settings.enabled) attach(); else detach();
      return api;
    },
    destroy() {
      if (destroyed) return;
      detach();
      effect.destroy();
      destroyed = true;
    },
    get settings() { return { ...settings }; },
  };
  attach();
  return api;
}

export const TILT_SHIFT_SHADER = Object.freeze({ vertex: VERTEX_SHADER, fragment: `${BLUR_FRAGMENT_SHADER}\n${COMPOSE_FRAGMENT_SHADER}` });
