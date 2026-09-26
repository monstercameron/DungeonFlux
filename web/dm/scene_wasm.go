//go:build js && wasm

package dm

import (
	"strings"

	dungeonfluxv1 "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
	"github.com/monstercameron/GoWebComponents/v6/html"
	"github.com/monstercameron/GoWebComponents/v6/router"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

// SceneComponent renders the painterly, 16:9 DM scene used by the TV screen.
func SceneComponent(view *dungeonfluxv1.DMView) router.Component {
	model := SceneModelFromView(view)
	locale := localeOrDefault(view.GetLocale())
	return func(_ router.Attrs) *router.Element {
		return html.Section(html.Props{Class: "df-dm-scene", Role: "img", Aria: map[string]string{"label": T(locale, "dm.scene_label", nil)}, Style: map[string]string{"position": "relative", "width": "100%", "height": "100%", "overflow": "hidden"}},
			html.Div(html.Props{Class: "df-dm-scene-stage", Style: sceneStageStyle(model.BackgroundURL)}, sceneLayers(model.Layers)...),
			sceneVignette(),
			scenePartyRail(model.Characters, locale),
			sceneTitlePlate(model, locale),
			sceneProgressRail(model, locale),
			sceneCaption(model.Caption, model.SpeakerPortraitURL, model.FrameURL, model.DividerURL, locale),
		)
	}
}

func sceneStageStyle(backgroundURL string) map[string]string {
	background := "radial-gradient(ellipse at 50% 42%, rgba(42,39,34,.04), rgba(5,7,11,.42) 78%), linear-gradient(180deg, #111722, #0b0d12)"
	if backgroundURL != "" {
		background += ", url('" + backgroundURL + "')"
	}
	return map[string]string{
		"position": "relative", "width": "100%", "height": "100%", "background-image": background,
		"background-size": "cover", "background-position": "center 42%", "filter": "saturate(.92) contrast(1.04)",
	}
}

func sceneVignette() ui.Node {
	return html.Div(html.Props{Class: "df-dm-scene-vignette", Style: map[string]string{
		"position": "absolute", "inset": "0", "z-index": "2", "pointer-events": "none",
		"box-shadow": "inset 0 0 9vw rgba(0,0,0,.68), inset 0 -15vw 12vw rgba(0,0,0,.64)",
	}})
}

func sceneLayers(layers []SceneLayer) []ui.Node {
	nodes := make([]ui.Node, 0, len(layers))
	for _, layer := range layers {
		class := "df-dm-scene-layer"
		if layer.Highlight {
			class += " is-highlighted"
		}
		if layer.Speaking {
			class += " is-speaking"
		}
		style := SceneLayerStyle(layer)
		style["position"] = "absolute"
		style["z-index"] = "1"
		style["max-width"] = "52%"
		style["max-height"] = "72%"
		nodes = append(nodes, html.Img(html.Props{ID: layer.ID, Class: class, Src: layer.URL, Alt: "", Style: style, Raw: map[string]any{"aria-hidden": "true"}}))
	}
	return nodes
}

func scenePartyRail(characters []SceneCharacter, locale string) ui.Node {
	items := sceneCharacters(characters, locale)
	return html.Div(html.Props{Class: "df-dm-scene-cards", Role: "list", Aria: map[string]string{"label": T(locale, "dm.scene_heroes", nil)}, Style: map[string]string{
		"position": "absolute", "left": "1.5%", "top": "16%", "bottom": "18%", "width": "15.5%", "z-index": "4",
		"display": "flex", "flex-direction": "column", "justify-content": "center", "gap": "1.2rem", "pointer-events": "none",
	}}, items...)
}

func sceneCharacters(characters []SceneCharacter, locale string) []ui.Node {
	nodes := make([]ui.Node, 0, len(characters))
	for _, character := range characters {
		portrait := html.Div(html.Props{Class: "df-dm-scene-card-portrait", Style: map[string]string{
			"width": "100%", "height": "clamp(8rem, 18vh, 14rem)", "background": "linear-gradient(145deg, rgba(38,42,53,.95), rgba(9,12,17,.92))",
			"overflow": "hidden", "display": "flex", "align-items": "flex-end", "justify-content": "center",
		}}, html.Img(html.Props{Src: character.PortraitURL, Alt: character.Name, Style: map[string]string{"width": "100%", "height": "100%", "object-fit": "cover"}}))
		name := character.Name
		if name == "" {
			name = SeatName(locale, character.Name, int(character.PlayerNumber))
		}
		nodes = append(nodes, html.Div(html.Props{Class: "df-dm-scene-card", Role: "listitem", Style: map[string]string{
			"position": "relative", "display": "flex", "flex-direction": "column", "min-width": "0", "overflow": "hidden",
			"border": "1px solid rgba(217,164,65,.78)", "border-radius": "10px", "background": "linear-gradient(165deg, rgba(16,20,28,.95), rgba(8,10,15,.92))",
			"box-shadow": "0 10px 30px rgba(0,0,0,.54), inset 0 0 0 1px rgba(239,230,210,.06)", "color": "#efe6d2",
		}},
			portrait,
			html.Div(html.Props{Class: "df-dm-scene-card-copy", Style: map[string]string{"padding": ".55rem .75rem .7rem", "text-align": "center", "text-shadow": "0 1px 2px #000"}},
				html.Strong(html.Props{Style: map[string]string{"display": "block", "font-family": "Georgia, 'Times New Roman', serif", "font-size": "clamp(1.05rem, 1.45vw, 1.8rem)", "line-height": "1.05"}}, ui.Text(name)),
				html.Small(html.Props{Style: map[string]string{"display": "block", "margin-top": ".28rem", "color": "#a89f8c", "font-size": "clamp(.75rem, .95vw, 1.15rem)", "letter-spacing": ".04em"}}, ui.Text(character.Class)),
			),
		))
	}
	return nodes
}

func sceneTitlePlate(model SceneModel, locale string) ui.Node {
	if !model.ShowTitle {
		return html.Div(html.Props{Hidden: true})
	}
	return html.Div(html.Props{Class: "df-dm-scene-title", Style: map[string]string{
		"position": "absolute", "left": "20%", "right": "20%", "top": "37%", "z-index": "4", "text-align": "center",
		"color": "#efe6d2", "text-shadow": "0 3px 14px rgba(0,0,0,.92)", "pointer-events": "none",
	}},
		html.Div(html.Props{Style: map[string]string{"color": "#d9a441", "font-family": "Georgia, serif", "font-size": "clamp(1rem, 1.3vw, 1.65rem)", "letter-spacing": ".34em", "font-weight": "700"}}, ui.Text(model.Act)),
		html.Div(html.Props{Style: map[string]string{"height": "1px", "margin": ".35rem auto .55rem", "max-width": "26rem", "background": "linear-gradient(90deg, transparent, #d9a441, transparent)"}}),
		html.H1(html.Props{Style: map[string]string{"margin": "0", "font-family": "Georgia, 'Times New Roman', serif", "font-size": "clamp(2.3rem, 4.5vw, 5.8rem)", "font-weight": "500", "line-height": "1.02", "letter-spacing": ".02em"}}, ui.Text(SceneTitle(locale))),
		html.P(html.Props{Style: map[string]string{"margin": ".7rem auto 0", "max-width": "42rem", "font-family": "Georgia, serif", "font-size": "clamp(1rem, 1.45vw, 1.8rem)", "line-height": "1.18", "color": "#d7cdbb"}}, ui.Text(model.Tagline)),
	)
}

func sceneProgressRail(model SceneModel, locale string) ui.Node {
	steps := make([]ui.Node, 0, len(model.Progress))
	for _, step := range model.Progress {
		color := "#59616b"
		if step.Completed {
			color = "#d9a441"
		}
		if step.Active {
			color = "#efe6d2"
		}
		steps = append(steps, html.Div(html.Props{Style: map[string]string{"display": "flex", "align-items": "center", "gap": ".7rem", "margin": ".9rem 0", "color": color, "font-size": "clamp(.85rem, 1.15vw, 1.35rem)"}},
			html.Span(html.Props{Role: "img", Aria: map[string]string{"label": step.Label}, Style: map[string]string{"display": "block", "width": ".9rem", "height": ".9rem", "flex": "0 0 .9rem", "border": "2px solid " + color, "border-radius": "50%", "box-shadow": "0 0 .5rem " + color}}),
			html.Span(html.Props{Style: map[string]string{"line-height": "1.1", "font-weight": boolWeight(step.Active)}}, ui.Text(step.Label)),
		))
	}
	return html.Aside(html.Props{Class: "df-dm-scene-progress", Aria: map[string]string{"label": T(locale, "dm.scene_label", nil)}, Style: map[string]string{
		"position": "absolute", "right": "1.5%", "top": "16%", "bottom": "18%", "width": "16%", "z-index": "4", "padding": "1.1rem 1rem",
		"border": "1px solid rgba(217,164,65,.7)", "border-radius": "5px", "background": panelBackground(model.FrameURL), "box-shadow": "0 12px 32px rgba(0,0,0,.5)", "color": "#efe6d2",
	}},
		html.H2(html.Props{Style: map[string]string{"margin": "0 0 .85rem", "font-family": "Georgia, serif", "font-size": "clamp(1rem, 1.45vw, 1.8rem)", "font-weight": "500"}}, ui.Text("Current Scene")),
		html.Div(html.Props{Style: map[string]string{"height": "clamp(4rem, 10vh, 7rem)", "margin-bottom": "1rem", "border": "1px solid rgba(217,164,65,.25)", "background-image": sceneThumbBackground(model.BackgroundURL), "background-size": "cover", "background-position": "center", "filter": "saturate(.82)"}}),
		html.Div(html.Props{Style: map[string]string{"height": "1px", "margin": ".35rem 0 .75rem", "background": "linear-gradient(90deg, #d9a441, transparent)"}}),
		html.Div(html.Props{Style: map[string]string{"font-size": "clamp(.9rem, 1.25vw, 1.5rem)", "font-family": "Georgia, serif", "line-height": "1.1"}}, ui.Text(SceneTitle(locale))),
		html.Div(html.Props{Style: map[string]string{"margin-top": ".6rem"}}, steps...),
	)
}

func sceneThumbBackground(url string) string {
	if url == "" {
		return "linear-gradient(145deg, #243348, #10141d)"
	}
	return "linear-gradient(180deg, rgba(12,16,22,.16), rgba(12,16,22,.66)), url('" + url + "')"
}

func panelBackground(frameURL string) string {
	base := "linear-gradient(145deg, rgba(13,17,24,.96), rgba(8,10,15,.93))"
	if frameURL != "" {
		return base + ", url('" + frameURL + "')"
	}
	return base
}

func boolWeight(active bool) string {
	if active {
		return "700"
	}
	return "400"
}

func sceneCaption(caption SceneCaption, portraitURL, frameURL, dividerURL, locale string) ui.Node {
	if !caption.Visible {
		return html.Div(html.Props{Class: "df-dm-scene-caption", Hidden: true})
	}
	name := SceneSpeaker(locale, caption.Speaker)
	style := map[string]string{
		"position": "absolute", "left": "25%", "right": "22%", "bottom": "3.8%", "z-index": "5", "min-height": "13%",
		"display": "flex", "align-items": "center", "gap": "1.1rem", "padding": "1.2rem 2rem 1.25rem 4.8rem", "border": "1px solid rgba(217,164,65,.78)",
		"border-radius": "5px", "background": panelBackground(frameURL), "box-shadow": "0 12px 36px rgba(0,0,0,.6), inset 0 0 0 1px rgba(239,230,210,.06)",
		"color": "#efe6d2", "font-family": "Georgia, 'Times New Roman', serif", "pointer-events": "none",
	}
	portrait := html.Div(html.Props{Class: "df-dm-scene-speaker", Style: map[string]string{
		"position": "absolute", "left": "-4.3rem", "bottom": "-1.2rem", "width": "8rem", "height": "8rem", "border": "1px solid #d9a441", "border-radius": "50%", "overflow": "hidden",
		"background": "radial-gradient(circle at 50% 38%, #2b3542, #07090d 70%)", "box-shadow": "0 0 0 4px rgba(15,17,23,.92), 0 8px 24px rgba(0,0,0,.65)",
	}}, html.Img(html.Props{Src: portraitURL, Alt: name, Style: map[string]string{"width": "100%", "height": "100%", "object-fit": "cover"}}))
	if portraitURL == "" {
		portrait = html.Div(html.Props{Class: "df-dm-scene-speaker", Style: map[string]string{
			"position": "absolute", "left": "-4.3rem", "bottom": "-1.2rem", "width": "8rem", "height": "8rem", "border": "1px solid #d9a441", "border-radius": "50%", "background": "radial-gradient(circle at 50% 38%, #2b3542, #07090d 70%)", "box-shadow": "0 0 0 4px rgba(15,17,23,.92), 0 8px 24px rgba(0,0,0,.65)",
		}}, html.Span(html.Props{Style: map[string]string{"margin": "auto", "color": "#d9a441", "font-size": "2rem"}}, ui.Text("✦")))
	}
	return html.Div(html.Props{Class: "df-dm-scene-caption", Role: "status", Aria: map[string]string{"live": "polite"}, Style: style},
		portrait,
		html.Div(html.Props{Style: map[string]string{"min-width": "0", "flex": "1"}},
			html.Strong(html.Props{Class: "df-dm-scene-caption-speaker", Style: map[string]string{"display": "block", "margin-bottom": ".35rem", "color": "#d9a441", "font-family": "Arial, sans-serif", "font-size": "clamp(.9rem, 1.1vw, 1.35rem)", "font-weight": "700", "letter-spacing": ".14em", "text-transform": "uppercase"}}, ui.Text(name)),
			html.Div(html.Props{Style: map[string]string{"height": "1px", "margin-bottom": ".6rem", "background": sceneDividerBackground(dividerURL)}}),
			html.P(html.Props{Class: "df-dm-scene-caption-text", Style: map[string]string{"margin": "0", "font-size": "clamp(1.15rem, 1.8vw, 2.2rem)", "line-height": "1.22", "font-weight": "500", "text-wrap": "balance"}}, ui.Text(strings.TrimSpace(caption.Text))),
		),
	)
}

func sceneDividerBackground(url string) string {
	if url == "" {
		return "linear-gradient(90deg, transparent, #d9a441 20%, #d9a441 80%, transparent)"
	}
	return "url('" + url + "') center / 100% 100% no-repeat"
}
