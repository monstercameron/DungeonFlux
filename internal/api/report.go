package api

import (
	"context"
	"errors"
	"sync"

	df "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/ports"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ClientReport retains API-local diagnostics that must not enter the engine.
type ClientReport struct {
	SeatToken string
	Kind      vocab.ReportKind
	ID        string
	State     string
	PNG       []byte
}

// ReportServer handles client playback and renderer reports.
type ReportServer struct {
	df.UnimplementedSessionServiceServer
	inbox   ports.Inbox
	mu      sync.Mutex
	clients []ClientReport
}

// NewReportServer creates a report handler backed by the room inbox.
func NewReportServer(inbox ports.Inbox) (*ReportServer, error) {
	if inbox == nil {
		return nil, errors.New("api: report inbox is required")
	}
	return &ReportServer{inbox: inbox}, nil
}

// Report accepts a client report and either posts an engine event or stores an
// API-local diagnostic, depending on the report kind.
func (s *ReportServer) Report(ctx context.Context, request *df.ReportRequest) (*df.ReportResponse, error) {
	if request == nil {
		return nil, status.Error(codes.InvalidArgument, "report request is required")
	}
	kind, ok := reportKind(request.GetKind())
	if !ok {
		return nil, status.Error(codes.InvalidArgument, "report kind is not supported")
	}
	if kind == vocab.ReportClientState || kind == vocab.ReportClientScreenshot {
		s.storeClientReport(request, kind)
		return &df.ReportResponse{}, nil
	}
	if !s.inbox.Post(ctx, domain.Envelope{Event: domain.Report{ReportKind: kind, ID: request.GetId()}}) {
		return nil, status.Error(codes.ResourceExhausted, "room inbox is full")
	}
	return &df.ReportResponse{}, nil
}

// ClientReports returns a snapshot of API-local diagnostic reports.
func (s *ReportServer) ClientReports() []ClientReport {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]ClientReport, len(s.clients))
	copy(out, s.clients)
	for index := range out {
		out[index].PNG = append([]byte(nil), out[index].PNG...)
	}
	return out
}

func (s *ReportServer) storeClientReport(request *df.ReportRequest, kind vocab.ReportKind) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.clients = append(s.clients, ClientReport{SeatToken: request.GetSeatToken(), Kind: kind,
		ID: request.GetId(), State: request.GetClientState(), PNG: append([]byte(nil), request.GetPng()...)})
}

func reportKind(kind df.ReportKind) (vocab.ReportKind, bool) {
	switch kind {
	case df.ReportKind_REPORT_KIND_PLAYBACK_DONE:
		return vocab.ReportPlaybackDone, true
	case df.ReportKind_REPORT_KIND_CLIP_ENDED:
		return vocab.ReportClipEnded, true
	case df.ReportKind_REPORT_KIND_SPLAT_READY:
		return vocab.ReportSplatReady, true
	case df.ReportKind_REPORT_KIND_SPLAT_FAILED:
		return vocab.ReportSplatFailed, true
	case df.ReportKind_REPORT_KIND_CLIENT_STATE:
		return vocab.ReportClientState, true
	case df.ReportKind_REPORT_KIND_CLIENT_SCREENSHOT:
		return vocab.ReportClientScreenshot, true
	default:
		return "", false
	}
}
