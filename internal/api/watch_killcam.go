package api

import "github.com/monstercameron/DungeonFlux/internal/domain"

// reconnectView advances only the presentation offset, never the engine timer.
// The caller holds the hub lock so publication time and snapshot stay paired.
func (h *WatchHub) reconnectView() domain.View {
	view := h.latest.DeepCopy()
	cam := &view.KillCam
	if cam.URL == "" || !cam.Playing {
		return view
	}
	elapsed := max(int64(0), h.clock.Since(h.published).Milliseconds())
	remaining := max(int64(0), cam.DurationMS-cam.OffsetMS)
	if elapsed >= remaining {
		view.KillCam = domain.KillCamView{}
		return view
	}
	cam.OffsetMS += elapsed
	return view
}
