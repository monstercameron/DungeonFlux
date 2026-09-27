package phase

import (
	"testing"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

// A debug goto must not replay the lines of the phases it skips: going to
// combat used to start the opening narration and the reveal while the TV
// already showed combat, so every goto replayed "Rain hammers the Drowned
// Lantern" over the fight.
func TestDebugGoto_OnlyTargetPhaseLinesPlay(t *testing.T) {
	tests := []struct {
		target vocab.StateID
		want   []domain.UtteranceID
	}{
		{vocab.StateOpening, []domain.UtteranceID{"opening"}},
		{vocab.StateExploration, nil},
		{vocab.StateResolution, []domain.UtteranceID{"reveal"}},
		{vocab.StateCombat, nil},
	}
	for _, tc := range tests {
		t.Run(string(tc.target), func(t *testing.T) {
			machine, err := NewWithSeed(domain.OneShot{}, []byte("debug-voice"))
			if err != nil {
				t.Fatal(err)
			}
			result, err := machine.DebugGoto(tc.target)
			if err != nil {
				t.Fatal(err)
			}
			var got []domain.UtteranceID
			for _, effect := range result.Effects {
				switch value := effect.(type) {
				case domain.StartLine:
					got = append(got, value.UtteranceID)
				case domain.PlayCanned:
					got = append(got, value.UtteranceID)
				}
			}
			if len(got) != len(tc.want) {
				t.Fatalf("lines = %v, want %v", got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Fatalf("lines = %v, want %v", got, tc.want)
				}
			}
		})
	}
}
