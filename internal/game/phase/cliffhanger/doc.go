// Package cliffhanger owns the terminal cliffhanger phase.
//
// The machine is pure and receives asset and line completion events from the
// room loop. It selects the best available clip without waiting for playback;
// the narration line controls the transition to the end card.
package cliffhanger
