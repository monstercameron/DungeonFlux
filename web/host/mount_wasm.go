//go:build js && wasm

package host

import (
	"context"
	"github.com/monstercameron/DungeonFlux/web/shell/watch"
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
	watch.InstallTriggers(&client.recovery)
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
	confirmReset := ui.UseState(false)
	snapshot := state.Get()
	locale := snapshot.Locale
	if locale == "" {
		locale = "en"
	}
	primary, dice, toggles := make([]ui.Node, 0, 5), make([]ui.Node, 0, 2), make([]ui.Node, 0, 3)
	for _, action := range hostActions {
		item := action
		if item.Command == v1.HostCommandKind_HOST_COMMAND_KIND_RESET {
			// Reset asks for confirmation instead of firing immediately
			// (§0.13: a stray tap should never wipe a live run).
			primary = append(primary, ui.CreateElement(hostActionButton, hostActionButtonProps{
				Action: item, Locale: locale, Phase: snapshot.Phase, Paused: snapshot.Paused,
				Click: func() { confirmReset.Set(true) },
			}))
			continue
		}
		button := ui.CreateElement(hostActionButton, hostActionButtonProps{Action: item, Locale: locale, Phase: snapshot.Phase, Paused: snapshot.Paused, TimersOn: snapshot.TimersOn, SplatOn: snapshot.SplatOn, SplatAvailable: snapshot.SplatAvailable, Connected: snapshot.Connected, Click: func() {
			on := toggleState(snapshot, item, safeMode.Get(), snapshot.TimersOn, snapshot.SplatOn)
			setToggleState(item, on, safeMode.Set)
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
		html.Div(html.Props{Class: "df-host-hero"},
			html.Div(html.Props{Class: "df-host-hero-title"}, html.Div(html.Props{Class: "df-host-kicker"}, html.Text(HostKicker(locale))), html.H1(html.Props{}, html.Text(HostTitle(locale)))),
			html.P(html.Props{Class: "df-host-status", Aria: map[string]string{"live": "polite"}}, html.Span(html.Props{Class: "df-host-status-dot"}), html.Text(snapshot.Status+" · "+status.Get())),
		),
		ui.CreateElement(statusStrip, statusStripProps{Snapshot: snapshot, TimersOn: snapshot.TimersOn, Locale: locale}),
		html.Section(html.Props{Class: "df-host-controls"},
			html.Div(html.Props{Class: "df-host-section-head"}, html.H2(html.Props{}, html.Text(ControlsHeading(locale))), html.Small(html.Props{}, html.Text(ControlsHint(locale)))),
			html.Div(html.Props{Class: "df-host-actions df-host-actions-primary"}, primary...),
			resetConfirmCard(confirmReset.Get(), locale, func() {
				confirmReset.Set(false)
				sendHostCommand(props.Client, token, locale, hostActions[4], true, status.Update)
			}, func() { confirmReset.Set(false) }),
			html.Div(html.Props{Class: "df-host-section-head df-host-section-head-tools"}, html.H2(html.Props{}, html.Text(StageToolsHeading(locale)))),
			html.Div(html.Props{Class: "df-host-subactions"},
				html.Div(html.Props{Class: "df-host-action-group"}, html.Small(html.Props{}, html.Text(StageToolsDice(locale))), html.Div(html.Props{Class: "df-host-actions"}, dice...)),
				html.Div(html.Props{Class: "df-host-action-group"}, html.Small(html.Props{}, html.Text(StageToolsFlags(locale))), html.Div(html.Props{Class: "df-host-actions"}, toggles...)),
			),
		),
		testerLinksSection(links, locale),
		roomLocaleSection(props.Client, token, snapshot, state.Set, status.Update),
		html.Section(html.Props{Class: "df-host-run"}, html.H2(html.Props{}, html.Text(RunSection(locale))), hostRunView(snapshot)),
		html.Section(html.Props{Class: "df-host-assets"}, html.H2(html.Props{}, html.Text(AssetsSection(locale))), assetSlots(snapshot.View, locale)),
		html.Section(html.Props{Class: "df-host-log"}, html.H2(html.Props{}, html.Text(LogSection(locale))), logTail(snapshot.View, locale)),
	)
}

// resetConfirmCard renders the two-step Reset confirmation inline under the
// primary controls. It renders nothing when not open, so it never disturbs
// layout until the host actually asks to reset.
func resetConfirmCard(open bool, locale string, confirm, cancel func()) ui.Node {
	if !open {
		return html.Div(html.Props{})
	}
	return html.Div(html.Props{Class: "df-host-confirm", Role: "alertdialog"},
		html.P(html.Props{Class: "df-host-confirm-title"}, html.Text(ResetConfirmTitle(locale))),
		html.P(html.Props{Class: "df-host-confirm-body"}, html.Text(ResetConfirmBody(locale))),
		html.Div(html.Props{Class: "df-host-confirm-actions"},
			html.Button(html.Props{Type: "button", Class: "df-host-action df-host-action-danger", OnClick: ui.UseEvent(func(ui.Event) { confirm() })}, html.Text(HostActionLabel(locale, hostActions[4]))),
			html.Button(html.Props{Type: "button", Class: "df-host-action", OnClick: ui.UseEvent(func(ui.Event) { cancel() })}, html.Text(ResetConfirmCancel(locale))),
		),
	)
}

type statusStripProps struct {
	Snapshot hostSnapshot
	TimersOn bool
	Locale   string
}

// statusStrip is the presenter's at-a-glance summary: phase, whose turn,
// seats connected, timers on/off, and any failures seen in the log tail.
func statusStrip(props statusStripProps) ui.Node {
	snapshot, locale := props.Snapshot, props.Locale
	turn := snapshot.TurnSeat
	if turn == "" {
		turn = snapshot.Spotlight
	}
	turnValue := turn
	if turnValue == "" {
		turnValue = StatusTurnNone(locale)
	}
	timerPill := statusPill("df-host-pill-info", StatusTimersOff(locale))
	if props.TimersOn {
		timerPill = statusPill("df-host-pill-info", StatusTimersOn(locale))
	}
	failurePill := statusPill("df-host-pill-ok", StatusAllClear(locale))
	if snapshot.Failures > 0 {
		failurePill = statusPill("df-host-pill-warn", StatusFailures(locale, snapshot.Failures))
	}
	pausedPill := ui.Node(nil)
	if snapshot.Paused {
		pausedPill = statusPill("df-host-pill-warn", StatusPaused(locale))
	}
	nodes := []ui.Node{
		statusPillLabeled(StatusPhaseLabel(locale), humanizePhase(snapshot.Phase)),
		statusPillLabeled(StatusTurnLabel(locale), turnValue),
		statusPillLabeled(StatusSeatsLabel(locale), StatusSeatsValue(locale, snapshot.SeatsJoined, snapshot.SeatsTotal)),
		timerPill,
		failurePill,
	}
	if pausedPill != nil {
		nodes = append(nodes, pausedPill)
	}
	if isLobbyPhase(snapshot.Phase) {
		nodes = append(nodes, statusPill("df-host-pill-ok", lobbyReadiness(snapshot)))
	}
	return html.Div(html.Props{Class: "df-host-status-strip", Aria: map[string]string{"live": "polite"}}, nodes...)
}

func statusPill(tone, text string) ui.Node {
	return html.Div(html.Props{Class: "df-host-pill " + tone}, html.Text(text))
}

func statusPillLabeled(label, value string) ui.Node {
	if value == "" {
		value = "—"
	}
	return html.Div(html.Props{Class: "df-host-pill df-host-pill-info"}, html.Span(html.Props{Class: "df-host-pill-label"}, html.Text(label)), html.Text(value))
}

func browserOrigin() string {
	location := js.Global().Get("location")
	if !location.Truthy() {
		return ""
	}
	return location.Get("origin").String()
}

func testerLinksSection(links testerLinks, locale string) ui.Node {
	return html.Section(html.Props{Class: "df-host-links"},
		html.Div(html.Props{Class: "df-host-section-head"}, html.H2(html.Props{}, html.Text(TesterLinksTitle(locale))), html.Small(html.Props{}, html.Text(LinksHint(locale)))),
		ui.CreateElement(linkRow, linkRowProps{Label: LinkLabelDM(locale), Value: links.DM, Locale: locale}),
		ui.CreateElement(linkRow, linkRowProps{Label: LinkLabelPhone(locale), Value: links.Phone, Locale: locale}),
		ui.CreateElement(linkRow, linkRowProps{Label: LinkLabelHost(locale), Value: links.Host, Locale: locale}),
	)
}

type linkRowProps struct {
	Label  string
	Value  string
	Locale string
}

// linkRow renders one copyable tester link. A link carrying a host or DM
// token (identified by maskLinkToken) is masked by default; a hold-to-reveal
// button shows it only while pressed, so the token is never left visible in
// plain text on the presenter's screen (critic HOST: host token shown in
// plain text).
func linkRow(props linkRowProps) ui.Node {
	revealed := ui.UseState(false)
	masked, hasToken := maskLinkToken(props.Value)
	display := props.Value
	title := props.Value
	if hasToken && !revealed.Get() {
		display = masked
		title = TokenHiddenTitle(props.Locale)
	}
	show := ui.UseEvent(func(ui.Event) { revealed.Set(true) })
	hide := ui.UseEvent(func(ui.Event) { revealed.Set(false) })
	reveal := html.Span(html.Props{})
	if hasToken {
		reveal = html.Button(html.Props{
			Type: "button", Class: "df-host-reveal",
			OnPointerDown: show, OnPointerUp: hide, OnMouseLeave: hide,
			OnTouchStart: show, OnTouchEnd: hide,
		}, html.Text(TokenRevealLabel(props.Locale)))
	}
	return html.Div(html.Props{Class: "df-host-link-row"},
		html.Label(html.Props{Class: "df-host-link-label"}, html.Text(props.Label)),
		html.Div(html.Props{Class: "df-host-link-input"},
			html.Input(html.Props{Type: "text", Value: display, ReadOnly: true, Title: title}),
			reveal,
			html.Button(html.Props{Type: "button", Class: "df-host-copy", OnClick: ui.UseEvent(func(ui.Event) { copyText(props.Value) })}, html.Text(CopyLinkLabel(props.Locale))),
		),
	)
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
		active := tag == selector.Selected
		class := "df-host-locale-option"
		pressed := "false"
		if active {
			class += " df-host-locale-option-active"
			pressed = "true"
		}
		options = append(options, html.Button(html.Props{Type: "button", Class: class, Aria: map[string]string{"pressed": pressed}, OnClick: ui.UseEvent(func() {
			if active {
				return
			}
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
	Action         hostAction
	Locale         string
	Phase          string
	Paused         bool
	TimersOn       bool
	SplatOn        bool
	SplatAvailable bool
	Connected      bool
	Click          func()
}

// hostActionButton renders one stage-control button, styled and enabled by
// the current phase so the presenter's eye always finds the next safe
// action: Start only matters in the lobby, Pause/Resume/Skip matter during
// play, and Reset is always available but never the loudest button on
// screen (§0.13 stage runbook; critic HOST: Start stays highlighted during
// combat).
func hostActionButton(props hostActionButtonProps) ui.Node {
	click := ui.UseEvent(func(event ui.Event) { event.PreventDefault(); props.Click() })
	class := "df-host-action"
	disabled := false
	switch props.Action.Command {
	case v1.HostCommandKind_HOST_COMMAND_KIND_RESET:
		class += " df-host-action-danger"
	case v1.HostCommandKind_HOST_COMMAND_KIND_START:
		disabled = !isLobbyPhase(props.Phase)
		if isLobbyPhase(props.Phase) {
			class += " df-host-action-primary"
		}
	case v1.HostCommandKind_HOST_COMMAND_KIND_PAUSE:
		disabled = props.Paused || !isPlayPhase(props.Phase)
		if !disabled {
			class += " df-host-action-primary"
		}
	case v1.HostCommandKind_HOST_COMMAND_KIND_RESUME:
		disabled = !props.Paused
		if !disabled {
			class += " df-host-action-primary"
		}
	case v1.HostCommandKind_HOST_COMMAND_KIND_SKIP:
		if isPlayPhase(props.Phase) {
			class += " df-host-action-accent"
		}
	}
	aria := map[string]string{}
	if props.Action.Command == v1.HostCommandKind_HOST_COMMAND_KIND_TIMERS_OFF {
		aria["pressed"] = "false"
		disabled = !props.Connected
		if props.TimersOn {
			aria["pressed"] = "true"
			class += " df-host-action-primary"
		}
	}
	if props.Action.Command == v1.HostCommandKind_HOST_COMMAND_KIND_SPLAT_OFF {
		aria["pressed"] = "false"
		disabled = !props.Connected || !props.SplatAvailable
		if props.SplatOn {
			aria["pressed"] = "true"
			class += " df-host-action-primary"
		}
	}
	return html.Button(html.Props{Type: "button", Class: class, Aria: aria, Disabled: disabled, OnClick: click}, html.Text(HostActionLabel(props.Locale, props.Action)))
}

const hostStyles = `
:root { color-scheme: dark; }
html, body { min-height: 100%; }
body { margin: 0; background: #0f1117; color: #efe6d2; font-family: Inter, ui-sans-serif, system-ui, sans-serif; }
button, input { font: inherit; }
.df-host { box-sizing: border-box; width: min(1180px, 100%); margin: 0 auto; padding: 40px 32px 64px; }
.df-host-hero { display: flex; align-items: flex-end; justify-content: space-between; flex-wrap: wrap; gap: 24px; margin-bottom: 20px; padding: 28px 32px; border: 1px solid #594a2c; border-radius: 14px; background: radial-gradient(circle at 85% 10%, #2a261d, #171a23 58%); box-shadow: inset 0 0 28px #0008, 0 18px 50px #0005; }
.df-host-hero-title { display: flex; flex-direction: column; align-items: flex-start; min-width: 0; }
.df-host-kicker { color: #d9a441; font-size: 12px; font-weight: 800; letter-spacing: .22em; }
.df-host h1, .df-host h2 { font-family: Georgia, 'Times New Roman', serif; }
.df-host h1 { margin: 8px 0 0; font-size: clamp(34px, 5vw, 60px); letter-spacing: -.025em; line-height: 1.05; }
.df-host h2 { margin: 0; font-size: 25px; }
.df-host-status { display: flex; align-items: center; gap: 10px; margin: 0; color: #b9b09d; font-size: 15px; white-space: nowrap; }
.df-host-status-dot { display: inline-block; width: 10px; height: 10px; border-radius: 50%; background: #3aa39a; box-shadow: 0 0 12px #3aa39a; }
.df-host-status-strip { display: flex; flex-wrap: wrap; gap: 10px; margin-bottom: 20px; }
.df-host-pill { display: flex; align-items: center; gap: 8px; min-height: 44px; padding: 8px 16px; border-radius: 999px; border: 1px solid #403b34; background: #191c25; color: #efe6d2; font-size: 14px; font-weight: 650; white-space: nowrap; }
.df-host-pill-label { color: #a89f8c; font-size: 11px; font-weight: 800; letter-spacing: .08em; text-transform: uppercase; }
.df-host-pill-ok { border-color: #3aa39a; color: #bfe9e3; }
.df-host-pill-warn { border-color: #b3372f; color: #f2b4a7; }
.df-host-pill-info { border-color: #625b4c; }
.df-host-controls, .df-host-links, .df-host-locale, .df-host-run, .df-host-assets, .df-host-log { margin-top: 20px; padding: 24px; border: 1px solid #403b34; border-radius: 12px; background: #191c25; box-shadow: inset 0 0 24px #0005; }
.df-host-section-head { display: flex; align-items: baseline; justify-content: space-between; gap: 16px; margin-bottom: 18px; }
.df-host-section-head-tools { margin-top: 22px; }
.df-host-section-head small, .df-host-action-group > small { color: #a89f8c; font-size: 12px; letter-spacing: .1em; text-transform: uppercase; }
.df-host-actions { display: grid; grid-template-columns: repeat(auto-fit, minmax(130px, 1fr)); gap: 12px; }
.df-host-actions-primary { grid-template-columns: repeat(5, 1fr); }
.df-host-action { min-height: 48px; padding: 12px 16px; border: 1px solid #625b4c; border-radius: 10px; background: #242936; color: #efe6d2; cursor: pointer; font-weight: 750; transition: transform .18s ease, border-color .18s ease, background .18s ease, opacity .18s ease; }
.df-host-actions-primary .df-host-action { min-height: 64px; }
.df-host-action:hover, .df-host-copy:hover, .df-host-reveal:hover { transform: translateY(-2px); border-color: #d9a441; background: #302f2b; }
.df-host-action:focus-visible, .df-host-copy:focus-visible, .df-host-reveal:focus-visible, .df-host-locale-option:focus-visible, .df-host-link-input input:focus { outline: 3px solid #d9a441; outline-offset: 2px; }
.df-host-action-primary { background: #9d7128; border-color: #d9a441; color: #fff7e7; }
.df-host-action-accent { border-color: #3aa39a; color: #bfe9e3; box-shadow: inset 0 0 0 1px #3aa39a55; }
.df-host-action-danger { border-color: #8d403c; color: #f2b4a7; }
.df-host-action:disabled { opacity: .4; cursor: not-allowed; transform: none; background: #1c2028; border-color: #4a463c; color: #8a8272; }
.df-host-action:disabled:hover { transform: none; border-color: #4a463c; background: #1c2028; }
.df-host-confirm { margin-top: 16px; padding: 18px 20px; border: 1px solid #8d403c; border-radius: 12px; background: #241716; }
.df-host-confirm-title { margin: 0 0 6px; color: #f2b4a7; font-weight: 800; font-size: 17px; }
.df-host-confirm-body { margin: 0 0 14px; color: #d8b9b2; }
.df-host-confirm-actions { display: flex; gap: 12px; flex-wrap: wrap; }
.df-host-subactions { display: grid; grid-template-columns: 1fr 1fr; gap: 22px; margin-top: 22px; }
.df-host-action-group > small { display: block; margin-bottom: 9px; }
.df-host-locale-options { display: flex; gap: 10px; flex-wrap: wrap; }
.df-host-locale-option { min-height: 44px; padding: 10px 20px; border-radius: 999px; border: 1px solid #625b4c; background: transparent; color: #c8bfad; cursor: pointer; font-weight: 700; transition: transform .18s ease, border-color .18s ease, background .18s ease, color .18s ease; }
.df-host-locale-option:hover { border-color: #d9a441; color: #efe6d2; }
.df-host-locale-option-active { background: #9d7128; border-color: #d9a441; color: #fff7e7; cursor: default; }
.df-host-locale-option-active:hover { transform: none; }
.df-host-link-row { display: grid; grid-template-columns: 150px 1fr; align-items: center; gap: 14px; margin-top: 10px; }
.df-host-link-label { color: #d9a441; font-weight: 700; }
.df-host-link-input { display: flex; min-width: 0; gap: 8px; }
.df-host-link-input input { min-width: 0; flex: 1; border: 1px solid #4b4a45; border-radius: 8px; padding: 12px; background: #10131a; color: #d0c7b4; font-variant-numeric: tabular-nums; letter-spacing: .01em; }
.df-host-copy, .df-host-reveal { min-width: 76px; min-height: 44px; border: 1px solid #625b4c; border-radius: 8px; background: #242936; color: #efe6d2; cursor: pointer; font-weight: 700; transition: transform .18s ease, border-color .18s ease, background .18s ease; }
.df-host-reveal { min-width: 96px; user-select: none; -webkit-user-select: none; touch-action: none; }
.df-host ul { margin: 10px 0 0; padding-left: 20px; color: #c8bfad; }
.df-host p { color: #c8bfad; }
@media (max-width: 700px) { .df-host { width: 100%; max-width: 100%; padding: 16px 14px 36px; } .df-host-hero { display: block; padding: 22px 20px; } .df-host-status { margin-top: 18px; white-space: normal; } .df-host-status-strip { margin-top: -6px; } .df-host-controls, .df-host-links, .df-host-locale, .df-host-run, .df-host-assets, .df-host-log { box-sizing: border-box; max-width: 100%; padding: 18px; } .df-host-section-head { display: block; } .df-host-section-head small { display: block; margin-top: 8px; } .df-host-actions, .df-host-action-group { min-width: 0; } .df-host-actions-primary { grid-template-columns: repeat(2, minmax(0, 1fr)); } .df-host-subactions { grid-template-columns: minmax(0, 1fr); gap: 16px; } .df-host-action-group > .df-host-actions { grid-template-columns: repeat(2, minmax(0, 1fr)); } .df-host-link-row { grid-template-columns: minmax(0, 1fr); gap: 7px; } .df-host-link-input { width: 100%; flex-wrap: wrap; } .df-host-link-input input { width: 0; flex-basis: 100%; } .df-host-copy, .df-host-reveal { min-height: 48px; } }
@media (prefers-reduced-motion: reduce) { .df-host-action, .df-host-copy, .df-host-reveal, .df-host-locale-option { transition: none; } .df-host-action:hover, .df-host-copy:hover, .df-host-reveal:hover { transform: none; } }
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

func setToggleState(action hostAction, on bool, safe func(bool)) {
	switch action.Command {
	case v1.HostCommandKind_HOST_COMMAND_KIND_SAFE_MODE:
		safe(on)
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
