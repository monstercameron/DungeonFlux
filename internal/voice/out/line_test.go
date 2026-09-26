package out

import (
	"context"
	"testing"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/ports"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

func TestLineExecutor_StreamsStartLineThroughTTS(t *testing.T) {
	audio := &testAudio{}
	in := &testInbox{}
	tts := &testTTS{chunks: []ports.PCMChunk{{SampleRate: 16000, S16LE: []byte{1, 2}}}}
	executor := NewLineExecutor(tts, audio)

	executor.Execute(context.Background(), domain.StartLine{
		UtteranceID: "line-1",
		Role:        vocab.RoleNPCReply,
		Voice:       "vell",
		Input:       "The bell remembers.",
	}, domain.Scope{Key: "conversation"}, in)

	if len(audio.frames) != 1 || !audio.frames[0].Final {
		t.Fatalf("frames = %+v, want one final frame", audio.frames)
	}
	if got := eventKinds(in.events); len(got) != 5 || got[0] != vocab.EventNarrationDelta || got[1] != vocab.EventLineFirstAudio || got[2] != vocab.EventNarrationDelta || got[3] != vocab.EventLineAudioFinal || got[4] != vocab.EventLineDone {
		t.Fatalf("events = %v", got)
	}
}

func TestLineExecutor_CancelStopsActiveLine(t *testing.T) {
	audio := &testAudio{}
	stream := &blockingPCM{started: make(chan struct{}, 1), stop: make(chan struct{})}
	executor := NewLineExecutor(&testTTS{stream: stream}, audio)
	done := make(chan struct{})
	go func() {
		executor.StartLine(context.Background(), domain.StartLine{UtteranceID: "line-2"}, domain.Scope{}, &testInbox{})
		close(done)
	}()
	<-stream.started

	executor.Cancel("line-2")
	<-done
	if len(audio.cancels) != 1 || audio.cancels[0] != "line-2" {
		t.Fatalf("cancels = %v", audio.cancels)
	}
}
