package modelchain

import (
	"context"
	"sync"
	"time"

	"github.com/monstercameron/DungeonFlux/internal/ports"
)

type raceStream struct {
	ctx    context.Context
	cancel context.CancelFunc
	links  []ports.LLM
	req    ports.TextRequest
	delay  time.Duration
	first  time.Duration
	items  chan raceItem
	close  sync.Once
	mu     sync.Mutex
	winner int
	chosen bool
	active map[int]ports.TextStream
}

type raceItem struct {
	index int
	text  string
	err   error
	first bool
}

func (s *raceStream) start() error {
	s.items = make(chan raceItem, len(s.links)*2)
	s.active = make(map[int]ports.TextStream, len(s.links))
	for index, link := range s.links {
		go s.runLink(index, link)
	}
	return nil
}

func (s *raceStream) runLink(index int, link ports.LLM) {
	if index > 0 && !waitDelay(s.ctx, s.delay) {
		return
	}
	stream, err := link.StreamText(s.ctx, s.req)
	if err != nil {
		s.emit(raceItem{index: index, err: err})
		return
	}
	s.mu.Lock()
	s.active[index] = stream
	s.mu.Unlock()
	firstCtx, cancel := withDeadline(s.ctx, s.first)
	defer cancel()
	for {
		text, recvErr := recvWithContext(firstCtx, stream)
		if recvErr != nil {
			s.emit(raceItem{index: index, err: recvErr})
			return
		}
		if text == "" {
			continue
		}
		s.emit(raceItem{index: index, text: text, first: true})
		for {
			text, recvErr = stream.Recv()
			if recvErr != nil {
				s.emit(raceItem{index: index, err: recvErr})
				return
			}
			s.emit(raceItem{index: index, text: text})
		}
	}
}

func recvWithContext(ctx context.Context, stream ports.TextStream) (string, error) {
	result := make(chan raceItem, 1)
	go func() {
		text, err := stream.Recv()
		result <- raceItem{text: text, err: err}
	}()
	select {
	case item := <-result:
		return item.text, item.err
	case <-ctx.Done():
		_ = stream.Close()
		return "", ctx.Err()
	}
}

func (s *raceStream) emit(item raceItem) {
	select {
	case s.items <- item:
	case <-s.ctx.Done():
	}
}

func (s *raceStream) Recv() (string, error) {
	for {
		select {
		case <-s.ctx.Done():
			s.Close()
			return "", s.ctx.Err()
		case item := <-s.items:
			if item.first {
				s.selectWinner(item.index)
				return item.text, nil
			}
			if item.index != s.winner {
				continue
			}
			if item.err != nil {
				return "", item.err
			}
			return item.text, nil
		}
	}
}

func (s *raceStream) selectWinner(index int) {
	s.mu.Lock()
	if !s.chosen {
		s.winner = index
		s.chosen = true
	}
	for candidate, stream := range s.active {
		if candidate != s.winner {
			_ = stream.Close()
		}
	}
	s.mu.Unlock()
}

func (s *raceStream) Close() error {
	s.close.Do(func() {
		s.cancel()
		s.mu.Lock()
		defer s.mu.Unlock()
		for _, stream := range s.active {
			_ = stream.Close()
		}
	})
	return nil
}

var _ ports.TextStream = (*raceStream)(nil)
