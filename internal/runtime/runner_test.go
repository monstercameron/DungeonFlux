package runtime

import (
	"context"
	"testing"
	"testing/synctest"
	"time"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/ports"
)

func TestRunner_Handle_runsRegisteredExecutor(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		inbox := NewInbox(2)
		runner := NewRunner(inbox, nil)
		done := make(chan domain.Scope, 1)
		Handle[domain.Interpret](runner, func(_ context.Context, effect domain.Interpret, scope domain.Scope, in ports.Inbox) {
			if effect.Transcript != "hello" {
				t.Errorf("transcript = %q", effect.Transcript)
			}
			done <- scope
			_ = in
		})
		runner.Run([]domain.Effect{domain.Interpret{Transcript: "hello"}})
		synctest.Wait()
		select {
		case <-done:
		default:
			t.Fatal("registered executor did not run")
		}
	})
}

func TestRunner_Handle_duplicateRegistrationPanics(t *testing.T) {
	runner := NewRunner(NewInbox(), nil)
	Handle[domain.Transcribe](runner, func(context.Context, domain.Transcribe, domain.Scope, ports.Inbox) {})
	defer func() {
		if recover() == nil {
			t.Fatal("duplicate registration did not panic")
		}
	}()
	Handle[domain.Transcribe](runner, func(context.Context, domain.Transcribe, domain.Scope, ports.Inbox) {})
}

func TestRunner_unregisteredPostsFailureEvent(t *testing.T) {
	inbox := NewInbox(1)
	runner := NewRunner(inbox, nil)
	runner.Run([]domain.Effect{domain.Interpret{UtteranceID: "u1"}})
	env, ok := inbox.Receive(context.Background())
	if !ok {
		t.Fatal("failure event was not posted")
	}
	event, ok := env.Event.(domain.InterpretFailed)
	if !ok || event.UtteranceID != "u1" {
		t.Fatalf("failure event = %#v", env.Event)
	}
}

func TestRunner_controlEffectsAreNotWorkExecutors(t *testing.T) {
	runner := NewRunner(NewInbox(), nil)
	runner.Run([]domain.Effect{domain.PauseAll{}, domain.StartTimer{After: time.Second}})
}
