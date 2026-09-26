package opening

import (
	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

// OpeningCannedAsset is the build-time fallback audio for the opening line.
const OpeningCannedAsset domain.AssetID = "canned_opening"

const (
	// OpeningClipSlot is the slot for the establishing video.
	OpeningClipSlot = "opening_clip"
	// OpeningStillSlot is the layered still fallback for the clip.
	OpeningStillSlot = "opening_still"
	// OpeningShot is the build-time establishing shot identifier.
	OpeningShot = "EST_WIDE_PUSH"
	// OpeningUtteranceID identifies the opening narration line.
	OpeningUtteranceID domain.UtteranceID = "opening"
)

// State identifies the lifecycle of an opening phase.
type State string

const (
	// StateReady means the phase has not started.
	StateReady State = "ready"
	// StatePlaying means the opening line is in progress.
	StatePlaying State = "playing"
)

// Result contains the next phase state and effects for the runtime.
type Result struct {
	State   State
	Effects []domain.Effect
	View    domain.View
}

// Machine is the pure Opening phase machine.
type Machine struct {
	state   State
	oneShot domain.OneShot
	view    domain.View
}

// New creates an Opening machine ready to play the canned opening line.
func New(oneShot ...domain.OneShot) Machine {
	var shot domain.OneShot
	if len(oneShot) != 0 {
		shot = oneShot[0]
	}
	return Machine{state: StateReady, oneShot: shot, view: battlefieldView(shot)}
}

// State reports the current opening lifecycle state.
func (m Machine) State() State {
	return m.state
}

// View returns the opening projection, including the hidden battlefield used
// by the client while the establishing clip is playing.
func (m Machine) View() domain.View {
	return m.view.DeepCopy()
}

// Enter starts the opening phase and emits the canned opening audio.
func (m *Machine) Enter() Result {
	if m.state != StateReady {
		return Result{State: m.state, View: m.View()}
	}
	m.state = StatePlaying
	return Result{
		State: m.state,
		View:  m.View(),
		Effects: []domain.Effect{
			domain.ComposeStill{Slot: OpeningStillSlot, Background: "background_tavern", Layers: []string{"pc_seat_1", "pc_seat_2", "mother_vell"}},
			domain.GenerateClip{Slot: OpeningClipSlot, Shot: OpeningShot, Resolution: "720p"},
			domain.StartLine{UtteranceID: OpeningUtteranceID, Role: vocab.RoleOpening, Voice: "dm", Hold: true, GateOnClip: true},
		},
	}
}

// Step advances the opening phase. A completed line returns to the caller's
// dispatcher; other events are ignored so late callbacks cannot add effects.
func (m *Machine) Step(event domain.Event) Result {
	if m == nil {
		return Result{}
	}
	if m.state != StatePlaying || event == nil {
		return Result{State: m.state, View: m.View()}
	}
	switch typed := event.(type) {
	case domain.LineFailed:
		if typed.UtteranceID != "" && typed.UtteranceID != OpeningUtteranceID {
			return Result{State: m.state, View: m.View()}
		}
		return Result{State: m.state, View: m.View(), Effects: []domain.Effect{domain.PlayCanned{UtteranceID: OpeningUtteranceID, AssetID: OpeningCannedAsset}}}
	case domain.LineDone:
		if typed.UtteranceID != "" && typed.UtteranceID != OpeningUtteranceID {
			return Result{State: m.state, View: m.View()}
		}
		m.state = StateReady
		return Result{State: m.state, View: m.View()}
	default:
		return Result{State: m.state, View: m.View()}
	}
}

func battlefieldView(oneShot domain.OneShot) domain.View {
	battlefield := oneShot.Encounter.Battlefield
	return domain.View{Path: vocab.StateOpening, Battlefield: &domain.BattlefieldView{
		Mode: battlefield.Mode, Visible: false, SceneURL: battlefield.SceneURL,
		LiteURL: battlefield.LiteURL, Transform: battlefield.Transform,
		Cameras: cloneCameras(battlefield.Cameras), Grid: battlefield.Grid, Flat: battlefield.Flat,
	}}
}

func cloneCameras(cameras map[string]domain.CameraDef) map[string]domain.CameraDef {
	if cameras == nil {
		return nil
	}
	out := make(map[string]domain.CameraDef, len(cameras))
	for name, camera := range cameras {
		out[name] = camera
	}
	return out
}
