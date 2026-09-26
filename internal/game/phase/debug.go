package phase

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/game/combat"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

// DebugGoto constructs a deterministic phase fixture and runs its normal
// entry effects. It is intended for the debug event surface; production
// transitions continue to use the ordinary player events.
func (m *Machine) DebugGoto(target vocab.StateID) (Result, error) {
	if m == nil {
		return Result{}, errors.New("phase machine is nil")
	}
	if !knownPhase(target) {
		return Result{}, fmt.Errorf("unknown debug phase %q", target)
	}
	if m.State() != vocab.StateLobby {
		return Result{}, errors.New("debug goto requires a fresh phase machine")
	}
	var out Result
	step := func(event domain.Event) error {
		result, err := m.Step(event)
		if err != nil {
			return err
		}
		out.Transition = result.Transition
		out.Paused = result.Paused
		out.Effects = append(out.Effects, result.Effects...)
		return nil
	}
	if target == vocab.StateLobby {
		return Result{Effects: lobbyAudioEffects(true)}, nil
	}
	if err := step(domain.HostCmd{Cmd: vocab.HostStart}); err != nil {
		return Result{}, err
	}
	if target == vocab.StateCreation {
		return out, nil
	}
	if err := m.debugBuildHeroes(step); err != nil {
		return Result{}, err
	}
	if target == vocab.StateOpening {
		return out, nil
	}
	if err := step(domain.LineDone{UtteranceID: "opening"}); err != nil {
		return Result{}, err
	}
	if target == vocab.StateExploration {
		return out, nil
	}
	if err := step(domain.Act{Seat: 1, Move: vocab.MoveTalkVell}); err != nil {
		return Result{}, err
	}
	if target == vocab.StateConversation {
		return out, nil
	}
	if err := step(domain.Act{Seat: 1, Move: vocab.MovePersuade}); err != nil {
		return Result{}, err
	}
	if target == vocab.StateCheck {
		return out, nil
	}
	if err := step(domain.TimerFired{Name: "roll_resolved"}); err != nil {
		return Result{}, err
	}
	if target == vocab.StateResolution {
		return out, nil
	}
	if err := step(domain.LineDone{UtteranceID: "reveal"}); err != nil {
		return Result{}, err
	}
	if err := step(domain.Act{Seat: 1, Move: vocab.MoveLeave}); err != nil {
		return Result{}, err
	}
	if target == vocab.StateHookEvent {
		return out, nil
	}
	if err := step(domain.LineDone{UtteranceID: hookArrivalUtterance}); err != nil {
		return Result{}, err
	}
	if err := step(domain.LineDone{UtteranceID: "stranger"}); err != nil {
		return Result{}, err
	}
	if target == vocab.StateCombat {
		return out, nil
	}
	if err := step(domain.HostCmd{Cmd: vocab.HostSkip}); err != nil {
		return Result{}, err
	}
	if target == vocab.StateCliffhanger {
		return out, nil
	}
	if err := step(domain.LineDone{UtteranceID: "cliffhanger"}); err != nil {
		return Result{}, err
	}
	return out, nil
}

func (m *Machine) debugBuildHeroes(step func(domain.Event) error) error {
	classes := []string{"paladin", "rogue"}
	for seat, class := range classes {
		id := domain.SeatID(seat + 1)
		for _, event := range []domain.Event{
			domain.Act{Seat: id, Move: vocab.MoveSpecies, Arg: "human"},
			domain.Act{Seat: id, Move: vocab.MoveGender, Arg: "nonbinary"},
			domain.Act{Seat: id, Move: vocab.MoveClass, Arg: class},
			domain.Act{Seat: id, Move: vocab.MoveRollHero},
			domain.Act{Seat: id, Move: vocab.MoveReady},
		} {
			if err := step(event); err != nil {
				return fmt.Errorf("build debug seat %d: %w", id, err)
			}
		}
	}
	return nil
}

func knownPhase(target vocab.StateID) bool {
	for _, definition := range phaseDefinitions {
		if definition.ID == target {
			return true
		}
	}
	return false
}

// DebugPatch applies the validated debug whitelist to a seat or combat token.
// It returns transition effects when a combat outcome ends the encounter.
func (m *Machine) DebugPatch(target domain.EntityID, fields map[string]string) (Result, error) {
	if m == nil {
		return Result{}, errors.New("phase machine is nil")
	}
	if len(fields) == 0 {
		return Result{}, errors.New("debug patch has no fields")
	}
	if strings.HasPrefix(string(target), "seat:") {
		return m.patchSeat(target, fields)
	}
	if strings.HasPrefix(string(target), "token:") {
		return m.patchToken(target, fields)
	}
	return Result{}, fmt.Errorf("unknown debug target %q", target)
}

func (m *Machine) patchSeat(target domain.EntityID, fields map[string]string) (Result, error) {
	seat, err := parseDebugNumber(string(target), "seat:", 1, 2)
	if err != nil {
		return Result{}, err
	}
	if m.State() == vocab.StateCreation {
		updated, err := m.creation.DebugPatch(domain.SeatID(seat), fields)
		if err != nil {
			return Result{}, err
		}
		m.updateCreationSeat(updated)
		return Result{}, nil
	}
	for field, value := range fields {
		switch field {
		case "name":
			if strings.TrimSpace(value) == "" {
				return Result{}, errors.New("seat name is required")
			}
			m.seats[seat-1].PlayerName = value
			if m.seats[seat-1].Character != nil {
				m.seats[seat-1].Character.Name = value
			}
			if m.seats[seat-1].Build != nil {
				m.seats[seat-1].Build.Name = value
			}
		default:
			return Result{}, fmt.Errorf("field %q is not patchable in %s", field, m.State())
		}
	}
	return Result{}, nil
}

func (m *Machine) patchToken(target domain.EntityID, fields map[string]string) (Result, error) {
	if m.State() != vocab.StateCombat {
		return Result{}, errors.New("combat token patch requires combat")
	}
	name := strings.TrimPrefix(string(target), "token:")
	if name == "pc1" {
		name = "pc-1"
	}
	if name == "pc2" {
		name = "pc-2"
	}
	if name == m.combat.Thrall.ID {
		return m.patchThrall(fields)
	}
	for _, participant := range m.combat.PCs {
		if participant.ID == name {
			return m.patchParticipant(participant, fields)
		}
	}
	return Result{}, fmt.Errorf("unknown combat token %q", target)
}

func (m *Machine) patchParticipant(participant combat.Participant, fields map[string]string) (Result, error) {
	for field, value := range fields {
		switch field {
		case "hp":
			hp, err := parseNonNegative(value)
			if err != nil || hp > participant.MaxHP {
				return Result{}, errors.New("combat HP is outside token range")
			}
			participant.HP = hp
		case "cell":
			cell, err := parseCell(value)
			if err != nil || !m.combat.Grid.IsWalkable(cell) {
				return Result{}, errors.New("combat cell is not walkable")
			}
			participant.Position = cell
		default:
			return Result{}, fmt.Errorf("field %q is not patchable on a combat token", field)
		}
	}
	if err := m.combat.SetParticipant(participant); err != nil {
		return Result{}, err
	}
	return Result{}, nil
}

func (m *Machine) patchThrall(fields map[string]string) (Result, error) {
	for field, value := range fields {
		switch field {
		case "hp":
			hp, err := parseNonNegative(value)
			if err != nil || hp > m.combat.Thrall.MaxHP {
				return Result{}, errors.New("combat HP is outside token range")
			}
			m.combat.Thrall.HP = hp
		case "cell":
			cell, err := parseCell(value)
			if err != nil || !m.combat.Grid.IsWalkable(cell) {
				return Result{}, errors.New("combat cell is not walkable")
			}
			m.combat.ThrallPosition = cell
		case "outcome":
			return m.patchOutcome(value)
		default:
			return Result{}, fmt.Errorf("field %q is not patchable on the thrall", field)
		}
	}
	return Result{}, nil
}

func (m *Machine) patchOutcome(value string) (Result, error) {
	reason := combat.ReasonBell
	if value == "slain" {
		reason = combat.ReasonHPZero
	} else if value != "fled" {
		return Result{}, errors.New("combat outcome must be slain or fled")
	}
	if _, err := m.combat.ResolveEnd(reason, 0); err != nil {
		return Result{}, err
	}
	return m.transition(eventCliffhanger, nil)
}

func parseDebugNumber(value, prefix string, min, max int) (int, error) {
	number, err := strconv.Atoi(strings.TrimPrefix(value, prefix))
	if err != nil || number < min || number > max || !strings.HasPrefix(value, prefix) {
		return 0, fmt.Errorf("invalid debug target %q", value)
	}
	return number, nil
}

func parseNonNegative(value string) (int, error) {
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 0 {
		return 0, errors.New("value must be non-negative")
	}
	return parsed, nil
}

func parseCell(value string) (combat.Cell, error) {
	parts := strings.Split(value, ",")
	if len(parts) != 2 {
		return combat.Cell{}, errors.New("cell must be c,r")
	}
	column, columnErr := strconv.Atoi(parts[0])
	row, rowErr := strconv.Atoi(parts[1])
	if columnErr != nil || rowErr != nil {
		return combat.Cell{}, errors.New("cell must be c,r")
	}
	return combat.Cell{X: column, Y: row}, nil
}
