package in

import (
	"context"
	"errors"
	"testing"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/ports"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

type fakeSTT struct {
	request ports.STTRequest
	result  ports.Transcript
	err     error
}

func (f *fakeSTT) Transcribe(_ context.Context, request ports.STTRequest) (ports.Transcript, error) {
	f.request = request
	return f.result, f.err
}

type eventInbox struct {
	events []domain.Envelope
}

func (i *eventInbox) Post(_ context.Context, event domain.Envelope) bool {
	i.events = append(i.events, event)
	return true
}

func TestTranscriber_Execute_postsTranscriptWithRequest(t *testing.T) {
	assembler := NewAssembler()
	session := Session{Seat: 2, UtteranceID: "u-1", MIME: "audio/webm"}
	if err := assembler.Start(context.Background(), session); err != nil {
		t.Fatal(err)
	}
	if err := assembler.Chunk(context.Background(), Chunk{Session: session, Data: []byte("header-and-audio")}); err != nil {
		t.Fatal(err)
	}
	if _, err := assembler.End(context.Background(), session); err != nil {
		t.Fatal(err)
	}
	stt := &fakeSTT{result: ports.Transcript{Text: " speak clearly "}}
	executor, err := NewTranscriber(stt, assembler)
	if err != nil {
		t.Fatal(err)
	}
	inbox := &eventInbox{}
	scope := domain.Scope{Machine: vocab.MachineSession, Epoch: 4, Key: "utterance/u-1"}
	executor.Execute(context.Background(), domain.Transcribe{Seat: 2, UtteranceID: "u-1", Keyterms: []string{"bell tower"}}, scope, inbox)
	if len(inbox.events) != 1 {
		t.Fatalf("events = %#v", inbox.events)
	}
	transcribed, ok := inbox.events[0].Event.(domain.Transcribed)
	if !ok || transcribed.Text != " speak clearly " || inbox.events[0].Scope != scope {
		t.Fatalf("event = %#v", inbox.events[0])
	}
	if stt.request.MIME != "audio/webm" || string(stt.request.Audio) != "header-and-audio" || len(stt.request.Keyterms) != 1 || stt.request.Meta.Seat != 2 {
		t.Fatalf("request = %#v", stt.request)
	}
}

func TestTranscriber_Execute_postsFailureKinds(t *testing.T) {
	tests := []struct {
		name          string
		stt           *fakeSTT
		makeRecording bool
		want          vocab.ErrKind
	}{
		{name: "missing recording", stt: &fakeSTT{}, want: vocab.ErrBadOutput},
		{name: "empty transcript", stt: &fakeSTT{result: ports.Transcript{Text: "  "}}, makeRecording: true, want: vocab.ErrBadOutput},
		{name: "adapter error", stt: &fakeSTT{err: &ports.CallError{Kind: vocab.ErrRateLimited}}, makeRecording: true, want: vocab.ErrRateLimited},
		{name: "unknown error", stt: &fakeSTT{err: errors.New("offline")}, makeRecording: true, want: vocab.ErrUnavailable},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assembler := NewAssembler()
			session := Session{UtteranceID: domain.UtteranceID(tc.name), MIME: "audio/mp4"}
			if tc.makeRecording {
				if err := assembler.Start(context.Background(), session); err != nil {
					t.Fatal(err)
				}
				if err := assembler.Chunk(context.Background(), Chunk{Session: session, Data: []byte("audio")}); err != nil {
					t.Fatal(err)
				}
				if _, err := assembler.End(context.Background(), session); err != nil {
					t.Fatal(err)
				}
			}
			executor, err := NewTranscriber(tc.stt, assembler)
			if err != nil {
				t.Fatal(err)
			}
			inbox := &eventInbox{}
			executor.Execute(context.Background(), domain.Transcribe{UtteranceID: session.UtteranceID}, domain.Scope{}, inbox)
			failed, ok := inbox.events[0].Event.(domain.STTError)
			if !ok || failed.FailureKind != tc.want {
				t.Fatalf("event = %#v", inbox.events)
			}
		})
	}
}

func TestNewTranscriber_validatesDependenciesAndCancellation(t *testing.T) {
	assembler := NewAssembler()
	if _, err := NewTranscriber(nil, assembler); err == nil {
		t.Fatal("nil STT accepted")
	}
	if _, err := NewTranscriber(&fakeSTT{}, nil); err == nil {
		t.Fatal("nil assembler accepted")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	inbox := &eventInbox{}
	executor, err := NewTranscriber(&fakeSTT{}, assembler)
	if err != nil {
		t.Fatal(err)
	}
	executor.Execute(ctx, domain.Transcribe{UtteranceID: "u"}, domain.Scope{}, inbox)
	failed, ok := inbox.events[0].Event.(domain.STTError)
	if !ok || failed.FailureKind != vocab.ErrCanceled {
		t.Fatalf("event = %#v", inbox.events)
	}
}
