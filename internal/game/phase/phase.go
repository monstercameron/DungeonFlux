package phase

import (
	"github.com/monstercameron/DungeonFlux/internal/core/fsm"
	"github.com/monstercameron/DungeonFlux/internal/domain"
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
)

// Definition describes one registered top-level phase.
type Definition struct {
	ID   vocab.StateID
	Stub bool
}

// Definitions returns the registered phases in their canonical run order.
func Definitions() []Definition {
	return append([]Definition(nil), phaseDefinitions...)
}

var phaseDefinitions = []Definition{
	{ID: vocab.StateLobby, Stub: true},
	{ID: vocab.StateCreation, Stub: true},
	{ID: vocab.StateOpening, Stub: true},
	{ID: vocab.StateExploration, Stub: true},
	{ID: vocab.StateConversation, Stub: true},
	{ID: vocab.StateCheck, Stub: true},
	{ID: vocab.StateResolution, Stub: true},
	{ID: vocab.StateHookEvent, Stub: true},
	{ID: vocab.StateCombat, Stub: true},
	{ID: vocab.StateCliffhanger, Stub: true},
	{ID: vocab.StateEnd, Stub: true},
}

// Machine is the pure top-level phase dispatcher.
type Machine struct {
	table            fsm.Machine
	paused           bool
	conversationDone bool
}

// Result reports a phase dispatch and whether the event was handled while
// leaving the machine paused.
type Result struct {
	Transition fsm.Result
	Paused     bool
}

// New creates a dispatcher in Lobby.
func New() (Machine, error) {
	table, err := fsm.New(definition())
	if err != nil {
		return Machine{}, err
	}
	return Machine{table: table}, nil
}

// State returns the current top-level phase.
func (m Machine) State() vocab.StateID { return m.table.State() }

// Paused reports whether top-level execution is paused.
func (m Machine) Paused() bool { return m.paused }

// Step dispatches one domain event. Events not belonging to the current
// phase are rejected by the underlying table without changing state.
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
		m.paused = false
		m.conversationDone = false
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
	phaseEvent := eventForDomain(m.State(), event)
	if phaseEvent == "" {
		return Result{}, &fsm.Rejection{State: m.State(), Event: event.Kind(), Reason: fsm.ReasonUnknownEvent}
	}
	return m.step(phaseEvent)
}

func (m *Machine) step(event vocab.EventKind) (Result, error) {
	transition, err := m.table.Step(event)
	if err == nil && transition.To == vocab.StateExploration && transition.From == vocab.StateResolution {
		m.conversationDone = true
	}
	return Result{Transition: transition, Paused: m.paused}, err
}

func definition() fsm.Def {
	states := make([]fsm.State, 0, len(phaseDefinitions))
	for _, phase := range phaseDefinitions {
		states = append(states, phase.ID)
	}
	transitions := []fsm.Transition{
		{From: vocab.StateLobby, Event: eventStart, To: vocab.StateCreation},
		{From: vocab.StateCreation, Event: eventCreationEnd, To: vocab.StateOpening},
		{From: vocab.StateOpening, Event: eventOpeningEnd, To: vocab.StateExploration},
		{From: vocab.StateExploration, Event: eventTalk, To: vocab.StateConversation},
		{From: vocab.StateExploration, Event: eventLeave, To: vocab.StateHookEvent},
		{From: vocab.StateConversation, Event: eventPersuade, To: vocab.StateCheck},
		{From: vocab.StateConversation, Event: eventStepAway, To: vocab.StateExploration},
		{From: vocab.StateCheck, Event: eventRoll, To: vocab.StateResolution},
		{From: vocab.StateResolution, Event: eventResolution, To: vocab.StateExploration},
		{From: vocab.StateHookEvent, Event: eventCombat, To: vocab.StateCombat},
		{From: vocab.StateCombat, Event: eventCliffhanger, To: vocab.StateCliffhanger},
		{From: vocab.StateCliffhanger, Event: eventCliffhanger, To: vocab.StateEnd},
	}
	for index, phase := range phaseDefinitions {
		transitions = append(transitions, fsm.Transition{From: phase.ID, Event: eventSkip, To: skipTarget(index)})
		transitions = append(transitions, fsm.Transition{From: phase.ID, Event: eventReset, To: vocab.StateLobby})
	}
	return fsm.Def{Initial: vocab.StateLobby, States: states, Transitions: transitions}
}

func skipTarget(index int) vocab.StateID {
	if index == len(phaseDefinitions)-1 {
		return vocab.StateLobby
	}
	if phaseDefinitions[index].ID == vocab.StateResolution {
		return vocab.StateExploration
	}
	return phaseDefinitions[index+1].ID
}

func eventForDomain(state vocab.StateID, event domain.Event) vocab.EventKind {
	switch typed := event.(type) {
	case domain.PCLocked:
		if state == vocab.StateCreation {
			return eventCreationEnd
		}
	case domain.TimerFired:
		if state == vocab.StateCreation && typed.Name == "creation_timeout" {
			return eventCreationEnd
		}
		if state == vocab.StateCheck && typed.Name == "roll_resolved" {
			return eventRoll
		}
	case domain.LineDone:
		switch state {
		case vocab.StateOpening:
			return eventOpeningEnd
		case vocab.StateResolution:
			return eventResolution
		case vocab.StateHookEvent:
			return eventCombat
		case vocab.StateCombat:
			return eventCliffhanger
		case vocab.StateCliffhanger:
			return eventCliffhanger
		}
	case domain.Act:
		switch {
		case state == vocab.StateExploration && typed.Move == vocab.MoveTalkVell:
			return eventTalk
		case state == vocab.StateExploration && typed.Move == vocab.MoveLeave:
			return eventLeave
		case state == vocab.StateConversation && typed.Move == vocab.MovePersuade:
			return eventPersuade
		case state == vocab.StateConversation && typed.Move == vocab.MoveStepAway:
			return eventStepAway
		}
	}
	return ""
}

func eventForHost(command vocab.HostCmd) vocab.EventKind {
	return vocab.EventKind(string(command))
}

const eventReset vocab.EventKind = "phase_reset"
