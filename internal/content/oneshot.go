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
				Enemy:       domain.Creature{ID: "thrall", Name: "Drowned Thrall", HP: 12, MaxHP: 12, AC: 8, Cell: domain.Cell{C: 2, R: 4}},
				Trigger:     "after_stranger_line",
				Battlefield: woodedPathBattlefield(),
				Loops: map[string]domain.Asset{
					"idle":   {ID: "thrall_loop_idle", Kind: string(vocab.AssetVideo)},
					"attack": {ID: "thrall_loop_attack", Kind: string(vocab.AssetVideo)},
					"hit":    {ID: "thrall_loop_hit", Kind: string(vocab.AssetVideo)},
					"fall":   {ID: "thrall_loop_fall", Kind: string(vocab.AssetVideo)},
				},
			},
			Music: domain.MusicCatalogue{Tracks: []domain.MusicTrack{
				{ID: "THEME_MAIN", Asset: "THEME_MAIN", LoopStartMS: 1200, LoopEndMS: 48100, BPM: 80, Level: 0.6, Duck: 0.18},
				{ID: "COMBAT_SKIRMISH_LOOP", Asset: "COMBAT_SKIRMISH_LOOP", LoopStartMS: 800, LoopEndMS: 36200, BPM: 160, Level: 0.5, Duck: 0.16},
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
		{"/splat/scenes/64bb46d5.json", vocab.AssetSplat},
		{"cb2fddd6", vocab.AssetSplat},
		{"level_still_64bb46d5_tactical", vocab.AssetImage}, {"battlefield_tavern_flat", vocab.AssetImage}, {"thrall_loop_idle", vocab.AssetVideo},
		{"thrall_loop_attack", vocab.AssetVideo}, {"thrall_loop_hit", vocab.AssetVideo}, {"thrall_loop_fall", vocab.AssetVideo},
		{"THEME_MAIN", vocab.AssetMusic}, {"COMBAT_SKIRMISH_LOOP", vocab.AssetMusic},
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
	if (field.Mode != "FLAT" && field.Mode != "SPLAT") || field.Grid.Cols <= 0 || field.Grid.Rows <= 0 || len(field.Grid.Walkable) != field.Grid.Cols*field.Grid.Rows {
		return fmt.Errorf("battlefield grid is invalid")
	}
	if field.Grid.CellM <= 0 {
		return fmt.Errorf("battlefield cell size must be positive")
	}
	if err := validateSpawns(field); err != nil {
		return err
	}
	return nil
}

func validateSpawns(field domain.Battlefield) error {
	seatCells := make(map[domain.SeatID]domain.Cell, 2)
	var thrall domain.Cell
	foundThrall := false
	for _, spawn := range field.Spawns {
		if !walkable(field.Grid, spawn.Cell) {
			return fmt.Errorf("spawn at (%d,%d) is not walkable", spawn.Cell.C, spawn.Cell.R)
		}
		switch {
		case spawn.Seat == 1 || spawn.Seat == 2:
			if _, exists := seatCells[spawn.Seat]; exists {
				return fmt.Errorf("duplicate seat %d spawn", spawn.Seat)
			}
			seatCells[spawn.Seat] = spawn.Cell
		case spawn.Entity == "thrall":
			if foundThrall {
				return fmt.Errorf("duplicate thrall spawn")
			}
			thrall, foundThrall = spawn.Cell, true
		default:
			return fmt.Errorf("spawn has no supported seat or entity")
		}
	}
	if len(seatCells) != 2 || !foundThrall {
		return fmt.Errorf("battlefield needs two seat spawns and one thrall spawn")
	}
	distance, ok := spawnDistance(field.Grid, seatCells[1], thrall)
	if !ok || distance < 4 || distance > 6 {
		return fmt.Errorf("thrall must be 4-6 walkable cells from seat 1")
	}
	return nil
}

func walkable(grid domain.Grid, cell domain.Cell) bool {
	if cell.C < 0 || cell.R < 0 || cell.C >= grid.Cols || cell.R >= grid.Rows {
		return false
	}
	return grid.Walkable[cell.R*grid.Cols+cell.C]
}

func spawnDistance(grid domain.Grid, from, to domain.Cell) (int, bool) {
	if !walkable(grid, from) || !walkable(grid, to) {
		return 0, false
	}
	type node struct {
		cell domain.Cell
		dist int
	}
	queue := []node{{cell: from}}
	seen := map[domain.Cell]bool{from: true}
	for head := 0; head < len(queue); head++ {
		current := queue[head]
		if current.cell == to {
			return current.dist, true
		}
		for row := -1; row <= 1; row++ {
			for column := -1; column <= 1; column++ {
				if row == 0 && column == 0 {
					continue
				}
				next := domain.Cell{C: current.cell.C + column, R: current.cell.R + row}
				if seen[next] || !walkable(grid, next) {
					continue
				}
				seen[next] = true
				queue = append(queue, node{cell: next, dist: current.dist + 1})
			}
		}
	}
	return 0, false
}
