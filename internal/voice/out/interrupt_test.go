package out

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/ports"
)

// A line that reaches its end must never cancel its Listen audio: the TV
// still holds the tail (jitter lead plus the last chunks) when line_done is
// posted, and a cancel there would cut the line short.
func TestExecutors_CompletedLineNeverCancelsAudio(t *testing.T) {
	pcmAudio := &testAudio{}
	chunks := []ports.PCMChunk{{SampleRate: 24000, S16LE: make([]byte, 4800)}, {SampleRate: 24000, S16LE: make([]byte, 4800)}}
	NewPCMExecutor(&testTTS{chunks: chunks}, pcmAudio).StartLine(context.Background(), domain.StartLine{UtteranceID: "p-1", Input: "Hello"}, domain.Scope{}, &testInbox{})
	cannedAudio := &testAudio{}
	data := bytes.Repeat([]byte{1}, cannedChunkSize+2)
	NewCannedExecutor(testAssets{reader: io.NopCloser(bytes.NewReader(data))}, cannedAudio).PlayCanned(context.Background(), domain.PlayCanned{UtteranceID: "c-1", AssetID: "canned_opening"}, domain.Scope{}, &testInbox{})
	for name, audio := range map[string]*testAudio{"pcm": pcmAudio, "canned": cannedAudio} {
		if len(audio.cancels) != 0 {
			t.Fatalf("%s: completed line cancelled its audio: %v", name, audio.cancels)
		}
		if len(audio.frames) != 2 || !audio.frames[1].Final {
			t.Fatalf("%s: frames = %d, want 2 ending in final", name, len(audio.frames))
		}
	}
}

// A scope cancel mid-line (host Skip, phase change) is a genuine
// interruption: the executor cancels the line's Listen audio once so the TV
// fades out what it has queued.
func TestExecutors_ScopeCancelMidLineCancelsAudioOnce(t *testing.T) {
	tests := []struct {
		name string
		run  func(context.Context, *testAudio, chan struct{})
	}{
		{"pcm", func(ctx context.Context, audio *testAudio, started chan struct{}) {
			stream := &blockingPCM{started: started, stop: make(chan struct{})}
			NewPCMExecutor(&testTTS{stream: stream}, audio).StartLine(ctx, domain.StartLine{UtteranceID: "u-1"}, domain.Scope{}, &testInbox{})
		}},
		{"canned", func(ctx context.Context, audio *testAudio, started chan struct{}) {
			reader := &blockingReader{started: started, stop: make(chan struct{})}
			NewCannedExecutor(testAssets{reader: reader}, audio).PlayCanned(ctx, domain.PlayCanned{UtteranceID: "u-1", AssetID: "canned_opening"}, domain.Scope{}, &testInbox{})
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			audio := &testAudio{}
			ctx, cancel := context.WithCancel(context.Background())
			started := make(chan struct{}, 1)
			done := make(chan struct{})
			go func() { tc.run(ctx, audio, started); close(done) }()
			<-started
			cancel()
			<-done
			if len(audio.cancels) != 1 || audio.cancels[0] != "u-1" {
				t.Fatalf("cancels = %v, want [u-1]", audio.cancels)
			}
		})
	}
}

// A newer line reusing the utterance ID replaces the old one; the old line's
// exit must not cancel the new line's audio.
func TestPCMExecutor_ReplacedLineDoesNotCancelSuccessor(t *testing.T) {
	audio := &testAudio{}
	executor := NewPCMExecutor(nil, audio)
	old := executor.track("u-1", func() {})
	executor.track("u-1", func() {})
	executor.interrupt("u-1", old)
	if len(audio.cancels) != 0 {
		t.Fatalf("cancels = %v, want none", audio.cancels)
	}
}
