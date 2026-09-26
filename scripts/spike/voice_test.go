package main

import (
	"context"
	"errors"
	"io"
	"testing"

	spikev1 "github.com/monstercameron/DungeonFlux/scripts/spike/gen"
	"google.golang.org/grpc/metadata"
)

type fakeTalkStream struct {
	spikev1.Voice_TalkServer
	chunks []*spikev1.AudioChunk
	result *spikev1.Transcript
}

func (s *fakeTalkStream) Recv() (*spikev1.AudioChunk, error) {
	if len(s.chunks) == 0 {
		return nil, io.EOF
	}
	chunk := s.chunks[0]
	s.chunks = s.chunks[1:]
	return chunk, nil
}
func (s *fakeTalkStream) SendAndClose(result *spikev1.Transcript) error {
	s.result = result
	return nil
}
func (s *fakeTalkStream) Context() context.Context     { return context.Background() }
func (s *fakeTalkStream) SetHeader(metadata.MD) error  { return nil }
func (s *fakeTalkStream) SendHeader(metadata.MD) error { return nil }
func (s *fakeTalkStream) SetTrailer(metadata.MD)       {}
func (s *fakeTalkStream) SendMsg(any) error            { return nil }
func (s *fakeTalkStream) RecvMsg(any) error            { return nil }

type fakeListenStream struct {
	spikev1.Voice_ListenServer
	frames []*spikev1.PCMFrame
}

func (s *fakeListenStream) Send(frame *spikev1.PCMFrame) error {
	s.frames = append(s.frames, frame)
	return nil
}
func (s *fakeListenStream) Context() context.Context     { return context.Background() }
func (s *fakeListenStream) SetHeader(metadata.MD) error  { return nil }
func (s *fakeListenStream) SendHeader(metadata.MD) error { return nil }
func (s *fakeListenStream) SetTrailer(metadata.MD)       {}
func (s *fakeListenStream) SendMsg(any) error            { return nil }
func (s *fakeListenStream) RecvMsg(any) error            { return nil }

func TestVoiceServer_TalkTranscribesAndPublishesPCM(t *testing.T) {
	hub := newAudioHub()
	stream := &fakeTalkStream{chunks: []*spikev1.AudioChunk{{PcmS16Le: []byte("header")}, {PcmS16Le: []byte("body"), Final: true}}}
	server := voiceServer{transcriber: fakeTranscriber{text: "open the door"}, hub: hub}
	if err := server.Talk(stream); err != nil {
		t.Fatalf("Talk() error = %v", err)
	}
	if stream.result.GetText() != "open the door" {
		t.Fatalf("transcript = %q", stream.result.GetText())
	}
	frames, _ := hub.snapshot()
	if len(frames) != 1 || !frames[0].GetFinal() {
		t.Fatalf("published frames = %#v", frames)
	}
}

func TestVoiceServer_TalkReturnsTranscriberError(t *testing.T) {
	want := errors.New("stt failed")
	transcriber := transcriberFunc(func(context.Context, string, []byte) (string, error) { return "", want })
	stream := &fakeTalkStream{chunks: []*spikev1.AudioChunk{{Final: true}}}
	server := voiceServer{transcriber: transcriber, hub: newAudioHub()}
	if err := server.Talk(stream); !errors.Is(err, want) {
		t.Fatalf("Talk() error = %v, want %v", err, want)
	}
}

func TestVoiceServer_ListenStreamsPublishedPCM(t *testing.T) {
	hub := newAudioHub()
	hub.publish(fakePCM("x"))
	stream := &fakeListenStream{}
	if err := (voiceServer{hub: hub}).Listen(&spikev1.ListenRequest{}, stream); err != nil {
		t.Fatalf("Listen() error = %v", err)
	}
	if len(stream.frames) != 1 || len(stream.frames[0].GetPcmS16Le()) == 0 {
		t.Fatalf("frames = %#v", stream.frames)
	}
}

type transcriberFunc func(context.Context, string, []byte) (string, error)

func (f transcriberFunc) Transcribe(ctx context.Context, typ string, audio []byte) (string, error) {
	return f(ctx, typ, audio)
}
