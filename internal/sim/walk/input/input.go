package input

import (
	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/game/phase"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

func step(machine *phase.Machine, event domain.Event) error {
	_, err := machine.Step(event)
	return err
}

func walkToConversation(machine *phase.Machine) error {
	steps := []domain.Event{
		domain.HostCmd{Cmd: vocab.HostStart},
		domain.PCLocked{Seat: 1},
		domain.LineDone{},
		domain.Act{Seat: 1, Move: vocab.MoveTalkVell},
	}
	for _, event := range steps {
		if err := step(machine, event); err != nil {
			return err
		}
	}
	return nil
}
