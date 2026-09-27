//go:build js && wasm

package host

import (
	"context"
	"sync"
	"syscall/js"

	v1 "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
	"github.com/monstercameron/GoWebComponents/v6/css"
	"github.com/monstercameron/GoWebComponents/v6/html"
	"github.com/monstercameron/GoWebComponents/v6/router"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

// sharedHostClients keeps one connection per endpoint; route re-renders
// (asset loads) otherwise reconnect and drop the host Watch stream.
var sharedHostClients = struct {
	sync.Mutex
	byEndpoint map[string]*hostClient
}{byEndpoint: map[string]*hostClient{}}

func sharedHostClient(endpoint string) (*hostClient, error) {
	sharedHostClients.Lock()
	defer sharedHostClients.Unlock()
	if client, ok := sharedHostClients.byEndpoint[endpoint]; ok {
		return client, nil
	}
	client, err := newHostClient(endpoint)
	if err != nil {
		return nil, err
	}
	sharedHostClients.byEndpoint[endpoint] = client
	return client, nil
}

// Mount returns the host control screen for the shared shell router.
func Mount(endpoint string) router.Component {
	return func(_ router.Attrs) *router.Element {
		client, err := sharedHostClient(endpoint)
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
	css.Inject("dungeonflux-host", hostStyles)
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
	if token == "" {
		// The shell saves the boot token before the router drops the query.
		if storage := js.Global().Get("sessionStorage"); storage.Truthy() {
			if saved := storage.Call("getItem", "df-host-token"); saved.Truthy() {
				token = saved.String()
			}
		}
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
	primary, dice, toggles := make([]ui.Node, 0, 5), make([]ui.Node, 0, 2), make([]ui.Node, 0, 3)
	for _, action := range hostActions {
		item := action
		button := ui.CreateElement(hostActionButton, hostActionButtonProps{Action: item, Locale: locale, Click: func() {
			on := toggleState(snapshot, item, safeMode.Get(), timersOn.Get(), splatOn.Get())
			setToggleState(item, on, safeMode.Set, timersOn.Set, splatOn.Set)
			sendHostCommand(props.Client, token, locale, item, on, status.Update)
		}})
		switch {
		case item.Command == v1.HostCommandKind_HOST_COMMAND_KIND_FORCE_D20:
			dice = append(dice, button)
		case item.Toggle:
			toggles = append(toggles, button)
		default:
			primary = append(primary, button)
		}
	}
	links := linksFromState(browserOrigin(), token, query.Get("dm_token"), query.Get("room"), snapshot.State)
	return html.Main(html.Props{Class: "df-host"},
		html.Div(html.Props{Class: "df-host-hero"}, html.Div(html.Props{Class: "df-host-kicker"}, html.Text("CONTROL ROOM")), html.H1(html.Props{}, html.Text(HostTitle(locale))), html.P(html.Props{Class: "df-host-status", Aria: map[string]string{"live": "polite"}}, html.Span(html.Props{Class: "df-host-status-dot"}), html.Text(snapshot.Status+" · "+status.Get()))),
		html.Section(html.Props{Class: "df-host-controls"}, html.Div(html.Props{Class: "df-host-section-head"}, html.H2(html.Props{}, html.Text("Run controls")), html.Small(html.Props{}, html.Text("Safe, reversible steering"))), html.Div(html.Props{Class: "df-host-actions df-host-actions-primary"}, primary...), html.Div(html.Props{Class: "df-host-subactions"}, html.Div(html.Props{Class: "df-host-action-group"}, html.Small(html.Props{}, html.Text("Dice")), html.Div(html.Props{Class: "df-host-actions"}, dice...)), html.Div(html.Props{Class: "df-host-action-group"}, html.Small(html.Props{}, html.Text("Options")), html.Div(html.Props{Class: "df-host-actions"}, toggles...)))),
		testerLinksSection(links, locale),
		roomLocaleSection(props.Client, token, snapshot, state.Set, status.Update),
		html.Section(html.Props{Class: "df-host-run"}, html.H2(html.Props{}, html.Text(RunSection(locale))), hostRunView(snapshot)),
		html.Section(html.Props{Class: "df-host-assets"}, html.H2(html.Props{}, html.Text(AssetsSection(locale))), assetSlots(snapshot.View, locale)),
		html.Section(html.Props{Class: "df-host-log"}, html.H2(html.Props{}, html.Text(LogSection(locale))), logTail(snapshot.View, locale)),
	)
}

func browserOrigin() string {
	location := js.Global().Get("location")
	if !location.Truthy() {
		return ""
	}
	return location.Get("origin").String()
}

func testerLinksSection(links testerLinks, locale string) ui.Node {
	return html.Section(html.Props{Class: "df-host-links"}, html.Div(html.Props{Class: "df-host-section-head"}, html.H2(html.Props{}, html.Text(TesterLinksTitle(locale))), html.Small(html.Props{}, html.Text("Open a client on each tester device"))), linkRow("DM screen", links.DM, locale), linkRow("Player phone", links.Phone, locale), linkRow("Host controls", links.Host, locale))
}

func linkRow(label, value, locale string) ui.Node {
	return html.Div(html.Props{Class: "df-host-link-row"}, html.Label(html.Props{Class: "df-host-link-label"}, html.Text(label)), html.Div(html.Props{Class: "df-host-link-input"}, html.Input(html.Props{Type: "text", Value: value, ReadOnly: true, Title: value}), html.Button(html.Props{Type: "button", Class: "df-host-copy", OnClick: ui.UseEvent(func(ui.Event) { copyText(value) })}, html.Text(CopyLinkLabel(locale)))))
}

func copyText(value string) {
	navigator := js.Global().Get("navigator")
	if navigator.Truthy() && navigator.Get("clipboard").Truthy() {
		navigator.Get("clipboard").Call("writeText", value)
	}
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
	class := "df-host-action"
	if props.Action.Command == v1.HostCommandKind_HOST_COMMAND_KIND_RESET {
		class += " df-host-action-danger"
	}
	if props.Action.Command == v1.HostCommandKind_HOST_COMMAND_KIND_START {
		class += " df-host-action-primary"
	}
	return html.Button(html.Props{Type: "button", Class: class, OnClick: click}, html.Text(HostActionLabel(props.Locale, props.Action)))
}

const hostStyles = `
:root { color-scheme: dark; }
html, body { min-height: 100%; }
body { margin: 0; background: #0f1117; color: #efe6d2; font-family: Inter, ui-sans-serif, system-ui, sans-serif; }
button, input { font: inherit; }
.df-host { box-sizing: border-box; width: min(1180px, 100%); margin: 0 auto; padding: 40px 32px 64px; }
.df-host-hero { display: flex; align-items: end; justify-content: space-between; gap: 24px; margin-bottom: 28px; padding: 28px 32px; border: 1px solid #594a2c; border-radius: 14px; background: radial-gradient(circle at 85% 10%, #2a261d, #171a23 58%); box-shadow: inset 0 0 28px #0008, 0 18px 50px #0005; }
.df-host-kicker { color: #d9a441; font-size: 12px; font-weight: 800; letter-spacing: .22em; }
.df-host h1, .df-host h2 { font-family: Georgia, 'Times New Roman', serif; }
.df-host h1 { margin: 8px 0 0; font-size: clamp(34px, 5vw, 60px); letter-spacing: -.025em; }
.df-host h2 { margin: 0; font-size: 25px; }
.df-host-status { display: flex; align-items: center; gap: 10px; margin: 0; color: #b9b09d; font-size: 15px; white-space: nowrap; }
.df-host-status-dot { display: inline-block; width: 10px; height: 10px; border-radius: 50%; background: #3aa39a; box-shadow: 0 0 12px #3aa39a; }
.df-host-controls, .df-host-links, .df-host-locale, .df-host-run, .df-host-assets, .df-host-log { margin-top: 20px; padding: 24px; border: 1px solid #403b34; border-radius: 12px; background: #191c25; box-shadow: inset 0 0 24px #0005; }
.df-host-section-head { display: flex; align-items: baseline; justify-content: space-between; gap: 16px; margin-bottom: 18px; }
.df-host-section-head small, .df-host-action-group > small { color: #a89f8c; font-size: 12px; letter-spacing: .1em; text-transform: uppercase; }
.df-host-actions { display: grid; grid-template-columns: repeat(auto-fit, minmax(130px, 1fr)); gap: 12px; }
.df-host-actions-primary { grid-template-columns: repeat(5, 1fr); }
.df-host-action { min-height: 64px; padding: 12px 16px; border: 1px solid #625b4c; border-radius: 10px; background: #242936; color: #efe6d2; cursor: pointer; font-weight: 750; transition: transform .18s ease, border-color .18s ease, background .18s ease; }
.df-host-action:hover, .df-host-copy:hover { transform: translateY(-2px); border-color: #d9a441; background: #302f2b; }
.df-host-action:focus-visible, .df-host-copy:focus-visible, .df-host-link-input input:focus { outline: 3px solid #d9a441; outline-offset: 2px; }
.df-host-action-primary { background: #9d7128; border-color: #d9a441; color: #fff7e7; }
.df-host-action-danger { border-color: #8d403c; color: #f2b4a7; }
.df-host-subactions { display: grid; grid-template-columns: 1fr 1fr; gap: 22px; margin-top: 22px; }
.df-host-action-group > small { display: block; margin-bottom: 9px; }
.df-host-link-row { display: grid; grid-template-columns: 150px 1fr; align-items: center; gap: 14px; margin-top: 10px; }
.df-host-link-label { color: #d9a441; font-weight: 700; }
.df-host-link-input { display: flex; min-width: 0; gap: 8px; }
.df-host-link-input input { min-width: 0; flex: 1; border: 1px solid #4b4a45; border-radius: 8px; padding: 12px; background: #10131a; color: #d0c7b4; }
.df-host-copy { min-width: 76px; border: 1px solid #625b4c; border-radius: 8px; background: #242936; color: #efe6d2; cursor: pointer; font-weight: 700; transition: transform .18s ease, border-color .18s ease, background .18s ease; }
.df-host ul { margin: 10px 0 0; padding-left: 20px; color: #c8bfad; }
.df-host p { color: #c8bfad; }
@media (max-width: 700px) { .df-host { width: 100%; max-width: 100%; padding: 16px 14px 36px; } .df-host-hero { display: block; padding: 22px 20px; } .df-host-status { margin-top: 18px; white-space: normal; } .df-host-controls, .df-host-links, .df-host-locale, .df-host-run, .df-host-assets, .df-host-log { box-sizing: border-box; max-width: 100%; padding: 18px; } .df-host-section-head { display: block; } .df-host-section-head small { display: block; margin-top: 8px; } .df-host-actions, .df-host-action-group { min-width: 0; } .df-host-actions-primary { grid-template-columns: repeat(2, minmax(0, 1fr)); } .df-host-subactions { grid-template-columns: minmax(0, 1fr); gap: 16px; } .df-host-action-group > .df-host-actions { grid-template-columns: repeat(2, minmax(0, 1fr)); } .df-host-link-row { grid-template-columns: minmax(0, 1fr); gap: 7px; } .df-host-link-input { width: 100%; } .df-host-link-input input { width: 0; } .df-host-copy { min-height: 48px; } }
@media (prefers-reduced-motion: reduce) { .df-host-action, .df-host-copy { transition: none; } .df-host-action:hover, .df-host-copy:hover { transform: none; } }
`

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
	lines := runStatusLines(snapshot)
	nodes := make([]ui.Node, 0, len(lines))
	for _, line := range lines {
		nodes = append(nodes, html.P(html.Props{}, html.Text(line)))
	}
	return html.Div(html.Props{}, nodes...)
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
