package main

import (
	"context"
	"errors"
	"github.com/monstercameron/DungeonFlux/web/shell/watch"
	"io"
	"testing"
	"time"

	dungeonfluxv1 "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// hangingSession hands out streams that deliver one state and then block, like
// a WebSocket that looks open but stopped delivering, until their context ends.
type hangingSession struct {
	fakeSession
	phases []string
	opened chan string
}

func (h *hangingSession) Watch(ctx context.Context, _ *dungeonfluxv1.WatchRequest, _ ...grpc.CallOption) (grpc.ServerStreamingClient[dungeonfluxv1.WatchMessage], error) {
	if len(h.phases) == 0 {
		return nil, errors.New("no stream")
	}
	phase := h.phases[0]
	h.phases = h.phases[1:]
	h.opened <- phase
	return &hangingStream{ctx: ctx, phase: phase}, nil
}

type hangingStream struct {
	ctx   context.Context
	phase string
	sent  bool
}

func (s *hangingStream) Recv() (*dungeonfluxv1.WatchMessage, error) {
	if !s.sent {
		s.sent = true
		return &dungeonfluxv1.WatchMessage{Message: &dungeonfluxv1.WatchMessage_State{State: &dungeonfluxv1.ScreenState{Phase: s.phase}}}, nil
	}
	<-s.ctx.Done()
	return nil, s.ctx.Err()
}
func (s *hangingStream) Header() (metadata.MD, error) { return nil, nil }
func (s *hangingStream) Trailer() metadata.MD         { return nil }
func (s *hangingStream) CloseSend() error             { return nil }
func (s *hangingStream) Context() context.Context     { return s.ctx }
func (s *hangingStream) SendMsg(any) error            { return nil }
func (s *hangingStream) RecvMsg(any) error            { return io.EOF }

func TestWatchLoop_ResubscribesWithoutSurfacingAnError(t *testing.T) {
	tests := []struct {
		name    string
		idle    time.Duration
		trigger func(*Client)
	}{
		{name: "Resync drops a silent stream and resubscribes", idle: time.Hour, trigger: func(c *Client) { c.Resync() }},
		{name: "the idle watchdog drops a silent stream and resubscribes", idle: 20 * time.Millisecond, trigger: func(*Client) {}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			session := &hangingSession{phases: []string{"opening", "exploration"}, opened: make(chan string, 2)}
			client := &Client{session: session, resync: watch.Controller{Idle: tc.idle}}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			results := client.Watch(ctx, &dungeonfluxv1.WatchRequest{})
			if got := (<-results).Message.GetState().GetPhase(); got != "opening" {
				t.Fatalf("first phase = %q, want opening", got)
			}
			<-session.opened
			tc.trigger(client)
			select {
			case result := <-results:
				if result.Err != nil || result.Message.GetState().GetPhase() != "exploration" {
					t.Fatalf("after resync got %+v, want the replayed exploration state and no error", result)
				}
			case <-time.After(2 * time.Second):
				t.Fatal("watch did not resubscribe")
			}
		})
	}
}

func TestResync_isSafeBeforeAnyWatchAndOnNil(t *testing.T) {
	var nilClient *Client
	nilClient.Resync()
	client := &Client{}
	client.Resync()
	client.Resync()
	if client.resyncSignal() == nil {
		t.Fatal("resync signal was not recreated")
	}
}
