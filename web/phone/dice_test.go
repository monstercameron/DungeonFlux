package phone

import (
	"context"
	"errors"
	"testing"

	df "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
)

func TestDiceModel_OffersRollAndResolves(t *testing.T) {
	fake := &actFake{result: ActResult{Value: &df.ActResponse{Accepted: true}}}
	model := NewDiceModel(fake, " seat-1 ")
	state := &df.ScreenState{Phase: "check", View: &df.ScreenState_Phone{Phone: &df.PhoneView{Moves: []*df.Move{{
		MoveId: "persuade", Enabled: true, Preview: &df.MovePreview{Modifier: 4, Vs: 10},
	}}}}}
	got := model.ApplyScreenState(state)
	if !got.CanRoll || got.Modifier != 4 || got.DC != 10 || got.Phase != DiceOffered {
		t.Fatalf("offered snapshot = %+v", got)
	}
	model.ApplyAct(<-model.Roll(context.Background()))
	if fake.request.GetMoveId() != "persuade" || fake.request.GetSeatToken() != "seat-1" || model.Snapshot().Phase != DiceRolling {
		t.Fatalf("rolling snapshot = %+v request = %+v", model.Snapshot(), fake.request)
	}
	resolved := &df.ScreenState{Phase: "resolution", View: &df.ScreenState_Phone{Phone: &df.PhoneView{StatusText: "Success"}}}
	if got := model.ApplyScreenState(resolved); got.Phase != DiceResolved || got.Outcome != "Success" {
		t.Fatalf("resolved snapshot = %+v", got)
	}
}

func TestDiceModel_RejectsUnavailableAndRejectedRoll(t *testing.T) {
	model := NewDiceModel(nil, "token")
	if got := (<-model.Roll(context.Background())).Err; got == nil || got.Error() != "dice client is unavailable" {
		t.Fatalf("missing client error = %v", got)
	}
	fake := &actFake{result: ActResult{Value: &df.ActResponse{Reason: "not your turn"}}}
	model = NewDiceModel(fake, "token")
	model.ApplyScreenState(&df.ScreenState{Phase: "check", View: &df.ScreenState_Phone{Phone: &df.PhoneView{Moves: []*df.Move{{MoveId: "persuade", Enabled: true}}}}})
	got := model.ApplyAct(<-model.Roll(context.Background()))
	if got.Phase != DiceOffered || !got.CanRoll || got.Error != "not your turn" {
		t.Fatalf("rejected snapshot = %+v", got)
	}
	model = NewDiceModel(fake, "token")
	model.ApplyScreenState(&df.ScreenState{Phase: "check", View: &df.ScreenState_Phone{Phone: &df.PhoneView{Moves: []*df.Move{{MoveId: "persuade", Enabled: true}}}}})
	got = model.ApplyAct(ActResult{Err: errors.New("offline")})
	if got.Phase != DiceFailed || got.Error != "offline" {
		t.Fatalf("failed snapshot = %+v", got)
	}
}

func TestDiceModel_PreservesStateWithoutPhone(t *testing.T) {
	model := NewDiceModel(&actFake{}, "token")
	model.ApplyScreenState(&df.ScreenState{Phase: "check"})
	if got := model.Snapshot(); got.Phase != DiceOffered || got.DC != 10 {
		t.Fatalf("empty state = %+v", got)
	}
	if got := (*DiceModel)(nil).Snapshot(); got.Phase != DiceFailed {
		t.Fatalf("nil snapshot = %+v", got)
	}
}

func TestDiceFace_UsesPhaseAndValidD20(t *testing.T) {
	tests := []struct {
		name, want string
		phase      DicePhase
		d20        int32
	}{
		{name: "offered placeholder", phase: DiceOffered, d20: 0, want: "—"},
		{name: "rolling glyph", phase: DiceRolling, d20: 12, want: "…"},
		{name: "resolved face", phase: DiceResolved, d20: 17, want: "17"},
		{name: "invalid resolved placeholder", phase: DiceResolved, d20: 21, want: "—"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := DiceFace(test.phase, test.d20); got != test.want {
				t.Fatalf("DiceFace(%q, %d) = %q, want %q", test.phase, test.d20, got, test.want)
			}
		})
	}
}
