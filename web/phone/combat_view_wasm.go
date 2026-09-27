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

// combatStyledScreen renders only the combat content inside PhoneFrame. The
// frame owns the header and bottom navigation shared by every phone screen.
func combatStyledScreen(model *CombatModel, locale string, _ ...ui.Node) ui.Node {
	if locale == "" {
		locale = "en"
	}
	if model == nil {
		return html.Section(html.Props{Class: "df-phone-combat", Role: "region"}, html.Text(T(locale, "phone.combat.unavailable", nil)))
	}
	refresh := ui.UseState(0)
	snapshot := model.Snapshot()
	theme := DefaultPhoneTheme()
	if snapshot.CanAct && model.MapOpen() {
		return html.Section(html.Props{Key: "combat-map", Class: "df-phone-combat is-map", Role: "region", Aria: map[string]string{"label": "Combat"}, Style: combatContentStyle(theme)},
			ui.CreateElement(combatMapPanel, combatMapProps{model: model, refresh: refresh, revision: refresh.Get(), timer: combatMapTimer(snapshot), percent: combatPercent(snapshot.TimerRemaining, snapshot.TimerTotal)}),
		)
	}
	return html.Section(html.Props{Key: "combat-list", Class: "df-phone-combat", Role: "region", Aria: map[string]string{"label": "Combat"}, Style: combatContentStyle(theme)},
		combatProfile(snapshot, theme),
		combatTurnStrip(snapshot, theme),
		combatStatusPanel(snapshot, locale),
		combatTarget(snapshot, theme),
		combatWatchMap(snapshot, model, refresh),
		combatActions(snapshot, model, refresh, locale),
	)
}

// combatMapTimer is the compact turn timer of map mode; empty when turn
// timers are off.
func combatMapTimer(snapshot CombatSnapshot) string {
	if snapshot.TimerTotal <= 0 {
		return ""
	}
	return combatTimerText(snapshot)
}

// combatWatchMap shows the waiting seat the same battlefield read-only.
func combatWatchMap(snapshot CombatSnapshot, model *CombatModel, refresh ui.State[int]) ui.Node {
	if snapshot.CanAct || snapshot.MiniGrid == nil || len(snapshot.MiniGrid.GetTokens()) == 0 {
		return nil
	}
	return ui.CreateElement(combatMapPanel, combatMapProps{model: model, refresh: refresh, revision: refresh.Get(), watching: true})
}

func combatContentStyle(theme PhoneTheme) map[string]string {
	return map[string]string{
		"display": "flex", "flex-direction": "column", "gap": "10px", "width": "100%", "max-width": "calc(100vw - 28px)", "box-sizing": "border-box", "min-width": "0",
		"padding-bottom": "10px", "color": theme.Parchment, "font-family": theme.Sans,
	}
}

func combatProfile(snapshot CombatSnapshot, theme PhoneTheme) ui.Node {
	name, role, portraitURL := "Your hero", "Combatant", ""
	if snapshot.Character != nil {
		if value := strings.TrimSpace(snapshot.Character.GetName()); value != "" {
			name = value
		}
		if value := strings.TrimSpace(snapshot.Character.GetClassName()); value != "" {
			role = value
		}
		// The server sends a logical art name (e.g. "ui/species_human") until
		// the seat's generated portrait exists; portraitSrc resolves it
		// through the gRPC art source instead of using it as a raw <img> src
		// (which 404s on a literal /ui/species_human request).
		portraitURL = portraitSrc(strings.TrimSpace(snapshot.Character.GetPortraitUrl()))
	}
	portrait := combatPortrait(name, portraitURL, theme)
	status := combatHPState(snapshot)
	if len(snapshot.Statuses) > 0 {
		status = strings.Join(snapshot.Statuses, " · ")
	}
	return html.Section(html.Props{Class: "df-phone-combat-profile", Style: map[string]string{
		"display": "flex", "align-items": "center", "gap": "11px", "padding": "10px",
		"border": "1px solid rgba(217,164,65,.62)", "border-radius": theme.BorderRadius,
		"background": "linear-gradient(135deg, rgba(37,31,25,.96), rgba(19,23,31,.96))",
		"box-shadow": "inset 0 0 18px rgba(217,164,65,.08)",
	}}, portrait, html.Div(html.Props{Style: map[string]string{"min-width": "0", "flex": "1 1 auto"}},
		html.H1(html.Props{Style: map[string]string{"margin": "0", "font-family": theme.Serif, "font-size": "22px", "line-height": "1.05", "white-space": "nowrap", "overflow": "hidden", "text-overflow": "ellipsis"}}, html.Text(name)),
		html.P(html.Props{Style: map[string]string{"margin": "3px 0 7px", "color": theme.GoldBright, "font-size": "13px"}}, html.Text(role)),
		html.Div(html.Props{Style: map[string]string{"display": "flex", "justify-content": "space-between", "gap": "8px", "margin-bottom": "4px", "color": theme.Muted, "font-size": "11px"}}, html.Span(html.Props{}, html.Text(T("en", "phone.combat.vitals", nil))), html.Span(html.Props{}, html.Text(status))),
		HPBar(snapshot.HP, snapshot.HPMax),
	))
}

func combatPortrait(name, portraitURL string, theme PhoneTheme) ui.Node {
	style := map[string]string{"width": "66px", "height": "66px", "flex": "0 0 66px", "display": "grid", "place-items": "center", "overflow": "hidden", "border": "1px solid " + theme.GoldBright, "border-radius": "8px", "background": theme.PanelRaised}
	if portraitURL != "" {
		return html.Div(html.Props{Style: style}, html.Img(html.Props{Src: portraitURL, Alt: name, Style: map[string]string{"width": "100%", "height": "100%", "object-fit": "cover"}}))
	}
	return html.Div(html.Props{Role: "img", Aria: map[string]string{"label": name}, Style: style}, html.Text(combatInitials(name)))
}

func combatInitials(value string) string {
	for _, r := range strings.TrimSpace(value) {
		return string(r)
	}
	return "?"
}

func combatTurnStrip(snapshot CombatSnapshot, theme PhoneTheme) ui.Node {
	percent := combatPercent(snapshot.TimerRemaining, snapshot.TimerTotal)
	caption := "TURN TIMER"
	if snapshot.TimerFrozen {
		caption += " - PAUSED"
	}
	turn := snapshot.TurnLabel
	if turn == "" {
		turn = "Combat"
	}
	return html.Section(html.Props{Class: "df-phone-combat-turn", Aria: map[string]string{"label": caption}, Style: combatCardStyle(theme)},
		html.Div(html.Props{Style: combatCardHeadStyle()}, html.Div(html.Props{}, html.Span(html.Props{Style: combatLabelStyle()}, html.Text(caption)), html.P(html.Props{Style: map[string]string{"margin": "3px 0 0", "font-family": theme.Serif, "font-size": "20px", "color": theme.Parchment}}, html.Text(turn))), html.Strong(html.Props{Style: map[string]string{"font-family": theme.Serif, "font-size": "20px", "color": theme.GoldBright}}, html.Text(combatTimerText(snapshot)))),
		combatBar(percent, theme.Gold, caption),
	)
}

func combatTimerText(snapshot CombatSnapshot) string {
	if snapshot.TimerLabel != "" {
		return snapshot.TimerLabel
	}
	return "-"
}

func combatStatusPanel(snapshot CombatSnapshot, locale string) ui.Node {
	if snapshot.Down {
		return ResultBanner(ResultBannerModel{Message: combatDownLabel(locale)})
	}
	message := combatStatus(snapshot, locale)
	if !snapshot.MyTurn {
		return NarrationCard("Combat", message, "")
	}
	theme := DefaultPhoneTheme()
	return html.P(html.Props{Class: "df-phone-combat-status", Role: "status", Aria: map[string]string{"live": "polite"}, Style: map[string]string{
		"margin": "0", "padding": "8px 12px", "border-left": "2px solid " + theme.Gold,
		"color": theme.Muted, "font-family": theme.Serif, "font-size": "16px", "font-style": "italic",
	}}, html.Text(message))
}

func combatTarget(snapshot CombatSnapshot, theme PhoneTheme) ui.Node {
	if snapshot.Attack == nil {
		return nil
	}
	preview := combatAttackPreview(snapshot.Attack.Preview)
	children := []ui.Node{choiceIcon(combatIconName("ui/icon_attack", "⚔")), html.Div(html.Props{Style: map[string]string{"min-width": "0", "flex": "1 1 auto"}},
		html.Small(html.Props{Style: map[string]string{"display": "block", "color": theme.Muted, "font-family": theme.Sans, "font-size": "11px", "letter-spacing": ".1em", "text-transform": "uppercase"}}, html.Text(T("en", "phone.combat.target", nil))),
		html.Strong(html.Props{Style: map[string]string{"display": "block", "margin-top": "2px", "color": theme.Parchment, "font-family": theme.Serif, "font-size": "19px", "white-space": "nowrap", "overflow": "hidden", "text-overflow": "ellipsis"}}, html.Text(combatTargetLabel(snapshot.Attack))),
	)}
	if preview != "" {
		children = append(children, html.Small(html.Props{Style: map[string]string{"max-width": "112px", "margin-left": "auto", "overflow": "hidden", "color": theme.GoldBright, "font-size": "11px", "text-align": "right", "text-overflow": "ellipsis", "white-space": "nowrap"}}, html.Text(preview)))
	}
	return html.Section(html.Props{Class: "df-phone-combat-target", Aria: map[string]string{"label": "Current target"}, Style: map[string]string{
		"display": "flex", "align-items": "center", "gap": "10px", "padding": "9px 11px", "border": "1px solid rgba(217,164,65,.4)",
		"border-radius": theme.BorderRadius, "background": "rgba(20,24,31,.92)",
	}}, children...)
}

func combatActions(snapshot CombatSnapshot, model *CombatModel, refresh ui.State[int], locale string) ui.Node {
	items := make([]ui.Node, 0, len(snapshot.Moves))
	for _, move := range snapshot.Moves {
		if move != nil {
			items = append(items, combatMoveChoice(model, refresh, move, locale))
		}
	}
	if len(items) == 0 {
		if snapshot.Down {
			return nil
		}
		return html.Section(html.Props{Class: "df-phone-combat-actions", Aria: map[string]string{"label": "Combat status"}, Style: map[string]string{"display": "grid", "gap": "8px"}}, NarrationCard("Combat", combatStatus(snapshot, locale), ""))
	}
	return html.Section(html.Props{Class: "df-phone-combat-actions", Aria: map[string]string{"label": "Combat actions"}, Style: map[string]string{"display": "grid", "gap": "8px"}}, items...)
}

func combatMoveChoice(model *CombatModel, refresh ui.State[int], move *df.Move, locale string) ui.Node {
	label := move.GetLabel()
	if move.GetMoveId() == "attack" {
		label = combatAttackLabel(move, locale)
	}
	row := ChoiceRowModel{ID: move.GetMoveId(), Label: label, Reason: MoveReason(MoveSnapshot{Enabled: move.GetEnabled(), Reason: move.GetReason()}), Icon: combatMoveIcon(move.GetMoveId()), Enabled: move.GetEnabled(), Highlighted: move.GetMoveId() == "attack" && move.GetEnabled()}
	tap := ui.UseEvent(func() {
		if move.GetMoveId() == "move" && len(model.Snapshot().MiniGrid.GetPaths()) > 0 {
			model.OpenMap()
			refresh.Set(refresh.Get() + 1)
			return
		}
		go func() { model.ApplyAct(<-model.Tap(context.Background(), move)); refresh.Set(refresh.Get() + 1) }()
	})
	choice := combatChoiceRow(row, tap)
	preview := combatAttackPreview(move.GetPreview())
	if preview == "" {
		return choice
	}
	theme := DefaultPhoneTheme()
	return html.Div(html.Props{Style: map[string]string{"display": "grid", "gap": "3px"}}, choice, html.Small(html.Props{Style: map[string]string{"margin-left": "46px", "color": theme.GoldBright, "font-size": "12px"}}, html.Text(preview)))
}

func combatChoiceRow(row ChoiceRowModel, tap ui.Handler) ui.Node {
	theme := DefaultPhoneTheme()
	style := map[string]string{
		"width": "100%", "min-height": "52px", "box-sizing": "border-box", "display": "flex",
		"align-items": "center", "gap": "12px", "padding": "10px 14px", "border-radius": "10px",
		"border": "1px solid rgba(168,159,140,.48)", "background": "rgba(23,26,35,.9)",
		"color": theme.Parchment, "font-family": theme.Serif, "font-size": "17px", "text-align": "left",
		"line-height": "1.18", "touch-action": "manipulation", "transition": "border-color " + theme.Transition,
	}
	if row.Highlighted {
		style["border"] = "1px solid " + theme.GoldBright
		style["box-shadow"] = "inset 0 0 14px rgba(217,164,65,.16), 0 0 12px rgba(217,164,65,.12)"
	}
	if !row.Enabled {
		style["color"] = theme.Muted
		style["opacity"] = ".66"
	}
	children := []ui.Node{choiceIcon(row.Icon)}
	text := html.Span(html.Props{Style: map[string]string{"min-width": "0", "display": "flex", "flex-direction": "column", "gap": "3px"}}, html.Text(row.Label))
	if reason := ChoiceRowReason(row); !row.Enabled && reason != "" {
		text = html.Span(html.Props{Style: map[string]string{"min-width": "0", "display": "flex", "flex-direction": "column", "gap": "3px"}}, html.Text(row.Label), html.Small(html.Props{Style: map[string]string{"font-family": theme.Sans, "font-size": "11px", "color": theme.Muted}}, html.Text(reason)))
	}
	children = append(children, text)
	return html.Button(html.Props{Type: "button", Class: choiceClass(row), Disabled: !row.Enabled, OnClick: tap, Aria: map[string]string{"label": choiceLabel(row)}, Style: style}, children...)
}

func combatMoveIcon(moveID string) string {
	asset := moveArtAsset(moveID)
	fallback := "•"
	switch moveID {
	case "attack":
		fallback = "⚔"
	case "move":
		fallback = "◇"
	case "end_turn":
		fallback = "◈"
	}
	return combatIconName(asset, fallback)
}

func combatIconName(asset, fallback string) string {
	if asset != "" && ArtURL(asset) != "" {
		return asset
	}
	return fallback
}

func combatBar(percent int32, color, label string) ui.Node {
	return html.Div(html.Props{Role: "progressbar", Aria: map[string]string{"label": label, "valuenow": strconv.FormatInt(int64(percent), 10), "valuemin": "0", "valuemax": "100"}, Style: map[string]string{"height": "8px", "overflow": "hidden", "border-radius": "99px", "background": "rgba(255,255,255,.1)", "box-shadow": "inset 0 1px 3px rgba(0,0,0,.55)"}}, html.Div(html.Props{Style: map[string]string{"width": strconv.FormatInt(int64(percent), 10) + "%", "height": "100%", "border-radius": "inherit", "background": color, "box-shadow": "0 0 12px " + color}}))
}

func combatCardStyle(theme PhoneTheme) map[string]string {
	return map[string]string{"padding": "10px 12px", "border": "1px solid rgba(217,164,65,.34)", "border-radius": theme.BorderRadius, "background": "rgba(19,23,31,.9)", "box-shadow": "inset 0 0 18px rgba(217,164,65,.04)"}
}

func combatCardHeadStyle() map[string]string {
	return map[string]string{"display": "flex", "align-items": "center", "justify-content": "space-between", "gap": "8px", "margin-bottom": "7px"}
}

func combatLabelStyle() map[string]string {
	return map[string]string{"font-size": "11px", "font-weight": "700", "letter-spacing": ".12em", "color": DefaultPhoneTheme().Muted}
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

func combatTargetLabel(attack *CombatAttack) string {
	if attack == nil || strings.TrimSpace(attack.Label) == "" {
		return "Drowned Thrall"
	}
	return attack.Label
}

func combatAttackLabel(move *df.Move, locale string) string {
	if move == nil || strings.TrimSpace(move.GetLabel()) == "" {
		if strings.HasPrefix(strings.ToLower(locale), "es") {
			return "Atacar al ahogado"
		}
		return "Attack the drowned thrall"
	}
	return move.GetLabel()
}

func combatAttackPreview(preview *df.MovePreview) string {
	if preview == nil {
		return ""
	}
	parts := make([]string, 0, 4)
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
