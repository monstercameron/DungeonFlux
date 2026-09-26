//go:build js && wasm

package dm

import (
	"fmt"
	"strconv"
	"strings"

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
	return GlassPanel(title, children...)
}

// GlassPanel renders the shared translucent navy panel with an ornate frame.
func GlassPanel(title string, children ...ui.Node) ui.Node {
	content := make([]ui.Node, 0, len(children)+1)
	if title != "" {
		content = append(content, html.H2(html.Props{Class: "df-ornate-panel-title"}, ui.Text(title)))
	}
	content = append(content, children...)
	return html.Section(html.Props{Class: "df-ornate-panel df-glass-panel", Role: "region"}, content...)
}

// TitlePlate renders the wordmark and the small fantasy subtitle.
func TitlePlate(wordmarkURL, title, subtitle string) ui.Node {
	var mark ui.Node
	if wordmarkURL != "" {
		// The wordmark art sits on a black plate. A luminance mask of the image
		// on itself drops the plate; mix-blend-mode cannot, because the scaled
		// canvas is its own stacking context and never sees the cover art.
		art := "url('" + wordmarkURL + "')"
		mark = html.Div(html.Props{Class: "df-title-plate-wordmark df-title-wordmark df-wordmark-art", Role: "img", Aria: map[string]string{"label": title}, Style: map[string]string{
			"background-image": art, "-webkit-mask-image": art, "mask-image": art, "mask-mode": "luminance",
		}})
	} else {
		mark = html.H1(html.Props{Class: "df-title-plate-fallback df-title-fallback"}, ui.Text(title))
	}
	return html.Header(html.Props{Class: "df-title-plate"}, mark, html.P(html.Props{Class: "df-title-plate-subtitle df-title-subtitle"}, ui.Text(subtitle)))
}

// GoldButton renders a highlighted status or primary-action plate.
func GoldButton(label string) ui.Node {
	return GoldPlateButton("", label)
}

// GoldPlateButton renders a burnished gold primary row with an optional icon.
func GoldPlateButton(icon, label string) ui.Node {
	return html.Div(html.Props{Class: "df-gold-button df-gold-plate-button", Role: "status"},
		html.Span(html.Props{Class: "df-menu-row-icon", Aria: map[string]string{"hidden": "true"}}, ui.Text(icon)),
		html.Span(html.Props{Class: "df-menu-row-label"}, ui.Text(label)),
		html.Span(html.Props{Class: "df-button-chevron", Aria: map[string]string{"hidden": "true"}}, ui.Text(T("en", "dm.glyph.chevron", nil))))
}

// DarkButton renders a restrained information row with an optional icon.
func DarkButton(icon, label string) ui.Node {
	return MenuRow(icon, label)
}

// MenuRow renders a beveled dark-glass menu row with a gold icon.
func MenuRow(icon, label string) ui.Node {
	return html.Div(html.Props{Class: "df-dark-button df-menu-row"},
		html.Span(html.Props{Class: "df-dark-button-icon df-menu-row-icon", Aria: map[string]string{"hidden": "true"}}, ui.Text(icon)),
		html.Span(html.Props{Class: "df-dark-button-label df-menu-row-label"}, ui.Text(label)))
}

// FeatureIcon renders a large line-art icon for a feature tile.
func FeatureIcon(name string) ui.Node {
	path := map[string]string{
		"book":     "M4 5.5C7 4 10 4 12 5.5V19c-2-1.5-5-1.5-8 0V5.5Zm16 0C17 4 14 4 12 5.5V19c2-1.5 5-1.5 8 0V5.5ZM12 5.5v13.5",
		"waveform": "M2 12h3l2-7 3 14 3-18 3 18 2-7h3",
		"die":      "M12 2.5 20.5 7v10L12 21.5 3.5 17V7L12 2.5Zm0 0v9.5m8.5-5L12 12 3.5 7M12 12v9.5",
		"brain":    "M9 4.5A3.5 3.5 0 0 0 5.8 7 3.5 3.5 0 0 0 5 13.5 3.5 3.5 0 0 0 8 19h2V5.2a3 3 0 0 0-1-.7Zm6 0A3.5 3.5 0 0 1 18.2 7a3.5 3.5 0 0 1 .8 6.5A3.5 3.5 0 0 1 16 19h-2V5.2a3 3 0 0 1 1-.7ZM6 10h4m4 0h4M7 15h3m4 0h3",
		"music":    "M9 18V5l10-2v13M9 18a3 3 0 1 1-3-3 3 3 0 0 1 3 3Zm10-2a3 3 0 1 1-3-3 3 3 0 0 1 3 3Z",
		"clapper":  "M3 6h18v13H3V6Zm0 4h18M6 3l3 3m1-3 3 3m1-3 3 3",
	}
	d, ok := path[strings.ToLower(strings.TrimSpace(name))]
	if !ok {
		d = path["die"]
	}
	return html.Svg(html.Props{Class: "df-feature-icon", Width: "58", Height: "58", Raw: map[string]any{"viewBox": "0 0 24 24", "aria-hidden": "true"}}, html.Path(html.Props{Raw: map[string]any{"d": d, "fill": "none", "stroke": "currentColor", "stroke-linecap": "round", "stroke-linejoin": "round", "stroke-width": "1.35"}}))
}

func featureTile(name, label string) ui.Node {
	if !strings.Contains("book waveform die brain music clapper", strings.ToLower(strings.TrimSpace(name))) {
		switch strings.ToLower(strings.TrimSpace(label)) {
		case "ai narration":
			name = "book"
		case "voice npcs":
			name = "waveform"
		case "rules engine":
			name = "die"
		case "character memory":
			name = "brain"
		case "dynamic music":
			name = "music"
		case "cliffhanger clips":
			name = "clapper"
		}
	}
	return html.Div(html.Props{Class: "df-table-feature"}, FeatureIcon(name), html.Span(html.Props{Class: "df-feature-label"}, ui.Text(label)))
}

// PortraitCard renders a party seat, including a dim empty-seat state.
func PortraitCard(model PortraitCardModel) ui.Node {
	name := strings.TrimSpace(model.Name)
	if name == "" && !model.Joined {
		name = "Waiting for a player…"
	}
	if name == "" {
		name = "Player"
	}
	subtitle := strings.TrimSpace(model.Subtitle)
	if subtitle == "" {
		if model.Joined {
			subtitle = "Joined"
		} else {
			subtitle = "Adventurer"
		}
	}
	portrait := model.PortraitURL
	if portrait == "" {
		portrait = ArtURL("ui/logo_emblem")
	}
	portraitNode := html.Div(html.Props{Class: "df-portrait-card-portrait", Aria: map[string]string{"label": name}},
		html.Img(html.Props{Src: portrait, Alt: "", Hidden: portrait == "", Raw: map[string]any{"aria-hidden": "true"}}),
		html.Span(html.Props{Class: "df-portrait-card-silhouette", Hidden: portrait != "", Aria: map[string]string{"hidden": "true"}}, ui.Text(T("en", "dm.glyph.star", nil))),
	)
	children := []ui.Node{portraitNode, html.Div(html.Props{Class: "df-portrait-card-copy"},
		html.Strong(html.Props{Class: "df-portrait-card-name"}, ui.Text(name)),
		html.Span(html.Props{Class: "df-portrait-card-subtitle"}, ui.Text(subtitle)),
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
		html.Div(html.Props{Class: "df-speaker-caption-ornament", Aria: map[string]string{"hidden": "true"}}, ui.Text(T("en", "dm.glyph.star", nil))),
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

// WordmarkBand renders only the lettering of the ui/logo_wordmark art (the
// lantern crest is cropped away) at the given width, through a luminance mask
// of itself so its black plate disappears. The art is 1536x1024 with the
// lettering at x 120-1425, y 430-790.
func WordmarkBand(url string, width int) ui.Node {
	scale := float64(width) / 1305
	size := fmt.Sprintf("%.0fpx %.0fpx", 1536*scale, 1024*scale)
	pos := fmt.Sprintf("-%.0fpx -%.0fpx", 120*scale, 445*scale)
	art := "url('" + url + "')"
	return html.Div(html.Props{Class: "df-wordmark-art df-wordmark-band", Role: "img", Aria: map[string]string{"label": "DungeonFlux"}, Style: map[string]string{
		"width": fmt.Sprintf("%dpx", width), "height": fmt.Sprintf("%.0fpx", 345*scale),
		"background-image": art, "background-size": size, "background-position": pos,
		"-webkit-mask-image": art, "mask-image": art, "mask-mode": "luminance",
		"-webkit-mask-size": size, "mask-size": size, "-webkit-mask-position": pos, "mask-position": pos,
	}})
}

// CornerBrand is the small top-left wordmark used on in-game screens: the
// lettering band of the logo art with the subtitle beneath (callers position it).
func CornerBrand() ui.Node {
	var mark ui.Node = html.Div(html.Props{Class: "df-corner-brand-text"}, ui.Text(T("en", "dm.brand", nil)))
	if url := ArtURL("ui/logo_wordmark"); url != "" {
		mark = WordmarkBand(url, 340)
	}
	return html.Div(html.Props{Class: "df-corner-brand"}, mark, html.P(html.Props{Class: "df-corner-brand-subtitle"}, ui.Text(titleSubtitle)))
}
