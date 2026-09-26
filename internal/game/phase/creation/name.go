package creation

import (
	"strconv"
	"strings"

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
