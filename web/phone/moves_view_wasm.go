//go:build js && wasm

package phone

import (
	"context"

	"github.com/monstercameron/GoWebComponents/v6/html"
	"github.com/monstercameron/GoWebComponents/v6/router"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

// MovesScreen renders the legal action menu for a portrait phone display.
func MovesScreen(model *MovesModel) router.Component {
	return func(_ router.Attrs) *router.Element {
		refresh := ui.UseState(0)
		current := model.Snapshot()
		locale := current.Locale
		if locale == "" {
			locale = "en"
		}
		children := make([]ui.Node, 0, len(current.Moves)+3)
		for _, move := range current.Moves {
			item := move
			tap := ui.UseEvent(func() {
				go func() { model.ApplyAct(<-model.Tap(context.Background(), item.ID)); refresh.Set(refresh.Get() + 1) }()
			})
			children = append(children, moveCard(item, tap))
		}
		if current.Error != "" {
			children = append(children, html.P(html.Props{Role: "alert", Style: map[string]string{
				"margin": "0", "padding": "12px 14px", "border-radius": "10px", "background": "#4b2025", "color": "#ffd8d2",
			}}, html.Text(current.Error)))
		}
		header := html.Div(html.Props{Style: map[string]string{"display": "flex", "flex-direction": "column", "gap": "6px"}},
			html.H1(html.Props{Style: map[string]string{"margin": "0", "font-family": "Georgia, serif", "font-size": "clamp(1.8rem, 8vw, 2.35rem)", "line-height": "1.05", "color": "#efe6d2"}}, html.Text(MovesTitle(locale))),
			html.P(html.Props{Role: "status", Aria: map[string]string{"live": "polite"}, Style: map[string]string{"margin": "0", "color": "#b8b1a1", "font-size": "0.98rem"}}, html.Text(current.StatusText)),
		)
		children = append([]ui.Node{header}, children...)
		return html.Main(html.Props{Class: "df-phone df-phone-moves", Role: "main", Style: map[string]string{
			"box-sizing": "border-box", "min-height": "100vh", "width": "100%", "max-width": "480px", "margin": "0 auto", "padding": "24px 18px calc(24px + env(safe-area-inset-bottom))", "display": "flex", "flex-direction": "column", "gap": "18px", "background": "radial-gradient(circle at 50% -10%, #2b2a30 0, #171921 42%, #0f1117 100%)", "color": "#efe6d2", "font-family": "system-ui, sans-serif",
		}}, children...)
	}
}

func moveCard(move MoveSnapshot, tap ui.Handler) ui.Node {
	style := map[string]string{"width": "100%", "min-height": "76px", "box-sizing": "border-box", "display": "flex", "flex-direction": "column", "align-items": "flex-start", "justify-content": "center", "gap": "5px", "padding": "14px 16px", "border-radius": "12px", "border": "1px solid #d9a441", "background": "linear-gradient(135deg, #3a2c18, #211b17)", "color": "#fff4dc", "text-align": "left", "font-size": "1.15rem", "font-weight": "700", "line-height": "1.2", "box-shadow": "0 5px 16px rgba(0,0,0,.26)", "touch-action": "manipulation"}
	if !move.Enabled {
		style = map[string]string{"width": "100%", "min-height": "76px", "box-sizing": "border-box", "display": "flex", "flex-direction": "column", "align-items": "flex-start", "justify-content": "center", "gap": "5px", "padding": "14px 16px", "border-radius": "12px", "border": "1px solid #454650", "background": "#252730", "color": "#777b87", "text-align": "left", "font-size": "1.15rem", "font-weight": "700", "line-height": "1.2", "opacity": "0.86"}
	}
	children := []ui.Node{html.Span(html.Props{}, html.Text(move.Label))}
	if preview := MovePreviewText(move); preview != "" {
		children = append(children, html.Small(html.Props{Style: map[string]string{"font-size": "0.82rem", "font-weight": "500", "color": "#d9a441"}}, html.Text(preview)))
	}
	button := html.Button(html.Props{Type: "button", Class: moveClass(move), OnClick: tap, Disabled: !move.Enabled, Aria: map[string]string{"label": moveAriaLabel(move), "describedby": "move-reason-" + move.ID}, Style: style}, children...)
	if !move.Enabled {
		return html.Div(html.Props{Style: map[string]string{"display": "flex", "flex-direction": "column", "gap": "6px"}}, button, html.Small(html.Props{ID: "move-reason-" + move.ID, Class: "df-phone-move-reason", Role: "note", Style: map[string]string{"padding": "0 4px", "color": "#a89f8c", "font-size": "0.86rem", "line-height": "1.25"}}, html.Text(MoveReason(move))))
	}
	return button
}

func moveAriaLabel(move MoveSnapshot) string {
	if !move.Enabled {
		return move.Label + ". " + MoveReason(move)
	}
	return move.Label
}

func moveClass(move MoveSnapshot) string {
	if move.Enabled {
		return "df-phone-move"
	}
	return "df-phone-move df-phone-move-disabled"
}
