package out

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

func TestCannedExecutor_PlayCannedStreamsAssetAndCompletes(t *testing.T) {
	audio := &testAudio{}
	in := &testInbox{}
	data := bytes.Repeat([]byte{7}, cannedChunkSize+2)
	executor := NewCannedExecutor(testAssets{reader: io.NopCloser(bytes.NewReader(data))}, audio)
	executor.PlayCanned(context.Background(), domain.PlayCanned{UtteranceID: "c-1", AssetID: "canned_npc_reply"}, domain.Scope{}, in)
	if len(audio.frames) != 2 || len(audio.frames[0].PCMS16LE) != cannedChunkSize || !audio.frames[1].Final {
		t.Fatalf("frames = %+v", audio.frames)
	}
	if got := eventKinds(in.events); len(got) != 5 || got[0] != vocab.EventNarrationDelta || got[1] != vocab.EventLineFirstAudio || got[2] != vocab.EventNarrationDelta || got[3] != vocab.EventLineAudioFinal || got[4] != vocab.EventLineDone {
		t.Fatalf("events = %v", got)
	}
}

func TestCannedExecutor_MissingAssetFallsBackToSilence(t *testing.T) {
	in := &testInbox{}
	audio := &testAudio{}
	executor := NewCannedExecutor(testAssets{err: errMissingAsset}, audio)
	executor.PlayCanned(context.Background(), domain.PlayCanned{UtteranceID: "c-2", AssetID: "canned_npc_reply"}, domain.Scope{}, in)
	if len(in.events) != 5 || len(audio.frames) != 1 || !audio.frames[0].Final {
		t.Fatalf("events = %+v", in.events)
	}
	if got := eventKinds(in.events); got[0] != vocab.EventNarrationDelta || got[1] != vocab.EventNarrationDelta || got[2] != vocab.EventLineFirstAudio || got[3] != vocab.EventLineAudioFinal || got[4] != vocab.EventLineDone {
		t.Fatalf("events = %v", got)
	}
}

var errMissingAsset = io.ErrUnexpectedEOF

func TestCannedExecutor_CancelRemovesQueuedAudio(t *testing.T) {
	audio := &testAudio{}
	reader := &blockingReader{started: make(chan struct{}, 1), stop: make(chan struct{})}
	executor := NewCannedExecutor(testAssets{reader: reader}, audio)
	done := make(chan struct{})
	go func() {
		executor.PlayCanned(context.Background(), domain.PlayCanned{UtteranceID: "c-3"}, domain.Scope{}, &testInbox{})
		close(done)
	}()
	<-reader.started
	executor.Cancel("c-3")
	select {
	case <-done:
	case <-context.Background().Done():
		t.Fatal("unreachable")
	}
	if len(audio.cancels) != 1 || audio.cancels[0] != "c-3" {
		t.Fatalf("cancels = %v", audio.cancels)
	}
}

type testAssets struct {
	reader io.ReadCloser
	err    error
}

func (a testAssets) Open(context.Context, domain.AssetID) (io.ReadCloser, error) {
	if a.err != nil {
		return nil, a.err
	}
	return a.reader, nil
}

type blockingReader struct {
	started chan struct{}
	stop    chan struct{}
}

func (r *blockingReader) Read([]byte) (int, error) {
	select {
	case r.started <- struct{}{}:
	default:
	}
	<-r.stop
	return 0, context.Canceled
}

func (r *blockingReader) Close() error {
	select {
	case <-r.stop:
	default:
		close(r.stop)
	}
	return nil
}
