//go:build js && wasm

package phone

import (
	"context"
	"strconv"

	"github.com/monstercameron/GoWebComponents/v6/html"
	"github.com/monstercameron/GoWebComponents/v6/router"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

// CreationScreen renders the touch-first character creation card.
func CreationScreen(model *CreationModel) router.Component {
	return func(_ router.Attrs) *router.Element {
		refresh := ui.UseState(0)
		snapshot := model.Snapshot()
		locale := snapshot.Locale
		if locale == "" {
			locale = "en"
		}
		roll := ui.UseEvent(func() {
			go func() { model.ApplyAct(<-model.RollHero(context.Background())); refresh.Set(refresh.Get() + 1) }()
		})
		lock := ui.UseEvent(func() {
			go func() { model.ApplyAct(<-model.Lock(context.Background())); refresh.Set(refresh.Get() + 1) }()
		})
		pickerDisabled := snapshot.Build != nil || snapshot.Phase == CreationRolling || snapshot.Phase == CreationLocked
		var action ui.Node = creationRollButton(roll, locale, snapshot)
		if snapshot.Build != nil && snapshot.Phase == CreationRolling {
			action = creationLockButton(lock, locale)
		}
		if snapshot.Phase == CreationLocked {
			action = creationLockedButton(locale)
		}
		return html.Section(html.Props{Class: "df-phone-create", Role: "main", Style: creationContentStyle()},
			creationHeading(locale),
			creationPicker(model, refresh, "species", "Species", creationSpecies, snapshot.Species, pickerDisabled),
			creationPicker(model, refresh, "gender", "Gender", creationGenders, snapshot.Gender, pickerDisabled),
			creationClassPicker(model, refresh, locale, snapshot.Class, pickerDisabled),
			creationBuildCard(snapshot),
			html.Div(html.Props{Style: map[string]string{"margin-top": "auto", "padding-top": "2px"}}, action,
				html.P(html.Props{Role: "status", Aria: map[string]string{"live": "polite"}, Style: map[string]string{"min-height": "18px", "margin": "7px 0 0", "color": "#a89f8c", "font-size": "12px", "line-height": "1.35", "text-align": "center"}}, html.Text(creationStatus(snapshot)))),
		)
	}
}

type stateCounter interface {
	Get() int
	Set(int)
}

func creationPicker(model *CreationModel, refresh stateCounter, id, label string, options []CreationOption, selected string, disabled bool) ui.Node {
	choices := make([]ui.Node, 0, len(options))
	for _, option := range options {
		choice := option
		tap := ui.UseEvent(func() {
			if id == "species" {
				_ = model.SelectSpecies(choice.ID)
			} else {
				_ = model.SelectGender(choice.ID)
			}
			refresh.Set(refresh.Get() + 1)
		})
		style := map[string]string{"min-height": "52px", "min-width": "0", "width": "100%", "padding": ".45rem .25rem", "display": "grid", "gap": "3px", "place-items": "center", "border-radius": "8px", "border": "1px solid rgba(168,159,140,.42)", "background": "rgba(18,22,29,.92)", "color": "#efe6d2", "font-family": "Cormorant Garamond, Georgia, serif", "font-size": ".96rem", "touch-action": "manipulation"}
		if selected == choice.ID {
			style["border-color"] = "#d9a441"
			style["background"] = "linear-gradient(180deg, rgba(67,48,23,.9), rgba(30,25,20,.96))"
			style["box-shadow"] = "inset 0 0 14px rgba(217,164,65,.16), 0 0 10px rgba(217,164,65,.12)"
		}
		choices = append(choices, html.Button(html.Props{Type: "button", OnClick: tap, Disabled: disabled, Aria: map[string]string{"pressed": strconv.FormatBool(selected == choice.ID)}, Style: style}, creationOptionArt(id, choice.ID, choice.Label), html.Text(choice.Label)))
	}
	return html.Fieldset(html.Props{Class: "df-phone-create-field", Style: createFieldStyle()}, html.Legend(html.Props{Style: map[string]string{"padding": "0 7px", "color": "#e7c27a", "font-family": "Cormorant Garamond, Georgia, serif", "font-size": "17px"}}, html.Text(label)), html.Div(html.Props{Style: map[string]string{"min-width": "0", "display": "grid", "grid-template-columns": "repeat(3, minmax(0, 1fr))", "gap": ".5rem"}}, choices...))
}

func creationBackground() string {
	if url := ArtURL(phoneBackgroundAsset); url != "" {
		return "linear-gradient(180deg, rgba(10, 13, 19, .24), rgba(10, 13, 19, .86)), url(\"" + url + "\")"
	}
	return "radial-gradient(circle at 50% 0%, #242b3b 0, #141923 42%, #0b0f16 100%)"
}

func creationContentStyle() map[string]string {
	return map[string]string{"min-height": "100%", "box-sizing": "border-box", "display": "flex", "flex-direction": "column", "gap": "12px", "padding": "4px 0 2px", "background": creationBackground(), "background-size": "cover", "background-position": "center", "color": "#efe6d2", "font-family": "Inter, ui-sans-serif, system-ui, sans-serif"}
}

func creationHeading(locale string) ui.Node {
	return html.Div(html.Props{Class: "df-phone-create-heading", Style: map[string]string{"padding": "8px 4px 5px", "text-align": "center"}},
		html.P(html.Props{Style: map[string]string{"margin": "0 0 3px", "color": "#d9a441", "font-size": "10px", "font-weight": "700", "letter-spacing": ".2em"}}, html.Text("YOUR PHONE CONTROLS THE HERO")),
		html.H1(html.Props{Style: map[string]string{"margin": "0", "color": "#efe6d2", "font-family": "Cormorant Garamond, Cinzel, Georgia, serif", "font-size": "29px", "line-height": "1.05"}}, html.Text(CreateTitle(locale))),
		html.P(html.Props{Style: map[string]string{"margin": "5px 0 0", "color": "#a89f8c", "font-family": "Cormorant Garamond, Georgia, serif", "font-size": "16px", "line-height": "1.25"}}, html.Text(creationHint(locale))),
	)
}

func createFieldStyle() map[string]string {
	return map[string]string{"min-width": "0", "width": "100%", "box-sizing": "border-box", "margin": "0", "padding": "7px", "border": "1px solid rgba(217,164,65,.34)", "border-radius": "10px", "background": "rgba(17,21,29,.82)", "box-shadow": "inset 0 0 18px rgba(0,0,0,.16)"}
}

func creationOptionArt(kind, id, label string) ui.Node {
	asset := ""
	if kind == "species" {
		asset = speciesArtAsset(id)
	}
	if url := ArtURL(asset); url != "" {
		return html.Img(html.Props{Src: url, Alt: label, Style: map[string]string{"width": "24px", "height": "26px", "object-fit": "cover", "border-radius": "5px"}})
	}
	mark := "•"
	if kind == "gender" {
		mark = "○"
	}
	return html.Span(html.Props{Role: "img", Aria: map[string]string{"label": label}, Style: map[string]string{"display": "grid", "place-items": "center", "width": "24px", "height": "24px", "border": "1px solid rgba(217,164,65,.55)", "border-radius": "50%", "color": "#e7c27a", "font-family": "Georgia, serif", "font-size": "14px"}}, html.Text(mark))
}

func creationBuildCard(snapshot CreationSnapshot) ui.Node {
	if snapshot.Build == nil {
		return html.Div(html.Props{Class: "df-phone-create-empty", Style: map[string]string{"padding": "13px 12px", "border": "1px dashed rgba(168,159,140,.35)", "border-radius": "10px", "color": "#a89f8c", "font-family": "Cormorant Garamond, Georgia, serif", "font-size": "16px", "text-align": "center"}}, html.Text("Your rolled hero will appear here."))
	}
	build := snapshot.Build
	portrait := html.Div(html.Props{Role: "img", Aria: map[string]string{"label": build.GetName()}, Style: map[string]string{"width": "76px", "height": "96px", "display": "grid", "place-items": "center", "flex": "0 0 76px", "border-radius": "8px", "border": "1px solid rgba(217,164,65,.5)", "background": "radial-gradient(circle, #354052, #171a23 70%)", "color": "#e7c27a", "font-family": "Georgia, serif", "font-size": "1.4rem"}}, html.Text("✦"))
	src := portraitSrc(build.GetPortraitUrl())
	if src == "" {
		src = heroProxyArt(snapshot.Species, build.GetClassName(), build.GetName())
	}
	if src != "" {
		portrait = html.Div(html.Props{Style: map[string]string{"width": "76px", "height": "96px", "flex": "0 0 76px", "border-radius": "8px", "border": "1px solid rgba(217,164,65,.5)", "background": "#25232b", "overflow": "hidden"}}, html.Img(html.Props{Src: src, Alt: build.GetName(), Style: map[string]string{"width": "100%", "height": "100%", "object-fit": "cover"}}))
	}
	return html.Section(html.Props{Class: "df-phone-create-build", Style: map[string]string{"display": "flex", "gap": "12px", "align-items": "center", "padding": "10px", "border": "1px solid #d9a441", "border-radius": "10px", "background": "linear-gradient(135deg, rgba(47,39,31,.96), rgba(17,21,29,.98))", "box-shadow": "inset 0 0 24px rgba(217,164,65,.08)"}}, portrait, html.Div(html.Props{Style: map[string]string{"min-width": "0"}}, html.P(html.Props{Style: map[string]string{"margin": "0 0 3px", "color": "#e7c27a", "font-family": "Cormorant Garamond, Georgia, serif", "font-size": "22px"}}, html.Text(build.GetName())), html.P(html.Props{Style: map[string]string{"margin": "0", "color": "#efe6d2", "font-size": "13px", "letter-spacing": ".08em", "text-transform": "uppercase"}}, html.Text(build.GetClassName()))))
}

func creationStatus(snapshot CreationSnapshot) string {
	if snapshot.Error != "" {
		return snapshot.Error
	}
	if snapshot.StatusText != "" {
		return snapshot.StatusText
	}
	switch snapshot.Phase {
	case CreationRolling:
		return "The engine is rolling your hero..."
	case CreationLocked:
		return "Hero locked in."
	default:
		return "Choose one from each list."
	}
}

func creationRollButton(roll ui.Handler, locale string, snapshot CreationSnapshot) ui.Node {
	disabled := snapshot.Species == "" || snapshot.Gender == "" || snapshot.Class == "" || snapshot.Phase == CreationRolling || snapshot.Phase == CreationLocked
	return PrimaryButton(RollHeroLabel(locale), roll, disabled)
}

func creationLockButton(lock ui.Handler, locale string) ui.Node {
	return PrimaryButton(creationReadyLabel(locale), lock, false)
}

func creationLockedButton(locale string) ui.Node {
	return PrimaryButton(creationLockedLabel(locale), ui.Handler{}, true)
}
