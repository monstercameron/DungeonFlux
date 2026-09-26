package dm

import (
	"reflect"
	"testing"

	dungeonfluxv1 "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
)

func TestDialogueModelFromState_ProjectsConversation(t *testing.T) {
	state := &dungeonfluxv1.ScreenState{
		Phase:         "conversation",
		SpotlightSeat: "2",
		Locale:        "en",
		View: &dungeonfluxv1.ScreenState_Dm{Dm: &dungeonfluxv1.DMView{
			Locale:    "es",
			Narration: &dungeonfluxv1.Narration{Speaker: "Mother Vell", TextSoFar: "The bell remembers."},
		}},
	}

	model := DialogueModelFromState(state)
	if !model.Visible || model.SpotlightSeat != "2" || model.NPCName != "Mother Vell" {
		t.Fatalf("conversation identity = %#v", model)
	}
	if model.Locale != "es" || model.Speaker != "Mother Vell" || model.Line != "The bell remembers." {
		t.Fatalf("conversation line = %#v", model)
	}
	if len(model.Options) != 3 || model.Options[0].ID != "persuade" || model.Options[1].ID != "ask_question" || model.Options[2].ID != "look_around" {
		t.Fatalf("conversation options = %#v", model.Options)
	}
	if !model.Options[0].Primary || model.Options[0].Detail != "+4 vs DC 10" {
		t.Fatalf("persuade presentation = %#v", model.Options[0])
	}
}

func TestDialogueModelFromState_UsesPhoneLegalMovesWhenPresent(t *testing.T) {
	state := &dungeonfluxv1.ScreenState{
		Phase: "conversation",
		View: &dungeonfluxv1.ScreenState_Phone{Phone: &dungeonfluxv1.PhoneView{Moves: []*dungeonfluxv1.Move{
			{MoveId: "persuade", Label: "Try to persuade her", Enabled: true},
			{MoveId: "leave", Label: "Leave the tavern", Enabled: false, Reason: "The door is barred"},
		}}},
	}

	model := DialogueModelFromState(state)
	if len(model.Options) != 2 || model.Options[0].Text != "Try to persuade her" || !model.Options[0].Enabled {
		t.Fatalf("legal moves = %#v", model.Options)
	}
	if model.Options[1].IconName != "ui/icon_step_away" || model.Options[1].Reason != "The door is barred" || model.Options[1].Enabled {
		t.Fatalf("disabled move = %#v", model.Options[1])
	}
}

func TestDialogueModelFromState_SkipsEmptyMovesAndUsesIDFallback(t *testing.T) {
	state := &dungeonfluxv1.ScreenState{
		Phase: "conversation",
		View: &dungeonfluxv1.ScreenState_Phone{Phone: &dungeonfluxv1.PhoneView{Moves: []*dungeonfluxv1.Move{
			nil,
			{MoveId: "", Label: "  "},
			{MoveId: "custom_move", Enabled: true},
		}}},
	}

	model := DialogueModelFromState(state)
	if len(model.Options) != 1 || model.Options[0].ID != "custom_move" || model.Options[0].Text != "custom_move" {
		t.Fatalf("fallback move = %#v", model.Options)
	}
	if model.Options[0].IconName != "ui/icon_talk" || !model.Options[0].Primary {
		t.Fatalf("fallback move styling = %#v", model.Options[0])
	}
}

func TestDialogueModelFromState_UsesSubtitleAndPausedReasons(t *testing.T) {
	state := &dungeonfluxv1.ScreenState{
		Phase:  "conversation",
		Paused: true,
		View: &dungeonfluxv1.ScreenState_Dm{Dm: &dungeonfluxv1.DMView{
			Subtitle: &dungeonfluxv1.Subtitle{Text: "Ask carefully."},
		}},
	}

	model := DialogueModelFromState(state)
	if model.Speaker != "Mother Vell" || model.Line != "Ask carefully." {
		t.Fatalf("subtitle fallback = %#v", model)
	}
	for _, option := range model.Options {
		if option.Enabled || option.Reason != "Game is paused" {
			t.Fatalf("paused option = %#v", option)
		}
	}
}

func TestDialogueModelFromState_HidesOutsideConversation(t *testing.T) {
	cases := []*dungeonfluxv1.ScreenState{
		nil,
		{Phase: "opening", View: &dungeonfluxv1.ScreenState_Dm{Dm: &dungeonfluxv1.DMView{}}},
		{Phase: "conversation"},
	}
	for index, state := range cases {
		t.Run(string(rune('a'+index)), func(t *testing.T) {
			model := DialogueModelFromState(state)
			if state == nil && !reflect.DeepEqual(model, DialogueModel{}) {
				t.Fatalf("nil state = %#v", model)
			}
			if state != nil && model.Visible != (state.GetPhase() == "conversation") {
				t.Fatalf("visibility = %#v", model)
			}
		})
	}
}

func TestDialogueModelFromState_EmptyNarrationIsSafe(t *testing.T) {
	state := &dungeonfluxv1.ScreenState{
		Phase: "conversation",
		View: &dungeonfluxv1.ScreenState_Dm{Dm: &dungeonfluxv1.DMView{
			Narration: &dungeonfluxv1.Narration{Speaker: "Mother Vell"},
			Subtitle:  &dungeonfluxv1.Subtitle{},
		}},
	}
	model := DialogueModelFromState(state)
	if model.Line != "" || model.Speaker != "" {
		t.Fatalf("empty line = %#v", model)
	}
}
