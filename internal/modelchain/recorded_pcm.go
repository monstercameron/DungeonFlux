package modelchain

import (
	"context"
	"errors"
	"io"
	"sync"

	"github.com/monstercameron/DungeonFlux/internal/ports"
)

const maxRecordedPCMBytes = 8 << 20
const maxRecordedPCMChunks = 4096

// recordedPCM leaves live playback streaming. Oversized audio continues to
// play but cannot replace a bounded rehearsal recording.
type recordedPCM struct {
	ctx             context.Context
	cancel          context.CancelFunc
	source          ports.PCMStream
	put             func(context.Context, []ports.PCMChunk) error
	mu              sync.Mutex
	chunks          []ports.PCMChunk
	size, rate      int
	done, oversized bool
	close           sync.Once
	closeErr        error
}

func newRecordedPCM(ctx context.Context, source ports.PCMStream, rate int, put func(context.Context, []ports.PCMChunk) error) *recordedPCM {
	ctx, cancel := context.WithCancel(ctx)
	return &recordedPCM{ctx: ctx, cancel: cancel, source: source, rate: rate, put: put}
}

func (s *recordedPCM) Recv() (ports.PCMChunk, error) {
	s.mu.Lock()
	done := s.done
	s.mu.Unlock()
	if done {
		return ports.PCMChunk{}, io.EOF
	}
	if err := s.ctx.Err(); err != nil {
		return ports.PCMChunk{}, err
	}
	chunk, err := s.source.Recv()
	s.mu.Lock()
	defer s.mu.Unlock()
	if canceled := s.ctx.Err(); canceled != nil {
		s.done = true
		return ports.PCMChunk{}, canceled
	}
	if s.done {
		return ports.PCMChunk{}, io.EOF
	}
	if err != nil && !errors.Is(err, io.EOF) {
		s.done = true
		s.cancel()
		return ports.PCMChunk{}, err
	}
	if len(chunk.S16LE) > 0 {
		if chunk.SampleRate > 0 {
			s.rate = chunk.SampleRate
		}
		chunk.SampleRate = s.rate
		if s.rate <= 0 || len(chunk.S16LE)%2 != 0 {
			s.done = true
			s.cancel()
			return ports.PCMChunk{}, errors.New("modelchain: invalid TTS PCM")
		}
		s.remember(chunk)
	}
	if err == nil {
		return chunk, nil
	}
	return s.finish(chunk)
}

// finish runs while mu is held so completion and Close cannot race a save.
func (s *recordedPCM) finish(chunk ports.PCMChunk) (ports.PCMChunk, error) {
	s.done = true
	defer s.cancel()
	if !s.oversized && len(s.chunks) > 0 {
		if putErr := s.put(s.ctx, s.chunks); putErr != nil {
			return ports.PCMChunk{}, putErr
		}
	}
	// Several providers return the final PCM alongside EOF. Deliver it first;
	// narration consumers treat EOF as completion before inspecting the chunk.
	if len(chunk.S16LE) > 0 {
		return chunk, nil
	}
	return ports.PCMChunk{}, io.EOF
}

func (s *recordedPCM) remember(chunk ports.PCMChunk) {
	if s.oversized {
		return
	}
	s.size += len(chunk.S16LE)
	if s.size > maxRecordedPCMBytes || len(s.chunks) >= maxRecordedPCMChunks {
		s.oversized = true
		s.chunks = nil
		return
	}
	chunk.S16LE = append([]byte(nil), chunk.S16LE...)
	s.chunks = append(s.chunks, chunk)
}

func (s *recordedPCM) Close() error {
	s.cancel()
	s.close.Do(func() { s.closeErr = s.source.Close() })
	s.mu.Lock()
	s.done = true
	s.chunks = nil
	s.mu.Unlock()
	return s.closeErr
}
