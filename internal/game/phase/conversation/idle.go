package conversation

import (
	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

const (
	idleTimerName                = "idle_elapsed"
	turnTimerName                = "turn_timer"
	nudgeAsset    domain.AssetID = "nudge_conversation"
	rearmDelay                   = 15 * 1000000000
)

// Rung identifies the next softest steering action.
type Rung uint8

const (
	RungNone Rung = iota
	RungLure
	RungPersonalHook
	RungClueRelocation
	RungConsequence
	RungWorldClosesIn
)

// Intent describes whether a player is progressing or drifting.
type Intent uint8

const (
	IntentOnFunnel Intent = iota
	IntentCurious
	IntentDrifting
	IntentStalled
)

// IdleState is the pure pacing state for a conversation spotlight.
type IdleState struct {
	Seat          domain.SeatID
	ElapsedBeats  int
	Intent        Intent
	IdleElapsed   bool
	NudgeInFlight bool
	VoiceBusy     bool
}

// IdleResult contains pacing state, steering choice, and runtime outputs.
type IdleResult struct {
	State   IdleState
	Rung    Rung
	Effects []domain.Effect
	Events  []domain.Event
}

// IdleStep handles idle timers and the conversation nudge. It is pure: the
// caller owns timer delivery and applies the returned event/effect data.
func IdleStep(state IdleState, event domain.Event) IdleResult {
	result := IdleResult{State: state}
	if event == nil {
		return result
	}
	switch typed := event.(type) {
	case domain.TimerFired:
		result.timer(typed)
	case domain.LineDone:
		result.lineDone(typed)
	case domain.LineFirstAudio:
		result.State.VoiceBusy = true
	case domain.LineFailed:
		result.State.VoiceBusy = false
	}
	return result
}

func (r *IdleResult) timer(event domain.TimerFired) {
	if event.Name == idleTimerName || (event.Name == turnTimerName && event.Stage == 1) {
		r.nudge()
		return
	}
	if event.Name == turnTimerName && event.Stage >= 2 && !r.State.VoiceBusy {
		r.Rung = selectRung(r.State.ElapsedBeats+1, r.State.Intent)
		r.Events = append(r.Events, domain.Act{Seat: seatOrOne(r.State.Seat), Move: vocab.MovePersuade})
	}
}

func (r *IdleResult) nudge() {
	if r.State.NudgeInFlight || r.State.VoiceBusy {
		return
	}
	r.State.IdleElapsed = true
	r.State.NudgeInFlight = true
	r.Rung = selectRung(r.State.ElapsedBeats+1, r.State.Intent)
	r.Effects = append(r.Effects, domain.PlayCanned{AssetID: nudgeAsset})
}

func (r *IdleResult) lineDone(event domain.LineDone) {
	if event.UtteranceID == "" || !r.State.NudgeInFlight {
		return
	}
	r.State.NudgeInFlight = false
	r.State.VoiceBusy = false
	r.State.ElapsedBeats++
	r.Effects = append(r.Effects, domain.StartTimer{
		Name: turnTimerName, After: rearmDelay, Pausable: true,
	})
}

func selectRung(beats int, intent Intent) Rung {
	if intent == IntentOnFunnel || intent == IntentCurious {
		return RungNone
	}
	if beats < 0 {
		beats = 0
	}
	if intent == IntentStalled && beats == 0 {
		beats = 1
	}
	if beats > int(RungWorldClosesIn) {
		beats = int(RungWorldClosesIn)
	}
	return Rung(beats)
}

func seatOrOne(seat domain.SeatID) domain.SeatID {
	if seat == 0 {
		return 1
	}
	return seat
}
