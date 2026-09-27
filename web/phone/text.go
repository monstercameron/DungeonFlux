package phone

import (
	"strconv"
	"strings"

	df "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
	"github.com/monstercameron/DungeonFlux/internal/i18n"
)

// catalog is the committed client catalog shared by every phone screen.
var catalog = i18n.Default()

// localeOverride is a client-only display-language preference set from the
// phone's Menu tab. It changes only how this device renders catalog text;
// the seat's server-side locale (which selects narration and voice content)
// is unaffected and keeps coming from the server on every phone view.
var localeOverride string

// SetLocaleOverride forces every phone screen on this device to render in
// locale until cleared with an empty string.
func SetLocaleOverride(locale string) {
	if strings.TrimSpace(locale) == "" {
		localeOverride = ""
		return
	}
	localeOverride = i18n.Settle(locale, "")
}

// LocaleOverride returns the current display-language override, or "" when
// the phone is following the server's seat locale.
func LocaleOverride() string { return localeOverride }

// phoneLocale resolves the render locale for a phone view: the device's own
// override when the player set one, otherwise the server's seat locale,
// defaulting to English when neither is set.
func phoneLocale(phone *df.PhoneView) string {
	if localeOverride != "" {
		return localeOverride
	}
	if phone == nil {
		return i18n.DefaultLocale
	}
	return i18n.Settle(phone.GetLocale(), i18n.DefaultLocale)
}

// T renders a catalog key for a phone locale with argument substitution and
// English fallback. Every phone screen renders static text through T.
func T(locale, key string, args map[string]string) string {
	return catalog.T(locale, key, args, 0, "")
}

// RenderMsg renders a server key-plus-arguments message for a phone locale.
// The server fallback covers keys the client catalog does not know yet.
func RenderMsg(locale string, msg *df.Text, fallback string) string {
	if msg == nil {
		return fallback
	}
	if msg.GetKey() == "" {
		if msg.GetFallback() != "" {
			return msg.GetFallback()
		}
		return fallback
	}
	return catalog.T(locale, msg.GetKey(), msg.GetArgs(), int(msg.GetCount()), msg.GetFallback())
}

// MovesTitle returns the localized action-menu heading.
func MovesTitle(locale string) string { return T(locale, "moves.title", nil) }

// CreateTitle returns the localized character-creation heading.
func CreateTitle(locale string) string { return T(locale, "create.title", nil) }

// CreateHint returns the localized creation hint shown before a build exists.
func CreateHint(locale string) string { return T(locale, "create.hint", nil) }

// SpeciesLabel returns the localized human species button label.
func SpeciesLabel(locale string) string { return T(locale, "create.species", nil) }

// GenderLabel returns the localized nonbinary gender button label.
func GenderLabel(locale string) string { return T(locale, "create.gender", nil) }

// RollHeroLabel returns the localized hero-roll button label.
func RollHeroLabel(locale string) string { return T(locale, "move.roll_hero", nil) }

// DiceTitle returns the localized persuasion-roll heading.
func DiceTitle(locale string) string { return T(locale, "dice.title", nil) }

// DiceResult returns the localized roll status line for a dice phase.
func DiceResult(locale string, phase DicePhase, d20 int32, outcome string) string {
	switch phase {
	case DiceRolling:
		return T(locale, "dice.rolling", nil)
	case DiceResolved:
		return T(locale, "dice.result", map[string]string{"d20": strconv.Itoa(int(d20)), "outcome": outcome})
	default:
		return T(locale, "dice.roll", nil)
	}
}

// DiceCheckLabel returns the localized persuasion check description.
func DiceCheckLabel(locale string, modifier, dc int32) string {
	sign := "+"
	if modifier < 0 {
		sign = "-"
		modifier = -modifier
	}
	return T(locale, "dice.label", map[string]string{
		"modifier": sign + strconv.Itoa(int(modifier)),
		"dc":       strconv.Itoa(int(dc)),
	})
}

// DiceButton returns the localized roll button label.
func DiceButton(locale string) string { return T(locale, "dice.button", nil) }

// TypedLabel returns the localized typed-input label.
func TypedLabel(locale string) string { return T(locale, "typed.label", nil) }

// TypedHint returns the localized typed-input placeholder.
func TypedHint(locale string) string { return T(locale, "ui.typed.hint", nil) }

// TypedSend returns the localized send button label.
func TypedSend(locale string) string { return T(locale, "typed.send", nil) }

// TypedSent returns the localized message-sent status.
func TypedSent(locale string) string { return T(locale, "typed.sent", nil) }

// SheetName returns the localized sheet heading, falling back to the
// character name and class when set.
func SheetName(locale, name, class string) string {
	if name == "" && class == "" {
		return T(locale, "sheet.yours", nil)
	}
	if name == "" {
		return class
	}
	if class == "" {
		return name
	}
	return name + " · " + class
}

// SheetHP returns the localized hit-point line.
func SheetHP(locale string, hp, max int32) string {
	if max <= 0 {
		return T(locale, "sheet.hp_none", nil)
	}
	return T(locale, "sheet.hp_short", map[string]string{"hp": strconv.Itoa(int(hp)), "max": strconv.Itoa(int(max))})
}

// SheetConditions returns the localized conditions line.
func SheetConditions(locale string, conditions []string) string {
	if len(conditions) == 0 {
		return T(locale, "sheet.no_conditions", nil)
	}
	return T(locale, "sheet.conditions", map[string]string{"first": conditions[0]})
}

// PTTStart returns the localized start-talking button label.
func PTTStart(locale string) string { return T(locale, "ptt.start", nil) }

// PTTStop returns the localized stop-talking button label.
func PTTStop(locale string) string { return T(locale, "ptt.stop", nil) }

// PTTReady returns the localized ready-to-talk status.
func PTTReady(locale string) string { return T(locale, "ptt.ready", nil) }

// PTTNoRecord returns the localized no-recording status.
func PTTNoRecord(locale string) string { return T(locale, "ptt.norecord", nil) }

// CombatTurnTitle returns the localized combat turn heading.
func CombatTurnTitle(locale string) string { return T(locale, "ui.combat.your_turn", nil) }

// ErrorTitle returns the localized phone error heading.
func ErrorTitle(locale string) string { return T(locale, "phone.error_title", nil) }

// ClientUnavailable returns the localized unavailable-client message.
func ClientUnavailable(locale string) string { return T(locale, "phone.client_unavail", nil) }

// ConnectionLabel returns the short, localized connection state for the frame.
func ConnectionLabel(locale string, state ConnectionState) string {
	if strings.HasPrefix(strings.ToLower(locale), "es") {
		switch state {
		case ConnectionOnline:
			return "Conectado"
		case ConnectionConnecting:
			return "Conectando…"
		default:
			return "Sin conexión"
		}
	}
	switch state {
	case ConnectionOnline:
		return "Connected"
	case ConnectionConnecting:
		return "Connecting…"
	default:
		return "Offline"
	}
}
