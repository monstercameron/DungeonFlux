package creation

import (
	"errors"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/monstercameron/DungeonFlux/internal/domain"
)

// DisplayName returns the player's name, or the stable seat fallback when no
// name was supplied by the lobby.
func DisplayName(joined string, seat domain.SeatID) string {
	if name := strings.TrimSpace(joined); name != "" {
		return name
	}
	return "Hero " + strconv.Itoa(int(seat))
}

// MaxHeroNameLength is the longest hero name the engine accepts from a
// rename move.
const MaxHeroNameLength = 24

// SanitizeHeroName trims a player-submitted hero name, strips control
// characters and markup-like punctuation, and enforces a 1-24 character
// length. It returns an error when the cleaned result is empty or too long.
func SanitizeHeroName(raw string) (string, error) {
	name := stripControlAndMarkup(strings.TrimSpace(raw))
	name = strings.Join(strings.Fields(name), " ")
	if name == "" {
		return "", errors.New("hero name is empty")
	}
	if utf8.RuneCountInString(name) > MaxHeroNameLength {
		return "", errors.New("hero name is too long")
	}
	return name, nil
}

// stripControlAndMarkup drops control characters and the small set of
// punctuation that could be used to inject markup or break rendering (angle
// brackets, braces, and backticks). Ordinary punctuation such as apostrophes
// and hyphens, common in hero names, is kept.
func stripControlAndMarkup(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch {
		case r < 0x20 || r == 0x7f:
			continue
		case r == '<' || r == '>' || r == '{' || r == '}' || r == '`' || r == '\\':
			continue
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}
