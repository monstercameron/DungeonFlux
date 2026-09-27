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

// DialoguePartyMember is one hero shown standing at the bar in the
// conversation's foreground, built from the DM view's build cards.
type DialoguePartyMember struct {
	Name        string
	Class       string
	PortraitURL string
}

// DialogueModel contains the conversation lower-third and its legal choices.
type DialogueModel struct {
	Visible       bool
	NPCName       string
	PortraitURL   string
	Speaker       string
	Line          string
	LineID        string
	Done          bool
	SpotlightSeat string
	Locale        string
	Paused        bool
	Options       []DialogueOption
	Party         []DialoguePartyMember
}

// maxDialogueParty caps the foreground party row; the demo table seats two.
const maxDialogueParty = 2

// DialogueModelFromState projects a DM snapshot into the TV conversation
// surface. When the snapshot also carries a phone view, its legal moves are
// used; otherwise the closed conversation vocabulary supplies the preview.
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
		model.Options = dialogueOptionsFromState(state)
		return model
	}
	model.Locale = localeOrDefault(view.GetLocale())
	model.Speaker, model.Line, model.LineID, model.Done = dialogueLine(view)
	if model.Speaker == "" && model.Line != "" {
		model.Speaker = model.NPCName
	}
	model.Options = dialogueOptionsFromState(state)
	model.Party = dialoguePartyFromBuildCards(view.GetBuildCards())
	return model
}

// dialoguePartyFromBuildCards projects the DM view's build cards into the
// foreground party row, skipping unnamed seats and capping at two heroes so
// the composition reads clean at the bar.
func dialoguePartyFromBuildCards(cards []*dungeonfluxv1.BuildCard) []DialoguePartyMember {
	party := make([]DialoguePartyMember, 0, maxDialogueParty)
	for _, card := range cards {
		if card == nil || len(party) >= maxDialogueParty {
			continue
		}
		name := strings.TrimSpace(card.GetName())
		if name == "" {
			continue
		}
		party = append(party, DialoguePartyMember{
			Name:        name,
			Class:       strings.TrimSpace(card.GetClassName()),
			PortraitURL: strings.TrimSpace(card.GetPortraitUrl()),
		})
	}
	return party
}

func dialogueLine(view *dungeonfluxv1.DMView) (string, string, string, bool) {
	if narration := view.GetNarration(); narration != nil {
		if text := strings.TrimSpace(narration.GetTextSoFar()); text != "" {
			return strings.TrimSpace(narration.GetSpeaker()), text, narration.GetLineId(), narration.GetDone()
		}
	}
	if subtitle := view.GetSubtitle(); subtitle != nil {
		return "", strings.TrimSpace(subtitle.GetText()), "", true
	}
	return "", "", "", false
}

func dialogueOptionsFromState(state *dungeonfluxv1.ScreenState) []DialogueOption {
	if state != nil {
		if phone := state.GetPhone(); phone != nil && len(phone.GetMoves()) > 0 {
			return dialogueOptionsFromMoves(phone.GetMoves(), state.GetPaused())
		}
		return dialogueOptions(state.GetPaused())
	}
	return nil
}

func dialogueOptionsFromMoves(moves []*dungeonfluxv1.Move, paused bool) []DialogueOption {
	options := make([]DialogueOption, 0, len(moves))
	for _, move := range moves {
		if move == nil {
			continue
		}
		label := strings.TrimSpace(move.GetLabel())
		if label == "" {
			label = strings.TrimSpace(move.GetMoveId())
		}
		if label == "" {
			continue
		}
		enabled := move.GetEnabled() && !paused
		reason := strings.TrimSpace(move.GetReason())
		if paused {
			reason = "Game is paused"
		}
		options = append(options, DialogueOption{
			ID: move.GetMoveId(), MoveLabel: label, Text: label, Reason: reason,
			IconName: dialogueIconName(move.GetMoveId()), Enabled: enabled,
			Primary: len(options) == 0,
		})
	}
	for index := range options {
		options[index].IconURL = ArtURL(options[index].IconName)
	}
	return options
}

func dialogueOptions(paused bool) []DialogueOption {
	options := []DialogueOption{
		{ID: "persuade", MoveLabel: "Persuade +4 vs DC 10", Text: "Try to persuade her", Detail: "+4 vs DC 10", IconName: "ui/icon_persuade", Primary: true},
		{ID: "ask_question", MoveLabel: "Ask a different question", Text: "Ask a different question", IconName: "ui/icon_talk"},
		{ID: "look_around", MoveLabel: "Look around the tavern", Text: "Look around the tavern", IconName: "ui/icon_move"},
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

func dialogueIconName(moveID string) string {
	switch strings.ToLower(strings.TrimSpace(moveID)) {
	case "persuade":
		return "ui/icon_persuade"
	case "leave", "step_away":
		return "ui/icon_step_away"
	case "talk", "talk_vell", "ask_question", "question":
		return "ui/icon_talk"
	case "move", "look_around", "explore":
		return "ui/icon_move"
	default:
		return "ui/icon_talk"
	}
}
