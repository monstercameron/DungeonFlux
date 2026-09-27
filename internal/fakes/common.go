package fakes

import (
	"context"
	"io"
	"sync"
	"time"

	"github.com/monstercameron/DungeonFlux/internal/clock"
	"github.com/monstercameron/DungeonFlux/internal/ports"
)

var (
	_ ports.LLM         = (*FakeLLM)(nil)
	_ ports.ImageGen    = (*FakeImageGen)(nil)
	_ ports.VideoGen    = (*FakeVideoGen)(nil)
	_ ports.TTS         = (*FakeTTS)(nil)
	_ ports.STT         = (*FakeSTT)(nil)
	_ ports.SoundGen    = (*FakeSoundGen)(nil)
	_ ports.Inbox       = (*FakeInbox)(nil)
	_ ports.AudioOut    = (*FakeAudioOut)(nil)
	_ ports.AssetWriter = (*FakeAssetWriter)(nil)
	_ ports.EventLog    = (*FakeEventLog)(nil)
	_ ports.Runs        = (*FakeRuns)(nil)
	_ ports.Assets      = (*FakeAssets)(nil)
	_ ports.Cache       = (*FakeCache)(nil)
	_ ports.Recordings  = (*FakeRecordings)(nil)
	_ ports.Engine      = (*FakeEngine)(nil)
)

type outcome[T any] struct {
	Value T
	Err   error
	Delay time.Duration
}

func wait(ctx context.Context, c clock.Clock, d time.Duration) error {
	if d <= 0 || c == nil {
		return nil
	}
	t := c.NewTimer(d)
	defer t.Stop()
	select {
	case <-t.C():
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

type textStream struct {
	mu     sync.Mutex
	chunks []string
	index  int
	err    error
	closed bool
}

func (s *textStream) Recv() (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return "", io.ErrClosedPipe
	}
	if s.index == len(s.chunks) {
		return "", s.err
	}
	v := s.chunks[s.index]
	s.index++
	return v, nil
}

func (s *textStream) Close() error {
	s.mu.Lock()
	s.closed = true
	s.mu.Unlock()
	return nil
}
