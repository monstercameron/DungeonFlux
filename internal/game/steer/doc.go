// Package steer selects deterministic, softest-first funnel interventions.
//
// It is deliberately independent of the game and domain packages: the engine
// supplies elapsed beats and the player's observed intent, then records the
// returned rung as a steering note.
package steer
