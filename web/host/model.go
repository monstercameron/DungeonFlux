package main

import df "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"

type hostAction struct {
	Label   string
	Command df.HostCommandKind
	D20     int32
}

var hostActions = []hostAction{
	{Label: "Start", Command: df.HostCommandKind_HOST_COMMAND_KIND_START},
	{Label: "Pause", Command: df.HostCommandKind_HOST_COMMAND_KIND_PAUSE},
	{Label: "Resume", Command: df.HostCommandKind_HOST_COMMAND_KIND_RESUME},
	{Label: "Skip", Command: df.HostCommandKind_HOST_COMMAND_KIND_SKIP},
	{Label: "Reset", Command: df.HostCommandKind_HOST_COMMAND_KIND_RESET},
	{Label: "Force d20 = 1", Command: df.HostCommandKind_HOST_COMMAND_KIND_FORCE_D20, D20: 1},
	{Label: "Force d20 = 20", Command: df.HostCommandKind_HOST_COMMAND_KIND_FORCE_D20, D20: 20},
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
