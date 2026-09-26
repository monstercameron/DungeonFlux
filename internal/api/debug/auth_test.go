package debug

import (
	"context"
	"net"
	"testing"

	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
)

func TestAuth_RequiresExactMetadataToken(t *testing.T) {
	base := context.Background()
	if authorized(base, "secret") {
		t.Fatal("unauthenticated context accepted")
	}
	ctx := metadata.NewIncomingContext(base, metadata.Pairs(tokenMetadata, "secret"))
	if !authorized(ctx, "secret") || authorized(ctx, "wrong") {
		t.Fatal("token validation failed")
	}
	if authorized(metadata.NewIncomingContext(base, metadata.Pairs(tokenMetadata, "a", tokenMetadata, "secret")), "secret") {
		t.Fatal("multiple tokens accepted")
	}
}

func TestAuth_RequiresLoopbackPeer(t *testing.T) {
	if loopbackPeer(context.Background()) {
		t.Fatal("missing peer accepted")
	}
	for _, address := range []string{"127.0.0.1:1", "[::1]:1"} {
		ctx := peer.NewContext(context.Background(), &peer.Peer{Addr: mustAddr(t, "tcp", address)})
		if !loopbackPeer(ctx) {
			t.Fatalf("loopback %s rejected", address)
		}
	}
	ctx := peer.NewContext(context.Background(), &peer.Peer{Addr: mustAddr(t, "tcp", "10.0.0.1:1")})
	if loopbackPeer(ctx) {
		t.Fatal("non-loopback peer accepted")
	}
}

func mustAddr(t *testing.T, network, address string) net.Addr {
	t.Helper()
	addr, err := net.ResolveTCPAddr(network, address)
	if err != nil {
		t.Fatal(err)
	}
	return addr
}
