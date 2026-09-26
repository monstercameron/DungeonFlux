// Package phone contains the player-facing phone view models and components.
package phone

import (
	"context"
	"errors"
	"strings"

	df "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
)

// ActResult is the result of an asynchronous session Act call.
type ActResult struct {
	Value *df.ActResponse
	Err   error
}

// ActClient is the smallest client surface required by phone screens.
type ActClient interface {
	Act(context.Context, *df.ActRequest) <-chan ActResult
}

// CreationPhase identifies the visible stage of character creation.
type CreationPhase string

const (
	// CreationPicking means the player is choosing species and gender.
	CreationPicking CreationPhase = "picking"
	// CreationRolling means the server is rolling the build.
	CreationRolling CreationPhase = "rolling"
	// CreationLocked means the character was submitted to the engine.
	CreationLocked CreationPhase = "pc_locked"
	// CreationFailed means the last action was rejected.
	CreationFailed CreationPhase = "failed"
)

// CreationSnapshot is the render-safe state of the creation screen.
type CreationSnapshot struct {
	SeatToken    string
	PlayerNumber int32
	Species      string
	Gender       string
	Build        *df.BuildCard
	Phase        CreationPhase
	Error        string
	Locale       string
}

// CreationOption is one choice shown by a character-creation picker.
type CreationOption struct {
	ID    string
	Label string
}

var creationSpecies = []CreationOption{
	{ID: "human", Label: "Human"}, {ID: "elf", Label: "Elf"}, {ID: "dwarf", Label: "Dwarf"},
	{ID: "halfling", Label: "Halfling"}, {ID: "orc", Label: "Orc"}, {ID: "tiefling", Label: "Tiefling"},
	{ID: "dragonborn", Label: "Dragonborn"}, {ID: "gnome", Label: "Gnome"}, {ID: "goliath", Label: "Goliath"},
}

var creationGenders = []CreationOption{
	{ID: "female", Label: "Female"}, {ID: "male", Label: "Male"}, {ID: "nonbinary", Label: "Nonbinary"},
}

// CreationSpecies returns the legal species choices in display order.
func CreationSpecies() []CreationOption { return append([]CreationOption(nil), creationSpecies...) }

// CreationGenders returns the legal gender choices in display order.
func CreationGenders() []CreationOption { return append([]CreationOption(nil), creationGenders...) }

// CreationModel owns creation selections and Act request construction.
type CreationModel struct {
	client ActClient
	state  CreationSnapshot
}

// NewCreationModel creates a phone creation model for one seat.
func NewCreationModel(client ActClient, seatToken string, playerNumber int32) *CreationModel {
	return &CreationModel{client: client, state: CreationSnapshot{
		SeatToken: strings.TrimSpace(seatToken), PlayerNumber: playerNumber, Phase: CreationPicking,
	}}
}

// Snapshot returns a copy of the current render state.
func (m *CreationModel) Snapshot() CreationSnapshot {
	if m == nil {
		return CreationSnapshot{Phase: CreationFailed, Error: "creation model is unavailable"}
	}
	state := m.state
	if state.Build != nil {
		state.Build = &df.BuildCard{
			PlayerNumber: state.Build.GetPlayerNumber(), Name: state.Build.GetName(),
			ClassName: state.Build.GetClassName(), PortraitUrl: state.Build.GetPortraitUrl(),
		}
	}
	return state
}

// SelectSpecies records a legal species selection.
func (m *CreationModel) SelectSpecies(species string) error {
	return m.selectValue(&m.state.Species, species, speciesValues, "species")
}

// SelectGender records a legal gender selection.
func (m *CreationModel) SelectGender(gender string) error {
	return m.selectValue(&m.state.Gender, gender, genderValues, "gender")
}

// RollHero starts the server-authoritative hero roll.
func (m *CreationModel) RollHero(ctx context.Context) <-chan ActResult {
	return m.send(ctx, "roll_hero", "")
}

// Lock submits the completed character to the engine.
func (m *CreationModel) Lock(ctx context.Context) <-chan ActResult {
	return m.send(ctx, "pc_locked", "")
}

// ApplyAct updates the model after an Act response arrives.
func (m *CreationModel) ApplyAct(result ActResult) CreationSnapshot {
	if m == nil {
		return CreationSnapshot{Phase: CreationFailed, Error: "creation model is unavailable"}
	}
	if result.Err != nil {
		m.state.Phase, m.state.Error = CreationFailed, result.Err.Error()
		return m.Snapshot()
	}
	if result.Value != nil && !result.Value.GetAccepted() {
		m.state.Phase, m.state.Error = CreationFailed, result.Value.GetReason()
		if m.state.Error == "" {
			m.state.Error = "server rejected character action"
		}
		return m.Snapshot()
	}
	m.state.Error = ""
	return m.Snapshot()
}

// ApplyScreenState projects the latest server view onto the creation card.
func (m *CreationModel) ApplyScreenState(state *df.ScreenState) CreationSnapshot {
	if m == nil {
		return CreationSnapshot{Phase: CreationFailed, Error: "creation model is unavailable"}
	}
	phone := state.GetPhone()
	if phone == nil {
		return m.Snapshot()
	}
	if character := phone.GetCharacter(); character != nil {
		m.state.Build = &df.BuildCard{Name: character.GetName(), ClassName: character.GetClassName(), PortraitUrl: character.GetPortraitUrl(), PlayerNumber: m.state.PlayerNumber}
	}
	if strings.Contains(strings.ToLower(state.GetPhase()), "creation") && m.state.Build != nil {
		m.state.Phase = CreationRolling
	}
	m.state.Locale = phoneLocale(phone)
	return m.Snapshot()
}

var speciesValues = map[string]struct{}{"human": {}, "elf": {}, "dwarf": {}, "halfling": {}, "orc": {}, "tiefling": {}, "dragonborn": {}, "gnome": {}, "goliath": {}}
var genderValues = map[string]struct{}{"female": {}, "male": {}, "nonbinary": {}}

func (m *CreationModel) selectValue(destination *string, value string, allowed map[string]struct{}, label string) error {
	if m == nil {
		return errors.New("creation model is unavailable")
	}
	value = strings.ToLower(strings.TrimSpace(value))
	if _, ok := allowed[value]; !ok {
		return errors.New("unknown " + label + " " + value)
	}
	*destination, m.state.Error = value, ""
	return nil
}

func (m *CreationModel) send(ctx context.Context, move, arg string) <-chan ActResult {
	result := make(chan ActResult, 1)
	if m == nil || m.client == nil {
		result <- ActResult{Err: errors.New("creation client is unavailable")}
		return result
	}
	if move == "roll_hero" {
		if m.state.Species == "" || m.state.Gender == "" {
			result <- ActResult{Err: errors.New("species and gender are required")}
			return result
		}
		m.state.Phase = CreationRolling
	}
	if move == "pc_locked" {
		m.state.Phase = CreationLocked
	}
	request := &df.ActRequest{SeatToken: m.state.SeatToken, MoveId: move, Arg: arg}
	return m.client.Act(ctx, request)
}
