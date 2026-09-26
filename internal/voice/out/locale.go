package out

import (
	"context"
	"io"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/i18n"
)

func (e *PCMExecutor) roomLocale() string {
	if e == nil || e.Room == nil {
		return i18n.DefaultLocale
	}
	return i18n.Settle(e.Room(), "")
}

// voiceFor resolves the TTS voice for a line. A per-locale override wins,
// then the engine voice, then the locale default voice.
func (e *PCMExecutor) voiceFor(voice string) string {
	locale := e.roomLocale()
	if e != nil && e.Voices != nil {
		if override, ok := e.Voices[locale]; ok && override != "" {
			return override
		}
	}
	if voice != "" {
		return voice
	}
	return i18n.TTSVoice(locale)
}

func (e *CannedExecutor) roomLocale() string {
	if e == nil || e.Room == nil {
		return i18n.DefaultLocale
	}
	return i18n.Settle(e.Room(), "")
}

// openLocalized opens the per-locale canned asset first and falls back to
// the base asset when no localized recording exists yet.
func (e *CannedExecutor) openLocalized(ctx context.Context, id domain.AssetID) (io.ReadCloser, error) {
	locale := e.roomLocale()
	if locale != i18n.DefaultLocale {
		if reader, err := e.assets.Open(ctx, domain.AssetID(i18n.CannedAssetID(string(id), locale))); err == nil {
			return reader, nil
		}
	}
	return e.assets.Open(ctx, id)
}
