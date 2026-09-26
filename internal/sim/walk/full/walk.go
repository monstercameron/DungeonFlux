package full

import (
	"time"

	"github.com/monstercameron/DungeonFlux/internal/game/combat"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

const maxWalkTime = 5 * time.Minute

func withinBudget(now time.Duration, entries map[vocab.StateID]int) bool {
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

func isOutcome(result combat.EndResult, outcome combat.Outcome, reason combat.EndReason) bool {
	return result.Outcome == outcome && result.Reason == reason
}
