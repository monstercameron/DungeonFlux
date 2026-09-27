package creation

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/monstercameron/DungeonFlux/internal/content"
	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/game/rules"
	"github.com/monstercameron/DungeonFlux/internal/game/rules/dice"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

const (
	seatCount       = 2
	creationTimeout = "creation_timeout"
)

var speciesOptions = map[string]struct{}{
	"human": {}, "elf": {}, "dwarf": {}, "halfling": {}, "orc": {},
	"tiefling": {}, "dragonborn": {}, "gnome": {}, "goliath": {},
}

var genderOptions = map[string]struct{}{
	"female": {}, "male": {}, "nonbinary": {},
}

// SeatState is the creation state for one player seat.
type SeatState struct {
	Seat    domain.SeatID
	Species string
	Gender  string
	Class   rules.Class
	Build   rules.Build
	Flavor  domain.Flavor
	Built   bool
	Locked  bool
}

// Result reports the accepted event and any work effects it produced.
type Result struct {
	Accepted bool
	Complete bool
	Seat     SeatState
	Effects  []domain.Effect
}

// Machine is the pure character-creation phase machine.
type Machine struct {
	seed  []byte
	seats [seatCount]SeatState
}

// New creates a creation machine. Classes are selected by players; DrawClass
// is used only when the creation timeout supplies a fallback.
func New(seed []byte) (Machine, error) {
	if len(seed) == 0 {
		return Machine{}, errors.New("creation seed is empty")
	}
	return Machine{
		seed:  append([]byte(nil), seed...),
		seats: [seatCount]SeatState{{Seat: 1}, {Seat: 2}},
	}, nil
}

// Seat returns a copy of one seat's current creation state.
func (m Machine) Seat(seat domain.SeatID) (SeatState, bool) {
	index, ok := seatIndex(seat)
	if !ok {
		return SeatState{}, false
	}
	return copySeat(m.seats[index]), true
}

// Seats returns copies of both seat states in seat order.
func (m Machine) Seats() []SeatState {
	out := make([]SeatState, len(m.seats))
	for i := range m.seats {
		out[i] = copySeat(m.seats[i])
	}
	return out
}

// Complete reports whether both seats have emitted pc_locked.
func (m Machine) Complete() bool {
	return m.seats[0].Locked && m.seats[1].Locked
}

// Step applies one creation event. Invalid or out-of-phase events are
// rejected without changing the machine.
func (m *Machine) Step(event domain.Event) (Result, error) {
	if event == nil {
		return Result{}, errors.New("creation event is nil")
	}
	switch value := event.(type) {
	case domain.Act:
		return m.stepAct(value)
	case domain.FlavorDone:
		return m.stepFlavor(value)
	case domain.FlavorFailed:
		return m.stepFlavorFailed(value)
	case domain.PCLocked:
		return m.stepLocked(value)
	case domain.TimerFired:
		if value.Name == creationTimeout {
			return m.stepTimeout()
		}
		if seat, ok := deadlineSeat(value.Name); ok {
			return m.stepSeatTimeout(seat)
		}
	}
	return Result{}, fmt.Errorf("creation event %q is not accepted", event.Kind())
}

func (m *Machine) stepAct(event domain.Act) (Result, error) {
	index, ok := seatIndex(event.Seat)
	if !ok {
		return Result{}, errors.New("seat must be 1 or 2")
	}
	seat := &m.seats[index]
	if event.Move == vocab.MoveRename {
		return m.rename(index, event.Arg)
	}
	if seat.Locked || seat.Built {
		return Result{}, errors.New("seat is already building or locked")
	}
	switch event.Move {
	case vocab.MoveSpecies:
		if _, ok := speciesOptions[event.Arg]; !ok {
			return Result{}, fmt.Errorf("unknown species %q", event.Arg)
		}
		seat.Species = event.Arg
	case vocab.MoveGender:
		if _, ok := genderOptions[event.Arg]; !ok {
			return Result{}, fmt.Errorf("unknown gender %q", event.Arg)
		}
		seat.Gender = event.Arg
	case vocab.MoveClass:
		class := rules.Class(event.Arg)
		if !rules.IsClass(class) {
			return Result{}, fmt.Errorf("unknown class %q", event.Arg)
		}
		seat.Class = class
	case vocab.MoveRollHero:
		if seat.Species == "" || seat.Gender == "" || seat.Class == "" {
			return Result{}, errors.New("species, gender, and class are required")
		}
		return m.roll(index)
	default:
		return Result{}, fmt.Errorf("creation move %q is not accepted", event.Move)
	}
	return Result{Accepted: true, Seat: copySeat(*seat)}, nil
}

func (m *Machine) roll(index int) (Result, error) {
	seed := append(append([]byte(nil), m.seed...), byte(index+1))
	build, err := rules.BuildHero(dice.New(seed), m.seats[index].Class, m.seats[index].Species, m.seats[index].Gender)
	if err != nil {
		return Result{}, fmt.Errorf("build seat %d: %w", index+1, err)
	}
	seat := &m.seats[index]
	seat.Build = build
	seat.Built = true
	effect := domain.CharacterFlavor{
		Seat: seat.Seat, Species: seat.Species, Gender: seat.Gender,
		Class: string(seat.Class), Background: background(seat.Class),
	}
	return Result{Accepted: true, Seat: copySeat(*seat), Effects: []domain.Effect{effect}}, nil
}

func (m *Machine) stepFlavor(event domain.FlavorDone) (Result, error) {
	index, ok := seatIndex(event.Seat)
	if !ok {
		return Result{}, errors.New("seat must be 1 or 2")
	}
	seat := &m.seats[index]
	if !seat.Built || seat.Locked {
		return Result{}, errors.New("flavor is not expected")
	}
	seat.Flavor = event.Flavor
	return Result{Accepted: true, Seat: copySeat(*seat)}, nil
}

// stepFlavorFailed applies the deterministic, curated fallback name (and a
// generic look and hook) for a seat whose character_flavor call failed, so
// the demo never shows a bare seat label like "Hero 1". It is idempotent: a
// seat whose flavor already arrived, or that is not yet built, is untouched.
func (m *Machine) stepFlavorFailed(event domain.FlavorFailed) (Result, error) {
	index, ok := seatIndex(event.Seat)
	if !ok {
		return Result{}, errors.New("seat must be 1 or 2")
	}
	seat := &m.seats[index]
	if !seat.Built || seat.Locked {
		return Result{Accepted: true, Seat: copySeat(*seat)}, nil
	}
	if strings.TrimSpace(seat.Flavor.Name) == "" {
		seat.Flavor = fallbackFlavor(*seat, m.seed, index)
	}
	return Result{Accepted: true, Seat: copySeat(*seat)}, nil
}

// rename applies a player-submitted hero name, valid from roll_hero (Built)
// until the seat locks. It is rejected outside that window, on an unknown
// seat, or when the text fails validation.
func (m *Machine) rename(index int, raw string) (Result, error) {
	seat := &m.seats[index]
	if !seat.Built {
		return Result{}, errors.New("roll a hero before renaming it")
	}
	if seat.Locked {
		return Result{}, errors.New("seat is already locked")
	}
	name, err := SanitizeHeroName(raw)
	if err != nil {
		return Result{}, fmt.Errorf("rename seat %d: %w", index+1, err)
	}
	seat.Flavor.Name = name
	return Result{Accepted: true, Seat: copySeat(*seat)}, nil
}

// fallbackFlavor builds a deterministic fallback flavor for a seat, using the
// curated per-species/gender hero name table (content.FallbackHeroName)
// seeded from the run seed and seat index, so the same run always yields the
// same fallback name.
func fallbackFlavor(seat SeatState, seed []byte, index int) domain.Flavor {
	nameSeed := append(append([]byte(nil), seed...), byte(index+1), 'n', 'a', 'm', 'e')
	name := content.FallbackHeroName(seat.Species, seat.Gender, nameSeed)
	return domain.Flavor{Name: name, Look: "A capable adventurer.", Hook: "A story still waits to be told."}
}

func (m *Machine) stepLocked(event domain.PCLocked) (Result, error) {
	index, ok := seatIndex(event.Seat)
	if !ok {
		return Result{}, errors.New("seat must be 1 or 2")
	}
	seat := &m.seats[index]
	if !seat.Built {
		return Result{}, errors.New("seat has no rolled build")
	}
	if seat.Locked {
		return Result{Accepted: true, Complete: m.Complete(), Seat: copySeat(*seat)}, nil
	}
	seat.Locked = true
	return Result{Accepted: true, Complete: m.Complete(), Seat: copySeat(*seat), Effects: []domain.Effect{referenceEffectFor(*seat)}}, nil
}

func (m *Machine) stepTimeout() (Result, error) {
	var effects []domain.Effect
	for i := range m.seats {
		result, err := m.stepSeatTimeout(m.seats[i].Seat)
		if err != nil {
			return Result{}, err
		}
		effects = append(effects, result.Effects...)
	}
	return Result{Accepted: true, Complete: m.Complete(), Effects: effects}, nil
}

func (m *Machine) stepSeatTimeout(seat domain.SeatID) (Result, error) {
	index, ok := seatIndex(seat)
	if !ok {
		return Result{}, errors.New("seat must be 1 or 2")
	}
	if m.seats[index].Locked {
		return Result{Accepted: true, Complete: m.Complete(), Seat: copySeat(m.seats[index])}, nil
	}
	if !m.seats[index].Built {
		classes, err := m.fallbackClasses()
		if err != nil {
			return Result{}, err
		}
		if m.seats[index].Class == "" {
			m.seats[index].Class = classes[index]
		}
		if m.seats[index].Species == "" {
			m.seats[index].Species = "human"
		}
		if m.seats[index].Gender == "" {
			m.seats[index].Gender = "nonbinary"
		}
		build, err := rules.BuildHero(dice.New(append(append([]byte(nil), m.seed...), byte(index+1))), m.seats[index].Class, m.seats[index].Species, m.seats[index].Gender)
		if err != nil {
			return Result{}, fmt.Errorf("default seat %d: %w", index+1, err)
		}
		m.seats[index].Build = build
		m.seats[index].Built = true
	}
	if strings.TrimSpace(m.seats[index].Flavor.Name) == "" {
		m.seats[index].Flavor = fallbackFlavor(m.seats[index], m.seed, index)
	}
	return m.stepLocked(domain.PCLocked{Seat: seat})
}

func (m Machine) fallbackClasses() ([seatCount]rules.Class, error) {
	roller := dice.New(append(append([]byte(nil), m.seed...), 0))
	first, err := rules.DrawClass(roller, 1, "")
	if err != nil {
		return [seatCount]rules.Class{}, fmt.Errorf("draw fallback seat 1 class: %w", err)
	}
	second, err := rules.DrawClass(roller, 2, first)
	if err != nil {
		return [seatCount]rules.Class{}, fmt.Errorf("draw fallback seat 2 class: %w", err)
	}
	return [seatCount]rules.Class{first, second}, nil
}

func seatIndex(seat domain.SeatID) (int, bool) {
	if seat < 1 || seat > seatCount {
		return 0, false
	}
	return int(seat - 1), true
}

func deadlineSeat(name string) (domain.SeatID, bool) {
	parts := strings.Split(name, ":")
	if len(parts) != 2 || parts[0] != "seat_deadline" {
		return 0, false
	}
	seat, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, false
	}
	return domain.SeatID(seat), seat >= 1 && seat <= seatCount
}

func copySeat(seat SeatState) SeatState {
	seat.Build.Skills = append([]string(nil), seat.Build.Skills...)
	seat.Build.SaveProficiencies = append([]string(nil), seat.Build.SaveProficiencies...)
	if seat.Build.SkillProficiencies != nil {
		seat.Build.SkillProficiencies = mapsClone(seat.Build.SkillProficiencies)
	}
	return seat
}

func mapsClone(values map[string]string) map[string]string {
	clone := make(map[string]string, len(values))
	for key, value := range values {
		clone[key] = value
	}
	return clone
}

func background(class rules.Class) string {
	switch class {
	case rules.Barbarian:
		return "soldier"
	case rules.Paladin:
		return "soldier"
	case rules.Fighter:
		return "soldier"
	case rules.Ranger:
		return "guide"
	case rules.Rogue:
		return "criminal"
	case rules.Monk:
		return "sage"
	case rules.Sorcerer, rules.Warlock, rules.Wizard:
		return "sage"
	default:
		return "acolyte"
	}
}
