package debug

import (
	"context"
	"encoding/json"
	"time"

	df "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/ports"
)

type eventSource interface {
	EventLog() ports.EventLog
	RunID(room string) domain.RunID
}

type eventSourceEngine interface {
	DebugEventSource() eventSource
}

func (s *Server) eventSource() (eventSource, bool) {
	provider, ok := s.engine.(eventSourceEngine)
	if !ok || provider.DebugEventSource() == nil {
		return nil, false
	}
	return provider.DebugEventSource(), true
}

func streamEvents(ctx context.Context, source eventSource, request *df.EventsRequest, stream df.DebugService_EventsServer) error {
	seq := request.GetSince()
	for {
		found := false
		for record, err := range source.EventLog().Read(ctx, source.RunID(request.GetRoom())) {
			if err != nil {
				return err
			}
			if record.Seq <= seq {
				continue
			}
			if err := stream.Send(eventRecord(record)); err != nil {
				return err
			}
			seq = record.Seq
			found = true
		}
		if !request.GetFollow() {
			return nil
		}
		if err := waitForRecord(ctx, found); err != nil {
			return err
		}
	}
}

func waitForRecord(ctx context.Context, found bool) error {
	delay := 25 * time.Millisecond
	if found {
		delay = 5 * time.Millisecond
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func eventRecord(record domain.LogRecord) *df.EventRecord {
	payload, _ := json.Marshal(record.Event)
	return &df.EventRecord{Seq: record.Seq, TMs: record.At.Milliseconds(), Event: string(record.Kind), PayloadJson: string(payload)}
}
