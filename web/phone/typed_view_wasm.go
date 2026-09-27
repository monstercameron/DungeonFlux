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
		locale := snapshot.Locale
		if locale == "" {
			locale = "en"
		}
		return html.Main(html.Props{Class: "df-phone df-phone-typed"},
			html.Label(html.Props{For: "typed-message"}, html.Text(TypedLabel(locale))),
			html.Input(html.Props{ID: "typed-message", Class: "df-phone-typed-input", Value: snapshot.Text, Placeholder: TypedHint(locale), OnInput: change, AutoFocus: snapshot.Open, Disabled: snapshot.Sending, MaxLength: typedInputLimit, Aria: map[string]string{"describedby": "typed-message-status", "label": TypedLabel(locale)}}),
			html.Div(html.Props{Class: "df-phone-typed-actions"},
				html.Span(html.Props{Class: "df-phone-typed-count", Aria: map[string]string{"live": "polite"}}, html.Textf("%d/%d", snapshot.Characters, typedInputLimit)),
				html.Button(html.Props{Type: "button", Class: "df-phone-typed-send", OnClick: send, Disabled: !snapshot.CanSubmit}, html.Text(TypedSend(locale))),
			),
			html.P(html.Props{ID: "typed-message-status", Role: "status", Class: "df-phone-typed-status"}, html.Text(errorOrStatus(snapshot))),
		)
	}
}

func errorOrStatus(s TypedInputSnapshot) string {
	if s.Error != "" {
		return s.Error
	}
	return s.StatusText
}
