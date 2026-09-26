package api

import (
	"context"
	"errors"
	"io"
	"testing"

	df "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/fakes"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type talkSink struct {
	started  []TalkSession
	chunks   []TalkChunk
	ended    []TalkSession
	canceled []TalkSession
	err      error
}

func (s *talkSink) Start(_ context.Context, session TalkSession) error {
	s.started = append(s.started, session)
	return s.err
}

func (s *talkSink) Chunk(_ context.Context, chunk TalkChunk) error {
	s.chunks = append(s.chunks, chunk)
	return s.err
}

func (s *talkSink) End(_ context.Context, session TalkSession) error {
	s.ended = append(s.ended, session)
	return s.err
}

func (s *talkSink) Cancel(_ context.Context, session TalkSession) error {
	s.canceled = append(s.canceled, session)
	return s.err
}

type talkStream struct {
	ctx       context.Context
	requests  []*df.TalkRequest
	responses []*df.TalkResponse
	recvErr   error
	onEmpty   func()
}

func (s *talkStream) Recv() (*df.TalkRequest, error) {
	if len(s.requests) == 0 {
		if s.onEmpty != nil {
			s.onEmpty()
		}
		if s.recvErr != nil {
			return nil, s.recvErr
		}
		return nil, io.EOF
	}
	request := s.requests[0]
	s.requests = s.requests[1:]
	return request, nil
}

func (s *talkStream) Send(response *df.TalkResponse) error {
	s.responses = append(s.responses, response)
	return nil
}

func (s *talkStream) SetHeader(metadata.MD) error  { return nil }
func (s *talkStream) SendHeader(metadata.MD) error { return nil }
func (s *talkStream) SetTrailer(metadata.MD)       {}
func (s *talkStream) Context() context.Context     { return s.ctx }
func (s *talkStream) SendMsg(any) error            { return nil }
func (s *talkStream) RecvMsg(any) error            { return nil }

func TestTalkServer_TalkForwardsChunksAndPostsLifecycle(t *testing.T) {
	inbox := &fakes.FakeInbox{PostResult: true}
	session, err := NewSessionServer(inbox, "ROOM", "HOST", "DM")
	if err != nil {
		t.Fatal(err)
	}
	join, err := session.Join(context.Background(), &df.JoinRequest{RoomCode: "ROOM", Kind: df.ClientKind_CLIENT_KIND_PHONE})
	if err != nil {
		t.Fatal(err)
	}
	sink := &talkSink{}
	server, err := NewTalkServer(inbox, session, sink)
	if err != nil {
		t.Fatal(err)
	}
	stream := &talkStream{ctx: context.Background(), requests: []*df.TalkRequest{
		{Message: &df.TalkRequest_Start{Start: &df.TalkStart{SeatToken: join.GetSeatToken(), MimeType: "audio/webm"}}},
		{Message: &df.TalkRequest_Chunk{Chunk: &df.AudioChunk{Seq: 4, Data: []byte{1, 2}}}},
		{Message: &df.TalkRequest_End{End: &df.TalkEnd{}}},
	}}
	if err := server.Talk(stream); err != nil {
		t.Fatal(err)
	}
	if len(sink.started) != 1 || len(sink.chunks) != 1 || len(sink.ended) != 1 {
		t.Fatalf("sink lifecycle = %d/%d/%d", len(sink.started), len(sink.chunks), len(sink.ended))
	}
	if sink.chunks[0].Seq != 4 || string(sink.chunks[0].Data) != "\x01\x02" {
		t.Fatalf("chunk = %+v", sink.chunks[0])
	}
	if len(stream.responses) != 1 || stream.responses[0].GetAck().GetSeq() != 4 {
		t.Fatalf("responses = %+v", stream.responses)
	}
	if len(inbox.Calls) != 3 {
		t.Fatalf("inbox calls = %d, want join/start/end", len(inbox.Calls))
	}
	if _, ok := inbox.Calls[1].Envelope.Event.(domain.TalkStart); !ok {
		t.Fatalf("start event = %T", inbox.Calls[1].Envelope.Event)
	}
	if _, ok := inbox.Calls[2].Envelope.Event.(domain.TalkEnd); !ok {
		t.Fatalf("end event = %T", inbox.Calls[2].Envelope.Event)
	}
}

func TestTalkServer_TalkCancelsOnContextClose(t *testing.T) {
	inbox := &fakes.FakeInbox{PostResult: true}
	session, _ := NewSessionServer(inbox, "ROOM", "HOST", "DM")
	join, _ := session.Join(context.Background(), &df.JoinRequest{RoomCode: "ROOM", Kind: df.ClientKind_CLIENT_KIND_PHONE})
	sink := &talkSink{}
	server, _ := NewTalkServer(inbox, session, sink)
	ctx, cancel := context.WithCancel(context.Background())
	stream := &talkStream{ctx: ctx, requests: []*df.TalkRequest{{Message: &df.TalkRequest_Start{Start: &df.TalkStart{SeatToken: join.GetSeatToken()}}}}, recvErr: context.Canceled, onEmpty: cancel}
	if err := server.Talk(stream); err != nil {
		t.Fatal(err)
	}
	if len(sink.canceled) != 1 {
		t.Fatalf("canceled sessions = %d", len(sink.canceled))
	}
}

func TestTalkServer_RejectsInvalidMessagesAndTokens(t *testing.T) {
	inbox := &fakes.FakeInbox{PostResult: true}
	session, _ := NewSessionServer(inbox, "ROOM", "HOST", "DM")
	sink := &talkSink{}
	server, _ := NewTalkServer(inbox, session, sink)
	tests := []struct {
		name   string
		stream *talkStream
	}{
		{"nil first message", &talkStream{ctx: context.Background(), requests: []*df.TalkRequest{{}}}},
		{"bad token", &talkStream{ctx: context.Background(), requests: []*df.TalkRequest{{Message: &df.TalkRequest_Start{Start: &df.TalkStart{SeatToken: "nope"}}}}}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := server.Talk(test.stream); err == nil {
				t.Fatal("Talk returned nil error")
			}
		})
	}
	if err := server.Talk(nil); err == nil {
		t.Fatal("nil stream was accepted")
	}
}

func TestTalkServer_ReportsSinkAndInboxFailures(t *testing.T) {
	inbox := &fakes.FakeInbox{PostResult: true}
	session, _ := NewSessionServer(inbox, "ROOM", "HOST", "DM")
	join, _ := session.Join(context.Background(), &df.JoinRequest{RoomCode: "ROOM", Kind: df.ClientKind_CLIENT_KIND_PHONE})
	sink := &talkSink{err: errors.New("sink failed")}
	server, _ := NewTalkServer(inbox, session, sink)
	stream := &talkStream{ctx: context.Background(), requests: []*df.TalkRequest{{Message: &df.TalkRequest_Start{Start: &df.TalkStart{SeatToken: join.GetSeatToken()}}}}}
	if err := server.Talk(stream); err == nil {
		t.Fatal("sink error was hidden")
	}

	goodSink := &talkSink{}
	fullInbox := &fakes.FakeInbox{PostResult: true}
	fullSession, _ := NewSessionServer(fullInbox, "ROOM", "HOST", "DM")
	fullJoin, _ := fullSession.Join(context.Background(), &df.JoinRequest{RoomCode: "ROOM", Kind: df.ClientKind_CLIENT_KIND_PHONE})
	fullInbox.PostResult = false
	fullServer, _ := NewTalkServer(fullInbox, fullSession, goodSink)
	fullStream := &talkStream{ctx: context.Background(), requests: []*df.TalkRequest{{Message: &df.TalkRequest_Start{Start: &df.TalkStart{SeatToken: fullJoin.GetSeatToken()}}}}}
	if err := fullServer.Talk(fullStream); err == nil {
		t.Fatal("full inbox error was hidden")
	}
}

var _ grpc.ServerStream = (*talkStream)(nil)
