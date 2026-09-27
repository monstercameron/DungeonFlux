// Loaded only by the local development supervisor after a successful build.
let version = "";
let pending = false;
const recovering = document.documentElement.hasAttribute("data-dev-recover");
async function checkBuild() {
  if (pending) return;
  pending = true;
  try {
    const response = await fetch("/__dev/version", { cache: "no-store", signal: AbortSignal.timeout(3000) });
    if (!response.ok) return;
    const next = (await response.text()).trim();
    if (!next) return;
    if (recovering || (version && next !== version)) window.location.reload();
    version = next;
  } catch { /* Keep the last working page while the server rebuilds. */ }
  finally { pending = false; }
}
checkBuild();
let timer = setInterval(checkBuild, 2000);
window.addEventListener("pagehide", () => { clearInterval(timer); timer = null; });
window.addEventListener("pageshow", () => {
  if (timer === null) timer = setInterval(checkBuild, 2000);
  checkBuild();
});
document.addEventListener("visibilitychange", () => {
  if (document.visibilityState === "visible") checkBuild();
});
