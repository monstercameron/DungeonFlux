package domain

import (
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

// GenerateCharacterReference asks media workers to create a four-angle
// turnaround for a locked player character. Scope is carried with the request
// so the work can be cancelled with the run.
type GenerateCharacterReference struct {
	Seat    SeatID `json:"seat"`
	Species string `json:"species"`
	Gender  string `json:"gender"`
	Class   string `json:"class"`
	Name    string `json:"name"`
	Flavor  string `json:"flavor"`
	Scope   Scope  `json:"scope"`
}

func (GenerateCharacterReference) sealedEffect() {}

// Kind identifies this effect for runtime dispatch and event logging.
func (GenerateCharacterReference) Kind() vocab.EffectKind {
	return vocab.EffectGenerateCharacterReference
}
