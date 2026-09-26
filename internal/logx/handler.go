package logx

import (
	"context"
	"log/slog"
	"os"
	"strings"
	"sync"
)

// Redact reports whether an attribute key must not be emitted.
func Redact(key string) bool {
	key = strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(key, "-", "_"), ".", "_"))
	for _, word := range strings.FieldsFunc(key, func(r rune) bool { return r == '_' || r == '/' || r == ':' }) {
		switch word {
		case "key", "token", "secret", "authorization", "password", "credential":
			return true
		}
	}
	return strings.Contains(key, "authorization") || strings.HasSuffix(key, "_token") ||
		strings.HasSuffix(key, "_secret") || strings.HasSuffix(key, "_key")
}

// NewRedactingHandler wraps next and removes sensitive attributes, including
// sensitive attributes nested inside slog groups.
func NewRedactingHandler(next slog.Handler) slog.Handler {
	return &redactingHandler{next: next}
}

type redactingHandler struct {
	next  slog.Handler
	attrs []slog.Attr
	group string
}

func (h *redactingHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.next.Enabled(ctx, level)
}

func (h *redactingHandler) Handle(ctx context.Context, record slog.Record) error {
	clean := slog.NewRecord(record.Time, record.Level, record.Message, record.PC)
	for _, attr := range h.attrs {
		clean.AddAttrs(filterAttr(h.group, attr))
	}
	record.Attrs(func(attr slog.Attr) bool { clean.AddAttrs(filterAttr(h.group, attr)); return true })
	return h.next.Handle(ctx, clean)
}

func (h *redactingHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	copyAttrs := append(append([]slog.Attr(nil), h.attrs...), attrs...)
	return &redactingHandler{next: h.next, attrs: copyAttrs, group: h.group}
}

func (h *redactingHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	group := name
	if h.group != "" {
		group = h.group + "." + name
	}
	return &redactingHandler{next: h.next, attrs: h.attrs, group: group}
}

func filterAttr(group string, attr slog.Attr) slog.Attr {
	if attr.Key != "" && Redact(attr.Key) {
		return slog.Attr{}
	}
	if attr.Value.Kind() != slog.KindGroup {
		return attr
	}
	children := attr.Value.Group()
	filtered := make([]slog.Attr, 0, len(children))
	for _, child := range children {
		if clean := filterAttr(group+"."+attr.Key, child); clean.Key != "" {
			filtered = append(filtered, clean)
		}
	}
	if len(filtered) == 0 {
		return slog.Attr{}
	}
	return slog.Group(attr.Key, attrsToAny(filtered)...)
}

func attrsToAny(attrs []slog.Attr) []any {
	values := make([]any, len(attrs))
	for i, attr := range attrs {
		values[i] = attr
	}
	return values
}

// NewJSONLHandler opens path and returns a redacting JSON-lines handler and
// the file that must be closed by the caller.
func NewJSONLHandler(path string, level slog.Leveler) (slog.Handler, *os.File, error) {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o600)
	if err != nil {
		return nil, nil, err
	}
	handler := slog.NewJSONHandler(file, &slog.HandlerOptions{Level: level})
	return NewRedactingHandler(handler), file, nil
}

// NewConsoleHandler returns a redacting text handler writing to stdout.
func NewConsoleHandler(level slog.Leveler) slog.Handler {
	return NewRedactingHandler(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level}))
}

// NewCapturingHandler returns a handler that stores cloned records for tests.
func NewCapturingHandler() *CapturingHandler { return &CapturingHandler{} }

// CapturingHandler stores records in memory for assertions.
type CapturingHandler struct {
	mu      sync.Mutex
	records []slog.Record
}

// Enabled accepts every slog level.
func (h *CapturingHandler) Enabled(_ context.Context, _ slog.Level) bool { return true }

// Handle stores a clone of record.
func (h *CapturingHandler) Handle(_ context.Context, record slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.records = append(h.records, record.Clone())
	return nil
}

// WithAttrs returns a handler with attrs attached to future records.
func (h *CapturingHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &capturingWithAttrs{parent: h, attrs: append([]slog.Attr(nil), attrs...)}
}

// WithGroup returns a capturing handler. Group names are represented by slog
// itself when records are handled, so the base handler remains reusable.
func (h *CapturingHandler) WithGroup(name string) slog.Handler {
	return &capturingWithAttrs{parent: h, group: name}
}

// Records returns a snapshot of captured records.
func (h *CapturingHandler) Records() []slog.Record {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]slog.Record(nil), h.records...)
}

type capturingWithAttrs struct {
	parent *CapturingHandler
	attrs  []slog.Attr
	group  string
}

func (h *capturingWithAttrs) Enabled(ctx context.Context, level slog.Level) bool {
	return h.parent.Enabled(ctx, level)
}
func (h *capturingWithAttrs) Handle(ctx context.Context, record slog.Record) error {
	for _, attr := range h.attrs {
		record.AddAttrs(attr)
	}
	return h.parent.Handle(ctx, record)
}
func (h *capturingWithAttrs) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &capturingWithAttrs{parent: h.parent, attrs: append(append([]slog.Attr(nil), h.attrs...), attrs...), group: h.group}
}
func (h *capturingWithAttrs) WithGroup(name string) slog.Handler {
	return &capturingWithAttrs{parent: h.parent, attrs: h.attrs, group: name}
}
