package basic

import "github.com/monstercameron/DungeonFlux/internal/vocab"

// expectedSkipStates returns the phase sequence produced by one host Skip
// from each state, including the Resolution-to-Exploration branch.
func expectedSkipStates() []vocab.StateID {
	return []vocab.StateID{
		vocab.StateCreation, vocab.StateOpening, vocab.StateExploration,
		vocab.StateConversation, vocab.StateCheck, vocab.StateResolution,
		vocab.StateExploration, vocab.StateHookEvent, vocab.StateCombat,
		vocab.StateCliffhanger, vocab.StateEnd,
	}
}
