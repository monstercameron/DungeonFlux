package debug

import (
	"context"
	"log/slog"
	"testing"
	"time"

	df "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
	"github.com/monstercameron/DungeonFlux/internal/fakes"
	"google.golang.org/grpc/metadata"
)

type logEngine struct {
	*fakes.FakeEngine
	records []slog.Record
}

func (e *logEngine) DebugLogSource() logSource { return e }
func (e *logEngine) LogRecords() []slog.Record { return append([]slog.Record(nil), e.records...) }

type logStream struct {
	ctx     context.Context
	records []*df.LogRecord
}

func (s *logStream) Send(record *df.LogRecord) error {
	s.records = append(s.records, record)
	return nil
}
func (s *logStream) SetHeader(metadata.MD) error  { return nil }
func (s *logStream) SendHeader(metadata.MD) error { return nil }
func (s *logStream) SetTrailer(metadata.MD)       {}
func (s *logStream) Context() context.Context     { return s.ctx }
func (s *logStream) SendMsg(any) error            { return nil }
func (s *logStream) RecvMsg(any) error            { return nil }

func TestLogs_FiltersLevelAndTrace(t *testing.T) {
	record := slog.NewRecord(time.Unix(0, 0), slog.LevelWarn, "kept", 0)
	record.AddAttrs(slog.String("trace_id", "trace-1"), slog.String("room", "room"))
	ignored := slog.NewRecord(time.Unix(0, 0), slog.LevelError, "other", 0)
	engine := &logEngine{FakeEngine: &fakes.FakeEngine{}, records: []slog.Record{record, ignored}}
	server, err := NewServer(engine, &fakes.FakeInbox{})
	if err != nil {
		t.Fatal(err)
	}
	stream := &logStream{ctx: context.Background()}
	if err := server.Logs(&df.LogsRequest{Level: "warn", TraceId: "trace-1"}, stream); err != nil {
		t.Fatal(err)
	}
	if len(stream.records) != 1 || stream.records[0].GetMessage() != "kept" {
		t.Fatalf("records = %+v", stream.records)
	}
}
