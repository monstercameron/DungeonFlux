package phase

import (
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
	"github.com/monstercameron/DungeonFlux/internal/game/rules"
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
	{ID: vocab.StateLobby, Stub: false},
	{ID: vocab.StateCreation, Stub: false},
	{ID: vocab.StateOpening, Stub: false},
	{ID: vocab.StateExploration, Stub: false},
	{ID: vocab.StateConversation, Stub: false},
	{ID: vocab.StateCheck, Stub: false},
	{ID: vocab.StateResolution, Stub: false},
	{ID: vocab.StateHookEvent, Stub: false},
	{ID: vocab.StateCombat, Stub: false},
	{ID: vocab.StateCliffhanger, Stub: false},
	{ID: vocab.StateEnd, Stub: false},
}

// Machine is the pure top-level phase dispatcher.
type Machine struct {
	table            fsm.Machine
	paused           bool
	conversationDone bool
	creation         creation.Machine
	opening          opening.Machine
	conversation     conversation.State
	check            check.Machine
	resolution       resolution.Machine
	hook             hook.Machine
	combat           combat.State
	combatDice       *dice.Roller
	cliffhanger      cliffhanger.Machine
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
	created, err := creation.New([]byte("dungeonflux-phase"))
	if err != nil {
		return Machine{}, err
	}
	return Machine{table: table, creation: created}, nil
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
	if m.State() == vocab.StateCombat {
		return m.stepCombat(event)
	}
	phaseEvent := eventForDomain(m.State(), event)
	if phaseEvent == "" {
		return Result{}, &fsm.Rejection{State: m.State(), Event: event.Kind(), Reason: fsm.ReasonUnknownEvent}
	}
	return m.step(phaseEvent)
}

func (m *Machine) stepCombat(event domain.Event) (Result, error) {
	if action, ok := event.(domain.Act); ok {
		if err := m.applyCombatAction(action); err != nil {
			return Result{}, err
		}
		return Result{Paused: m.paused}, nil
	}
	if _, ok := event.(domain.LineDone); ok && m.combat.Phase != combat.Done {
		if line := event.(domain.LineDone); line.UtteranceID == "" || line.UtteranceID == "combat-outcome" {
			if _, err := m.combat.ResolveEnd(combat.ReasonSkip, 0); err != nil {
				return Result{}, err
			}
			return m.step(eventCliffhanger)
		}
		return Result{}, &fsm.Rejection{State: m.State(), Event: event.Kind(), Reason: fsm.ReasonGuardRejected}
	}
	phaseEvent := eventForDomain(m.State(), event)
	if phaseEvent == "" {
		return Result{}, &fsm.Rejection{State: m.State(), Event: event.Kind(), Reason: fsm.ReasonUnknownEvent}
	}
	return m.step(phaseEvent)
}

func (m *Machine) applyCombatAction(action domain.Act) error {
	if action.Move == vocab.MoveMove {
		_, err := m.combat.Move(combat.Cell{X: action.Cell.C, Y: action.Cell.R})
		return err
	}
	if action.Move == vocab.MoveAttack {
		result, err := m.combat.Attack(m.combatDice, string(action.Target))
		if err != nil {
			return err
		}
		if result.Outcome.HPAfter <= 0 {
			_, err = m.combat.ResolveEnd(combat.ReasonHPZero, result.Seat)
			return err
		}
		m.combat.Phase = combat.PCTurn
		return m.finishCombatTurn()
	}
	if action.Move == vocab.MoveEndTurn {
		return m.finishCombatTurn()
	}
	return &fsm.Rejection{State: m.State(), Event: eventForDomain(m.State(), action), Reason: fsm.ReasonUnknownEvent}
}

func (m *Machine) finishCombatTurn() error {
	if m.combat.Phase != combat.PCTurn {
		return nil
	}
	if err := m.combat.EndPlayerTurn(); err != nil {
		return err
	}
	if m.combat.Phase != combat.EnemyTurn {
		return nil
	}
	if _, err := m.combat.EnemyTurn(m.combatDice, 1200); err != nil {
		return err
	}
	return m.combat.EndEnemyTurn()
}

func (m *Machine) step(event vocab.EventKind) (Result, error) {
	if err := m.route(event); err != nil {
		return Result{}, err
	}
	transition, err := m.table.Step(event)
	if err == nil && transition.To == vocab.StateExploration && transition.From == vocab.StateResolution {
		m.conversationDone = true
	}
	return Result{Transition: transition, Paused: m.paused}, err
}

func (m *Machine) route(event vocab.EventKind) error {
	if event == eventStart {
		return nil
	}
	switch m.State() {
	case vocab.StateCreation:
		if event == eventCreationEnd {
			return nil
		}
	case vocab.StateOpening:
		if m.opening.State() == opening.StateReady {
			m.opening = opening.New()
			m.opening.Enter()
		}
		if event == eventOpeningEnd {
			m.opening.Step(domain.LineDone{})
		}
	case vocab.StateExploration:
		if event == eventTalk {
			m.conversation = conversation.State{Seat: 1}
		}
		if event == eventLeave {
			var err error
			m.hook, err = hook.New(hook.Config{ArrivalClip: "hook-arrival", StrangerUtterance: "stranger", CannedUtterance: "stranger-canned", CannedLine: "canned-stranger"})
			if err != nil {
				return err
			}
			_, err = m.hook.Start()
			return err
		}
	case vocab.StateConversation:
		if event == eventPersuade {
			var err error
			m.check, err = check.New(check.Config{CheckID: "persuasion", Seat: 1, Charisma: 14, Proficient: true, DC: 10}, newDice())
			if err != nil {
				return err
			}
			_, err = m.check.Step(domain.Act{Move: vocab.MovePersuade})
			return err
		}
	case vocab.StateCheck:
		if event == eventRoll {
			if _, err := m.check.Step(domain.TimerFired{Name: "roll_resolved"}); err != nil {
				return err
			}
			outcome, ok := m.check.Outcome()
			if !ok {
				return nil
			}
			var err error
			m.resolution, err = resolution.New(outcome.Success, resolution.Config{SuccessUtterance: "reveal", FailureUtterance: "refuse", SuccessText: "The clue is yours.", FailureText: "She refuses.", SuccessCanned: "canned-reveal", FailureCanned: "canned-refuse"})
			if err != nil {
				return err
			}
			_, err = m.resolution.Start()
			return err
		}
	case vocab.StateResolution:
		if m.resolution.State() != resolution.Done && event == eventResolution {
			_, _ = m.resolution.Step(domain.LineDone{})
		}
	case vocab.StateHookEvent:
		if event == eventCombat {
			_, _ = m.hook.Step(domain.LineDone{})
			var err error
			m.combat, err = combat.New(combatConfig())
			if err != nil {
				return err
			}
			m.combatDice = newDice()
			return m.combat.Start()
		}
	case vocab.StateCombat:
		if event == eventSkip && m.combat.Phase != combat.Done {
			_, err := m.combat.ResolveEnd(combat.ReasonSkip, 0)
			return err
		}
	case vocab.StateCliffhanger:
		if event == eventCliffhanger {
			var err error
			m.cliffhanger, err = cliffhanger.New(cliffhanger.Config{
				LiveClip: domain.Asset{ID: "live-cliffhanger"}, GenericClip: domain.Asset{ID: "generic-cliffhanger"}, AnimatedStill: domain.Asset{ID: "cliffhanger-still"},
				LineID: "cliffhanger", CannedLineID: "cliffhanger-canned", CannedAssetID: "canned-cliffhanger", NarrationInput: "The road continues.",
			})
			if err != nil {
				return err
			}
			if _, err = m.cliffhanger.Enter(); err != nil {
				return err
			}
			_, err = m.cliffhanger.Step(domain.LineDone{UtteranceID: "cliffhanger"})
			return err
		}
	}
	return nil
}

func newDice() *dice.Roller { return dice.New([]byte("dungeonflux-check")) }

func combatConfig() combat.Config {
	return combat.Config{
		PCs: [2]combat.Participant{
			{Seat: 1, ID: "pc-1", Build: combatBuild(rules.Paladin), Position: combat.Cell{X: 1, Y: 0}, HP: 10, MaxHP: 10, AC: 14},
			{Seat: 2, ID: "pc-2", Build: combatBuild(rules.Rogue), Position: combat.Cell{X: 2, Y: 0}, HP: 10, MaxHP: 10, AC: 14},
		},
		Thrall: rules.Thrall("thrall"), Grid: combat.Grid{Cols: 4, Rows: 4},
	}
}

func combatBuild(class rules.Class) rules.Build {
	return rules.Build{Class: class, AttackBonus: 5, HP: 10, MaxHP: 10, AC: 14}
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
