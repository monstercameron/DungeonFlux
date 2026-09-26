//go:build js && wasm

package host

import (
	"context"
	"strconv"

	v1 "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
	"github.com/monstercameron/GoWebComponents/v6/html"
	"github.com/monstercameron/GoWebComponents/v6/router"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

// Mount returns the host control screen for the shared shell router.
func Mount(endpoint string) router.Component {
	return func(_ router.Attrs) *router.Element {
		client, err := newHostClient(endpoint)
		if err != nil {
			return ui.CreateElement(errorView, err.Error())
		}
		return ui.CreateElement(hostView, hostViewProps{Client: client})
	}
}

type hostViewProps struct{ Client *hostClient }

func errorView(message string) ui.Node {
	return html.Main(html.Props{Class: "df-host"}, html.H1(html.Props{}, html.Text("DungeonFlux Host")), html.P(html.Props{}, html.Text(message)))
}

func hostView(props hostViewProps) ui.Node {
	state := ui.UseState(hostSnapshot{Status: "Connecting…"})
	status := ui.UseState("Ready")
	safeMode := ui.UseState(false)
	timersOn := ui.UseState(false)
	splatOn := ui.UseState(true)
	query := router.UseQuery()
	token := query.Get("t")
	if token == "" {
		token = query.Get("token")
	}
	ui.UseEffect(func() func() {
		ctx, cancel := context.WithCancel(context.Background())
		updates := props.Client.watch(ctx, token)
		go func() {
			for message := range updates {
				if message.GetState() != nil {
					state.Set(snapshotFromState(message.GetState()))
				}
			}
		}()
		return cancel
	})
	snapshot := state.Get()
	buttons := make([]ui.Node, 0, len(hostActions))
	for _, action := range hostActions {
		item := action
		buttons = append(buttons, ui.CreateElement(hostActionButton, hostActionButtonProps{Action: item, Click: func() {
			on := toggleState(snapshot, item, safeMode.Get(), timersOn.Get(), splatOn.Get())
			setToggleState(item, on, safeMode.Set, timersOn.Set, splatOn.Set)
			sendHostCommand(props.Client, token, item, on, status.Update)
		}}))
	}
	return html.Main(html.Props{Class: "df-host"},
		html.H1(html.Props{}, html.Text("DungeonFlux Host")),
		html.P(html.Props{Class: "df-host-status", Aria: map[string]string{"live": "polite"}}, html.Text(snapshot.Status+" · "+status.Get())),
		html.Div(html.Props{Class: "df-host-actions"}, buttons...),
		html.Section(html.Props{Class: "df-host-run"}, html.H2(html.Props{}, html.Text("Run status")), hostRunView(snapshot)),
		html.Section(html.Props{Class: "df-host-assets"}, html.H2(html.Props{}, html.Text("Asset slots")), assetSlots(snapshot.View)),
		html.Section(html.Props{Class: "df-host-log"}, html.H2(html.Props{}, html.Text("Log tail")), logTail(snapshot.View)),
	)
}

type hostActionButtonProps struct {
	Action hostAction
	Click  func()
}

func hostActionButton(props hostActionButtonProps) ui.Node {
	click := ui.UseEvent(func(event ui.Event) { event.PreventDefault(); props.Click() })
	return html.Button(html.Props{Type: "button", Class: "df-host-action", OnClick: click}, html.Text(props.Action.Label))
}

func sendHostCommand(client *hostClient, token string, action hostAction, on bool, update func(func(string) string)) {
	update(func(string) string { return "Sending " + action.Label + "…" })
	go func() {
		ack, err := client.command(context.Background(), commandForToggle(action, token, on))
		if err != nil {
			update(func(string) string { return "Error: " + err.Error() })
			return
		}
		message := "Accepted"
		if !ack.GetOk() {
			message = "Rejected: " + ack.GetReason()
		}
		update(func(string) string { return message })
	}()
}

func toggleState(snapshot hostSnapshot, action hostAction, safe, timers, splat bool) bool {
	switch action.Command {
	case v1.HostCommandKind_HOST_COMMAND_KIND_SAFE_MODE:
		return !safe
	case v1.HostCommandKind_HOST_COMMAND_KIND_TIMERS_OFF:
		return !timers
	case v1.HostCommandKind_HOST_COMMAND_KIND_SPLAT_OFF:
		return !splat
	default:
		return true
	}
}

func setToggleState(action hostAction, on bool, safe, timers, splat func(bool)) {
	switch action.Command {
	case v1.HostCommandKind_HOST_COMMAND_KIND_SAFE_MODE:
		safe(on)
	case v1.HostCommandKind_HOST_COMMAND_KIND_TIMERS_OFF:
		timers(on)
	case v1.HostCommandKind_HOST_COMMAND_KIND_SPLAT_OFF:
		splat(on)
	}
}

func hostRunView(snapshot hostSnapshot) ui.Node {
	if snapshot.View == nil {
		return html.P(html.Props{}, html.Text("No snapshot yet"))
	}
	view := snapshot.View
	return html.Div(html.Props{}, html.P(html.Props{}, html.Text("Mode: "+view.GetRunMode())), html.P(html.Props{}, html.Text("Next d20: "+strconv.FormatInt(int64(view.GetNextD20()), 10))), html.P(html.Props{}, html.Text("Combat cap: "+strconv.FormatInt(view.GetCombatCapRemainingMs(), 10)+" ms")))
}

func assetSlots(view interface{ GetAssetSlots() []*v1.AssetSlot }) ui.Node {
	if view == nil {
		return html.P(html.Props{}, html.Text("No assets"))
	}
	items := make([]ui.Node, 0, len(view.GetAssetSlots()))
	for _, slot := range view.GetAssetSlots() {
		items = append(items, html.Li(html.Props{}, html.Text(slot.GetName()+": "+slot.GetState())))
	}
	return html.Ul(html.Props{}, items...)
}

func logTail(view interface{ GetLogTail() []string }) ui.Node {
	if view == nil {
		return html.P(html.Props{}, html.Text("No logs"))
	}
	items := make([]ui.Node, 0, len(view.GetLogTail()))
	for _, line := range view.GetLogTail() {
		items = append(items, html.Li(html.Props{}, html.Text(line)))
	}
	return html.Ul(html.Props{}, items...)
}
