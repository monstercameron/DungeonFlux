package resolution

import (
	"errors"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

// State is the outcome narration lifecycle.
type State string

const (
	// Success is the reveal branch.
	Success State = "success"
	// Failure is the refusal branch.
	Failure State = "failure"
	// Done means the active line has completed.
	Done State = "done"
)

// Config identifies the live and canned lines for both outcomes.
type Config struct {
	SuccessUtterance, FailureUtterance string
	SuccessText, FailureText           string
	SuccessCanned, FailureCanned       domain.AssetID
}

// Result reports effects emitted while handling an event.
type Result struct {
	State   State
	Effects []domain.Effect
}

// Machine narrates one check outcome.
type Machine struct {
	config  Config
	state   State
	active  domain.UtteranceID
	canned  domain.UtteranceID
	asset   domain.AssetID
	started bool
}

// New creates a resolution machine and validates both branch utterances.
func New(success bool, config Config) (Machine, error) {
	if config.SuccessUtterance == "" || config.FailureUtterance == "" {
		return Machine{}, errors.New("resolution utterance IDs are required")
	}
	machine := Machine{config: config}
	if success {
		machine.state = Success
		machine.active, machine.canned, machine.asset = domain.UtteranceID(config.SuccessUtterance), domain.UtteranceID(config.SuccessUtterance+"-canned"), config.SuccessCanned
	} else {
		machine.state = Failure
		machine.active, machine.canned, machine.asset = domain.UtteranceID(config.FailureUtterance), domain.UtteranceID(config.FailureUtterance+"-canned"), config.FailureCanned
	}
	return machine, nil
}

// State reports the current narration state.
func (m Machine) State() State { return m.state }

// Start emits the live NPC line for the selected branch.
func (m *Machine) Start() (Result, error) {
	if m == nil || m.active == "" {
		return Result{}, errors.New("resolution machine is nil")
	}
	if m.started {
		return Result{}, errors.New("resolution line already started")
	}
	m.started = true
	return Result{State: m.state, Effects: []domain.Effect{m.startLine()}}, nil
}

// Step handles completion or failure of the active line. A failed live line
// is replaced by a canned line; completion of that fallback finishes the
// resolution branch.
func (m *Machine) Step(event domain.Event) (Result, error) {
	if m == nil || m.active == "" {
		return Result{}, errors.New("resolution machine is nil")
	}
	if !m.started {
		return Result{}, errors.New("resolution line has not started")
	}
	switch value := event.(type) {
	case domain.LineFailed:
		if value.UtteranceID != m.active || m.state == Done {
			return Result{}, errors.New("line failure does not match active utterance")
		}
		m.active = m.canned
		return Result{State: m.state, Effects: []domain.Effect{
			domain.DropLine{UtteranceID: value.UtteranceID},
			domain.PlayCanned{UtteranceID: m.canned, AssetID: m.asset},
		}}, nil
	case domain.LineDone:
		if value.UtteranceID != m.active {
			return Result{}, errors.New("line completion does not match active utterance")
		}
		m.state = Done
		return Result{State: Done}, nil
	default:
		return Result{}, errors.New("resolution event is not a line completion")
	}
}

func (m Machine) startLine() domain.StartLine {
	role := vocab.RoleNPCRefuse
	text := m.config.FailureText
	if m.state == Success {
		role = vocab.RoleNPCReveal
		text = m.config.SuccessText
	}
	return domain.StartLine{UtteranceID: m.active, Role: role, Input: text}
}
