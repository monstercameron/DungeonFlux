package main

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"strings"
	"testing"

	dungeonfluxv1 "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/test/bufconn"
)

func TestRun_StateWritesProtoJSONAndMetadata(t *testing.T) {
	listener := bufconn.Listen(1024 * 1024)
	server := grpc.NewServer()
	debug := &fakeDebugServer{state: &dungeonfluxv1.DebugState{Phase: "opening", RunIndex: 3}}
	dungeonfluxv1.RegisterDebugServiceServer(server, debug)
	go server.Serve(listener)
	defer server.Stop()

	dial := func(ctx context.Context, _ string) (dungeonfluxv1.DebugServiceClient, io.Closer, error) {
		conn, err := grpc.NewClient("passthrough:///bufnet", grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return listener.Dial()
		}))
		if err != nil {
			return nil, nil, err
		}
		return dungeonfluxv1.NewDebugServiceClient(conn), conn, nil
	}

	var output, errors strings.Builder
	code := run(context.Background(), []string{"--room", "room-1", "--token", "secret", "state"}, &output, &errors, dial)
	if code != exitOK || errors.Len() != 0 {
		t.Fatalf("run() = code %d, stderr %q", code, errors.String())
	}
	var decoded map[string]any
	if err := json.Unmarshal([]byte(output.String()), &decoded); err != nil {
		t.Fatalf("output is not JSON: %v", err)
	}
	if decoded["phase"] != "opening" || decoded["run_index"] != "3" {
		t.Fatalf("unexpected output: %s", output.String())
	}
	if debug.token != "secret" || debug.room != "room-1" {
		t.Fatalf("metadata/request not passed: token=%q room=%q", debug.token, debug.room)
	}
}

func TestRun_PrettyAndParseErrors(t *testing.T) {
	var output, errors strings.Builder
	dial := func(context.Context, string) (dungeonfluxv1.DebugServiceClient, io.Closer, error) {
		return &fakeClient{state: &dungeonfluxv1.DebugState{Phase: "lobby"}}, io.NopCloser(strings.NewReader("")), nil
	}
	if code := run(context.Background(), []string{"--pretty", "state"}, &output, &errors, dial); code != exitOK {
		t.Fatalf("pretty run() = %d, want %d", code, exitOK)
	}
	if !strings.Contains(output.String(), "\n  \"phase\"") {
		t.Fatalf("pretty output = %q", output.String())
	}
	output.Reset()
	if code := run(context.Background(), []string{"--unknown", "state"}, &output, &errors, dial); code != exitTransport {
		t.Fatalf("parse failure code = %d, want %d", code, exitTransport)
	}
}

func TestRun_TransportErrorUsesExitTwo(t *testing.T) {
	var output, errors strings.Builder
	dial := func(context.Context, string) (dungeonfluxv1.DebugServiceClient, io.Closer, error) {
		return nil, nil, context.DeadlineExceeded
	}
	if code := run(context.Background(), []string{"state"}, &output, &errors, dial); code != exitTransport {
		t.Fatalf("run() = %d, want %d", code, exitTransport)
	}
	if !strings.Contains(errors.String(), "deadline exceeded") {
		t.Fatalf("stderr = %q", errors.String())
	}
}

func TestWriteMessage_UsesProtoFieldNames(t *testing.T) {
	var output strings.Builder
	if err := writeMessage(&output, &dungeonfluxv1.DebugState{RunIndex: 2}, false); err != nil {
		t.Fatal(err)
	}
	if output.String() != "{\"run_index\":\"2\"}\n" {
		t.Fatalf("output = %q", output.String())
	}
}

func TestRejected(t *testing.T) {
	if !rejected(&dungeonfluxv1.SendResponse{}) || rejected(&dungeonfluxv1.SendResponse{Accepted: true}) {
		t.Fatal("SendResponse rejection classification is wrong")
	}
	if !rejected(&dungeonfluxv1.SnapshotResponse{}) || rejected(&dungeonfluxv1.SnapshotResponse{Ok: true}) {
		t.Fatal("SnapshotResponse rejection classification is wrong")
	}
	if !rejected(&dungeonfluxv1.VendorResponse{}) || rejected(&dungeonfluxv1.VendorResponse{Ok: true}) {
		t.Fatal("VendorResponse rejection classification is wrong")
	}
	if !rejected(&dungeonfluxv1.ClientResponse{}) || rejected(&dungeonfluxv1.ClientResponse{Ok: true}) {
		t.Fatal("ClientResponse rejection classification is wrong")
	}
	if rejected(&dungeonfluxv1.DebugState{}) {
		t.Fatal("read response was classified as rejected")
	}
}

func TestParseOptions_DefaultsAndVerb(t *testing.T) {
	options, verb, err := parseOptions([]string{"--addr", "localhost:9444", "--room", "demo", "--token", "t", "state"}, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if options.address != "localhost:9444" || options.room != "demo" || options.token != "t" || options.pretty || verb != "state" {
		t.Fatalf("parseOptions() = %#v, %q", options, verb)
	}
	if _, _, err := parseOptions([]string{}, io.Discard); err == nil {
		t.Fatal("missing verb was accepted")
	}
}

func TestWriteError_IsJSON(t *testing.T) {
	var output strings.Builder
	writeError(&output, context.DeadlineExceeded)
	var decoded map[string]string
	if err := json.Unmarshal([]byte(output.String()), &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded["error"] != "context deadline exceeded" {
		t.Fatalf("decoded error = %#v", decoded)
	}
}

type fakeDebugServer struct {
	dungeonfluxv1.UnimplementedDebugServiceServer
	state *dungeonfluxv1.DebugState
	token string
	room  string
}

func (s *fakeDebugServer) State(ctx context.Context, request *dungeonfluxv1.DebugRoom) (*dungeonfluxv1.DebugState, error) {
	s.room = request.GetRoom()
	if values := metadataFromContext(ctx); len(values) > 0 {
		s.token = values[0]
	}
	return s.state, nil
}

func metadataFromContext(ctx context.Context) []string {
	// Kept in this test file so the fake asserts the wire-level contract.
	values, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil
	}
	return values.Get("x-df-debug-token")
}

type fakeClient struct {
	dungeonfluxv1.DebugServiceClient
	state *dungeonfluxv1.DebugState
}

func (c *fakeClient) State(context.Context, *dungeonfluxv1.DebugRoom, ...grpc.CallOption) (*dungeonfluxv1.DebugState, error) {
	return c.state, nil
}
