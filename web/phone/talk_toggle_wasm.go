//go:build js && wasm

package phone

import (
	"context"
	"errors"
	"sync"
	"syscall/js"
	"time"
)

// browserTalk holds one model's live browser recording: the MediaRecorder, the
// microphone stream, and the Talk context. It is keyed by the PTTModel, which
// is created once per phone, so a talk screen that remounts mid-recording
// finds the same session instead of cancelling it on unmount.
type browserTalk struct {
	mu       sync.Mutex
	recorder *BrowserRecorder
	stream   js.Value
	cancel   context.CancelFunc
	refresh  func()
}

var browserTalks sync.Map // *PTTModel -> *browserTalk

func browserTalkFor(model *PTTModel) *browserTalk {
	session, _ := browserTalks.LoadOrStore(model, &browserTalk{})
	return session.(*browserTalk)
}

// setRefresh registers the mounted talk button's re-render, so goroutines
// update whichever copy of the screen is on the page now.
func (s *browserTalk) setRefresh(refresh func()) {
	s.mu.Lock()
	s.refresh = refresh
	s.mu.Unlock()
}

func (s *browserTalk) redraw() {
	s.mu.Lock()
	refresh := s.refresh
	s.mu.Unlock()
	if refresh != nil {
		refresh()
	}
}

func (s *browserTalk) take() (*BrowserRecorder, js.Value, context.CancelFunc) {
	s.mu.Lock()
	defer s.mu.Unlock()
	recorder, stream, cancel := s.recorder, s.stream, s.cancel
	s.recorder, s.stream, s.cancel = nil, js.Value{}, nil
	return recorder, stream, cancel
}

// beginTalk opens the microphone and the Talk stream, then reports to the
// toggle; a tap that arrived while opening finishes the recording at once.
func beginTalk(model *PTTModel, locale string) {
	session := browserTalkFor(model)
	ctx, cancel := context.WithCancel(context.Background())
	recorder, stream, err := openTalk(ctx, model, locale)
	if err != nil {
		cancel()
	} else {
		session.mu.Lock()
		session.recorder, session.stream, session.cancel = recorder, stream, cancel
		session.mu.Unlock()
		go watchTalk(ctx, model, recorder)
	}
	if model.Toggle().Started(err) == ToggleFinish {
		endTalk(model)
		return
	}
	session.redraw()
}

func openTalk(ctx context.Context, model *PTTModel, locale string) (*BrowserRecorder, js.Value, error) {
	stream, err := requestMicrophone(ctx)
	if err != nil {
		return nil, js.Value{}, err
	}
	mimeType := recorderMIME()
	if mimeType == "" {
		stopTracks(stream)
		return nil, js.Value{}, errors.New(PTTNoRecord(locale))
	}
	if err := model.Start(ctx, mimeType); err != nil {
		stopTracks(stream)
		return nil, js.Value{}, err
	}
	recorder, err := newBrowserRecorder(stream, mimeType, model.QueueChunk)
	if err == nil {
		err = recorder.Start()
		if err != nil {
			recorder.Dispose()
		}
	}
	if err != nil {
		stopTracks(stream)
		<-model.Stop(context.Background())
		return nil, js.Value{}, err
	}
	return recorder, stream, nil
}

// endTalk stops the recorder, drains the upload, and sends TalkEnd.
func endTalk(model *PTTModel) {
	session := browserTalkFor(model)
	recorder, stream, cancel := session.take()
	if recorder == nil && cancel == nil {
		return // another path (a tap or a recorder failure) already ended it
	}
	notice := ""
	if recorder != nil {
		if err := recorder.Stop(); err != nil {
			notice = err.Error()
		} else {
			<-recorder.Done()
			if err := recorder.Err(); err != nil {
				notice = err.Error()
			}
		}
		recorder.Dispose()
	}
	stopTracks(stream)
	stopErr := <-model.Stop(context.Background())
	if stopErr != nil && notice == "" {
		notice = stopErr.Error()
	}
	releaseTalk(cancel, stopErr == nil && notice == "")
	model.Toggle().Finished(notice)
	session.redraw()
}

// watchTalk ends a recording whose browser recorder fails mid-take, so the
// button never stays red over a dead microphone.
func watchTalk(ctx context.Context, model *PTTModel, recorder *BrowserRecorder) {
	select {
	case <-recorder.Done():
		if recorder.Err() == nil || model.Toggle().Phase() != ToggleRecording {
			return
		}
		endTalk(model)
	case <-ctx.Done():
	}
}

// talkReleaseDelay is how long a sent recording's Talk stream stays open after
// TalkEnd. Cancelling at once aborted the stream before the server read
// TalkEnd, so the server saw a dropped stream and discarded the recording.
const talkReleaseDelay = 10 * time.Second

// releaseTalk frees a Talk context. A clean send lets the stream close on its
// own and releases the context later; a failed one is cancelled now.
func releaseTalk(cancel context.CancelFunc, sent bool) {
	if cancel == nil {
		return
	}
	if !sent {
		cancel()
		return
	}
	time.AfterFunc(talkReleaseDelay, cancel)
}
