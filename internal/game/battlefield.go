package game

import (
	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/game/combat"
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

// BattlefieldViewFromCombat combines authored navigation data with the
// engine-owned combat scene state. The renderer receives this snapshot and
// does not calculate paths, targets, or camera choices.
func BattlefieldViewFromCombat(source domain.Battlefield, reports BattlefieldReports, visible bool, state combat.State) *domain.BattlefieldView {
	view := BattlefieldViewFrom(source, reports, visible)
	combatView := CombatViewFrom(state, 0, 0, 0)
	view.Tokens = append([]domain.TokenView(nil), combatView.Tokens...)
	view.Highlights = append([]domain.HighlightView(nil), combatView.Highlights...)
	view.TurnOrder = append([]domain.TurnEntry(nil), combatView.TurnOrder...)
	view.Round = combatView.Round
	presentation := state.Presentation.Camera
	view.Camera = domain.CameraView{Preset: presentation.Preset, FocusTokenID: domain.TokenID(presentation.FocusTokenID), Seq: presentation.Seq}
	return view
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
