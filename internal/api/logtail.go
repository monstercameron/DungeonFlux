package api

import (
	"bytes"
	"context"
	"log/slog"
	"sync"

	df "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
	"github.com/monstercameron/DungeonFlux/internal/domain"
)

const defaultLogTailSize = 50

// LogTailHandler keeps the most recent warning and error log records for the
// host debug view. It is safe for concurrent use by slog callers and readers.
type LogTailHandler struct {
	state *logTailState
	attrs []slog.Attr
	group string
}

type logTailState struct {
	mu       sync.Mutex
	capacity int
	records  []slog.Record
}

// NewLogTailHandler creates a handler retaining the last 50 warning and error
// records. An optional capacity is provided for bounded tests and callers that
// need a smaller display; non-positive values use the default of 50.
func NewLogTailHandler(capacity ...int) *LogTailHandler {
	size := defaultLogTailSize
	if len(capacity) > 0 && capacity[0] > 0 {
		size = capacity[0]
	}
	return &LogTailHandler{state: &logTailState{capacity: size}}
}

// Enabled accepts warning and error records only.
func (h *LogTailHandler) Enabled(_ context.Context, level slog.Level) bool {
	return h != nil && level >= slog.LevelWarn
}

// Handle appends a cloned record and drops the oldest record when full.
func (h *LogTailHandler) Handle(_ context.Context, record slog.Record) error {
	if h == nil || record.Level < slog.LevelWarn {
		return nil
	}
	for _, attr := range h.attrs {
		if h.group != "" {
			attr = slog.Group(h.group, attr)
		}
		record.AddAttrs(attr)
	}
	h.state.mu.Lock()
	defer h.state.mu.Unlock()
	h.state.records = append(h.state.records, record.Clone())
	if len(h.state.records) > h.state.capacity {
		h.state.records = append([]slog.Record(nil), h.state.records[len(h.state.records)-h.state.capacity:]...)
	}
	return nil
}

// WithAttrs returns a handler that attaches attrs to records it receives.
func (h *LogTailHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if h == nil {
		return NewLogTailHandler()
	}
	return &LogTailHandler{state: h.state,
		attrs: append(append([]slog.Attr(nil), h.attrs...), attrs...), group: h.group}
}

// WithGroup returns a handler retaining the group's name for slog's handler
// contract. Grouped attributes are rendered by the standard text formatter.
func (h *LogTailHandler) WithGroup(name string) slog.Handler {
	if h == nil {
		return NewLogTailHandler()
	}
	return &LogTailHandler{state: h.state,
		attrs: append([]slog.Attr(nil), h.attrs...), group: name}
}

// Lines returns a snapshot of the retained records formatted as slog text.
func (h *LogTailHandler) Lines() []string {
	if h == nil {
		return nil
	}
	h.state.mu.Lock()
	defer h.state.mu.Unlock()
	lines := make([]string, 0, len(h.state.records))
	for _, record := range h.state.records {
		lines = append(lines, formatLogRecord(record))
	}
	return lines
}

// LogTail is an alias for Lines, named after the HostView field it supplies.
func (h *LogTailHandler) LogTail() []string { return h.Lines() }

// ProjectHostWithLogTail projects a host view and attaches the handler's
// retained warning and error records to HostView.log_tail.
func ProjectHostWithLogTail(view domain.View, handler *LogTailHandler) *df.HostView {
	out := projectHost(view)
	if handler != nil {
		out.LogTail = handler.Lines()
	}
	return out
}

func formatLogRecord(record slog.Record) string {
	var buffer bytes.Buffer
	if err := slog.NewTextHandler(&buffer, nil).Handle(context.Background(), record); err != nil {
		return record.Message
	}
	return buffer.String()
}
