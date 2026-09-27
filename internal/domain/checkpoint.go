package domain

import "github.com/monstercameron/DungeonFlux/internal/vocab"

// DebugCheckpoint requests a named in-memory save or load through the engine.
type DebugCheckpoint struct {
	Operation string `json:"operation"`
	Name      string `json:"name"`
}

func (DebugCheckpoint) sealedEvent() {}

// Kind returns the checkpoint command's event-log kind.
func (DebugCheckpoint) Kind() vocab.EventKind { return "debug_checkpoint" }

// Checkpoint applies a validated checkpoint command synchronously in the room.
type Checkpoint struct {
	Operation string
	Name      string
}

func (Checkpoint) sealedEffect() {}

// Kind returns the synchronous checkpoint control-effect kind.
func (Checkpoint) Kind() vocab.EffectKind { return "checkpoint" }
