package llmexec

import (
	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/i18n"
	"github.com/monstercameron/DungeonFlux/internal/ports"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

// LocaleSource resolves the output language for model requests. Room
// supplies the room default (DM narration) and Seat supplies per-seat
// locales; both are nil-safe and fall back to English. Composition code
// wires them to the room state; the zero value keeps English behavior.
type LocaleSource struct {
	Room func() string
	Seat func(domain.SeatID) string
}

// ForRoom returns the settled room locale.
func (s LocaleSource) ForRoom() string {
	if s.Room == nil {
		return i18n.DefaultLocale
	}
	return i18n.Settle(s.Room(), "")
}

// ForSeat returns the settled locale for a seat, falling back to the room.
func (s LocaleSource) ForSeat(seat domain.SeatID) string {
	room := s.ForRoom()
	if s.Seat == nil {
		return room
	}
	return i18n.Settle(s.Seat(seat), room)
}

// WithLocale stamps a text request with its output locale and appends the
// language instruction to the system message. English requests are
// byte-identical apart from the settled Meta.Locale.
func WithLocale(request ports.TextRequest, locale string) ports.TextRequest {
	tag := i18n.Settle(locale, "")
	request.Meta.Locale = tag
	instruction := i18n.PromptLanguage(tag)
	if instruction == "" {
		return request
	}
	for index, message := range request.Messages {
		if message.Role == vocab.MsgSystem {
			request.Messages[index].Text = i18n.WithPromptLanguage(message.Text, tag)
			return request
		}
	}
	return request
}
