package modelchain

import (
	"context"
	"errors"
	"io"
	"sync"

	"github.com/monstercameron/DungeonFlux/internal/ports"
)

// storedStream publishes only a complete, uncancelled response. One consumer
// receives text while Close may interrupt it from the scope's cancellation path.
type storedStream struct {
	ctx      context.Context
	cancel   context.CancelFunc
	source   ports.TextStream
	put      func(context.Context, []byte) error
	mu       sync.Mutex
	text     []byte
	done     bool
	close    sync.Once
	closeErr error
}

func newStoredStream(ctx context.Context, source ports.TextStream, put func(context.Context, []byte) error) *storedStream {
	ctx, cancel := context.WithCancel(ctx)
	return &storedStream{ctx: ctx, cancel: cancel, source: source, put: put}
}

func (s *storedStream) Recv() (string, error) {
	s.mu.Lock()
	done := s.done
	s.mu.Unlock()
	if done {
		return "", io.EOF
	}
	if err := s.ctx.Err(); err != nil {
		return "", err
	}
	text, err := s.source.Recv()
	s.mu.Lock()
	defer s.mu.Unlock()
	if canceled := s.ctx.Err(); canceled != nil {
		s.done = true
		return "", canceled
	}
	if s.done {
		return "", io.EOF
	}
	s.text = append(s.text, text...)
	if err == nil {
		return text, nil
	}
	s.done = true
	defer s.cancel()
	if errors.Is(err, io.EOF) {
		if putErr := s.put(s.ctx, append([]byte(nil), s.text...)); putErr != nil {
			return text, putErr
		}
		// Deliver terminal text as a normal chunk before EOF, matching the
		// provider streams consumed by the narration executors.
		if text != "" {
			return text, nil
		}
	}
	return text, err
}

func (s *storedStream) Close() error {
	// Cancel first: persistence or Recv may be blocked while holding the state
	// lock. The source owns the mechanism that interrupts its pending read.
	s.cancel()
	s.close.Do(func() { s.closeErr = s.source.Close() })
	s.mu.Lock()
	s.done = true
	s.mu.Unlock()
	return s.closeErr
}

func cancellationError(ctx context.Context, err error) error {
	if canceled := ctx.Err(); canceled != nil {
		return canceled
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	return nil
}

var _ ports.TextStream = (*storedStream)(nil)
