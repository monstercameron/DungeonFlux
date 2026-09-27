package combat

import (
	"testing"

	"github.com/monstercameron/DungeonFlux/internal/game/rules"
	"github.com/monstercameron/DungeonFlux/internal/game/rules/dice"
)

func TestPresentation_startMoveAndAttack(t *testing.T) {
	c := testConfig()
	c.Grid = Grid{Cols: 4, Rows: 4}
	c.PCs[0].Position = Cell{X: 3, Y: 3}
	s, err := NewState(c)
	if err != nil {
		t.Fatal(err)
	}
	if got := s.Presentation.Camera.Preset; got != "COMBAT_EST" {
		t.Fatalf("intro camera = %q", got)
	}
	if err := s.Start(); err != nil {
		t.Fatal(err)
	}
	if got := s.Presentation.Camera.FocusTokenID; got != "pc-1" {
		t.Fatalf("turn focus = %q", got)
	}
	if len(s.Presentation.Highlights) != 1 || s.Presentation.Highlights[0].Kind != "reach" {
		t.Fatalf("reach highlights = %#v", s.Presentation.Highlights)
	}
	moved, err := s.Move(Cell{X: 2, Y: 2})
	if err != nil || len(moved.Path) != 1 {
		t.Fatalf("move = %#v, %v", moved, err)
	}
	visual := s.Presentation.Tokens["pc-1"]
	if visual.Anim != "walk" || len(visual.Path) != 1 {
		t.Fatalf("move visual = %#v", visual)
	}
	roller := dice.New([]byte("presentation-attack"))
	if err := roller.ForceD20(20); err != nil {
		t.Fatal(err)
	}
	attack, err := s.Attack(roller, "thrall")
	if err != nil || !attack.Outcome.Hit {
		t.Fatalf("attack = %#v, %v", attack, err)
	}
	if s.Presentation.Tokens["pc-1"].Anim != "attack" || s.Presentation.Camera.Preset != "IMPACT" {
		t.Fatalf("attack presentation = %#v", s.Presentation)
	}
	if s.Presentation.ContactMS != 1200 || s.Presentation.ContactTotalMS != 2000 || s.Presentation.Shake.Seq != 1 {
		t.Fatalf("impact timing = %#v", s.Presentation)
	}
}

func TestPresentation_enemyDownAndTransitions(t *testing.T) {
	c := testConfig()
	c.Grid = Grid{Cols: 8, Rows: 8}
	c.PCs[0].Position = Cell{X: 2, Y: 0}
	c.PCs[1].Position = Cell{X: 5, Y: 5}
	s, err := NewState(c)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Start(); err != nil {
		t.Fatal(err)
	}
	if err := s.EndPlayerTurn(); err != nil {
		t.Fatal(err)
	}
	if s.Presentation.Camera.FocusTokenID != "thrall" || len(s.Presentation.Highlights) != 0 {
		t.Fatalf("enemy transition = %#v", s.Presentation)
	}
	roller := dice.New([]byte("presentation-enemy"))
	if err := roller.ForceD20(20); err != nil {
		t.Fatal(err)
	}
	result, err := s.EnemyTurn(roller, 1200)
	if err != nil || !result.Outcome.Hit {
		t.Fatalf("enemy = %#v, %v", result, err)
	}
	if s.Presentation.Tokens["thrall"].Anim != "walk" || s.Presentation.Tokens["pc-1"].Anim != "down" {
		t.Fatalf("enemy animation = %#v", s.Presentation.Tokens)
	}
	if s.Presentation.ContactMS != 1200 || s.Presentation.Camera.Preset != "KO" {
		t.Fatalf("enemy impact = %#v", s.Presentation)
	}
	if err := s.EndEnemyTurn(); err != nil {
		t.Fatal(err)
	}
	if s.Presentation.Camera.FocusTokenID != "pc-2" || len(s.Presentation.Highlights) == 0 {
		t.Fatalf("next player transition = %#v", s.Presentation)
	}
}

func TestPresentation_terminalReasonsAndDown(t *testing.T) {
	tests := []struct {
		name   string
		reason EndReason
	}{
		{name: "cap", reason: ReasonCap},
		{name: "bell", reason: ReasonBell},
		{name: "skip", reason: ReasonSkip},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			s, err := NewState(testConfig())
			if err != nil {
				t.Fatal(err)
			}
			if err := s.Start(); err != nil {
				t.Fatal(err)
			}
			if _, err := s.ResolveEnd(test.reason, 0); err != nil {
				t.Fatal(err)
			}
			if s.Phase != Done || s.Presentation.Tokens["thrall"].Anim != "flee" {
				t.Fatalf("terminal presentation = %#v", s.Presentation)
			}
		})
	}

	s, err := NewState(testConfig())
	if err != nil {
		t.Fatal(err)
	}
	s.Phase = Rolling
	s.Thrall.HP = 0
	s.PCs[0].HP = 0
	s.PCs[0].Conditions = []rules.Condition{rules.Down, rules.Prone, rules.Defeated}
	result, err := s.ResolveEnd(ReasonCap, 1)
	if err != nil || result.Outcome != Slain || s.Presentation.Tokens["thrall"].Anim != "fall" {
		t.Fatalf("slain terminal = %#v, %#v", result, s.Presentation)
	}
	pc, ok := s.Participant(1)
	if !ok || pc.HP != 1 {
		t.Fatalf("down PC was not stabilized = %#v", pc)
	}
}

func TestPresentation_skipIsIdempotent(t *testing.T) {
	s, err := NewState(testConfig())
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Skip(); err != nil {
		t.Fatal(err)
	}
	if err := s.Skip(); err != nil || s.Presentation.Tokens["thrall"].Anim != "flee" {
		t.Fatalf("skip = %v, %#v", err, s.Presentation)
	}
}

func TestEnemyTurn_usesAndUpdatesThrallSpawnCell(t *testing.T) {
	c := testConfig()
	c.Grid = Grid{Cols: 8, Rows: 8}
	c.SpawnCell = Cell{X: 3, Y: 3}
	c.PCs[0].Position = Cell{X: 5, Y: 3}
	c.PCs[1].Position = Cell{X: 7, Y: 7}
	s, err := NewState(c)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Start(); err != nil {
		t.Fatal(err)
	}
	if err := s.EndPlayerTurn(); err != nil {
		t.Fatal(err)
	}
	result, err := s.EnemyTurn(dice.New([]byte("spawn-cell")), 900)
	if err != nil || len(result.Path) != 1 {
		t.Fatalf("enemy approach = %#v, %v", result, err)
	}
	if !adjacent(s.ThrallCell(), c.PCs[0].Position) || s.ThrallCell() == c.SpawnCell {
		t.Fatalf("thrall cell = %#v", s.ThrallCell())
	}
}
