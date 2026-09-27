package check

import (
	"errors"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/game/rules/dice"
	"github.com/monstercameron/DungeonFlux/internal/game/rules/rulings"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

// State is the lifecycle of a persuasion check.
type State string

const (
	// Offered means the phone may offer the persuasion move.
	Offered State = "offered"
	// Rolling means the d20 result is visible and awaits the roll timer.
	Rolling State = "rolling"
	// Resolved means the outcome line may finish the phase.
	Resolved State = "resolved"
)

// EventKind identifies a check event emitted by the machine.
type EventKind string

const (
	// EventOffered announces the initial dice view.
	EventOffered EventKind = "check_offered"
	// EventRollMade records the d20 and modifier.
	EventRollMade EventKind = "roll_made"
	// EventResolved records the success or failure outcome.
	EventResolved EventKind = "check_resolved"
	// EventNarrated marks the end of the dice narration.
	EventNarrated EventKind = "check_narrated"
)

// Event is one deterministic check event.
type Event struct {
	Kind    EventKind
	CheckID string
	Outcome *rulings.CheckOutcome
}

// Config contains the content and character inputs for a persuasion check.
type Config struct {
	CheckID    string
	Seat       domain.SeatID
	Charisma   int
	Proficient bool
	DC         int
	Adv        int8
}

// Result reports the new state and events emitted by Step.
type Result struct {
	State  State
	Events []Event
}

// Machine resolves one persuasion check with a deterministic dice source.
type Machine struct {
	config  Config
	roller  *dice.Roller
	state   State
	outcome *rulings.CheckOutcome
	started bool
}

// New creates an offered persuasion check. DC zero uses the demo DC of 10.
func New(config Config, roller *dice.Roller) (Machine, error) {
	if roller == nil {
		return Machine{}, errors.New("check dice source is nil")
	}
	if config.CheckID == "" {
		return Machine{}, errors.New("check ID is required")
	}
	if config.DC == 0 {
		config.DC = 10
	}
	if config.DC < 1 {
		return Machine{}, errors.New("check DC must be positive")
	}
	if config.Adv < -1 || config.Adv > 1 {
		return Machine{}, errors.New("check advantage must be -1, 0, or 1")
	}
	return Machine{config: config, roller: roller, state: Offered}, nil
}

// State reports the current check lifecycle state.
func (m Machine) State() State { return m.state }

// Outcome returns a copy of the resolved outcome, if one exists.
func (m Machine) Outcome() (rulings.CheckOutcome, bool) {
	if m.outcome == nil {
		return rulings.CheckOutcome{}, false
	}
	out := *m.outcome
	out.Roll.Faces = append([]int(nil), out.Roll.Faces...)
	out.Breakdown = append([]rulings.Term(nil), out.Breakdown...)
	return out, true
}

// Offer returns the initial OFFERED event without changing state.
func (m Machine) Offer() Event {
	return Event{Kind: EventOffered, CheckID: m.config.CheckID}
}

// Step applies persuade, roll completion, or host skip. Unrelated events are
// rejected without changing state.
func (m *Machine) Step(event domain.Event) (Result, error) {
	if m == nil || m.roller == nil {
		return Result{}, errors.New("check machine is nil")
	}
	if event == nil {
		return Result{}, errors.New("check event is nil")
	}
	switch m.state {
	case Offered:
		return m.stepOffered(event)
	case Rolling:
		return m.stepRolling(event)
	case Resolved:
		return Result{State: Resolved}, nil
	default:
		return Result{}, errors.New("unknown check state")
	}
}

func (m *Machine) stepOffered(event domain.Event) (Result, error) {
	act, ok := event.(domain.Act)
	if !ok || act.Move != vocab.MovePersuade {
		return Result{}, errors.New("persuade is required while check is offered")
	}
	outcome, err := rulings.Persuasion(m.roller, m.config.CheckID, m.config.Charisma, m.config.Proficient, m.config.DC, m.config.Adv)
	if err != nil {
		return Result{}, err
	}
	m.outcome = &outcome
	m.state = Rolling
	m.started = true
	return Result{State: Rolling, Events: []Event{{Kind: EventRollMade, CheckID: m.config.CheckID, Outcome: &outcome}, {Kind: EventResolved, CheckID: m.config.CheckID, Outcome: &outcome}}}, nil
}

func (m *Machine) stepRolling(event domain.Event) (Result, error) {
	if !isRollResolved(event) {
		return Result{}, errors.New("roll_resolved is required while check is rolling")
	}
	m.state = Resolved
	return Result{State: Resolved, Events: []Event{{Kind: EventNarrated, CheckID: m.config.CheckID, Outcome: m.outcome}}}, nil
}

func isRollResolved(event domain.Event) bool {
	if timer, ok := event.(domain.TimerFired); ok {
		return timer.Name == "roll_resolved"
	}
	if cmd, ok := event.(domain.HostCmd); ok {
		return cmd.Cmd == vocab.HostSkip
	}
	return false
}
