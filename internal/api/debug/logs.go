package debug

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"time"

	df "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
)

type logSource interface {
	LogRecords() []slog.Record
}

type logSourceEngine interface {
	DebugLogSource() logSource
}

func (s *Server) logSource() (logSource, bool) {
	provider, ok := s.engine.(logSourceEngine)
	if !ok || provider.DebugLogSource() == nil {
		return nil, false
	}
	return provider.DebugLogSource(), true
}

func streamLogs(ctx context.Context, source logSource, request *df.LogsRequest, stream df.DebugService_LogsServer) error {
	seen := 0
	for {
		records := source.LogRecords()
		if seen > len(records) {
			seen = 0
		}
		for _, record := range records[seen:] {
			if matchesLog(record, request.GetLevel(), request.GetTraceId()) {
				if err := stream.Send(logRecord(record)); err != nil {
					return err
				}
			}
		}
		seen = len(records)
		if !request.GetFollow() {
			return nil
		}
		timer := time.NewTimer(25 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
}

func matchesLog(record slog.Record, requested, trace string) bool {
	if requested != "" && !strings.EqualFold(requested, record.Level.String()) {
		return false
	}
	if trace == "" {
		return true
	}
	found := false
	record.Attrs(func(attr slog.Attr) bool {
		if attr.Key == "trace_id" && attr.Value.String() == trace {
			found = true
		}
		return true
	})
	return found
}

func logRecord(record slog.Record) *df.LogRecord {
	attributes := make(map[string]any)
	record.Attrs(func(attr slog.Attr) bool { attributes[attr.Key] = attr.Value.Any(); return true })
	encoded, _ := json.Marshal(attributes)
	return &df.LogRecord{Time: record.Time.UTC().Format(time.RFC3339Nano), Level: record.Level.String(), Message: record.Message, AttributesJson: string(encoded)}
}
