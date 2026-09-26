package phase

import (
	"testing"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/game/combat"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

// mapBattlefield is a 12 x 3 content grid with the middle of row 0 blocked.
func mapBattlefield() domain.Battlefield {
	grid := domain.Grid{Cols: 12, Rows: 3, Walkable: make([]bool, 36)}
	for index := range grid.Walkable {
		grid.Walkable[index] = true
	}
	grid.Walkable[5] = false // (5,0)
	return domain.Battlefield{Grid: grid, Spawns: []domain.Spawn{
		{Seat: 1, Cell: domain.Cell{C: 0, R: 1}},
		{Seat: 2, Cell: domain.Cell{C: 5, R: 0}}, // not walkable: falls back
		{Entity: "thrall", Cell: domain.Cell{C: 11, R: 1}},
	}}
}

func combatMachine(t *testing.T) Machine {
	t.Helper()
	machine, err := NewWithSeed(domain.OneShot{Encounter: domain.Encounter{Battlefield: mapBattlefield()}}, []byte("map"))
	if err != nil {
		t.Fatal(err)
	}
	if err := machine.Goto(vocab.StateCombat); err != nil {
		t.Fatal(err)
	}
	return machine
}

func TestContentCombatConfig_usesTheContentGridAndSpawns(t *testing.T) {
	config := contentCombatConfig(mapBattlefield())
	if config.Grid.Cols != 12 || config.Grid.Rows != 3 || config.Grid.IsWalkable(combat.Cell{X: 5, Y: 0}) {
		t.Fatalf("grid = %+v", config.Grid)
	}
	if config.PCs[0].Position != (combat.Cell{X: 0, Y: 1}) || config.SpawnCell != (combat.Cell{X: 11, Y: 1}) {
		t.Fatalf("spawns = %+v %+v", config.PCs[0].Position, config.SpawnCell)
	}
	if config.PCs[1].Position != (combat.Cell{X: 0, Y: 0}) {
		t.Fatalf("unwalkable spawn fallback = %+v", config.PCs[1].Position)
	}
	placeholder := contentCombatConfig(domain.Battlefield{})
	if placeholder.Grid.Cols != 4 || placeholder.Grid.Rows != 4 {
		t.Fatalf("placeholder grid = %+v", placeholder.Grid)
	}
}

func TestView_combatMapForActiveAndWatchingSeat(t *testing.T) {
	machine := combatMachine(t)
	view := machine.View()
	active, watching := view.Seats[0].CombatMap, view.Seats[1].CombatMap
	if active == nil || watching == nil {
		t.Fatalf("maps = %+v / %+v", active, watching)
	}
	if active.Cols != 12 || active.Rows != 3 || len(active.Walkable) != 35 || active.Me != "pc-1" || watching.Me != "pc-2" {
		t.Fatalf("active map header = %+v", active)
	}
	if len(active.Tokens) != 3 || !active.Tokens[2].Enemy || active.Tokens[2].Cell != (domain.Cell{C: 11, R: 1}) || !active.Tokens[0].Active {
		t.Fatalf("tokens = %+v", active.Tokens)
	}
	if active.MoveLeft != 6 || !active.CanDash || active.WalkStepMS != combat.WalkStepMS || active.DashStepMS != combat.DashStepMS {
		t.Fatalf("active movement = %+v", active)
	}
	normal, dash := 0, 0
	for _, entry := range active.Reach {
		if entry.Dash {
			dash++
		} else {
			normal++
		}
		if len(entry.Path) == 0 || entry.Path[len(entry.Path)-1] != entry.Cell {
			t.Fatalf("reach path does not end on its cell: %+v", entry)
		}
	}
	if normal == 0 || dash == 0 {
		t.Fatalf("reach normal=%d dash=%d", normal, dash)
	}
	if len(watching.Reach) != 0 || watching.CanDash {
		t.Fatalf("watching seat has reach: %+v", watching)
	}
	moves := machine.LegalMoveViews(1)
	if len(moves) != 3 || moves[0].ID != vocab.MoveMove || !moves[0].Enabled {
		t.Fatalf("active moves = %+v", moves)
	}
}

func TestStep_moveAndDashOnTheCombatMap(t *testing.T) {
	machine := combatMachine(t)
	if _, err := machine.Step(domain.Act{Seat: 2, Move: vocab.MoveMove, Cell: domain.Cell{C: 1, R: 1}}); err == nil {
		t.Fatal("the waiting seat moved")
	}
	if _, err := machine.Step(domain.Act{Seat: 1, Move: vocab.MoveMove, Cell: domain.Cell{C: 9, R: 1}}); err == nil {
		t.Fatal("a dash-only cell was accepted as a move")
	}
	if _, err := machine.Step(domain.Act{Seat: 1, Move: vocab.MoveMove, Cell: domain.Cell{C: 2, R: 1}}); err != nil {
		t.Fatal(err)
	}
	view := machine.View()
	if tokens := view.Seats[0].CombatMap.Tokens; tokens[0].Cell != (domain.Cell{C: 2, R: 1}) || tokens[0].Anim != "walk" || len(tokens[0].Path) != 2 {
		t.Fatalf("after move = %+v", tokens[0])
	}
	if view.Seats[0].CombatMap.MoveLeft != 4 {
		t.Fatalf("move left = %d", view.Seats[0].CombatMap.MoveLeft)
	}
	result, err := machine.Step(domain.Act{Seat: 1, Move: vocab.MoveDash, Cell: domain.Cell{C: 10, R: 1}})
	if err != nil {
		t.Fatal(err)
	}
	timer, ok := result.Effects[0].(domain.StartTimer)
	if !ok || timer.Name != dashResolvedTimer || timer.After.Milliseconds() != combat.DashWalkMS(8) {
		t.Fatalf("dash effects = %+v", result.Effects)
	}
	view = machine.View()
	if view.Combat.Tokens[0].StepMS != combat.DashStepMS || view.Seats[0].CombatMap.Tokens[0].Anim != "dash" {
		t.Fatalf("dash pace = %+v / %+v", view.Combat.Tokens[0], view.Seats[0].CombatMap.Tokens[0])
	}
	if machine.combat.Phase != combat.PCTurn || machine.combat.TurnSeat != 1 {
		t.Fatal("the dash ended the turn before the walk played")
	}
	for _, item := range machine.LegalMoveViews(1) {
		if item.ID == vocab.MoveAttack && (item.Enabled || item.Reason != "You dashed this turn") {
			t.Fatalf("attack after a dash = %+v", item)
		}
	}
	if _, err := machine.Step(domain.TimerFired{Name: dashResolvedTimer}); err != nil {
		t.Fatal(err)
	}
	if machine.combat.TurnSeat != 2 || machine.combat.Phase != combat.PCTurn {
		t.Fatalf("after the dash walk: seat %d phase %s", machine.combat.TurnSeat, machine.combat.Phase)
	}
	if _, err := machine.Step(domain.TimerFired{Name: dashResolvedTimer}); err != nil {
		t.Fatal("a stale dash timer was rejected")
	}
	if machine.combat.TurnSeat != 2 {
		t.Fatal("a stale dash timer ended the next turn")
	}
}

func TestCombatMoveUI_flagOff(t *testing.T) {
	machine := combatMachine(t)
	machine.ConfigureCombatMoveUI(false)
	if machine.CombatMoveUI() {
		t.Fatal("flag still on")
	}
	for _, item := range machine.LegalMoveViews(1) {
		if item.ID == vocab.MoveMove {
			t.Fatal("Move offered with the flag off")
		}
	}
	if view := machine.View(); view.Seats[0].CombatMap != nil {
		t.Fatal("map sent with the flag off")
	}
	if _, err := machine.Step(domain.Act{Seat: 1, Move: vocab.MoveMove, Cell: domain.Cell{C: 1, R: 1}}); err == nil {
		t.Fatal("move accepted with the flag off")
	}
}
