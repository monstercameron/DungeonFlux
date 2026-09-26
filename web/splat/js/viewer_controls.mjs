/** Install visibility and input toggles for a viewer control panel. */
export function installViewerControls({ panel, runtime, container = panel?.parentElement, visible = true, cameraControls = true, ownsPanel = true } = {}) {
  if (!panel) return Object.freeze({ setVisible() {}, setCameraControlsEnabled() {}, dispose() {} });
  const host = container ?? panel.parentElement;
  let panelVisible = Boolean(visible);
  let inputEnabled = Boolean(cameraControls);
  const show = document.createElement("button");
  show.type = "button";
  show.textContent = "Show controls";
  show.setAttribute("aria-label", "Show viewer controls");
  show.hidden = panelVisible;
  show.className = "df-show-controls";
  const onShow = () => setVisible(true);
  show.addEventListener("click", onShow);
  host?.append(show);
  const sync = () => {
    panel.hidden = !panelVisible;
    show.hidden = panelVisible;
    panel.setAttribute("aria-hidden", String(!panelVisible));
    panel.dataset.cameraControls = String(inputEnabled);
  };
  const setVisible = (next) => { panelVisible = Boolean(next); sync(); };
  const setCameraControlsEnabled = (next) => {
    inputEnabled = Boolean(next);
    runtime?.setCameraControlsEnabled?.(inputEnabled);
    sync();
  };
  sync();
  return Object.freeze({
    setVisible,
    setCameraControlsEnabled,
    dispose() { show.removeEventListener("click", onShow); show.remove(); if (ownsPanel) panel.remove(); },
  });
}


/** installStandaloneControls adds independent panel and camera input toggles to the preview. */
export function installStandaloneControls({ panel, container, onCameraControls, search = globalThis.location?.search ?? "" } = {}) {
  if (!panel) return { dispose() {}, setControlsVisible() {}, setCameraControlsEnabled() {} };
  const params = new URLSearchParams(search);
  const input = document.createElement("button"); input.type = "button"; input.textContent = "Camera input";
  const hide = document.createElement("button"); hide.type = "button"; hide.textContent = "Hide controls";
  const inputEnabled = params.get("cameraControls") !== "off";
  const runtime = { setCameraControlsEnabled(on) { onCameraControls?.(on); input.setAttribute("aria-pressed", String(on)); } };
  const controls = installViewerControls({ panel, container, runtime, visible: params.get("controls") !== "off", cameraControls: inputEnabled, ownsPanel: false });
  input.setAttribute("aria-pressed", String(inputEnabled)); runtime.setCameraControlsEnabled(inputEnabled);
  const onInput = () => controls.setCameraControlsEnabled(input.getAttribute("aria-pressed") !== "true");
  const onHide = () => controls.setVisible(false);
  input.addEventListener("click", onInput); hide.addEventListener("click", onHide); panel.append(input, hide);
  return Object.freeze({ setControlsVisible: controls.setVisible, setCameraControlsEnabled: controls.setCameraControlsEnabled,
    dispose() { input.removeEventListener("click", onInput); hide.removeEventListener("click", onHide); input.remove(); hide.remove(); controls.dispose(); } });
}
