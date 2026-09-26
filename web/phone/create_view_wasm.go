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
		roll := ui.UseEvent(func() {
			go func() { model.ApplyAct(<-model.RollHero(context.Background())); refresh.Set(refresh.Get() + 1) }()
		})
		snapshot := model.Snapshot()
		locale := snapshot.Locale
		if locale == "" {
			locale = "en"
		}
		return html.Main(html.Props{Class: "df-phone df-phone-create", Role: "main", Style: map[string]string{
			"min-height": "100svh", "box-sizing": "border-box", "display": "flex", "flex-direction": "column", "gap": "1rem", "padding": "1.25rem 1rem 1rem", "background": "#10131b", "color": "#efe6d2", "overflow-x": "hidden",
		}},
			html.Div(html.Props{Style: map[string]string{"max-width": "34rem", "width": "100%", "margin": "0 auto"}},
				html.P(html.Props{Style: map[string]string{"margin": "0 0 .35rem", "color": "#d9a441", "font-size": ".75rem", "letter-spacing": ".16em", "text-transform": "uppercase"}}, html.Text("DUNGEONFLUX")),
				html.H1(html.Props{Style: map[string]string{"margin": "0", "font-family": "Georgia, serif", "font-size": "clamp(2rem, 9vw, 3rem)", "line-height": "1.05"}}, html.Text(CreateTitle(locale))),
				html.P(html.Props{Style: map[string]string{"margin": ".55rem 0 0", "color": "#a89f8c", "font-size": "1rem", "line-height": "1.45"}}, html.Text(CreateHint(locale))),
			),
			creationPicker(model, refresh, "species", "Species", creationSpecies, snapshot.Species),
			creationPicker(model, refresh, "gender", "Gender", creationGenders, snapshot.Gender),
			creationBuildCard(snapshot),
			html.Div(html.Props{Style: map[string]string{"margin-top": "auto", "max-width": "34rem", "width": "100%", "margin-left": "auto", "margin-right": "auto"}},
				html.Button(html.Props{Type: "button", OnClick: roll, Disabled: snapshot.Species == "" || snapshot.Gender == "" || snapshot.Phase == CreationRolling || snapshot.Phase == CreationLocked, Style: map[string]string{"width": "100%", "min-height": "56px", "border": "0", "border-radius": "12px", "background": "#d9a441", "color": "#16130d", "font-size": "1.1rem", "font-weight": "700", "box-shadow": "0 5px 18px rgba(217,164,65,.2)"}}, html.Text(RollHeroLabel(locale))),
				html.P(html.Props{Role: "status", Aria: map[string]string{"live": "polite"}, Style: map[string]string{"min-height": "1.4rem", "margin": ".65rem 0 0", "text-align": "center", "color": "#bdb4a2", "font-size": ".9rem"}}, html.Text(creationStatus(snapshot))),
			),
		)
	}
}

type stateCounter interface {
	Get() int
	Set(int)
}

func creationPicker(model *CreationModel, refresh stateCounter, id, label string, options []CreationOption, selected string) ui.Node {
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
		style := map[string]string{"min-height": "48px", "min-width": "0", "width": "100%", "padding": ".7rem .35rem", "border-radius": "10px", "border": "1px solid #4b4b4c", "background": "#1a1d26", "color": "#efe6d2", "font-size": ".95rem"}
		if selected == choice.ID {
			style["border-color"] = "#d9a441"
			style["background"] = "#332a19"
			style["box-shadow"] = "inset 0 0 0 1px #d9a441"
		}
		choices = append(choices, html.Button(html.Props{Type: "button", OnClick: tap, Aria: map[string]string{"pressed": strconv.FormatBool(selected == choice.ID)}, Style: style}, html.Text(choice.Label)))
	}
	return html.Fieldset(html.Props{Style: map[string]string{"max-width": "34rem", "min-width": "0", "width": "100%", "box-sizing": "border-box", "margin": "0 auto", "padding": ".8rem", "border": "1px solid #3a3a42", "border-radius": "12px", "background": "#171a23"}}, html.Legend(html.Props{Style: map[string]string{"padding": "0 .35rem", "color": "#efe6d2", "font-weight": "700"}}, html.Text(label)), html.Div(html.Props{Style: map[string]string{"min-width": "0", "display": "grid", "grid-template-columns": "repeat(3, minmax(0, 1fr))", "gap": ".55rem"}}, choices...))
}

func creationBuildCard(snapshot CreationSnapshot) ui.Node {
	if snapshot.Build == nil {
		return html.Div(html.Props{Style: map[string]string{"max-width": "34rem", "min-width": "0", "width": "100%", "box-sizing": "border-box", "margin": "0 auto", "padding": "1rem", "border": "1px dashed #484750", "border-radius": "12px", "color": "#a89f8c", "text-align": "center"}}, html.Text("Your rolled hero will appear here."))
	}
	build := snapshot.Build
	var portrait ui.Node = html.Div(html.Props{Role: "img", Aria: map[string]string{"label": build.GetName()}, Style: map[string]string{"width": "72px", "height": "92px", "display": "flex", "align-items": "center", "justify-content": "center", "border-radius": "10px", "background": "#25232b", "color": "#d9a441", "font-family": "Georgia, serif", "font-size": "1.4rem"}}, html.Text("✦"))
	if build.GetPortraitUrl() != "" {
		portrait = html.Div(html.Props{Style: map[string]string{"width": "72px", "height": "92px", "border-radius": "10px", "background": "#25232b", "overflow": "hidden"}}, html.Img(html.Props{Src: build.GetPortraitUrl(), Alt: build.GetName(), Style: map[string]string{"width": "100%", "height": "100%", "object-fit": "cover"}}))
	}
	return html.Section(html.Props{Style: map[string]string{"max-width": "34rem", "min-width": "0", "width": "100%", "box-sizing": "border-box", "margin": "0 auto", "padding": "1rem", "display": "flex", "gap": "1rem", "align-items": "center", "border": "1px solid #d9a441", "border-radius": "12px", "background": "linear-gradient(135deg, #26222a, #171a23)", "box-shadow": "inset 0 0 24px rgba(217,164,65,.08)"}}, portrait, html.Div(html.Props{Style: map[string]string{"min-width": "0"}}, html.P(html.Props{Style: map[string]string{"margin": "0 0 .25rem", "font-family": "Georgia, serif", "font-size": "1.35rem"}}, html.Text(build.GetName())), html.P(html.Props{Style: map[string]string{"margin": "0", "color": "#d9a441", "font-weight": "700"}}, html.Text(build.GetClassName()))))
}

func creationStatus(snapshot CreationSnapshot) string {
	if snapshot.Error != "" {
		return snapshot.Error
	}
	switch snapshot.Phase {
	case CreationRolling:
		return "The engine is rolling your hero…"
	case CreationLocked:
		return "Hero locked in."
	default:
		return "Choose one from each list."
	}
}
