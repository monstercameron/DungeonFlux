package domain

import (
	"encoding/json"
	"testing"
)

func TestOneShotRoundTripKeepsEncounterNavigation(t *testing.T) {
	original := OneShot{ID: "demo", Encounter: Encounter{Enemy: Creature{ID: "thrall", HP: 20}, Trigger: "after_stranger_line", Battlefield: Battlefield{Mode: "FLAT", Grid: Grid{CellM: 1.524, Cols: 8, Rows: 6, Walkable: make([]bool, 48)}, Spawns: []Spawn{{Seat: 1, Cell: Cell{C: 1, R: 1}}}}}, Music: MusicCatalogue{Tracks: []MusicTrack{{ID: "combat", BPM: 160}}}}
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatal(err)
	}
	var got OneShot
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if got.Encounter.Battlefield.Grid.Cols != 8 || got.Encounter.Battlefield.Grid.Rows != 6 || len(got.Encounter.Battlefield.Grid.Walkable) != 48 || got.Music.Tracks[0].BPM != 160 {
		t.Fatalf("encounter round trip mismatch: %#v", got)
	}
}
