package host

import df "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"

type hostAction struct {
	Label   string
	Command df.HostCommandKind
	D20     int32
	Toggle  bool
}

var hostActions = []hostAction{
	{Label: "Start", Command: df.HostCommandKind_HOST_COMMAND_KIND_START},
	{Label: "Pause", Command: df.HostCommandKind_HOST_COMMAND_KIND_PAUSE},
	{Label: "Resume", Command: df.HostCommandKind_HOST_COMMAND_KIND_RESUME},
	{Label: "Skip", Command: df.HostCommandKind_HOST_COMMAND_KIND_SKIP},
	{Label: "Reset", Command: df.HostCommandKind_HOST_COMMAND_KIND_RESET},
	{Label: "Force d20 = 1", Command: df.HostCommandKind_HOST_COMMAND_KIND_FORCE_D20, D20: 1},
	{Label: "Force d20 = 20", Command: df.HostCommandKind_HOST_COMMAND_KIND_FORCE_D20, D20: 20},
	{Label: "Safe Mode", Command: df.HostCommandKind_HOST_COMMAND_KIND_SAFE_MODE, Toggle: true},
	{Label: "Turn timers", Command: df.HostCommandKind_HOST_COMMAND_KIND_TIMERS_OFF, Toggle: true},
	{Label: "Splat", Command: df.HostCommandKind_HOST_COMMAND_KIND_SPLAT_OFF, Toggle: true},
}

func commandFor(action hostAction, token string) *df.HostCommand {
	return &df.HostCommand{HostToken: token, Command: action.Command, D20: action.D20}
}

func actionLabels() []string {
	labels := make([]string, 0, len(hostActions))
	for _, action := range hostActions {
		labels = append(labels, action.Label)
	}
	return labels
}

type hostSnapshot struct {
	State      *df.ScreenState
	View       *df.HostView
	Status     string
	SafeMode   bool
	TimersOn   bool
	SplatOn    bool
	Connected  bool
	Locale     string
	RoomLocale string
	Selector   RoomLocaleSelector
}

func snapshotFromState(state *df.ScreenState) hostSnapshot {
	snapshot := hostSnapshot{State: state, Status: T("en", "ui.host.waiting", nil), Locale: "en", RoomLocale: "en", Selector: NewRoomLocaleSelector("en")}
	if state == nil {
		return snapshot
	}
	if state.GetHost() != nil {
		snapshot.View = state.GetHost()
		snapshot.Locale = localeOrDefault(state.GetHost().GetLocale())
		snapshot.RoomLocale = localeOrDefault(state.GetHost().GetRoomLocale())
		snapshot.Selector = NewRoomLocaleSelector(snapshot.RoomLocale)
		snapshot.Status = state.GetHost().GetRunMode()
		if snapshot.Status == "" {
			snapshot.Status = state.GetPhase()
		}
	}
	if state.GetPaused() {
		snapshot.Status = T(snapshot.Locale, "ui.dm.paused", nil)
	}
	snapshot.Connected = true
	return snapshot
}

func commandForToggle(action hostAction, token string, on bool) *df.HostCommand {
	command := commandFor(action, token)
	command.On = on
	return command
}
