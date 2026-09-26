import { installViewerControls } from "./viewer_controls.mjs";
import { THEMES } from "./theme_grades.mjs";

const PRESETS = ["COMBAT_EST", "TACTICAL", "TURN_FOCUS", "IMPACT", "KO", "VICTORY", "SOURCE", "SURVEY"];
const CSS = ".df-battle-controls[hidden],.df-show-controls[hidden]{display:none!important}.df-show-controls{pointer-events:auto;border:1px solid #526981;border-radius:5px;padding:6px 9px;background:#162538;color:#eaf2ff;cursor:pointer}.df-battle-controls{pointer-events:auto;max-width:100%;padding:8px;border:1px solid #334358;border-radius:8px;background:rgb(8 14 24 / 90%)}.df-battle-controls{display:flex;flex-wrap:wrap;gap:6px;align-items:center;font:14px/1.35 system-ui,sans-serif;color:#eaf2ff}.df-battle-controls .df-control-row{display:flex;flex-wrap:wrap;gap:4px;align-items:center}.df-battle-controls .df-control-label{color:#8da5bc;font-size:12px}.df-battle-controls button,.df-battle-controls select,.df-battle-controls input{border:1px solid #526981;border-radius:5px;padding:5px 7px;color:#eaf2ff;background:#162538;font:inherit}.df-battle-controls button:hover,.df-battle-controls button[aria-pressed=true]{background:#245070;border-color:#72d7e6}.df-battle-controls button:disabled{opacity:.55}.df-battle-controls [role=status]{color:#ffaaa8;flex-basis:100%}";

function button(label, action, pressed = false) { const node = document.createElement("button"); node.type = "button"; node.textContent = label; node.dataset.action = action; if (pressed) node.setAttribute("aria-pressed", "true"); return node; }
function select(label, values) { const node = document.createElement("select"); node.dataset.action = label; node.setAttribute("aria-label", label); values.forEach(([value, text]) => node.append(new Option(text, value))); return node; }
function addRow(panel, label, nodes) { const row = document.createElement("div"); row.className = "df-control-row"; if (label) { const text = document.createElement("span"); text.className = "df-control-label"; text.textContent = label; row.append(text); } nodes.forEach((node) => row.append(node)); panel.append(row); return row; }
function addStyle(panel) { const style = document.createElement("style"); style.textContent = CSS; panel.append(style); return style; }
function setPressed(node, value) { node.setAttribute("aria-pressed", String(Boolean(value))); }

function buildPanel({ onMove, cameraControls }) {
  const panel = document.createElement("section"); panel.className = "df-battle-controls"; panel.setAttribute("aria-label", "Battle viewer controls");
  const cameras = PRESETS.map((name) => button(name.replaceAll("_", " "), `camera:${name}`)); addRow(panel, "Camera", cameras);
  const grid = button("Grid", "grid", true); const characters = button("Characters", "characters", true); const pause = button("Pause", "pause"); const move = typeof onMove === "function" ? button("Move demo token", "move") : null; addRow(panel, "Scene", [grid, characters, pause, ...(move ? [move] : [])]);
  const motion = button("Motion", "motion", true); const tilt = button("Tilt", "tilt"); const shake = button("Shake", "shake"); const pan = button("Pan", "pan"); const stop = button("Stop", "stop"); addRow(panel, "Effects", [motion, tilt, shake, pan, stop]);
  const follow = select("follow", [["", "Follow off"]]); addRow(panel, "Follow", [follow]);
  const theme = select("theme", Object.entries(THEMES).map(([value, grade]) => [value, grade.label])); const strength = document.createElement("input"); strength.type = "range"; strength.min = "0"; strength.max = "1"; strength.step = "0.05"; strength.setAttribute("aria-label", "Color grade strength"); const lod = select("lod", [["auto", "Auto"]]); addRow(panel, "Display", [theme, strength, lod]);
  const cameraToggle = button("Camera input", "camera-controls", cameraControls); addRow(panel, "Input", [cameraToggle]); const hide = button("Hide controls", "hide-controls"); panel.append(hide);
  const status = document.createElement("div"); status.setAttribute("role", "status"); status.setAttribute("aria-live", "polite"); status.hidden = true; panel.append(status); addStyle(panel);
  return { panel, cameras, grid, characters, pause, motion, tilt, shake, pan, stop, move, follow, theme, strength, lod, cameraToggle, status };
}

function syncPanel(ui, state = {}) {
  setPressed(ui.grid, state.gridVisible !== false); setPressed(ui.characters, state.charactersVisible !== false); setPressed(ui.pause, state.paused === true); setPressed(ui.motion, state.effectsEnabled !== false && !state.reducedMotion); setPressed(ui.tilt, Boolean(state.tiltShift?.enabled)); setPressed(ui.cameraToggle, state.cameraControls !== false);
  const available = Object.keys(state.cameras ?? {}); const names = [...new Set([...PRESETS, ...available])];
  for (const name of names) if (!ui.cameras.some((node) => node.dataset.action === `camera:${name}`)) { const node = button(name.replaceAll("_", " "), `camera:${name}`); ui.panel.querySelector(".df-control-row")?.append(node); ui.cameras.push(node); }
  ui.cameras.forEach((node) => { const name = node.dataset.action.slice(7); node.disabled = available.length > 0 && !available.includes(name); setPressed(node, name === state.cameraPreset); });
  const tokens = state.tokens?.tokens ?? []; const labels = tokens.map((token) => [String(token.id), String(token.name ?? token.id)]); const known = [...ui.follow.options].slice(1).map((option) => [option.value, option.textContent]);
  if (JSON.stringify(labels) !== JSON.stringify(known)) { ui.follow.replaceChildren(new Option("Follow off", "")); tokens.forEach((token) => ui.follow.append(new Option(token.name ?? token.id, token.id))); }
  ui.follow.value = state.tokens?.followId ?? "";
  if (state.colorGrade?.theme) ui.theme.value = state.colorGrade.theme; if (state.colorGrade?.strength !== undefined) ui.strength.value = String(state.colorGrade.strength);
  const count = Number(state.lodLevelCount ?? 0); if (count > 0 && ui.lod.options.length !== count + 1) { ui.lod.replaceChildren(new Option("Auto", "auto")); for (let index = 0; index < count; index += 1) ui.lod.append(new Option(`LOD ${index}`, String(index))); }
  if (state.lod !== undefined) ui.lod.value = String(state.lod);
  ui.pan.disabled = !available.includes("SURVEY");
  const effectsOff = state.effectsEnabled === false || state.reducedMotion === true; ui.shake.disabled = effectsOff; ui.pan.disabled ||= effectsOff; ui.tilt.disabled = state.effectsEnabled === false;
}

function actionHandler(ui, controls, runtime, onMove, state) {
  const call = (fn, ...args) => { try { const result = runtime?.[fn]?.(...args); result?.catch?.((error) => { ui.status.hidden = false; ui.status.textContent = error?.message ?? String(error); }); return result; } catch (error) { ui.status.hidden = false; ui.status.textContent = error?.message ?? String(error); return undefined; } };
  return (event) => { const action = event.target?.dataset?.action; if (!action) return;
    if (action.startsWith("camera:")) call("setCamera", action.slice(7)); else if (action === "grid") call("setGridVisible", ui.grid.getAttribute("aria-pressed") !== "true"); else if (action === "characters") call("setCharactersVisible", ui.characters.getAttribute("aria-pressed") !== "true"); else if (action === "pause") call("pause", ui.pause.getAttribute("aria-pressed") !== "true"); else if (action === "move") onMove?.(state.value); else if (action === "camera-controls") controls.setCameraControlsEnabled(ui.cameraToggle.getAttribute("aria-pressed") !== "true"); else if (action === "motion") call("effects", { enabled: true, reduced_motion: ui.motion.getAttribute("aria-pressed") === "true" }); else if (action === "hide-controls") controls.setVisible(false); else if (action === "tilt") call("effects", { enabled: true, tilt_shift: { enabled: ui.tilt.getAttribute("aria-pressed") !== "true", center: 0.5, band: 0.3, falloff: 0.35, blur_px: 2 } }); else if (action === "shake") call("effects", { enabled: true, shake: { amplitude_px: 8, duration_ms: 200 } }); else if (action === "pan") call("effects", { enabled: true, pan: { preset: "SURVEY", duration_ms: 2500 } }); else if (action === "stop") call("effects", { enabled: true, stop: true });
  };
}

/** Build accessible controls with isolated state and cleanup for one battle runtime. */
export function createBattleControls({ runtime, container, visible = true, cameraControls = true, onMove, className = "df-battle-controls" } = {}) {
  const ui = buildPanel({ onMove, cameraControls }); ui.panel.className = `df-battle-controls ${className}`; (container ?? document.body).append(ui.panel); const state = { value: runtime?.getState?.() ?? {} }; const controls = installViewerControls({ panel: ui.panel, runtime, container: container ?? ui.panel.parentElement, visible, cameraControls });
  const onClick = actionHandler(ui, controls, runtime, onMove, state); ui.panel.addEventListener("click", onClick); const update = (next = {}) => { state.value = next; syncPanel(ui, next); }; const unsubscribe = runtime?.onEvent?.((event) => { if (event.type === "error") { ui.status.hidden = false; ui.status.textContent = event.detail ?? "Viewer error"; } update(event.state ?? event); }); update(state.value);
  const handlers = [];
  const listen = (node, name, callback) => { node.addEventListener(name, callback); handlers.push(() => node.removeEventListener(name, callback)); };
  listen(ui.follow, "change", () => runtime.follow(ui.follow.value || null));
  const grade = () => runtime.setColorGrade({ theme: ui.theme.value, strength: Number(ui.strength.value), enabled: ui.theme.value !== "neutral" });
  listen(ui.theme, "change", grade); listen(ui.strength, "input", grade);
  listen(ui.lod, "change", () => runtime.setLOD(ui.lod.value === "auto" ? "auto" : Number(ui.lod.value)));
  return Object.freeze({ panel: ui.panel, ...controls, update, dispose() { unsubscribe?.(); handlers.forEach(cleanup => cleanup()); ui.panel.removeEventListener("click", onClick); controls.dispose(); } });
}
