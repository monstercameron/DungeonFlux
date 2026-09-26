package dm

import (
	"strings"

	dungeonfluxv1 "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
)

// DialogueOption is one server-authoritative conversation move presented on
// the TV. The TV is display-only; the matching phone sends the move.
type DialogueOption struct {
	ID        string
	MoveLabel string
	Text      string
	Detail    string
	Reason    string
	IconName  string
	IconURL   string
	Enabled   bool
	Primary   bool
}

// DialogueModel contains the conversation lower-third and its legal choices.
type DialogueModel struct {
	Visible       bool
	NPCName       string
	PortraitURL   string
	Speaker       string
	Line          string
	SpotlightSeat string
	Locale        string
	Paused        bool
	Options       []DialogueOption
}

// DialogueModelFromState projects a DM snapshot into the TV conversation
// surface. The DM wire view does not yet carry PhoneView.Moves, so the
// conversation projection keeps the engine's closed move vocabulary here.
func DialogueModelFromState(state *dungeonfluxv1.ScreenState) DialogueModel {
	if state == nil {
		return DialogueModel{}
	}
	view := state.GetDm()
	model := DialogueModel{
		Visible:       strings.EqualFold(strings.TrimSpace(state.GetPhase()), "conversation"),
		SpotlightSeat: strings.TrimSpace(state.GetSpotlightSeat()),
		Paused:        state.GetPaused(),
		Locale:        localeOrDefault(state.GetLocale()),
		NPCName:       "Mother Vell",
		PortraitURL:   ArtURL("mother_vell"),
	}
	if view == nil {
		return model
	}
	model.Locale = localeOrDefault(view.GetLocale())
	model.Speaker, model.Line = dialogueLine(view)
	if model.Speaker == "" && model.Line != "" {
		model.Speaker = model.NPCName
	}
	model.Options = dialogueOptions(model.Paused)
	return model
}

func dialogueLine(view *dungeonfluxv1.DMView) (string, string) {
	if narration := view.GetNarration(); narration != nil {
		if text := strings.TrimSpace(narration.GetTextSoFar()); text != "" {
			return strings.TrimSpace(narration.GetSpeaker()), text
		}
	}
	if subtitle := view.GetSubtitle(); subtitle != nil {
		return "", strings.TrimSpace(subtitle.GetText())
	}
	return "", ""
}

func dialogueOptions(paused bool) []DialogueOption {
	options := []DialogueOption{
		{ID: "persuade", MoveLabel: "Persuade +4 vs DC 10", Text: "Try to persuade her", Detail: "+4 vs DC 10", IconName: "ui/icon_persuade", Primary: true},
		{ID: "step_away", MoveLabel: "Step away", Text: "Step away", IconName: "ui/icon_step_away"},
	}
	for index := range options {
		options[index].Enabled = !paused
		if paused {
			options[index].Reason = "Game is paused"
		}
		options[index].IconURL = ArtURL(options[index].IconName)
	}
	return options
}
