package api

import (
	"github.com/monstercameron/DungeonFlux/internal/domain"
	"testing"
)

func TestProjectKillcam(t *testing.T) {
	if got := ProjectDMWithLobby(domain.View{}, LobbyProjection{}).GetKillCam(); got != nil {
		t.Fatal("empty cinematic projected")
	}
	view := domain.View{KillCam: domain.KillCamView{URL: "/assets/finisher.mp4", Sequence: 2, Outcome: "defeat", Attacker: "Thrall", Victim: "Hero", OffsetMS: 1000, DurationMS: 4000, Playing: false}}
	got := ProjectDMWithLobby(view, LobbyProjection{}).GetKillCam()
	if got.GetUrl() != view.KillCam.URL || got.GetSequence() != 2 || got.GetOffsetMs() != 1000 || got.GetDurationMs() != 4000 || got.GetPlaying() || got.GetAttacker() != "Thrall" || got.GetVictim() != "Hero" || got.GetOutcome() != "defeat" {
		t.Fatalf("projection = %v", got)
	}
}
