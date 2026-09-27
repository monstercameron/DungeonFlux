package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/ports"
)

// CacheStore implements ports.Cache on a shared Store.
type CacheStore struct{ store *Store }

// RecordingStore implements ports.Recordings on a shared Store.
type RecordingStore struct{ store *Store }

// NewCache returns a cache storage view over store.
func NewCache(store *Store) *CacheStore { return &CacheStore{store: store} }

// NewRecordings returns a recordings storage view over store.
func NewRecordings(store *Store) *RecordingStore { return &RecordingStore{store: store} }

// Get returns a cached adapter response by adapter and input hash.
func (s *CacheStore) Get(ctx context.Context, adapter, inputHash string) ([]byte, bool, error) {
	if s == nil || s.store == nil || s.store.read == nil {
		return nil, false, fmt.Errorf("get cache: closed store")
	}
	var value []byte
	err := s.store.read.QueryRowContext(ctx, `SELECT response FROM cache WHERE adapter = ? AND input_hash = ?`, adapter, inputHash).Scan(&value)
	if err == sql.ErrNoRows {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("get cache %q/%q: %w", adapter, inputHash, err)
	}
	return append([]byte(nil), value...), true, nil
}

// PutCache stores or replaces an adapter response.
func (s *CacheStore) Put(ctx context.Context, adapter, inputHash string, value []byte) error {
	if s == nil || s.store == nil || s.store.writer == nil {
		return fmt.Errorf("put cache: closed store")
	}
	copyValue := append([]byte(nil), value...)
	return s.store.writer.submit(ctx, func(ctx context.Context, conn *sql.Conn) error {
		_, err := conn.ExecContext(ctx, `INSERT INTO cache(adapter, input_hash, response) VALUES (?, ?, ?) ON CONFLICT(adapter, input_hash) DO UPDATE SET response=excluded.response`, adapter, inputHash, copyValue)
		if err != nil {
			return fmt.Errorf("put cache %q/%q: %w", adapter, inputHash, err)
		}
		return nil
	})
}

// GetRecording returns a recorded response by sequence key.
func (s *RecordingStore) Get(ctx context.Context, key ports.RecKey) (domain.Recording, bool, error) {
	if s == nil || s.store == nil || s.store.read == nil {
		return domain.Recording{}, false, fmt.Errorf("get recording: closed store")
	}
	var record domain.Recording
	err := s.store.read.QueryRowContext(ctx, `SELECT recording_id, text, audio, mime FROM recordings WHERE adapter = ? AND phase = ? AND seat = ? AND call_index = ?`, key.Adapter, string(key.Phase), key.Seat, key.Index).Scan(&record.ID, &record.Text, &record.Audio, &record.MIME)
	if err == sql.ErrNoRows {
		return domain.Recording{}, false, nil
	}
	if err != nil {
		return domain.Recording{}, false, fmt.Errorf("get recording: %w", err)
	}
	return record, true, nil
}

// Put stores or replaces a recorded response.
func (s *RecordingStore) Put(ctx context.Context, key ports.RecKey, record domain.Recording) error {
	if s == nil || s.store == nil || s.store.writer == nil {
		return fmt.Errorf("put recording: closed store")
	}
	return s.store.writer.submit(ctx, func(ctx context.Context, conn *sql.Conn) error {
		_, err := conn.ExecContext(ctx, `INSERT INTO recordings(adapter, phase, seat, call_index, recording_id, text, audio, mime) VALUES (?, ?, ?, ?, ?, ?, ?, ?) ON CONFLICT(adapter, phase, seat, call_index) DO UPDATE SET recording_id=excluded.recording_id, text=excluded.text, audio=excluded.audio, mime=excluded.mime`, key.Adapter, string(key.Phase), key.Seat, key.Index, string(record.ID), record.Text, record.Audio, record.MIME)
		if err != nil {
			return fmt.Errorf("put recording: %w", err)
		}
		return nil
	})
}

var _ ports.Cache = (*CacheStore)(nil)
var _ ports.Recordings = (*RecordingStore)(nil)
