package logx

import (
	"context"
	"log/slog"
	"sync"
)

// NewRingHandler returns a handler retaining the latest capacity Warn and
// Error records. A non-positive capacity uses the default of 50.
func NewRingHandler(next slog.Handler, capacity int) *RingHandler {
	if capacity <= 0 {
		capacity = 50
	}
	return &RingHandler{next: next, tail: &ringTail{capacity: capacity}}
}

// RingHandler forwards records and retains a bounded severity tail.
type RingHandler struct {
	next slog.Handler
	tail *ringTail
}

type ringTail struct {
	mu       sync.Mutex
	capacity int
	records  []slog.Record
}

// Enabled reports whether next accepts level.
func (h *RingHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.next.Enabled(ctx, level)
}

// Handle forwards and retains Warn and Error records.
func (h *RingHandler) Handle(ctx context.Context, record slog.Record) error {
	if record.Level >= slog.LevelWarn {
		h.tail.mu.Lock()
		h.tail.records = append(h.tail.records, record.Clone())
		if len(h.tail.records) > h.tail.capacity {
			h.tail.records = h.tail.records[len(h.tail.records)-h.tail.capacity:]
		}
		h.tail.mu.Unlock()
	}
	return h.next.Handle(ctx, record)
}

// WithAttrs returns a ring handler with attrs attached by the downstream
// handler.
func (h *RingHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &RingHandler{next: h.next.WithAttrs(attrs), tail: h.tail}
}

// WithGroup returns a ring handler scoped to a slog group.
func (h *RingHandler) WithGroup(name string) slog.Handler {
	return &RingHandler{next: h.next.WithGroup(name), tail: h.tail}
}

// Records returns the retained Warn and Error records, oldest first.
func (h *RingHandler) Records() []slog.Record {
	h.tail.mu.Lock()
	defer h.tail.mu.Unlock()
	return append([]slog.Record(nil), h.tail.records...)
}
