package hook

import (
	"errors"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

// State is the stranger-arrival lifecycle.
type State string

const (
	// ArrivalClip means the pre-made arrival clip is playing.
	ArrivalClip State = "arrival_clip"
	// StrangerLine means the stranger's hook line is playing.
	StrangerLine State = "stranger_line"
	// Done means the hook was delivered and combat may begin.
	Done State = "done"
)

// Config identifies the arrival clip and stranger lines.
type Config struct {
	ArrivalClip       domain.AssetID
	StrangerUtterance string
	StrangerText      string
	CannedUtterance   string
	CannedLine        domain.AssetID
	StrangerHook      string
	ClueRelocated     bool
}

// Result reports effects and whether the combat transition is ready.
type Result struct {
	State   State
	Effects []domain.Effect
	Combat  bool
}

// Machine sequences one stranger arrival.
type Machine struct {
	config Config
	state  State
	active domain.UtteranceID
	ready  bool
}

// New creates a hook machine waiting for its arrival clip.
func New(config Config) (Machine, error) {
	if config.ArrivalClip == "" {
		return Machine{}, errors.New("arrival clip is required")
	}
	if config.StrangerUtterance == "" || config.CannedUtterance == "" {
		return Machine{}, errors.New("stranger utterance IDs are required")
	}
	return Machine{config: config, state: ArrivalClip}, nil
}

// State reports the current hook lifecycle state.
func (m Machine) State() State { return m.state }

// Start starts the arrival clip slot.
func (m *Machine) Start() (Result, error) {
	if m == nil || m.config.ArrivalClip == "" {
		return Result{}, errors.New("hook machine is nil")
	}
	if m.state != ArrivalClip {
		return Result{}, errors.New("arrival clip already started")
	}
	return Result{State: ArrivalClip, Effects: []domain.Effect{
		domain.PlayCanned{AssetID: m.config.ArrivalClip},
	}}, nil
}

// Step advances the clip or stranger line. A completed stranger line marks
// the combat transition as ready.
func (m *Machine) Step(event domain.Event) (Result, error) {
	if m == nil {
		return Result{}, errors.New("hook machine is nil")
	}
	switch m.state {
	case ArrivalClip:
		return m.stepClip(event)
	case StrangerLine:
		return m.stepLine(event)
	case Done:
		return Result{State: Done, Combat: true}, nil
	default:
		return Result{}, errors.New("unknown hook state")
	}
}

func (m *Machine) stepClip(event domain.Event) (Result, error) {
	clip, ok := event.(domain.ClipDone)
	if !ok || clip.AssetID != m.config.ArrivalClip {
		return Result{}, errors.New("arrival clip completion is required")
	}
	m.state = StrangerLine
	m.active = domain.UtteranceID(m.config.StrangerUtterance)
	return Result{State: StrangerLine, Effects: []domain.Effect{domain.StartLine{
		UtteranceID: m.active, Role: vocab.RoleStrangerLines, Input: m.config.StrangerText,
	}}}, nil
}

func (m *Machine) stepLine(event domain.Event) (Result, error) {
	switch value := event.(type) {
	case domain.LineFailed:
		if value.UtteranceID != m.active {
			return Result{}, errors.New("stranger line failure does not match active utterance")
		}
		m.active = domain.UtteranceID(m.config.CannedUtterance)
		return Result{State: StrangerLine, Effects: []domain.Effect{
			domain.DropLine{UtteranceID: value.UtteranceID},
			domain.PlayCanned{UtteranceID: m.active, AssetID: m.config.CannedLine},
		}}, nil
	case domain.LineDone:
		if value.UtteranceID != m.active {
			return Result{}, errors.New("stranger line completion does not match active utterance")
		}
		m.state, m.ready = Done, true
		return Result{State: Done, Combat: true}, nil
	default:
		return Result{}, errors.New("hook event is not a clip or line completion")
	}
}
