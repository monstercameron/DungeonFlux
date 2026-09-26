//go:build js && wasm

package host

import (
	"context"

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
	return html.Main(html.Props{Class: "df-host"}, html.H1(html.Props{}, html.Text(HostTitle("en"))), html.P(html.Props{}, html.Text(message)))
}

func hostView(props hostViewProps) ui.Node {
	state := ui.UseState(hostSnapshot{Status: ConnectingLine("en"), Locale: "en", RoomLocale: "en", Selector: NewRoomLocaleSelector("en")})
	status := ui.UseState(ReadyLine("en"))
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
	locale := snapshot.Locale
	if locale == "" {
		locale = "en"
	}
	buttons := make([]ui.Node, 0, len(hostActions))
	for _, action := range hostActions {
		item := action
		buttons = append(buttons, ui.CreateElement(hostActionButton, hostActionButtonProps{Action: item, Locale: locale, Click: func() {
			on := toggleState(snapshot, item, safeMode.Get(), timersOn.Get(), splatOn.Get())
			setToggleState(item, on, safeMode.Set, timersOn.Set, splatOn.Set)
			sendHostCommand(props.Client, token, locale, item, on, status.Update)
		}}))
	}
	return html.Main(html.Props{Class: "df-host"},
		html.H1(html.Props{}, html.Text(HostTitle(locale))),
		html.P(html.Props{Class: "df-host-status", Aria: map[string]string{"live": "polite"}}, html.Text(snapshot.Status+" · "+status.Get())),
		html.Div(html.Props{Class: "df-host-actions"}, buttons...),
		roomLocaleSection(props.Client, token, snapshot, state.Set, status.Update),
		html.Section(html.Props{Class: "df-host-run"}, html.H2(html.Props{}, html.Text(RunSection(locale))), hostRunView(snapshot)),
		html.Section(html.Props{Class: "df-host-assets"}, html.H2(html.Props{}, html.Text(AssetsSection(locale))), assetSlots(snapshot.View, locale)),
		html.Section(html.Props{Class: "df-host-log"}, html.H2(html.Props{}, html.Text(LogSection(locale))), logTail(snapshot.View, locale)),
	)
}

// roomLocaleSection renders the room-language selector. Choosing a language
// sends a room-locale command so later joins default to it.
func roomLocaleSection(client *hostClient, token string, snapshot hostSnapshot, setSnapshot func(hostSnapshot), setStatus func(func(string) string)) ui.Node {
	locale := snapshot.Locale
	if locale == "" {
		locale = "en"
	}
	selector := snapshot.Selector
	options := make([]ui.Node, 0, len(selector.Options))
	for _, option := range selector.Options {
		tag := option
		options = append(options, html.Button(html.Props{Type: "button", Disabled: tag == selector.Selected, OnClick: ui.UseEvent(func() {
			next := snapshot.Selector
			next.Select(tag)
			updated := snapshot
			updated.Selector = next
			updated.RoomLocale = next.Selected
			setSnapshot(updated)
			setStatus(func(string) string { return SendingLine(locale, OptionLabel(tag)) })
			go func() {
				ack, err := client.command(context.Background(), &v1.HostCommand{HostToken: token, Command: v1.HostCommandKind_HOST_COMMAND_KIND_ROOM_LOCALE, Locale: tag})
				if err != nil {
					setStatus(func(string) string { return ErrorLine(locale, err.Error()) })
					return
				}
				if !ack.GetOk() {
					setStatus(func(string) string { return RejectedLine(locale, ack.GetReason()) })
					return
				}
				setStatus(func(string) string { return AcceptedLine(locale) })
			}()
		})}, html.Text(OptionLabel(tag))))
	}
	return html.Section(html.Props{Class: "df-host-locale"},
		html.H2(html.Props{}, html.Text(RoomLocaleLabel(locale))),
		html.Div(html.Props{Class: "df-host-locale-options"}, options...),
	)
}

type hostActionButtonProps struct {
	Action hostAction
	Locale string
	Click  func()
}

func hostActionButton(props hostActionButtonProps) ui.Node {
	click := ui.UseEvent(func(event ui.Event) { event.PreventDefault(); props.Click() })
	return html.Button(html.Props{Type: "button", Class: "df-host-action", OnClick: click}, html.Text(HostActionLabel(props.Locale, props.Action)))
}

func sendHostCommand(client *hostClient, token, locale string, action hostAction, on bool, update func(func(string) string)) {
	if locale == "" {
		locale = "en"
	}
	label := HostActionLabel(locale, action)
	update(func(string) string { return SendingLine(locale, label) })
	go func() {
		request := commandForToggle(action, token, on)
		request.Locale = locale
		ack, err := client.command(context.Background(), request)
		if err != nil {
			update(func(string) string { return ErrorLine(locale, err.Error()) })
			return
		}
		message := AcceptedLine(locale)
		if !ack.GetOk() {
			message = RejectedLine(locale, ack.GetReason())
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
	locale := snapshot.Locale
	if locale == "" {
		locale = "en"
	}
	if snapshot.View == nil {
		return html.P(html.Props{}, html.Text(NoSnapshot(locale)))
	}
	view := snapshot.View
	return html.Div(html.Props{}, html.P(html.Props{}, html.Text(ModeLine(locale, view.GetRunMode()))), html.P(html.Props{}, html.Text(NextD20Line(locale, view.GetNextD20()))), html.P(html.Props{}, html.Text(CombatCapLine(locale, view.GetCombatCapRemainingMs()))))
}

func assetSlots(view interface{ GetAssetSlots() []*v1.AssetSlot }, locale string) ui.Node {
	if locale == "" {
		locale = "en"
	}
	if view == nil {
		return html.P(html.Props{}, html.Text(NoAssets(locale)))
	}
	items := make([]ui.Node, 0, len(view.GetAssetSlots()))
	for _, slot := range view.GetAssetSlots() {
		items = append(items, html.Li(html.Props{}, html.Text(slot.GetName()+": "+slot.GetState())))
	}
	return html.Ul(html.Props{}, items...)
}

func logTail(view interface{ GetLogTail() []string }, locale string) ui.Node {
	if locale == "" {
		locale = "en"
	}
	if view == nil {
		return html.P(html.Props{}, html.Text(NoLogs(locale)))
	}
	items := make([]ui.Node, 0, len(view.GetLogTail()))
	for _, line := range view.GetLogTail() {
		items = append(items, html.Li(html.Props{}, html.Text(line)))
	}
	return html.Ul(html.Props{}, items...)
}
