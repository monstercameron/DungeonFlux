package debug

import (
	"context"
	"testing"

	df "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/fakes"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

func TestWrites_PostDemoEvents(t *testing.T) {
	inbox := &fakes.FakeInbox{PostResult: true}
	server, err := NewServer(&fakes.FakeEngine{}, inbox)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if got, err := server.Act(ctx, &df.DebugActRequest{Seat: "2", MoveId: "attack", TargetId: "thrall", Cell: &df.Cell{C: 1, R: 3}}); err != nil || !got.GetAccepted() {
		t.Fatalf("act = %+v, err = %v", got, err)
	}
	if got, err := server.Say(ctx, &df.DebugSayRequest{Seat: "1", Text: "hello"}); err != nil || !got.GetAccepted() {
		t.Fatalf("say = %+v, err = %v", got, err)
	}
	if got, err := server.DiceForce(ctx, &df.DiceForceRequest{D20: 17}); err != nil || !got.GetAccepted() {
		t.Fatalf("dice = %+v, err = %v", got, err)
	}
	if got, err := server.Reset(ctx, &df.ResetRequest{Seed: "0102"}); err != nil || !got.GetAccepted() {
		t.Fatalf("reset = %+v, err = %v", got, err)
	}
	if len(inbox.Calls) != 4 {
		t.Fatalf("calls = %d, want 4", len(inbox.Calls))
	}
	if _, ok := inbox.Calls[0].Envelope.Event.(domain.Act); !ok {
		t.Fatalf("first event = %T", inbox.Calls[0].Envelope.Event)
	}
}

func TestWrites_SendDecodesHostAndDebugEvents(t *testing.T) {
	inbox := &fakes.FakeInbox{PostResult: true}
	server, _ := NewServer(&fakes.FakeEngine{}, inbox)
	for _, tc := range []struct {
		name, kind, payload string
		want                vocab.HostCmd
	}{
		{"pause", "host_pause", "", vocab.HostPause},
		{"force", "host_force_d20", `{"n":19}`, vocab.HostForceD20},
		{"reset", "host_reset", "", vocab.HostReset},
		{"goto", "debug_goto", `{"phase":"combat"}`, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := server.Send(context.Background(), &df.SendRequest{Event: tc.kind, PayloadJson: tc.payload})
			if err != nil || !got.GetAccepted() {
				t.Fatalf("send = %+v, err = %v", got, err)
			}
			if tc.want != "" {
				host, ok := inbox.Calls[len(inbox.Calls)-1].Envelope.Event.(domain.HostCmd)
				if !ok || host.Cmd != tc.want {
					t.Fatalf("host event = %#v, want %s", inbox.Calls[len(inbox.Calls)-1].Envelope.Event, tc.want)
				}
			}
		})
	}
	failed := &fakes.FakeInbox{PostResult: false}
	rejected, _ := NewServer(&fakes.FakeEngine{}, failed)
	response, err := rejected.Send(context.Background(), &df.SendRequest{Event: "host_pause"})
	if err != nil || response.GetAccepted() || response.GetReason() == "" {
		t.Fatalf("rejection = %+v, err = %v", response, err)
	}
}

func TestWrites_RejectInvalidRequests(t *testing.T) {
	server, _ := NewServer(&fakes.FakeEngine{}, &fakes.FakeInbox{PostResult: true})
	for _, call := range []func() error{
		func() error { _, err := server.Send(context.Background(), nil); return err },
		func() error { _, err := server.Send(context.Background(), &df.SendRequest{Event: "nope"}); return err },
		func() error { _, err := server.Act(context.Background(), &df.DebugActRequest{}); return err },
		func() error { _, err := server.Say(context.Background(), &df.DebugSayRequest{}); return err },
		func() error {
			_, err := server.DiceForce(context.Background(), &df.DiceForceRequest{D20: 21})
			return err
		},
		func() error { _, err := server.Reset(context.Background(), &df.ResetRequest{Seed: "nope"}); return err },
	} {
		if err := call(); err == nil {
			t.Fatal("invalid request accepted")
		}
	}
}
