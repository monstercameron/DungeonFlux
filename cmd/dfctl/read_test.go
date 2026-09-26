package main

import (
	"context"
	"io"
	"strings"
	"testing"

	dungeonfluxv1 "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

func TestRunRead_UnaryVerbs(t *testing.T) {
	for _, verb := range []string{"state", "view", "legal", "scopes", "assets", "clients", "costs"} {
		t.Run(verb, func(t *testing.T) {
			client := &readFakeClient{}
			args := []string{verb}
			if verb == "view" {
				args = []string{"view", "--dm"}
			}
			if verb == "legal" {
				args = []string{"legal", "--seat", "2"}
			}
			var output, stderr strings.Builder
			dial := func(context.Context, string) (dungeonfluxv1.DebugServiceClient, io.Closer, error) {
				return client, io.NopCloser(strings.NewReader("")), nil
			}
			if code, handled := runRead(context.Background(), args, &output, &stderr, dial); !handled || code != exitOK {
				t.Fatalf("runRead() = %d, %v, stderr=%q", code, handled, stderr.String())
			}
			if output.Len() == 0 || client.method != verb {
				t.Fatalf("method=%q output=%q", client.method, output.String())
			}
		})
	}
}

func TestRunRead_StreamsRecords(t *testing.T) {
	client := &readFakeClient{events: []*dungeonfluxv1.EventRecord{{Seq: 3}}, logs: []*dungeonfluxv1.LogRecord{{Message: "hello"}}}
	for _, test := range []struct {
		name   string
		args   []string
		method string
	}{
		{"events", []string{"events", "--since", "3", "--follow"}, "events"},
		{"logs", []string{"logs", "--level", "warn", "--trace-id", "trace", "--follow"}, "logs"},
	} {
		t.Run(test.name, func(t *testing.T) {
			var output, stderr strings.Builder
			dial := func(context.Context, string) (dungeonfluxv1.DebugServiceClient, io.Closer, error) {
				return client, io.NopCloser(strings.NewReader("")), nil
			}
			if code, handled := runRead(context.Background(), test.args, &output, &stderr, dial); !handled || code != exitOK {
				t.Fatalf("runRead()=%d,%v stderr=%q", code, handled, stderr.String())
			}
			if client.method != test.method || output.Len() == 0 {
				t.Fatalf("method=%q output=%q", client.method, output.String())
			}
		})
	}
}

func TestReadParsingAndUnsupported(t *testing.T) {
	if _, _, err := parseReadOptions([]string{"events", "--since", "bad"}, "events", io.Discard); err == nil {
		t.Fatal("bad since accepted")
	}
	if _, _, err := parseReadOptions([]string{"logs", "--unknown"}, "logs", io.Discard); err == nil {
		t.Fatal("bad logs flag accepted")
	}
	if _, err := makeReadCommand("view", "", []string{"--dm", "extra"}); err == nil {
		t.Fatal("extra dm argument accepted")
	}
	if got := readError(status.Error(12, "missing")); got.Error() != "not supported by server" {
		t.Fatalf("unsupported error=%v", got)
	}
	if findReadVerb([]string{"--room", "x", "act"}) != "" {
		t.Fatal("write verb detected as read")
	}
}

type readFakeClient struct {
	dungeonfluxv1.DebugServiceClient
	method string
	events []*dungeonfluxv1.EventRecord
	logs   []*dungeonfluxv1.LogRecord
}

func (c *readFakeClient) State(context.Context, *dungeonfluxv1.DebugRoom, ...grpc.CallOption) (*dungeonfluxv1.DebugState, error) {
	c.method = "state"
	return &dungeonfluxv1.DebugState{Phase: "opening"}, nil
}
func (c *readFakeClient) View(context.Context, *dungeonfluxv1.ViewRequest, ...grpc.CallOption) (*dungeonfluxv1.ScreenState, error) {
	c.method = "view"
	return &dungeonfluxv1.ScreenState{}, nil
}
func (c *readFakeClient) Legal(context.Context, *dungeonfluxv1.SeatRequest, ...grpc.CallOption) (*dungeonfluxv1.LegalMoves, error) {
	c.method = "legal"
	return &dungeonfluxv1.LegalMoves{}, nil
}
func (c *readFakeClient) Scopes(context.Context, *dungeonfluxv1.DebugRoom, ...grpc.CallOption) (*dungeonfluxv1.ScopeTree, error) {
	c.method = "scopes"
	return &dungeonfluxv1.ScopeTree{}, nil
}
func (c *readFakeClient) Assets(context.Context, *dungeonfluxv1.DebugRoom, ...grpc.CallOption) (*dungeonfluxv1.AssetList, error) {
	c.method = "assets"
	return &dungeonfluxv1.AssetList{}, nil
}
func (c *readFakeClient) Clients(context.Context, *dungeonfluxv1.DebugRoom, ...grpc.CallOption) (*dungeonfluxv1.ClientList, error) {
	c.method = "clients"
	return &dungeonfluxv1.ClientList{}, nil
}
func (c *readFakeClient) Costs(context.Context, *dungeonfluxv1.DebugRoom, ...grpc.CallOption) (*dungeonfluxv1.CostReport, error) {
	c.method = "costs"
	return &dungeonfluxv1.CostReport{}, nil
}
func (c *readFakeClient) Events(context.Context, *dungeonfluxv1.EventsRequest, ...grpc.CallOption) (grpc.ServerStreamingClient[dungeonfluxv1.EventRecord], error) {
	c.method = "events"
	return &fakeEventStream{items: c.events}, nil
}
func (c *readFakeClient) Logs(context.Context, *dungeonfluxv1.LogsRequest, ...grpc.CallOption) (grpc.ServerStreamingClient[dungeonfluxv1.LogRecord], error) {
	c.method = "logs"
	return &fakeLogStream{items: c.logs}, nil
}

type fakeEventStream struct {
	grpc.ClientStream
	items []*dungeonfluxv1.EventRecord
}

func (s *fakeEventStream) Recv() (*dungeonfluxv1.EventRecord, error) {
	if len(s.items) == 0 {
		return nil, io.EOF
	}
	item := s.items[0]
	s.items = s.items[1:]
	return item, nil
}

type fakeLogStream struct {
	grpc.ClientStream
	items []*dungeonfluxv1.LogRecord
}

func (s *fakeLogStream) Recv() (*dungeonfluxv1.LogRecord, error) {
	if len(s.items) == 0 {
		return nil, io.EOF
	}
	item := s.items[0]
	s.items = s.items[1:]
	return item, nil
}
