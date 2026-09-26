package cliffhanger

import (
	"errors"

	"github.com/monstercameron/DungeonFlux/internal/core/fsm"
	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

// State identifies the two terminal phase states.
type State = vocab.StateID

const (
	// StateReady is the phase before its entry actions have run.
	StateReady State = "ready"
	// StatePlaying is the active cliffhanger narration and clip.
	StatePlaying State = "playing"
	// StateEnd is the end-card state.
	StateEnd State = "end"
)

const (
	eventEnter    fsm.Event = "cliffhanger_enter"
	eventLineDone fsm.Event = vocab.EventLineDone
	eventLineFail fsm.Event = vocab.EventLineFailed
)

// Config supplies the pre-emptive clip assets and narration identities.
type Config struct {
	LiveClip        domain.Asset
	GenericClip     domain.Asset
	AnimatedStill   domain.Asset
	LineID          domain.UtteranceID
	CannedLineID    domain.UtteranceID
	CannedAssetID   domain.AssetID
	ClueFoundViaNPC bool
	NarrationInput  string
	VoiceID         string
}

// Result reports a state transition, selected clip, and runtime effects.
type Result struct {
	Transition fsm.Result
	Effects    []domain.Effect
	Clip       domain.Asset
	EndCard    bool
}

// Machine is the pure cliffhanger phase machine.
type Machine struct {
	table    fsm.Machine
	clip     clipSlot
	config   Config
	lineID   domain.UtteranceID
	started  bool
	selected domain.Asset
	endCard  bool
}

// New creates a cliffhanger machine with the ordered clip fallback chain.
func New(config Config) (Machine, error) {
	if config.LiveClip.ID == "" || config.GenericClip.ID == "" || config.AnimatedStill.ID == "" {
		return Machine{}, errors.New("cliffhanger requires live, generic, and still assets")
	}
	if config.LineID == "" || config.CannedLineID == "" || config.CannedAssetID == "" {
		return Machine{}, errors.New("cliffhanger requires line and canned identities")
	}
	clip := newClipSlot(config.GenericClip, config.AnimatedStill)
	definition := fsm.Def{
		Initial: StateReady,
		States:  []fsm.State{StateReady, StatePlaying, StateEnd},
		Transitions: []fsm.Transition{
			{From: StateReady, Event: eventEnter, To: StatePlaying},
			{From: StatePlaying, Event: eventLineDone, To: StateEnd},
		},
	}
	table, err := fsm.New(definition)
	if err != nil {
		return Machine{}, err
	}
	return Machine{table: table, clip: clip, config: config, lineID: config.LineID}, nil
}

// State returns the current cliffhanger state.
func (m Machine) State() State { return State(m.table.State()) }

// Clip returns the clip or animated still selected at entry.
func (m Machine) Clip() (domain.Asset, bool) { return m.selected, m.selected.ID != "" }

// EndCard reports whether the end card should be shown.
func (m Machine) EndCard() bool { return m.endCard }

// Step applies one asset or narration event. Asset events may arrive before
// entry because the live clip is generated in the run scope.
func (m *Machine) Step(event domain.Event) (Result, error) {
	if event == nil {
		return Result{}, errors.New("cliffhanger event is nil")
	}
	if assetEvent(event) {
		if err := m.clip.Step(event); err != nil {
			return Result{}, err
		}
		return Result{Clip: m.selected}, nil
	}
	switch value := event.(type) {
	case domain.LineDone:
		if m.State() != StatePlaying || value.UtteranceID != m.lineID {
			return Result{}, errors.New("line_done does not belong to cliffhanger")
		}
		transition, err := m.table.Step(eventLineDone)
		if err != nil {
			return Result{}, err
		}
		m.endCard = true
		return Result{Transition: transition, Clip: m.selected, EndCard: true}, nil
	case domain.LineFailed:
		return m.lineFailed(value)
	default:
		return Result{}, errors.New("event is not accepted by cliffhanger")
	}
}

// Enter starts the phase. The clip is selected immediately and narration is
// started independently, so a slow video never delays the end card.
func (m *Machine) Enter() (Result, error) {
	if m.started {
		return Result{}, errors.New("cliffhanger already entered")
	}
	transition, err := m.table.Step(eventEnter)
	if err != nil {
		return Result{}, err
	}
	m.started = true
	m.selected, _ = m.clip.Asset()
	if m.selected.ID == "" {
		m.selected = m.config.GenericClip
	}
	effect := domain.StartLine{
		UtteranceID: m.lineID,
		Role:        vocab.RoleCliffhanger,
		Voice:       m.config.VoiceID,
		Input:       m.config.NarrationInput,
	}
	return Result{Transition: transition, Effects: []domain.Effect{effect}, Clip: m.selected}, nil
}

func (m *Machine) lineFailed(event domain.LineFailed) (Result, error) {
	if m.State() != StatePlaying || event.UtteranceID != m.lineID {
		return Result{}, errors.New("line_failed does not belong to cliffhanger")
	}
	m.lineID = m.config.CannedLineID
	effect := domain.PlayCanned{UtteranceID: m.lineID, AssetID: m.config.CannedAssetID}
	return Result{Effects: []domain.Effect{effect}, Clip: m.selected}, nil
}

func assetEvent(event domain.Event) bool {
	switch event.(type) {
	case domain.AssetPartial, domain.AssetReady, domain.AssetFailed:
		return true
	default:
		return false
	}
}

type clipSlot struct {
	state     clipState
	fallbacks []domain.Asset
	index     int
	asset     domain.Asset
}

type clipState string

const (
	clipPending  clipState = "pending"
	clipReady    clipState = "ready"
	clipFallback clipState = "fallback"
)

func newClipSlot(generic, still domain.Asset) clipSlot {
	return clipSlot{state: clipPending, fallbacks: []domain.Asset{generic, still}}
}

func (s *clipSlot) Step(event domain.Event) error {
	switch value := event.(type) {
	case domain.AssetPartial:
		return nil
	case domain.AssetReady:
		if value.Slot != string(vocab.SlotCliffhangerClip) || s.state != clipPending {
			return nil
		}
		s.asset = value.Asset
		s.state = clipReady
	case domain.AssetFailed:
		if value.Slot != string(vocab.SlotCliffhangerClip) {
			return nil
		}
		s.nextFallback()
	default:
		return errors.New("event is not a clip-slot event")
	}
	return nil
}

func (s *clipSlot) nextFallback() {
	if s.index >= len(s.fallbacks) {
		s.asset = domain.Asset{}
		s.state = clipPending
		return
	}
	s.asset = s.fallbacks[s.index]
	s.index++
	s.state = clipFallback
}

func (s clipSlot) Asset() (domain.Asset, bool) {
	return s.asset, s.asset.ID != ""
}
