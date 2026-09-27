package api

import (
	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

// reconnectView advances cached presentation clocks, never engine timers.
// The caller holds the hub lock so publication time and snapshot stay paired.
func (h *WatchHub) reconnectView() domain.View {
	view := h.latest.DeepCopy()
	elapsed := max(int64(0), h.clock.Since(h.published).Milliseconds())
	if view.Path == vocab.StateCreation && !view.Paused {
		for index := range view.Seats {
			timer := &view.Seats[index].TurnTimer
			if timer.TotalMS > 0 && !timer.Frozen {
				timer.RemainingMS = max(int64(0), timer.RemainingMS-elapsed)
			}
		}
	}
	cam := &view.KillCam
	if cam.URL == "" || !cam.Playing {
		return view
	}
	remaining := max(int64(0), cam.DurationMS-cam.OffsetMS)
	if elapsed >= remaining {
		view.KillCam = domain.KillCamView{}
		return view
	}
	cam.OffsetMS += elapsed
	return view
}
