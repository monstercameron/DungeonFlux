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
		roomHint := html.P(html.Props{Class: "df-join-hint", Style: joinHintStyle()}, html.Text("Use the room code shown on the Dungeon Master screen."))
		if roomCode != "" {
			roomHint = html.P(html.Props{Class: "df-join-hint df-join-hint-link", Role: "status", Style: joinHintStyle()}, html.Text("Room link recognized - your code is ready."))
		}
		invalid := state.Phase == JoinFailed
		frame := phone.NewFrameModel("Player", active)
		frame.Title = "Join the table"
		frame.Connection = phone.ConnectionConnecting
		content := html.Section(html.Props{Class: "df-join", Role: "main", Style: joinPageStyle()},
			html.Section(html.Props{Class: "df-join-card", Style: joinCardStyle()},
				html.Div(html.Props{Class: "df-join-topline", Style: map[string]string{"display": "flex", "justify-content": "space-between", "align-items": "center", "min-height": "32px"}}, html.Span(html.Props{Class: "df-join-mark", Style: map[string]string{"color": "#d9a441", "font-size": "25px"}}, html.Text("✦")), languageSwitcher(locale, view)),
				html.P(html.Props{Class: "df-join-kicker", Style: map[string]string{"margin": "17px 0 7px", "color": "#d9a441", "font-size": "10px", "letter-spacing": ".2em", "font-weight": "700", "text-align": "center"}}, html.Text("DUNGEONFLUX · PLAYER TABLE")),
				html.H1(html.Props{Class: "df-join-title", Style: map[string]string{"margin": "0", "font-family": "Cormorant Garamond, Cinzel, Georgia, serif", "font-size": "32px", "line-height": "1.05", "color": "#efe6d2", "text-align": "center"}}, html.Text(locale.T("shell.join_title", nil))),
				html.P(html.Props{Class: "df-join-intro", Style: map[string]string{"margin": "8px 0 17px", "color": "#a89f8c", "font-family": "Cormorant Garamond, Georgia, serif", "font-size": "17px", "line-height": "1.3", "text-align": "center"}}, html.Text("Step into the story. No account, no app — just a room code.")),
				html.Div(html.Props{Class: "df-join-rule", Style: map[string]string{"height": "1px", "margin": "0 0 13px", "background": "linear-gradient(90deg, transparent, rgba(217,164,65,.68), transparent)"}}),
				html.Div(html.Props{Class: "df-join-form", Style: map[string]string{"display": "flex", "flex-direction": "column", "gap": "7px", "width": "100%"}},
					html.Label(html.Props{For: "player-name", Class: "df-join-label", Style: joinLabelStyle()}, html.Text("Your name")),
					html.Input(html.Props{ID: "player-name", Class: "df-join-input", Style: joinInputStyle(), Name: "name", Value: name.Get(), Placeholder: "What should the table call you?", AutoComplete: "nickname", MaxLength: 40, OnInput: nameChange}),
					html.Label(html.Props{For: "room-code", Class: "df-join-label", Style: joinLabelStyle()}, html.Text(locale.T("ui.join.room_code", nil))),
					html.Input(html.Props{ID: "room-code", Class: "df-join-input df-join-code", Style: joinCodeStyle(), Name: "room", Value: roomCode, Placeholder: locale.T("shell.room_ph", nil), AutoComplete: "one-time-code", AutoFocus: roomCode == "", MaxLength: 12, Aria: map[string]string{"invalid": boolString(invalid)}, OnInput: change}),
					roomHint,
					html.Button(html.Props{Type: "button", Class: "df-join-button", Style: joinButtonStyle(), OnClick: join, Disabled: state.Phase == JoinPending || strings.TrimSpace(roomCode) == ""}, html.Text(locale.T("shell.join_table", nil))),
				),
				html.P(html.Props{Class: "df-join-status", Style: map[string]string{"min-height": "22px", "margin": "11px 0 0", "color": "#b3372f", "font-size": "12px", "line-height": "1.35", "text-align": "center"}, Role: statusRole(state), Aria: map[string]string{"live": "polite"}}, html.Text(message)),
				html.P(html.Props{Class: "df-join-foot", Style: map[string]string{"margin": "12px 0 0", "color": "#777467", "font-size": "10px", "line-height": "1.35", "text-align": "center"}}, html.Text("Your seat is saved on this device so you can reconnect.")),
			),
		)
		return ui.CreateElement(phone.PhoneFrame(frame, content, nil, nil))
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
		}), Style: map[string]string{"min-height": "32px", "padding": "4px 8px", "border": "1px solid rgba(217,164,65,.35)", "border-radius": "6px", "background": "rgba(15,17,23,.62)", "color": "#a89f8c", "font-size": "10px"}}, html.Text(label)))
	}
	return html.Div(html.Props{Class: "df-join-locale", Style: map[string]string{"display": "flex", "gap": "4px"}}, options...)
}

func browserRoomCode() string {
	value := js.Global().Get("URLSearchParams").New(js.Global().Get("location").Get("search")).Call("get", "room")
	if !value.IsNull() && !value.IsUndefined() && value.String() != "" {
		return value.String()
	}
	// The router drops the query on boot; rememberBootQuery keeps the room so
	// a reload of /p still knows it (and can rejoin with the saved seat).
	if storage := js.Global().Get("sessionStorage"); storage.Truthy() {
		if saved := storage.Call("getItem", "df-room"); saved.Truthy() {
			return saved.String()
		}
	}
	return ""
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
	return map[string]string{"min-height": "100%", "box-sizing": "border-box", "display": "flex", "flex-direction": "column", "justify-content": "center", "padding": "4px 0", "background": "radial-gradient(circle at 50% 0%, rgba(41,42,55,.92) 0, rgba(21,25,34,.86) 44%, rgba(11,15,22,.94) 100%)", "color": "#efe6d2", "font-family": "Inter, ui-sans-serif, system-ui, sans-serif"}
}

func joinCardStyle() map[string]string {
	return map[string]string{"width": "100%", "box-sizing": "border-box", "padding": "15px 13px", "border": "1px solid rgba(217,164,65,.56)", "border-radius": "12px", "background": "linear-gradient(145deg, rgba(31,31,43,.97), rgba(16,20,28,.98))", "box-shadow": "0 18px 45px rgba(0,0,0,.4), inset 0 0 0 1px rgba(239,230,210,.04)"}
}

func joinHintStyle() map[string]string {
	return map[string]string{"min-height": "17px", "margin": "2px 0 0", "color": "#a89f8c", "font-size": "11px", "line-height": "1.3"}
}

func joinLabelStyle() map[string]string {
	return map[string]string{"margin-top": "5px", "color": "#c8bda8", "font-size": "11px", "font-weight": "600", "letter-spacing": ".05em", "text-transform": "uppercase"}
}

func joinInputStyle() map[string]string {
	return map[string]string{"width": "100%", "height": "50px", "box-sizing": "border-box", "padding": "0 12px", "border": "1px solid rgba(239,230,210,.24)", "border-radius": "8px", "background": "#11131b", "color": "#efe6d2", "font-size": "16px", "outline": "none"}
}

func joinCodeStyle() map[string]string {
	style := joinInputStyle()
	style["letter-spacing"] = "4px"
	style["font-weight"] = "700"
	style["font-family"] = "Cormorant Garamond, Georgia, serif"
	return style
}

func joinButtonStyle() map[string]string {
	return map[string]string{"width": "100%", "min-height": "56px", "margin-top": "6px", "border": "1px solid #e7c27a", "border-radius": "7px", "background": "linear-gradient(180deg, #60461f, #2d2419)", "box-shadow": "inset 0 0 14px rgba(217,164,65,.18), 0 5px 16px rgba(0,0,0,.3)", "color": "#efe6d2", "font-family": "Cormorant Garamond, Georgia, serif", "font-size": "20px", "cursor": "pointer", "touch-action": "manipulation"}
}
