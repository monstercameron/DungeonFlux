package phase

import (
	"errors"
	"fmt"

	"github.com/monstercameron/DungeonFlux/internal/core/fsm"
	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/game/combat"
	"github.com/monstercameron/DungeonFlux/internal/game/phase/check"
	"github.com/monstercameron/DungeonFlux/internal/game/phase/cliffhanger"
	"github.com/monstercameron/DungeonFlux/internal/game/phase/conversation"
	"github.com/monstercameron/DungeonFlux/internal/game/phase/creation"
	"github.com/monstercameron/DungeonFlux/internal/game/phase/hook"
	"github.com/monstercameron/DungeonFlux/internal/game/phase/opening"
	"github.com/monstercameron/DungeonFlux/internal/game/phase/resolution"
	"github.com/monstercameron/DungeonFlux/internal/game/rules/dice"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

const (
	eventStart       vocab.EventKind = "phase_start"
	eventCreationEnd vocab.EventKind = "phase_creation_end"
	eventOpeningEnd  vocab.EventKind = "phase_opening_end"
	eventTalk        vocab.EventKind = "phase_talk"
	eventPersuade    vocab.EventKind = "phase_persuade"
	eventStepAway    vocab.EventKind = "phase_step_away"
	eventLeave       vocab.EventKind = "phase_leave"
	eventRoll        vocab.EventKind = "phase_roll"
	eventResolution  vocab.EventKind = "phase_resolution"
	eventCombat      vocab.EventKind = "phase_combat"
	eventCliffhanger vocab.EventKind = "phase_cliffhanger"
	eventSkip        vocab.EventKind = "phase_skip"
	eventReset       vocab.EventKind = "phase_reset"
)

// Definition describes one registered top-level phase.
type Definition struct {
	ID   vocab.StateID
	Stub bool
}

// Definitions returns the registered phases in their canonical run order.
func Definitions() []Definition { return append([]Definition(nil), phaseDefinitions...) }

var phaseDefinitions = []Definition{
	{ID: vocab.StateLobby}, {ID: vocab.StateCreation}, {ID: vocab.StateOpening},
	{ID: vocab.StateExploration}, {ID: vocab.StateConversation}, {ID: vocab.StateCheck},
	{ID: vocab.StateResolution}, {ID: vocab.StateHookEvent}, {ID: vocab.StateCombat},
	{ID: vocab.StateCliffhanger}, {ID: vocab.StateEnd},
}

// Machine is the pure top-level phase dispatcher and active child state.
type Machine struct {
	table                                    fsm.Machine
	paused, conversationDone, strictCreation bool
	oneShot                                  domain.OneShot
	seats                                    []domain.SeatView
	spotlight                                domain.SeatID
	forcedD20                                int
	creation                                 creation.Machine
	opening                                  opening.Machine
	conversation                             conversation.State
	check                                    check.Machine
	resolution                               resolution.Machine
	hook                                     hook.Machine
	combat                                   combat.State
	combatDice                               *dice.Roller
	cliffhanger                              cliffhanger.Machine
}

// Result reports phase effects and the accepted top-level transition.
type Result struct {
	Transition fsm.Result
	Effects    []domain.Effect
	Paused     bool
}

// New creates a dispatcher in Lobby for standalone phase-machine tests.
func New() (Machine, error) {
	return buildMachine(domain.OneShot{}, []byte("dungeonflux-phase"), false)
}

// NewWithSeed creates the game dispatcher with the two-seat creation contract.
func NewWithSeed(oneShot domain.OneShot, seed []byte) (Machine, error) {
	if len(seed) == 0 {
		seed = []byte("dungeonflux-phase")
	}
	return buildMachine(oneShot, seed, true)
}

func buildMachine(oneShot domain.OneShot, seed []byte, strict bool) (Machine, error) {
	table, err := fsm.New(definition())
	if err != nil {
		return Machine{}, err
	}
	created, err := creation.New(seed)
	if err != nil {
		return Machine{}, err
	}
	return Machine{table: table, strictCreation: strict, oneShot: oneShot, creation: created, opening: opening.New(oneShot), seats: initialSeats(), spotlight: 1}, nil
}

// State returns the current top-level phase.
func (m Machine) State() vocab.StateID { return m.table.State() }

// Paused reports whether top-level execution is paused.
func (m Machine) Paused() bool { return m.paused }

// ForceD20 makes the next check or combat roll use face.
func (m *Machine) ForceD20(face int) error {
	if face < 1 || face > 20 {
		return errors.New("d20 must be between 1 and 20")
	}
	m.forcedD20 = face
	return nil
}

// Goto advances through host skips to a debug phase.
func (m *Machine) Goto(target vocab.StateID) error {
	for i := 0; i < len(phaseDefinitions)+1 && m.State() != target; i++ {
		if _, err := m.Step(domain.HostCmd{Cmd: vocab.HostSkip}); err != nil {
			return err
		}
	}
	if m.State() != target {
		return fmt.Errorf("unknown debug phase %q", target)
	}
	return nil
}

// Step dispatches one domain event and returns effects from the active child.
func (m *Machine) Step(event domain.Event) (Result, error) {
	if event == nil {
		return Result{}, &fsm.Rejection{State: m.State(), Reason: fsm.ReasonUnknownEvent}
	}
	if cmd, ok := event.(domain.HostCmd); ok {
		return m.stepHost(cmd)
	}
	if m.paused {
		return Result{}, &fsm.Rejection{State: m.State(), Event: event.Kind(), Reason: fsm.ReasonGuardRejected}
	}
	return m.stepPhase(event)
}

func (m *Machine) stepHost(cmd domain.HostCmd) (Result, error) {
	switch cmd.Cmd {
	case vocab.HostPause:
		m.paused = true
		return Result{Paused: true}, nil
	case vocab.HostResume:
		m.paused = false
		return Result{}, nil
	case vocab.HostReset:
		m.paused, m.conversationDone = false, false
		return m.step(eventReset)
	case vocab.HostSkip:
		m.paused = false
		if m.State() == vocab.StateExploration && m.conversationDone {
			return m.step(eventLeave)
		}
		return m.step(eventSkip)
	case vocab.HostStart:
		return m.step(eventStart)
	default:
		return Result{}, &fsm.Rejection{State: m.State(), Event: eventForHost(cmd.Cmd), Reason: fsm.ReasonUnknownEvent}
	}
}

func (m *Machine) stepPhase(event domain.Event) (Result, error) {
	switch m.State() {
	case vocab.StateLobby, vocab.StateEnd:
		return m.passive(event)
	case vocab.StateCreation:
		return m.stepCreation(event)
	case vocab.StateOpening:
		return m.stepOpening(event)
	case vocab.StateExploration:
		return m.stepExploration(event)
	case vocab.StateConversation:
		return m.stepConversation(event)
	case vocab.StateCheck:
		return m.stepCheck(event)
	case vocab.StateResolution:
		return m.stepResolution(event)
	case vocab.StateHookEvent:
		return m.stepHook(event)
	case vocab.StateCombat:
		return m.stepCombat(event)
	case vocab.StateCliffhanger:
		return m.stepCliffhanger(event)
	default:
		return Result{}, &fsm.Rejection{State: m.State(), Event: event.Kind(), Reason: fsm.ReasonUnknownEvent}
	}
}

func (m *Machine) stepCreation(event domain.Event) (Result, error) {
	if act, ok := event.(domain.Act); ok && act.Move == vocab.MoveReady {
		event = domain.PCLocked{Seat: act.Seat}
	}
	if isPassive(event) {
		return Result{}, nil
	}
	if _, ok := event.(domain.PCLocked); ok && !m.strictCreation {
		return m.step(eventCreationEnd)
	}
	result, err := m.creation.Step(event)
	if err != nil {
		return Result{}, err
	}
	m.updateCreationSeat(result.Seat)
	if !result.Complete {
		return Result{Effects: result.Effects}, nil
	}
	return m.transition(eventCreationEnd, result.Effects)
}

func (m *Machine) stepOpening(event domain.Event) (Result, error) {
	result := m.opening.Step(event)
	if _, ok := event.(domain.LineDone); ok && result.State == opening.StateReady {
		return m.transition(eventOpeningEnd, result.Effects)
	}
	return Result{Effects: result.Effects}, nil
}

func (m *Machine) stepExploration(event domain.Event) (Result, error) {
	act, ok := event.(domain.Act)
	if !ok {
		return m.passive(event)
	}
	switch act.Move {
	case vocab.MoveTalkVell:
		m.spotlight, m.conversation = act.Seat, conversation.State{Seat: act.Seat}
		return m.step(eventTalk)
	case vocab.MoveLeave:
		m.spotlight = act.Seat
		if err := m.startHook(); err != nil {
			return Result{}, err
		}
		return m.step(eventLeave)
	default:
		return m.unhandled(event)
	}
}

func (m *Machine) stepConversation(event domain.Event) (Result, error) {
	if act, ok := event.(domain.Act); ok {
		switch act.Move {
		case vocab.MovePersuade:
			return m.startCheck()
		case vocab.MoveStepAway:
			return m.step(eventStepAway)
		}
	}
	result, err := conversation.Step(m.conversation, conversation.Event{Event: event})
	if err != nil {
		return Result{}, err
	}
	m.conversation = result.State
	out := Result{Effects: result.Effects}
	for _, derived := range result.Events {
		next, nextErr := m.stepConversation(derived)
		if nextErr != nil {
			return Result{}, nextErr
		}
		out.Effects = append(out.Effects, next.Effects...)
	}
	return out, nil
}

func (m *Machine) startCheck() (Result, error) {
	roller := newDice()
	if m.forcedD20 != 0 {
		if err := roller.ForceD20(m.forcedD20); err != nil {
			return Result{}, err
		}
		m.forcedD20 = 0
	}
	created, err := check.New(check.Config{CheckID: "persuasion", Seat: m.spotlight, Charisma: 14, Proficient: true, DC: 10}, roller)
	if err != nil {
		return Result{}, err
	}
	m.check = created
	if _, err = m.check.Step(domain.Act{Move: vocab.MovePersuade}); err != nil {
		return Result{}, err
	}
	return m.step(eventPersuade)
}

func (m *Machine) stepCheck(event domain.Event) (Result, error) {
	if _, ok := event.(domain.TimerFired); !ok {
		return m.passive(event)
	}
	if _, err := m.check.Step(event); err != nil {
		return Result{}, err
	}
	outcome, ok := m.check.Outcome()
	if !ok {
		return Result{}, errors.New("check has no outcome")
	}
	created, err := resolution.New(outcome.Success, resolution.Config{SuccessUtterance: "reveal", FailureUtterance: "refuse", SuccessText: "The clue is yours.", FailureText: "She refuses.", SuccessCanned: "canned-reveal", FailureCanned: "canned-refuse"})
	if err != nil {
		return Result{}, err
	}
	m.resolution = created
	started, err := m.resolution.Start()
	if err != nil {
		return Result{}, err
	}
	return m.transition(eventRoll, started.Effects)
}

func (m *Machine) stepResolution(event domain.Event) (Result, error) {
	if line, ok := event.(domain.LineDone); ok && line.UtteranceID == "" {
		line.UtteranceID = "reveal"
		if _, err := m.resolution.Step(line); err != nil {
			line.UtteranceID = "refuse"
		}
		event = line
	}
	result, err := m.resolution.Step(event)
	if err != nil {
		if line, ok := event.(domain.LineDone); ok && line.UtteranceID != "" {
			alternate := "reveal"
			if string(line.UtteranceID) == alternate {
				alternate = "refuse"
			}
			result, err = m.resolution.Step(domain.LineDone{UtteranceID: domain.UtteranceID(alternate)})
		}
	}
	if err != nil {
		return Result{}, err
	}
	if result.State != resolution.Done {
		return Result{Effects: result.Effects}, nil
	}
	out, err := m.transition(eventResolution, result.Effects)
	if err == nil {
		m.conversationDone = true
	}
	return out, err
}
