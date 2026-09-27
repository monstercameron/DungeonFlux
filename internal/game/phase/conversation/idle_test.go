package conversation

import (
	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
	"testing"
)

func TestIdleStep_IdleElapsedNudgesAndSelectsSoftestRung(t *testing.T) {
	result := IdleStep(IdleState{Seat: 2, Intent: IntentStalled}, domain.TimerFired{Name: idleTimerName})
	if !result.State.IdleElapsed || !result.State.NudgeInFlight || result.Rung != RungLure {
		t.Fatalf("result = %#v", result)
	}
	if len(result.Effects) != 1 {
		t.Fatalf("effects = %#v", result.Effects)
	}
	if line, ok := result.Effects[0].(domain.PlayCanned); !ok || line.AssetID != nudgeAsset {
		t.Fatalf("nudge effect = %#v", result.Effects[0])
	}
}

func TestIdleStep_NudgeCompletionRearmsSteeringTimer(t *testing.T) {
	state := IdleState{NudgeInFlight: true}
	result := IdleStep(state, domain.LineDone{UtteranceID: "nudge-1"})
	if result.State.NudgeInFlight || result.State.ElapsedBeats != 1 || len(result.Effects) != 1 {
		t.Fatalf("result = %#v", result)
	}
	timer, ok := result.Effects[0].(domain.StartTimer)
	if !ok || timer.Name != turnTimerName || timer.After != rearmDelay {
		t.Fatalf("timer = %#v", result.Effects[0])
	}
}

func TestIdleStep_SecondExpiryDispatchesPersuade(t *testing.T) {
	result := IdleStep(IdleState{Seat: 2, Intent: IntentDrifting, ElapsedBeats: 1}, domain.TimerFired{Name: turnTimerName, Stage: 2})
	if result.Rung != RungPersonalHook || len(result.Events) != 1 {
		t.Fatalf("result = %#v", result)
	}
	act, ok := result.Events[0].(domain.Act)
	if !ok || act.Seat != 2 || act.Move != vocab.MovePersuade {
		t.Fatalf("act = %#v", result.Events[0])
	}
}

func TestIdleStep_SkipsNudgeDuringVoiceAndIgnoresWrongLine(t *testing.T) {
	state := IdleState{VoiceBusy: true}
	result := IdleStep(state, domain.TimerFired{Name: turnTimerName, Stage: 1})
	if result.State.NudgeInFlight || len(result.Effects) != 0 {
		t.Fatalf("busy result = %#v", result)
	}
	result = IdleStep(IdleState{NudgeInFlight: true}, domain.LineDone{})
	if result.State.ElapsedBeats != 0 || len(result.Effects) != 0 {
		t.Fatalf("wrong line result = %#v", result)
	}
	result = IdleStep(IdleState{}, nil)
	if result.State != (IdleState{}) {
		t.Fatalf("nil result = %#v", result)
	}
}
