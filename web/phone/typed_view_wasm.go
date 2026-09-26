//go:build js && wasm

package phone

import (
	"context"

	"github.com/monstercameron/GoWebComponents/v6/html"
	"github.com/monstercameron/GoWebComponents/v6/router"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

// TypedInputScreen renders the touch-first text fallback for speech input.
func TypedInputScreen(model *TypedInputModel) router.Component {
	return func(_ router.Attrs) *router.Element {
		refresh := ui.UseState(0)
		change := ui.UseEvent(func(event ui.InputEvent) {
			if model.SetText(event.GetValue()) == nil {
				refresh.Set(refresh.Get() + 1)
			}
		})
		send := ui.UseEvent(func() {
			go func() { model.ApplySay(<-model.Submit(context.Background())); refresh.Set(refresh.Get() + 1) }()
		})
		snapshot := model.Snapshot()
		return html.Main(html.Props{Class: "df-phone df-phone-typed"},
			html.Label(html.Props{For: "typed-message"}, html.Text("Type your message")),
			html.Input(html.Props{ID: "typed-message", Value: snapshot.Text, Placeholder: "What do you say?", OnInput: change, AutoFocus: snapshot.Open}),
			html.Button(html.Props{Type: "button", OnClick: send, Disabled: snapshot.Sending}, html.Text("Send")),
			html.P(html.Props{Role: "status"}, html.Text(errorOrStatus(snapshot))),
		)
	}
}

func errorOrStatus(s TypedInputSnapshot) string {
	if s.Error != "" {
		return s.Error
	}
	return s.StatusText
}
