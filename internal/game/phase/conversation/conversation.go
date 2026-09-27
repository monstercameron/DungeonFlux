package conversation

import (
	"errors"
	"strings"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

// InterpretationDialogue is the interpretation kind for ordinary NPC speech.
const InterpretationDialogue = "dialogue"

// InterpretationMove is the interpretation kind for a legal player action.
const InterpretationMove = "move"

// State is the serializable state of one conversation phase.
type State struct {
	Seat              domain.SeatID
	NPCReplies        int
	UtteranceInFlight bool
	ActiveUtteranceID domain.UtteranceID
	VoiceBusy         bool
	IdleElapsed       bool
	Done              bool
	LastText          string
	Transcript        string
}

// Event is an input to the conversation dispatcher.
type Event struct {
	Event domain.Event
}

// Result contains the updated phase state and pure outputs for the room loop.
type Result struct {
	State   State
	Events  []domain.Event
	Effects []domain.Effect
}

// Step applies one conversation event. Unsupported events are ignored so the
// room loop can safely pass phase-wide callbacks to this package.
func Step(state State, input Event) (Result, error) {
	if input.Event == nil {
		return Result{State: state}, errors.New("conversation event is nil")
	}
	result := Result{State: state}
	switch event := input.Event.(type) {
	case domain.Transcribed:
		result.transcribed(event)
	case domain.Say:
		result.typed(event)
	case domain.TalkEnd:
		result.talkEnded(event)
	case domain.Interpreted:
		result.interpreted(event)
	case domain.InterpretFailed:
		result.interpretFailed(event)
	case domain.LineFirstAudio:
		result.voiceBusy(event.UtteranceID, true)
	case domain.LineDone:
		result.voiceBusy(event.UtteranceID, false)
	case domain.LineFailed:
		result.voiceBusy(event.UtteranceID, false)
	case domain.TimerFired:
		if event.Name == idleTimerName {
			result.State.IdleElapsed = true
		}
	case domain.UtteranceFinal:
		result.State.LastText = event.CleanText
	}
	return result, nil
}

func (r *Result) transcribed(event domain.Transcribed) {
	if event.UtteranceID == "" || r.State.Done {
		return
	}
	r.State.UtteranceInFlight = true
	r.State.ActiveUtteranceID = event.UtteranceID
	r.State.Transcript = event.Text
	r.Effects = append(r.Effects, domain.Interpret{
		Seat: eventSeat(r.State.Seat), UtteranceID: event.UtteranceID,
		Transcript: event.Text, Moves: legalMoves(), NPCLastLine: r.State.LastText,
	})
}

// typed enters a typed line (SessionService.Say) at the transcript stage, as
// plan §0.9 stage 0 requires: it skips capture and STT and is interpreted like
// speech. Only the seat in the conversation may speak; other seats are ignored.
func (r *Result) typed(event domain.Say) {
	if r.State.Seat != 0 && event.Seat != r.State.Seat {
		return
	}
	text := strings.TrimSpace(event.Text)
	if text == "" {
		return
	}
	r.transcribed(domain.Transcribed{UtteranceID: event.UtteranceID, Text: text})
}

// talkEnded asks for the finished push-to-talk recording to be transcribed.
// The Transcribed result then enters at the transcript stage like typed text.
// Only the seat in the conversation may speak; other seats are ignored.
func (r *Result) talkEnded(event domain.TalkEnd) {
	if r.State.Done || event.UtteranceID == "" {
		return
	}
	if r.State.Seat != 0 && event.Seat != r.State.Seat {
		return
	}
	r.Effects = append(r.Effects, domain.Transcribe{Seat: eventSeat(event.Seat), UtteranceID: event.UtteranceID})
}

func (r *Result) interpreted(event domain.Interpreted) {
	if !r.acceptUtterance(event.UtteranceID) {
		return
	}
	text := strings.TrimSpace(event.CleanText)
	// The interpret schema's enum is upper case ("DIALOGUE", "MOVE"), so the
	// kind is compared without case.
	kind := strings.ToLower(strings.TrimSpace(event.InterpretationKind))
	if kind == InterpretationMove && event.Move != "" {
		r.emitMove(event.Move)
		return
	}
	if kind != "" && kind != InterpretationDialogue {
		return
	}
	if text == "" {
		return
	}
	r.State.NPCReplies++
	r.State.LastText = text
	r.Events = append(r.Events, domain.UtteranceFinal{Seat: r.State.Seat, UtteranceID: event.UtteranceID, CleanText: text})
	r.Effects = append(r.Effects, domain.StartLine{UtteranceID: event.UtteranceID, Role: vocab.RoleNPCReply, Speaker: "Mother Vell", Input: text})
}

func (r *Result) interpretFailed(event domain.InterpretFailed) {
	if !r.acceptUtterance(event.UtteranceID) {
		return
	}
	move := keywordMove(r.State.Transcript)
	if move == "" {
		return
	}
	r.emitMove(move)
}

func (r *Result) acceptUtterance(id domain.UtteranceID) bool {
	if id == "" || r.State.Done || !r.State.UtteranceInFlight || id != r.State.ActiveUtteranceID {
		return false
	}
	r.State.UtteranceInFlight = false
	return true
}

func (r *Result) emitMove(move vocab.MoveID) {
	r.Events = append(r.Events, domain.Act{Seat: r.State.Seat, Move: move})
}

func (r *Result) voiceBusy(id domain.UtteranceID, busy bool) {
	if id == "" || r.State.Done {
		return
	}
	r.State.VoiceBusy = busy
}

func eventSeat(seat domain.SeatID) domain.SeatID {
	if seat == 0 {
		return 1
	}
	return seat
}

func legalMoves() []vocab.MoveID {
	return []vocab.MoveID{vocab.MovePersuade, vocab.MoveStepAway}
}

func keywordMove(text string) vocab.MoveID {
	words := strings.Fields(strings.ToLower(text))
	for _, word := range words {
		switch strings.Trim(word, ".,!?;:") {
		case "persuade", "convince", "plead":
			return vocab.MovePersuade
		case "leave", "step", "away":
			return vocab.MoveStepAway
		}
	}
	return ""
}
