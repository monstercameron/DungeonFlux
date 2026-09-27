package main

import (
	"context"
	"sync/atomic"
)

// Resync requests a fresh snapshot on all active subscriptions.
func (c *Client) Resync() {
	if c != nil {
		c.resync.Resync()
	}
}
func (c *Client) resyncSignal() <-chan struct{} { return c.resync.Signal() }
func (c *Client) superviseStream(ctx context.Context, cancel context.CancelFunc, received <-chan struct{}) *atomic.Bool {
	return c.resync.Supervise(ctx, cancel, received)
}
