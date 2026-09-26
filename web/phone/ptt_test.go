package phone

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	df "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
)

func TestPTTModel_UploadsChunksInOrderAndKeepsFirstChunk(t *testing.T) {
	fake := &talkFake{}
	m := NewPTTModel(fake, "seat", 4)
	if err := m.Start(context.Background(), "audio/webm"); err != nil {
		t.Fatal(err)
	}
	if !m.QueueChunk([]byte("header")) || !m.QueueChunk([]byte("tail")) {
		t.Fatal("chunks were not accepted")
	}
	if err := <-m.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(fake.requests) != 4 || fake.requests[0].GetStart().GetMimeType() != "audio/webm" {
		t.Fatalf("requests = %+v", fake.requests)
	}
	if string(fake.requests[1].GetChunk().GetData()) != "header" || fake.requests[1].GetChunk().GetSeq() != 0 {
		t.Fatalf("first chunk = %+v", fake.requests[1].GetChunk())
	}
	if string(fake.requests[2].GetChunk().GetData()) != "tail" || fake.requests[2].GetChunk().GetSeq() != 1 {
		t.Fatalf("second chunk = %+v", fake.requests[2].GetChunk())
	}
	if fake.requests[3].GetEnd() == nil {
		t.Fatal("TalkEnd was not sent")
	}
	if state, err := m.State(); state != PTTIdle || err != nil {
		t.Fatalf("state = %q, err = %v", state, err)
	}
}

func TestPTTModel_QueueDoesNotBlockAndReportsFull(t *testing.T) {
	fake := &talkFake{sendGate: make(chan struct{}), sendStarted: make(chan struct{})}
	m := NewPTTModel(fake, "seat", 1)
	if err := m.Start(context.Background(), "audio/mp4"); err != nil {
		t.Fatal(err)
	}
	if !m.QueueChunk([]byte{1}) {
		t.Fatal("first chunk rejected")
	}
	select {
	case <-fake.sendStarted:
	case <-time.After(time.Second):
		t.Fatal("uploader did not start")
	}
	if m.QueueChunk([]byte{2}) {
		t.Fatal("full queue accepted a chunk")
	}
	if state, err := m.State(); state != PTTFailed || err == nil {
		t.Fatalf("state = %q, err = %v", state, err)
	}
	close(fake.sendGate)
	select {
	case <-m.done:
	case <-time.After(time.Second):
		t.Fatal("uploader did not finish")
	}
}

func TestPTTModel_StartAndStopErrors(t *testing.T) {
	if err := NewPTTModel(nil, "", 0).Start(context.Background(), "audio/webm"); err == nil {
		t.Fatal("nil opener accepted")
	}
	fake := &talkFake{openErr: errors.New("offline")}
	m := NewPTTModel(fake, "seat", 1)
	if err := m.Start(context.Background(), "audio/webm"); err == nil {
		t.Fatal("open error missing")
	}
	if err := <-m.Stop(context.Background()); err == nil {
		t.Fatal("inactive stop error missing")
	}
}

func TestPTTModel_StartSendError(t *testing.T) {
	fake := &talkFake{sendErr: errors.New("closed")}
	m := NewPTTModel(fake, "seat", 1)
	if err := m.Start(context.Background(), "audio/webm"); err == nil {
		t.Fatal("send error missing")
	}
	if state, _ := m.State(); state != PTTFailed {
		t.Fatalf("state = %q", state)
	}
}

func TestPTTModel_ControlSnapshotStates(t *testing.T) {
	model := NewPTTModel(nil, "seat", 1)
	if got := model.ControlSnapshot(""); !got.CanStart || got.CanStop || got.ShowFallback || got.StatusText != PTTReady("en") {
		t.Fatalf("idle snapshot = %+v", got)
	}
	model.fail(errors.New("permission denied"))
	got := model.ControlSnapshot("en")
	if !got.CanStart || got.CanStop || !got.ShowFallback || got.ErrorText != "permission denied" {
		t.Fatalf("failed snapshot = %+v", got)
	}
	model.mu.Lock()
	model.state = PTTRecording
	model.err = nil
	model.mu.Unlock()
	got = model.ControlSnapshot("es")
	if got.CanStart || !got.CanStop || got.ShowFallback || got.StatusText != T("es", "ui.ptt.recording", nil) {
		t.Fatalf("recording snapshot = %+v", got)
	}
	model.mu.Lock()
	model.state = PTTTranscribing
	model.mu.Unlock()
	got = model.ControlSnapshot("en")
	if got.CanStart || got.CanStop || got.ShowFallback || got.StatusText != T("en", "ptt.finishing", nil) {
		t.Fatalf("transcribing snapshot = %+v", got)
	}
}

type talkFake struct {
	requests    []*df.TalkRequest
	openErr     error
	sendErr     error
	sendGate    chan struct{}
	sendStarted chan struct{}
	sendOnce    sync.Once
}

func (f *talkFake) OpenTalk(context.Context) (TalkStream, error) {
	if f.openErr != nil {
		return nil, f.openErr
	}
	return f, nil
}

func (f *talkFake) Send(request *df.TalkRequest) error {
	if f.sendErr != nil {
		return f.sendErr
	}
	if request.GetStart() == nil && f.sendGate != nil {
		f.sendOnce.Do(func() { close(f.sendStarted) })
		<-f.sendGate
	}
	f.requests = append(f.requests, request)
	return nil
}

func (f *talkFake) CloseSend() error { return nil }
