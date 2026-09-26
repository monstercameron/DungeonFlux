package host

import (
	"reflect"
	"testing"

	df "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
)

func TestCommandFor_buildsAuthenticatedHostCommand(t *testing.T) {
	action := hostActions[5]
	got := commandFor(action, "host-secret")
	if got.GetHostToken() != "host-secret" || got.GetCommand() != df.HostCommandKind_HOST_COMMAND_KIND_FORCE_D20 || got.GetD20() != 1 {
		t.Fatalf("commandFor() = %#v", got)
	}
}

func TestCommandFor_carriesEachAction(t *testing.T) {
	for _, action := range hostActions {
		t.Run(action.Label, func(t *testing.T) {
			got := commandFor(action, "token")
			if got.GetHostToken() != "token" || got.GetCommand() != action.Command || got.GetD20() != action.D20 {
				t.Fatalf("commandFor() = %#v, want command %v and d20 %d", got, action.Command, action.D20)
			}
		})
	}
}

func TestActionLabels_returnsStageControlsInOrder(t *testing.T) {
	want := []string{"Start", "Pause", "Resume", "Skip", "Reset", "Force d20 = 1", "Force d20 = 20", "Safe Mode", "Turn timers", "Splat"}
	if got := actionLabels(); !reflect.DeepEqual(got, want) {
		t.Fatalf("actionLabels() = %v, want %v", got, want)
	}
}

func TestSnapshotFromState_projectsHostViewAndPause(t *testing.T) {
	state := &df.ScreenState{Phase: "combat", Paused: true, View: &df.ScreenState_Host{Host: &df.HostView{RunMode: "live", LogTail: []string{"warn"}}}}
	snapshot := snapshotFromState(state)
	if snapshot.View.GetLogTail()[0] != "warn" || snapshot.Status != "Paused" || !snapshot.Connected {
		t.Fatalf("snapshot = %+v", snapshot)
	}
}

func TestSnapshotFromState_handlesMissingState(t *testing.T) {
	snapshot := snapshotFromState(nil)
	if snapshot.Status != "Waiting for the room" || snapshot.Connected {
		t.Fatalf("snapshot = %+v", snapshot)
	}
}

func TestCommandForToggle_setsOnFlag(t *testing.T) {
	command := commandForToggle(hostActions[7], "token", true)
	if !command.GetOn() || command.GetHostToken() != "token" {
		t.Fatalf("command = %+v", command)
	}
}

func TestLinksFor_buildsCopyableTesterURLs(t *testing.T) {
	got := linksFor("https://dm.example/", "host token")
	want := testerLinks{DM: "https://dm.example/dm?t=host+token", Phone: "https://dm.example/p?t=host+token", Host: "https://dm.example/host?t=host+token"}
	if got != want {
		t.Fatalf("linksFor() = %+v, want %+v", got, want)
	}
}

func TestLinksFor_allowsTokenlessLocalLinks(t *testing.T) {
	got := linksFor("http://localhost:18146", "")
	if got.DM != "http://localhost:18146/dm" || got.Phone != "http://localhost:18146/p" || got.Host != "http://localhost:18146/host" {
		t.Fatalf("linksFor() = %+v", got)
	}
}
