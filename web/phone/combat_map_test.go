package phone

import (
	"context"
	"errors"
	"strings"
	"testing"

	df "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
)

// mapState is a 5 x 3 grid (wider than tall, so it turns) with one blocked
// cell, pc-1 at (0,1), pc-2 at (1,0), the thrall at (4,1); (1,1) is a normal
// move and (3,2) is dash-only.
func mapScreenState(seq uint64, me *df.Cell, path []*df.Cell, myTurn bool) *df.ScreenState {
	grid := &df.MiniGrid{Cols: 5, Rows: 3, MeTokenId: "pc-1", CanDash: true, WalkStepMs: 250, DashStepMs: 150}
	for row := int32(0); row < 3; row++ {
		for column := int32(0); column < 5; column++ {
			if column == 2 && row == 0 {
				continue
			}
			grid.Walkable = append(grid.Walkable, &df.Cell{C: column, R: row})
		}
	}
	anim := "idle"
	if len(path) > 0 {
		anim = "walk"
	}
	grid.Tokens = []*df.MapToken{
		{TokenId: "pc-1", Name: "Astra", Kind: "rogue", Cell: me, Me: true, Active: myTurn, Seat: 1, Path: path, Anim: anim, AnimSeq: seq},
		{TokenId: "pc-2", Name: "Bram", Kind: "paladin", Cell: &df.Cell{C: 1, R: 0}, Seat: 2, AnimSeq: 1},
		{TokenId: "thrall", Name: "Drowned thrall", Kind: "thrall", Cell: &df.Cell{C: 4, R: 1}, Enemy: true, AnimSeq: 1},
	}
	if myTurn {
		grid.Paths = []*df.ReachPath{
			{Cell: &df.Cell{C: 1, R: 1}, Path: []*df.Cell{{C: 1, R: 1}}},
			{Cell: &df.Cell{C: 3, R: 2}, Path: []*df.Cell{{C: 1, R: 2}, {C: 2, R: 2}, {C: 3, R: 2}}, Dash: true},
		}
		grid.Reachable = []*df.Cell{{C: 1, R: 1}}
		grid.DashReachable = []*df.Cell{{C: 3, R: 2}}
	}
	return &df.ScreenState{Phase: "combat", View: &df.ScreenState_Phone{Phone: &df.PhoneView{Combat: &df.CombatView{TokenId: "pc-1", Hp: 9, HpMax: 12, MyTurn: myTurn, MoveLeftCells: 6, MiniGrid: grid}}}}
}

func TestCombatMap_selectCommitMoveAndDash(t *testing.T) {
	client := &recordingActFake{result: ActResult{Value: &df.ActResponse{Accepted: true}}}
	model := NewCombatModel(client, "seat-token")
	model.ApplyScreenState(mapScreenState(1, &df.Cell{C: 0, R: 1}, nil, true))
	model.OpenMap()
	if !model.MapOpen() {
		t.Fatal("map did not open")
	}
	model.SelectCell(&df.Cell{C: 4, R: 2})
	if model.mapState.selected != nil {
		t.Fatal("an unreachable cell was selected")
	}
	if result := <-model.CommitMove(context.Background()); result.Err == nil {
		t.Fatal("commit without a cell was sent")
	}
	model.SelectCell(&df.Cell{C: 1, R: 1})
	layout := model.MapLayout(0)
	if layout.Preview != "Move · 1 cell" || layout.Dash || layout.Selected.GetC() != 1 || !layout.Interactive || layout.MoveLeft != 6 {
		t.Fatalf("move layout = %+v", layout)
	}
	model.ApplyCommit(<-model.CommitMove(context.Background()))
	if got := client.requests[0]; got.GetMoveId() != "move" || got.GetCell().GetC() != 1 || got.GetSeatToken() != "seat-token" {
		t.Fatalf("move request = %+v", got)
	}
	if model.mapState.selected != nil || model.mapState.pending || !model.MapOpen() {
		t.Fatal("an accepted commit must clear the pick and keep the map open")
	}
	model.SelectCell(&df.Cell{C: 3, R: 2})
	if layout := model.MapLayout(0); layout.Preview != "Dash · 3 cells (ends your turn)" || !layout.Dash {
		t.Fatalf("dash layout = %+v", layout)
	}
	model.ApplyCommit(<-model.CommitMove(context.Background()))
	if got := client.requests[1]; got.GetMoveId() != "dash" || got.GetCell().GetC() != 3 || got.GetCell().GetR() != 2 {
		t.Fatalf("dash request = %+v", got)
	}
	model.SelectCell(&df.Cell{C: 1, R: 1})
	model.ClearSelection()
	if model.mapState.selected != nil {
		t.Fatal("clear kept the pick")
	}
	model.CloseMap()
	if model.MapOpen() {
		t.Fatal("map did not close")
	}
}

func TestCombatMap_rejectedCommitKeepsThePick(t *testing.T) {
	client := &recordingActFake{result: ActResult{Value: &df.ActResponse{Accepted: false, Reason: "cell 1,1 is not reachable"}}}
	model := NewCombatModel(client, "seat")
	model.ApplyScreenState(mapScreenState(1, &df.Cell{C: 0, R: 1}, nil, true))
	model.OpenMap()
	model.SelectCell(&df.Cell{C: 1, R: 1})
	snapshot := model.ApplyCommit(<-model.CommitMove(context.Background()))
	if snapshot.Error != "cell 1,1 is not reachable" || model.mapState.selected == nil {
		t.Fatalf("rejected commit = %q, pick %v", snapshot.Error, model.mapState.selected)
	}
	failing := NewCombatModel(&recordingActFake{result: ActResult{Err: errors.New("offline")}}, "seat")
	failing.ApplyScreenState(mapScreenState(1, &df.Cell{C: 0, R: 1}, nil, true))
	failing.SelectCell(&df.Cell{C: 1, R: 1})
	if snapshot := failing.ApplyCommit(<-failing.CommitMove(context.Background())); snapshot.Error != "offline" {
		t.Fatalf("transport error = %q", snapshot.Error)
	}
}

func TestCombatMap_turnEndClosesTheMap(t *testing.T) {
	model := NewCombatModel(&recordingActFake{}, "seat")
	model.ApplyScreenState(mapScreenState(1, &df.Cell{C: 0, R: 1}, nil, true))
	model.OpenMap()
	model.SelectCell(&df.Cell{C: 1, R: 1})
	model.ApplyScreenState(mapScreenState(2, &df.Cell{C: 0, R: 1}, nil, false))
	if model.MapOpen() || model.mapState.selected != nil {
		t.Fatal("map stayed open after the turn ended")
	}
	if layout := model.MapLayout(0); layout.Interactive || layout.Preview != "" {
		t.Fatalf("watching layout is interactive: %+v", layout)
	}
	model.ApplyScreenState(&df.ScreenState{View: &df.ScreenState_Phone{Phone: &df.PhoneView{}}})
	if model.MapOpen() {
		t.Fatal("map open outside combat")
	}
}

func TestCombatMap_walkAnimationFollowsAnimSeq(t *testing.T) {
	model := NewCombatModel(&recordingActFake{}, "seat")
	clock := 1000.0
	model.SetMapClock(func() float64 { return clock })
	model.ApplyScreenState(mapScreenState(1, &df.Cell{C: 0, R: 1}, []*df.Cell{{C: 0, R: 1}}, true))
	if model.MapLayout(clock).Tokens[0].Walk != nil {
		t.Fatal("a path seen on the first snapshot was animated")
	}
	path := []*df.Cell{{C: 1, R: 2}, {C: 2, R: 2}}
	model.ApplyScreenState(mapScreenState(2, &df.Cell{C: 2, R: 2}, path, true))
	walk := model.MapLayout(clock + 100).Tokens[0].Walk
	if walk == nil || walk.DurationMS != 500 || walk.ElapsedMS != 100 || walk.Dash || walk.Name != "dfcm-pc-1-2" {
		t.Fatalf("walk = %+v", walk)
	}
	if !strings.HasPrefix(walk.Keyframes, "@keyframes dfcm-pc-1-2{0.00%{transform:translate(") || !strings.Contains(walk.Keyframes, "100.00%{transform:translate(0%,0%)}") {
		t.Fatalf("keyframes = %s", walk.Keyframes)
	}
	if model.MapLayout(clock + 600).Tokens[0].Walk != nil {
		t.Fatal("a finished walk is still animated")
	}
	if model.MapNow() != clock {
		t.Fatal("map clock not used")
	}
	model.ApplyScreenState(mapScreenState(2, &df.Cell{C: 2, R: 2}, path, true))
	if model.mapState.walks["pc-1"].startMS != clock {
		t.Fatal("a repeated anim_seq restarted the walk")
	}
}

func TestCombatMap_layoutTurnsAndTrimsTheGrid(t *testing.T) {
	model := NewCombatModel(&recordingActFake{}, "seat")
	model.ApplyScreenState(mapScreenState(1, &df.Cell{C: 0, R: 1}, nil, true))
	layout := model.MapLayout(0)
	if !layout.Rotated || layout.Cols != 3 || layout.Rows != 5 || len(layout.Cells) != 15 {
		t.Fatalf("layout = %d x %d rotated %v cells %d", layout.Cols, layout.Rows, layout.Rotated, len(layout.Cells))
	}
	me := layout.Tokens[0]
	if me.X != 1 || me.Y != 0 || !me.Me || layout.Tokens[2].X != 1 || layout.Tokens[2].Y != 4 || !layout.Tokens[2].Enemy {
		t.Fatalf("tokens = %+v", layout.Tokens)
	}
	for _, cell := range layout.Cells {
		if cell.Cell.GetC() == 2 && cell.Cell.GetR() == 0 && cell.Walkable {
			t.Fatal("blocked cell drawn walkable")
		}
	}
	grid := &df.MiniGrid{Cols: 6, Rows: 6, Walkable: []*df.Cell{{C: 2, R: 1}, {C: 3, R: 4}}}
	c0, r0, c1, r1 := mapBounds(grid)
	if c0 != 2 || r0 != 1 || c1 != 3 || r1 != 4 {
		t.Fatalf("bounds = %d %d %d %d", c0, r0, c1, r1)
	}
	trimmed := buildMapLayout(grid, combatMapState{}, false, 0)
	if trimmed.Rotated || trimmed.Cols != 2 || trimmed.Rows != 4 || trimmed.Cells[0].Cell.GetC() != 2 {
		t.Fatalf("trimmed = %+v", trimmed)
	}
	if empty := buildMapLayout(&df.MiniGrid{Cols: 2, Rows: 2}, combatMapState{}, false, 0); empty.Cols != 2 || empty.Rows != 2 {
		t.Fatalf("empty grid layout = %+v", empty)
	}
	if none := buildMapLayout(nil, combatMapState{}, false, 0); none.Cols != 0 {
		t.Fatal("nil grid has a layout")
	}
}

func TestCombatMap_labelsAndHelpers(t *testing.T) {
	cases := []struct {
		got, want string
	}{
		{movePreviewLabel(4, false), "Move · 4 cells"},
		{movePreviewLabel(1, false), "Move · 1 cell"},
		{movePreviewLabel(9, true), "Dash · 9 cells (ends your turn)"},
		{mapHeadline(6, true), "6 of 6 cells left · Dash to go further"},
		{mapHeadline(1, false), "1 of 6 cells left"},
		{cssIdent("pc-1 <x>"), "pc-1x"},
	}
	for _, tc := range cases {
		if tc.got != tc.want {
			t.Fatalf("got %q, want %q", tc.got, tc.want)
		}
	}
	if route := walkRoute(&df.Cell{C: 0, R: 0}, []*df.Cell{{C: 1, R: 1}}, &df.Cell{C: 1, R: 1}); len(route) != 2 {
		t.Fatalf("route with start = %v", route)
	}
	if route := walkRoute(&df.Cell{C: 5, R: 5}, []*df.Cell{{C: 1, R: 1}}, &df.Cell{C: 1, R: 1}); len(route) != 1 {
		t.Fatalf("route from a far cell = %v", route)
	}
	if route := walkRoute(nil, []*df.Cell{{C: 1, R: 1}}, &df.Cell{C: 2, R: 2}); route != nil {
		t.Fatalf("route not ending on the token = %v", route)
	}
	var nilModel *CombatModel
	nilModel.OpenMap()
	nilModel.CloseMap()
	nilModel.SelectCell(nil)
	nilModel.ClearSelection()
	nilModel.SetMapClock(nil)
	if nilModel.MapOpen() || nilModel.MapNow() != 0 || nilModel.MapLayout(0).Cols != 0 || nilModel.ApplyCommit(ActResult{}).Error == "" {
		t.Fatal("nil model helpers")
	}
	if result := <-nilModel.CommitMove(context.Background()); result.Err == nil {
		t.Fatal("nil model committed")
	}
}
