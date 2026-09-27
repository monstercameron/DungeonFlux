package content

import (
	"testing"

	"github.com/monstercameron/DungeonFlux/internal/domain"
)

func TestDefaultOneShot_Validates(t *testing.T) {
	story := DefaultOneShot()
	if err := story.Validate(); err != nil {
		t.Fatalf("default one-shot should validate: %v", err)
	}
	if story.Encounter.Enemy.Name != "Drowned Thrall" || len(story.Encounter.Loops) != 4 {
		t.Fatalf("default encounter is incomplete: %#v", story.Encounter)
	}
}

func TestOneShotValidate_RejectsMissingLogicalAsset(t *testing.T) {
	story := DefaultOneShot()
	story.Encounter.Battlefield.SceneURL = "missing"
	if err := story.Validate(); err == nil {
		t.Fatal("missing battlefield asset should be rejected")
	}
}

func TestOneShotValidate_RejectsMalformedInputs(t *testing.T) {
	tests := []struct {
		name string
		edit func(*OneShot)
	}{
		{"identity", func(s *OneShot) { s.ID = "" }},
		{"beats", func(s *OneShot) { s.Beats = nil }},
		{"thrall", func(s *OneShot) { s.Encounter.Enemy.HP = 15 }},
		{"trigger", func(s *OneShot) { s.Encounter.Trigger = "never" }},
		{"grid", func(s *OneShot) { s.Encounter.Battlefield.Grid.Walkable = nil }},
		{"cell size", func(s *OneShot) { s.Encounter.Battlefield.Grid.CellM = 0 }},
		{"duplicate asset", func(s *OneShot) { s.Catalogue = append(s.Catalogue, s.Catalogue[0]) }},
		{"loop asset", func(s *OneShot) { s.Encounter.Loops["idle"] = domain.Asset{ID: "missing"} }},
		{"music asset", func(s *OneShot) { s.Music.Tracks[0].Asset = "missing" }},
		{"music loop", func(s *OneShot) { s.Music.Tracks[0].LoopEndMS = 1 }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			story := DefaultOneShot()
			test.edit(&story)
			if err := story.Validate(); err == nil {
				t.Fatal("Validate accepted malformed content")
			}
		})
	}
}

func TestOneShotValidate_RejectsUnwalkableSpawn(t *testing.T) {
	story := DefaultOneShot()
	story.Encounter.Battlefield.Spawns[0].Cell = domain.Cell{C: 10, R: 0}
	if err := story.Validate(); err == nil {
		t.Fatal("Validate accepted a spawn on blocked terrain")
	}
}

func TestOneShotValidate_RequiresThrallDistanceFromSeatOne(t *testing.T) {
	story := DefaultOneShot()
	story.Encounter.Battlefield.Spawns[2].Cell = domain.Cell{C: 9, R: 5}
	if err := story.Validate(); err == nil {
		t.Fatal("Validate accepted a thrall spawn too close to seat one")
	}
}
