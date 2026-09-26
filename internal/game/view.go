package game

import "github.com/monstercameron/DungeonFlux/internal/domain"

// ViewForDM returns the engine view visible to the DM screen.
//
// The DM sees the complete lobby, including every seat. The returned value is
// detached from the input so a client cannot mutate engine-owned slices.
func ViewForDM(view domain.View) domain.View {
	return detachedView(view)
}

// ViewForSeat returns the engine view visible to one phone seat.
//
// A phone receives only its own seat card. Shared room metadata remains
// visible, while the returned view is detached from the input.
func ViewForSeat(view domain.View, seat domain.SeatID) domain.View {
	out := detachedView(view)
	out.Seats = seatViewsFor(out.Seats, seat)
	return out
}

// ViewForHost returns the engine view visible to the host screen.
//
// The host sees the complete lobby and both seat cards. The returned value is
// detached from the input so it is safe to hand to an asynchronous client.
func ViewForHost(view domain.View) domain.View {
	return detachedView(view)
}

func detachedView(view domain.View) domain.View {
	out := view.DeepCopy()
	out.Seats = cloneSeatViews(out.Seats)
	return out
}

func cloneSeatViews(seats []domain.SeatView) []domain.SeatView {
	out := make([]domain.SeatView, len(seats))
	copy(out, seats)
	for index := range out {
		out[index].Character = cloneCharacter(out[index].Character)
		out[index].Build = cloneBuild(out[index].Build)
		out[index].Moves = append([]domain.MoveView(nil), seats[index].Moves...)
		for moveIndex := range out[index].Moves {
			out[index].Moves[moveIndex].Options = append([]domain.OptionView(nil), seats[index].Moves[moveIndex].Options...)
		}
	}
	return out
}

func cloneCharacter(character *domain.Character) *domain.Character {
	if character == nil {
		return nil
	}
	copyCharacter := *character
	return &copyCharacter
}

func cloneBuild(build *domain.BuildCard) *domain.BuildCard {
	if build == nil {
		return nil
	}
	copyBuild := *build
	return &copyBuild
}

func seatViewsFor(seats []domain.SeatView, wanted domain.SeatID) []domain.SeatView {
	for _, seat := range seats {
		if seat.Seat == wanted {
			return []domain.SeatView{seat}
		}
	}
	return nil
}
