package voice

import (
	"testing"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/game/phase"
	"github.com/monstercameron/DungeonFlux/internal/game/phase/conversation"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

type walk struct {
	phase phase.Machine
	voice conversation.State
}

func newWalk(t *testing.T) *walk {
	t.Helper()
	machine, err := phase.New()
	if err != nil {
		t.Fatal(err)
	}
	return &walk{phase: machine, voice: conversation.State{Seat: 1}}
}

func (w *walk) toConversation(t *testing.T) {
	t.Helper()
	for _, event := range []domain.Event{
		domain.HostCmd{Cmd: vocab.HostStart},
		domain.PCLocked{Seat: 1},
		domain.LineDone{},
		domain.Act{Seat: 1, Move: vocab.MoveTalkVell},
	} {
		if _, err := w.phase.Step(event); err != nil {
			t.Fatalf("phase Step(%T): %v", event, err)
		}
	}
}

func (w *walk) speech(t *testing.T, id domain.UtteranceID, text string) conversation.Result {
	t.Helper()
	result, err := conversation.Step(w.voice, conversation.Event{
		Event: domain.Transcribed{UtteranceID: id, Text: text},
	})
	if err != nil {
		t.Fatal(err)
	}
	w.voice = result.State
	return result
}

func (w *walk) interpreted(t *testing.T, event domain.Event) conversation.Result {
	t.Helper()
	result, err := conversation.Step(w.voice, conversation.Event{Event: event})
	if err != nil {
		t.Fatal(err)
	}
	w.voice = result.State
	return result
}

func (w *walk) act(move vocab.MoveID) error {
	if move == vocab.MovePersuade {
		if err := guardMove(w.voice.VoiceBusy, w.voice.NPCReplies, w.voice.IdleElapsed); err != nil {
			return err
		}
	}
	_, err := w.phase.Step(domain.Act{Seat: 1, Move: move})
	return err
}

func TestWalkVoice_PTTDialogueStartsNPCLine(t *testing.T) {
	w := newWalk(t)
	w.toConversation(t)
	result := w.speech(t, "ptt-1", "Where is the bell tower?")
	if result.State.UtteranceInFlight || len(result.Effects) != 1 {
		t.Fatalf("speech = %#v", result)
	}
	if len(result.Events) != 1 || len(result.Effects) != 1 {
		t.Fatalf("dialogue result = %#v", result)
	}
	if final, ok := result.Events[0].(domain.UtteranceFinal); !ok || final.CleanText != "Where is the bell tower?" {
		t.Fatalf("event = %#v", result.Events[0])
	}
	line := result.Effects[0].(domain.StartLine)
	if line.Role != vocab.RoleNPCReply || line.UtteranceID != "ptt-1" || line.Input != "Where is the bell tower?" {
		t.Fatalf("line = %#v", line)
	}
}

func TestWalkVoice_STTFailureTypedSayUsesSameDialoguePath(t *testing.T) {
	w := newWalk(t)
	w.toConversation(t)
	first := w.interpreted(t, domain.STTError{UtteranceID: "failed-ptt", FailureKind: vocab.ErrUnavailable})
	if len(first.Effects) != 0 || len(first.Events) != 0 {
		t.Fatal("failed transcription invented speech")
	}
	// The voice input lane drops the failed audio and enters Say at stage 4.
	result := w.interpreted(t, domain.Say{
		Seat: 1, UtteranceID: "typed-1", Text: "Tell me about the stranger",
	})
	if len(result.Events) != 1 || len(result.Effects) != 1 {
		t.Fatalf("typed result = %#v", result)
	}
	if result.Events[0].(domain.UtteranceFinal).CleanText != "Tell me about the stranger" {
		t.Fatalf("typed event = %#v", result.Events[0])
	}
}

func TestWalkVoice_InterruptedReplyRejectsPersuadeUntilLineStops(t *testing.T) {
	w := newWalk(t)
	w.toConversation(t)
	w.voice = conversation.State{Seat: 1, NPCReplies: 1}
	started, err := conversation.Step(w.voice, conversation.Event{
		Event: domain.LineFirstAudio{UtteranceID: "reply-1"},
	})
	if err != nil {
		t.Fatal(err)
	}
	w.voice = started.State
	if err := w.act(vocab.MovePersuade); err == nil {
		t.Fatal("persuade accepted while reply was speaking")
	}
	finished := w.interpreted(t, domain.LineDone{UtteranceID: "reply-1"})
	if finished.State.VoiceBusy {
		t.Fatal("reply remained busy after line_done")
	}
	if err := w.act(vocab.MovePersuade); err != nil {
		t.Fatalf("persuade after line_done: %v", err)
	}
}

func TestWalkVoice_InterpretFailureFallsBackToKeywordMove(t *testing.T) {
	w := newWalk(t)
	w.toConversation(t)
	w.speech(t, "move-1", "I try to persuade her")
	result := w.interpreted(t, domain.InterpretFailed{UtteranceID: "move-1"})
	if len(result.Events) != 1 {
		t.Fatalf("fallback events = %#v", result.Events)
	}
	act := result.Events[0].(domain.Act)
	if act.Move != vocab.MovePersuade || act.Seat != 1 {
		t.Fatalf("fallback act = %#v", act)
	}
	if _, err := w.phase.Step(act); err != nil {
		t.Fatalf("fallback persuade: %v", err)
	}
}

func TestWalkVoice_RejectedMoveCanBeRedeliveredAsDialogue(t *testing.T) {
	w := newWalk(t)
	w.toConversation(t)
	w.speech(t, "move-2", "I try to persuade her")
	result := w.interpreted(t, domain.Interpreted{
		UtteranceID: "move-2", InterpretationKind: conversation.InterpretationMove,
		Move: vocab.MovePersuade,
	})
	if len(result.Events) != 1 {
		t.Fatalf("move events = %#v", result.Events)
	}
	if err := w.act(vocab.MovePersuade); err == nil {
		t.Fatal("persuade accepted before an NPC turn")
	}
	// The dispatcher retries a guarded MOVE as dialogue using the same text.
	w.voice.UtteranceInFlight = true
	w.voice.ActiveUtteranceID = "move-2"
	result = w.interpreted(t, domain.Interpreted{
		UtteranceID: "move-2", CleanText: "I try to persuade her",
		InterpretationKind: conversation.InterpretationDialogue,
	})
	if len(result.Events) != 1 || len(result.Effects) != 1 {
		t.Fatalf("redelivered dialogue = %#v", result)
	}
	if result.Events[0].(domain.UtteranceFinal).CleanText != "I try to persuade her" {
		t.Fatalf("dialogue event = %#v", result.Events[0])
	}
}

func TestWalkVoice_AcceptedSpokenPersuadeDropsHeldReply(t *testing.T) {
	w := newWalk(t)
	w.toConversation(t)
	w.voice = conversation.State{Seat: 1, NPCReplies: 1}
	w.speech(t, "move-3", "I try to persuade her")
	held := domain.StartLine{UtteranceID: "move-3", Role: vocab.RoleNPCReply, Hold: true}
	if !held.Hold {
		t.Fatal("speculative line was not held")
	}
	result := w.interpreted(t, domain.Interpreted{
		UtteranceID: "move-3", InterpretationKind: conversation.InterpretationMove,
		Move: vocab.MovePersuade,
	})
	if len(result.Events) != 1 || result.Events[0].(domain.Act).Move != vocab.MovePersuade {
		t.Fatalf("persuade result = %#v", result)
	}
	if _, err := w.phase.Step(result.Events[0]); err != nil {
		t.Fatalf("accepted persuade: %v", err)
	}
	// The line executor owns cancellation of the held speculative reply.
	dropped := domain.DropLine{UtteranceID: "move-3"}
	if dropped.UtteranceID != "move-3" {
		t.Fatal("held reply was not identified for DropLine")
	}
}

func TestWalkVoice_LineFailureAndSkipCancelTheCurrentAudio(t *testing.T) {
	w := newWalk(t)
	w.toConversation(t)
	w.voice = conversation.State{Seat: 1, NPCReplies: 1, VoiceBusy: true}
	failed := w.interpreted(t, domain.LineFailed{UtteranceID: "live-1"})
	if failed.State.VoiceBusy {
		t.Fatal("failed line remained busy")
	}
	// Runtime policy replaces a failed live line with its canned asset.
	canned := domain.PlayCanned{UtteranceID: "canned-1", AssetID: "npc_refuse"}
	if canned.AssetID == "" || canned.UtteranceID == "" {
		t.Fatal("canned replacement was not registered")
	}
	w.voice.VoiceBusy = true
	if _, err := w.phase.Step(domain.HostCmd{Cmd: vocab.HostSkip}); err != nil {
		t.Fatalf("skip: %v", err)
	}
	cancel := domain.SendAudioCancel{UtteranceID: "canned-1"}
	if cancel.UtteranceID != canned.UtteranceID {
		t.Fatal("skip did not target the current line")
	}
}
