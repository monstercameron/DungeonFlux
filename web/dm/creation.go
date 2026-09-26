package dm

import (
	"strconv"
	"strings"

	dungeonfluxv1 "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
)

// CreationSeat is the TV projection for one character-creation seat.
type CreationSeat struct {
	Number        int32
	Name          string
	Species       string
	Gender        string
	Class         string
	ClassCrestURL string
	PortraitURL   string
	Status        string
	Ready         bool
}

// CreationModel contains the two seats and the shared creation prompt.
type CreationModel struct {
	Seats  [2]CreationSeat
	Prompt string
}

// featuredCreationSeat chooses the seat with the most useful live preview.
// The first seat remains the deterministic fallback while both players are
// still choosing on their phones.
func featuredCreationSeat(model CreationModel) CreationSeat {
	for _, seat := range model.Seats {
		if seat.Ready || seat.PortraitURL != "" || seat.Species != "" || seat.Gender != "" || seat.Class != "" {
			return seat
		}
	}
	return model.Seats[0]
}

func creationDisplayValue(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

// CreationModelFromView projects the current DM creation view.
//
// BuildCard is the wire-level result of a roll. Until the shared wire contract
// carries in-progress picks, the engine may publish a compact callout such as
// "creation seat=1 species=elf gender=female class=rogue"; DecodeCreationCallout accepts
// that additive transition format without making the renderer depend on it.
func CreationModelFromView(view *dungeonfluxv1.DMView) CreationModel {
	model := CreationModel{Prompt: "Players: choose a species, gender, and class on your phones, then roll your hero."}
	model.Seats[0] = newCreationSeat(1)
	model.Seats[1] = newCreationSeat(2)
	if view == nil {
		return model
	}
	for _, card := range view.GetBuildCards() {
		if card == nil || card.GetPlayerNumber() < 1 || card.GetPlayerNumber() > 2 {
			continue
		}
		seat := &model.Seats[card.GetPlayerNumber()-1]
		seat.Name, seat.Class, seat.PortraitURL = card.GetName(), card.GetClassName(), card.GetPortraitUrl()
		if seat.Name != "" || seat.Class != "" || seat.PortraitURL != "" {
			seat.Status, seat.Ready = "Hero ready", true
		}
	}
	if update, ok := DecodeCreationCallout(view.GetCallout()); ok {
		seat := &model.Seats[update.Number-1]
		seat.Species, seat.Gender = update.Species, update.Gender
		if update.Class != "" {
			seat.Class = update.Class
		}
		if update.ClassCrestURL != "" {
			seat.ClassCrestURL = update.ClassCrestURL
		}
		if !seat.Ready {
			seat.Status = creationPickStatus(*seat)
		}
	}
	return model
}

func newCreationSeat(number int32) CreationSeat {
	return CreationSeat{Number: number, Status: "Waiting for player"}
}

func creationPickStatus(seat CreationSeat) string {
	if seat.Species != "" && seat.Gender != "" && seat.Class != "" {
		return "Ready to roll"
	}
	if seat.Species != "" || seat.Gender != "" || seat.Class != "" {
		return "Choice received"
	}
	return "Waiting for player"
}

func speciesArtName(species string) string {
	return "ui/species_" + strings.ToLower(strings.TrimSpace(species))
}

func classArtName(className string) string {
	return "ui/class_" + strings.ToLower(strings.TrimSpace(className))
}

func creationAssetURL(value string) string {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, "/assets/preview/") {
		return ""
	}
	return value
}

// CreationCallout is the temporary additive representation for live picks.
type CreationCallout struct {
	Number        int32
	Species       string
	Gender        string
	Class         string
	ClassCrestURL string
}

// DecodeCreationCallout reads the transition format used by the live view.
func DecodeCreationCallout(value string) (CreationCallout, bool) {
	fields := strings.Fields(strings.ToLower(strings.TrimSpace(value)))
	if len(fields) < 2 || fields[0] != "creation" {
		return CreationCallout{}, false
	}
	var result CreationCallout
	for _, field := range fields[1:] {
		key, val, found := strings.Cut(field, "=")
		if !found {
			continue
		}
		switch key {
		case "seat":
			number, err := strconv.ParseInt(val, 10, 32)
			if err != nil || number < 1 || number > 2 {
				return CreationCallout{}, false
			}
			result.Number = int32(number)
		case "species":
			result.Species = val
		case "gender":
			result.Gender = val
		case "class":
			result.Class = val
		case "class_crest":
			result.ClassCrestURL = val
		}
	}
	return result, result.Number > 0 && (result.Species != "" || result.Gender != "" || result.Class != "")
}
