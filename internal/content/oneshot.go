package content

import (
	"fmt"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

// OneShot is the fixed Drowned Lantern story together with its logical asset
// catalogue. Asset IDs are resolved to files by the composition root.
type OneShot struct {
	domain.OneShot
	Catalogue []domain.Asset
}

// DefaultOneShot returns the build-time content for the demo.
func DefaultOneShot() OneShot {
	return OneShot{
		OneShot: domain.OneShot{
			ID:    "drowned_lantern",
			Title: "The Drowned Lantern",
			NPCs: []domain.NPC{
				{ID: "mother_vell", Name: "Mother Vell", Personality: "Gravelly, amused, protective of her regulars; evasive until persuaded.", VoiceID: "voice_mother_vell"},
				{ID: "courier", Name: "The Courier", Personality: "A hooded courier dripping river water and carrying a sealed letter.", VoiceID: "voice_courier"},
			},
			Beats: []domain.Beat{
				{ID: "opening", Text: "Rain hammers the Drowned Lantern as the lamplighter's disappearance brings two travellers to Mother Vell's bar.", Leads: []string{"conversation"}},
				{ID: "conversation", Text: "Mother Vell knows the lamplighter was dragged toward the old bell tower, but she does not answer questions for free.", Leads: []string{"stranger"}},
				{ID: "stranger", Text: "A dripping courier delivers a sealed letter. Whatever was in the water followed him from the river.", Leads: []string{"combat"}},
				{ID: "combat", Text: "A drowned thrall bound to the tower bell bursts through the tavern door.", Leads: []string{"cliffhanger"}},
				{ID: "cliffhanger", Text: "The tower bell tolls midnight, every lantern dies, and whoever rang it knows the heroes' names."},
			},
			Encounter: domain.Encounter{
				Enemy:   domain.Creature{ID: "thrall", Name: "Drowned Thrall", HP: 12, MaxHP: 12, AC: 8, Cell: domain.Cell{C: 6, R: 3}},
				Trigger: "after_stranger_line",
				Battlefield: domain.Battlefield{
					Mode:     "FLAT",
					SceneURL: "battlefield_tavern_splat",
					LiteURL:  "battlefield_tavern_lite",
					Grid: domain.Grid{Origin: [2]float64{0, 0}, CellM: 1.524, Cols: 8, Rows: 6, Walkable: []bool{
						true, true, true, true, true, true, true, true,
						true, true, true, true, true, true, true, true,
						true, true, false, false, false, true, true, true,
						true, true, true, true, true, true, true, true,
						true, true, false, false, true, true, true, true,
						true, true, true, true, true, true, true, true}},
					Spawns: []domain.Spawn{
						{Seat: 1, Cell: domain.Cell{C: 1, R: 4}},
						{Seat: 2, Cell: domain.Cell{C: 1, R: 5}},
						{Entity: "thrall", Cell: domain.Cell{C: 6, R: 3}},
					},
					Door: domain.Cell{C: 0, R: 3},
					Flat: domain.FlatBattlefield{ImageURL: "battlefield_tavern_flat"},
				},
				Loops: map[string]domain.Asset{
					"idle":   {ID: "thrall_loop_idle", Kind: string(vocab.AssetVideo)},
					"attack": {ID: "thrall_loop_attack", Kind: string(vocab.AssetVideo)},
					"hit":    {ID: "thrall_loop_hit", Kind: string(vocab.AssetVideo)},
					"fall":   {ID: "thrall_loop_fall", Kind: string(vocab.AssetVideo)},
				},
			},
			Music: domain.MusicCatalogue{Tracks: []domain.MusicTrack{
				{ID: "theme_drowned_lantern", Asset: "music_theme_drowned_lantern", LoopStartMS: 1200, LoopEndMS: 48100, BPM: 72, Level: 0.55, Duck: 0.18},
				{ID: "combat_thrall", Asset: "music_combat_thrall", LoopStartMS: 800, LoopEndMS: 36200, BPM: 160, Level: 0.62, Duck: 0.16},
			}},
		},
		Catalogue: defaultCatalogue(),
	}
}

func defaultCatalogue() []domain.Asset {
	ids := []struct {
		id   domain.AssetID
		kind vocab.AssetKind
	}{
		{"battlefield_tavern_splat", vocab.AssetSplat}, {"battlefield_tavern_lite", vocab.AssetSplat},
		{"battlefield_tavern_flat", vocab.AssetImage}, {"thrall_loop_idle", vocab.AssetVideo},
		{"thrall_loop_attack", vocab.AssetVideo}, {"thrall_loop_hit", vocab.AssetVideo}, {"thrall_loop_fall", vocab.AssetVideo},
		{"music_theme_drowned_lantern", vocab.AssetMusic}, {"music_combat_thrall", vocab.AssetMusic},
		{"establishing_tavern", vocab.AssetVideo}, {"arrival_door", vocab.AssetVideo},
	}
	assets := make([]domain.Asset, 0, len(ids))
	for _, item := range ids {
		assets = append(assets, domain.Asset{ID: item.id, Kind: string(item.kind)})
	}
	return assets
}

// Validate checks story structure and that every logical asset reference is
// present in the catalogue.
func (s OneShot) Validate() error {
	if s.ID == "" || s.Title == "" {
		return fmt.Errorf("one-shot identity is incomplete")
	}
	if len(s.NPCs) != 2 || s.NPCs[0].ID == "" || s.NPCs[1].ID == "" {
		return fmt.Errorf("one-shot must define Mother Vell and the courier")
	}
	if len(s.Beats) < 5 {
		return fmt.Errorf("one-shot needs opening through cliffhanger beats")
	}
	if s.Encounter.Enemy.ID == "" || s.Encounter.Enemy.HP != 12 || s.Encounter.Enemy.AC != 8 {
		return fmt.Errorf("drowned thrall stat block is invalid")
	}
	if s.Encounter.Trigger != "after_stranger_line" {
		return fmt.Errorf("unexpected encounter trigger %q", s.Encounter.Trigger)
	}
	if err := validateBattlefield(s.Encounter.Battlefield); err != nil {
		return err
	}
	known := make(map[domain.AssetID]bool, len(s.Catalogue))
	for _, asset := range s.Catalogue {
		if asset.ID == "" || known[asset.ID] {
			return fmt.Errorf("invalid or duplicate catalogue asset %q", asset.ID)
		}
		known[asset.ID] = true
	}
	for name, asset := range s.Encounter.Loops {
		if !known[asset.ID] {
			return fmt.Errorf("loop %q references missing asset %q", name, asset.ID)
		}
	}
	if !known[domain.AssetID(s.Encounter.Battlefield.SceneURL)] || !known[domain.AssetID(s.Encounter.Battlefield.LiteURL)] || !known[domain.AssetID(s.Encounter.Battlefield.Flat.ImageURL)] {
		return fmt.Errorf("battlefield references an uncatalogued asset")
	}
	for _, track := range s.Music.Tracks {
		if !known[track.Asset] {
			return fmt.Errorf("music track %q references missing asset %q", track.ID, track.Asset)
		}
		if track.LoopEndMS <= track.LoopStartMS || track.BPM <= 0 {
			return fmt.Errorf("music track %q has invalid loop or BPM", track.ID)
		}
	}
	return nil
}

func validateBattlefield(field domain.Battlefield) error {
	if field.Mode != "FLAT" || field.Grid.Cols <= 0 || field.Grid.Rows <= 0 || len(field.Grid.Walkable) != field.Grid.Cols*field.Grid.Rows {
		return fmt.Errorf("battlefield grid is invalid")
	}
	if field.Grid.CellM <= 0 {
		return fmt.Errorf("battlefield cell size must be positive")
	}
	return nil
}
