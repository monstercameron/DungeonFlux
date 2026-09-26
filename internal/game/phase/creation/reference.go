package creation

import (
	"strings"

	"github.com/monstercameron/DungeonFlux/internal/domain"
)

// referenceEffectFor builds the non-blocking media request emitted when a
// rolled character is locked. Flavor is flattened because the media worker
// only needs a stable descriptive brief, not a second content call.
func referenceEffectFor(seat SeatState) domain.GenerateCharacterReference {
	return domain.GenerateCharacterReference{
		Seat: seat.Seat, Species: seat.Species, Gender: seat.Gender,
		Class: string(seat.Class), Name: seat.Flavor.Name,
		Flavor: strings.TrimSpace(seat.Flavor.Look + " " + seat.Flavor.Hook),
	}
}

// ReferenceSlotName returns the logical slot used for a character reference
// angle or sheet.
func ReferenceSlotName(seat domain.SeatID, angle string) string {
	return "reference:" + seatString(seat) + ":" + angle
}

func seatString(seat domain.SeatID) string {
	if seat < 0 {
		return "0"
	}
	return string(rune('0' + seat))
}
