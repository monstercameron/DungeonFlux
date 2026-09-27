package story

import (
	"time"

	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

const maxWalkTime = 5 * time.Minute

func withinWalkBudget(now time.Duration, entries map[vocab.StateID]int) bool {
	if now > maxWalkTime {
		return false
	}
	for _, count := range entries {
		if count > 3 {
			return false
		}
	}
	return true
}
