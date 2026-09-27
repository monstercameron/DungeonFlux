package runtime

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/ports"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

func TestRunner_recoveredPanicLogsScopeAndPostsFailure(t *testing.T) {
	var logs bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logs, nil))
	inbox := NewInbox(1)
	runner := NewRunner(inbox, logger)
	Handle[domain.Transcribe](runner, func(context.Context, domain.Transcribe, domain.Scope, ports.Inbox) {
		panic("adapter failed")
	})

	runner.Run([]domain.Effect{domain.Transcribe{UtteranceID: "u-1"}})
	ctx := context.Background()
	env, ok := inbox.Receive(ctx)
	if !ok {
		t.Fatal("panic did not produce a failure event")
	}
	failed, ok := env.Event.(domain.STTError)
	if !ok || failed.UtteranceID != "u-1" || failed.FailureKind != vocab.ErrUnavailable {
		t.Fatalf("failure event = %#v", env.Event)
	}
	for _, want := range []string{"recovered work effect panic", "goroutine_role=work", "effect=transcribe", "stack="} {
		if !strings.Contains(logs.String(), want) {
			t.Fatalf("log does not contain %q: %s", want, logs.String())
		}
	}
}

func TestRunRecovered_postsFailureWithScope(t *testing.T) {
	inbox := NewInbox(1)
	scope := domain.Scope{Machine: vocab.MachineSession, Epoch: 7, Key: "utterance/u-2"}
	effect := domain.Interpret{UtteranceID: "u-2"}
	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	runRecovered(context.Background(), logger, effect, scope, inbox, func(context.Context, domain.Effect, domain.Scope, ports.Inbox) {
		panic("boom")
	})
	env, ok := inbox.Receive(context.Background())
	if !ok {
		t.Fatal("panic did not post failure")
	}
	if env.Scope != scope {
		t.Fatalf("scope = %#v, want %#v", env.Scope, scope)
	}
	if _, ok := env.Event.(domain.InterpretFailed); !ok {
		t.Fatalf("event = %#v", env.Event)
	}
}

func TestRunRecovered_doesNotPostUnsupportedFailure(t *testing.T) {
	inbox := NewInbox(1)
	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	runRecovered(context.Background(), logger, domain.ReleaseLine{}, domain.Scope{}, inbox, func(context.Context, domain.Effect, domain.Scope, ports.Inbox) {
		panic("boom")
	})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, ok := inbox.Receive(ctx); ok {
		t.Fatal("unsupported effect produced a failure event")
	}
}
