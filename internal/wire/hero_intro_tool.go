package wire

import (
	"time"

	imageopenai "github.com/monstercameron/DungeonFlux/internal/adapters/image/openai"
	"github.com/monstercameron/DungeonFlux/internal/adapters/video/fal"
)

// NewHeroIntroImage composes the reference-image vendor for build-time portraits.
func NewHeroIntroImage(key, endpoint string) *imageopenai.Adapter {
	return imageopenai.New(key, endpoint, 5*time.Minute, nil)
}

// NewHeroIntroVideo composes the frame-pinned build-time animation vendor.
func NewHeroIntroVideo(key, model string) *fal.Adapter {
	return fal.New(key, model, "", nil)
}
