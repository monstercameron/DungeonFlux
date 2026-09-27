// Package nested contains the small pure machines hosted by a game phase.
// It depends on domain values and vocabulary only; effects are data for the
// runtime to execute.
package nested

import (
	"time"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

const (
	// TurnTimerName is the timer used for a spotlight seat.
	TurnTimerName = "turn_timer"
	// IdleTimerName is the flag timer used when turn timers are disabled.
	IdleTimerName = "idle_elapsed"
	// ExplorationNudgeAsset is the build-time exploration nudge asset.
	ExplorationNudgeAsset domain.AssetID = "nudge_exploration"
	// ConversationNudgeAsset is the build-time conversation nudge asset.
	ConversationNudgeAsset domain.AssetID = "nudge_conversation"
)

const (
	// EventSeatGained starts a timer for a newly spotlighted seat.
	EventSeatGained vocab.EventKind = "seat_gained"
	// EventRotate passes the spotlight to the next seat in seat order.
	EventRotate vocab.EventKind = "rotate"
	// EventAcceptedMove restarts the timer after an accepted player move.
	EventAcceptedMove vocab.EventKind = "accepted_move"
	// EventUtteranceFinal restarts the timer after accepted speech.
	EventUtteranceFinal vocab.EventKind = "utterance_final"
	// EventTimerNudge fires the first timer stage.
	EventTimerNudge vocab.EventKind = "timer_nudge"
	// EventTimerExpired fires the final timer stage.
	EventTimerExpired vocab.EventKind = "timer_expired"
	// EventIdleElapsed sets the conversation flag.
	EventIdleElapsed vocab.EventKind = "idle_elapsed"
	// EventLineDone releases a timer frozen for a nudge line.
	EventLineDone vocab.EventKind = "line_done"
	// EventTalkStart freezes the timer while speech is captured.
	// EventTalkEnd thaws the timer after speech capture ends.
	// EventPause freezes the timer.
	EventPause vocab.EventKind = "pause"
	// EventResume thaws the timer.
	EventResume vocab.EventKind = "resume"
)

// Phase controls which timer policy is active.
type Phase string

const (
	// PhaseExploration uses the exploration nudge and auto-action policy.
	PhaseExploration Phase = "exploration"
	// PhaseConversation uses the conversation nudge and two-stage expiry.
	PhaseConversation Phase = "conversation"
)

// SpotlightEvent is an input to the spotlight machine. Seat is used by
// seat-specific events.
type SpotlightEvent struct {
	Kind vocab.EventKind
	Seat domain.SeatID
}

// State is the serializable spotlight state owned by a phase.
type State struct {
	Seats         []domain.SeatID
	Spotlight     domain.SeatID
	Phase         Phase
	TimersEnabled bool
	IdleElapsed   bool
	TimerRunning  bool
	TimerFrozen   bool
	NudgeInFlight bool
	TimerStage    int
}

// Result contains the next state, runtime effects, and an optional automatic
// move requested by a timer expiry.
type Result struct {
	State    State
	Effects  []domain.Effect
	AutoMove vocab.MoveID
	AutoSeat domain.SeatID
}

// SpotlightNew returns a spotlight state with the first seat active. A copy of seats is
// retained so callers can safely reuse their input slice.
func SpotlightNew(seats []domain.SeatID, phase Phase, timersEnabled bool) State {
	copySeats := append([]domain.SeatID(nil), seats...)
	state := State{Seats: copySeats, Phase: phase, TimersEnabled: timersEnabled}
	if len(copySeats) > 0 {
		state.Spotlight = copySeats[0]
	}
	return state
}

// SpotlightStep applies one spotlight event without performing I/O or reading
// a clock.
func SpotlightStep(state State, event SpotlightEvent) Result {
	state.Seats = append([]domain.SeatID(nil), state.Seats...)
	result := Result{State: state}
	switch event.Kind {
	case EventSeatGained:
		result.startTimer()
	case EventRotate:
		result.State = Rotate(result.State)
		result.startTimer()
	case EventAcceptedMove, EventUtteranceFinal:
		if event.Seat != state.Spotlight {
			return result
		}
		result.State.IdleElapsed = false
		result.startTimer()
	case EventTimerNudge:
		result.nudge()
	case EventTimerExpired:
		result.expire()
	case EventIdleElapsed:
		result.State.IdleElapsed = true
	case EventLineDone:
		if state.NudgeInFlight {
			result.State.NudgeInFlight = false
			result.State.TimerFrozen = false
			result.startTimer()
		}
	case vocab.EventTalkStart:
		result.State.TimerFrozen = true
		result.freezeTimer()
	case vocab.EventTalkEnd, EventResume:
		if !state.NudgeInFlight {
			result.State.TimerFrozen = false
			result.thawTimer()
		}
	case EventPause:
		result.State.TimerFrozen = true
		result.freezeTimer()
	}
	return result
}

func (r *Result) startTimer() {
	r.State.TimerRunning = true
	r.State.TimerFrozen = false
	r.State.TimerStage = 0
	if r.State.Phase == PhaseConversation && !r.State.TimersEnabled {
		r.Effects = append(r.Effects, startTimer(IdleTimerName, 20*time.Second))
		return
	}
	if r.State.TimersEnabled {
		r.Effects = append(r.Effects, startTimer(TurnTimerName, 12*time.Second))
	}
}

func (r *Result) nudge() {
	if !r.State.TimersEnabled || r.State.NudgeInFlight || r.State.TimerFrozen {
		return
	}
	r.State.NudgeInFlight = true
	r.State.TimerFrozen = true
	r.State.TimerStage = 1
	r.Effects = append(r.Effects, domain.FreezeTimer{Name: TurnTimerName})
	asset := ExplorationNudgeAsset
	if r.State.Phase == PhaseConversation {
		asset = ConversationNudgeAsset
	}
	r.Effects = append(r.Effects, domain.PlayCanned{AssetID: asset})
}

func (r *Result) expire() {
	if !r.State.TimersEnabled || r.State.TimerFrozen || r.State.NudgeInFlight {
		return
	}
	r.State.TimerRunning = false
	r.State.TimerStage++
	r.AutoSeat = r.State.Spotlight
	if r.State.Phase == PhaseConversation && r.State.TimerStage == 1 {
		r.State.IdleElapsed = true
		r.State.TimerRunning = true
		r.Effects = append(r.Effects, startTimer(TurnTimerName, 15*time.Second))
		return
	}
	if r.State.Phase == PhaseConversation {
		r.AutoMove = vocab.MovePersuade
		return
	}
	if move := explorationMove(r.State); move != "" {
		r.AutoMove = move
	}
}

func (r *Result) freezeTimer() {
	if r.State.TimerRunning {
		r.Effects = append(r.Effects, domain.FreezeTimer{Name: activeTimer(r.State)})
	}
}

func (r *Result) thawTimer() {
	if r.State.TimerRunning {
		r.Effects = append(r.Effects, domain.ThawTimer{Name: activeTimer(r.State)})
	}
}

func activeTimer(state State) string {
	if state.TimersEnabled {
		return TurnTimerName
	}
	return IdleTimerName
}

func startTimer(name string, after time.Duration) domain.Effect {
	return domain.StartTimer(domain.TimerEffect{Name: name, After: after, Pausable: true})
}

func explorationMove(state State) vocab.MoveID {
	return vocab.MoveTalkVell
}

// Rotate advances the spotlight to the next configured seat. It preserves the
// current phase and timer policy and is a no-op when fewer than two seats join.
func Rotate(state State) State {
	if len(state.Seats) < 2 {
		return state
	}
	for index, seat := range state.Seats {
		if seat == state.Spotlight {
			state.Spotlight = state.Seats[(index+1)%len(state.Seats)]
			return state
		}
	}
	state.Spotlight = state.Seats[0]
	return state
}
