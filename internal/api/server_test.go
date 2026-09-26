package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/websocket"
	"google.golang.org/grpc"
)

func TestMountGRPC_WebSocketOriginAllowlist(t *testing.T) {
	grpcServer := grpc.NewServer()
	apiServer, err := NewServer(grpcServer, []string{"https://allowed.example"})
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}
	httpServer := httptest.NewServer(apiServer.Handler())
	defer httpServer.Close()

	url := "ws" + httpServer.URL[len("http"):] + "/grpc"
	dialer := websocket.Dialer{}
	conn, response, err := dialer.Dial(url, map[string][]string{"Origin": {"https://allowed.example"}})
	if err != nil {
		t.Fatalf("Dial() error = %v", err)
	}
	defer conn.Close()
	if response.StatusCode != 101 {
		t.Fatalf("status = %d, want 101", response.StatusCode)
	}
}

func TestMountGRPC_RejectsDisallowedOrigin(t *testing.T) {
	grpcServer := grpc.NewServer()
	apiServer, err := NewServer(grpcServer, []string{"https://allowed.example"})
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}
	httpServer := httptest.NewServer(apiServer.Handler())
	defer httpServer.Close()

	url := "ws" + httpServer.URL[len("http"):] + "/grpc"
	_, response, err := (&websocket.Dialer{}).Dial(url, map[string][]string{"Origin": {"https://blocked.example"}})
	if err == nil {
		t.Fatal("Dial() error = nil, want origin rejection")
	}
	if response == nil || response.StatusCode != 403 {
		t.Fatalf("status = %v, want 403", response)
	}
}

func TestNewServer_RejectsNilGRPCServer(t *testing.T) {
	if _, err := NewServer(nil, nil); err == nil {
		t.Fatal("NewServer(nil) error = nil")
	}
}

func TestMountGRPC_RejectsNilArguments(t *testing.T) {
	grpcServer := grpc.NewServer()
	if err := MountGRPC(nil, grpcServer, nil); err == nil {
		t.Fatal("MountGRPC(nil mux) error = nil")
	}
	if err := MountGRPC(http.NewServeMux(), nil, nil); err == nil {
		t.Fatal("MountGRPC(nil grpc) error = nil")
	}
}
