package creation

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/game/nested"
	"github.com/monstercameron/DungeonFlux/internal/game/rules"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

// FlavorState describes the availability of a seat's character flavor.
type FlavorState string

const (
	// FlavorPending means character_flavor is still being generated.
	FlavorPending FlavorState = "pending"
	// FlavorReady means generated flavor is available.
	FlavorReady FlavorState = "ready"
	// FlavorFallback means the class default flavor is being used.
	FlavorFallback FlavorState = "fallback"
)

// SeatSlots tracks the portrait and flavor work for one seat.
type SeatSlots struct {
	Seat     domain.SeatID
	Class    rules.Class
	Portrait nested.SlotMachine
	Flavor   domain.Flavor
	State    FlavorState
}

// Slots tracks creation assets for both player seats.
type Slots struct {
	seats [2]SeatSlots
}

// NewSlots creates portrait slots with their class-specific template fallback.
func NewSlots(classes [2]rules.Class) (Slots, error) {
	var slots Slots
	for i, class := range classes {
		portrait, err := nested.NewSlot(portraitSlot(domain.SeatID(i+1)), []domain.Asset{fallbackPortrait(class)})
		if err != nil {
			return Slots{}, fmt.Errorf("create portrait slot: %w", err)
		}
		slots.seats[i] = SeatSlots{Seat: domain.SeatID(i + 1), Class: class, Portrait: portrait, State: FlavorPending}
	}
	return slots, nil
}

// Requests returns the parallel runtime requests for portraits and flavor.
func (s Slots) Requests(seats []SeatState) []domain.Effect {
	requests := make([]domain.Effect, 0, len(s.seats)*2)
	for _, seat := range seats {
		if seat.Seat < 1 || seat.Seat > 2 || !seat.Built {
			continue
		}
		requests = append(requests,
			domain.CharacterFlavor{Seat: seat.Seat, Species: seat.Species, Gender: seat.Gender, Class: string(seat.Class), Background: background(seat.Class)},
			domain.GenerateImage{Slot: portraitSlot(seat.Seat), Prompt: portraitPrompt(seat), Size: "1024x1536", Transparent: true, Partials: 2},
		)
	}
	return requests
}

// Seat returns a copy of one seat's slots.
func (s Slots) Seat(seat domain.SeatID) (SeatSlots, bool) {
	index, ok := slotIndex(seat)
	if !ok {
		return SeatSlots{}, false
	}
	out := s.seats[index]
	return out, true
}

// Step applies an asset or flavor callback. Late portrait callbacks after a
// fallback or ready result are ignored by the nested slot machine.
func (s *Slots) Step(event domain.Event) error {
	if s == nil || event == nil {
		return errors.New("creation slots or event is nil")
	}
	switch value := event.(type) {
	case domain.AssetPartial, domain.AssetReady, domain.AssetFailed:
		seat, ok := seatFromSlot(slotName(event))
		if !ok {
			return fmt.Errorf("unknown creation slot %q", slotName(event))
		}
		index, _ := slotIndex(seat)
		return s.seats[index].Portrait.Step(event)
	case domain.TimerFired:
		seat, ok := seatFromDeadline(value.Name)
		if !ok {
			return fmt.Errorf("unknown creation timer %q", value.Name)
		}
		index, _ := slotIndex(seat)
		return s.seats[index].Portrait.Step(value)
	case domain.FlavorDone:
		index, ok := slotIndex(value.Seat)
		if !ok {
			return errors.New("seat must be 1 or 2")
		}
		s.seats[index].Flavor = value.Flavor
		s.seats[index].State = FlavorReady
		return nil
	case domain.FlavorFailed:
		index, ok := slotIndex(value.Seat)
		if !ok {
			return errors.New("seat must be 1 or 2")
		}
		s.seats[index].Flavor = defaultFlavor(s.seats[index].Class)
		s.seats[index].State = FlavorFallback
		return nil
	default:
		return fmt.Errorf("event %q is not a creation slot event", event.Kind())
	}
}

func portraitSlot(seat domain.SeatID) string { return "portrait:" + strconv.Itoa(int(seat)) }

func fallbackPortrait(class rules.Class) domain.Asset {
	return domain.Asset{ID: domain.AssetID("portrait_fallback_" + string(class)), Kind: string(vocab.AssetImage), MIME: "image/png"}
}

func portraitPrompt(seat SeatState) string {
	return fmt.Sprintf("full-body painterly fantasy portrait; %s %s %s; transparent background", seat.Species, seat.Gender, seat.Class)
}

func slotName(event domain.Event) string {
	switch value := event.(type) {
	case domain.AssetPartial:
		return value.Slot
	case domain.AssetReady:
		return value.Slot
	case domain.AssetFailed:
		return value.Slot
	default:
		return ""
	}
}

func seatFromSlot(name string) (domain.SeatID, bool) {
	parts := strings.Split(name, ":")
	if len(parts) != 2 || parts[0] != "portrait" {
		return 0, false
	}
	seat, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, false
	}
	return domain.SeatID(seat), seat >= 1 && seat <= 2
}

func seatFromDeadline(name string) (domain.SeatID, bool) {
	parts := strings.Split(name, ":")
	if len(parts) != 2 || parts[0] != "seat_deadline" {
		return 0, false
	}
	seat, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, false
	}
	return domain.SeatID(seat), seat >= 1 && seat <= 2
}

func slotIndex(seat domain.SeatID) (int, bool) {
	if seat < 1 || seat > 2 {
		return 0, false
	}
	return int(seat - 1), true
}

func defaultFlavor(class rules.Class) domain.Flavor {
	return domain.Flavor{Name: className(class), Look: "A capable adventurer", Hook: "A story still waits to be told."}
}

func className(class rules.Class) string {
	switch class {
	case rules.Paladin:
		return "Paladin"
	case rules.Rogue:
		return "Rogue"
	case rules.Bard:
		return "Bard"
	case rules.Cleric:
		return "Cleric"
	default:
		return "Adventurer"
	}
}
