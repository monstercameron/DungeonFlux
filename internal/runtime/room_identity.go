package runtime

import "github.com/monstercameron/DungeonFlux/internal/domain"

// Accepted joins must enter the room registry as well as the current engine.
// Otherwise NewRun replaces the only record of the connected player's name.
func (r *Room) retainIdentity(event domain.Event, ack *domain.Ack) {
	if r.state == nil || (ack != nil && !ack.Accepted) {
		return
	}
	join, ok := event.(domain.Join)
	if !ok || join.Seat < 1 || join.Seat > 2 {
		return
	}
	r.state.retainJoin(join)
}

func (r *RoomState) retainJoin(join domain.Join) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.seats == nil {
		r.seats = make(map[domain.SeatID]domain.Seat)
	}
	if r.names == nil {
		r.names = make(map[domain.SeatID]string)
	}
	if r.locales == nil {
		r.locales = make(map[domain.SeatID]string)
	}
	seat := r.seats[join.Seat]
	seat.ID, seat.PlayerNumber = join.Seat, int(join.Seat)
	seat.Connected, seat.Locale = true, join.Locale
	r.seats[join.Seat] = seat
	r.names[join.Seat], r.locales[join.Seat] = join.Name, join.Locale
}
