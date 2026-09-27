package main

import (
	"context"
	"encoding/json"
	"io"
	"strings"
	"testing"

	dungeonfluxv1 "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
	"google.golang.org/grpc"
)

func TestControlCommands_CallDeclaredRPCs(t *testing.T) {
	tests := []struct {
		name   string
		args   []string
		method string
	}{
		{"goto", []string{"goto", "combat", "--turn", "pc1"}, "send"},
		{"seat set", []string{"seat", "set", "1", "--name", "Astra", "--hp", "12"}, "send"},
		{"seat ready", []string{"seat", "ready", "1"}, "act"},
		{"timers", []string{"timers"}, "state"},
		{"timers off", []string{"timers", "off"}, "send"},
		{"timer", []string{"timer", "set", "turn_timer", "250"}, "send"},
		{"pause", []string{"pause"}, "send"},
		{"combat", []string{"combat", "hp", "thrall", "4"}, "send"},
		{"combat end", []string{"combat", "end", "slain"}, "send"},
		{"snapshot", []string{"snapshot", "save", "before"}, "snapshot"},
		{"vendor", []string{"vendor", "llm", "slow", "50"}, "vendor"},
		{"client", []string{"client", "DM", "reload"}, "client"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			client := &controlFakeClient{}
			_, command, _, err := parseControlOptions(tc.args, tc.args[0], io.Discard)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := command(context.Background(), client); err != nil {
				t.Fatal(err)
			}
			if client.method != tc.method {
				t.Fatalf("method = %q, want %q", client.method, tc.method)
			}
		})
	}
}

func TestRunControl_DryRunDoesNotDial(t *testing.T) {
	var output, stderr strings.Builder
	dialed := false
	dial := func(context.Context, string) (dungeonfluxv1.DebugServiceClient, io.Closer, error) {
		dialed = true
		return nil, nil, nil
	}
	code, handled := runControl(context.Background(), []string{"--dry-run", "combat", "move", "pc1", "2,3"}, &output, &stderr, dial)
	if !handled || code != exitOK || dialed || stderr.Len() != 0 {
		t.Fatalf("runControl = %d/%v dialed=%v stdout=%q stderr=%q", code, handled, dialed, output.String(), stderr.String())
	}
	if !strings.Contains(output.String(), `"event":"debug_patch"`) || !strings.Contains(output.String(), `"cell":"2,3"`) {
		t.Fatalf("dry run = %q", output.String())
	}
	output.Reset()
	code, handled = runControl(context.Background(), []string{"--dry-run", "snapshot", "save", "before"}, &output, &stderr, dial)
	if !handled || code != exitOK || !strings.Contains(output.String(), `"operation":"save"`) {
		t.Fatalf("snapshot dry run = %d/%v %q", code, handled, output.String())
	}
}

func TestRunWriteDryRun_CoversDemoWrites(t *testing.T) {
	for _, args := range [][]string{
		{"--dry-run", "send", "host_skip"},
		{"--dry-run", "act", "--seat", "1", "attack", "--target", "thrall"},
		{"--dry-run", "say", "--seat", "1", "hello", "there"},
		{"--dry-run", "reset", "--seed", "abcd"},
		{"--dry-run", "dice", "force", "d20=17"},
	} {
		var output, stderr strings.Builder
		if code, handled := runWrite(context.Background(), args, &output, &stderr, nil); !handled || code != exitOK || output.Len() == 0 {
			t.Fatalf("args=%v code=%d handled=%v output=%q stderr=%q", args, code, handled, output.String(), stderr.String())
		}
	}
}

func TestControlParsingRejectsInvalidCommands(t *testing.T) {
	for _, args := range [][]string{
		{"goto", "nope"}, {"goto", "opening", "--turn", "pc1"},
		{"seat", "set", "1", "--unknown", "x"}, {"timer", "set", "x", "bad"},
		{"combat", "end", "thrall", "extra"}, {"vendor", "x", "slow"},
		{"client", "DM", "screenshot"},
	} {
		if _, _, _, err := parseControlOptions(args, args[0], io.Discard); err == nil {
			t.Fatalf("accepted %v", args)
		}
	}
}

func TestCombatEnd_DocumentedSyntaxTargetsTheEnemy(t *testing.T) {
	for _, tc := range []struct {
		name    string
		args    []string
		outcome string
	}{
		{"slain", []string{"end", "slain"}, "slain"},
		{"fled", []string{"end", "fled"}, "fled"},
		{"legacy explicit enemy", []string{"end", "thrall", "slain"}, "slain"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, message, err := makeCombatCommand("test-room", tc.args)
			if err != nil {
				t.Fatal(err)
			}
			request := message.(*dungeonfluxv1.SendRequest)
			var payload struct {
				Target string
				Fields map[string]string
			}
			if err := json.Unmarshal([]byte(request.GetPayloadJson()), &payload); err != nil {
				t.Fatal(err)
			}
			if request.GetRoom() != "test-room" || request.GetEvent() != "debug_patch" || payload.Target != "token:thrall" || payload.Fields["outcome"] != tc.outcome {
				t.Fatalf("combat end request: %v, payload=%+v", request, payload)
			}
		})
	}
	for _, args := range [][]string{{}, {"end"}, {"end", "dead"}, {"end", "pc1", "slain"}, {"end", "slain", "extra"}} {
		if _, _, err := makeCombatCommand("test-room", args); err == nil {
			t.Fatalf("accepted malformed combat command %v", args)
		}
	}
}

func TestControlHelpers(t *testing.T) {
	if findControlVerb([]string{"--room", "x", "act"}) != "" {
		t.Fatal("write verb detected as control")
	}
	if !validPhase("hook_event") || validPhase("bad") || !validSeatField("cell") || validSeatField("bad") {
		t.Fatal("validation helpers are wrong")
	}
	if _, _, err := parseCell("bad"); err == nil {
		t.Fatal("bad cell accepted")
	}
	if got := marshalJSON(map[string]string{"ok": "yes"}); got != `{"ok":"yes"}` {
		t.Fatalf("marshalJSON = %q", got)
	}
}

type controlFakeClient struct {
	dungeonfluxv1.DebugServiceClient
	method string
}

func (c *controlFakeClient) Send(context.Context, *dungeonfluxv1.SendRequest, ...grpc.CallOption) (*dungeonfluxv1.SendResponse, error) {
	c.method = "send"
	return &dungeonfluxv1.SendResponse{Accepted: true}, nil
}
func (c *controlFakeClient) Act(context.Context, *dungeonfluxv1.DebugActRequest, ...grpc.CallOption) (*dungeonfluxv1.SendResponse, error) {
	c.method = "act"
	return &dungeonfluxv1.SendResponse{Accepted: true}, nil
}
func (c *controlFakeClient) State(context.Context, *dungeonfluxv1.DebugRoom, ...grpc.CallOption) (*dungeonfluxv1.DebugState, error) {
	c.method = "state"
	return &dungeonfluxv1.DebugState{}, nil
}
func (c *controlFakeClient) Snapshot(context.Context, *dungeonfluxv1.SnapshotRequest, ...grpc.CallOption) (*dungeonfluxv1.SnapshotResponse, error) {
	c.method = "snapshot"
	return &dungeonfluxv1.SnapshotResponse{Ok: true}, nil
}
func (c *controlFakeClient) Vendor(context.Context, *dungeonfluxv1.VendorRequest, ...grpc.CallOption) (*dungeonfluxv1.VendorResponse, error) {
	c.method = "vendor"
	return &dungeonfluxv1.VendorResponse{Ok: true}, nil
}
func (c *controlFakeClient) Client(context.Context, *dungeonfluxv1.ClientRequest, ...grpc.CallOption) (*dungeonfluxv1.ClientResponse, error) {
	c.method = "client"
	return &dungeonfluxv1.ClientResponse{Ok: true}, nil
}
