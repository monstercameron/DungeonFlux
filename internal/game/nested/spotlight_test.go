package nested

import (
	"testing"
	"time"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

func TestNew_CopiesSeatsAndSelectsFirst(t *testing.T) {
	seats := []domain.SeatID{1, 2}
	state := SpotlightNew(seats, PhaseExploration, true)
	seats[0] = 9
	if state.Spotlight != 1 || state.Seats[0] != 1 {
		t.Fatalf("state = %#v", state)
	}
}

func TestStep_SeatGainedStartsNudgeTimer(t *testing.T) {
	result := SpotlightStep(SpotlightNew([]domain.SeatID{1}, PhaseExploration, true), SpotlightEvent{Kind: EventSeatGained})
	if !result.State.TimerRunning || len(result.Effects) != 1 {
		t.Fatalf("result = %#v", result)
	}
	timer, ok := result.Effects[0].(domain.StartTimer)
	if !ok || timer.Name != TurnTimerName || timer.After != 12*time.Second || !timer.Pausable {
		t.Fatalf("effect = %#v", result.Effects[0])
	}
}

func TestRotate_CyclesSeats(t *testing.T) {
	state := SpotlightNew([]domain.SeatID{1, 2, 3}, PhaseExploration, true)
	state = Rotate(state)
	if state.Spotlight != 2 {
		t.Fatalf("first rotation = %d", state.Spotlight)
	}
	state = Rotate(state)
	state = Rotate(state)
	if state.Spotlight != 1 {
		t.Fatalf("wrapped rotation = %d", state.Spotlight)
	}
}

func TestStep_NudgeFreezesAndPlaysPhaseLine(t *testing.T) {
	state := SpotlightNew([]domain.SeatID{1}, PhaseConversation, true)
	state.TimerRunning = true
	result := SpotlightStep(state, SpotlightEvent{Kind: EventTimerNudge})
	if !result.State.NudgeInFlight || !result.State.TimerFrozen || len(result.Effects) != 2 {
		t.Fatalf("result = %#v", result)
	}
	if got, ok := result.Effects[1].(domain.PlayCanned); !ok || got.AssetID != ConversationNudgeAsset {
		t.Fatalf("nudge effect = %#v", result.Effects[1])
	}
	released := SpotlightStep(result.State, SpotlightEvent{Kind: EventLineDone})
	if released.State.NudgeInFlight || released.State.TimerFrozen {
		t.Fatalf("released = %#v", released.State)
	}
}

func TestStep_ConversationExpirySetsIdleThenPersuades(t *testing.T) {
	state := SpotlightNew([]domain.SeatID{1}, PhaseConversation, true)
	state.TimerRunning = true
	first := SpotlightStep(state, SpotlightEvent{Kind: EventTimerExpired})
	if !first.State.IdleElapsed || first.AutoMove != "" || len(first.Effects) != 1 {
		t.Fatalf("first = %#v", first)
	}
	second := SpotlightStep(first.State, SpotlightEvent{Kind: EventTimerExpired})
	if second.AutoMove != vocab.MovePersuade || second.AutoSeat != 1 {
		t.Fatalf("second = %#v", second)
	}
}

func TestStep_TimersOffUsesIdleFlagWithoutNudgeOrAutoMove(t *testing.T) {
	state := SpotlightNew([]domain.SeatID{1}, PhaseConversation, false)
	started := SpotlightStep(state, SpotlightEvent{Kind: EventSeatGained})
	if len(started.Effects) != 1 {
		t.Fatalf("started = %#v", started)
	}
	timer, ok := started.Effects[0].(domain.StartTimer)
	if !ok || timer.Name != IdleTimerName || timer.After != 20*time.Second {
		t.Fatalf("effect = %#v", started.Effects[0])
	}
	idle := SpotlightStep(started.State, SpotlightEvent{Kind: EventIdleElapsed})
	if !idle.State.IdleElapsed || idle.AutoMove != "" {
		t.Fatalf("idle = %#v", idle)
	}
}

func TestStep_TalkStartAndEndFreezeOnlyActiveSeatTimer(t *testing.T) {
	state := SpotlightNew([]domain.SeatID{1}, PhaseExploration, true)
	state.TimerRunning = true
	state.Spotlight = 1
	started := SpotlightStep(state, SpotlightEvent{Kind: vocab.EventTalkStart, Seat: 1})
	if !started.State.TimerFrozen || len(started.Effects) != 1 {
		t.Fatalf("started = %#v", started)
	}
	ended := SpotlightStep(started.State, SpotlightEvent{Kind: vocab.EventTalkEnd, Seat: 1})
	if ended.State.TimerFrozen || len(ended.Effects) != 1 {
		t.Fatalf("ended = %#v", ended)
	}
}

func TestStep_IgnoresMoveFromOtherSeat(t *testing.T) {
	state := SpotlightNew([]domain.SeatID{1, 2}, PhaseExploration, true)
	state.TimerRunning = true
	result := SpotlightStep(state, SpotlightEvent{Kind: EventAcceptedMove, Seat: 2})
	if result.State.IdleElapsed || len(result.Effects) != 0 {
		t.Fatalf("result = %#v", result)
	}
}
