package watch

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

// Controller owns resubscription signals for a client. The zero value is ready to use.
// Idle may be configured before starting any subscriptions.
type Controller struct {
	mu     sync.Mutex
	signal chan struct{}
	Idle   time.Duration
}

// idleWatchResync is how long a watch stream may stay silent before the
// client drops it and resubscribes. A phone that slept or changed networks can
// hold a WebSocket that looks open but no longer delivers; resubscribing makes
// the server replay its latest snapshot. A repeated snapshot is harmless:
// screens are keyed by the snapshot version, so the same version does not
// remount anything.
const idleWatchResync = 25 * time.Second

// Resync makes every active watch stream on this client drop its connection
// and resubscribe at once, so the server replays its latest snapshot. The
// browser calls it when the page becomes visible again or comes back online.
func (c *Controller) Resync() {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.signal != nil {
		close(c.signal)
	}
	c.signal = make(chan struct{})
}

// Signal returns the current resubscription notification.
func (c *Controller) Signal() <-chan struct{} {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.signal == nil {
		c.signal = make(chan struct{})
	}
	return c.signal
}

func (c *Controller) watchIdle() time.Duration {
	if c.Idle > 0 {
		return c.Idle
	}
	return idleWatchResync
}

// Supervise cancels one watch stream when a Resync arrives or when no
// message arrives within the idle window, and reports whether it did so, so
// the watch loop reconnects at once without surfacing an error. It returns
// when ctx ends.
func (c *Controller) Supervise(ctx context.Context, cancel context.CancelFunc, received <-chan struct{}) *atomic.Bool {
	kicked := &atomic.Bool{}
	resync := c.Signal()
	idle := c.watchIdle()
	go func() {
		timer := time.NewTimer(idle)
		defer timer.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-resync:
			case <-timer.C:
			case <-received:
				timer.Reset(idle)
				continue
			}
			kicked.Store(true)
			cancel()
			return
		}
	}()
	return kicked
}
