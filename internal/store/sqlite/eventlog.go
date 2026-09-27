package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"iter"
	"time"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

// Append enqueues log records for the writer. It returns without waiting for SQLite.
func (s *Store) Append(ctx context.Context, records []domain.LogRecord) error {
	if len(records) == 0 {
		return nil
	}
	if s == nil || s.writer == nil {
		return fmt.Errorf("append events: closed store")
	}
	copyRecords := append([]domain.LogRecord(nil), records...)
	if err := ctx.Err(); err != nil {
		return err
	}
	// Once accepted, a record is written even if the caller's context ends
	// (for example the room shutting down), so queued events are not dropped.
	if !s.writer.enqueue(context.WithoutCancel(ctx), func(ctx context.Context, conn *sql.Conn) error {
		return insertRecords(ctx, conn, copyRecords)
	}) {
		return fmt.Errorf("append events: writer queue full")
	}
	return nil
}

// Read returns the append-only records for run in sequence order.
func (s *Store) Read(ctx context.Context, run domain.RunID) iter.Seq2[domain.LogRecord, error] {
	return func(yield func(domain.LogRecord, error) bool) {
		if s == nil || s.read == nil {
			yield(domain.LogRecord{}, fmt.Errorf("read events: closed store"))
			return
		}
		rows, err := s.read.QueryContext(ctx, `SELECT seq, at_ms, kind, event_json, note_json FROM events WHERE run_id = ? ORDER BY seq`, string(run))
		if err != nil {
			yield(domain.LogRecord{}, fmt.Errorf("query events: %w", err))
			return
		}
		defer rows.Close()
		for rows.Next() {
			var rec domain.LogRecord
			var at int64
			var kind string
			var eventJSON, noteJSON []byte
			if err := rows.Scan(&rec.Seq, &at, &kind, &eventJSON, &noteJSON); err != nil {
				yield(domain.LogRecord{}, fmt.Errorf("scan event: %w", err))
				return
			}
			rec.Run = run
			rec.At = time.Duration(at) * time.Millisecond
			rec.Kind = vocab.EventKind(kind)
			rec.Event = decodeEvent(rec.Kind, eventJSON)
			if len(noteJSON) != 0 {
				rec.Note = new(domain.LogNote)
				if err := json.Unmarshal(noteJSON, rec.Note); err != nil {
					yield(domain.LogRecord{}, fmt.Errorf("decode event note: %w", err))
					return
				}
			}
			if !yield(rec, nil) {
				return
			}
		}
		if err := rows.Err(); err != nil {
			yield(domain.LogRecord{}, fmt.Errorf("read events: %w", err))
		}
	}
}

func insertRecords(ctx context.Context, conn *sql.Conn, records []domain.LogRecord) error {
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin event append: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	for _, rec := range records {
		eventJSON, err := json.Marshal(rec.Event)
		if err != nil {
			return fmt.Errorf("encode event %d: %w", rec.Seq, err)
		}
		var noteJSON []byte
		if rec.Note != nil {
			noteJSON, err = json.Marshal(rec.Note)
			if err != nil {
				return fmt.Errorf("encode event note %d: %w", rec.Seq, err)
			}
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO events(run_id, seq, at_ms, kind, event_json, note_json) VALUES (?, ?, ?, ?, ?, ?)`, string(rec.Run), rec.Seq, rec.At.Milliseconds(), string(rec.Kind), eventJSON, noteJSON); err != nil {
			return fmt.Errorf("insert event %d: %w", rec.Seq, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit event append: %w", err)
	}
	return nil
}

func decodeEvent(kind vocab.EventKind, data []byte) domain.Event {
	var event domain.Event
	switch kind {
	case vocab.EventJoin:
		event = &domain.Join{}
	case vocab.EventAct:
		event = &domain.Act{}
	case vocab.EventSay:
		event = &domain.Say{}
	case vocab.EventReport:
		event = &domain.Report{}
	case vocab.EventTimerFired:
		event = &domain.TimerFired{}
	case vocab.EventTranscribed:
		event = &domain.Transcribed{}
	case vocab.EventInterpreted:
		event = &domain.Interpreted{}
	case vocab.EventPCLocked:
		event = &domain.PCLocked{}
	case "debug_checkpoint":
		event = &domain.DebugCheckpoint{}
	default:
		return nil
	}
	if err := json.Unmarshal(data, event); err != nil {
		return nil
	}
	return event
}
