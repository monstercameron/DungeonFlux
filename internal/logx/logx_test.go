package logx

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRedact_detectsSecretNames(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{"api_key", true}, {"access-token", true}, {"authorization", true},
		{"secret_value", true}, {"request_id", false}, {"monkey", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := Redact(tc.name); got != tc.want {
				t.Fatalf("Redact = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestRedactingHandler_dropsNestedSecrets(t *testing.T) {
	capture := NewCapturingHandler()
	logger := slog.New(NewRedactingHandler(capture))
	logger.Info("call", "api_key", "hidden", slog.Group("request", slog.String("token", "hidden"), slog.String("id", "ok")))
	records := capture.Records()
	if len(records) != 1 {
		t.Fatalf("records = %d", len(records))
	}
	var text string
	records[0].Attrs(func(attr slog.Attr) bool { text += attr.Key + "=" + attr.Value.String() + ";"; return true })
	if strings.Contains(text, "key") || strings.Contains(text, "token") {
		t.Fatalf("secret leaked: %s", text)
	}
	if !strings.Contains(text, "request") {
		t.Fatalf("group missing: %s", text)
	}
}

func TestRingHandler_keepsLatestWarnAndError(t *testing.T) {
	capture := NewCapturingHandler()
	ring := NewRingHandler(capture, 2)
	logger := slog.New(ring)
	logger.Info("ignored")
	logger.Warn("first")
	logger.Error("second")
	logger.Warn("third")
	records := ring.Records()
	if len(records) != 2 || records[0].Message != "second" || records[1].Message != "third" {
		t.Fatalf("tail = %#v", records)
	}
}

func TestCapturingHandler_WithAttrs(t *testing.T) {
	capture := NewCapturingHandler()
	slog.New(capture.WithAttrs([]slog.Attr{slog.String("scope", "test")})).Info("message")
	records := capture.Records()
	if len(records) != 1 {
		t.Fatalf("records = %d", len(records))
	}
	found := false
	records[0].Attrs(func(attr slog.Attr) bool {
		found = found || attr.Key == "scope" && attr.Value.String() == "test"
		return true
	})
	if !found {
		t.Fatal("scope attr missing")
	}
}

func TestHandlers_supportGroupsAndDerivedHandlers(t *testing.T) {
	capture := NewCapturingHandler()
	logger := slog.New(NewRedactingHandler(capture).WithGroup("scope").WithAttrs([]slog.Attr{slog.String("id", "ok")}))
	logger.Info("grouped")
	if len(capture.Records()) != 1 {
		t.Fatal("grouped record was not captured")
	}
	if got := NewConsoleHandler(slog.LevelError); got == nil {
		t.Fatal("console handler is nil")
	}
	ring := NewRingHandler(capture, 1)
	derived := slog.New(ring.WithAttrs([]slog.Attr{slog.String("source", "derived")})).Warn
	derived("derived")
	if len(ring.Records()) != 1 {
		t.Fatal("derived ring record missing")
	}
}

func TestJSONLHandler_writesFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "logs", "server.jsonl")
	if err := os.Mkdir(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	handler, file, err := NewJSONLHandler(path, slog.LevelInfo)
	if err != nil {
		t.Fatal(err)
	}
	slog.New(handler).Info("written", "token", "hidden")
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if !strings.Contains(text, "written") || strings.Contains(text, "hidden") {
		t.Fatalf("jsonl = %s", text)
	}
	if !handler.Enabled(context.Background(), slog.LevelInfo) {
		t.Fatal("handler unexpectedly disabled")
	}
}
