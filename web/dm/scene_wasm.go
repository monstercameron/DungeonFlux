//go:build js && wasm

package dm

import (
	"strings"

	dungeonfluxv1 "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
	"github.com/monstercameron/GoWebComponents/v6/html"
	"github.com/monstercameron/GoWebComponents/v6/router"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

// SceneComponent renders the painterly scene on the fixed DM canvas.
func SceneComponent(view *dungeonfluxv1.DMView, phase ...string) router.Component {
	model := SceneModelFromView(view)
	composition := sceneComposition("opening")
	if len(phase) > 0 {
		composition = sceneComposition(phase[0])
	}
	locale := localeOrDefault(view.GetLocale())
	return func(_ router.Attrs) *router.Element {
		children := []ui.Node{
			html.Div(html.Props{Class: "df-dm-scene-stage", Style: sceneStageStyle(model.BackgroundURL)}, sceneLayers(model.Layers)...),
			sceneVignette(),
		}
		if composition.chrome {
			children = append(children, sceneBrand(model), sceneLocation(model), scenePartyRail(model.Characters, locale))
		}
		if composition.opening {
			children = append(children, sceneTitle(model), sceneProgressRail(model, locale))
		}
		if composition.caption {
			children = append(children, sceneCaption(model.Caption, model.SpeakerPortraitURL, model.FrameURL, model.DividerURL, locale))
		}
		return html.Section(html.Props{Class: "df-dm-scene", Aria: map[string]string{"label": T(locale, "dm.scene_label", nil)}, Style: map[string]string{"position": "absolute", "inset": "0", "width": "1920px", "height": "1080px", "overflow": "hidden"}}, children...)
	}
}

func sceneStageStyle(backgroundURL string) map[string]string {
	// Layers paint top to bottom: overlays, then the scene art, then the solid
	// fallback (the art used to come after the opaque fallback and never showed).
	background := "linear-gradient(180deg, rgba(7,10,16,.16), rgba(5,7,11,.76)), radial-gradient(ellipse at 50% 40%, rgba(42,39,34,.04), rgba(5,7,11,.55) 84%)"
	if backgroundURL = artSrc(backgroundURL); backgroundURL != "" {
		background += ", url('" + backgroundURL + "')"
	}
	background += ", linear-gradient(180deg, #111722, #0b0d12)"
	return map[string]string{"position": "absolute", "inset": "0", "background-image": background, "background-size": "cover", "background-position": "center 42%", "filter": "saturate(.92) contrast(1.04)"}
}

func sceneVignette() ui.Node {
	return html.Div(html.Props{Class: "df-dm-scene-vignette", Style: map[string]string{"position": "absolute", "inset": "0", "z-index": "2", "pointer-events": "none", "box-shadow": "inset 0 0 130px rgba(0,0,0,.68), inset 0 -180px 145px rgba(0,0,0,.66)"}})
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
		style["position"], style["z-index"] = "absolute", "1"
		style["max-width"], style["max-height"] = "52%", "72%"
		nodes = append(nodes, html.Img(html.Props{ID: layer.ID, Class: class, Src: artSrc(layer.URL), Alt: "", Style: style, Raw: map[string]any{"aria-hidden": "true"}}))
	}
	return nodes
}

func sceneBrand(model SceneModel) ui.Node {
	return html.Div(html.Props{Class: "df-dm-scene-brand", Style: map[string]string{"position": "absolute", "left": "34px", "top": "22px", "z-index": "5"}}, CornerBrand())
}

func sceneLocation(model SceneModel) ui.Node {
	return html.Div(html.Props{Style: map[string]string{"position": "absolute", "right": "38px", "top": "28px", "width": "390px", "z-index": "5"}}, LocationTitle("The Drowned Lantern", "Act I · The Tavern"))
}

func sceneTitle(model SceneModel) ui.Node {
	if !model.ShowTitle {
		return html.Div(html.Props{Hidden: true})
	}
	return html.Div(html.Props{Class: "df-dm-scene-title", Style: map[string]string{"position": "absolute", "left": "510px", "top": "438px", "width": "900px", "z-index": "4", "text-align": "center", "pointer-events": "none", "text-shadow": "0 3px 14px rgba(0,0,0,.96)"}},
		html.Div(html.Props{Style: map[string]string{"color": "#e7c27a", "font-family": "Cinzel, Georgia, serif", "font-size": "26px", "font-weight": "600", "letter-spacing": ".34em"}}, ui.Text(model.Act)),
		html.Div(html.Props{Style: map[string]string{"height": "1px", "margin": "12px auto 15px", "width": "380px", "background": "linear-gradient(90deg,transparent,#d9a441,transparent)"}}),
		html.H1(html.Props{Style: map[string]string{"margin": "0", "color": "#efe6d2", "font-family": "Cinzel, 'Cormorant Garamond', Georgia, serif", "font-size": "58px", "font-weight": "500", "line-height": "1.03", "letter-spacing": ".02em"}}, ui.Text(model.Title)),
		html.P(html.Props{Style: map[string]string{"margin": "14px auto 0", "max-width": "800px", "color": "#d7cdbb", "font-family": "Cormorant Garamond, Georgia, serif", "font-size": "27px", "line-height": "1.14"}}, ui.Text(model.Tagline)),
	)
}

func scenePartyRail(characters []SceneCharacter, locale string) ui.Node {
	return html.Div(html.Props{Class: "df-dm-scene-cards", Role: "list", Aria: map[string]string{"label": T(locale, "dm.scene_heroes", nil)}, Style: map[string]string{"position": "absolute", "left": "30px", "top": "150px", "width": "220px", "z-index": "4", "display": "flex", "flex-direction": "column", "gap": "22px", "pointer-events": "none"}}, sceneCharacters(characters, locale)...)
}

func sceneCharacters(characters []SceneCharacter, locale string) []ui.Node {
	nodes := make([]ui.Node, 0, len(characters))
	for _, character := range characters {
		name := character.Name
		if name == "" {
			name = SeatName(locale, "", int(character.PlayerNumber))
		}
		portrait := html.Div(html.Props{Class: "df-dm-scene-card-portrait", Style: map[string]string{"height": "180px", "overflow": "hidden", "background": "radial-gradient(circle at 50% 30%,#4a5564,#111722 70%)"}}, heroPortrait(character.PortraitURL, character.Class, name))
		nodes = append(nodes, html.Div(html.Props{Class: "df-dm-scene-card", Role: "listitem", Style: map[string]string{"overflow": "hidden", "border": "1px solid rgba(217,164,65,.84)", "border-radius": "9px", "background": "linear-gradient(165deg,rgba(16,20,28,.96),rgba(8,10,15,.94))", "box-shadow": "0 10px 30px rgba(0,0,0,.58), inset 0 0 0 1px rgba(239,230,210,.06)", "color": "#efe6d2"}}, portrait, html.Div(html.Props{Style: map[string]string{"padding": "10px 8px 12px", "text-align": "center", "text-shadow": "0 1px 2px #000"}}, html.Strong(html.Props{Style: map[string]string{"display": "block", "font-family": "Cinzel, Georgia, serif", "font-size": "24px", "line-height": "1.05"}}, ui.Text(name)), html.Small(html.Props{Style: map[string]string{"display": "block", "margin-top": "6px", "color": "#c8bda8", "font-family": "Cormorant Garamond, Georgia, serif", "font-size": "17px"}}, ui.Text(character.Class)))))
	}
	return nodes
}

func sceneProgressRail(model SceneModel, locale string) ui.Node {
	steps := make([]ui.Node, 0, len(model.Progress))
	for _, step := range model.Progress {
		color := "#68717b"
		if step.Completed {
			color = "#d9a441"
		}
		if step.Active {
			color = "#efe6d2"
		}
		steps = append(steps, html.Div(html.Props{Style: map[string]string{"display": "flex", "align-items": "center", "gap": "12px", "margin": "14px 0", "color": color, "font-family": "Cormorant Garamond, Georgia, serif", "font-size": "22px"}}, html.Span(html.Props{Role: "img", Aria: map[string]string{"label": step.Label}, Style: map[string]string{"width": "20px", "height": "20px", "flex": "0 0 20px", "border": "2px solid " + color, "border-radius": "50%", "box-shadow": "0 0 9px " + color}}), html.Span(html.Props{Style: map[string]string{"line-height": "1.05", "font-weight": boolWeight(step.Active)}}, ui.Text(step.Label))))
	}
	return html.Div(html.Props{Class: "df-dm-scene-progress", Aria: map[string]string{"label": T(locale, "dm.scene_label", nil)}, Style: map[string]string{"position": "absolute", "right": "36px", "top": "174px", "width": "290px", "height": "560px", "z-index": "4", "padding": "24px 20px", "border": "1px solid rgba(217,164,65,.76)", "border-radius": "5px", "background": "linear-gradient(145deg,rgba(8,15,23,.9),rgba(5,9,14,.92))", "box-shadow": "0 12px 32px rgba(0,0,0,.62)", "color": "#efe6d2"}}, html.H2(html.Props{Style: map[string]string{"margin": "0 0 18px", "font-family": "Cormorant Garamond, Georgia, serif", "font-size": "26px", "font-weight": "500"}}, ui.Text(T(locale, "dm.scene.current", nil))), html.Div(html.Props{Style: map[string]string{"height": "125px", "margin-bottom": "18px", "border": "1px solid rgba(217,164,65,.32)", "background-image": sceneThumbBackground(model.BackgroundURL), "background-size": "cover", "background-position": "center", "filter": "saturate(.82)"}}), html.Div(html.Props{Style: map[string]string{"height": "1px", "margin": "0 0 14px", "background": "linear-gradient(90deg,#d9a441,transparent)"}}), html.Div(html.Props{Style: map[string]string{"color": "#efe6d2", "font-family": "Cormorant Garamond, Georgia, serif", "font-size": "24px", "line-height": "1.1"}}, ui.Text(SceneTitle(locale))), html.Div(html.Props{Style: map[string]string{"margin-top": "18px"}}, steps...))
}

func sceneCaption(caption SceneCaption, portraitURL, frameURL, dividerURL, locale string) ui.Node {
	if !caption.Visible {
		return html.Div(html.Props{Hidden: true})
	}
	name := SceneSpeaker(locale, caption.Speaker)
	portraitURL = artSrc(portraitURL)
	portrait := html.Div(html.Props{Class: "df-dm-scene-speaker", Style: map[string]string{"position": "absolute", "left": "-104px", "top": "30px", "width": "150px", "height": "150px", "overflow": "hidden", "border": "2px solid #d9a441", "border-radius": "50%", "background": "radial-gradient(circle at 50% 38%,#2b3542,#07090d 70%)", "box-shadow": "0 0 0 5px rgba(15,17,23,.92), 0 8px 24px rgba(0,0,0,.65)"}}, html.Img(html.Props{Src: portraitURL, Alt: name, Hidden: portraitURL == "", Style: map[string]string{"width": "100%", "height": "100%", "object-fit": "cover"}}), html.Span(html.Props{Hidden: portraitURL != "", Style: map[string]string{"display": "grid", "place-items": "center", "width": "100%", "height": "100%", "color": "#d9a441", "font-size": "42px"}}, ui.Text(T("en", "dm.glyph.star", nil))))
	_ = frameURL
	_ = dividerURL
	panel := OrnatePanel("", SpeakerCaption(CaptionModel{Speaker: name, Text: strings.TrimSpace(caption.Text)}))
	return html.Div(html.Props{Class: "df-dm-scene-caption", Role: "status", Aria: map[string]string{"live": "polite"}, Style: map[string]string{"position": "absolute", "left": "560px", "bottom": "92px", "width": "920px", "min-height": "205px", "z-index": "5", "padding": "0 20px", "pointer-events": "none"}}, portrait, panel)
}

func sceneThumbBackground(url string) string {
	url = artSrc(url)
	if url == "" {
		return "linear-gradient(145deg,#243348,#10141d)"
	}
	return "linear-gradient(180deg,rgba(12,16,22,.16),rgba(12,16,22,.66)),url('" + url + "')"
}

func boolWeight(active bool) string {
	if active {
		return "700"
	}
	return "400"
}

// heroPortrait shows the hero portrait, or while it is missing (fake mode, or
// still generating) the class art, then the lantern emblem; never an empty img.
func heroPortrait(portraitURL, className, name string) ui.Node {
	src := artSrc(portraitURL)
	if src == "" {
		src = heroProxyArt("", className, name)
	}
	if src == "" {
		return html.Span(html.Props{Aria: map[string]string{"label": name}, Style: map[string]string{"display": "grid", "place-items": "center", "height": "100%", "color": "#d9a441", "font-size": "40px"}}, ui.Text(T("en", "dm.glyph.star", nil)))
	}
	return html.Img(html.Props{Src: src, Alt: name, Style: map[string]string{"width": "100%", "height": "100%", "object-fit": "cover"}})
}
