package game

import (
	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

const (
	// BattlefieldModeSplat is the accelerated battlefield renderer.
	BattlefieldModeSplat = "SPLAT"
	// BattlefieldModeFlat is the portable battlefield fallback renderer.
	BattlefieldModeFlat = "FLAT"
)

// BattlefieldReports is the room-level, monotonic state of the splat client.
// A failure or host disable latches FLAT for the current run. Repeated ready
// reports are harmless and do not clear either latch.
type BattlefieldReports struct {
	SplatReady  bool
	SplatFailed bool
	SplatOff    bool
}

// Report records a client report relevant to battlefield selection.
func (r *BattlefieldReports) Report(kind vocab.ReportKind) {
	switch kind {
	case vocab.ReportSplatReady:
		r.SplatReady = true
	case vocab.ReportSplatFailed:
		r.SplatFailed = true
	}
}

// Disable records the replayable host SPLAT_OFF decision.
func (r *BattlefieldReports) Disable() { r.SplatOff = true }

// Mode applies the battlefield mode rule to the room-level reports.
func (r BattlefieldReports) Mode() string {
	if r.SplatReady && !r.SplatFailed && !r.SplatOff {
		return BattlefieldModeSplat
	}
	return BattlefieldModeFlat
}

// BattlefieldViewFrom projects the Opening battlefield into the client view.
// Opening callers should pass false for visible; Combat callers may pass true.
func BattlefieldViewFrom(source domain.Battlefield, reports BattlefieldReports, visible bool) *domain.BattlefieldView {
	return &domain.BattlefieldView{
		Mode:      reports.Mode(),
		Visible:   visible,
		SceneURL:  source.SceneURL,
		LiteURL:   source.LiteURL,
		Transform: source.Transform,
		Cameras:   cloneCameras(source.Cameras),
		Grid:      cloneGrid(source.Grid),
		Flat:      source.Flat,
	}
}

func cloneCameras(source map[string]domain.CameraDef) map[string]domain.CameraDef {
	if source == nil {
		return nil
	}
	out := make(map[string]domain.CameraDef, len(source))
	for name, camera := range source {
		out[name] = camera
	}
	return out
}

func cloneGrid(source domain.Grid) domain.Grid {
	source.Walkable = append([]bool(nil), source.Walkable...)
	return source
}
