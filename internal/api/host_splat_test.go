package api

import (
	"context"
	"testing"

	df "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

func TestHostSplat_MapsBothCommandsAndProjectsPolicy(t *testing.T) {
	for _, tc := range []struct {
		name    string
		command df.HostCommandKind
		want    vocab.HostCmd
		enabled bool
	}{
		{"off", df.HostCommandKind_HOST_COMMAND_KIND_SPLAT_OFF, vocab.HostSplatOff, false},
		{"on", df.HostCommandKind_HOST_COMMAND_KIND_SPLAT_ON, vocab.HostCmd("SPLAT_ON"), true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			inbox := newAcceptingInbox()
			server, err := NewHostServer(inbox, "test-host")
			if err != nil {
				t.Fatal(err)
			}
			ack, err := server.Command(context.Background(), &df.HostCommand{HostToken: "test-host", Command: tc.command})
			if err != nil || !ack.GetOk() {
				t.Fatalf("ack=%v err=%v", ack, err)
			}
			if event := inbox.Calls[0].Envelope.Event.(domain.HostCmd); event.Cmd != tc.want {
				t.Fatalf("command=%s", event.Cmd)
			}
			view := ProjectHost(domain.View{SplatEnabled: tc.enabled, SplatAvailable: true})
			if view.GetSplatEnabled() != tc.enabled || !view.GetSplatAvailable() {
				t.Fatal("host lost authoritative renderer policy")
			}
		})
	}
}
