package modelchain

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"time"

	"github.com/monstercameron/DungeonFlux/internal/ports"
)

// Config controls link racing and deadline behavior.
type Config struct {
	HedgeDelay         time.Duration
	FirstTokenDeadline time.Duration
	Deadline           time.Duration
}

// Chain calls links in parallel or as a hedge and returns the first usable
// result. Links are ordered: link zero starts immediately and later links are
// started after HedgeDelay.
type Chain struct {
	links  []ports.LLM
	config Config
}

// New constructs a chain. It copies links so callers may safely reuse their
// input slice.
func New(links []ports.LLM, config Config) *Chain {
	return &Chain{links: append([]ports.LLM(nil), links...), config: config}
}

// JSON races the configured links and returns the first successful response.
func (c *Chain) JSON(ctx context.Context, req ports.TextRequest, schema ports.Schema) (json.RawMessage, error) {
	if len(c.links) == 0 {
		return nil, errors.New("modelchain: no links")
	}
	callCtx, cancel := withDeadline(ctx, c.config.Deadline)
	defer cancel()
	type result struct {
		value json.RawMessage
		err   error
	}
	results := make(chan result, len(c.links))
	var wg sync.WaitGroup
	for i, link := range c.links {
		wg.Add(1)
		go func(index int, llm ports.LLM) {
			defer wg.Done()
			if index > 0 && !waitDelay(callCtx, c.config.HedgeDelay) {
				results <- result{err: callCtx.Err()}
				return
			}
			value, err := llm.JSON(callCtx, req, schema)
			select {
			case results <- result{value: value, err: err}:
			case <-callCtx.Done():
			}
		}(i, link)
	}
	go func() {
		wg.Wait()
		close(results)
	}()
	var last error
	for result := range results {
		if result.err == nil {
			cancel()
			return result.value, nil
		}
		last = result.err
	}
	if last == nil {
		last = callCtx.Err()
	}
	return nil, last
}

// StreamText starts a hedged stream. The first non-empty token wins and all
// losing streams are canceled before the token is returned.
func (c *Chain) StreamText(ctx context.Context, req ports.TextRequest) (ports.TextStream, error) {
	if len(c.links) == 0 {
		return nil, errors.New("modelchain: no links")
	}
	callCtx, cancel := withDeadline(ctx, c.config.Deadline)
	streams := &raceStream{ctx: callCtx, cancel: cancel, links: c.links, req: req, delay: c.config.HedgeDelay, first: c.config.FirstTokenDeadline}
	if err := streams.start(); err != nil {
		cancel()
		return nil, err
	}
	return streams, nil
}

func withDeadline(ctx context.Context, deadline time.Duration) (context.Context, context.CancelFunc) {
	if deadline <= 0 {
		return context.WithCancel(ctx)
	}
	return context.WithTimeout(ctx, deadline)
}

func waitDelay(ctx context.Context, delay time.Duration) bool {
	if delay <= 0 {
		return true
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-timer.C:
		return true
	case <-ctx.Done():
		return false
	}
}

// Deadline wraps an LLM with a whole-call context deadline.
func Deadline(next ports.LLM, duration time.Duration) ports.LLM {
	return deadlineLLM{next: next, duration: duration}
}

type deadlineLLM struct {
	next     ports.LLM
	duration time.Duration
}

func (d deadlineLLM) JSON(ctx context.Context, req ports.TextRequest, schema ports.Schema) (json.RawMessage, error) {
	callCtx, cancel := withDeadline(ctx, d.duration)
	defer cancel()
	return d.next.JSON(callCtx, req, schema)
}

func (d deadlineLLM) StreamText(ctx context.Context, req ports.TextRequest) (ports.TextStream, error) {
	callCtx, cancel := withDeadline(ctx, d.duration)
	stream, err := d.next.StreamText(callCtx, req)
	if err != nil {
		cancel()
		return nil, err
	}
	return &ownedStream{TextStream: stream, cancel: cancel}, nil
}

type ownedStream struct {
	ports.TextStream
	cancel context.CancelFunc
}

func (s *ownedStream) Close() error {
	s.cancel()
	return s.TextStream.Close()
}

var _ ports.TextStream = (*ownedStream)(nil)
