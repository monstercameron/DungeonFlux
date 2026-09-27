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
			dialoguePartyForeground(model.Party),
			dialogueCaption(model),
			dialogueChoices(model.Options),
		)
	}
}

// dialogueStyle darkens only the edges and the bottom choice band so the bar
// itself (the concept's "full brightness" read) stays visible through the
// scene layer beneath; the atmosphere overlay adds its own gentle vignette.
func dialogueStyle() map[string]string {
	return map[string]string{
		"position": "absolute", "inset": "0", "z-index": "6", "overflow": "hidden", "pointer-events": "none",
		"background": "linear-gradient(90deg, rgba(8,10,15,.5), rgba(8,10,15,.1) 17%, transparent 29%, transparent 71%, rgba(8,10,15,.1) 83%, rgba(8,10,15,.5)), linear-gradient(180deg, transparent 0%, transparent 52%, rgba(8,10,15,.1) 60%, rgba(8,10,15,.62) 76%, rgba(8,10,15,.94) 90%, #090b10 100%)",
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
		html.Div(html.Props{Class: "df-dm-dialogue-choice-icon", Style: map[string]string{"display": "flex", "align-items": "center", "justify-content": "center", "width": "64px", "height": "64px", "flex": "0 0 auto", "border": "1px solid " + border, "border-radius": "8px", "background": "rgba(6,10,15,.55)", "box-shadow": "inset 0 0 0 1px rgba(239,230,210,.05)", "color": "#e8b651", "font-size": "28px"}}, dialogueIcon(option)),
		html.Div(html.Props{Style: map[string]string{"min-width": "0", "text-align": "left"}},
			html.Strong(html.Props{Class: "df-dm-dialogue-choice-label", Style: map[string]string{"display": "block", "color": "#f3ead8", "font-family": "Cormorant Garamond, Georgia, serif", "font-size": "30px", "font-weight": "600", "line-height": "1.05", "white-space": "nowrap"}}, ui.Text(option.Text)),
			dialogueDetail(option),
		),
	}
	return html.Div(html.Props{Class: "df-dm-dialogue-choice", Role: "listitem", Style: map[string]string{"display": "flex", "align-items": "center", "gap": "16px", "height": "88px", "padding": "12px 18px", "border": "1px solid " + border, "border-radius": "4px", "background": background, "box-shadow": "0 8px 24px rgba(0,0,0,.28), inset 0 1px 0 rgba(239,230,210,.06)", "opacity": opacity(option.Enabled), "pointer-events": "none"}, Raw: map[string]any{"data-choice": option.ID, "data-index": index, "data-enabled": option.Enabled}}, copy...)
}

// dialogueIcon shows the move's icon glyph at full legible size: contained,
// not cropped to a circle, so a non-square icon asset never reads as a blob.
func dialogueIcon(option DialogueOption) ui.Node {
	if iconURL := artSrc(option.IconURL); iconURL != "" {
		return html.Img(html.Props{Src: iconURL, Alt: "", Style: map[string]string{"width": "72%", "height": "72%", "object-fit": "contain"}, Raw: map[string]any{"aria-hidden": "true"}})
	}
	return ui.Text(T("en", "dm.glyph.star", nil))
}

func dialogueDetail(option DialogueOption) ui.Node {
	if option.Detail == "" && option.Enabled {
		return html.Span(html.Props{Hidden: true})
	}
	detail := option.Detail
	if detail == "" {
		detail = option.Reason
	}
	// The DC line reads as a rules callout, so it takes the display face
	// (Cinzel), not the panel's Arial fallback (dm.glyph.star sits in the
	// same family for a consistent voice).
	return html.Small(html.Props{Class: "df-dm-dialogue-choice-detail", Style: map[string]string{
		"display": "block", "margin-top": "5px", "color": "#e0c07f", "font-family": "Cinzel,'Cormorant Garamond',Georgia,serif",
		"font-size": "17px", "letter-spacing": ".04em", "line-height": "1.1",
	}}, ui.Text(detail))
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
// art includes its own background, so a radial mask blends it in; a ground
// shadow anchors her to the floor and a cool rim light on her near edge (the
// same fog-teal used across the scene) separates her from the backdrop
// without a hard cut. The server's own scene layer for this NPC (the small
// composited portrait behind her) is hidden by df-mood CSS so only this one
// large figure reads — otherwise the two overlap as a "ghost" double.
func dialogueFigure(npcName string) ui.Node {
	asset := "stranger"
	if strings.Contains(strings.ToLower(npcName), "vell") {
		asset = "mother_vell"
	}
	url := ArtURL(asset)
	if url == "" {
		return html.Div(html.Props{Class: "df-dm-dialogue-figure-wrap", Hidden: true})
	}
	mask := "radial-gradient(ellipse 50% 52% at 50% 46%, #000 58%, transparent 100%)"
	return html.Div(html.Props{Class: "df-dm-dialogue-figure-wrap", Aria: map[string]string{"hidden": "true"}, Style: map[string]string{
		"position": "absolute", "right": "170px", "top": "70px", "width": "660px", "height": "720px", "z-index": "7",
	}},
		html.Div(html.Props{Class: "df-dm-dialogue-figure-shadow"}),
		html.Div(html.Props{Class: "df-dm-dialogue-figure", Style: map[string]string{
			"position": "absolute", "inset": "0",
			"background-image": "url('" + url + "')", "background-size": "cover", "background-position": "center 6%",
			"-webkit-mask-image": mask, "mask-image": mask, "filter": "saturate(1.05) contrast(1.05)",
		}}),
		html.Div(html.Props{Class: "df-dm-dialogue-figure-rim", Style: map[string]string{
			"position": "absolute", "inset": "0", "-webkit-mask-image": mask, "mask-image": mask,
		}}),
	)
}

// dialoguePartyForeground composites the party at the bar, left of the NPC,
// so the concept's three-figure read (both heroes plus the barkeep) survives
// on the TV even before generated group art exists. Portraits fall back to
// class or species art, then a glyph silhouette, through heroPortrait.
func dialoguePartyForeground(party []DialoguePartyMember) ui.Node {
	if len(party) == 0 {
		return html.Div(html.Props{Class: "df-dm-dialogue-party", Hidden: true})
	}
	members := make([]ui.Node, 0, len(party))
	for _, member := range party {
		members = append(members, dialoguePartyMember(member))
	}
	return html.Div(html.Props{Class: "df-dm-dialogue-party", Role: "list", Aria: map[string]string{"label": "The party at the bar"}, Style: map[string]string{
		"position": "absolute", "left": "70px", "bottom": "250px", "z-index": "6",
		"display": "flex", "align-items": "flex-end", "gap": "28px", "pointer-events": "none",
	}}, members...)
}

func dialoguePartyMember(member DialoguePartyMember) ui.Node {
	mask := "linear-gradient(180deg,#000 76%,transparent 100%)"
	return html.Div(html.Props{Class: "df-dm-dialogue-party-member", Role: "listitem", Aria: map[string]string{"label": member.Name}, Style: map[string]string{
		"position": "relative", "width": "190px", "height": "370px", "overflow": "hidden",
		"-webkit-mask-image": mask, "mask-image": mask, "filter": "saturate(.97) contrast(1.02) brightness(.9)",
	}},
		heroPortrait(member.PortraitURL, member.Class, member.Name),
		html.Div(html.Props{Class: "df-dm-dialogue-party-shadow"}),
	)
}
