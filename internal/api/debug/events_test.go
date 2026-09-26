package debug

import (
	"context"
	"testing"

	df "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/fakes"
	"github.com/monstercameron/DungeonFlux/internal/ports"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type eventEngine struct {
	*fakes.FakeEngine
	log *fakes.FakeEventLog
}

func (e *eventEngine) DebugEventSource() eventSource { return eventReader{log: e.log} }

type eventReader struct{ log *fakes.FakeEventLog }

func (r eventReader) EventLog() ports.EventLog  { return r.log }
func (r eventReader) RunID(string) domain.RunID { return "run-1" }

type eventStream struct {
	ctx     context.Context
	records []*df.EventRecord
}

func (s *eventStream) Send(record *df.EventRecord) error {
	s.records = append(s.records, record)
	return nil
}
func (s *eventStream) SetHeader(metadata.MD) error  { return nil }
func (s *eventStream) SendHeader(metadata.MD) error { return nil }
func (s *eventStream) SetTrailer(metadata.MD)       {}
func (s *eventStream) Context() context.Context     { return s.ctx }
func (s *eventStream) SendMsg(any) error            { return nil }
func (s *eventStream) RecvMsg(any) error            { return nil }

var _ df.DebugService_EventsServer = (*eventStream)(nil)
var _ grpc.ServerStream = (*eventStream)(nil)

func TestEvents_StreamsRecordsAfterSequence(t *testing.T) {
	log := &fakes.FakeEventLog{Records: []domain.LogRecord{{Seq: 1, Run: "run-1", Kind: "join"}, {Seq: 2, Run: "run-1", Kind: "act"}}}
	server, err := NewServer(&eventEngine{FakeEngine: &fakes.FakeEngine{}, log: log}, &fakes.FakeInbox{})
	if err != nil {
		t.Fatal(err)
	}
	stream := &eventStream{ctx: context.Background()}
	if err := server.Events(&df.EventsRequest{Room: "room", Since: 1}, stream); err != nil {
		t.Fatal(err)
	}
	if len(stream.records) != 1 || stream.records[0].GetSeq() != 2 {
		t.Fatalf("records = %+v", stream.records)
	}
}
