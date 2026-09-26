package creation

import (
	"errors"
	"fmt"

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
	seed    []byte
	classes [seatCount]rules.Class
	seats   [seatCount]SeatState
}

// New creates a creation machine and draws both classes before either player
// taps. The class draw therefore does not depend on tap order.
func New(seed []byte) (Machine, error) {
	if len(seed) == 0 {
		return Machine{}, errors.New("creation seed is empty")
	}
	classRoller := dice.New(seed)
	first, err := rules.DrawClass(classRoller, 1, "")
	if err != nil {
		return Machine{}, fmt.Errorf("draw seat 1 class: %w", err)
	}
	second, err := rules.DrawClass(classRoller, 2, first)
	if err != nil {
		return Machine{}, fmt.Errorf("draw seat 2 class: %w", err)
	}
	return Machine{
		seed:    append([]byte(nil), seed...),
		classes: [seatCount]rules.Class{first, second},
		seats: [seatCount]SeatState{
			{Seat: 1, Class: first},
			{Seat: 2, Class: second},
		},
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
	case domain.PCLocked:
		return m.stepLocked(value)
	case domain.TimerFired:
		if value.Name == creationTimeout {
			return m.stepTimeout()
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
	case vocab.MoveRollHero:
		if seat.Species == "" || seat.Gender == "" {
			return Result{}, errors.New("species and gender are required")
		}
		return m.roll(index)
	default:
		return Result{}, fmt.Errorf("creation move %q is not accepted", event.Move)
	}
	return Result{Accepted: true, Seat: copySeat(*seat)}, nil
}

func (m *Machine) roll(index int) (Result, error) {
	seed := append(append([]byte(nil), m.seed...), byte(index+1))
	build, err := rules.BuildHero(dice.New(seed), m.classes[index], m.seats[index].Species, m.seats[index].Gender)
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
	return Result{Accepted: true, Complete: m.Complete(), Seat: copySeat(*seat)}, nil
}

func (m *Machine) stepTimeout() (Result, error) {
	for i := range m.seats {
		if m.seats[i].Built {
			continue
		}
		m.seats[i].Species = "human"
		m.seats[i].Gender = "nonbinary"
		build, err := rules.BuildHero(dice.New(append(append([]byte(nil), m.seed...), byte(i+1))), m.classes[i], "human", "nonbinary")
		if err != nil {
			return Result{}, fmt.Errorf("default seat %d: %w", i+1, err)
		}
		m.seats[i].Build = build
		m.seats[i].Built = true
		m.seats[i].Locked = true
	}
	return Result{Accepted: true, Complete: m.Complete()}, nil
}

func seatIndex(seat domain.SeatID) (int, bool) {
	if seat < 1 || seat > seatCount {
		return 0, false
	}
	return int(seat - 1), true
}

func copySeat(seat SeatState) SeatState {
	seat.Build.Skills = append([]string(nil), seat.Build.Skills...)
	return seat
}

func background(class rules.Class) string {
	switch class {
	case rules.Paladin:
		return "soldier"
	case rules.Rogue:
		return "criminal"
	default:
		return "acolyte"
	}
}
