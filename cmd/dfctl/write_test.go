package main

import (
	"context"
	"io"
	"strings"
	"testing"

	dungeonfluxv1 "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
	"google.golang.org/grpc"
)

func TestRunWrite_AllDemoVerbs(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{"send", []string{"send", "host_skip", `{"x":1}`}, "send"},
		{"act", []string{"act", "--seat", "2", "attack", "--arg", "fast", "--target", "thrall", "--cell", "3,4"}, "act"},
		{"say", []string{"say", "--seat", "1", "hello", "there"}, "say"},
		{"dice", []string{"dice", "force", "d20=17"}, "dice"},
		{"reset", []string{"reset", "--seed", "abcd"}, "reset"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client := &writeFakeClient{}
			var output, stderr strings.Builder
			dial := func(context.Context, string) (dungeonfluxv1.DebugServiceClient, io.Closer, error) {
				return client, io.NopCloser(strings.NewReader("")), nil
			}
			if code, handled := runWrite(context.Background(), test.args, &output, &stderr, dial); !handled || code != exitOK {
				t.Fatalf("runWrite() = %d, %v, stderr=%q", code, handled, stderr.String())
			}
			if client.method != test.want {
				t.Fatalf("method = %q, want %q", client.method, test.want)
			}
			if output.String() == "" {
				t.Fatal("write response was not emitted")
			}
		})
	}
}

func TestMakeWriteCommand_RejectsBadInputs(t *testing.T) {
	for _, args := range [][]string{
		{"send"}, {"send", "event", "not-json"}, {"say", "--seat", "1"},
		{"dice", "force", "d20=21"}, {"reset", "--seed"},
	} {
		if _, err := makeWriteCommand(args[0], "room", args[1:]); err == nil {
			t.Fatalf("makeWriteCommand(%q) accepted invalid input", args)
		}
	}
}

func TestMakeActCommand_RejectsBadCell(t *testing.T) {
	if _, err := makeActCommand("room", []string{"--seat", "1", "move", "--cell", "bad"}); err == nil {
		t.Fatal("bad cell was accepted")
	}
}

type writeFakeClient struct {
	dungeonfluxv1.DebugServiceClient
	method string
}

func (c *writeFakeClient) Send(context.Context, *dungeonfluxv1.SendRequest, ...grpc.CallOption) (*dungeonfluxv1.SendResponse, error) {
	c.method = "send"
	return &dungeonfluxv1.SendResponse{Accepted: true}, nil
}

func (c *writeFakeClient) Act(context.Context, *dungeonfluxv1.DebugActRequest, ...grpc.CallOption) (*dungeonfluxv1.SendResponse, error) {
	c.method = "act"
	return &dungeonfluxv1.SendResponse{Accepted: true}, nil
}

func (c *writeFakeClient) Say(context.Context, *dungeonfluxv1.DebugSayRequest, ...grpc.CallOption) (*dungeonfluxv1.SendResponse, error) {
	c.method = "say"
	return &dungeonfluxv1.SendResponse{Accepted: true}, nil
}

func (c *writeFakeClient) DiceForce(context.Context, *dungeonfluxv1.DiceForceRequest, ...grpc.CallOption) (*dungeonfluxv1.SendResponse, error) {
	c.method = "dice"
	return &dungeonfluxv1.SendResponse{Accepted: true}, nil
}

func (c *writeFakeClient) Reset(context.Context, *dungeonfluxv1.ResetRequest, ...grpc.CallOption) (*dungeonfluxv1.SendResponse, error) {
	c.method = "reset"
	return &dungeonfluxv1.SendResponse{Accepted: true}, nil
}
