/** Owns the optional fullscreen presentation styles and element resize hooks. */
export function createCanvasSurface({ canvas, layout = "embedded", onResize = () => {} } = {}) {
  if (!canvas || typeof canvas.addEventListener !== "function") throw new TypeError("canvas is required");
  const fullscreen = layout === "fullscreen";
  const original = snapshotStyles(canvas, fullscreen);
  let disposed = false;
  let resizeHandler = onResize;
  const measure = () => {
    if (disposed) return { width: 0, height: 0, pixelWidth: 0, pixelHeight: 0 };
    const rect = canvas.getBoundingClientRect?.() ?? { width: canvas.clientWidth, height: canvas.clientHeight };
    const width = Math.max(0, Number(rect.width || canvas.clientWidth || 0));
    const height = Math.max(0, Number(rect.height || canvas.clientHeight || 0));
    const dpr = Math.max(1, Math.min(2, Number(globalThis.devicePixelRatio || 1)));
    const size = { width, height, pixelWidth: Math.max(0, Math.round(width * dpr)), pixelHeight: Math.max(0, Math.round(height * dpr)) };
    resizeHandler(size);
    return size;
  };
  if (fullscreen) applyFullscreenStyles(canvas);
  const onWindowResize = () => measure();
  let observer = null;
  if (typeof ResizeObserver === "function") {
    observer = new ResizeObserver(measure);
    observer.observe(canvas);
  } else globalThis.addEventListener?.("resize", onWindowResize);
  measure();
  return Object.freeze({
    canvas,
    setResizeHandler(handler) { resizeHandler = typeof handler === "function" ? handler : () => {}; },
    resize: measure,
    dispose() {
      if (disposed) return;
      disposed = true;
      observer?.disconnect();
      globalThis.removeEventListener?.("resize", onWindowResize);
      if (original) restoreStyles(canvas, original);
    },
  });
}

function snapshotStyles(canvas, fullscreen) {
  const keys = fullscreen ? ["display", "position", "inset", "width", "height", "zIndex", "pointerEvents", "opacity", "transition"] : ["opacity", "transition"];
  return Object.fromEntries(keys.map(key => [key, canvas.style?.[key] ?? ""]));
}

function applyFullscreenStyles(canvas) {
  Object.assign(canvas.style, { display: "block", position: "fixed", inset: "0", width: "100%", height: "100%", zIndex: "0", pointerEvents: "none", opacity: "0" });
}

function restoreStyles(canvas, original) {
  for (const [key, value] of Object.entries(original)) canvas.style[key] = value;
}
