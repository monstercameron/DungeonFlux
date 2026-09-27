package dm

import (
	"strconv"
	"strings"

	dungeonfluxv1 "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
)

// HUDPartyMember is the TV-facing party card for one built hero.
type HUDPartyMember struct {
	Number      int32
	Name        string
	Class       string
	PortraitURL string
	CrestArt    string
	Spotlight   bool
	HP          int32
	HPMax       int32
	HPKnown     bool
	HPPercent   int
}

// HUDAction is one legal exploration hint shown along the lower edge of the TV.
type HUDAction struct {
	ID      string
	Label   string
	Icon    string
	Hotkey  string
	Enabled bool
	Primary bool
	Reason  string
}

// HUDChecklistItem is one optional objective checkpoint shown on the TV.
type HUDChecklistItem struct {
	Label string
	Done  bool
}

// HUDModel contains the exploration-only overlay projected from a DM snapshot.
type HUDModel struct {
	Visible          bool
	SpotlightSeat    int32
	Party            []HUDPartyMember
	Objective        string
	ObjectiveVisible bool
	Checklist        []HUDChecklistItem
	Actions          []HUDAction
	NarrationSpeaker string
	NarrationText    string
	Location         string
	Act              string
	MinimapURL       string
}

// HUDModelFromState projects the exploration HUD without mutating the wire view.
// DMView does not carry phone move lists, so exploration hints are the two
// engine-defined moves for this phase and are gated by the shared spotlight.
func HUDModelFromState(state *dungeonfluxv1.ScreenState) HUDModel {
	if state == nil || state.GetDm() == nil {
		return HUDModel{}
	}
	spotlight := parseSeat(state.GetSpotlightSeat())
	view := state.GetDm()
	model := HUDModel{
		Visible:       strings.EqualFold(strings.TrimSpace(state.GetPhase()), "exploration"),
		SpotlightSeat: spotlight,
		Party:         projectHUDParty(view, spotlight),
		Objective:     projectHUDObjective(view),
		Actions:       projectHUDActions(spotlight),
		Location:      "The Drowned Lantern",
		Act:           "Act I · The Tavern",
		MinimapURL:    ArtURL("battlefield_flat"),
	}
	model.NarrationSpeaker, model.NarrationText = projectHUDNarration(view)
	model.ObjectiveVisible = model.Objective != ""
	return model
}

func parseSeat(value string) int32 {
	number, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || number < 1 {
		return 0
	}
	return int32(number)
}

func projectHUDParty(view *dungeonfluxv1.DMView, spotlight int32) []HUDPartyMember {
	party := make([]HUDPartyMember, 0, len(view.GetBuildCards()))
	for _, card := range view.GetBuildCards() {
		if card == nil {
			continue
		}
		member := HUDPartyMember{
			Number:      card.GetPlayerNumber(),
			Name:        card.GetName(),
			Class:       card.GetClassName(),
			PortraitURL: card.GetPortraitUrl(),
			CrestArt:    classCrestArt(card.GetClassName()),
			Spotlight:   card.GetPlayerNumber() == spotlight,
		}
		if build := card.GetCharacter().GetBuild(); build != nil {
			member.HP, member.HPMax = build.GetHp(), build.GetHpMax()
			member.HPKnown, member.HPPercent = member.HPMax > 0, hpPercent(member.HP, member.HPMax)
		}
		if token, ok := tokenForMember(view.GetTokens(), member.Name); ok {
			member.HP, member.HPMax = token.GetHp(), token.GetHpMax()
			member.HPKnown = member.HPMax > 0
			member.HPPercent = hpPercent(member.HP, member.HPMax)
		}
		party = append(party, member)
	}
	return party
}

func tokenForMember(tokens []*dungeonfluxv1.Token, name string) (*dungeonfluxv1.Token, bool) {
	for _, token := range tokens {
		if token != nil && strings.EqualFold(strings.TrimSpace(token.GetName()), strings.TrimSpace(name)) && strings.TrimSpace(name) != "" {
			return token, true
		}
	}
	return nil, false
}

func hpPercent(hp, maxHP int32) int {
	if maxHP <= 0 {
		return 0
	}
	if hp < 0 {
		hp = 0
	}
	if hp > maxHP {
		hp = maxHP
	}
	return int((hp*100 + maxHP/2) / maxHP)
}

func projectHUDObjective(view *dungeonfluxv1.DMView) string {
	if notice := view.GetNotice(); notice != nil {
		if text := strings.TrimSpace(notice.GetFallback()); text != "" {
			return text
		}
	}
	return strings.TrimSpace(view.GetCallout())
}

func projectHUDActions(spotlight int32) []HUDAction {
	active := spotlight > 0
	reason := ""
	if !active {
		reason = "Waiting for the spotlight"
	}
	return []HUDAction{
		{ID: "talk_vell", Label: "Talk", Icon: "✦", Hotkey: "1", Enabled: active, Primary: true, Reason: reason},
		{ID: "leave", Label: "Leave", Icon: "↗", Hotkey: "2", Enabled: active, Reason: reason},
	}
}

func projectHUDNarration(view *dungeonfluxv1.DMView) (string, string) {
	if view == nil {
		return "", ""
	}
	narration := view.GetNarration()
	if narration != nil && strings.TrimSpace(narration.GetTextSoFar()) != "" {
		speaker := strings.TrimSpace(narration.GetSpeaker())
		if speaker == "" {
			speaker = "Dungeon Master"
		}
		return speaker, strings.TrimSpace(narration.GetTextSoFar())
	}
	if subtitle := view.GetSubtitle(); subtitle != nil && strings.TrimSpace(subtitle.GetText()) != "" {
		return "Dungeon Master", strings.TrimSpace(subtitle.GetText())
	}
	return "", ""
}

func classCrestArt(className string) string {
	className = strings.ToLower(strings.TrimSpace(className))
	className = strings.ReplaceAll(className, " ", "_")
	if className == "" {
		return ""
	}
	return "ui/class_" + className
}
