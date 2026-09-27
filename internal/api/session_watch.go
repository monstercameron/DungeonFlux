package api

import "github.com/monstercameron/DungeonFlux/internal/domain"

// WatchSeat reports the seat a phone token already holds, without posting a
// join. A watch subscription only needs to know who is watching; the seat was
// announced when the phone joined, and nothing marks a seat disconnected, so a
// resubscribe after an idle or dropped stream must not re-announce it (each
// announcement steps the engine and redraws every screen).
func (s *SessionServer) WatchSeat(token string) (domain.SeatID, bool) {
	if s == nil || token == "" {
		return 0, false
	}
	seat, ok := s.seatForToken(token)
	if !ok {
		return 0, false
	}
	return seat.id, true
}
