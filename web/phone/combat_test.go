package phone

import (
	"context"
	"errors"
	"testing"

	df "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
)

func TestCombatModel_ProjectsTimerGridMovesAndCopies(t *testing.T) {
	fake := &actFake{result: ActResult{Value: &df.ActResponse{Accepted: true}}}
	model := NewCombatModel(fake, " seat-2 ")
	state := &df.ScreenState{View: &df.ScreenState_Phone{Phone: &df.PhoneView{
		StatusText: "Your turn", TurnTimer: &df.Timer{RemainingMs: 6000, TotalMs: 10000, Frozen: true},
		Moves: []*df.Move{{MoveId: "attack", Enabled: true, TargetId: "thrall"}},
		Combat: &df.CombatView{TokenId: "pc-2", Hp: 8, HpMax: 10, Statuses: []string{"bloodied"}, MyTurn: true, MoveLeftCells: 3, ContactInMs: 1200,
			MiniGrid: &df.MiniGrid{Cols: 2, Rows: 2, Reachable: []*df.Cell{{C: 1, R: 0}}}},
	}}}
	got := model.ApplyScreenState(state)
	if got.TokenID != "pc-2" || !got.MyTurn || got.TimerRemaining != 6000 || !got.TimerFrozen || got.StatusText != "Your turn" {
		t.Fatalf("combat snapshot = %+v", got)
	}
	got.Statuses[0], got.Moves[0].MoveId, got.MiniGrid.Reachable[0].C = "changed", "changed", 9
	copy := model.Snapshot()
	if copy.Statuses[0] != "bloodied" || copy.Moves[0].GetMoveId() != "attack" || copy.MiniGrid.GetReachable()[0].GetC() != 1 {
		t.Fatal("snapshot exposed mutable combat state")
	}
}

func TestCombatModel_TapsCreateCombatRequests(t *testing.T) {
	fake := &actFake{result: ActResult{Value: &df.ActResponse{Accepted: true}}}
	model := NewCombatModel(fake, "seat")
	cases := []struct {
		name string
		move *df.Move
		id   string
		want string
	}{
		{"attack", &df.Move{MoveId: "attack", Enabled: true, TargetId: "thrall"}, "thrall", "attack"},
		{"move", &df.Move{MoveId: "move", Enabled: true, Cell: &df.Cell{C: 2, R: 1}}, "", "move"},
		{"end turn", &df.Move{MoveId: "end_turn", Enabled: true}, "", "end_turn"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := (<-model.Tap(context.Background(), tc.move)).Err; err != nil {
				t.Fatal(err)
			}
			if fake.request.GetMoveId() != tc.want || fake.request.GetSeatToken() != "seat" || fake.request.GetTargetId() != tc.id {
				t.Fatalf("request = %+v", fake.request)
			}
		})
	}
	if fake.request.GetMoveId() != "end_turn" {
		t.Fatal("last tap was not sent")
	}
}

func TestCombatModel_RejectsUnavailableTapsAndRecordsFailures(t *testing.T) {
	model := NewCombatModel(nil, "seat")
	for _, action := range []func() <-chan ActResult{
		func() <-chan ActResult { return model.Tap(context.Background(), nil) },
		func() <-chan ActResult { return model.Attack(context.Background(), "") },
		func() <-chan ActResult { return model.Move(context.Background(), nil) },
		func() <-chan ActResult { return model.EndTurn(context.Background()) },
	} {
		if (<-action()).Err == nil {
			t.Fatal("invalid action was accepted")
		}
	}
	fake := &actFake{result: ActResult{Err: errors.New("offline")}}
	got := NewCombatModel(fake, "seat").ApplyAct(<-NewCombatModel(fake, "seat").Attack(context.Background(), "thrall"))
	if got.Error != "offline" {
		t.Fatalf("failure = %+v", got)
	}
	if NewCombatModel(fake, "seat").ApplyAct(ActResult{Value: &df.ActResponse{Accepted: false}}).Error == "" {
		t.Fatal("empty rejection was not given a message")
	}
}

func TestCombatModel_NilAndMissingCombatAreSafe(t *testing.T) {
	var model *CombatModel
	if model.Snapshot().Error == "" || model.ApplyScreenState(&df.ScreenState{}).Error == "" {
		t.Fatal("nil model was not safe")
	}
	model = NewCombatModel(nil, "seat")
	got := model.ApplyScreenState(&df.ScreenState{View: &df.ScreenState_Phone{Phone: &df.PhoneView{StatusText: "Brace yourself"}}})
	if got.StatusText != "Brace yourself" || len(got.Moves) != 0 {
		t.Fatalf("missing combat = %+v", got)
	}
}
