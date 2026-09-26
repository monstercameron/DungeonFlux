package in

import (
	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/i18n"
)

func (e *Transcriber) localeFor(seat domain.SeatID) string {
	if e == nil || e.Locales == nil {
		return i18n.DefaultLocale
	}
	return i18n.Settle(e.Locales(seat), "")
}

func sttLanguage(locale string) string {
	return i18n.STTLanguage(locale)
}
