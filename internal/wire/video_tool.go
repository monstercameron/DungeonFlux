package wire

import (
	"github.com/monstercameron/DungeonFlux/internal/adapters/video/fal"
	"github.com/monstercameron/DungeonFlux/internal/ports"
)

// KillcamVideoModel identifies the reference-driven cinematic model.
const KillcamVideoModel = fal.ReferenceModel

// NewKillcamVideo composes the build-time video vendor behind its shared port.
func NewKillcamVideo(key string) ports.ReferenceVideoGen {
	return fal.NewReference(key, KillcamVideoModel, "", nil)
}
