package combatsim

import (
	"errors"
	"time"

	"github.com/monstercameron/DungeonFlux/internal/game/combat"
	"github.com/monstercameron/DungeonFlux/internal/game/rules/dice"
)

const combatCap time.Duration = 30 * 1_000_000_000

// ActionKind identifies one scripted simulator input.
type ActionKind string

const (
	// Start begins the combat intro.
	Start ActionKind = "start"
	// Move moves the active PC to Cell.
	Move ActionKind = "move"
	// Attack resolves an attack against Target.
	Attack ActionKind = "attack"
	// Bell requests a flee through the bell.
	Bell ActionKind = "bell"
	// Skip requests a host skip.
	Skip ActionKind = "skip"
	// Advance moves virtual time forward by Delta.
	Advance ActionKind = "advance"
	// Pause freezes the combat clock.
	Pause ActionKind = "pause"
	// Resume restarts the combat clock.
	Resume ActionKind = "resume"
	// ForceD20 forces the next attack d20 to Value.
	ForceD20 ActionKind = "force_d20"
)

// Action is one deterministic simulator input.
type Action struct {
	Kind   ActionKind
	Cell   combat.Cell
	Target string
	Delta  time.Duration
	Value  int
}

// Instance drives one combat state with virtual time and seeded dice.
type Instance struct {
	State   combat.State
	Dice    *dice.Roller
	Now     time.Duration
	Paused  bool
	Outcome combat.EndResult
}

// New creates a simulator instance in intro.
func New(config combat.Config, seed []byte) (Instance, error) {
	state, err := combat.NewState(config)
	if err != nil {
		return Instance{}, err
	}
	return Instance{State: state, Dice: dice.New(seed)}, nil
}

// Run applies a scripted sequence until it is exhausted or combat ends.
func (i *Instance) Run(actions ...Action) error {
	if i == nil {
		return errors.New("combat simulator is nil")
	}
	for _, action := range actions {
		if err := i.Step(action); err != nil {
			return err
		}
		if i.State.Phase == combat.Done {
			return nil
		}
	}
	return nil
}

// Step applies one scripted input.
func (i *Instance) Step(action Action) error {
	if i == nil || i.Dice == nil {
		return errors.New("combat simulator is nil")
	}
	switch action.Kind {
	case Start:
		return i.State.Start()
	case Move:
		_, err := i.State.Move(action.Cell)
		return err
	case Attack:
		return i.attack(action.Target)
	case Bell:
		return i.finish(combat.ReasonBell, 0)
	case Skip:
		return i.finish(combat.ReasonSkip, 0)
	case Advance:
		return i.advance(action.Delta)
	case Pause:
		i.Paused = true
		return nil
	case Resume:
		i.Paused = false
		return nil
	case ForceD20:
		return i.Dice.ForceD20(action.Value)
	default:
		return errors.New("unknown combat simulator action")
	}
}

func (i *Instance) attack(target string) error {
	if i.State.Phase != combat.PCTurn {
		return errors.New("attack is not available")
	}
	seat := i.State.TurnSeat
	result, err := i.State.Attack(i.Dice, target)
	if err != nil {
		return err
	}
	if result.Outcome.HPAfter <= 0 {
		return i.finish(combat.ReasonHPZero, seat)
	}
	if i.State.Phase == combat.Rolling {
		i.State.Phase = combat.PCTurn
	}
	if err := i.State.EndPlayerTurn(); err != nil {
		return err
	}
	if i.State.Phase == combat.EnemyTurn {
		if _, err := i.State.EnemyTurn(i.Dice, 1200); err != nil {
			return err
		}
		if err := i.State.EndEnemyTurn(); err != nil {
			return err
		}
	}
	return nil
}

func (i *Instance) advance(delta time.Duration) error {
	if delta < 0 {
		return errors.New("virtual time cannot move backwards")
	}
	if !i.Paused {
		i.Now += delta
	}
	if i.Now >= combatCap && i.State.Phase != combat.Done {
		return i.finish(combat.ReasonCap, 0)
	}
	return nil
}

func (i *Instance) finish(reason combat.EndReason, seat int) error {
	if i.State.Phase == combat.Done {
		return nil
	}
	result, err := i.State.ResolveEnd(reason, seat)
	if err != nil {
		return err
	}
	i.Outcome = result
	return nil
}
