package phone

import "testing"

func TestNewExploreSnapshot_UsesStatusAsNarrationAndSceneDefaults(t *testing.T) {
	got := NewExploreSnapshot(MovesSnapshot{StatusText: "The door groans.", Locale: "en", Moves: []MoveSnapshot{{ID: "leave", Label: "Leave", Enabled: true}}})
	if got.Title != "The Drowned Lantern" || got.Location != "The Forgotten Depths" || got.SceneArt != "tavern_interior" || got.Narration != "The door groans." || got.Locale != "en" || len(got.Moves) != 1 {
		t.Fatalf("snapshot = %+v", got)
	}
}

func TestNewExploreSnapshot_UsesCannedNarrationWhenStatusIsEmpty(t *testing.T) {
	got := NewExploreSnapshot(MovesSnapshot{})
	if got.Narration == "" || got.Narration == "The Drowned Lantern" {
		t.Fatalf("narration = %q", got.Narration)
	}
}

func TestIsExplorationMoves_RecognizesSceneAndConversationTransitions(t *testing.T) {
	tests := []struct {
		name  string
		moves []MoveSnapshot
		want  bool
	}{
		{name: "talk", moves: []MoveSnapshot{{ID: "talk_vell"}}, want: true},
		{name: "step away", moves: []MoveSnapshot{{ID: "step_away"}}, want: true},
		{name: "investigate", moves: []MoveSnapshot{{ID: "investigate"}}, want: true},
		{name: "creation", moves: []MoveSnapshot{{ID: "species"}}, want: false},
		{name: "empty", want: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := IsExplorationMoves(test.moves); got != test.want {
				t.Fatalf("IsExplorationMoves() = %v, want %v", got, test.want)
			}
		})
	}
}
