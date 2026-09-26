//go:build js && wasm

package main

import (
	"context"
	"syscall/js"

	"github.com/monstercameron/DungeonFlux/web/phone"
	"github.com/monstercameron/GoWebComponents/v6/html"
	"github.com/monstercameron/GoWebComponents/v6/router"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

const phoneSeatStorageKey = "dungeonflux.phone.seat_token"

// JoinScreen returns the phone room-code and QR-link join component.
func JoinScreen(client *Client) router.Component {
	return func(_ router.Attrs) *router.Element {
		room := ui.UseState(browserRoomCode())
		locale := NewLocaleModel(BrowserLocales())
		view := ui.UseState(JoinSnapshot{RoomCode: room.Get(), Phase: JoinIdle, Locale: locale.Active()})
		join := ui.UseEvent(func() {
			model := NewJoinModel(client, room.Get())
			model.SetLocale(locale.Active())
			modelState := view.Get()
			modelState.Phase = JoinPending
			modelState.Locale = locale.Active()
			view.Set(modelState)
			result := model.StartJoin(context.Background(), browserSeatToken())
			go func() {
				state := model.ApplyJoin(<-result)
				if state.Phase == JoinJoined {
					storeBrowserSeatToken(state.SeatToken)
				}
				view.Set(state)
			}()
		})
		change := ui.UseEvent(func(event ui.InputEvent) { room.Set(event.GetValue()) })
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
		} else if state.Phase == JoinJoined {
			message = locale.T("shell.seat_ready", map[string]string{"seat": state.SeatID})
		}
		return html.Main(html.Props{Class: "df-join"},
			languageSwitcher(locale, view),
			html.H1(html.Props{}, html.Text(locale.T("shell.join_title", nil))),
			html.P(html.Props{}, html.Text(locale.T("shell.join_scan", nil))),
			html.Label(html.Props{For: "room-code"}, html.Text(locale.T("ui.join.room_code", nil))),
			html.Input(html.Props{ID: "room-code", Value: room.Get(), Placeholder: locale.T("shell.room_ph", nil), AutoFocus: true, OnInput: change}),
			html.Button(html.Props{Type: "button", OnClick: join, Disabled: state.Phase == JoinPending}, html.Text(locale.T("shell.join_table", nil))),
			html.P(html.Props{Class: "df-join-status", Role: "status"}, html.Text(message)),
		)
	}
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

func browserSeatToken() string {
	value := js.Global().Get("localStorage").Call("getItem", phoneSeatStorageKey)
	if value.IsNull() || value.IsUndefined() {
		return ""
	}
	return value.String()
}

func storeBrowserSeatToken(token string) {
	js.Global().Get("localStorage").Call("setItem", phoneSeatStorageKey, token)
}
