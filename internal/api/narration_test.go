package api

import (
	"testing"

	df "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
	"github.com/monstercameron/DungeonFlux/internal/domain"
)

func TestProject_NarrationReachesDMAndPhone(t *testing.T) {
	view := domain.View{Scene: domain.SceneView{
		Narration: "The bell waits.", NarrationSpeaker: "Dungeon Master",
		NarrationLineID: "opening", NarrationDone: true,
	}}
	dm := ProjectDM(view).GetNarration()
	phone := ProjectPhone(view, 1).GetNarration()
	for name, narration := range map[string]*df.Narration{"dm": dm, "phone": phone} {
		t.Run(name, func(t *testing.T) {
			if narration == nil || narration.GetSpeaker() != "Dungeon Master" || narration.GetTextSoFar() != "The bell waits." || narration.GetLineId() != "opening" || !narration.GetDone() {
				t.Fatalf("narration = %#v", narration)
			}
		})
	}
}
