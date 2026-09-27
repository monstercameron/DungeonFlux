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
	// Scores, HP, HPMax, and AC carry the seat's real rolled build (DM-036).
	// HasStats is false until the engine has published a roll, so the TV
	// shows placeholders instead of invented numbers.
	Scores   AbilityScores
	HP       int32
	HPMax    int32
	AC       int32
	HasStats bool
}

// AbilityScores holds the six SRD ability scores for one rolled hero.
type AbilityScores struct {
	STR, DEX, CON, INT, WIS, CHA int32
}

func creationTimerRemainingAt(remainingMS int64, anchorMS, nowMS float64) int64 {
	if nowMS <= anchorMS {
		return remainingMS
	}
	remainingMS -= int64(nowMS - anchorMS)
	if remainingMS < 0 {
		return 0
	}
	return remainingMS
}

// abilityModifier returns the standard SRD modifier for an ability score.
func abilityModifier(score int32) int32 {
	if score == 0 {
		return 0
	}
	mod := (score - 10) / 2
	if score < 10 && (score-10)%2 != 0 {
		mod--
	}
	return mod
}

// titleCaseWord upper-cases the first letter of each hyphen or space
// separated segment, so wire values such as "half-orc" or "bard" render as
// "Half-Orc" and "Bard" instead of lowercase raw strings.
func titleCaseWord(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return value
	}
	var out strings.Builder
	upperNext := true
	for _, r := range value {
		if upperNext && r >= 'a' && r <= 'z' {
			out.WriteRune(r - ('a' - 'A'))
			upperNext = false
			continue
		}
		out.WriteRune(r)
		upperNext = r == '-' || r == ' ' || r == '\''
	}
	return out.String()
}

// CreationModel contains the two seats and the shared creation prompt.
type CreationModel struct {
	Seats  [2]CreationSeat
	Prompt string
	Timer  TimerView
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
	model.Timer = TimerViewFromDMView(view)
	for _, card := range view.GetBuildCards() {
		if card == nil || card.GetPlayerNumber() < 1 || card.GetPlayerNumber() > 2 {
			continue
		}
		seat := &model.Seats[card.GetPlayerNumber()-1]
		seat.Name, seat.PortraitURL = card.GetName(), artSrc(card.GetPortraitUrl())
		if class := card.GetClassName(); class != "" {
			seat.Class = titleCaseWord(class)
		}
		if seat.Name != "" || seat.Class != "" || seat.PortraitURL != "" {
			seat.Status, seat.Ready = "Hero ready", true
		}
		if hero := card.GetCharacter(); hero != nil {
			seat.Species, seat.Gender = titleCaseWord(hero.GetSpecies()), titleCaseWord(hero.GetGender())
			seat.Ready, seat.Status = card.GetReady(), "Reviewing hero"
			if seat.Ready {
				seat.Status = "Ready for adventure"
			}
			if build := hero.GetBuild(); len(build.GetAbilities()) == 6 {
				a := build.GetAbilities()
				seat.Scores = AbilityScores{a[0], a[1], a[2], a[3], a[4], a[5]}
				seat.HP, seat.HPMax, seat.AC, seat.HasStats = build.GetHp(), build.GetHpMax(), build.GetAc(), true
			}
		}
	}
	if update, ok := DecodeCreationCallout(view.GetCallout()); ok {
		seat := &model.Seats[update.Number-1]
		seat.Species, seat.Gender = titleCaseWord(update.Species), titleCaseWord(update.Gender)
		if update.Class != "" {
			seat.Class = titleCaseWord(update.Class)
		}
		if update.ClassCrestURL != "" {
			seat.ClassCrestURL = artSrc(update.ClassCrestURL)
		}
		if update.HasStats {
			seat.Scores, seat.HP, seat.HPMax, seat.AC, seat.HasStats = update.Scores, update.HP, update.HPMax, update.AC, true
		}
		if !seat.Ready {
			seat.Status = creationPickStatus(*seat)
		}
	}
	return model
}

// creationHeroStandIn resolves the best available portrait for a seat: its
// generated portrait, else a gendered species stand-in (ui/species_<species>_
// <gender>), else the plain species or class stand-in, else a seeded random
// species so the portrait well never shows an empty gradient once a player
// has made a choice.
func creationHeroStandIn(seat CreationSeat) string {
	if seat.PortraitURL != "" {
		return seat.PortraitURL
	}
	species := strings.ToLower(strings.TrimSpace(seat.Species))
	gender := strings.ToLower(strings.TrimSpace(seat.Gender))
	if species != "" && gender != "" {
		if url := ArtURL("ui/species_" + species + "_" + gender); url != "" {
			return url
		}
		return ArtURL(classArtName(seat.Class))
	}
	if species != "" {
		if url := ArtURL("ui/species_" + species); url != "" {
			return url
		}
	}
	if class := strings.ToLower(strings.TrimSpace(seat.Class)); class != "" {
		if url := ArtURL("ui/class_" + class); url != "" {
			return url
		}
	}
	if seat.Name == "" && seat.Species == "" && seat.Class == "" {
		return ""
	}
	return heroProxyArt(seat.Species, seat.Class, seat.Name+strconv.Itoa(int(seat.Number)))
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
	return artSrc(value)
}

// CreationCallout is the temporary additive representation for live picks.
//
// Scores/HP/HPMax/AC/HasStats are an additive extension of the same compact
// callout format for the seat's real roll (DM-036), following the pattern
// already established for species/gender/class: BuildCard has no ability-
// score fields yet (a wire contract gap flagged in this round's hand-in), so
// the engine may publish "creation seat=1 ... str=16 dex=14 con=14 int=10
// wis=12 cha=10 hp=12 hpmax=12 ac=16" and the TV renders it without a proto
// change. Until the engine emits these fields, HasStats stays false and the
// TV shows placeholders instead of an invented roll.
type CreationCallout struct {
	Number        int32
	Species       string
	Gender        string
	Class         string
	ClassCrestURL string
	Scores        AbilityScores
	HP            int32
	HPMax         int32
	AC            int32
	HasStats      bool
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
		case "str":
			result.Scores.STR, result.HasStats = parseStatInt(val), true
		case "dex":
			result.Scores.DEX, result.HasStats = parseStatInt(val), true
		case "con":
			result.Scores.CON, result.HasStats = parseStatInt(val), true
		case "int":
			result.Scores.INT, result.HasStats = parseStatInt(val), true
		case "wis":
			result.Scores.WIS, result.HasStats = parseStatInt(val), true
		case "cha":
			result.Scores.CHA, result.HasStats = parseStatInt(val), true
		case "hp":
			result.HP = parseStatInt(val)
		case "hpmax":
			result.HPMax = parseStatInt(val)
		case "ac":
			result.AC = parseStatInt(val)
		}
	}
	return result, result.Number > 0 && (result.Species != "" || result.Gender != "" || result.Class != "" || result.HasStats)
}

func parseStatInt(value string) int32 {
	number, err := strconv.ParseInt(value, 10, 32)
	if err != nil {
		return 0
	}
	return int32(number)
}
