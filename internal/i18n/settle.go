package i18n

// SeatLocales tracks one settled locale per seat plus the room default.
type SeatLocales struct {
	room  string
	seats map[string]string
}

// NewSeatLocales creates a seat locale table with a room default.
func NewSeatLocales(roomDefault string) *SeatLocales {
	return &SeatLocales{room: Settle("", roomDefault), seats: make(map[string]string)}
}

// Settle resolves a client's requested locale, records it for the seat, and
// returns the settled tag.
func (s *SeatLocales) Settle(seat, requested string) string {
	if s == nil {
		return Settle(requested, DefaultLocale)
	}
	tag := Settle(requested, s.room)
	if s.seats == nil {
		s.seats = make(map[string]string)
	}
	s.seats[seat] = tag
	return tag
}

// For returns the settled locale for a seat, or the room default when the
// seat has not joined with a locale yet.
func (s *SeatLocales) For(seat string) string {
	if s == nil {
		return DefaultLocale
	}
	if tag, ok := s.seats[seat]; ok {
		return tag
	}
	return s.room
}

// SetRoom updates the room default used for new seats and unsettled lookups.
func (s *SeatLocales) SetRoom(locale string) string {
	tag := Settle(locale, DefaultLocale)
	if s != nil {
		s.room = tag
	}
	return tag
}

// Room returns the current room default locale.
func (s *SeatLocales) Room() string {
	if s == nil {
		return DefaultLocale
	}
	return s.room
}
