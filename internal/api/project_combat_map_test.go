package api

import (
	"testing"

	df "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

func combatMapView() domain.View {
	tokens := []domain.CombatMapToken{
		{ID: "pc-1", Seat: 1, Kind: "rogue", Cell: domain.Cell{C: 2, R: 0}, HP: 9, HPMax: 11, Active: true, Path: []domain.Cell{{C: 1, R: 0}, {C: 2, R: 0}}, Anim: "walk", AnimSeq: 4, StepMS: 250},
		{ID: "pc-2", Seat: 2, Kind: "paladin", Cell: domain.Cell{C: 3, R: 0}, HP: 12, HPMax: 12},
		{ID: "thrall", Kind: "thrall", Cell: domain.Cell{C: 6, R: 0}, HP: 12, HPMax: 12, Enemy: true},
	}
	active := &domain.CombatMapView{Cols: 16, Rows: 10, Walkable: []domain.Cell{{C: 2, R: 0}, {C: 3, R: 0}}, Tokens: tokens, Me: "pc-1", MoveLeft: 4, CanDash: true, WalkStepMS: 250, DashStepMS: 150,
		Reach: []domain.CombatMapReach{{Cell: domain.Cell{C: 2, R: 1}, Path: []domain.Cell{{C: 2, R: 1}}}, {Cell: domain.Cell{C: 9, R: 1}, Path: []domain.Cell{{C: 3, R: 1}, {C: 9, R: 1}}, Dash: true}}}
	watching := &domain.CombatMapView{Cols: 16, Rows: 10, Tokens: tokens, Me: "pc-2"}
	return domain.View{Path: vocab.StateCombat, Combat: &domain.CombatView{Tokens: []domain.TokenView{{ID: "pc-1", HP: 9, HPMax: 11, Active: true}}}, Seats: []domain.SeatView{
		{Seat: 1, Character: &domain.Character{Name: "Astra Vale", Species: "Elf"}, CombatMap: active},
		{Seat: 2, Character: &domain.Character{Name: "Bram", Portrait: "sha-bram"}, CombatMap: watching},
	}}
}

func TestProjectPhone_combatMapForTheActiveSeat(t *testing.T) {
	phone := projectPhone(combatMapView(), 1)
	combat := phone.GetCombat()
	grid := combat.GetMiniGrid()
	if combat.GetTokenId() != "pc-1" || !combat.GetMyTurn() || combat.GetHp() != 9 || combat.GetMoveLeftCells() != 4 {
		t.Fatalf("combat = %+v", combat)
	}
	if grid.GetCols() != 16 || grid.GetRows() != 10 || len(grid.GetWalkable()) != 2 || grid.GetMeTokenId() != "pc-1" || !grid.GetCanDash() || grid.GetDashStepMs() != 150 || grid.GetWalkStepMs() != 250 {
		t.Fatalf("grid header = %+v", grid)
	}
	if len(grid.GetReachable()) != 1 || len(grid.GetDashReachable()) != 1 || len(grid.GetPaths()) != 2 || !grid.GetPaths()[1].GetDash() || len(grid.GetPaths()[1].GetPath()) != 2 {
		t.Fatalf("reach = %+v", grid)
	}
	me, other, thrall := grid.GetTokens()[0], grid.GetTokens()[1], grid.GetTokens()[2]
	if !me.GetMe() || me.GetName() != "Astra Vale" || me.GetPortraitUrl() != "ui/species_elf" || me.GetAnim() != "walk" || me.GetAnimSeq() != 4 || len(me.GetPath()) != 2 || me.GetStepMs() != 250 {
		t.Fatalf("me token = %+v", me)
	}
	if other.GetMe() || other.GetName() != "Bram" || other.GetPortraitUrl() != "sha-bram" || other.GetSeat() != 2 {
		t.Fatalf("other token = %+v", other)
	}
	if !thrall.GetEnemy() || thrall.GetName() != "Drowned thrall" || grid.GetThrall().GetC() != 6 || grid.GetMe().GetC() != 2 {
		t.Fatalf("thrall token = %+v", thrall)
	}
}

func TestProjectPhone_combatMapForTheWatchingSeat(t *testing.T) {
	combat := projectPhone(combatMapView(), 2).GetCombat()
	grid := combat.GetMiniGrid()
	if combat.GetTokenId() != "pc-2" || combat.GetMyTurn() || combat.GetHp() != 12 {
		t.Fatalf("watching combat = %+v", combat)
	}
	if len(grid.GetReachable()) != 0 || len(grid.GetPaths()) != 0 || grid.GetCanDash() || len(grid.GetTokens()) != 3 {
		t.Fatalf("watching grid = %+v", grid)
	}
}

func TestProjectPhoneCombatMap_withoutAMap(t *testing.T) {
	if projectPhoneCombatMap(nil, domain.View{}, 1) != nil {
		t.Fatal("nil combat view was replaced")
	}
	in := &df.CombatView{TokenId: "x"}
	if out := projectPhoneCombatMap(in, domain.View{Seats: []domain.SeatView{{Seat: 1}}}, 1); out != in || out.GetMiniGrid() != nil {
		t.Fatalf("seat without a map = %+v", out)
	}
}
