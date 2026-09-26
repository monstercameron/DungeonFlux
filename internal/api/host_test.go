package api

import (
	"context"
	"testing"

	df "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/fakes"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

func TestHostServer_CommandPostsMappedEvent(t *testing.T) {
	inbox := &fakes.FakeInbox{PostResult: true}
	server, err := NewHostServer(inbox, "HOST")
	if err != nil {
		t.Fatal(err)
	}
	response, err := server.Command(context.Background(), &df.HostCommand{HostToken: "HOST", Command: df.HostCommandKind_HOST_COMMAND_KIND_FORCE_D20, D20: 17})
	if err != nil || !response.GetOk() {
		t.Fatalf("Command() = %+v, %v", response, err)
	}
	event, ok := inbox.Calls[0].Envelope.Event.(domain.HostCmd)
	if !ok || event.Cmd != vocab.HostForceD20 || event.N != 17 {
		t.Fatalf("event = %#v", inbox.Calls[0].Envelope.Event)
	}
}

func TestHostServer_RejectsAuthAndUnsupportedCommands(t *testing.T) {
	server, err := NewHostServer(&fakes.FakeInbox{PostResult: true}, "HOST")
	if err != nil {
		t.Fatal(err)
	}
	for _, request := range []*df.HostCommand{
		nil,
		{HostToken: "bad", Command: df.HostCommandKind_HOST_COMMAND_KIND_START},
		{HostToken: "HOST", Command: df.HostCommandKind_HOST_COMMAND_KIND_UNSPECIFIED},
	} {
		if _, err := server.Command(context.Background(), request); err == nil {
			t.Fatalf("Command(%v) error = nil", request)
		}
	}
}

func TestHostServer_EngineAckAndFullInbox(t *testing.T) {
	engine := &fakes.FakeEngine{Steps: []domain.StepOut{{Ack: &domain.Ack{Accepted: false, Reason: "paused"}}}}
	server, err := NewHostServerWithEngine(&fakes.FakeInbox{PostResult: true}, engine, "HOST")
	if err != nil {
		t.Fatal(err)
	}
	response, err := server.Command(context.Background(), &df.HostCommand{HostToken: "HOST", Command: df.HostCommandKind_HOST_COMMAND_KIND_START})
	if err != nil || response.GetOk() || response.GetReason() != "paused" {
		t.Fatalf("response = %+v, err = %v", response, err)
	}
	full, err := NewHostServer(&fakes.FakeInbox{PostResult: false}, "HOST")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := full.Command(context.Background(), &df.HostCommand{HostToken: "HOST", Command: df.HostCommandKind_HOST_COMMAND_KIND_RESET}); err == nil {
		t.Fatal("full inbox error = nil")
	}
}
