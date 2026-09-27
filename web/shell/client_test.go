package main

import (
	"context"
	"errors"
	"io"
	"testing"

	dungeonfluxv1 "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func TestBridgeWebSocketURL(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
		err  bool
	}{
		{name: "http adds path", in: "http://localhost:8443", want: "ws://localhost:8443/grpc"},
		{name: "https keeps path", in: "https://dm.test/grpc", want: "wss://dm.test/grpc"},
		{name: "websocket unchanged", in: "ws://localhost/grpc", want: "ws://localhost/grpc"},
		{name: "unsupported scheme", in: "ftp://localhost", err: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := BridgeWebSocketURL(test.in)
			if (err != nil) != test.err {
				t.Fatalf("error = %v, want error %t", err, test.err)
			}
			if err == nil && got != test.want {
				t.Fatalf("URL = %q, want %q", got, test.want)
			}
		})
	}
}

func TestWatchLoop_ReconnectsAndDeliversMessages(t *testing.T) {
	firstErr := errors.New("dropped")
	fake := &fakeSession{streams: []*fakeWatchStream{
		{messages: []*dungeonfluxv1.WatchMessage{{Message: &dungeonfluxv1.WatchMessage_State{State: &dungeonfluxv1.ScreenState{Phase: "first"}}}}, err: firstErr},
		{messages: []*dungeonfluxv1.WatchMessage{{Message: &dungeonfluxv1.WatchMessage_State{State: &dungeonfluxv1.ScreenState{Phase: "second"}}}}, err: io.EOF},
	}}
	client := &Client{session: fake}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	results := client.Watch(ctx, &dungeonfluxv1.WatchRequest{})
	first := <-results
	if first.Message == nil || first.Message.GetState().GetPhase() != "first" {
		t.Fatalf("first result = %#v", first)
	}
	if got := (<-results).Err; !errors.Is(got, firstErr) {
		t.Fatalf("drop error = %v, want %v", got, firstErr)
	}
	second := <-results
	if second.Message == nil || second.Message.GetState().GetPhase() != "second" {
		t.Fatalf("second result = %#v", second)
	}
	cancel()
}

func TestWatchLoop_StopsWhenContextIsCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	results := (&Client{session: &fakeSession{}}).Watch(ctx, &dungeonfluxv1.WatchRequest{})
	if _, ok := <-results; ok {
		t.Fatal("watch channel remained open")
	}
}

func TestAsyncJoin_DoesNotRequireCallerToBlock(t *testing.T) {
	fake := &fakeSession{join: &dungeonfluxv1.JoinResponse{SeatId: "seat-1"}}
	result := (&Client{session: fake}).Join(context.Background(), &dungeonfluxv1.JoinRequest{})
	got := <-result
	if got.Err != nil || got.Value.GetSeatId() != "seat-1" {
		t.Fatalf("result = %#v, want seat-1", got)
	}
}

func TestAsyncJoin_RecordsPlayerNumberForPhoneAudio(t *testing.T) {
	client := &Client{session: &fakeSession{join: &dungeonfluxv1.JoinResponse{PlayerNumber: 2}}}
	if result := <-client.Join(context.Background(), &dungeonfluxv1.JoinRequest{}); result.Err != nil {
		t.Fatalf("join error = %v", result.Err)
	}
	if got := client.PlayerNumber(); got != 2 {
		t.Fatalf("player number = %d, want 2", got)
	}
}

func TestClient_ListenRejectsUnavailableConnection(t *testing.T) {
	if _, err := (&Client{}).Listen(context.Background(), &dungeonfluxv1.ListenRequest{}); err == nil {
		t.Fatal("Listen accepted a client without a connection")
	}
}

func TestAsyncActAndSay_ReturnResponses(t *testing.T) {
	client := &Client{session: &fakeSession{}}
	act := <-client.Act(context.Background(), &dungeonfluxv1.ActRequest{})
	if act.Err != nil || !act.Value.GetAccepted() {
		t.Fatalf("act result = %#v", act)
	}
	say := <-client.Say(context.Background(), &dungeonfluxv1.SayRequest{})
	if say.Err != nil || !say.Value.GetAccepted() {
		t.Fatalf("say result = %#v", say)
	}
}

func TestNewClient_ValidEndpointCanClose(t *testing.T) {
	client, err := NewClient(context.Background(), "ws://localhost:1/grpc")
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	if err := client.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
}

func TestWaitWatchRetry_ContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if waitWatchRetry(ctx, minWatchRetry) {
		t.Fatal("waitWatchRetry returned true after cancellation")
	}
}

type fakeSession struct {
	streams []*fakeWatchStream
	join    *dungeonfluxv1.JoinResponse
}

func (f *fakeSession) Join(context.Context, *dungeonfluxv1.JoinRequest, ...grpc.CallOption) (*dungeonfluxv1.JoinResponse, error) {
	return f.join, nil
}
func (f *fakeSession) Act(context.Context, *dungeonfluxv1.ActRequest, ...grpc.CallOption) (*dungeonfluxv1.ActResponse, error) {
	return &dungeonfluxv1.ActResponse{Accepted: true}, nil
}
func (f *fakeSession) Say(context.Context, *dungeonfluxv1.SayRequest, ...grpc.CallOption) (*dungeonfluxv1.SayResponse, error) {
	return &dungeonfluxv1.SayResponse{Accepted: true}, nil
}
func (f *fakeSession) Watch(context.Context, *dungeonfluxv1.WatchRequest, ...grpc.CallOption) (grpc.ServerStreamingClient[dungeonfluxv1.WatchMessage], error) {
	if len(f.streams) == 0 {
		return nil, errors.New("no stream")
	}
	stream := f.streams[0]
	f.streams = f.streams[1:]
	return stream, nil
}

type fakeWatchStream struct {
	messages []*dungeonfluxv1.WatchMessage
	err      error
}

func (f *fakeWatchStream) Recv() (*dungeonfluxv1.WatchMessage, error) {
	if len(f.messages) == 0 {
		return nil, f.err
	}
	message := f.messages[0]
	f.messages = f.messages[1:]
	return message, nil
}
func (f *fakeWatchStream) Header() (metadata.MD, error) { return nil, nil }
func (f *fakeWatchStream) Trailer() metadata.MD         { return nil }
func (f *fakeWatchStream) CloseSend() error             { return nil }
func (f *fakeWatchStream) Context() context.Context     { return context.Background() }
func (f *fakeWatchStream) SendMsg(any) error            { return nil }
func (f *fakeWatchStream) RecvMsg(any) error            { return io.EOF }
