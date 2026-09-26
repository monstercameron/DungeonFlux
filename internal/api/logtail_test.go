package api

import (
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/monstercameron/DungeonFlux/internal/domain"
)

func TestLogTailHandler_RetainsWarningsAndErrorsInOrder(t *testing.T) {
	handler := NewLogTailHandler(2)
	logger := slog.New(handler)
	logger.Info("ignored")
	logger.Warn("first", "scope", "room")
	logger.Error("second")
	logger.Warn("third")

	lines := handler.Lines()
	if len(lines) != 2 || !strings.Contains(lines[0], "second") || !strings.Contains(lines[1], "third") {
		t.Fatalf("tail = %#v", lines)
	}
}

func TestLogTailHandler_WithAttrsAndNilProjection(t *testing.T) {
	handler := NewLogTailHandler()
	logger := slog.New(handler).With("run", "r1")
	logger.Warn("fallback")
	if lines := handler.LogTail(); len(lines) != 1 || !strings.Contains(lines[0], "run=r1") {
		t.Fatalf("tail with attrs = %#v", lines)
	}
	if got := ProjectHostWithLogTail(domain.View{}, handler); len(got.LogTail) != 1 {
		t.Fatalf("projected tail = %#v", got.LogTail)
	}
	if got := ProjectHostWithLogTail(domain.View{}, nil); got.LogTail != nil {
		t.Fatalf("nil handler tail = %#v", got.LogTail)
	}
	if !handler.Enabled(context.Background(), slog.LevelError) || handler.Enabled(context.Background(), slog.LevelInfo) {
		t.Fatal("unexpected level filtering")
	}
}

func TestLogTailHandler_WithGroupAndNilReceiver(t *testing.T) {
	handler := NewLogTailHandler(1)
	grouped := slog.New(handler).WithGroup("scope").With("id", "x")
	grouped.Warn("grouped")
	if lines := handler.Lines(); len(lines) != 1 || !strings.Contains(lines[0], "scope.id=x") {
		t.Fatalf("grouped tail = %#v", lines)
	}
	var nilHandler *LogTailHandler
	if nilHandler.Lines() != nil || nilHandler.Enabled(context.Background(), slog.LevelWarn) {
		t.Fatal("nil handler should be inert")
	}
	if err := nilHandler.Handle(context.Background(), slog.Record{Level: slog.LevelError, Message: "ignored"}); err != nil {
		t.Fatalf("nil handle error = %v", err)
	}
}
