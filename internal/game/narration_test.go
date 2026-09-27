package game

import (
	"testing"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

func TestStateNarration_StoresDeltasAndFinalState(t *testing.T) {
	state := New(domain.OneShot{}, []byte("narration"))
	state.beginNarration("line-1", "Dungeon Master", string(vocab.RoleOpening))
	state.Step(domain.Envelope{Event: domain.NarrationDelta{
		UtteranceID: "line-1", Speaker: "Dungeon Master", Text: "Rain ", TextSoFar: "Rain ",
	}})
	state.Step(domain.Envelope{Event: domain.NarrationDelta{
		UtteranceID: "line-1", Speaker: "Dungeon Master", Text: "falls.", TextSoFar: "Rain falls.", Final: true,
	}})

	view := state.View()
	if view.Scene.Narration != "Rain falls." || view.Scene.NarrationSpeaker != "Dungeon Master" || view.Scene.NarrationLineID != "line-1" || !view.Scene.NarrationDone {
		t.Fatalf("narration view = %#v", view.Scene)
	}
	state.Step(domain.Envelope{Event: domain.LineDone{UtteranceID: "line-1"}})
	if !state.View().Scene.NarrationDone {
		t.Fatal("line completion cleared the final flag")
	}
}

func TestStateNarration_StaleDeltaIsIgnoredAndPhaseChangeClears(t *testing.T) {
	state := New(domain.OneShot{}, []byte("narration"))
	state.beginNarration("line-1", "Mother Vell", string(vocab.RoleNPCReply))
	state.Step(domain.Envelope{Event: domain.NarrationDelta{UtteranceID: "line-2", Speaker: "Stranger", TextSoFar: "stale"}})
	if got := state.View().Scene.Narration; got != "" {
		t.Fatalf("stale line text = %q", got)
	}
	state.beginNarration("line-1", "Mother Vell", string(vocab.RoleNPCReply))
	state.Step(domain.Envelope{Event: domain.NarrationDelta{UtteranceID: "line-1", TextSoFar: "before transition"}})
	readyLobby(t, state)
	state.Step(domain.Envelope{Event: domain.HostCmd{Cmd: vocab.HostStart}})
	if got := state.View().Scene.Narration; got != "" {
		t.Fatalf("phase transition retained narration %q", got)
	}
}
