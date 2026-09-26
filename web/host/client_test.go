package host

import (
	"context"
	"testing"

	df "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
	"google.golang.org/grpc"
)

func TestNewHostClient_rejectsEmptyEndpoint(t *testing.T) {
	if _, err := newHostClient(""); err == nil {
		t.Fatal("newHostClient(\"\") returned nil error")
	}
}

func TestHostClientClose_acceptsNilClient(t *testing.T) {
	var client *hostClient
	if err := client.close(); err != nil {
		t.Fatalf("close() error = %v", err)
	}
}

func TestHostClientClose_closesConnection(t *testing.T) {
	client, err := newHostClient("passthrough:///127.0.0.1:1")
	if err != nil {
		t.Fatalf("newHostClient() error = %v", err)
	}
	if err := client.close(); err != nil {
		t.Fatalf("close() error = %v", err)
	}
}

func TestHostClientCommand_delegatesToService(t *testing.T) {
	want := &df.HostAck{Ok: true, Reason: "accepted"}
	client := &hostClient{service: fakeHostCommandClient{ack: want}}
	got, err := client.command(context.Background(), &df.HostCommand{})
	if err != nil || got != want {
		t.Fatalf("command() = %#v, %v; want %#v, nil", got, err, want)
	}
}

func TestHostClientWatch_withoutSessionClosesChannel(t *testing.T) {
	client := &hostClient{}
	if _, ok := <-client.watch(context.Background(), "token"); ok {
		t.Fatal("watch returned a message without a session")
	}
}

func TestTransportDialOption_returnsOption(t *testing.T) {
	if transportDialOption("endpoint") == nil {
		t.Fatal("transportDialOption() returned nil")
	}
}

type fakeHostCommandClient struct {
	ack *df.HostAck
}

func (f fakeHostCommandClient) Command(context.Context, *df.HostCommand, ...grpc.CallOption) (*df.HostAck, error) {
	return f.ack, nil
}
