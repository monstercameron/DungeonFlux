//go:build js && wasm

package dm

import (
	"strconv"
	"strings"

	"github.com/monstercameron/GoWebComponents/v6/html"
	"github.com/monstercameron/GoWebComponents/v6/router"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

// DialogueComponent renders the TV conversation panel and display-only move
// choices in the visual language of the tavern concept.
func DialogueComponent(model DialogueModel) router.Component {
	return func(_ router.Attrs) *router.Element {
		if !model.Visible {
			return html.Section(html.Props{Class: "df-dm-dialogue", Hidden: true})
		}
		return html.Section(html.Props{Class: "df-dm-dialogue", Role: "region", Aria: map[string]string{"label": "Conversation with " + model.NPCName}, Style: dialogueStyle()},
			dialogueTitlePlate(),
			dialogueLocation(model.Locale),
			dialogueFigure(model.NPCName),
			dialogueCaption(model),
			dialogueChoices(model.Options),
		)
	}
}

func dialogueStyle() map[string]string {
	return map[string]string{
		"position": "absolute", "inset": "0", "z-index": "6", "overflow": "hidden", "pointer-events": "none",
		"background": "linear-gradient(90deg, rgba(8,10,15,.72), rgba(8,10,15,.18) 17%, transparent 29%, transparent 71%, rgba(8,10,15,.18) 83%, rgba(8,10,15,.72)), linear-gradient(180deg, rgba(8,10,15,.08) 0%, transparent 48%, rgba(8,10,15,.16) 56%, rgba(8,10,15,.72) 73%, rgba(8,10,15,.97) 88%, #090b10 100%)",
		"color":      "#efe6d2", "font-family": "Arial, sans-serif",
	}
}

func dialogueTitlePlate() ui.Node {
	return html.Div(html.Props{Class: "df-dm-dialogue-title-plate", Style: map[string]string{"position": "absolute", "left": "34px", "top": "22px", "z-index": "8"}}, CornerBrand())
}

func dialogueLocation(locale string) ui.Node {
	return html.Div(html.Props{Class: "df-dm-dialogue-location", Style: map[string]string{
		"position": "absolute", "left": "1480px", "top": "24px", "width": "410px", "height": "90px", "z-index": "8",
	}}, LocationTitle(SceneTitle(locale), "Act I · The Tavern"))
}

func dialogueCaption(model DialogueModel) ui.Node {
	speaker := model.Speaker
	if speaker == "" {
		speaker = model.NPCName
	}
	line := model.Line
	if line == "" {
		line = "Mother Vell is listening."
	}
	return html.Div(html.Props{Class: "df-dm-dialogue-caption", Style: map[string]string{
		"position": "absolute", "left": "350px", "top": "760px", "width": "1220px", "min-height": "130px", "z-index": "8",
	}}, SpeakerCaption(CaptionModel{Speaker: speaker, Text: line}))
}

func dialogueChoices(options []DialogueOption) ui.Node {
	children := make([]ui.Node, 0, len(options))
	for index, option := range options {
		children = append(children, dialogueChoice(index, option))
	}
	return html.Div(html.Props{Class: "df-dm-dialogue-choices", Role: "list", Aria: map[string]string{"label": "Available conversation choices"}, Style: map[string]string{
		"position": "absolute", "left": "370px", "top": "905px", "z-index": "8", "display": "grid",
		"grid-template-columns": "repeat(" + strconv.Itoa(max(1, len(options))) + ", minmax(0, 1fr))", "gap": "30px", "width": "1180px", "height": "88px",
	}}, children...)
}

func dialogueChoice(index int, option DialogueOption) ui.Node {
	border := "rgba(239,230,210,.28)"
	background := "linear-gradient(135deg, rgba(12,14,19,.9), rgba(25,27,34,.82))"
	if option.Primary {
		border = "#e4ae4c"
		background = "linear-gradient(135deg, rgba(52,38,18,.96), rgba(24,23,22,.92))"
	}
	if !option.Enabled {
		border = "rgba(168,159,140,.22)"
		background = "rgba(15,17,23,.76)"
	}
	copy := []ui.Node{
		html.Div(html.Props{Class: "df-dm-dialogue-choice-icon", Style: map[string]string{"display": "flex", "align-items": "center", "justify-content": "center", "width": "48px", "height": "48px", "flex": "0 0 auto", "border": "1px solid " + border, "border-radius": "7px", "color": "#e8b651", "font-size": "26px"}}, dialogueIcon(option)),
		html.Div(html.Props{Style: map[string]string{"min-width": "0", "text-align": "left"}},
			html.Strong(html.Props{Class: "df-dm-dialogue-choice-label", Style: map[string]string{"display": "block", "color": "#f3ead8", "font-family": "Cormorant Garamond, Georgia, serif", "font-size": "30px", "font-weight": "600", "line-height": "1.05", "white-space": "nowrap"}}, ui.Text(option.Text)),
			dialogueDetail(option),
		),
	}
	return html.Div(html.Props{Class: "df-dm-dialogue-choice", Role: "listitem", Style: map[string]string{"display": "flex", "align-items": "center", "gap": "16px", "height": "88px", "padding": "12px 18px", "border": "1px solid " + border, "border-radius": "4px", "background": background, "box-shadow": "0 8px 24px rgba(0,0,0,.28), inset 0 1px 0 rgba(239,230,210,.06)", "opacity": opacity(option.Enabled), "pointer-events": "none"}, Raw: map[string]any{"data-choice": option.ID, "data-index": index, "data-enabled": option.Enabled}}, copy...)
}

func dialogueIcon(option DialogueOption) ui.Node {
	if iconURL := artSrc(option.IconURL); iconURL != "" {
		return html.Img(html.Props{Src: iconURL, Alt: "", Style: map[string]string{"width": "100%", "height": "100%", "object-fit": "cover", "border-radius": "50%"}, Raw: map[string]any{"aria-hidden": "true"}})
	}
	return ui.Text("✦")
}

func dialogueDetail(option DialogueOption) ui.Node {
	if option.Detail == "" && option.Enabled {
		return html.Span(html.Props{Hidden: true})
	}
	detail := option.Detail
	if detail == "" {
		detail = option.Reason
	}
	return html.Small(html.Props{Class: "df-dm-dialogue-choice-detail", Style: map[string]string{"display": "block", "margin-top": "4px", "color": "#b9ae98", "font-size": "18px", "line-height": "1.1"}}, ui.Text(detail))
}

func opacity(enabled bool) string {
	if enabled {
		return "1"
	}
	return ".58"
}

func max(left, right int) int {
	if left > right {
		return left
	}
	return right
}

// dialogueFigure shows the speaking NPC as a large portrait feathered into the
// scene (the concept frames the NPC across the right of the TV). The portrait
// art includes its own background, so a radial mask blends it in.
func dialogueFigure(npcName string) ui.Node {
	asset := "stranger"
	if strings.Contains(strings.ToLower(npcName), "vell") {
		asset = "mother_vell"
	}
	url := ArtURL(asset)
	if url == "" {
		return html.Div(html.Props{Class: "df-dm-dialogue-figure", Hidden: true})
	}
	mask := "radial-gradient(ellipse 50% 52% at 50% 46%, #000 58%, transparent 100%)"
	return html.Div(html.Props{Class: "df-dm-dialogue-figure", Aria: map[string]string{"hidden": "true"}, Style: map[string]string{
		"position": "absolute", "right": "170px", "top": "70px", "width": "660px", "height": "720px", "z-index": "7",
		"background-image": "url('" + url + "')", "background-size": "cover", "background-position": "center 6%",
		"-webkit-mask-image": mask, "mask-image": mask, "filter": "saturate(1.05) contrast(1.05)",
	}})
}
