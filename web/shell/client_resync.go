package main

import (
	"context"
	"sync/atomic"
	"time"
)

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
func (c *Client) Resync() {
	if c == nil {
		return
	}
	c.resyncMu.Lock()
	defer c.resyncMu.Unlock()
	if c.resyncC != nil {
		close(c.resyncC)
	}
	c.resyncC = make(chan struct{})
}

func (c *Client) resyncSignal() <-chan struct{} {
	c.resyncMu.Lock()
	defer c.resyncMu.Unlock()
	if c.resyncC == nil {
		c.resyncC = make(chan struct{})
	}
	return c.resyncC
}

func (c *Client) watchIdle() time.Duration {
	if c.idleResync > 0 {
		return c.idleResync
	}
	return idleWatchResync
}

// superviseStream cancels one watch stream when a Resync arrives or when no
// message arrives within the idle window, and reports whether it did so, so
// the watch loop reconnects at once without surfacing an error. It returns
// when ctx ends.
func (c *Client) superviseStream(ctx context.Context, cancel context.CancelFunc, received <-chan struct{}) *atomic.Bool {
	kicked := &atomic.Bool{}
	resync := c.resyncSignal()
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
