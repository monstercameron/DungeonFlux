package runtime

import (
	"context"

	"github.com/monstercameron/DungeonFlux/internal/domain"
)

// Inbox is a bounded queue for envelopes entering a room. Posting blocks when
// the queue is full, allowing the caller's context to provide backpressure.
type Inbox struct {
	queue chan domain.Envelope
}

// NewInbox constructs an inbox. With no argument it uses the room's standard
// capacity of 256 envelopes.
func NewInbox(capacity ...int) *Inbox {
	size := roomInboxCapacity
	if len(capacity) > 0 && capacity[0] > 0 {
		size = capacity[0]
	}
	return &Inbox{queue: make(chan domain.Envelope, size)}
}

// Post enqueues env, returning false if ctx is done before it is accepted.
func (i *Inbox) Post(ctx context.Context, env domain.Envelope) bool {
	if ctx.Err() != nil {
		return false
	}
	select {
	case i.queue <- env:
		return true
	case <-ctx.Done():
		return false
	}
}

// Receive waits for an envelope or context cancellation. The boolean is false
// only when the context is done or the inbox has been closed.
func (i *Inbox) Receive(ctx context.Context) (domain.Envelope, bool) {
	select {
	case env, ok := <-i.queue:
		return env, ok
	case <-ctx.Done():
		return domain.Envelope{}, false
	}
}

// Close prevents future receives after queued envelopes have drained. It is
// intended for shutdown after all posters have stopped.
func (i *Inbox) Close() { close(i.queue) }
