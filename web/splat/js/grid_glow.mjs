const DEFAULT_CORE_WIDTH = 0.055;
const DEFAULT_GLOW_WIDTH = 0.26;
const DEFAULT_CORE_COLOR = [1.0, 0.9, 0.55];
const DEFAULT_HALO_COLOR = [1.0, 0.6, 0.15];

const VERTEX_GLSL = `
attribute vec3 aPosition;
attribute vec2 aUv0;
uniform mat4 matrix_model;
uniform mat4 matrix_viewProjection;
varying vec2 vEdge;
void main(void) {
  vEdge = aUv0;
  gl_Position = matrix_viewProjection * matrix_model * vec4(aPosition, 1.0);
}`;

const FRAGMENT_GLSL = `
// Match the high precision used by PlayCanvas' injected view uniform block.
// A mediump fragment declaration makes the shared ub_view block incompatible
// with the vertex stage on WebGL2, so the program is rejected at link time.
precision highp float;
uniform vec3 uCoreColor;
uniform vec3 uHaloColor;
uniform float uCoreWidth;
uniform float uGlowWidth;
uniform float uOpacity;
uniform float uHaloStrength;
varying vec2 vEdge;
void main(void) {
  float edge = abs(vEdge.x);
  float aa = max(fwidth(edge), 0.0015);
  float radius = clamp(max(uCoreWidth / uGlowWidth, 0.65 * aa), 0.01, 0.9);
  float core = 1.0 - smoothstep(radius - aa, radius + aa, edge);
  float haloDistance = max(0.0, (edge - radius) / (1.0 - radius));
  float halo = exp(-4.5 * haloDistance * haloDistance) * uHaloStrength;
  float strokeWidth = 0.12;
  float stroke = (1.0 - core) * (1.0 - smoothstep(radius + strokeWidth - aa, radius + strokeWidth + aa, edge));
  float alpha = clamp(max(max(core, halo), stroke * 0.9) * uOpacity, 0.0, 1.0);
  vec3 color = mix(uHaloColor, vec3(0.16, 0.07, 0.015), stroke);
  color = mix(color, uCoreColor, core);
  gl_FragColor = vec4(color, alpha);
}`;

function vector(value, fallback) {
  return Array.isArray(value) && value.length >= 3 ? value : fallback;
}

/** Creates the amber terrain grid shader with an antialiased core and halo. */
export function createGridGlowMaterial(pc, options = {}) {
  if (!pc?.ShaderMaterial) throw new TypeError("PlayCanvas ShaderMaterial is required");
  const coreWidth = Number(options.coreWidth ?? DEFAULT_CORE_WIDTH);
  const glowWidth = Math.max(coreWidth, Number(options.glowWidth ?? DEFAULT_GLOW_WIDTH));
  const material = new pc.ShaderMaterial({
    // ShaderMaterial keys its compiled variants with uniqueName.  `name` is
    // the inherited display label and is ignored by the shader descriptor.
    uniqueName: "df-grid-amber-glow",
    vertexGLSL: VERTEX_GLSL,
    fragmentGLSL: FRAGMENT_GLSL,
    attributes: { aPosition: pc.SEMANTIC_POSITION, aUv0: pc.SEMANTIC_TEXCOORD0 },
  });
  material.name = "df-grid-amber-glow";
  material.setParameter("uCoreColor", vector(options.coreColor, DEFAULT_CORE_COLOR));
  material.setParameter("uHaloColor", vector(options.haloColor, DEFAULT_HALO_COLOR));
  material.setParameter("uCoreWidth", coreWidth);
  material.setParameter("uGlowWidth", glowWidth);
  material.setParameter("uOpacity", Number(options.opacity ?? 0.84));
  material.setParameter("uHaloStrength", Number(options.haloStrength ?? 0.6));
  material.blendType = pc.BLEND_NORMAL;
  material.depthTest = true;
  material.depthWrite = false;
  material.cull = pc.CULLFACE_NONE;
  material.update();
  return material;
}

export const GRID_GLOW_SHADER = Object.freeze({ vertex: VERTEX_GLSL, fragment: FRAGMENT_GLSL });
