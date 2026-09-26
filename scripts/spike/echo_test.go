package main

import (
	"context"
	"errors"
	"io"
	"testing"

	spikev1 "github.com/monstercameron/DungeonFlux/scripts/spike/gen"
	"google.golang.org/grpc/metadata"
)

type fakeEchoStream struct {
	spikev1.Echo_StreamServer
	chunks []*spikev1.AudioChunk
	sent   []*spikev1.AudioChunk
	err    error
}

func (s *fakeEchoStream) Recv() (*spikev1.AudioChunk, error) {
	if len(s.chunks) == 0 {
		if s.err != nil {
			return nil, s.err
		}
		return nil, io.EOF
	}
	chunk := s.chunks[0]
	s.chunks = s.chunks[1:]
	return chunk, nil
}

func (s *fakeEchoStream) Send(chunk *spikev1.AudioChunk) error {
	s.sent = append(s.sent, chunk)
	return nil
}

func (s *fakeEchoStream) Context() context.Context     { return context.Background() }
func (s *fakeEchoStream) SetHeader(metadata.MD) error  { return nil }
func (s *fakeEchoStream) SendHeader(metadata.MD) error { return nil }
func (s *fakeEchoStream) SetTrailer(metadata.MD)       {}
func (s *fakeEchoStream) SendMsg(any) error            { return nil }
func (s *fakeEchoStream) RecvMsg(any) error            { return nil }

func TestEchoServer_Stream_echoesChunksUntilFinal(t *testing.T) {
	chunks := []*spikev1.AudioChunk{
		{Sequence: 1, SampleRate: 24000, PcmS16Le: []byte{1, 2}},
		{Sequence: 2, SampleRate: 24000, PcmS16Le: []byte{3, 4}, Final: true},
	}
	stream := &fakeEchoStream{chunks: chunks}

	if err := (echoServer{}).Stream(stream); err != nil {
		t.Fatalf("Stream() error = %v", err)
	}
	if len(stream.sent) != len(chunks) {
		t.Fatalf("sent %d chunks, want %d", len(stream.sent), len(chunks))
	}
	for i, want := range chunks {
		got := stream.sent[i]
		if got.GetSequence() != want.GetSequence() || string(got.GetPcmS16Le()) != string(want.GetPcmS16Le()) || got.GetFinal() != want.GetFinal() {
			t.Errorf("chunk %d = %#v, want %#v", i, got, want)
		}
	}
}

func TestEchoServer_Stream_returnsReceiveError(t *testing.T) {
	want := errors.New("receive failed")
	stream := &fakeEchoStream{err: want}

	if err := (echoServer{}).Stream(stream); !errors.Is(err, want) {
		t.Fatalf("Stream() error = %v, want %v", err, want)
	}
}
