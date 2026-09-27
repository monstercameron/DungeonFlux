// Loaded only by the local development supervisor after a successful build.
let version = "";
let pending = false;
async function checkBuild() {
  if (pending) return;
  pending = true;
  try {
    const response = await fetch("/__dev/version", { cache: "no-store" });
    if (!response.ok) return;
    const next = (await response.text()).trim();
    if (!next) return;
    if (version && next !== version) window.location.reload();
    version = next;
  } catch { /* Keep the last working page while the server rebuilds. */ }
  finally { pending = false; }
}
checkBuild();
const timer = setInterval(checkBuild, 2000);
window.addEventListener("pagehide", () => clearInterval(timer), { once: true });
