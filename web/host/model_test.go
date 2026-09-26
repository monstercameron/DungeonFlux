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
	want := []string{"Start", "Pause", "Resume", "Skip", "Reset", "Force d20 = 1", "Force d20 = 20"}
	if got := actionLabels(); !reflect.DeepEqual(got, want) {
		t.Fatalf("actionLabels() = %v, want %v", got, want)
	}
}
