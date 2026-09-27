package api

import (
	"context"
	"testing"

	df "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/fakes"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

func TestHostServer_ExplicitTimerCommandsAndProjection(t *testing.T) {
	for _, tc := range []struct {
		name    string
		command df.HostCommandKind
		want    vocab.HostCmd
		enabled bool
	}{
		{"off", df.HostCommandKind_HOST_COMMAND_KIND_TIMERS_OFF, vocab.HostTimersOff, false},
		{"on", df.HostCommandKind_HOST_COMMAND_KIND_TIMERS_ON, vocab.HostCmd("TIMERS_ON"), true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			inbox := &fakes.FakeInbox{PostResult: true}
			server, err := NewHostServer(inbox, "test-host")
			if err != nil {
				t.Fatal(err)
			}
			ack, err := server.Command(context.Background(), &df.HostCommand{HostToken: "test-host", Command: tc.command})
			if err != nil || !ack.GetOk() {
				t.Fatalf("ack=%v err=%v", ack, err)
			}
			event := inbox.Calls[0].Envelope.Event.(domain.HostCmd)
			if event.Cmd != tc.want {
				t.Fatalf("command=%s want=%s", event.Cmd, tc.want)
			}
			view := ProjectHost(domain.View{TurnTimersEnabled: tc.enabled})
			if view.GetTurnTimersEnabled() != tc.enabled {
				t.Fatal("policy omitted from host snapshot")
			}
		})
	}
}
