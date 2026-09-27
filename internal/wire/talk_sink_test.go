package wire

import (
	"context"
	"strings"
	"testing"

	"github.com/monstercameron/DungeonFlux/internal/api"
	"github.com/monstercameron/DungeonFlux/internal/content"
	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/fakes"
	"github.com/monstercameron/DungeonFlux/internal/ports"
	voicein "github.com/monstercameron/DungeonFlux/internal/voice/in"
)

// talkThroughSink plays one push-to-talk recording into the sink the way the
// Talk server does: TalkStart, the audio chunks in order, then TalkEnd.
func talkThroughSink(t *testing.T, sink api.TalkSink, session api.TalkSession, chunks ...string) {
	t.Helper()
	ctx := context.Background()
	if err := sink.Start(ctx, session); err != nil {
		t.Fatalf("start: %v", err)
	}
	for seq, data := range chunks {
		if err := sink.Chunk(ctx, api.TalkChunk{Session: session, Seq: uint64(seq), Data: []byte(data)}); err != nil {
			t.Fatalf("chunk %d: %v", seq, err)
		}
	}
	if err := sink.End(ctx, session); err != nil {
		t.Fatalf("end: %v", err)
	}
}

func TestAssemblerTalkSink_recordingReachesSpeechToText(t *testing.T) {
	assembler := voicein.NewAssembler()
	session := api.TalkSession{Seat: 1, UtteranceID: "utterance-1", MIME: "audio/webm;codecs=opus"}
	talkThroughSink(t, assemblerTalkSink{assembler: assembler}, session, "webm-header|", "opus-frames")

	stt := &fakes.FakeSTT{Script: []fakes.STTResult{{Transcript: ports.Transcript{Text: "Evening. I'm looking for the lamplighter."}}}}
	transcriber, err := voicein.NewTranscriber(stt, assembler)
	if err != nil {
		t.Fatal(err)
	}
	inbox := &recordingInbox{}
	keyterms := content.DefaultWorldBible().Keyterms
	transcriber.Execute(context.Background(), withKeyterms(domain.Transcribe{Seat: 1, UtteranceID: "utterance-1"}, keyterms), domain.Scope{}, inbox)

	if len(stt.Calls) != 1 {
		t.Fatalf("speech-to-text calls = %d, want 1 (the recording never reached STT)", len(stt.Calls))
	}
	call := stt.Calls[0]
	if string(call.Audio) != "webm-header|opus-frames" {
		t.Fatalf("audio sent to STT = %q, want the chunks joined in order", call.Audio)
	}
	if call.MIME != "audio/webm" {
		t.Fatalf("mime = %q, want audio/webm", call.MIME)
	}
	if !strings.Contains(strings.Join(call.Keyterms, "|"), "Mother Vell") {
		t.Fatalf("keyterms = %v, want the setting's names", call.Keyterms)
	}
	if len(inbox.events) != 1 {
		t.Fatalf("events = %#v, want one transcribed", inbox.events)
	}
	if got, ok := inbox.events[0].(domain.Transcribed); !ok || got.UtteranceID != "utterance-1" || got.Text != "Evening. I'm looking for the lamplighter." {
		t.Fatalf("event = %#v, want the transcript for utterance-1", inbox.events[0])
	}
}

func TestAssemblerTalkSink_cancelledRecordingIsNotTranscribed(t *testing.T) {
	assembler := voicein.NewAssembler()
	sink := assemblerTalkSink{assembler: assembler}
	session := api.TalkSession{Seat: 1, UtteranceID: "utterance-2", MIME: "audio/webm"}
	if err := sink.Start(context.Background(), session); err != nil {
		t.Fatal(err)
	}
	if err := sink.Chunk(context.Background(), api.TalkChunk{Session: session, Seq: 0, Data: []byte("partial")}); err != nil {
		t.Fatal(err)
	}
	if err := sink.Cancel(context.Background(), session); err != nil {
		t.Fatal(err)
	}
	if _, ok, err := assembler.Take(context.Background(), "utterance-2"); err != nil || ok {
		t.Fatalf("take after cancel = ok %v, err %v; want nothing to transcribe", ok, err)
	}
}

func TestAssemblerTalkSink_rejectsUnsupportedAudio(t *testing.T) {
	sink := assemblerTalkSink{assembler: voicein.NewAssembler()}
	if err := sink.Start(context.Background(), api.TalkSession{Seat: 1, UtteranceID: "utterance-3", MIME: "video/avi"}); err == nil {
		t.Fatal("start accepted an unsupported recording format")
	}
}
