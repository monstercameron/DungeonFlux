package out

import (
	"context"
	"errors"
	"io"
	"sync"
	"testing"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/ports"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

func TestPCMExecutor_StartLinePublishesFramesAndEvents(t *testing.T) {
	audio := &testAudio{}
	in := &testInbox{}
	executor := NewPCMExecutor(&testTTS{chunks: []ports.PCMChunk{{SampleRate: 16000, S16LE: []byte{1, 2}}, {SampleRate: 16000, S16LE: []byte{3, 4}}}}, audio)
	executor.StartLine(context.Background(), domain.StartLine{UtteranceID: "u-1", Role: vocab.RoleOpening, Voice: "dm", Input: "hello"}, domain.Scope{Key: "line"}, in)
	if len(audio.frames) != 2 || !audio.frames[1].Final {
		t.Fatalf("frames = %+v, want two with final frame", audio.frames)
	}
	if audio.frames[0].UtteranceID != "u-1" || audio.frames[0].Speaker != string(vocab.RoleOpening) {
		t.Fatalf("frame metadata = %+v", audio.frames[0])
	}
	if got := eventKinds(in.events); len(got) != 5 || got[0] != vocab.EventNarrationDelta || got[1] != vocab.EventLineFirstAudio || got[2] != vocab.EventNarrationDelta || got[3] != vocab.EventLineAudioFinal || got[4] != vocab.EventLineDone {
		t.Fatalf("events = %v", got)
	}
	if final := in.events[2].Event.(domain.NarrationDelta); !final.Final || final.TextSoFar != "hello" || final.Speaker != "Dungeon Master" {
		t.Fatalf("final narration = %#v", final)
	}
}

func TestPCMExecutor_CancelStopsLineAndAudio(t *testing.T) {
	audio := &testAudio{}
	stream := &blockingPCM{started: make(chan struct{}, 1), stop: make(chan struct{})}
	executor := NewPCMExecutor(&testTTS{stream: stream}, audio)
	done := make(chan struct{})
	go func() {
		executor.StartLine(context.Background(), domain.StartLine{UtteranceID: "u-2"}, domain.Scope{}, &testInbox{})
		close(done)
	}()
	<-stream.started
	executor.Cancel("u-2")
	select {
	case <-done:
	case <-context.Background().Done():
		t.Fatal("unreachable")
	}
	if len(audio.cancels) != 1 || audio.cancels[0] != "u-2" {
		t.Fatalf("cancels = %v", audio.cancels)
	}
}

func TestPCMExecutor_TTSFailurePostsLineFailed(t *testing.T) {
	in := &testInbox{}
	executor := NewPCMExecutor(&testTTS{err: errors.New("unavailable")}, &testAudio{})
	executor.StartLine(context.Background(), domain.StartLine{UtteranceID: "u-3"}, domain.Scope{}, in)
	if len(in.events) != 1 {
		t.Fatalf("events = %+v", in.events)
	}
	failure, ok := in.events[0].Event.(domain.LineFailed)
	if !ok || failure.FailureKind != vocab.ErrUnavailable {
		t.Fatalf("event = %#v", in.events[0].Event)
	}
}

type testTTS struct {
	chunks []ports.PCMChunk
	stream ports.PCMStream
	err    error
}

func (t *testTTS) Stream(context.Context, ports.TTSRequest, ports.TextStream) (ports.PCMStream, error) {
	if t.err != nil {
		return nil, t.err
	}
	if t.stream != nil {
		return t.stream, nil
	}
	return &slicePCM{chunks: t.chunks}, nil
}

type slicePCM struct {
	chunks []ports.PCMChunk
	index  int
}

func (s *slicePCM) Recv() (ports.PCMChunk, error) {
	if s.index == len(s.chunks) {
		return ports.PCMChunk{}, io.EOF
	}
	chunk := s.chunks[s.index]
	s.index++
	return chunk, nil
}
func (s *slicePCM) Close() error { return nil }

type blockingPCM struct {
	started chan struct{}
	stop    chan struct{}
}

func (s *blockingPCM) Recv() (ports.PCMChunk, error) {
	select {
	case s.started <- struct{}{}:
	default:
	}
	<-s.stop
	return ports.PCMChunk{}, context.Canceled
}
func (s *blockingPCM) Close() error {
	select {
	case <-s.stop:
	default:
		close(s.stop)
	}
	return nil
}

type testAudio struct {
	frames  []domain.AudioFrame
	cancels []domain.UtteranceID
	mu      sync.Mutex
}

func (a *testAudio) Frame(frame domain.AudioFrame) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.frames = append(a.frames, frame)
}
func (a *testAudio) Cancel(id domain.UtteranceID) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.cancels = append(a.cancels, id)
}

type testInbox struct{ events []domain.Envelope }

func (i *testInbox) Post(_ context.Context, envelope domain.Envelope) bool {
	i.events = append(i.events, envelope)
	return true
}

func eventKinds(events []domain.Envelope) []vocab.EventKind {
	result := make([]vocab.EventKind, 0, len(events))
	for _, event := range events {
		result = append(result, event.Event.Kind())
	}
	return result
}
