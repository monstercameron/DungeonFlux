package host

import "github.com/monstercameron/DungeonFlux/internal/i18n"

// RoomLocaleSelector is the host page's room-language control: the ordered
// options plus the selected room default, rendered through T.
type RoomLocaleSelector struct {
	Selected string
	Options  []string
}

// NewRoomLocaleSelector builds a selector settled on the room default.
func NewRoomLocaleSelector(roomDefault string) RoomLocaleSelector {
	options := make([]string, len(i18n.SupportedLocales))
	copy(options, i18n.SupportedLocales)
	return RoomLocaleSelector{Selected: i18n.Settle(roomDefault, ""), Options: options}
}

// Select changes the room default, falling back for unsupported tags.
func (s *RoomLocaleSelector) Select(locale string) string {
	if s == nil {
		return i18n.DefaultLocale
	}
	s.Selected = i18n.Settle(locale, s.Selected)
	return s.Selected
}

// Label returns the localized selector caption for a render locale.
func (s RoomLocaleSelector) Label(renderLocale string) string {
	return RoomLocaleLabel(renderLocale)
}

// OptionLabel returns the display name of a locale option.
func OptionLabel(locale string) string {
	switch i18n.Normalize(locale) {
	case "es":
		return "Español"
	default:
		return "English"
	}
}
