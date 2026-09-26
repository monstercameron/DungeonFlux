package creation

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/game/rules"
)

// DebugPatch applies the seat fields accepted by the local debug surface and
// rebuilds a deterministic rules character when identity fields change.
func (m *Machine) DebugPatch(seat domain.SeatID, fields map[string]string) (SeatState, error) {
	if m == nil {
		return SeatState{}, errors.New("creation machine is nil")
	}
	index, ok := seatIndex(seat)
	if !ok {
		return SeatState{}, errors.New("seat must be 1 or 2")
	}
	current := &m.seats[index]
	rebuild := false
	for field, value := range fields {
		switch field {
		case "name":
			if strings.TrimSpace(value) == "" {
				return SeatState{}, errors.New("seat name is required")
			}
			current.Flavor.Name = value
		case "species":
			if _, ok := speciesOptions[value]; !ok {
				return SeatState{}, fmt.Errorf("unknown species %q", value)
			}
			current.Species, rebuild = value, true
		case "gender":
			if _, ok := genderOptions[value]; !ok {
				return SeatState{}, fmt.Errorf("unknown gender %q", value)
			}
			current.Gender, rebuild = value, true
		case "class":
			class := rules.Class(value)
			if !rules.IsClass(class) {
				return SeatState{}, fmt.Errorf("unknown class %q", value)
			}
			current.Class, rebuild = class, true
		case "hp":
			hp, err := strconv.Atoi(value)
			if err != nil || hp < 0 || hp > current.Build.MaxHP {
				return SeatState{}, errors.New("seat HP is outside the rolled range")
			}
			if !current.Built {
				return SeatState{}, errors.New("seat must be rolled before HP is patched")
			}
			current.Build.HP = hp
		default:
			return SeatState{}, fmt.Errorf("field %q is not patchable during creation", field)
		}
	}
	if rebuild {
		if current.Species == "" || current.Gender == "" || current.Class == "" {
			return SeatState{}, errors.New("species, gender, and class are required")
		}
		if _, err := m.roll(index); err != nil {
			return SeatState{}, err
		}
	}
	return copySeat(*current), nil
}
