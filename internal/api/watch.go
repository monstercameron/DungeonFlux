package api

import (
	"context"
	"sync"

	df "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
	"github.com/monstercameron/DungeonFlux/internal/domain"
)

// WatchHub distributes complete projected snapshots to connected clients.
// Each subscriber has one sender goroutine and a one-item latest-value queue.
type WatchHub struct {
	mu      sync.Mutex
	next    uint64
	subs    map[uint64]*watchSubscriber
	kinds   map[domain.SeatID]df.ClientKind
	locales map[domain.SeatID]string
}

type watchSubscriber struct {
	hub    *WatchHub
	id     uint64
	kind   df.ClientKind
	seat   domain.SeatID
	queue  chan domain.View
	output chan *df.WatchMessage
	done   chan struct{}
	closed bool
}

// NewWatchHub creates an empty snapshot hub.
func NewWatchHub() *WatchHub {
	return &WatchHub{subs: make(map[uint64]*watchSubscriber), kinds: make(map[domain.SeatID]df.ClientKind), locales: make(map[domain.SeatID]string)}
}

// RememberLocale records the settled locale for a seat. Views published
// afterwards are projected in that locale for the seat's subscribers.
func (h *WatchHub) RememberLocale(seat domain.SeatID, locale string) {
	tag := locale
	if tag == "" {
		tag = "en"
	}
	h.mu.Lock()
	if h.locales == nil {
		h.locales = make(map[domain.SeatID]string)
	}
	h.locales[seat] = tag
	h.mu.Unlock()
}

func (h *WatchHub) localeFor(seat domain.SeatID) string {
	h.mu.Lock()
	defer h.mu.Unlock()
	if tag, ok := h.locales[seat]; ok && tag != "" {
		return tag
	}
	return "en"
}

// RememberKind records the authenticated kind associated with a seat. A
// reconnect may omit the kind; Subscribe then reuses this remembered value.
func (h *WatchHub) RememberKind(seat domain.SeatID, kind df.ClientKind) {
	if kind == df.ClientKind_CLIENT_KIND_UNSPECIFIED {
		return
	}
	h.mu.Lock()
	h.kinds[seat] = kind
	h.mu.Unlock()
}

// Subscribe registers a client and starts its owned snapshot sender.
func (h *WatchHub) Subscribe(ctx context.Context, kind df.ClientKind, seat domain.SeatID) *WatchSubscription {
	if ctx == nil {
		ctx = context.Background()
	}
	h.mu.Lock()
	if kind == df.ClientKind_CLIENT_KIND_UNSPECIFIED {
		kind = h.kinds[seat]
	}
	h.next++
	sub := &watchSubscriber{hub: h, id: h.next, kind: kind, seat: seat, queue: make(chan domain.View, 1), output: make(chan *df.WatchMessage, 1), done: make(chan struct{})}
	h.subs[sub.id] = sub
	h.mu.Unlock()
	go sub.send(ctx)
	return &WatchSubscription{sub: sub}
}

// Publish offers a full view to every subscriber. If a queue is full, its old
// snapshot is discarded before the new snapshot is stored.
func (h *WatchHub) Publish(view domain.View) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, sub := range h.subs {
		if sub.closed {
			continue
		}
		select {
		case sub.queue <- view.DeepCopy():
		default:
			select {
			case <-sub.queue:
			default:
			}
			sub.queue <- view.DeepCopy()
		}
	}
}

func (h *WatchHub) remove(sub *watchSubscriber) {
	h.mu.Lock()
	if current, ok := h.subs[sub.id]; ok && current == sub {
		delete(h.subs, sub.id)
	}
	if !sub.closed {
		sub.closed = true
		close(sub.done)
		close(sub.queue)
	}
	h.mu.Unlock()
}

func (s *watchSubscriber) send(ctx context.Context) {
	defer close(s.output)
	for {
		select {
		case <-ctx.Done():
			s.hub.remove(s)
			return
		case <-s.done:
			return
		case view, ok := <-s.queue:
			if !ok {
				return
			}
			message := &df.WatchMessage{Message: &df.WatchMessage_State{State: ProjectLocalized(view, s.kind, s.seat, s.hub.localeFor(s.seat))}}
			select {
			case s.output <- message:
			default:
				select {
				case <-s.output:
				default:
				}
				s.output <- message
			}
		}
	}
}

// WatchSubscription is a client-owned receive queue.
type WatchSubscription struct{ sub *watchSubscriber }

// Messages returns snapshots in arrival order, with stale snapshots dropped.
func (s *WatchSubscription) Messages() <-chan *df.WatchMessage { return s.sub.output }

// Close removes the subscription and closes its output queue.
func (s *WatchSubscription) Close() {
	if s == nil || s.sub == nil {
		return
	}
	s.sub.hub.remove(s.sub)
}
