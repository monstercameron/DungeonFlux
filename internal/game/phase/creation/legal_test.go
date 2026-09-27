package creation

import (
	"testing"

	"github.com/monstercameron/DungeonFlux/internal/game/rules"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

func TestLegalMoveViews_CreationSeatStates(t *testing.T) {
	tests := []struct {
		name    string
		state   SeatState
		enabled []vocab.MoveID
		reasons []string
	}{
		{
			name:    "before either choice",
			state:   SeatState{},
			enabled: []vocab.MoveID{vocab.MoveSpecies, vocab.MoveGender, vocab.MoveClass},
			reasons: []string{"", "", "", reasonChooseIdentity, reasonRenameFirst, reasonReady},
		},
		{
			name:  "species only",
			state: SeatState{Species: "elf"},
			enabled: []vocab.MoveID{
				vocab.MoveSpecies, vocab.MoveGender, vocab.MoveClass,
			},
			reasons: []string{"", "", "", reasonChooseIdentity, reasonRenameFirst, reasonReady},
		},
		{
			name:    "all choices",
			state:   SeatState{Species: "elf", Gender: "female", Class: rules.Wizard},
			enabled: []vocab.MoveID{vocab.MoveSpecies, vocab.MoveGender, vocab.MoveClass, vocab.MoveRollHero},
			reasons: []string{"", "", "", "", reasonRenameFirst, reasonReady},
		},
		{
			name:    "rolled",
			state:   SeatState{Species: "elf", Gender: "female", Built: true},
			enabled: []vocab.MoveID{vocab.MoveRename, vocab.MoveReady},
			reasons: []string{reasonRollHero, reasonRollHero, reasonRollHero, reasonRollHero, "", ""},
		},
		{
			name:    "locked",
			state:   SeatState{Species: "elf", Gender: "female", Built: true, Locked: true},
			enabled: nil,
			reasons: []string{reasonLocked, reasonLocked, reasonLocked, reasonLocked, reasonLocked, reasonLocked},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			moves := LegalMoveViews(test.state)
			if len(moves) != 6 {
				t.Fatalf("moves = %#v", moves)
			}
			wantEnabled := make(map[vocab.MoveID]bool, len(test.enabled))
			for _, id := range test.enabled {
				wantEnabled[id] = true
			}
			for index, move := range moves {
				if move.Enabled != wantEnabled[move.ID] || move.Reason != test.reasons[index] {
					t.Fatalf("move %d = %#v, want enabled=%t reason=%q", index, move, wantEnabled[move.ID], test.reasons[index])
				}
			}
		})
	}
}

func TestLegalMoveViews_UsesStableMoveOrder(t *testing.T) {
	moves := LegalMoveViews(SeatState{Species: "human", Gender: "male"})
	want := []vocab.MoveID{vocab.MoveSpecies, vocab.MoveGender, vocab.MoveClass, vocab.MoveRollHero, vocab.MoveRename, vocab.MoveReady}
	for index, move := range moves {
		if move.ID != want[index] {
			t.Fatalf("move %d = %q, want %q", index, move.ID, want[index])
		}
	}
	if len(moves[2].Options) != len(rules.Classes()) {
		t.Fatalf("class options = %#v", moves[2].Options)
	}
}
