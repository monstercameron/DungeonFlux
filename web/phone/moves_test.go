package phone

import (
	"context"
	"errors"
	"testing"

	df "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
)

func TestMovesModel_ProjectsReasonsAndSendsTargetedAct(t *testing.T) {
	fake := &actFake{result: ActResult{Value: &df.ActResponse{Accepted: true}}}
	model := NewMovesModel(fake, " seat-2 ")
	state := &df.ScreenState{View: &df.ScreenState_Phone{Phone: &df.PhoneView{StatusText: "Your turn", Moves: []*df.Move{
		{MoveId: "attack", Label: "Attack", Enabled: true, TargetId: "thrall", Cell: &df.Cell{C: 2, R: 3}, Preview: &df.MovePreview{Modifier: 5}},
		{MoveId: "persuade", Label: "Persuade", Reason: "Not during combat"},
	}}}}
	got := model.ApplyScreenState(state)
	if len(got.Moves) != 2 || got.Moves[1].Reason != "Not during combat" || got.StatusText != "Your turn" {
		t.Fatalf("snapshot = %+v", got)
	}
	if err := (<-model.Tap(context.Background(), "attack")).Err; err != nil {
		t.Fatal(err)
	}
	if fake.request.GetSeatToken() != "seat-2" || fake.request.GetMoveId() != "attack" || fake.request.GetTargetId() != "thrall" || fake.request.GetCell().GetC() != 2 {
		t.Fatalf("request = %+v", fake.request)
	}
}

func TestMovesModel_DisablesIllegalAndUnknownMoves(t *testing.T) {
	fake := &actFake{}
	model := NewMovesModel(fake, "token")
	model.ApplyScreenState(&df.ScreenState{View: &df.ScreenState_Phone{Phone: &df.PhoneView{Moves: []*df.Move{{MoveId: "attack", Enabled: false, Reason: "Target out of range (60 ft)"}}}}})
	for _, test := range []struct{ name, id, want string }{{"disabled", "attack", "Target out of range (60 ft)"}, {"unknown", "missing", "move is unavailable"}} {
		t.Run(test.name, func(t *testing.T) {
			if err := (<-model.Tap(context.Background(), test.id)).Err; err == nil || err.Error() != test.want {
				t.Fatalf("error = %v", err)
			}
		})
	}
	if fake.request != nil {
		t.Fatal("illegal tap reached the client")
	}
}

func TestMovesModel_RecordsActFailuresAndCopiesNestedValues(t *testing.T) {
	fake := &actFake{result: ActResult{Err: errors.New("offline")}}
	model := NewMovesModel(fake, "token")
	model.ApplyScreenState(&df.ScreenState{View: &df.ScreenState_Phone{Phone: &df.PhoneView{Moves: []*df.Move{{MoveId: "move", Label: "Move", Enabled: true, Options: []*df.Option{{Id: "north", Label: "North"}}, Cell: &df.Cell{C: 1}}}}}})
	got := model.Snapshot()
	got.Moves[0].Options[0].Label = "changed"
	got.Moves[0].Cell.C = 9
	if model.Snapshot().Moves[0].Options[0].Label != "North" || model.Snapshot().Moves[0].Cell.GetC() != 1 {
		t.Fatal("snapshot exposed nested mutable values")
	}
	model.ApplyAct(<-model.Tap(context.Background(), "move"))
	if model.Snapshot().Error != "offline" {
		t.Fatalf("error = %q", model.Snapshot().Error)
	}
	if NewMovesModel(nil, "").Snapshot().Error != "" {
		t.Fatal("new model should start without an error")
	}
}

func TestMovesModel_NilAndMissingPhoneAreSafe(t *testing.T) {
	var model *MovesModel
	if model.Snapshot().Error != "moves model is unavailable" || model.ApplyScreenState(nil).Error != "moves model is unavailable" {
		t.Fatal("nil model should be safe")
	}
	model = NewMovesModel(&actFake{}, "token")
	if got := model.ApplyScreenState(&df.ScreenState{}); len(got.Moves) != 0 {
		t.Fatalf("missing phone moves = %+v", got)
	}
}

func TestMoveReason_UsesFallbackOnlyForDisabledMoves(t *testing.T) {
	tests := []struct {
		name string
		move MoveSnapshot
		want string
	}{
		{name: "server reason", move: MoveSnapshot{Enabled: false, Reason: "Target out of range"}, want: "Target out of range"},
		{name: "fallback", move: MoveSnapshot{Enabled: false}, want: "Move unavailable"},
		{name: "enabled blank", move: MoveSnapshot{Enabled: true}, want: ""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := MoveReason(test.move); got != test.want {
				t.Fatalf("MoveReason() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestMovePreviewText_DescribesRollAndDamage(t *testing.T) {
	tests := []struct {
		name string
		move MoveSnapshot
		want string
	}{
		{name: "attack", move: MoveSnapshot{Preview: &df.MovePreview{Vs: 13, Damage: &df.DamagePreview{Dice: "1d8", Bonus: 3}}}, want: "vs DC 13 · 1d8+3 damage"},
		{name: "modifier", move: MoveSnapshot{Preview: &df.MovePreview{Modifier: 4}}, want: "+4 modifier"},
		{name: "empty", move: MoveSnapshot{}, want: ""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := MovePreviewText(test.move); got != test.want {
				t.Fatalf("MovePreviewText() = %q, want %q", got, test.want)
			}
		})
	}
}
