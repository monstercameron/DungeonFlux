//go:build js && wasm

package phone

import (
	"context"
	"strconv"
	"strings"

	df "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
	"github.com/monstercameron/GoWebComponents/v6/html"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

func combatStyledScreen(model *CombatModel, locale string, _ ...ui.Node) ui.Node {
	if locale == "" {
		locale = "en"
	}
	refresh := ui.UseState(0)
	snapshot := model.Snapshot()
	theme := DefaultPhoneTheme()
	status := combatStatus(snapshot, locale)
	return html.Main(html.Props{Class: "df-phone df-phone-frame df-phone-combat", Role: "main", Style: combatPageStyle(theme)},
		combatHeader(snapshot, locale),
		html.Section(html.Props{Class: "df-phone-combat-body", Role: "region", Aria: map[string]string{"label": "Combat turn"}},
			combatIntro(snapshot, status, locale),
			combatHealth(snapshot, theme, locale),
			combatTimer(snapshot, theme, locale),
			combatTarget(snapshot, locale),
			combatGrid(snapshot, model, refresh, theme, locale),
			combatActions(snapshot, model, refresh, theme, locale),
		),
		combatFooter(snapshot, locale),
	)
}

func combatPageStyle(theme PhoneTheme) map[string]string {
	background := "radial-gradient(circle at 50% -10%, #31303a 0, #171a22 40%, #0f1117 100%)"
	if url := ArtURL(phoneBackgroundAsset); url != "" {
		background = "linear-gradient(180deg, rgba(8,10,14,.28), rgba(8,10,14,.88)), url(\"" + url + "\")"
	}
	return map[string]string{
		"box-sizing": "border-box", "width": "100%", "max-width": "390px", "min-height": "100vh",
		"margin": "0", "padding": "12px 12px calc(12px + env(safe-area-inset-bottom))",
		"display": "flex", "flex-direction": "column", "gap": "10px", "background": background,
		"background-size": "cover", "background-position": "center", "color": theme.Parchment,
		"font-family": "Inter, ui-sans-serif, system-ui, sans-serif", "overflow": "hidden",
	}
}

func combatHeader(snapshot CombatSnapshot, locale string) ui.Node {
	turn := snapshot.TurnLabel
	if snapshot.Down {
		turn = combatDownLabel(locale)
	}
	return html.Header(html.Props{Class: "df-phone-combat-header", Style: map[string]string{
		"display": "flex", "align-items": "flex-start", "justify-content": "space-between", "gap": "12px",
		"padding": "4px 2px 10px", "border-bottom": "1px solid rgba(217,164,65,.28)",
	}},
		html.Div(html.Props{},
			html.Div(html.Props{Style: map[string]string{"font-family": "Georgia, serif", "font-size": "1.65rem", "font-weight": "700", "letter-spacing": "-0.04em", "color": "#f0bb63", "text-shadow": "0 0 14px rgba(217,164,65,.18)"}}, html.Text("DungeonFlux")),
			html.Div(html.Props{Style: map[string]string{"margin-top": "2px", "font-family": "Georgia, serif", "font-size": ".63rem", "letter-spacing": ".13em", "text-transform": "uppercase", "color": "#a89f8c"}}, html.Text("The Drowned Lantern · Combat")),
		),
		html.Div(html.Props{Style: map[string]string{"text-align": "right", "font-size": ".69rem", "font-weight": "700", "letter-spacing": ".08em", "text-transform": "uppercase", "color": combatTurnColor(snapshot)}}, html.Text(turn)),
	)
}

func combatIntro(snapshot CombatSnapshot, status, locale string) ui.Node {
	label := "Combat"
	if snapshot.MyTurn && !snapshot.Down {
		label = CombatTurnTitle(locale)
	}
	return html.Div(html.Props{Class: "df-phone-combat-intro", Style: map[string]string{"display": "flex", "flex-direction": "column", "gap": "3px", "padding": "4px 2px 0"}},
		html.H1(html.Props{Style: map[string]string{"margin": "0", "font-family": "Georgia, serif", "font-size": "clamp(1.8rem, 8vw, 2.25rem)", "line-height": "1.05", "letter-spacing": "-.03em", "color": "#efe6d2"}}, html.Text(label)),
		html.P(html.Props{Role: "status", Aria: map[string]string{"live": "polite"}, Style: map[string]string{"margin": "0", "min-height": "1.35em", "color": "#bdb4a2", "font-size": ".9rem", "line-height": "1.35"}}, html.Text(status)),
	)
}

func combatHealth(snapshot CombatSnapshot, theme PhoneTheme, locale string) ui.Node {
	percent := combatPercent(int64(snapshot.HP), int64(snapshot.HPMax))
	label := SheetHP(locale, snapshot.HP, snapshot.HPMax)
	barColor := theme.Teal
	if snapshot.Down || combatHPState(snapshot) == "BLOODIED" {
		barColor = theme.Blood
	}
	return html.Section(html.Props{Class: "df-phone-combat-card df-phone-combat-health", Aria: map[string]string{"label": label}, Style: combatCardStyle(theme)},
		html.Div(html.Props{Style: combatCardHeadStyle()}, html.Span(html.Props{Style: combatLabelStyle()}, html.Text("YOUR VITALS")), html.Strong(html.Props{Style: map[string]string{"font-family": "Georgia, serif", "font-size": "1.1rem", "color": "#f3ead7"}}, html.Text(label))),
		combatBar(percent, barColor, label),
		html.Div(html.Props{Style: map[string]string{"display": "flex", "justify-content": "space-between", "margin-top": "6px", "font-size": ".72rem", "color": "#a89f8c"}},
			html.Span(html.Props{}, html.Text("HP")), html.Span(html.Props{}, html.Text(combatHPState(snapshot)))),
	)
}

func combatTimer(snapshot CombatSnapshot, theme PhoneTheme, locale string) ui.Node {
	percent := combatPercent(snapshot.TimerRemaining, snapshot.TimerTotal)
	if snapshot.TimerTotal <= 0 {
		percent = 0
	}
	label := snapshot.TimerLabel
	if label == "" {
		label = "—"
	}
	caption := "TURN TIMER"
	if snapshot.TimerFrozen {
		caption = "TURN TIMER · PAUSED"
	}
	return html.Section(html.Props{Class: "df-phone-combat-card df-phone-combat-timer", Aria: map[string]string{"label": caption}, Style: combatCardStyle(theme)},
		html.Div(html.Props{Style: combatCardHeadStyle()}, html.Span(html.Props{Style: combatLabelStyle()}, html.Text(caption)), html.Strong(html.Props{Style: map[string]string{"font-family": "Georgia, serif", "font-size": "1.2rem", "color": "#e2ad4e"}}, html.Text(label))),
		combatBar(percent, theme.Gold, caption),
	)
}

func combatTarget(snapshot CombatSnapshot, locale string) ui.Node {
	if snapshot.Attack == nil {
		return html.Div(html.Props{Class: "df-phone-combat-target", Style: map[string]string{"display": "none"}}, html.Text(""))
	}
	preview := combatAttackPreview(snapshot.Attack.Preview)
	return html.Section(html.Props{Class: "df-phone-combat-target", Style: map[string]string{"display": "flex", "align-items": "center", "gap": "12px", "padding": "11px 13px", "border": "1px solid rgba(217,164,65,.34)", "border-radius": "12px", "background": "linear-gradient(135deg,rgba(39,30,24,.92),rgba(22,25,32,.94))"}},
		combatIcon("ui/icon_attack", "†", "", "36px"),
		html.Div(html.Props{Style: map[string]string{"min-width": "0", "display": "flex", "flex-direction": "column", "gap": "2px"}},
			html.Span(html.Props{Style: map[string]string{"font-size": ".68rem", "letter-spacing": ".1em", "text-transform": "uppercase", "color": "#a89f8c"}}, html.Text("Target")),
			html.Strong(html.Props{Style: map[string]string{"font-family": "Georgia, serif", "font-size": "1.08rem", "color": "#efe6d2", "white-space": "nowrap", "overflow": "hidden", "text-overflow": "ellipsis"}}, html.Text(combatTargetLabel(snapshot.Attack))),
			html.Small(html.Props{Style: map[string]string{"color": "#d9a441", "font-size": ".78rem"}}, html.Text(preview)),
		),
	)
}

func combatGrid(snapshot CombatSnapshot, model *CombatModel, refresh ui.State[int], theme PhoneTheme, locale string) ui.Node {
	if len(snapshot.Grid) == 0 {
		return html.Div(html.Props{Class: "df-phone-combat-grid-empty", Style: map[string]string{"display": "none"}}, html.Text(""))
	}
	cols := int32(1)
	if snapshot.MiniGrid != nil && snapshot.MiniGrid.GetCols() > 0 {
		cols = snapshot.MiniGrid.GetCols()
	}
	style := map[string]string{"display": "grid", "grid-template-columns": "repeat(" + strconv.FormatInt(int64(cols), 10) + ", 1fr)", "gap": "5px", "padding": "10px", "border": "1px solid rgba(217,164,65,.24)", "border-radius": theme.BorderRadius, "background": "rgba(9,12,18,.74)"}
	cells := make([]ui.Node, 0, len(snapshot.Grid))
	for _, cell := range snapshot.Grid {
		item := cell
		cells = append(cells, combatGridCell(item, model, refresh, snapshot.CanAct, theme))
	}
	return html.Section(html.Props{Class: "df-phone-combat-grid-wrap", Aria: map[string]string{"label": "Movement grid"}},
		html.Div(html.Props{Style: map[string]string{"display": "flex", "align-items": "center", "gap": "8px", "margin": "0 2px 7px"}}, combatIcon("ui/icon_move", "◇", "", "28px"), html.Span(html.Props{Style: combatLabelStyle()}, html.Text("MOVE · "+strconv.FormatInt(int64(snapshot.MoveLeftCells), 10)+" CELLS"))),
		html.Div(html.Props{Class: "df-phone-combat-grid", Style: style}, cells...),
	)
}

func combatGridCell(cell CombatCell, model *CombatModel, refresh ui.State[int], canAct bool, theme PhoneTheme) ui.Node {
	clickable := canAct && cell.Reachable && !cell.Me && !cell.Thrall
	props := html.Props{Type: "button", Class: "df-phone-combat-cell", Disabled: !clickable, Aria: map[string]string{"label": combatCellLabel(cell)}, Style: combatCellStyle(cell, clickable, theme)}
	if clickable {
		selected := df.Cell{C: cell.Column, R: cell.Row}
		props.OnClick = ui.UseEvent(func() {
			go func() { model.ApplyAct(<-model.Move(context.Background(), &selected)); refresh.Set(refresh.Get() + 1) }()
		})
	}
	return html.Button(props, html.Text(combatCellGlyph(cell)))
}

func combatActions(snapshot CombatSnapshot, model *CombatModel, refresh ui.State[int], theme PhoneTheme, locale string) ui.Node {
	attack := combatMove(snapshot.Moves, "attack")
	end := combatMove(snapshot.Moves, "end_turn")
	actions := make([]ui.Node, 0, 2)
	if attack != nil {
		actions = append(actions, combatActionButton(model, refresh, attack, theme, "ui/icon_attack", "†", combatAttackLabel(attack, locale)))
	}
	if end != nil {
		actions = append(actions, combatActionButton(model, refresh, end, theme, "ui/icon_end_turn", "◈", end.GetLabel()))
	}
	if len(actions) == 0 {
		actions = append(actions, html.Div(html.Props{Style: map[string]string{"padding": "14px", "border": "1px solid rgba(168,159,140,.25)", "border-radius": theme.BorderRadius, "color": "#a89f8c", "text-align": "center"}}, html.Text(combatWaitingFor(snapshot.MyTurn, snapshot.Down))))
	}
	return html.Section(html.Props{Class: "df-phone-combat-actions", Aria: map[string]string{"label": "Combat actions"}, Style: map[string]string{"display": "flex", "flex-direction": "column", "gap": "9px", "margin-top": "auto"}}, actions...)
}

func combatActionButton(model *CombatModel, refresh ui.State[int], move *df.Move, theme PhoneTheme, asset, fallback, label string) ui.Node {
	click := ui.UseEvent(func() {
		go func() { model.ApplyAct(<-model.Tap(context.Background(), move)); refresh.Set(refresh.Get() + 1) }()
	})
	background := "linear-gradient(135deg,#3a2b1a,#211a15)"
	if !move.GetEnabled() {
		background = "linear-gradient(135deg,#252831,#191b22)"
	}
	style := map[string]string{"box-sizing": "border-box", "width": "100%", "min-height": theme.TouchTarget, "display": "flex", "align-items": "center", "gap": "12px", "padding": "12px 14px", "border": "1px solid rgba(217,164,65,.7)", "border-radius": theme.BorderRadius, "background": background, "color": "#f1e6d1", "font-family": "Inter, ui-sans-serif, system-ui, sans-serif", "font-size": "1rem", "font-weight": "700", "text-align": "left", "box-shadow": "0 6px 18px rgba(0,0,0,.24)", "touch-action": "manipulation"}
	if !move.GetEnabled() {
		style["border-color"] = "rgba(168,159,140,.28)"
		style["color"] = "#777b87"
	}
	children := []ui.Node{combatIcon(asset, fallback, "", "40px"), html.Span(html.Props{Style: map[string]string{"display": "flex", "flex-direction": "column", "gap": "3px"}}, html.Span(html.Props{}, html.Text(label)), combatMovePreview(move))}
	return html.Button(html.Props{Type: "button", Class: "df-phone-combat-action", OnClick: click, Disabled: !move.GetEnabled(), Aria: map[string]string{"label": combatActionAria(move, label)}, Style: style}, children...)
}

func combatFooter(snapshot CombatSnapshot, locale string) ui.Node {
	items := []string{"Character", "Journal", "Play", "Map", "Menu"}
	active := 2
	buttons := make([]ui.Node, 0, len(items))
	for index, item := range items {
		color := "#8e8e8b"
		if index == active {
			color = "#e4b357"
		}
		buttons = append(buttons, html.Div(html.Props{Style: map[string]string{"flex": "1", "display": "flex", "flex-direction": "column", "align-items": "center", "gap": "3px", "color": color, "font-size": ".62rem", "letter-spacing": ".02em"}}, html.Span(html.Props{Style: map[string]string{"font-size": "1.08rem", "line-height": "1"}}, html.Text(combatFooterGlyph(index))), html.Span(html.Props{}, html.Text(item))))
	}
	return html.Footer(html.Props{Class: "df-phone-combat-footer", Style: map[string]string{"display": "flex", "align-items": "center", "gap": "3px", "padding": "10px 2px 2px", "border-top": "1px solid rgba(217,164,65,.22)"}}, buttons...)
}

func combatBar(percent int32, color, label string) ui.Node {
	return html.Div(html.Props{Role: "progressbar", Aria: map[string]string{"label": label, "valuenow": strconv.FormatInt(int64(percent), 10), "valuemin": "0", "valuemax": "100"}, Style: map[string]string{"height": "8px", "overflow": "hidden", "border-radius": "99px", "background": "rgba(255,255,255,.1)", "box-shadow": "inset 0 1px 3px rgba(0,0,0,.55)"}}, html.Div(html.Props{Style: map[string]string{"width": strconv.FormatInt(int64(percent), 10) + "%", "height": "100%", "border-radius": "inherit", "background": color, "box-shadow": "0 0 12px " + color, "transition": "width 180ms ease-out"}}))
}

func combatIcon(asset, fallback, alt, size string) ui.Node {
	if url := ArtURL(asset); url != "" {
		return html.Img(html.Props{Src: url, Alt: alt, Aria: map[string]string{"hidden": "true"}, Style: map[string]string{"width": size, "height": size, "flex": "0 0 " + size, "object-fit": "contain"}})
	}
	return html.Span(html.Props{Aria: map[string]string{"hidden": "true"}, Style: map[string]string{"width": size, "height": size, "flex": "0 0 " + size, "display": "grid", "place-items": "center", "border": "1px solid rgba(217,164,65,.6)", "border-radius": "50%", "color": "#e2ad4e", "font-family": "Georgia, serif", "font-size": "1.35rem"}}, html.Text(fallback))
}

func combatCardStyle(theme PhoneTheme) map[string]string {
	return map[string]string{"padding": "12px 13px", "border": "1px solid rgba(217,164,65,.3)", "border-radius": theme.BorderRadius, "background": "linear-gradient(135deg,rgba(31,27,25,.94),rgba(20,24,31,.94))", "box-shadow": "0 7px 18px rgba(0,0,0,.22)"}
}

func combatCardHeadStyle() map[string]string {
	return map[string]string{"display": "flex", "align-items": "center", "justify-content": "space-between", "gap": "8px", "margin-bottom": "8px"}
}

func combatLabelStyle() map[string]string {
	return map[string]string{"font-size": ".68rem", "font-weight": "700", "letter-spacing": ".13em", "color": "#a89f8c"}
}

func combatPercent(value, max int64) int32 {
	if max <= 0 || value <= 0 {
		return 0
	}
	if value >= max {
		return 100
	}
	return int32(value * 100 / max)
}

func combatHPState(snapshot CombatSnapshot) string {
	if snapshot.Down {
		return "DOWN"
	}
	if snapshot.HPMax > 0 && snapshot.HP*2 <= snapshot.HPMax {
		return "BLOODIED"
	}
	return "READY"
}

func combatStatus(snapshot CombatSnapshot, locale string) string {
	if snapshot.Down {
		return combatDownLabel(locale)
	}
	if strings.TrimSpace(snapshot.StatusText) != "" {
		return snapshot.StatusText
	}
	if snapshot.MyTurn {
		return "Choose an action. The thrall is within reach."
	}
	return "Watch the turn strip. Your actions are ready when it is your turn."
}

func combatDownLabel(locale string) string {
	if strings.HasPrefix(strings.ToLower(locale), "es") {
		return "Estás derribado — los demás continúan"
	}
	return "You're down — the others fight on"
}

func combatTurnColor(snapshot CombatSnapshot) string {
	if snapshot.Down {
		return "#d76a61"
	}
	if snapshot.MyTurn {
		return "#e2ad4e"
	}
	return "#a89f8c"
}

func combatTargetLabel(attack *CombatAttack) string {
	if attack == nil || strings.TrimSpace(attack.Label) == "" {
		return "Drowned thrall"
	}
	return attack.Label
}

func combatMove(moves []*df.Move, id string) *df.Move {
	for _, move := range moves {
		if move != nil && move.GetMoveId() == id {
			return move
		}
	}
	return nil
}

func combatAttackLabel(move *df.Move, locale string) string {
	if move == nil || strings.TrimSpace(move.GetLabel()) == "" {
		return "Attack the drowned thrall"
	}
	return move.GetLabel()
}

func combatMovePreview(move *df.Move) ui.Node {
	preview := combatAttackPreview(move.GetPreview())
	if preview == "" {
		return html.Small(html.Props{Style: map[string]string{"font-size": ".75rem", "font-weight": "500", "color": "#a89f8c"}}, html.Text(MoveReason(MoveSnapshot{Enabled: move.GetEnabled(), Reason: move.GetReason()})))
	}
	return html.Small(html.Props{Style: map[string]string{"font-size": ".75rem", "font-weight": "500", "color": "#d9a441"}}, html.Text(preview))
}

func combatAttackPreview(preview *df.MovePreview) string {
	if preview == nil {
		return ""
	}
	parts := make([]string, 0, 3)
	if preview.GetModifier() != 0 {
		parts = append(parts, signedNumber(preview.GetModifier())+" to hit")
	}
	if preview.GetVs() > 0 {
		parts = append(parts, "AC "+numberText(preview.GetVs()))
	}
	if chance := preview.GetPSuccess(); chance > 0 {
		parts = append(parts, strconv.Itoa(int(chance*100+0.5))+"%")
	}
	if damage := preview.GetDamage(); damage != nil && damage.GetDice() != "" {
		damageText := damage.GetDice()
		if damage.GetBonus() > 0 {
			damageText += "+" + numberText(damage.GetBonus())
		}
		parts = append(parts, damageText)
	}
	return strings.Join(parts, " · ")
}

func combatActionAria(move *df.Move, label string) string {
	if move.GetEnabled() {
		return label
	}
	return label + ". " + MoveReason(MoveSnapshot{Enabled: false, Reason: move.GetReason()})
}

func combatCellLabel(cell CombatCell) string {
	if cell.Me {
		return "Your position"
	}
	if cell.Thrall {
		return "Drowned thrall"
	}
	if cell.Reachable {
		return "Move to reachable cell"
	}
	return "Blocked cell"
}

func combatCellGlyph(cell CombatCell) string {
	if cell.Me {
		return "●"
	}
	if cell.Thrall {
		return "✦"
	}
	if cell.Reachable {
		return "·"
	}
	return ""
}

func combatCellStyle(cell CombatCell, clickable bool, theme PhoneTheme) map[string]string {
	background := "rgba(34,39,48,.78)"
	border := "1px solid rgba(168,159,140,.18)"
	color := "#545a64"
	if cell.Walkable {
		background = "rgba(44,49,57,.88)"
	}
	if cell.Reachable {
		background = "rgba(58,163,154,.17)"
		border = "1px solid rgba(58,163,154,.65)"
		color = theme.Teal
	}
	if cell.Me {
		background = "rgba(217,164,65,.23)"
		border = "1px solid rgba(217,164,65,.9)"
		color = "#f1c46c"
	}
	if cell.Thrall {
		background = "rgba(179,55,47,.28)"
		border = "1px solid rgba(210,92,82,.9)"
		color = "#e99486"
	}
	if !clickable && cell.Reachable && !cell.Me && !cell.Thrall {
		color = "#548c87"
	}
	return map[string]string{"min-height": "42px", "padding": "0", "border": border, "border-radius": "8px", "background": background, "color": color, "font-size": "1.1rem", "font-weight": "700", "touch-action": "manipulation"}
}

func combatFooterGlyph(index int) string {
	return []string{"♙", "▤", "●", "⌖", "☰"}[index]
}
