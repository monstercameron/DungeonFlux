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
		view := ui.UseState(JoinSnapshot{RoomCode: room.Get(), Phase: JoinIdle})
		join := ui.UseEvent(func() {
			model := NewJoinModel(client, room.Get())
			modelState := view.Get()
			modelState.Phase = JoinPending
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
		if state.Phase == JoinJoined {
			return ui.CreateElement(phone.Mount(phoneClientAdapter{client: client}, state.SeatToken))
		}
		message := state.Error
		if state.Phase == JoinPending {
			message = "Joining the table…"
		} else if state.Phase == JoinJoined {
			message = "Seat " + state.SeatID + " is ready."
		}
		return html.Main(html.Props{Class: "df-join"},
			html.H1(html.Props{}, html.Text("Join DungeonFlux")),
			html.P(html.Props{}, html.Text("Scan the QR code or enter the room code shown on the DM screen.")),
			html.Label(html.Props{For: "room-code"}, html.Text("Room code")),
			html.Input(html.Props{ID: "room-code", Value: room.Get(), Placeholder: "ROOM CODE", AutoFocus: true, OnInput: change}),
			html.Button(html.Props{Type: "button", OnClick: join, Disabled: state.Phase == JoinPending}, html.Text("Join table")),
			html.P(html.Props{Class: "df-join-status", Role: "status"}, html.Text(message)),
		)
	}
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
