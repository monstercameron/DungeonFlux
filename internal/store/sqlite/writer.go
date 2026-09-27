package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"path/filepath"
	"strings"
	"sync"
)

const writerQueueSize = 1024

type writeRequest struct {
	ctx  context.Context
	work func(context.Context, *sql.Conn) error
	ack  chan error
}

type writeLoop struct {
	// mu guards queue sends against close: a write submitted while the store
	// shuts down used to race close(queue) and could panic with "send on
	// closed channel". Senders hold the read lock; close takes the write lock.
	mu     sync.RWMutex
	closed bool
	queue  chan writeRequest
	done   chan struct{}
	logger *slog.Logger
}

// errWriterClosed is returned for writes that arrive after Close.
var errWriterClosed = errors.New("sqlite writer is closed")

// Store owns a SQLite write connection and a separate read pool.
type Store struct {
	read   *sql.DB
	writer *writeLoop
}

// Open opens a SQLite store at path and applies all known migrations.
func Open(ctx context.Context, path string, logger *slog.Logger) (*Store, error) {
	if strings.TrimSpace(path) == "" {
		return nil, fmt.Errorf("open sqlite store: empty path")
	}
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	if err := ensureParent(path); err != nil {
		return nil, err
	}
	dsn := sqliteDSN(path)
	read, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite read pool: %w", err)
	}
	read.SetMaxOpenConns(8)
	read.SetMaxIdleConns(8)
	if err := ApplyMigrations(ctx, read); err != nil {
		_ = read.Close()
		return nil, err
	}
	conn, err := read.Conn(ctx)
	if err != nil {
		_ = read.Close()
		return nil, fmt.Errorf("reserve sqlite writer connection: %w", err)
	}
	w := &writeLoop{queue: make(chan writeRequest, writerQueueSize), done: make(chan struct{}), logger: logger}
	go w.run(conn)
	return &Store{read: read, writer: w}, nil
}

// Close stops the writer and closes the read pool.
func (s *Store) Close() error {
	if s == nil {
		return nil
	}
	if s.writer != nil {
		s.writer.close()
	}
	if s.read != nil {
		return s.read.Close()
	}
	return nil
}

func (w *writeLoop) run(conn *sql.Conn) {
	defer close(w.done)
	defer conn.Close()
	for req := range w.queue {
		if req.ctx == nil {
			req.ctx = context.Background()
		}
		err := req.work(req.ctx, conn)
		if req.ack != nil {
			req.ack <- err
		} else if err != nil {
			// Enqueued writes have no caller waiting for the result; log the
			// failure so a rejected write is never silent.
			w.logger.Error("sqlite write failed", "err", err)
		}
	}
}

func (w *writeLoop) close() {
	w.mu.Lock()
	if w.closed {
		w.mu.Unlock()
		<-w.done
		return
	}
	w.closed = true
	close(w.queue)
	w.mu.Unlock()
	<-w.done
}

func (w *writeLoop) submit(ctx context.Context, work func(context.Context, *sql.Conn) error) error {
	if ctx == nil {
		ctx = context.Background()
	}
	ack := make(chan error, 1)
	req := writeRequest{ctx: ctx, work: work, ack: ack}
	w.mu.RLock()
	if w.closed {
		w.mu.RUnlock()
		return errWriterClosed
	}
	select {
	case w.queue <- req:
		w.mu.RUnlock()
	case <-ctx.Done():
		w.mu.RUnlock()
		return ctx.Err()
	}
	select {
	case err := <-ack:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (w *writeLoop) enqueue(ctx context.Context, work func(context.Context, *sql.Conn) error) bool {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return false
	}
	w.mu.RLock()
	defer w.mu.RUnlock()
	if w.closed {
		return false
	}
	select {
	case w.queue <- writeRequest{ctx: ctx, work: work}:
		return true
	default:
		w.logger.Error("sqlite writer queue full", "capacity", cap(w.queue))
		return false
	}
}

func sqliteDSN(path string) string {
	if strings.HasPrefix(path, "file:") {
		return path + "&_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)&_pragma=foreign_keys(1)&_txlock=immediate"
	}
	return "file:" + filepath.ToSlash(path) + "?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)&_pragma=foreign_keys(1)&_txlock=immediate"
}

func ensureParent(path string) error {
	if strings.HasPrefix(path, "file:") {
		return nil
	}
	parent := filepath.Dir(path)
	if parent == "." || parent == "" {
		return nil
	}
	return nil
}
