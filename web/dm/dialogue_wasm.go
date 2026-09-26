//go:build js && wasm

package dm

import (
	"strconv"

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
			dialoguePortrait(model),
			dialogueCopy(model),
			dialogueChoices(model.Options),
		)
	}
}

func dialogueStyle() map[string]string {
	return map[string]string{
		"position": "absolute", "inset": "0", "z-index": "6", "display": "flex", "flex-direction": "column",
		"justify-content": "flex-end", "padding": "0 3.5% 3.5%", "overflow": "hidden", "pointer-events": "none",
		"background": "linear-gradient(180deg, transparent 37%, rgba(8,10,15,.06) 44%, rgba(8,10,15,.56) 58%, rgba(8,10,15,.94) 73%, #090b10 100%)",
		"color":      "#efe6d2", "font-family": "Arial, sans-serif",
	}
}

func dialoguePortrait(model DialogueModel) ui.Node {
	style := map[string]string{
		"position": "absolute", "right": "3.5%", "bottom": "25%", "width": "clamp(180px, 22vw, 430px)", "height": "55%",
		"display": "flex", "align-items": "flex-end", "justify-content": "center", "overflow": "hidden", "border-radius": "50% 50% 8px 8px",
		"background": "radial-gradient(ellipse at 50% 35%, rgba(90,65,42,.75), rgba(15,17,23,.92) 70%)",
		"box-shadow": "0 0 42px rgba(217,164,65,.16), inset 0 0 30px rgba(0,0,0,.55)", "opacity": ".96",
	}
	if model.PortraitURL != "" {
		return html.Div(html.Props{Class: "df-dm-dialogue-portrait", Style: style}, html.Img(html.Props{Src: model.PortraitURL, Alt: model.NPCName, Style: map[string]string{"width": "100%", "height": "100%", "object-fit": "contain", "object-position": "center bottom"}}))
	}
	return html.Div(html.Props{Class: "df-dm-dialogue-portrait is-fallback", Style: style, Aria: map[string]string{"hidden": "true"}},
		html.Span(html.Props{Style: map[string]string{"margin-bottom": "18%", "color": "rgba(217,164,65,.8)", "font-family": "Georgia,serif", "font-size": "clamp(3rem,7vw,8rem)", "text-shadow": "0 0 28px rgba(217,164,65,.32)"}}, ui.Text("✦")),
	)
}

func dialogueCopy(model DialogueModel) ui.Node {
	speaker := model.Speaker
	if speaker == "" {
		speaker = model.NPCName
	}
	line := model.Line
	if line == "" {
		line = "Mother Vell is listening."
	}
	return html.Div(html.Props{Class: "df-dm-dialogue-copy", Style: map[string]string{"position": "relative", "z-index": "2", "width": "min(82%, 1160px)", "margin": "0 auto clamp(1rem,2.2vh,2rem)", "text-align": "center", "text-shadow": "0 2px 5px rgba(0,0,0,.82)"}},
		html.Strong(html.Props{Class: "df-dm-dialogue-speaker", Style: map[string]string{"display": "block", "color": "#e9b954", "font-family": "Georgia,serif", "font-size": "clamp(1.05rem,1.8vw,2.05rem)", "font-weight": "600", "letter-spacing": ".02em"}}, ui.Text(speaker)),
		html.Div(html.Props{Class: "df-dm-dialogue-ornament", Style: map[string]string{"display": "flex", "align-items": "center", "gap": "1rem", "margin": ".2rem auto .65rem", "color": "rgba(217,164,65,.78)", "font-size": "clamp(.8rem,1vw,1.2rem)"}},
			html.Span(html.Props{Style: map[string]string{"height": "1px", "flex": "1", "background": "linear-gradient(90deg, transparent, rgba(217,164,65,.72))"}}),
			ui.Text("✦"),
			html.Span(html.Props{Style: map[string]string{"height": "1px", "flex": "1", "background": "linear-gradient(90deg, rgba(217,164,65,.72), transparent)"}}),
		),
		html.P(html.Props{Class: "df-dm-dialogue-line", Style: map[string]string{"margin": "0", "color": "#f4ecdd", "font-family": "Georgia, 'Times New Roman', serif", "font-size": "clamp(1.35rem,2.35vw,2.7rem)", "font-weight": "600", "line-height": "1.16", "text-wrap": "balance"}}, ui.Text(line)),
	)
}

func dialogueChoices(options []DialogueOption) ui.Node {
	children := make([]ui.Node, 0, len(options))
	for index, option := range options {
		children = append(children, dialogueChoice(index, option))
	}
	return html.Div(html.Props{Class: "df-dm-dialogue-choices", Role: "list", Aria: map[string]string{"label": "Available conversation choices"}, Style: map[string]string{"position": "relative", "z-index": "3", "display": "grid", "grid-template-columns": "repeat(" + strconv.Itoa(max(1, len(options))) + ", minmax(0, 1fr))", "gap": "clamp(.75rem,1.7vw,2rem)", "width": "min(94%, 1540px)", "margin": "0 auto"}}, children...)
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
		html.Div(html.Props{Class: "df-dm-dialogue-choice-icon", Style: map[string]string{"display": "flex", "align-items": "center", "justify-content": "center", "width": "clamp(2rem,3vw,3.6rem)", "height": "clamp(2rem,3vw,3.6rem)", "flex": "0 0 auto", "border": "1px solid " + border, "border-radius": "50%", "color": "#e8b651", "font-size": "clamp(1rem,1.5vw,1.7rem)"}}, dialogueIcon(option)),
		html.Div(html.Props{Style: map[string]string{"min-width": "0", "text-align": "left"}},
			html.Strong(html.Props{Class: "df-dm-dialogue-choice-label", Style: map[string]string{"display": "block", "color": "#f3ead8", "font-family": "Georgia,serif", "font-size": "clamp(1rem,1.45vw,1.7rem)", "font-weight": "600", "line-height": "1.14"}}, ui.Text(option.Text)),
			dialogueDetail(option),
		),
	}
	return html.Div(html.Props{Class: "df-dm-dialogue-choice", Role: "listitem", Style: map[string]string{"display": "flex", "align-items": "center", "gap": "clamp(.7rem,1.1vw,1.2rem)", "min-height": "clamp(4.5rem,8vh,6.5rem)", "padding": "clamp(.7rem,1.25vw,1.35rem)", "border": "1px solid " + border, "border-radius": "8px", "background": background, "box-shadow": "0 8px 24px rgba(0,0,0,.28), inset 0 1px 0 rgba(239,230,210,.06)", "opacity": opacity(option.Enabled), "pointer-events": "none"}, Raw: map[string]any{"data-choice": option.ID, "data-index": index, "data-enabled": option.Enabled}}, copy...)
}

func dialogueIcon(option DialogueOption) ui.Node {
	if option.IconURL != "" {
		return html.Img(html.Props{Src: option.IconURL, Alt: "", Style: map[string]string{"width": "100%", "height": "100%", "object-fit": "cover", "border-radius": "50%"}, Raw: map[string]any{"aria-hidden": "true"}})
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
	return html.Small(html.Props{Class: "df-dm-dialogue-choice-detail", Style: map[string]string{"display": "block", "margin-top": ".25rem", "color": "#b9ae98", "font-size": "clamp(.76rem,1vw,1.05rem)", "line-height": "1.15"}}, ui.Text(detail))
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
