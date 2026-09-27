function positionOf(camera) {
  const value = camera.getPosition?.() ?? { x: 0, y: 7, z: 10 };
  return [Number(value.x), Number(value.y), Number(value.z)];
}

function distanceFrom(target, position) {
  return Math.max(0.01, Math.hypot(position[0] - target[0], position[1] - target[1], position[2] - target[2]));
}

/** Computes an orbit position while preserving radius around the target. */
export function orbitPosition(target, position, yawDelta, pitchDelta) {
  const dx = position[0] - target[0];
  const dy = position[1] - target[1];
  const dz = position[2] - target[2];
  const radius = distanceFrom(target, position);
  const yaw = Math.atan2(dx, dz) + yawDelta;
  const pitch = Math.max(0.12, Math.min(1.35, Math.asin(dy / radius) + pitchDelta));
  const horizontal = radius * Math.cos(pitch);
  return [target[0] + Math.sin(yaw) * horizontal, target[1] + Math.sin(pitch) * radius, target[2] + Math.cos(yaw) * horizontal];
}

/** Installs bounded drag orbit and wheel zoom controls and returns preset hooks. */
export function installOrbitControls({ canvas, camera, target = [0, 0, 0] }) {
  let orbitTarget = [...target];
  let position = positionOf(camera);
  let radius = distanceFrom(orbitTarget, position);
  let minimum = Math.max(0.25, radius * 0.35);
  let maximum = Math.max(minimum, radius * 3);
  let dragging = false;
  let moved = false;
  let activePointerId = null;
  let lastX = 0;
  let lastY = 0;
  let startX = 0;
  let startY = 0;
  const apply = (next) => { position = next; camera.setPosition(...position); camera.lookAt(...orbitTarget); };
  const releaseCapture = id => { if (!canvas.hasPointerCapture || canvas.hasPointerCapture(id)) canvas.releasePointerCapture?.(id); };
  const onDown = (event) => { if (dragging) return; dragging = true; activePointerId = event.pointerId; moved = false; startX = lastX = event.clientX; startY = lastY = event.clientY; canvas.setPointerCapture?.(event.pointerId); };
  const onMove = (event) => {
    if (!dragging || event.pointerId !== activePointerId) return;
    const dx = event.clientX - lastX;
    const dy = event.clientY - lastY;
    moved ||= Math.hypot(event.clientX - startX, event.clientY - startY) > 2;
    apply(orbitPosition(orbitTarget, position, -dx * 0.008, dy * 0.008));
    lastX = event.clientX; lastY = event.clientY;
  };
  const onUp = (event) => { if (event.pointerId !== activePointerId) return; dragging = false; activePointerId = null; releaseCapture(event.pointerId); };
  const onClick = (event) => { if (moved) { event.stopImmediatePropagation?.(); moved = false; } };
  const onWheel = (event) => {
    event.preventDefault?.();
    radius = Math.max(minimum, Math.min(maximum, radius * (1 + event.deltaY * 0.001)));
    const next = orbitPosition(orbitTarget, position, 0, 0);
    const scale = radius / distanceFrom(orbitTarget, next);
    apply(orbitTarget.map((value, index) => value + (next[index] - value) * scale));
  };
  canvas.addEventListener("pointerdown", onDown);
  canvas.addEventListener("pointermove", onMove);
  canvas.addEventListener("pointerup", onUp);
  canvas.addEventListener("pointercancel", onUp);
  canvas.addEventListener("click", onClick);
  canvas.addEventListener("wheel", onWheel, { passive: false });
  return Object.freeze({
    setTarget(next) { orbitTarget = [...next]; position = positionOf(camera); radius = distanceFrom(orbitTarget, position); minimum = Math.max(0.25, radius * 0.35); maximum = Math.max(minimum, radius * 3); },
    dispose() { if (activePointerId !== null) releaseCapture(activePointerId); dragging = false; activePointerId = null; canvas.removeEventListener?.("pointerdown", onDown); canvas.removeEventListener?.("pointermove", onMove); canvas.removeEventListener?.("pointerup", onUp); canvas.removeEventListener?.("pointercancel", onUp); canvas.removeEventListener?.("click", onClick); canvas.removeEventListener?.("wheel", onWheel); },
  });
}
