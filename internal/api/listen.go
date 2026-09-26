package api

import (
	"context"
	"sync"

	"github.com/monstercameron/DungeonFlux/internal/domain"
)

const listenLagLimitMS int64 = 2000

// ListenHub fans PCM frames out to DM listeners. A listener whose unread
// queue exceeds two seconds is removed so a stale tab cannot retain memory.
type ListenHub struct {
	mu          sync.Mutex
	nextID      uint64
	subscribers map[uint64]*ListenSubscription
	latest      *ListenSubscription
}

// ListenSubscription is one bounded listener queue owned by a client.
type ListenSubscription struct {
	mu      sync.Mutex
	frames  chan domain.AudioFrame
	done    chan struct{}
	dropped bool
	pending int64
	remove  func()
}

// NewListenHub creates an empty PCM fan-out hub.
func NewListenHub() *ListenHub {
	return &ListenHub{subscribers: make(map[uint64]*ListenSubscription)}
}

// Subscribe adds a listener. Cancellation of ctx removes the listener on its
// next hub operation; no goroutine is created for the subscription.
func (h *ListenHub) Subscribe(ctx context.Context) *ListenSubscription {
	if ctx == nil {
		ctx = context.Background()
	}
	h.mu.Lock()
	h.nextID++
	id := h.nextID
	sub := &ListenSubscription{frames: make(chan domain.AudioFrame, 64), done: make(chan struct{})}
	sub.remove = func() { h.remove(id, sub) }
	previous := h.latest
	h.latest = sub
	h.subscribers[id] = sub
	h.mu.Unlock()
	if previous != nil {
		previous.remove()
	}
	if ctx.Err() != nil {
		sub.Close()
	}
	return sub
}

// Frame publishes one PCM frame, copying its bytes for each subscriber.
func (h *ListenHub) Frame(frame domain.AudioFrame) {
	duration := frameDurationMS(frame)
	h.mu.Lock()
	deferred := make([]*ListenSubscription, 0)
	for id, sub := range h.subscribers {
		if sub.closed() || sub.queue(frame, duration) {
			deferred = append(deferred, sub)
			delete(h.subscribers, id)
		}
	}
	h.mu.Unlock()
	for _, sub := range deferred {
		sub.finish()
	}
}

// Cancel removes queued frames for one utterance from every listener.
func (h *ListenHub) Cancel(utteranceID domain.UtteranceID) {
	h.mu.Lock()
	for _, sub := range h.subscribers {
		sub.removeUtterance(utteranceID)
	}
	h.mu.Unlock()
}

func (h *ListenHub) remove(id uint64, sub *ListenSubscription) {
	h.mu.Lock()
	if current, ok := h.subscribers[id]; ok && current == sub {
		delete(h.subscribers, id)
		if h.latest == sub {
			h.latest = nil
		}
	}
	h.mu.Unlock()
	sub.finish()
}

// Frames returns the subscription's receive-only PCM queue.
func (s *ListenSubscription) Frames() <-chan domain.AudioFrame { return s.frames }

// Done returns a channel closed when the subscription is no longer active.
func (s *ListenSubscription) Done() <-chan struct{} { return s.done }

// Close removes the subscription and releases its queue.
func (s *ListenSubscription) Close() {
	if s == nil || s.remove == nil {
		return
	}
	s.remove()
}

func (s *ListenSubscription) queue(frame domain.AudioFrame, duration int64) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.dropped || s.pending+duration > listenLagLimitMS {
		return true
	}
	copyFrame := frame
	copyFrame.PCMS16LE = append([]byte(nil), frame.PCMS16LE...)
	select {
	case s.frames <- copyFrame:
		s.pending += duration
		return false
	default:
		return true
	}
}

func (s *ListenSubscription) removeUtterance(id domain.UtteranceID) {
	s.mu.Lock()
	defer s.mu.Unlock()
	kept := make([]domain.AudioFrame, 0, len(s.frames))
	for {
		select {
		case frame := <-s.frames:
			if frame.UtteranceID != id {
				kept = append(kept, frame)
			}
		default:
			for _, frame := range kept {
				s.frames <- frame
			}
			s.pending = 0
			for _, frame := range kept {
				s.pending += frameDurationMS(frame)
			}
			return
		}
	}
}

func (s *ListenSubscription) closed() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.dropped
}

func (s *ListenSubscription) finish() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.dropped {
		return
	}
	s.dropped = true
	close(s.done)
	for {
		select {
		case <-s.frames:
		default:
			s.pending = 0
			close(s.frames)
			return
		}
	}
}

func frameDurationMS(frame domain.AudioFrame) int64 {
	if frame.SampleRate <= 0 {
		return listenLagLimitMS
	}
	return int64(len(frame.PCMS16LE)) * 1000 / int64(frame.SampleRate*2)
}
