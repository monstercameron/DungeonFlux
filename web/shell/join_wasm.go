//go:build js && wasm

package main

import (
	"context"
	"strings"
	"syscall/js"

	"github.com/monstercameron/DungeonFlux/web/phone"
	"github.com/monstercameron/GoWebComponents/v6/html"
	"github.com/monstercameron/GoWebComponents/v6/router"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

// JoinScreen returns the phone room-code and QR-link join component.
func JoinScreen(client *Client) router.Component {
	return func(_ router.Attrs) *router.Element {
		initialRoom := browserRoomCode()
		savedToken := browserSeatToken(initialRoom)
		room := ui.UseState(initialRoom)
		name := ui.UseState(browserPlayerName())
		locale := NewLocaleModel(BrowserLocales())
		model := ui.UseState(NewJoinModel(client, initialRoom))
		view := ui.UseState(JoinSnapshot{RoomCode: initialRoom, Phase: initialJoinPhase(initialRoom, savedToken), Locale: locale.Active()})
		ui.UseEffect(func() func() {
			if !hasSavedSeat(room.Get(), savedToken) {
				return func() {}
			}
			startPhoneJoin(model.Get(), savedToken, name.Get(), locale.Active(), view.Set)
			return func() {}
		}, client, initialRoom)
		join := ui.UseEvent(func() {
			current := model.Get()
			current.SetRoomCode(room.Get())
			current.SetPlayerName(name.Get())
			current.SetLocale(locale.Active())
			modelState := view.Get()
			modelState.Phase = JoinPending
			modelState.Locale = locale.Active()
			view.Set(modelState)
			startPhoneJoin(current, browserSeatToken(room.Get()), name.Get(), locale.Active(), view.Set)
			storeBrowserPlayerName(name.Get())
		})
		change := ui.UseEvent(func(event ui.InputEvent) {
			room.Set(event.GetValue())
			model.Get().SetRoomCode(event.GetValue())
		})
		nameChange := ui.UseEvent(func(event ui.InputEvent) {
			name.Set(event.GetValue())
			model.Get().SetPlayerName(event.GetValue())
		})
		state := view.Get()
		active := state.Locale
		if active == "" {
			active = locale.Active()
		}
		if state.Phase == JoinJoined {
			return ui.CreateElement(phone.Mount(phoneClientAdapter{client: client}, state.SeatToken, active))
		}
		message := state.Error
		if state.Phase == JoinPending {
			message = locale.T("shell.joining", nil)
		}
		roomCode := room.Get()
		roomHint := html.P(html.Props{Class: "df-join-hint"}, html.Text("Use the room code shown on the Dungeon Master screen."))
		if roomCode != "" {
			roomHint = html.P(html.Props{Class: "df-join-hint df-join-hint-link", Role: "status"}, html.Text("Room link recognized — your code is ready."))
		}
		invalid := state.Phase == JoinFailed
		return html.Main(html.Props{Class: "df-join", Role: "main", Style: joinPageStyle()},
			html.Div(html.Props{Class: "df-join-glow"}),
			html.Section(html.Props{Class: "df-join-card", Style: joinCardStyle()},
				html.Div(html.Props{Class: "df-join-topline", Style: map[string]string{"display": "flex", "justify-content": "space-between", "align-items": "center", "min-height": "32px"}}, html.Span(html.Props{Class: "df-join-mark", Style: map[string]string{"color": "#d9a441", "font-size": "24px"}}, html.Text("✦")), languageSwitcher(locale, view)),
				html.P(html.Props{Class: "df-join-kicker", Style: map[string]string{"margin": "26px 0 10px", "color": "#d9a441", "font-size": "11px", "letter-spacing": "2px", "font-weight": "700"}}, html.Text("DUNGEONFLUX · PLAYER TABLE")),
				html.H1(html.Props{Class: "df-join-title", Style: map[string]string{"margin": "0", "font-family": "Georgia, serif", "font-size": "clamp(30px, 8vw, 42px)", "line-height": "1.05", "color": "#efe6d2"}}, html.Text(locale.T("shell.join_title", nil))),
				html.P(html.Props{Class: "df-join-intro", Style: map[string]string{"margin": "18px 0 24px", "color": "#a89f8c", "font-size": "15px", "line-height": "1.5"}}, html.Text("Step into the story. No account, no app — just a room code.")),
				html.Div(html.Props{Class: "df-join-rule", Style: map[string]string{"height": "1px", "margin": "0 0 24px", "background": "rgba(217,164,65,.22)"}}),
				html.Div(html.Props{Class: "df-join-form", Style: map[string]string{"display": "flex", "flex-direction": "column", "gap": "8px", "width": "100%"}},
					html.Label(html.Props{For: "player-name", Class: "df-join-label", Style: joinLabelStyle()}, html.Text("Your name")),
					html.Input(html.Props{ID: "player-name", Class: "df-join-input", Style: joinInputStyle(), Name: "name", Value: name.Get(), Placeholder: "What should the table call you?", AutoComplete: "nickname", MaxLength: 40, OnInput: nameChange}),
					html.Label(html.Props{For: "room-code", Class: "df-join-label", Style: joinLabelStyle()}, html.Text(locale.T("ui.join.room_code", nil))),
					html.Input(html.Props{ID: "room-code", Class: "df-join-input df-join-code", Style: joinCodeStyle(), Name: "room", Value: roomCode, Placeholder: locale.T("shell.room_ph", nil), AutoComplete: "one-time-code", AutoFocus: roomCode == "", MaxLength: 12, Aria: map[string]string{"invalid": boolString(invalid)}, OnInput: change}),
					roomHint,
					html.Button(html.Props{Type: "button", Class: "df-join-button", Style: joinButtonStyle(), OnClick: join, Disabled: state.Phase == JoinPending || strings.TrimSpace(roomCode) == ""}, html.Text(locale.T("shell.join_table", nil))),
				),
				html.P(html.Props{Class: "df-join-status", Style: map[string]string{"min-height": "22px", "margin": "16px 0 0", "color": "#b3372f", "font-size": "13px", "line-height": "1.4"}, Role: statusRole(state), Aria: map[string]string{"live": "polite"}}, html.Text(message)),
				html.P(html.Props{Class: "df-join-foot", Style: map[string]string{"margin": "20px 0 0", "color": "#777467", "font-size": "12px", "line-height": "1.4"}}, html.Text("Your seat is saved on this device so you can reconnect.")),
			),
		)
	}
}

func startPhoneJoin(model *JoinModel, token, playerName, locale string, set func(JoinSnapshot)) {
	if model == nil {
		return
	}
	model.SetPlayerName(playerName)
	model.SetLocale(locale)
	roomCode := model.Snapshot().RoomCode
	result := model.StartJoin(context.Background(), token)
	go func() {
		state := model.ApplyJoin(<-result)
		if state.Phase == JoinJoined {
			storeBrowserSeatToken(roomCode, state.SeatToken)
		} else if strings.TrimSpace(token) != "" {
			clearBrowserSeatToken(roomCode)
		}
		set(state)
	}()
}

// languageSwitcher renders the join-screen language control. Choosing a
// language re-renders the join copy and travels in the next join request.
func languageSwitcher(locale *LocaleModel, view ui.State[JoinSnapshot]) ui.Node {
	options := make([]ui.Node, 0, len(locale.Options()))
	for _, option := range locale.Options() {
		tag := option
		label := "English"
		if tag == "es" {
			label = "Español"
		}
		options = append(options, html.Button(html.Props{Type: "button", Disabled: tag == locale.Active(), OnClick: ui.UseEvent(func() {
			locale.Set(tag)
			current := view.Get()
			current.Locale = tag
			view.Set(current)
		})}, html.Text(label)))
	}
	return html.Div(html.Props{Class: "df-join-locale"}, options...)
}

func browserRoomCode() string {
	value := js.Global().Get("URLSearchParams").New(js.Global().Get("location").Get("search")).Call("get", "room")
	if value.IsNull() || value.IsUndefined() {
		return ""
	}
	return value.String()
}

func statusRole(state JoinSnapshot) string {
	if state.Phase == JoinFailed {
		return "alert"
	}
	return "status"
}

func boolString(value bool) string {
	if value {
		return "true"
	}
	return "false"
}

func joinPageStyle() map[string]string {
	return map[string]string{
		"min-height": "100vh", "box-sizing": "border-box", "display": "grid", "place-items": "center",
		"padding": "24px", "position": "relative", "overflow": "hidden", "background": "radial-gradient(circle at 50% 0%, #252331 0%, #15151d 45%, #0e1016 100%)", "color": "#efe6d2", "font-family": "system-ui, sans-serif",
	}
}

func joinCardStyle() map[string]string {
	return map[string]string{
		"width": "100%", "max-width": "440px", "box-sizing": "border-box", "padding": "clamp(24px, 7vw, 46px)", "border": "1px solid rgba(217,164,65,.5)", "border-radius": "14px", "background": "linear-gradient(145deg, rgba(31,31,43,.97), rgba(19,20,29,.98))", "box-shadow": "0 24px 70px rgba(0,0,0,.45), inset 0 0 0 1px rgba(239,230,210,.04)",
	}
}

func joinLabelStyle() map[string]string {
	return map[string]string{"margin-top": "8px", "color": "#c8bda8", "font-size": "13px", "font-weight": "600"}
}

func joinInputStyle() map[string]string {
	return map[string]string{"width": "100%", "height": "50px", "box-sizing": "border-box", "padding": "0 14px", "border": "1px solid rgba(239,230,210,.22)", "border-radius": "8px", "background": "#11131b", "color": "#efe6d2", "font-size": "16px", "outline": "none"}
}

func joinCodeStyle() map[string]string {
	style := joinInputStyle()
	style["letter-spacing"] = "4px"
	style["font-weight"] = "700"
	return style
}

func joinButtonStyle() map[string]string {
	return map[string]string{"width": "100%", "height": "52px", "margin-top": "14px", "border": "0", "border-radius": "8px", "background": "#d9a441", "color": "#17130d", "font-size": "16px", "font-weight": "800", "cursor": "pointer"}
}
