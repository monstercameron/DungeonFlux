//go:build js && wasm

package phone

import (
	"context"

	"github.com/monstercameron/GoWebComponents/v6/html"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

// creationNameRow renders the generated hero name prominently, with a pencil
// control that opens an inline editor (tap-to-edit). Renaming is legal from
// roll_hero until the seat locks; once locked, the row is read-only.
func creationNameRow(model *CreationModel, refresh stateCounter, locale string, snapshot CreationSnapshot) ui.Node {
	if snapshot.Build == nil {
		return nil
	}
	if snapshot.Renaming {
		return creationRenameEditor(model, refresh, locale, snapshot)
	}
	name := snapshot.Build.GetName()
	locked := snapshot.Phase == CreationLocked
	edit := ui.UseEvent(func() {
		model.BeginRename()
		refresh.Set(refresh.Get() + 1)
	})
	row := []ui.Node{
		html.P(html.Props{Style: map[string]string{"margin": "0 0 3px", "color": "#e7c27a", "font-family": "Cormorant Garamond, Georgia, serif", "font-size": "22px"}}, html.Text(name)),
	}
	if !locked {
		row = append(row, html.Button(html.Props{
			Type: "button", OnClick: edit, Class: "df-phone-create-rename-btn",
			Aria: map[string]string{"label": T(locale, "phone.create.rename_action", nil)},
			Style: map[string]string{
				"margin": "0", "padding": "2px 8px", "border": "1px solid rgba(217,164,65,.5)", "border-radius": "999px",
				"background": "rgba(17,21,29,.6)", "color": "#e7c27a", "font-size": "12px", "line-height": "1.6",
				"touch-action": "manipulation",
			},
		}, html.Text("✎ "+T(locale, "phone.create.rename_action", nil))))
	}
	return html.Div(html.Props{Style: map[string]string{"display": "flex", "align-items": "center", "gap": "8px", "flex-wrap": "wrap"}}, row...)
}

func creationRenameEditor(model *CreationModel, refresh stateCounter, locale string, snapshot CreationSnapshot) ui.Node {
	change := ui.UseEvent(func(event ui.InputEvent) {
		model.SetRenameDraft(event.GetValue())
		refresh.Set(refresh.Get() + 1)
	})
	save := ui.UseEvent(func() {
		go func() { <-model.SubmitRename(context.Background()); refresh.Set(refresh.Get() + 1) }()
	})
	cancel := ui.UseEvent(func() {
		model.CancelRename()
		refresh.Set(refresh.Get() + 1)
	})
	errorText := renameErrorMessage(locale, snapshot.RenameError)
	fieldStyle := map[string]string{
		"flex": "1 1 auto", "min-width": "0", "padding": ".4rem .55rem", "border-radius": "6px",
		"border": "1px solid rgba(217,164,65,.55)", "background": "rgba(10,13,19,.9)", "color": "#efe6d2",
		"font-family": "Cormorant Garamond, Georgia, serif", "font-size": "18px",
	}
	if errorText != "" {
		fieldStyle["border-color"] = "#d96a4a"
	}
	return html.Div(html.Props{Class: "df-phone-create-rename", Style: map[string]string{"display": "grid", "gap": "6px", "width": "100%"}},
		html.Label(html.Props{For: "hero-name-input", Style: map[string]string{"margin": "0", "color": "#a89f8c", "font-size": "11px", "letter-spacing": ".08em", "text-transform": "uppercase"}}, html.Text(T(locale, "phone.create.rename_title", nil))),
		html.Div(html.Props{Style: map[string]string{"display": "flex", "gap": "6px", "align-items": "center"}},
			html.Input(html.Props{
				ID: "hero-name-input", Value: snapshot.RenameDraft, OnInput: change, AutoFocus: true,
				MaxLength: heroNameMaxLength, Placeholder: T(locale, "phone.create.rename_placeholder", nil),
				Aria: map[string]string{"label": T(locale, "phone.create.rename_title", nil)}, Style: fieldStyle,
			}),
			html.Button(html.Props{Type: "button", OnClick: save, Style: renameActionStyle(true)}, html.Text(T(locale, "phone.create.rename_save", nil))),
			html.Button(html.Props{Type: "button", OnClick: cancel, Style: renameActionStyle(false)}, html.Text(T(locale, "phone.create.rename_cancel", nil))),
		),
		html.P(html.Props{Role: "status", Aria: map[string]string{"live": "polite"}, Style: map[string]string{"margin": "0", "min-height": "14px", "color": "#d96a4a", "font-size": "12px"}}, html.Text(errorText)),
	)
}

func renameActionStyle(primary bool) map[string]string {
	style := map[string]string{
		"margin": "0", "padding": ".4rem .65rem", "border-radius": "6px", "font-size": "13px",
		"touch-action": "manipulation", "border": "1px solid rgba(217,164,65,.5)",
	}
	if primary {
		style["background"] = "linear-gradient(180deg, #d9a441, #b3813a)"
		style["color"] = "#1a1408"
		style["font-weight"] = "700"
	} else {
		style["background"] = "rgba(17,21,29,.6)"
		style["color"] = "#e7c27a"
	}
	return style
}

// renameErrorMessage maps a rename error to localized text. The two local
// validation codes ("empty", "too_long") go through the catalog; any other
// value is a server-supplied reason, rendered as-is (existing precedent:
// CreationSnapshot.Error is server text too, see create.go ApplyAct).
func renameErrorMessage(locale, code string) string {
	switch code {
	case "":
		return ""
	case "empty":
		return T(locale, "phone.create.rename_error_empty", nil)
	case "too_long":
		return T(locale, "phone.create.rename_error_too_long", nil)
	default:
		return code
	}
}
