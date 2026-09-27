//go:build js && wasm

package phone

import (
	"context"
	"strconv"
	"strings"
	"syscall/js"

	"github.com/monstercameron/GoWebComponents/v6/html"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

// combatMapProps carries the model and a revision so the panel re-renders
// when the parent's refresh counter moves (prop-less closures would not).
type combatMapProps struct {
	model    *CombatModel
	refresh  ui.State[int]
	revision int
	watching bool
	// timer and percent are the turn timer, shown compactly in map mode.
	timer   string
	percent int32
}

// combatMapPanel is the top-down combat map. For the active seat it is the
// Move picker (tap a cell, see the engine path, Commit or Back); for the
// watching seat it is a read-only view of the same battlefield.
func combatMapPanel(props combatMapProps) ui.Node {
	model := props.model
	model.SetMapClock(performanceNow)
	combatMapStyle()
	layout := model.MapLayout(model.MapNow())
	selectedKey := ""
	if layout.Selected != nil {
		selectedKey = strconv.Itoa(int(layout.Selected.GetC())) + "," + strconv.Itoa(int(layout.Selected.GetR()))
	}
	ui.UseEffect(func() func() {
		if selectedKey != "" {
			if cell := js.Global().Get("document").Call("querySelector", ".df-cm-cell.is-selected"); cell.Truthy() {
				cell.Call("scrollIntoView", map[string]any{"block": "nearest", "behavior": "smooth"})
			}
		}
		return nil
	}, selectedKey)
	if layout.Cols == 0 {
		return nil
	}
	combatMapKeyframes(layout)
	class := "df-cm"
	if props.watching {
		class += " is-watching"
	}
	if layout.Rotated {
		class += " is-rotated"
	}
	board := combatMapBoard(layout, props)
	children := []ui.Node{combatMapHead(layout, props), board}
	if !props.watching {
		children = append(children, ui.CreateElement(combatMapBar, combatMapBarProps{model: model, refresh: props.refresh, revision: props.revision, preview: layout.Preview, dash: layout.Dash, selected: layout.Selected != nil, pending: layout.Pending}))
	}
	return html.Section(html.Props{Class: class, Aria: map[string]string{"label": "Battlefield map"}}, children...)
}

func combatMapHead(layout CombatMapLayout, props combatMapProps) ui.Node {
	watching := props.watching
	title, detail := "Move", mapHeadline(layout.MoveLeft, layout.CanDash)
	if watching {
		title, detail = "Battlefield", "Watch the fight unfold"
	}
	legend := []ui.Node{html.Span(html.Props{Class: "df-cm-key is-reach"}, html.Text(T("en", "phone.combat.move", nil)))}
	if layout.CanDash && !watching {
		legend = append(legend, html.Span(html.Props{Class: "df-cm-key is-dash"}, html.Text(T("en", "phone.combat.dash", nil))))
	}
	if watching {
		legend = nil
	}
	head := []ui.Node{html.Div(html.Props{Class: "df-cm-title"}, html.H2(html.Props{}, html.Text(title)), html.Small(html.Props{}, html.Text(detail)),
		html.Div(html.Props{Class: "df-cm-legend", Aria: map[string]string{"hidden": "true"}}, legend...))}
	if props.timer != "" && !watching {
		head = append(head, html.Div(html.Props{Class: "df-cm-timer", Role: "timer", Aria: map[string]string{"label": "Turn timer " + props.timer}},
			html.Strong(html.Props{}, html.Text(props.timer)),
			html.Span(html.Props{Class: "df-cm-timer-bar"}, html.Span(html.Props{Style: map[string]string{"width": strconv.Itoa(int(props.percent)) + "%"}}))))
	}
	return html.Header(html.Props{Class: "df-cm-head"}, head...)
}

func combatMapBoard(layout CombatMapLayout, props combatMapProps) ui.Node {
	nodes := make([]ui.Node, 0, len(layout.Cells)+len(layout.Tokens))
	for _, cell := range layout.Cells {
		nodes = append(nodes, ui.CreateElement(combatMapCellButton, combatMapCellProps{model: props.model, refresh: props.refresh, cell: cell, interactive: layout.Interactive && !layout.Pending, revision: props.revision}))
	}
	for _, token := range layout.Tokens {
		nodes = append(nodes, combatMapTokenNode(token))
	}
	style := map[string]string{"grid-template-columns": "repeat(" + strconv.Itoa(layout.Cols) + ",minmax(0,1fr))"}
	return html.Div(html.Props{Class: "df-cm-board", Style: style}, nodes...)
}

type combatMapCellProps struct {
	model       *CombatModel
	refresh     ui.State[int]
	cell        CombatMapCell
	interactive bool
	revision    int
}

func combatMapCellButton(props combatMapCellProps) ui.Node {
	cell := props.cell
	target := cloneMessage(cell.Cell)
	tap := ui.UseEvent(func() {
		props.model.SelectCell(target)
		props.refresh.Set(props.refresh.Get() + 1)
	})
	classes := []string{"df-cm-cell"}
	switch {
	case !cell.Walkable:
		classes = append(classes, "is-blocked")
	case cell.Dash:
		classes = append(classes, "is-dash")
	case cell.Reach:
		classes = append(classes, "is-reach")
	}
	if cell.Selected {
		classes = append(classes, "is-selected")
	}
	if cell.PathStep > 0 && !cell.Selected {
		classes = append(classes, "is-path")
	}
	clickable := props.interactive && (cell.Reach || cell.Dash)
	style := map[string]string{"grid-column": strconv.Itoa(cell.X + 1), "grid-row": strconv.Itoa(cell.Y + 1)}
	children := []ui.Node{}
	if cell.PathStep > 0 && !cell.Selected {
		children = append(children, html.Span(html.Props{Class: "df-cm-step", Aria: map[string]string{"hidden": "true"}}))
	}
	if !clickable {
		return html.Div(html.Props{Class: strings.Join(classes, " "), Style: style, Aria: map[string]string{"hidden": "true"}}, children...)
	}
	return html.Button(html.Props{Type: "button", Class: strings.Join(classes, " "), Style: style, OnClick: tap, Aria: map[string]string{"label": combatMapCellLabel(cell)}}, children...)
}

func combatMapCellLabel(cell CombatMapCell) string {
	where := "column " + strconv.Itoa(int(cell.Cell.GetC())+1) + ", row " + strconv.Itoa(int(cell.Cell.GetR())+1)
	if cell.Dash {
		return "Dash to " + where
	}
	return "Move to " + where
}

func combatMapTokenNode(token CombatMapToken) ui.Node {
	classes := []string{"df-cm-token"}
	for _, flag := range []struct {
		on    bool
		class string
	}{{token.Me, "is-me"}, {token.Enemy, "is-enemy"}, {token.Down, "is-down"}, {token.Active, "is-active"}, {token.Walk != nil, "is-walking"}, {token.Walk != nil && token.Walk.Dash, "is-dashing"}} {
		if flag.on {
			classes = append(classes, flag.class)
		}
	}
	style := map[string]string{"grid-column": strconv.Itoa(token.X + 1), "grid-row": strconv.Itoa(token.Y + 1)}
	key := "token-" + token.ID
	if token.Walk != nil {
		style["animation"] = token.Walk.Name + " " + strconv.Itoa(token.Walk.DurationMS) + "ms linear -" + strconv.Itoa(token.Walk.ElapsedMS) + "ms both"
		key += "-" + token.Walk.Name
	}
	label := token.Name
	if token.Me {
		label += " (you)"
	}
	return html.Div(html.Props{Key: key, Class: strings.Join(classes, " "), Style: style, Role: "img", Aria: map[string]string{"label": label}},
		html.Span(html.Props{Class: "df-cm-chip"}, combatMapPortrait(token)))
}

func combatMapPortrait(token CombatMapToken) ui.Node {
	if token.Enemy {
		return html.Span(html.Props{Class: "df-cm-glyph", Aria: map[string]string{"hidden": "true"}}, html.Text("☠"))
	}
	src := portraitSrc(token.Portrait)
	if src == "" {
		src = heroProxyArt("", token.Kind, token.ID)
	}
	if src == "" {
		return html.Span(html.Props{Class: "df-cm-glyph"}, html.Text(combatInitials(token.Name)))
	}
	return html.Img(html.Props{Src: src, Alt: ""})
}

type combatMapBarProps struct {
	model    *CombatModel
	refresh  ui.State[int]
	revision int
	preview  string
	dash     bool
	selected bool
	pending  bool
}

// combatMapBar is the sticky commit bar under the map.
func combatMapBar(props combatMapBarProps) ui.Node {
	back := ui.UseEvent(func() {
		if props.selected {
			props.model.ClearSelection()
		} else {
			props.model.CloseMap()
		}
		props.refresh.Set(props.refresh.Get() + 1)
	})
	commit := ui.UseEvent(func() {
		// The RPC starts off the event loop: a blocking call inside a JS
		// callback would stall the Watch stream that carries the walk.
		go func() {
			result := props.model.CommitMove(context.Background())
			props.refresh.Set(props.refresh.Get() + 1)
			props.model.ApplyCommit(<-result)
			props.refresh.Set(props.refresh.Get() + 1)
		}()
	})
	class := "df-cm-bar"
	if props.dash {
		class += " is-dash"
	}
	label := props.preview
	if !props.selected {
		label = "Tap a lit cell to plan your path"
	}
	if props.pending {
		label = "Moving…"
	}
	commitText := "Commit"
	if props.dash {
		commitText = "Commit dash"
	}
	backText := "Back"
	if props.selected {
		backText = "Clear"
	}
	buttons := []ui.Node{html.Button(html.Props{Type: "button", Class: "df-phone-secondary df-cm-back", OnClick: back}, html.Text(backText))}
	if props.selected {
		buttons = append(buttons, html.Button(html.Props{Type: "button", Class: "df-phone-primary df-cm-commit", Disabled: props.pending, OnClick: commit}, html.Text(commitText)))
	}
	return html.Footer(html.Props{Class: class},
		html.P(html.Props{Class: "df-cm-preview", Role: "status", Aria: map[string]string{"live": "polite"}}, html.Text(label)),
		html.Div(html.Props{Class: "df-cm-actions"}, buttons...),
	)
}

func performanceNow() float64 {
	performance := js.Global().Get("performance")
	if !performance.Truthy() {
		return 0
	}
	return performance.Call("now").Float()
}

// combatMapKeyframes writes the running walks' keyframes into their own
// <style> element (never html.Text inside <style>, which escapes the rules).
func combatMapKeyframes(layout CombatMapLayout) {
	var rules strings.Builder
	for _, token := range layout.Tokens {
		if token.Walk != nil {
			rules.WriteString(token.Walk.Keyframes)
		}
	}
	text := rules.String()
	if text == "" {
		return
	}
	setStyleText("df-combat-map-walks", text)
}

func setStyleText(id, text string) {
	doc := js.Global().Get("document")
	if !doc.Truthy() {
		return
	}
	el := doc.Call("getElementById", id)
	if !el.Truthy() {
		el = doc.Call("createElement", "style")
		el.Set("id", id)
		doc.Get("head").Call("appendChild", el)
	}
	if el.Get("textContent").String() != text {
		el.Set("textContent", text)
	}
}
