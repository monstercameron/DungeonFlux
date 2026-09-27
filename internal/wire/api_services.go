package wire

import (
	"context"
	"io"
	"strconv"
	"sync"

	df "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
	"github.com/monstercameron/DungeonFlux/internal/api"
	"github.com/monstercameron/DungeonFlux/internal/domain"
	voicein "github.com/monstercameron/DungeonFlux/internal/voice/in"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// sessionService combines the session methods whose implementations are split
// across API concerns, and supplies the watch stream at the composition root.
type sessionService struct {
	df.UnimplementedSessionServiceServer
	*api.SessionServer
	*api.ReportServer
	watch *api.WatchHub
	room  string
	dm    string
	host  string
}

func (s *sessionService) Join(ctx context.Context, req *df.JoinRequest) (*df.JoinResponse, error) {
	return s.SessionServer.Join(ctx, req)
}

func (s *sessionService) Act(ctx context.Context, req *df.ActRequest) (*df.ActResponse, error) {
	return s.SessionServer.Act(ctx, req)
}

func (s *sessionService) Say(ctx context.Context, req *df.SayRequest) (*df.SayResponse, error) {
	return s.SessionServer.Say(ctx, req)
}

func (s *sessionService) Report(ctx context.Context, req *df.ReportRequest) (*df.ReportResponse, error) {
	return s.ReportServer.Report(ctx, req)
}

func (s *sessionService) Watch(req *df.WatchRequest, stream df.SessionService_WatchServer) error {
	ctx := stream.Context()
	if req == nil || stream == nil {
		return status.Error(codes.InvalidArgument, "watch request is required")
	}
	join, err := s.SessionServer.Join(ctx, &df.JoinRequest{RoomCode: s.room, SeatToken: req.GetSeatToken(), DmToken: req.GetSeatToken(), HostToken: req.GetSeatToken()})
	if err != nil {
		return err
	}
	kind := df.ClientKind_CLIENT_KIND_PHONE
	seatNumber, _ := strconv.Atoi(join.GetSeatId())
	seat := domain.SeatID(seatNumber)
	if req.GetSeatToken() == s.dm {
		kind = df.ClientKind_CLIENT_KIND_DM
		seat = 0
	} else if req.GetSeatToken() == s.host {
		kind = df.ClientKind_CLIENT_KIND_HOST
		seat = 0
	}
	sub := s.watch.Subscribe(ctx, kind, seat)
	defer sub.Close()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case message, ok := <-sub.Messages():
			if !ok {
				return nil
			}
			if err := stream.Send(message); err != nil {
				return err
			}
		}
	}
}

// audioService exposes the bounded PCM listener hub through the protobuf API.
type audioService struct {
	df.UnimplementedAudioServiceServer
	mu  sync.Mutex
	hub *api.ListenHub
}

func (s *audioService) Listen(req *df.ListenRequest, stream df.AudioService_ListenServer) error {
	if req == nil || stream == nil || s == nil || s.hub == nil {
		return status.Error(codes.InvalidArgument, "listen request is required")
	}
	if req.GetSeatToken() == "" {
		return status.Error(codes.Unauthenticated, "seat token is required")
	}
	s.mu.Lock()
	sub := s.hub.Subscribe(stream.Context())
	if err := stream.SendHeader(metadata.MD{}); err != nil {
		s.mu.Unlock()
		sub.Close()
		return err
	}
	s.mu.Unlock()
	defer sub.Close()
	for {
		select {
		case <-stream.Context().Done():
			return stream.Context().Err()
		case <-sub.Done():
			stream.SetTrailer(metadata.Pairs("dungeonflux-listen", "replaced"))
			return io.EOF
		case frame, ok := <-sub.Frames():
			if !ok {
				return nil
			}
			message := &df.AudioMessage{Message: &df.AudioMessage_Frame{Frame: &df.AudioFrame{
				UtteranceId: string(frame.UtteranceID), Speaker: frame.Speaker, Seq: uint64(frame.Seq),
				SampleRate: int32(frame.SampleRate), PcmS16Le: frame.PCMS16LE, Final: frame.Final,
			}}}
			if err := stream.Send(message); err != nil {
				return err
			}
		}
	}
}

// assemblerTalkSink feeds push-to-talk audio into the assembler the
// transcriber reads. The Talk server used noopTalkSink, which dropped every
// chunk, so every Transcribe found no recording and no STT call was made.
type assemblerTalkSink struct{ assembler *voicein.Assembler }

func (s assemblerTalkSink) Start(ctx context.Context, session api.TalkSession) error {
	return s.assembler.Start(ctx, voiceSession(session))
}

func (s assemblerTalkSink) Chunk(ctx context.Context, chunk api.TalkChunk) error {
	return s.assembler.Chunk(ctx, voicein.Chunk{Session: voiceSession(chunk.Session), Seq: chunk.Seq, Data: chunk.Data})
}

func (s assemblerTalkSink) End(ctx context.Context, session api.TalkSession) error {
	_, err := s.assembler.End(ctx, voiceSession(session))
	return err
}

func (s assemblerTalkSink) Cancel(ctx context.Context, session api.TalkSession) error {
	return s.assembler.Cancel(ctx, session.UtteranceID)
}

func voiceSession(session api.TalkSession) voicein.Session {
	return voicein.Session{Seat: session.Seat, UtteranceID: session.UtteranceID, MIME: session.MIME}
}

var _ api.TalkSink = assemblerTalkSink{}

type hubAudioOut struct{ hub *api.ListenHub }

func (a hubAudioOut) Frame(frame domain.AudioFrame) { a.hub.Frame(frame) }
func (a hubAudioOut) Cancel(id domain.UtteranceID)  { a.hub.Cancel(id) }

var _ interface {
	Frame(domain.AudioFrame)
	Cancel(domain.UtteranceID)
} = hubAudioOut{}
