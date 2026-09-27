package debug

import (
	"context"
	"testing"

	"github.com/monstercameron/DungeonFlux/internal/fakes"
)

func TestServer_ValidatesDependenciesAndToken(t *testing.T) {
	inbox := &fakes.FakeInbox{PostResult: true}
	if _, err := NewServer(nil, inbox); err == nil {
		t.Fatal("nil engine accepted")
	}
	if _, err := NewServer(&fakes.FakeEngine{}, nil); err == nil {
		t.Fatal("nil inbox accepted")
	}
	server, err := NewServer(&fakes.FakeEngine{}, inbox)
	if err != nil {
		t.Fatal(err)
	}
	if err := server.Register(nil); err == nil {
		t.Fatal("nil grpc server accepted")
	}
	if _, err := NewGRPCServer(server, ""); err == nil {
		t.Fatal("empty token accepted")
	}
	grpcServer, err := NewGRPCServer(server, "secret")
	if err != nil {
		t.Fatal(err)
	}
	grpcServer.Stop()
	if err := Serve(context.Background(), nil, server, "secret"); err == nil {
		t.Fatal("nil listener accepted")
	}
}
