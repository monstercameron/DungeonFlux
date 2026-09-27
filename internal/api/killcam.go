package api

import (
	df "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
	"github.com/monstercameron/DungeonFlux/internal/domain"
)

func projectKillcam(view domain.KillCamView) *df.KillCam {
	if view.URL == "" {
		return nil
	}
	return &df.KillCam{Url: view.URL, Sequence: view.Sequence, Outcome: view.Outcome, Attacker: view.Attacker, Victim: view.Victim, OffsetMs: view.OffsetMS, DurationMs: view.DurationMS, Playing: view.Playing}
}
