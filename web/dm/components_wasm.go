//go:build js && wasm

package dm

import (
	"strconv"

	"github.com/monstercameron/GoWebComponents/v6/html"
	"github.com/monstercameron/GoWebComponents/v6/router"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

// PortraitCardModel contains the readable identity shown on a TV card.
type PortraitCardModel struct {
	Name        string
	Subtitle    string
	Flavor      string
	PortraitURL string
	Joined      bool
	Ready       bool
	HP          int32
	HPMax       int32
}

// CaptionModel contains one lower-third speaker line.
type CaptionModel struct {
	Speaker string
	Text    string
}

// ActionButtonModel contains one display-only legal action.
type ActionButtonModel struct {
	Icon    string
	Label   string
	Hotkey  string
	Primary bool
	Enabled bool
	Reason  string
}

// OrnatePanel renders the shared translucent navy panel with a gold frame.
func OrnatePanel(title string, children ...ui.Node) ui.Node {
	content := make([]ui.Node, 0, len(children)+1)
	if title != "" {
		content = append(content, html.H2(html.Props{Class: "df-ornate-panel-title"}, ui.Text(title)))
	}
	content = append(content, children...)
	return html.Section(html.Props{Class: "df-ornate-panel", Role: "region"}, content...)
}

// TitlePlate renders the wordmark and the small fantasy subtitle.
func TitlePlate(wordmarkURL, title, subtitle string) ui.Node {
	var mark ui.Node
	if wordmarkURL != "" {
		mark = html.Img(html.Props{Class: "df-title-plate-wordmark", Src: wordmarkURL, Alt: title})
	} else {
		mark = html.H1(html.Props{Class: "df-title-plate-fallback"}, ui.Text(title))
	}
	return html.Header(html.Props{Class: "df-title-plate"}, mark, html.P(html.Props{Class: "df-title-plate-subtitle"}, ui.Text(subtitle)))
}

// GoldButton renders a highlighted status or primary-action plate.
func GoldButton(label string) ui.Node {
	return html.Div(html.Props{Class: "df-gold-button", Role: "status"}, ui.Text(label), html.Span(html.Props{Class: "df-button-chevron", Aria: map[string]string{"hidden": "true"}}, ui.Text("›")))
}

// DarkButton renders a restrained information row with an optional icon.
func DarkButton(icon, label string) ui.Node {
	return html.Div(html.Props{Class: "df-dark-button"}, html.Span(html.Props{Class: "df-dark-button-icon", Aria: map[string]string{"hidden": "true"}}, ui.Text(icon)), html.Span(html.Props{Class: "df-dark-button-label"}, ui.Text(label)))
}

// PortraitCard renders a party seat, including a dim empty-seat state.
func PortraitCard(model PortraitCardModel) ui.Node {
	name := model.Name
	if name == "" {
		name = "Waiting for a player…"
	}
	portrait := model.PortraitURL
	if portrait == "" {
		portrait = ArtURL("ui/logo_emblem")
	}
	portraitNode := html.Div(html.Props{Class: "df-portrait-card-portrait", Aria: map[string]string{"label": name}},
		html.Img(html.Props{Src: portrait, Alt: "", Hidden: portrait == "", Raw: map[string]any{"aria-hidden": "true"}}),
		html.Span(html.Props{Class: "df-portrait-card-silhouette", Hidden: portrait != "", Aria: map[string]string{"hidden": "true"}}, ui.Text("✦")),
	)
	children := []ui.Node{portraitNode, html.Div(html.Props{Class: "df-portrait-card-copy"},
		html.Strong(html.Props{Class: "df-portrait-card-name"}, ui.Text(name)),
		html.Span(html.Props{Class: "df-portrait-card-subtitle"}, ui.Text(model.Subtitle)),
	)}
	if model.Flavor != "" {
		children = append(children, html.P(html.Props{Class: "df-portrait-card-flavor"}, ui.Text("“"+model.Flavor+"”")))
	}
	if model.HPMax > 0 {
		children = append(children, html.Span(html.Props{Class: "df-portrait-card-hp"}, ui.Text("HP "+strconv.Itoa(int(model.HP))+"/"+strconv.Itoa(int(model.HPMax)))))
	}
	return html.Article(html.Props{Class: "df-portrait-card", Role: "listitem", Raw: map[string]any{"data-joined": model.Joined, "data-ready": model.Ready}}, children...)
}

// SpeakerCaption renders a centered speaker name and parchment line.
func SpeakerCaption(model CaptionModel) ui.Node {
	return html.Div(html.Props{Class: "df-speaker-caption", Role: "status", Aria: map[string]string{"live": "polite"}},
		html.Strong(html.Props{Class: "df-speaker-caption-name"}, ui.Text(model.Speaker)),
		html.P(html.Props{Class: "df-speaker-caption-text"}, ui.Text(model.Text)),
		html.Div(html.Props{Class: "df-speaker-caption-ornament", Aria: map[string]string{"hidden": "true"}}, ui.Text("✦")),
	)
}

// ActionButton renders one action tile with an optional selected state.
func ActionButton(model ActionButtonModel) ui.Node {
	class := "df-action-button"
	if model.Primary {
		class += " is-primary"
	}
	if !model.Enabled {
		class += " is-disabled"
	}
	return html.Div(html.Props{Class: class, Role: "listitem", Aria: map[string]string{"label": model.Label}, Raw: map[string]any{"data-reason": model.Reason}},
		html.Span(html.Props{Class: "df-action-button-icon", Aria: map[string]string{"hidden": "true"}}, ui.Text(model.Icon)),
		html.Span(html.Props{Class: "df-action-button-label"}, ui.Text(model.Label)),
		html.Span(html.Props{Class: "df-action-button-hotkey", Hidden: model.Hotkey == ""}, ui.Text(model.Hotkey)),
	)
}

// LocationTitle renders the top-right scene location and subtitle.
func LocationTitle(title, subtitle string) ui.Node {
	return html.Div(html.Props{Class: "df-location-title"},
		html.Strong(html.Props{Class: "df-location-title-name"}, ui.Text(title)),
		html.Div(html.Props{Class: "df-location-title-rule", Aria: map[string]string{"hidden": "true"}}),
		html.Span(html.Props{Class: "df-location-title-subtitle"}, ui.Text(subtitle)),
	)
}

// SharedComponent is a small named factory used by screen workers.
type SharedComponent func(router.Attrs) *router.Element
