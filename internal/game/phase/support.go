package phase

import (
	"fmt"

	"github.com/monstercameron/DungeonFlux/internal/core/fsm"
	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/game/combat"
	"github.com/monstercameron/DungeonFlux/internal/game/phase/cliffhanger"
	"github.com/monstercameron/DungeonFlux/internal/game/phase/creation"
	"github.com/monstercameron/DungeonFlux/internal/game/phase/hook"
	"github.com/monstercameron/DungeonFlux/internal/game/rules"
	"github.com/monstercameron/DungeonFlux/internal/game/rules/dice"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

// hookArrivalUtterance names the arrival clip and its canned stand-in line.
const hookArrivalUtterance = "hook-arrival"

func (m *Machine) stepHook(event domain.Event) (Result, error) {
	if isPassive(event) {
		return Result{}, nil
	}
	if line, ok := event.(domain.LineDone); ok && m.hook.State() == hook.ArrivalClip {
		if line.UtteranceID != "" && line.UtteranceID != hookArrivalUtterance {
			return Result{}, nil
		}
		// The arrival clip finished: start the stranger's line. Its effects
		// used to be discarded and the same event re-fed as the stranger's
		// line_done, so the hook either skipped the line or waited forever.
		clip, err := m.hook.Step(domain.ClipDone{AssetID: hookArrivalUtterance})
		if err != nil {
			return Result{}, err
		}
		return Result{Effects: clip.Effects}, nil
	}
	result, err := m.hook.Step(event)
	if err != nil {
		return Result{}, err
	}
	if !result.Combat {
		return Result{Effects: result.Effects}, nil
	}
	return m.transition(eventCombat, result.Effects)
}

func (m *Machine) stepCombat(event domain.Event) (Result, error) {
	if timer, ok := event.(domain.TimerFired); ok && timer.Name == combatDeadlineTimer {
		return m.expireCombatDeadline()
	}
	if result, handled, err := m.stepKillcam(event); handled {
		return result, err
	}
	if result, handled, err := m.stepCombatDash(event); handled {
		return result, err
	}
	if action, ok := event.(domain.Act); ok {
		effects, err := m.applyCombatAction(action)
		if err != nil {
			return Result{}, err
		}
		if m.combat.Phase == combat.Done && m.killcam.URL == "" {
			m.syncCombatSeats()
			return m.transition(eventCliffhanger, effects)
		}
		return Result{Effects: effects}, nil
	}
	if line, ok := event.(domain.LineDone); ok && (line.UtteranceID == "" || line.UtteranceID == "combat-outcome") {
		var effects []domain.Effect
		if m.combat.Phase != combat.Done {
			end, err := m.combat.ResolveEnd(combat.ReasonSkip, 0)
			if err != nil {
				return Result{}, err
			}
			if end.Outcome == combat.Fled {
				effects = combat.FledAudio()
			}
		}
		m.syncCombatSeats()
		return m.transition(eventCliffhanger, effects)
	}
	return m.passive(event)
}

func (m *Machine) applyCombatAction(action domain.Act) ([]domain.Effect, error) {
	if effects, handled, err := m.applyCombatMove(action); handled {
		return effects, err
	}
	if action.Move == vocab.MoveAttack {
		result, err := m.combat.Attack(m.combatDice, string(action.Target))
		if err != nil {
			return nil, err
		}
		effects := combat.AttackAudio(result)
		resolved := combat.AttackResolvedMS(result)
		if result.Outcome.HPAfter <= 0 {
			end, endErr := m.combat.ResolveEnd(combat.ReasonHPZero, result.Seat)
			if endErr != nil {
				return nil, endErr
			}
			if end.Outcome == combat.Slain {
				effects = append(effects, combat.VictoryAudio(end.SlainBySeat, resolved)...)
				effects = append(effects, m.beginKillcam(result.Seat, "victory")...)
			}
			return effects, nil
		}
		m.combat.Phase = combat.PCTurn
		enemyEffects, err := m.finishCombatTurn(resolved)
		return append(effects, enemyEffects...), err
	}
	if action.Move == vocab.MoveEndTurn {
		return m.finishCombatTurn(0)
	}
	return nil, fmt.Errorf("combat move %q is not accepted", action.Move)
}

// finishCombatTurn ends the active PC's turn and, when the thrall acts next,
// plays its turn at once. startMS is when the ending turn finishes playing on
// the TV, so the thrall's sounds and the next seat's turn chime follow it.
func (m *Machine) finishCombatTurn(startMS int) ([]domain.Effect, error) {
	if m.combat.Phase != combat.PCTurn {
		return nil, nil
	}
	if err := m.combat.EndPlayerTurn(); err != nil {
		return nil, err
	}
	if m.combat.Phase != combat.EnemyTurn {
		return combat.TurnAudio(m.combat.TurnSeat, startMS), nil
	}
	result, err := m.combat.EnemyTurn(m.combatDice, 1200)
	if err != nil {
		return nil, err
	}
	effects := combat.EnemyAudio(result, startMS)
	if result.Outcome.Hit && result.Outcome.HPBefore > 0 && result.Outcome.HPAfter <= 0 {
		effects = append(effects, m.beginKillcam(result.TargetSeat, "defeat")...)
	}
	if err := m.combat.EndEnemyTurn(); err != nil {
		return nil, err
	}
	if m.combat.Phase == combat.PCTurn {
		effects = append(effects, combat.TurnAudio(m.combat.TurnSeat, combat.EnemyResolvedMS(result, startMS))...)
	}
	return effects, nil
}

func (m *Machine) stepCliffhanger(event domain.Event) (Result, error) {
	if line, ok := event.(domain.LineDone); ok && line.UtteranceID == "" {
		line.UtteranceID = "cliffhanger"
		event = line
	}
	result, err := m.cliffhanger.Step(event)
	if err != nil {
		return Result{}, err
	}
	if result.EndCard {
		return m.transition(eventCliffhanger, result.Effects)
	}
	return Result{Effects: result.Effects}, nil
}

func (m *Machine) step(event vocab.EventKind) (Result, error) { return m.transition(event, nil) }

func (m *Machine) transition(event vocab.EventKind, effects []domain.Effect) (Result, error) {
	transition, err := m.table.Step(event)
	if err != nil {
		return Result{}, err
	}
	out := Result{Transition: transition, Effects: append([]domain.Effect(nil), effects...), Paused: m.paused}
	if transition.From == vocab.StateCombat && transition.To != vocab.StateCombat {
		out.Effects = append(out.Effects, m.leaveCombatTimers()...)
	}
	if transition.From == vocab.StateResolution && transition.To == vocab.StateExploration {
		m.conversationDone = true
		if m.timersEnabled {
			out.Effects = append(out.Effects, domain.StartTimer{
				Name: idleHookTimerName, After: idleHookDelay, Pausable: true,
				Scope: domain.Scope{Machine: vocab.MachineSession},
			})
		}
	}
	if event == eventStart && m.timersEnabled {
		out.Effects = append(out.Effects, domain.StartTimer{Name: "creation_timeout", After: 30e9, Pausable: true, Scope: domain.Scope{Machine: vocab.MachineSession}})
	}
	if transition.From == vocab.StateCreation && transition.To != vocab.StateCreation {
		out.Effects = append(out.Effects, domain.CancelTimer{Name: "creation_timeout"})
	}
	if transition.To == vocab.StateOpening {
		out.Effects = append(out.Effects, m.opening.Enter().Effects...)
	}
	if transition.To == vocab.StateCombat {
		if err := m.startCombat(); err != nil {
			return Result{}, err
		}
		out.Effects = append(out.Effects, combatDeadline())
	}
	if transition.To == vocab.StateCliffhanger {
		started, err := m.startCliffhanger()
		if err != nil {
			return Result{}, err
		}
		out.Effects = append(out.Effects, started...)
	}
	return out, nil
}

func (m *Machine) beginHook(seat domain.SeatID) (Result, error) {
	if seat == 1 || seat == 2 {
		m.spotlight = seat
	}
	started, err := m.startHook()
	if err != nil {
		return Result{}, err
	}
	result, err := m.step(eventLeave)
	if err != nil {
		return Result{}, err
	}
	result.Effects = append(started, result.Effects...)
	if m.conversationDone {
		result.Effects = append(result.Effects, domain.CancelTimer{Name: idleHookTimerName})
	}
	return result, nil
}

func (m *Machine) startHook() ([]domain.Effect, error) {
	var err error
	m.hook, err = hook.New(hook.Config{ArrivalClip: "hook-arrival", StrangerUtterance: "stranger", StrangerText: "It followed me from the river.", CannedUtterance: "stranger-canned", CannedLine: "canned-stranger"})
	if err != nil {
		return nil, err
	}
	started, err := m.hook.Start()
	if err != nil {
		return nil, err
	}
	return started.Effects, nil
}

func (m *Machine) startCombat() error {
	m.killcam = domain.KillCamView{}
	var err error
	config := partyCombatConfig(contentCombatConfig(m.oneShot.Encounter.Battlefield), m.creation.Seats())
	m.combat, err = combat.New(config)
	if err != nil {
		return err
	}
	m.combatDice = newDice()
	if m.forcedD20 != 0 {
		if err := m.combatDice.ForceD20(m.forcedD20); err != nil {
			return err
		}
		m.forcedD20 = 0
	}
	return m.combat.Start()
}

func (m *Machine) syncCombatSeats() {
	for index, participant := range m.combat.PCs {
		if index >= len(m.seats) {
			continue
		}
		seat := &m.seats[index]
		if seat.Character != nil {
			seat.Character.HP, seat.Character.MaxHP, seat.Character.AC = participant.HP, participant.MaxHP, participant.AC
		}
		if seat.Build != nil && seat.Build.Stats != nil {
			seat.Build.Stats.HP, seat.Build.Stats.MaxHP, seat.Build.Stats.AC = participant.HP, participant.MaxHP, participant.AC
		}
	}
}

func (m *Machine) startCliffhanger() ([]domain.Effect, error) {
	var err error
	m.cliffhanger, err = cliffhanger.New(cliffhanger.Config{LiveClip: domain.Asset{ID: "live-cliffhanger"}, GenericClip: domain.Asset{ID: "generic-cliffhanger"}, AnimatedStill: domain.Asset{ID: "cliffhanger-still"}, LineID: "cliffhanger", CannedLineID: "cliffhanger-canned", CannedAssetID: "canned-cliffhanger", NarrationInput: "The road continues."})
	if err != nil {
		return nil, err
	}
	result, err := m.cliffhanger.Enter()
	if err != nil {
		return nil, err
	}
	return result.Effects, nil
}

func (m *Machine) unhandled(event domain.Event) (Result, error) {
	return Result{}, &fsm.Rejection{State: m.State(), Event: event.Kind(), Reason: fsm.ReasonUnknownEvent}
}
func (m *Machine) passive(event domain.Event) (Result, error) {
	if isPassive(event) {
		if m.State() == vocab.StateLobby && !m.lobbyAudioSent {
			m.lobbyAudioSent = true
			return Result{Effects: lobbyAudioEffects(true)}, nil
		}
		return Result{}, nil
	}
	return m.unhandled(event)
}

func isPassive(event domain.Event) bool {
	switch event.(type) {
	case domain.Join, domain.Say, domain.TalkStart, domain.TalkEnd, domain.StreamClosed, domain.Report, domain.STTError, domain.FlavorFailed, domain.NarrationDelta, domain.AssetPartial, domain.AssetReady, domain.AssetFailed, domain.PrerenderTextDone, domain.PrerenderDone, domain.PrerenderFailed:
		return true
	default:
		return false
	}
}

func (m *Machine) updateCreationSeat(state creation.SeatState) {
	if state.Seat < 1 || state.Seat > domain.SeatID(len(m.seats)) {
		return
	}
	m.seats[state.Seat-1] = creationSeatView(state)
}

func initialSeats() []domain.SeatView {
	return []domain.SeatView{{Seat: 1, PlayerNumber: 1}, {Seat: 2, PlayerNumber: 2}}
}
func newDice() *dice.Roller { return dice.New([]byte("dungeonflux-check")) }

func combatConfig() combat.Config {
	return combat.Config{PCs: [2]combat.Participant{{Seat: 1, ID: "pc-1", Build: combatBuild(rules.Paladin), Position: combat.Cell{X: 1, Y: 0}, HP: 10, MaxHP: 10, AC: 14}, {Seat: 2, ID: "pc-2", Build: combatBuild(rules.Rogue), Position: combat.Cell{X: 2, Y: 0}, HP: 10, MaxHP: 10, AC: 14}}, Thrall: rules.Thrall("thrall"), Grid: combat.Grid{Cols: 4, Rows: 4}}
}

func combatBuild(class rules.Class) rules.Build {
	return rules.Build{Class: class, AttackBonus: 5, HP: 10, MaxHP: 10, AC: 14}
}

func definition() fsm.Def {
	states := make([]fsm.State, 0, len(phaseDefinitions))
	for _, phase := range phaseDefinitions {
		states = append(states, phase.ID)
	}
	transitions := []fsm.Transition{{From: vocab.StateLobby, Event: eventStart, To: vocab.StateCreation}, {From: vocab.StateCreation, Event: eventCreationEnd, To: vocab.StateOpening}, {From: vocab.StateOpening, Event: eventOpeningEnd, To: vocab.StateExploration}, {From: vocab.StateExploration, Event: eventTalk, To: vocab.StateConversation}, {From: vocab.StateExploration, Event: eventLeave, To: vocab.StateHookEvent}, {From: vocab.StateConversation, Event: eventPersuade, To: vocab.StateCheck}, {From: vocab.StateConversation, Event: eventStepAway, To: vocab.StateExploration}, {From: vocab.StateCheck, Event: eventRoll, To: vocab.StateResolution}, {From: vocab.StateResolution, Event: eventResolution, To: vocab.StateExploration}, {From: vocab.StateHookEvent, Event: eventCombat, To: vocab.StateCombat}, {From: vocab.StateCombat, Event: eventCliffhanger, To: vocab.StateCliffhanger}, {From: vocab.StateCliffhanger, Event: eventCliffhanger, To: vocab.StateEnd}}
	for index, phase := range phaseDefinitions {
		transitions = append(transitions, fsm.Transition{From: phase.ID, Event: eventSkip, To: skipTarget(index)}, fsm.Transition{From: phase.ID, Event: eventReset, To: vocab.StateLobby})
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

func eventForHost(command vocab.HostCmd) vocab.EventKind { return vocab.EventKind(string(command)) }
