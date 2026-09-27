package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"

	df "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/ports"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// TalkSession identifies one client-owned recording delivered to the voice
// pipeline.
type TalkSession struct {
	Seat        domain.SeatID
	UtteranceID domain.UtteranceID
	MIME        string
	Locale      string
}

// TalkChunk is one immutable recorded-media chunk from a Talk stream.
type TalkChunk struct {
	Session TalkSession
	Seq     uint64
	Data    []byte
}

// TalkSink receives an assembled Talk stream's lifecycle in order. Methods
// must return promptly when ctx is cancelled; the sink owns any later STT
// work.
type TalkSink interface {
	Start(context.Context, TalkSession) error
	Chunk(context.Context, TalkChunk) error
	End(context.Context, TalkSession) error
	Cancel(context.Context, TalkSession) error
}

// TalkServer receives phone microphone chunks and forwards them to voice/in.
type TalkServer struct {
	df.UnimplementedVoiceServiceServer
	inbox   ports.Inbox
	session *SessionServer
	sink    TalkSink
}

// NewTalkServer creates a voice service backed by the room inbox and sink.
func NewTalkServer(inbox ports.Inbox, session *SessionServer, sink TalkSink) (*TalkServer, error) {
	if inbox == nil {
		return nil, errors.New("api: talk inbox is required")
	}
	if session == nil {
		return nil, errors.New("api: talk session is required")
	}
	if sink == nil {
		return nil, errors.New("api: talk sink is required")
	}
	return &TalkServer{inbox: inbox, session: session, sink: sink}, nil
}

// Talk accepts one recording per stream. The first message must be TalkStart,
// chunks are acknowledged only after the sink accepts them, and TalkEnd
// closes the stream after the room event is posted.
func (s *TalkServer) Talk(stream df.VoiceService_TalkServer) error {
	if stream == nil {
		return status.Error(codes.InvalidArgument, "talk stream is required")
	}
	request, err := stream.Recv()
	if err != nil {
		return talkRecvError(stream.Context(), err)
	}
	start := request.GetStart()
	if start == nil {
		return status.Error(codes.InvalidArgument, "talk must start with TalkStart")
	}
	seat, ok := s.seatForToken(start.GetSeatToken())
	if !ok {
		return status.Error(codes.PermissionDenied, "seat token is invalid")
	}
	session := TalkSession{Seat: seat.id, UtteranceID: newUtteranceID(), MIME: start.GetMimeType(), Locale: s.session.LocaleFor(start.GetSeatToken())}
	if err := s.sink.Start(stream.Context(), session); err != nil {
		return talkSinkError(err)
	}
	if !s.post(stream.Context(), domain.TalkStart{Seat: seat.id, UtteranceID: session.UtteranceID, MIME: session.MIME}) {
		_ = s.sink.Cancel(stream.Context(), session)
		return status.Error(codes.ResourceExhausted, "room inbox is full")
	}
	return s.receiveTalk(stream, session)
}

func (s *TalkServer) receiveTalk(stream df.VoiceService_TalkServer, session TalkSession) error {
	for {
		request, err := stream.Recv()
		if err != nil {
			if stream.Context().Err() != nil {
				return s.cancelTalk(stream.Context(), session)
			}
			return talkRecvError(stream.Context(), err)
		}
		switch {
		case request.GetChunk() != nil:
			if err := s.receiveChunk(stream, session, request.GetChunk()); err != nil {
				return err
			}
		case request.GetEnd() != nil:
			return s.endTalk(stream, session)
		default:
			return status.Error(codes.InvalidArgument, "talk message must contain a chunk or end")
		}
	}
}

func (s *TalkServer) receiveChunk(stream df.VoiceService_TalkServer, session TalkSession, chunk *df.AudioChunk) error {
	if len(chunk.GetData()) == 0 {
		return status.Error(codes.InvalidArgument, "audio chunk is empty")
	}
	item := TalkChunk{Session: session, Seq: chunk.GetSeq(), Data: append([]byte(nil), chunk.GetData()...)}
	if err := s.sink.Chunk(stream.Context(), item); err != nil {
		return talkSinkError(err)
	}
	if err := stream.Send(&df.TalkResponse{Message: &df.TalkResponse_Ack{Ack: &df.ChunkAck{Seq: chunk.GetSeq()}}}); err != nil {
		return talkSendError(err)
	}
	return nil
}

func (s *TalkServer) endTalk(stream df.VoiceService_TalkServer, session TalkSession) error {
	if err := s.sink.End(stream.Context(), session); err != nil {
		return talkSinkError(err)
	}
	if !s.post(stream.Context(), domain.TalkEnd{Seat: session.Seat, UtteranceID: session.UtteranceID}) {
		return status.Error(codes.ResourceExhausted, "room inbox is full")
	}
	return nil
}

func (s *TalkServer) cancelTalk(ctx context.Context, session TalkSession) error {
	_ = s.sink.Cancel(ctx, session)
	s.post(context.WithoutCancel(ctx), domain.StreamClosed{Seat: session.Seat, Stream: vocab.StreamTalk})
	return nil
}

func (s *TalkServer) post(ctx context.Context, event domain.Event) bool {
	return s.inbox.Post(ctx, domain.Envelope{Event: event})
}

func (s *TalkServer) seatForToken(token string) (seatSession, bool) {
	return s.session.seatForToken(token)
}

func newUtteranceID() domain.UtteranceID {
	data := make([]byte, 12)
	if _, err := rand.Read(data); err != nil {
		return domain.UtteranceID("utterance-unknown")
	}
	return domain.UtteranceID("utterance-" + hex.EncodeToString(data))
}

func talkRecvError(ctx context.Context, err error) error {
	if ctx != nil && ctx.Err() != nil {
		return nil
	}
	return status.Errorf(codes.Unknown, "receive talk message: %v", err)
}

func talkSinkError(err error) error {
	return status.Errorf(codes.Internal, "voice pipeline: %v", err)
}

func talkSendError(err error) error {
	return status.Errorf(codes.Unavailable, "send talk response: %v", err)
}
