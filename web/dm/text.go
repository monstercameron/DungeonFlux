package dm

import (
	"strconv"

	"github.com/monstercameron/DungeonFlux/internal/i18n"
)

// catalog is the committed client catalog shared by every DM screen.
var catalog = i18n.Default()

// T renders a catalog key for a DM locale with argument substitution and
// English fallback. Every DM screen renders static text through T.
func T(locale, key string, args map[string]string) string {
	return catalog.T(locale, key, args, 0, "")
}

// localeOrDefault normalizes a render locale for DM views.
func localeOrDefault(locale string) string {
	if locale == "" {
		return i18n.DefaultLocale
	}
	return i18n.Settle(locale, i18n.DefaultLocale)
}

// LobbyTitle returns the localized lobby heading.
func LobbyTitle(locale string) string { return T(locale, "dm.lobby_title", nil) }

// LobbyIntro returns the localized lobby join explanation.
func LobbyIntro(locale string) string { return T(locale, "dm.lobby_intro", nil) }

// RoomCodeLabel returns the localized room-code caption.
func RoomCodeLabel(locale string) string { return T(locale, "dm.room_code", nil) }

// SeatsLabel returns the localized seats section label.
func SeatsLabel(locale string) string { return T(locale, "dm.seats_label", nil) }

// ListenLabel returns the localized listen section heading.
func ListenLabel(locale string) string { return T(locale, "dm.listen", nil) }

// SeatStatus returns the localized lobby seat status.
func SeatStatus(locale string, joined, ready bool) string {
	switch {
	case ready:
		return T(locale, "dm.ready", nil)
	case joined:
		return T(locale, "dm.joined", nil)
	default:
		return T(locale, "dm.waiting_join", nil)
	}
}

// SeatName returns the stable TV label for a seat.
func SeatName(locale string, name string, number int) string {
	if name != "" {
		return name
	}
	return T(locale, "dm.seat", map[string]string{"n": strconv.Itoa(number)})
}

// AudioUnlock returns the localized enable-audio button label.
func AudioUnlock(locale string) string { return T(locale, "dm.audio_unlock", nil) }

// ErrorTitle returns the localized DM error heading.
func ErrorTitle(locale string) string { return T(locale, "dm.error_title", nil) }

// DiceKindLabel returns the localized dice kind name.
func DiceKindLabel(locale, kind string) string {
	switch kind {
	case "attack":
		return T(locale, "dm.kind_attack", nil)
	default:
		return T(locale, "dm.kind_check", nil)
	}
}

// DiceHeading returns the localized dice result heading.
func DiceHeading(locale, kind string) string {
	return T(locale, "dm.dice_label", map[string]string{"kind": DiceKindLabel(locale, kind)})
}

// VsDC returns the localized versus-DC fragment.
func VsDC(locale string, dc int32) string {
	return T(locale, "dm.vs_dc", map[string]string{"dc": strconv.Itoa(int(dc))})
}

// CritLabel returns the localized critical-hit marker.
func CritLabel(locale string) string { return T(locale, "dm.crit", nil) }

// TimerPaused returns the localized paused-timer label.
func TimerPaused(locale string) string { return T(locale, "dm.timer_paused", nil) }

// TimerLabel returns the localized remaining-time label.
func TimerLabel(locale string, ms int64) string {
	return T(locale, "dm.timer_label", map[string]string{"ms": strconv.FormatInt(ms, 10)})
}

// EndTitle returns the localized end-card title.
func EndTitle(locale string) string { return T(locale, "dm.end_title", nil) }

// EndSubtitle returns the localized end-card subtitle.
func EndSubtitle(locale string) string { return T(locale, "dm.end_subtitle", nil) }

// EndRules returns the localized attribution heading.
func EndRules(locale string) string { return T(locale, "dm.end_rules", nil) }
