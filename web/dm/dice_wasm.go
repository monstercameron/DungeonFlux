//go:build js && wasm

package dm

import (
	"strconv"

	"github.com/monstercameron/GoWebComponents/v6/html"
	"github.com/monstercameron/GoWebComponents/v6/router"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

// DiceComponent renders the check backdrop, d20 art, and result banner on the
// fixed 1920x1080 DM canvas.
func DiceComponent(view DiceView) router.Component {
	return func(_ router.Attrs) *router.Element {
		locale := localeOrDefault(view.Locale)
		if view.State == "" {
			return html.Div(html.Props{Class: "df-dm-dice", Hidden: true})
		}
		presentation := PresentDice(view)
		return html.Section(html.Props{Class: "df-dm-dice df-dm-check-screen " + presentation.StateClass, Role: "status", Aria: map[string]string{"label": T(locale, "dm.dice_aria", nil)}, Style: diceScreenStyle()},
			canvasRepairStyle(),
			diceHeader(locale, view), diceHero(view, presentation), diceResultPanel(locale, view, presentation), diceAnimationStyle(),
		)
	}
}

func diceScreenStyle() map[string]string {
	style := map[string]string{
		"position": "absolute", "left": "0", "top": "-194px", "width": "1920px", "height": "1080px",
		"overflow": "hidden", "color": "#efe6d2", "font-family": "Inter,ui-sans-serif,system-ui,sans-serif",
		"background": "radial-gradient(circle at 50% 45%, rgba(28,36,47,.24), rgba(8,10,15,.92) 75%), #0f1117",
	}
	if artURL := ArtURL("ui/check_backdrop"); artURL != "" {
		style["background-image"] = "linear-gradient(180deg, rgba(7,10,15,.26), rgba(7,10,15,.82)), url('" + artURL + "')"
		style["background-size"] = "cover"
		style["background-position"] = "center"
	}
	return style
}

func diceHeader(locale string, view DiceView) ui.Node {
	return html.Div(html.Props{Style: map[string]string{"position": "absolute", "left": "46px", "top": "36px", "width": "500px", "text-align": "left"}},
		TitlePlate(ArtURL("ui/logo_wordmark"), "DungeonFlux", "AI DUNGEON MASTER FOR FIFTH-EDITION FANTASY"),
		html.Div(html.Props{Style: map[string]string{"margin-top": "22px", "color": "#e7c27a", "font-family": "Cormorant Garamond,Georgia,serif", "font-size": "32px", "font-style": "italic"}}, ui.Text(DiceHeading(locale, view.Kind))),
	)
}

func diceHero(view DiceView, presentation DicePresentation) ui.Node {
	art := ArtURL(diceArtName(view, presentation))
	faceStyle := map[string]string{"position": "relative", "width": "300px", "height": "300px", "display": "grid", "place-items": "center", "border": "2px solid #e7c27a", "border-radius": "28px", "background": "radial-gradient(circle at 35% 24%,#fff8e7,#d9a441 66%,#704b17)", "box-shadow": "0 0 0 8px rgba(12,18,28,.72), 0 28px 60px rgba(0,0,0,.62), 0 0 45px rgba(217,164,65,.22)", "overflow": "hidden"}
	if presentation.IsRolling {
		faceStyle["animation"] = "df-dice-roll 650ms ease-in-out infinite alternate"
	}
	children := make([]ui.Node, 0, 2)
	if art != "" {
		children = append(children, html.Img(html.Props{Src: art, Alt: "", Style: map[string]string{"position": "absolute", "inset": "0", "width": "100%", "height": "100%", "object-fit": "cover"}, Raw: map[string]any{"aria-hidden": "true"}}))
	}
	children = append(children, html.Span(html.Props{Style: map[string]string{"position": "relative", "z-index": "1", "font-family": "Cinzel,'Cormorant Garamond',Georgia,serif", "font-size": "138px", "font-weight": "600", "line-height": "1", "color": "#17130c", "text-shadow": "0 2px 6px rgba(255,248,231,.55)"}}, ui.Text(presentation.FaceLabel)))
	return html.Div(html.Props{Style: map[string]string{"position": "absolute", "left": "810px", "top": "220px", "width": "300px", "text-align": "center"}},
		html.Div(html.Props{Style: faceStyle}, children...),
		html.Div(html.Props{Style: map[string]string{"margin-top": "26px", "color": "#a89f8c", "font-family": "Cormorant Garamond,Georgia,serif", "font-size": "28px", "letter-spacing": ".08em", "text-transform": "uppercase"}}, ui.Text(modifierText(localeOrDefault(view.Locale), view))),
	)
}

func diceResultPanel(locale string, view DiceView, presentation DicePresentation) ui.Node {
	result := presentation.ResultText
	if view.Crit {
		result = CritLabel(locale)
	}
	if result == "" {
		result = stringsTitle(view.State)
	}
	color := "#a89f8c"
	if presentation.IsSuccess || view.Crit {
		color = "#3aa39a"
	} else if presentation.IsFailure {
		color = "#b3372f"
	}
	damage := ""
	if view.Damage != nil && view.Damage.Total != 0 {
		damage = strconv.Itoa(int(view.Damage.Total)) + " " + view.Damage.Type
	}
	return html.Div(html.Props{Style: map[string]string{"position": "absolute", "left": "350px", "right": "350px", "top": "625px", "min-height": "245px", "padding": "26px 70px", "box-sizing": "border-box", "text-align": "center", "background": "rgba(12,18,28,.88)", "border": "1px solid #b8893a", "border-radius": "12px", "box-shadow": "0 18px 42px rgba(0,0,0,.5), inset 0 0 0 1px rgba(12,12,16,.78)"}},
		html.Div(html.Props{Style: map[string]string{"color": "#e7c27a", "font-family": "Cinzel,'Cormorant Garamond',Georgia,serif", "font-size": "24px", "letter-spacing": ".16em", "text-transform": "uppercase"}}, ui.Text(vsLabel(locale, view))),
		html.Div(html.Props{Style: map[string]string{"margin-top": "14px", "color": color, "font-family": "Cinzel,'Cormorant Garamond',Georgia,serif", "font-size": "54px", "font-weight": "600", "letter-spacing": ".08em", "text-transform": "uppercase"}}, ui.Text(result)),
		html.Div(html.Props{Hidden: damage == "", Style: map[string]string{"margin-top": "8px", "color": "#efe6d2", "font-family": "Cormorant Garamond,Georgia,serif", "font-size": "28px"}}, ui.Text(damage)),
		html.Div(html.Props{Style: map[string]string{"width": "520px", "max-width": "80%", "height": "1px", "margin": "22px auto 0", "background": "linear-gradient(90deg,transparent,#d9a441,transparent)"}}),
	)
}

func modifierText(locale string, view DiceView) string {
	text := "d20"
	if view.Modifier >= 0 {
		text += " + " + strconv.Itoa(int(view.Modifier))
	} else {
		text += " - " + strconv.Itoa(int(-view.Modifier))
	}
	if view.DC > 0 {
		text += " " + VsDC(locale, view.DC)
	}
	if view.VsLabel != "" {
		text += " " + view.VsLabel
	}
	return text
}

func vsLabel(locale string, view DiceView) string {
	if view.DC > 0 {
		return "DC " + strconv.Itoa(int(view.DC)) + " · " + DiceHeading(locale, view.Kind)
	}
	if view.VsLabel != "" {
		return view.VsLabel
	}
	return DiceHeading(locale, view.Kind)
}

func stringsTitle(value string) string {
	if value == "" {
		return "Awaiting roll"
	}
	if value[0] >= 'a' && value[0] <= 'z' {
		return string(value[0]-('a'-'A')) + value[1:]
	}
	return value
}

func diceAnimationStyle() ui.Node {
	return html.Tag("style", html.Props{ID: "df-dm-dice-animation"}, html.Text("@keyframes df-dice-roll{from{transform:rotate(-7deg) scale(.96)}to{transform:rotate(7deg) scale(1.04)}}@media (prefers-reduced-motion:reduce){.df-dm-dice-face{animation:none!important}}"))
}

// TimerComponent renders the remaining turn time as a progress bar.
func TimerComponent(view TimerView) router.Component {
	return func(_ router.Attrs) *router.Element {
		locale := localeOrDefault(view.Locale)
		props := html.Props{Class: "df-dm-timer", Role: "timer", Aria: map[string]string{"label": timerLabel(locale, view)}}
		return html.Section(props,
			html.Div(html.Props{Class: "df-dm-timer-label"}, ui.Text(timerLabel(locale, view))),
			html.Progress(html.Props{Class: "df-dm-timer-bar", Raw: map[string]any{"max": strconv.FormatInt(view.TotalMS, 10), "value": strconv.FormatInt(maxTimerMS(view.RemainingMS), 10)}}),
		)
	}
}

func timerLabel(locale string, view TimerView) string {
	if view.Frozen {
		return TimerPaused(locale)
	}
	return TimerLabel(locale, maxTimerMS(view.RemainingMS))
}

func maxTimerMS(value int64) int64 {
	if value < 0 {
		return 0
	}
	return value
}
