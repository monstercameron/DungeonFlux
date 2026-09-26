//go:build js && wasm

package dm

import (
	"strconv"
	"strings"

	dungeonfluxv1 "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
	"github.com/monstercameron/GoWebComponents/v6/html"
	"github.com/monstercameron/GoWebComponents/v6/router"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

// CombatComponent renders the battle stage with its HUD and FLAT fallback.
func CombatComponent(view *dungeonfluxv1.DMView, sequence ...uint64) router.Component {
	version := uint64(1)
	if len(sequence) > 0 && sequence[0] > 0 {
		version = sequence[0]
	}
	model := CombatModelFromViewAt(view, version)
	stage := BattleStageFromView(view, version)
	locale := localeOrDefault(view.GetLocale())
	return func(_ router.Attrs) *router.Element {
		handleRef := ui.UseRef((*battleStageHandle)(nil))
		ui.UseEffect(func() func() {
			if !stage.Enabled {
				return nil
			}
			handle := claimBattleStage(stage)
			handleRef.Set(handle)
			return func() {
				releaseBattleStage(handle)
				handleRef.Set(nil)
			}
		}, stage.Init.SceneURL)
		ui.UseEffect(func() func() {
			if handle := handleRef.Get(); handle != nil {
				handle.apply(stage)
			}
			return nil
		}, stage.Scene.Seq, stage.Scene.Visible, stage.Scene.Camera.FocusTokenID, stageSnapshotKey(stage))
		children := make([]ui.Node, 0, 4)
		if stage.Enabled {
			// No width/height attributes and no opacity here: PlayCanvas sizes the
			// canvas and the battle stage handle fades it in. When these were
			// props, every re-render reset them (clearing the drawing buffer
			// and hiding the canvas), so the splat kept fading away. The
			// starting opacity lives in dmCombatStageCSS.
			children = append(children, html.Canvas(html.Props{ID: stage.Init.CanvasID, Class: "df-dm-combat-splat", Style: map[string]string{"position": "absolute", "inset": "0", "width": "100%", "height": "100%", "transition": "opacity 1s ease", "z-index": "0", "pointer-events": "none"}}))
		}
		fallback := []ui.Node{combatGrid(model.Segments), combatHighlights(model.Highlights)}
		fallback = append(fallback, combatTokens(model.Tokens)...)
		if stage.Enabled {
			children = append(children, html.Div(html.Props{ID: "df-combat-flat-fallback", Style: map[string]string{"position": "absolute", "inset": "0", "z-index": "1", "transition": "opacity 180ms ease", "pointer-events": "none"}}, fallback...))
		} else if !model.UseSplat {
			children = append(children, fallback...)
		}
		hud := []ui.Node{combatVignette(), combatPartyRail(model), combatEnemyCard(model), combatTimer(model.Timer), combatTopTitle(locale, model), combatActionBar()}
		children = append(children, html.Div(html.Props{Style: map[string]string{"position": "absolute", "inset": "0", "z-index": "2", "pointer-events": "none"}}, hud...))
		return html.Section(html.Props{Class: "df-dm-combat", Role: "img", Aria: map[string]string{"label": T(locale, "dm.combat_label", nil)}, Style: map[string]string{"position": "relative", "width": "100%", "height": "100%", "overflow": "hidden"}},
			html.Div(html.Props{Class: "df-dm-combat-stage", Style: combatStageStyle(model)}, children...),
		)
	}
}

func combatStageStyle(model CombatModel) map[string]string {
	style := map[string]string{"position": "relative", "width": "100%", "height": "100%", "background": "radial-gradient(circle at 50% 45%,#27303b,#0d1118 76%)", "background-size": "cover", "background-position": "center", "filter": "saturate(.94) contrast(1.05)"}
	if imageURL := combatImageURL(model); imageURL != "" {
		style["background-image"] = "linear-gradient(180deg,rgba(7,10,15,.08),rgba(7,10,15,.52)),url('" + imageURL + "')"
	}
	return style
}

func combatTopTitle(locale string, model CombatModel) ui.Node {
	return html.Div(html.Props{Style: map[string]string{"position": "absolute", "left": "460px", "right": "460px", "top": "34px", "text-align": "center", "color": "#efe6d2", "text-shadow": "0 3px 12px #000"}},
		html.Div(html.Props{Style: map[string]string{"color": "#e7c27a", "font-family": "Cinzel,'Cormorant Garamond',Georgia,serif", "font-size": "34px", "letter-spacing": ".16em", "text-transform": "uppercase"}}, ui.Text("The Drowned Lantern")),
		html.Div(html.Props{Style: map[string]string{"width": "440px", "max-width": "80%", "height": "1px", "margin": "10px auto", "background": "linear-gradient(90deg,transparent,#d9a441,transparent)"}}),
		html.Div(html.Props{Style: map[string]string{"color": "#efe6d2", "font-family": "Cormorant Garamond,Georgia,serif", "font-size": "27px"}}, ui.Text(combatBanner(model))),
		html.Span(html.Props{Hidden: locale == "", Style: map[string]string{"display": "none"}}, ui.Text(locale)),
	)
}

func combatGrid(segments []CombatSegment) ui.Node {
	children := make([]ui.Node, 0, len(segments))
	for _, segment := range segments {
		children = append(children, html.Line(html.Props{Raw: map[string]any{"x1": number(segment.From.X), "y1": number(segment.From.Y), "x2": number(segment.To.X), "y2": number(segment.To.Y), "stroke": "#e7c27a", "stroke-opacity": "0.62", "stroke-width": "3"}}))
	}
	return html.Svg(html.Props{Class: "df-dm-combat-grid", Style: map[string]string{"position": "absolute", "inset": "0", "width": "100%", "height": "100%", "pointer-events": "none"}, Raw: map[string]any{"viewBox": "0 0 1920 1080", "preserveAspectRatio": "none", "aria-hidden": "true"}}, children...)
}

func combatHighlights(points []CombatPoint) ui.Node {
	nodes := make([]ui.Node, 0, len(points))
	for _, point := range points {
		nodes = append(nodes, html.Div(html.Props{Class: "df-dm-combat-highlight", Style: map[string]string{"position": "absolute", "left": percent(point.X / 1920 * 100), "top": percent(point.Y / 1080 * 100), "transform": "translate(-50%,-50%)", "width": "4.5%", "aspect-ratio": "1", "border": "3px solid #3aa39a", "border-radius": "50%", "background": "rgba(58,163,154,.2)", "box-sizing": "border-box"}}))
	}
	return html.Div(html.Props{Class: "df-dm-combat-highlights", Style: map[string]string{"position": "absolute", "inset": "0", "pointer-events": "none"}}, nodes...)
}

func combatTokens(tokens []CombatToken) []ui.Node {
	nodes := make([]ui.Node, 0, len(tokens))
	for _, token := range tokens {
		status := strings.Join(token.Statuses, " · ")
		label := token.Name
		if status != "" {
			label += " · " + status
		}
		portrait := artSrc(token.Portrait)
		if portrait == "" {
			portrait = artSrc("ui/logo_emblem")
		}
		bar := html.Div(html.Props{Style: map[string]string{"position": "absolute", "left": "8%", "right": "8%", "bottom": "12px", "height": "10px", "border": "2px solid #efe6d2", "border-radius": "8px", "background": "#291b1b", "overflow": "hidden"}}, html.Div(html.Props{Style: map[string]string{"width": combatHPPercent(token.HP, token.HPMax), "height": "100%", "background": combatHPColor(token.HP, token.HPMax)}}))
		imageChildren := []ui.Node{html.Img(html.Props{Src: portrait, Alt: token.Name, Style: map[string]string{"width": "100%", "height": "100%", "object-fit": "contain"}}), bar}
		imageChildren = append(imageChildren, combatStatusIcons(token.Statuses)...)
		image := html.Div(html.Props{Style: map[string]string{"position": "relative", "filter": tokenFilter(token.Statuses)}}, imageChildren...)
		nodes = append(nodes, html.Div(html.Props{Class: "df-dm-combat-token", Style: map[string]string{"position": "absolute", "left": percent(token.X / 1920 * 100), "top": percent(token.Y / 1080 * 100), "transform": "translate(-50%,-50%)", "width": "10%", "text-align": "center"}, Raw: map[string]any{"data-token-id": token.ID, "data-active": strconv.FormatBool(token.Active)}},
			image, html.Div(html.Props{Style: map[string]string{"color": "#efe6d2", "font-family": "Cormorant Garamond,Georgia,serif", "font-size": "22px", "font-weight": "600", "text-shadow": "0 2px 5px #000"}}, html.Text(label))),
		)
	}
	return nodes
}

func combatPartyRail(model CombatModel) ui.Node {
	items := make([]ui.Node, 0, len(model.Tokens))
	for _, token := range model.Tokens {
		if strings.Contains(strings.ToLower(token.Name), "thrall") {
			continue
		}
		items = append(items, html.Div(html.Props{Style: map[string]string{"display": "grid", "grid-template-columns": "68px 1fr", "gap": "12px", "align-items": "center", "padding": "8px", "border": "1px solid rgba(184,137,58,.68)", "border-radius": "8px", "background": "rgba(12,18,28,.86)"}},
			html.Img(html.Props{Src: artSrc(token.Portrait), Alt: token.Name, Style: map[string]string{"width": "64px", "height": "64px", "object-fit": "cover", "border": "1px solid #b8893a"}}),
			html.Div(html.Props{Style: map[string]string{"min-width": "0"}}, html.Strong(html.Props{Style: map[string]string{"display": "block", "font-family": "Cinzel,'Cormorant Garamond',Georgia,serif", "font-size": "24px", "white-space": "nowrap", "overflow": "hidden", "text-overflow": "ellipsis"}}, ui.Text(token.Name)), html.Span(html.Props{Style: map[string]string{"color": "#2fb59a", "font-size": "18px"}}, ui.Text("HP "+strconv.Itoa(int(token.HP))+"/"+strconv.Itoa(int(token.HPMax))))),
		))
	}
	return html.Aside(html.Props{Class: "df-dm-combat-party", Style: map[string]string{"position": "absolute", "left": "30px", "top": "150px", "width": "300px", "display": "flex", "flex-direction": "column", "gap": "12px"}}, items...)
}

func combatEnemyCard(model CombatModel) ui.Node {
	for _, token := range model.Tokens {
		if strings.Contains(strings.ToLower(token.Name), "thrall") {
			return enemyCard(token)
		}
	}
	return html.Div(html.Props{Hidden: true})
}

func enemyCard(token CombatToken) ui.Node {
	return html.Div(html.Props{Class: "df-dm-combat-enemy", Style: map[string]string{"position": "absolute", "right": "36px", "top": "135px", "width": "310px", "padding": "14px", "border": "1px solid #b3372f", "border-radius": "10px", "background": "rgba(12,18,28,.9)", "box-shadow": "0 12px 28px rgba(0,0,0,.52)"}},
		html.Div(html.Props{Style: map[string]string{"color": "#e7c27a", "font-family": "Cinzel,'Cormorant Garamond',Georgia,serif", "font-size": "22px", "letter-spacing": ".1em", "text-transform": "uppercase"}}, ui.Text("Enemy")),
		html.Div(html.Props{Style: map[string]string{"display": "flex", "gap": "14px", "align-items": "center", "margin-top": "10px"}}, html.Img(html.Props{Src: artSrc(token.Portrait), Alt: token.Name, Style: map[string]string{"width": "92px", "height": "92px", "object-fit": "cover", "border": "2px solid #b3372f"}}), html.Div(html.Props{Style: map[string]string{"font-family": "Cormorant Garamond,Georgia,serif", "font-size": "28px"}}, ui.Text(token.Name))),
		html.Div(html.Props{Style: map[string]string{"margin-top": "12px", "height": "8px", "background": "#291b1b", "border-radius": "5px", "overflow": "hidden"}}, html.Div(html.Props{Style: map[string]string{"width": combatHPPercent(token.HP, token.HPMax), "height": "100%", "background": "#b3372f"}})),
	)
}

func combatTimer(view TimerView) ui.Node {
	return html.Div(html.Props{Class: "df-dm-combat-timer", Style: map[string]string{"position": "absolute", "left": "730px", "top": "112px", "width": "460px", "padding": "10px 18px", "border": "1px solid #b8893a", "border-radius": "8px", "background": "rgba(12,18,28,.84)"}}, TimerComponent(view)(router.Attrs{}))
}

func combatActionBar() ui.Node {
	actions := []ActionButtonModel{{Icon: "⚔", Label: "Attack", Hotkey: "1", Primary: true, Enabled: true}, {Icon: "◇", Label: "Move", Hotkey: "2", Enabled: true}, {Icon: "◈", Label: "End turn", Hotkey: "3", Enabled: true}}
	items := make([]ui.Node, 0, len(actions))
	for _, action := range actions {
		items = append(items, ActionButton(action))
	}
	return html.Div(html.Props{Class: "df-dm-combat-actions", Role: "list", Style: map[string]string{"position": "absolute", "left": "710px", "bottom": "42px", "display": "flex", "gap": "16px", "padding": "14px", "border": "1px solid #b8893a", "border-radius": "12px", "background": "rgba(12,18,28,.88)", "box-shadow": "0 16px 36px rgba(0,0,0,.5)"}}, items...)
}

func combatVignette() ui.Node {
	return html.Div(html.Props{Style: map[string]string{"position": "absolute", "inset": "0", "pointer-events": "none", "box-shadow": "inset 0 0 120px rgba(0,0,0,.72),inset 0 -180px 160px rgba(0,0,0,.48)"}})
}

func combatImageURL(model CombatModel) string {
	if artURL := ArtURL("battlefield_tavern_flat"); artURL != "" {
		return artURL
	}
	return artSrc(model.ImageURL)
}

func combatBanner(model CombatModel) string {
	// The server sends banner tokens (pc_turn, enemy_turn, ...); show words.
	switch strings.ToLower(strings.TrimSpace(model.Banner)) {
	case "":
		return "Combat"
	case "pc_turn":
		for _, token := range model.Tokens {
			if token.Active && token.Name != "" {
				return token.Name + "'s turn"
			}
		}
		return "Your move"
	case "enemy_turn", "thrall_turn", "npc_turn":
		return "The thrall moves"
	case "intro":
		return "To arms!"
	case "done", "outcome":
		return "The fight is over"
	default:
		if strings.Contains(model.Banner, "_") {
			return strings.ReplaceAll(model.Banner, "_", " ")
		}
		return model.Banner
	}
}

func classCrest(className string) string {
	className = strings.ToLower(strings.TrimSpace(className))
	if className == "" {
		return ""
	}
	return ArtURL("ui/class_" + strings.ReplaceAll(className, " ", "_"))
}

func combatStatusIcons(statuses []string) []ui.Node {
	nodes := make([]ui.Node, 0, len(statuses))
	for _, status := range statuses {
		name := strings.ToLower(strings.TrimSpace(status))
		if name == "" {
			continue
		}
		if artURL := ArtURL("ui/status_" + name); artURL != "" {
			nodes = append(nodes, html.Img(html.Props{Src: artURL, Alt: status, Style: map[string]string{"position": "absolute", "right": "4px", "top": "4px", "width": "28px", "height": "28px", "object-fit": "contain"}}))
		}
	}
	return nodes
}

func combatHPPercent(hp, max int32) string {
	if max <= 0 || hp <= 0 {
		return "0%"
	}
	if hp >= max {
		return "100%"
	}
	return strconv.Itoa(int(hp*100/max)) + "%"
}

func combatHPColor(hp, max int32) string {
	if max > 0 && hp*2 <= max {
		return "#b3372f"
	}
	return "#2fb59a"
}

func tokenFilter(statuses []string) string {
	for _, status := range statuses {
		if strings.EqualFold(status, "down") || strings.EqualFold(status, "defeated") {
			return "grayscale(1) opacity(.7)"
		}
	}
	return "drop-shadow(0 6px 8px rgba(0,0,0,.6))"
}
