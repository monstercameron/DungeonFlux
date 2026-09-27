package api

import (
	"context"
	"testing"
	"testing/synctest"
	"time"

	df "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/fakes"
	"github.com/monstercameron/DungeonFlux/internal/ports"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// acceptingInbox explicitly simulates an engine accepting each posted action.
// Queue-only fakes must never make production code fabricate engine success.
type acceptingInbox struct{ fakes.FakeInbox }

func newAcceptingInbox() *acceptingInbox {
	return &acceptingInbox{FakeInbox: fakes.FakeInbox{PostResult: true}}
}

func (i *acceptingInbox) Post(ctx context.Context, env domain.Envelope) bool {
	if !i.FakeInbox.Post(ctx, env) {
		return false
	}
	if env.Reply != nil {
		env.Reply <- domain.Ack{Accepted: true}
	}
	return true
}

type deferredInbox struct{ posted chan domain.Envelope }

func (i deferredInbox) Post(ctx context.Context, env domain.Envelope) bool {
	if ctx.Err() != nil {
		return false
	}
	if env.Reply != nil {
		i.posted <- env
	}
	return true
}

type actionResult struct {
	ack domain.Ack
	err error
}

func callAcknowledgedAction(t *testing.T, ctx context.Context, inbox ports.Inbox, kind string) actionResult {
	t.Helper()
	if kind == "host" {
		host, err := NewHostServer(inbox, "test-host")
		if err != nil {
			t.Fatal(err)
		}
		result, err := host.Command(ctx, &df.HostCommand{HostToken: "test-host", Command: df.HostCommandKind_HOST_COMMAND_KIND_SAFE_MODE})
		return actionResult{domain.Ack{Accepted: result.GetOk(), Reason: result.GetReason()}, err}
	}
	server, err := NewSessionServer(inbox, "ROOM", "HOST", "DM")
	if err != nil {
		t.Fatal(err)
	}
	server.seats["test-seat"] = seatSession{id: 1}
	if kind == "act" {
		result, err := server.Act(ctx, &df.ActRequest{SeatToken: "test-seat", MoveId: "attack"})
		return actionResult{domain.Ack{Accepted: result.GetAccepted(), Reason: result.GetReason()}, err}
	}
	result, err := server.Say(ctx, &df.SayRequest{SeatToken: "test-seat", Text: "hello"})
	if !result.GetAccepted() && result.GetUtteranceId() != "" {
		t.Error("rejected speech published an utterance ID")
	}
	return actionResult{domain.Ack{Accepted: result.GetAccepted(), Reason: result.GetReason()}, err}
}

func TestActions_WaitForAuthoritativeReply(t *testing.T) {
	for _, kind := range []string{"host", "act", "say"} {
		for _, ack := range []domain.Ack{{Accepted: true}, {Reason: "unaccepted_event"}} {
			t.Run(kind+"/"+ack.Reason, func(t *testing.T) {
				synctest.Test(t, func(t *testing.T) {
					inbox := deferredInbox{posted: make(chan domain.Envelope, 1)}
					done := make(chan actionResult, 1)
					go func() { done <- callAcknowledgedAction(t, context.Background(), inbox, kind) }()
					synctest.Wait()
					select {
					case result := <-done:
						t.Fatalf("returned before engine decision: %+v", result)
					default:
					}
					env := <-inbox.posted
					env.Reply <- ack
					result := <-done
					if result.err != nil || result.ack != ack {
						t.Fatalf("result=%+v want=%+v", result, ack)
					}
				})
			})
		}
	}
}

func TestActions_CancellationDoesNotBlockLateEngineReply(t *testing.T) {
	for _, kind := range []string{"host", "act", "say"} {
		t.Run(kind, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				inbox := deferredInbox{posted: make(chan domain.Envelope, 1)}
				done := make(chan actionResult, 1)
				go func() { done <- callAcknowledgedAction(t, ctx, inbox, kind) }()
				synctest.Wait()
				env := <-inbox.posted
				cancel()
				result := <-done
				if status.Code(result.err) != codes.Canceled || result.ack.Accepted {
					t.Fatalf("canceled result=%+v", result)
				}
				select {
				case env.Reply <- domain.Ack{Accepted: true}:
				default:
					t.Fatal("abandoned RPC would block the room loop")
				}
			})
		})
	}
}

func TestActions_EnqueueFailures(t *testing.T) {
	for _, kind := range []string{"host", "act", "say"} {
		for _, tc := range []struct {
			name string
			code codes.Code
		}{
			{"full", codes.ResourceExhausted}, {"canceled", codes.Canceled}, {"deadline", codes.DeadlineExceeded},
		} {
			t.Run(kind+"/"+tc.name, func(t *testing.T) {
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				if tc.name == "canceled" {
					cancel()
				}
				if tc.name == "deadline" {
					var deadlineCancel context.CancelFunc
					ctx, deadlineCancel = context.WithDeadline(ctx, time.Time{})
					defer deadlineCancel()
				}
				result := callAcknowledgedAction(t, ctx, &fakes.FakeInbox{}, kind)
				if result.ack.Accepted || status.Code(result.err) != tc.code {
					t.Fatalf("result=%+v want code=%v", result, tc.code)
				}
			})
		}
	}
}
