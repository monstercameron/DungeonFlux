package game

import (
	"testing"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

func TestBattlefieldReports_Mode(t *testing.T) {
	tests := []struct {
		name    string
		reports BattlefieldReports
		want    string
	}{
		{name: "ready", reports: BattlefieldReports{SplatReady: true}, want: BattlefieldModeSplat},
		{name: "failed", reports: BattlefieldReports{SplatReady: true, SplatFailed: true}, want: BattlefieldModeFlat},
		{name: "missing", reports: BattlefieldReports{}, want: BattlefieldModeFlat},
		{name: "host disabled", reports: BattlefieldReports{SplatReady: true, SplatOff: true}, want: BattlefieldModeFlat},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := test.reports.Mode(); got != test.want {
				t.Fatalf("mode = %q, want %q", got, test.want)
			}
		})
	}
}

func TestBattlefieldReports_ReportLatchesFailure(t *testing.T) {
	var reports BattlefieldReports
	reports.Report(vocab.ReportSplatReady)
	reports.Report(vocab.ReportSplatFailed)
	reports.Report(vocab.ReportSplatReady)
	if got := reports.Mode(); got != BattlefieldModeFlat {
		t.Fatalf("mode after repeated ready = %q, want %q", got, BattlefieldModeFlat)
	}

	reports = BattlefieldReports{}
	reports.Report(vocab.ReportSplatReady)
	reports.Disable()
	if got := reports.Mode(); got != BattlefieldModeFlat {
		t.Fatalf("mode after disable = %q, want %q", got, BattlefieldModeFlat)
	}
}

func TestBattlefieldViewFrom_ProjectsOpeningBattlefield(t *testing.T) {
	source := domain.Battlefield{
		SceneURL: "scene.sog",
		LiteURL:  "lite.sog",
		Grid:     domain.Grid{Cols: 2, Rows: 1, Walkable: []bool{true, false}},
		Cameras:  map[string]domain.CameraDef{"COMBAT_EST": {FOV: 35}},
		Flat:     domain.FlatBattlefield{ImageURL: "tavern.png"},
	}
	view := BattlefieldViewFrom(source, BattlefieldReports{SplatReady: true}, false)
	if view.Mode != BattlefieldModeSplat || view.Visible {
		t.Fatalf("opening view = %#v", view)
	}
	if view.SceneURL != source.SceneURL || view.LiteURL != source.LiteURL || view.Flat.ImageURL != source.Flat.ImageURL {
		t.Fatalf("navigation data was not projected: %#v", view)
	}
	view.Grid.Walkable[0] = false
	view.Cameras["COMBAT_EST"] = domain.CameraDef{FOV: 90}
	if !source.Grid.Walkable[0] || source.Cameras["COMBAT_EST"].FOV != 35 {
		t.Fatal("projected battlefield shares mutable navigation data")
	}

	combat := BattlefieldViewFrom(source, BattlefieldReports{}, true)
	if combat.Mode != BattlefieldModeFlat || !combat.Visible {
		t.Fatalf("combat fallback view = %#v", combat)
	}
}
