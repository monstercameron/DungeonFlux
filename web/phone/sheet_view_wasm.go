//go:build js && wasm

package phone

import (
	"strconv"
	"strings"

	"github.com/monstercameron/GoWebComponents/v6/html"
	"github.com/monstercameron/GoWebComponents/v6/router"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

// sheetTabID identifies one of the character sheet's four sections.
type sheetTabID string

const (
	sheetTabActions   sheetTabID = "actions"
	sheetTabInventory sheetTabID = "inventory"
	sheetTabSpells    sheetTabID = "spells"
	sheetTabInfo      sheetTabID = "info"
)

// SheetScreen renders the character sheet inside the shared phone frame.
func SheetScreen(model *SheetModel) router.Component {
	return func(_ router.Attrs) *router.Element {
		state := model.Snapshot()
		locale := state.Locale
		if locale == "" {
			locale = "en"
		}
		tab := ui.UseState(sheetTabActions)
		selectActions := ui.UseEvent(func() { tab.Set(sheetTabActions) })
		selectInventory := ui.UseEvent(func() { tab.Set(sheetTabInventory) })
		selectSpells := ui.UseEvent(func() { tab.Set(sheetTabSpells) })
		selectInfo := ui.UseEvent(func() { tab.Set(sheetTabInfo) })
		taps := map[sheetTabID]ui.Handler{
			sheetTabActions: selectActions, sheetTabInventory: selectInventory,
			sheetTabSpells: selectSpells, sheetTabInfo: selectInfo,
		}
		return sheetPage(locale, state, tab.Get(), taps)
	}
}

func sheetPage(locale string, state SheetSnapshot, active sheetTabID, taps map[sheetTabID]ui.Handler) ui.Node {
	return html.Main(html.Props{Class: "df-phone-sheet", Role: "main", Style: sheetPageStyle()},
		sheetHero(locale, state),
		sheetStats(locale, state),
		sheetTabs(locale, active, taps),
		sheetTabPanel(locale, state, active),
		sheetFooter(state),
	)
}

func sheetTabPanel(locale string, state SheetSnapshot, active sheetTabID) ui.Node {
	switch active {
	case sheetTabInventory:
		return sheetInventoryPanel(locale, state)
	case sheetTabSpells:
		return sheetEmptyPanel(locale, T(locale, "sheet.tab_spells", nil), T(locale, "sheet.no_spells", nil))
	case sheetTabInfo:
		return html.Section(html.Props{Class: "df-phone-sheet-section", Style: map[string]string{"display": "grid", "gap": "7px"}},
			sheetSectionHeading(T(locale, "sheet.tab_info", nil), "", false), sheetBuildDetails(locale, state))
	default:
		return html.Div(html.Props{Style: map[string]string{"display": "grid", "gap": "12px"}}, sheetActionsPanel(locale, state), sheetOtherPanel(locale))
	}
}

// sheetInventoryPanel lists the class's starting-equipment option A (RULES-008,
// internal/game/rules.Equipment), so the demo shows real seeded gear rather
// than placeholder copy.
func sheetInventoryPanel(locale string, state SheetSnapshot) ui.Node {
	if len(state.Equipment) == 0 {
		return sheetEmptyPanel(locale, T(locale, "inventory.title", nil), T(locale, "inventory.empty", nil))
	}
	theme := DefaultPhoneTheme()
	rows := make([]ui.Node, 0, len(state.Equipment))
	for _, item := range state.Equipment {
		rows = append(rows, sheetInventoryRow(locale, item, theme))
	}
	return html.Section(html.Props{Class: "df-phone-sheet-section", Aria: map[string]string{"label": T(locale, "inventory.title", nil)}, Style: map[string]string{"display": "grid", "gap": "7px"}},
		sheetSectionHeading(T(locale, "inventory.title", nil), "", false),
		html.Div(html.Props{Style: map[string]string{"display": "grid", "gap": "7px"}}, rows...),
	)
}

func sheetInventoryRow(locale string, item SheetEquipmentItem, theme PhoneTheme) ui.Node {
	children := []ui.Node{html.Div(html.Props{Style: map[string]string{"min-width": "0", "display": "grid", "gap": "1px"}},
		html.Strong(html.Props{Style: map[string]string{"color": theme.Parchment, "font-family": theme.Serif, "font-size": "16px"}}, html.Text(item.Name)),
		html.Small(html.Props{Style: map[string]string{"overflow": "hidden", "color": theme.Muted, "font-family": theme.Sans, "font-size": "11px", "text-overflow": "ellipsis"}}, html.Text(item.Description)),
	)}
	if item.Worn {
		children = append(children, html.Span(html.Props{Style: map[string]string{
			"padding": "2px 8px", "border": "1px solid " + theme.GoldBright, "border-radius": "999px",
			"color": theme.GoldBright, "font-family": theme.Sans, "font-size": "10px", "letter-spacing": ".04em", "white-space": "nowrap",
		}}, html.Text(strings.ToUpper(T(locale, "inventory.worn", nil)))))
	}
	return html.Div(html.Props{Class: "df-phone-inventory-row", Style: map[string]string{
		"display": "flex", "align-items": "center", "justify-content": "space-between", "gap": "10px",
		"min-height": "48px", "padding": "9px 13px", "border": "1px solid rgba(168,159,140,.42)", "border-radius": "9px",
		"background": "rgba(23,26,35,.92)",
	}}, children...)
}

func sheetHero(locale string, state SheetSnapshot) ui.Node {
	name := state.Name
	if name == "" {
		name = SheetName(locale, "", state.Class)
	}
	role := sheetRole(state.Species, state.Class)
	level := ""
	if state.Level > 0 {
		level = "Level " + strconv.Itoa(int(state.Level))
	}
	if state.AC > 0 {
		if level != "" {
			level += "  ·  "
		}
		level += "AC " + strconv.Itoa(int(state.AC))
	}
	if level == "" {
		level = "—"
	}
	return html.Section(html.Props{Class: "df-phone-sheet-hero", Style: map[string]string{
		"display": "grid", "grid-template-columns": "112px minmax(0, 1fr)", "gap": "14px", "align-items": "center",
		"padding": "12px", "border": "1px solid rgba(217,164,65,.7)", "border-radius": "12px",
		"background": "linear-gradient(135deg, rgba(38,34,36,.98), rgba(17,22,29,.96))",
		"box-shadow": "inset 0 0 22px rgba(217,164,65,.08)",
	}}, sheetPortrait(state), html.Div(html.Props{Style: map[string]string{"min-width": "0", "display": "grid", "gap": "4px"}},
		html.H1(html.Props{Style: map[string]string{"margin": "0", "overflow": "hidden", "color": "#efe6d2", "font-family": "Cormorant Garamond, Cinzel, Georgia, serif", "font-size": "26px", "line-height": "1.05", "text-overflow": "ellipsis", "white-space": "nowrap"}}, html.Text(name)),
		html.P(html.Props{Style: map[string]string{"margin": "0", "color": "#d9a441", "font-family": "Cormorant Garamond, Cinzel, Georgia, serif", "font-size": "16px"}}, html.Text(role)),
		html.P(html.Props{Style: map[string]string{"margin": "0 0 4px", "color": "#a89f8c", "font-family": "Inter, system-ui, sans-serif", "font-size": "12px"}}, html.Text(level)),
		sheetHPBar(locale, state),
	))
}

func sheetHPBar(locale string, state SheetSnapshot) ui.Node {
	if state.HPMax <= 0 {
		return html.P(html.Props{Class: "df-phone-sheet-unknown", Style: map[string]string{"margin": "0", "color": "#a89f8c", "font-size": "12px"}}, html.Text(T(locale, "sheet.hp_none", nil)))
	}
	return HPBar(state.HP, state.HPMax)
}

func sheetStats(locale string, state SheetSnapshot) ui.Node {
	if len(state.Abilities) == 0 {
		return sheetEmptyPanel(locale, "Ability scores", "Your rolled abilities will appear here.")
	}
	return html.Section(html.Props{Class: "df-phone-sheet-section", Aria: map[string]string{"label": "Ability scores"}, Style: map[string]string{"display": "grid", "gap": "7px"}},
		sheetSectionHeading("Ability scores", "", false), StatRow(state.Abilities), sheetBuildDetails(locale, state),
	)
}

func sheetBuildDetails(locale string, state SheetSnapshot) ui.Node {
	saves := "—"
	if len(state.SaveProficiencies) > 0 {
		saves = strings.ToUpper(strings.Join(state.SaveProficiencies, ", "))
	}
	skills := "—"
	if len(state.SkillProficiencies) > 0 {
		items := make([]string, 0, len(state.SkillProficiencies))
		for _, skill := range SortedSkillProficiencies(state.SkillProficiencies) {
			items = append(items, titleCase(strings.ReplaceAll(skill, "_", " "))+" ("+state.SkillProficiencies[skill]+")")
		}
		skills = strings.Join(items, ", ")
	}
	attack := state.AttackName
	if attack == "" {
		attack = "—"
	}
	if state.AttackDice != "" {
		attack += " · +" + strconv.Itoa(int(state.AttackBonus)) + " · " + state.AttackDice + " " + state.AttackDamageType
	}
	return html.Div(html.Props{Style: map[string]string{"display": "grid", "gap": "3px", "padding-top": "2px", "color": DefaultPhoneTheme().Muted, "font-family": "Inter, system-ui, sans-serif", "font-size": "12px"}},
		html.Div(html.Props{}, html.Strong(html.Props{}, html.Text(localizedSheet(locale, "Saves", "Salvaciones")+": ")), html.Text(saves)),
		html.Div(html.Props{}, html.Strong(html.Props{}, html.Text(localizedSheet(locale, "Skills", "Habilidades")+": ")), html.Text(skills)),
		html.Div(html.Props{}, html.Strong(html.Props{}, html.Text(localizedSheet(locale, "Attack", "Ataque")+": ")), html.Text(attack)),
	)
}

func sheetTabs(locale string, active sheetTabID, taps map[sheetTabID]ui.Handler) ui.Node {
	labels := []struct {
		id   sheetTabID
		icon string
		key  string
	}{
		{sheetTabActions, "⚔", "sheet.tab_actions"}, {sheetTabInventory, "▣", "sheet.tab_inventory"},
		{sheetTabSpells, "✦", "sheet.tab_spells"}, {sheetTabInfo, "i", "sheet.tab_info"},
	}
	items := make([]ui.Node, 0, len(labels))
	for _, item := range labels {
		label := T(locale, item.key, nil)
		selected := item.id == active
		style := map[string]string{
			"min-height": "44px", "display": "grid", "gap": "2px", "place-items": "center", "padding": "5px 2px",
			"border": "0", "border-bottom": "1px solid rgba(168,159,140,.3)", "background": "transparent",
			"color": "#a89f8c", "font-family": "Inter, system-ui, sans-serif", "font-size": "10px", "touch-action": "manipulation",
		}
		if selected {
			style["color"] = "#e7c27a"
			style["border-bottom-color"] = "#d9a441"
		}
		items = append(items, html.Button(html.Props{Type: "button", Class: "df-phone-sheet-tab", OnClick: taps[item.id], Aria: map[string]string{"label": label, "selected": strconv.FormatBool(selected)}, Style: style}, html.Span(html.Props{Style: map[string]string{"font-family": "Georgia, serif", "font-size": "19px", "line-height": "1"}}, html.Text(item.icon)), html.Span(html.Props{}, html.Text(label))))
	}
	return html.Nav(html.Props{Class: "df-phone-sheet-tabs", Aria: map[string]string{"label": "Character sheet sections"}, Style: map[string]string{"display": "grid", "grid-template-columns": "repeat(4, minmax(0, 1fr))", "gap": "4px", "border-bottom": "1px solid rgba(168,159,140,.3)"}}, items...)
}

func sheetActionsPanel(locale string, state SheetSnapshot) ui.Node {
	actions := SheetActions(state.Class)
	items := make([]ui.Node, 0, len(actions))
	for _, action := range actions {
		items = append(items, sheetActionRow(action))
	}
	return html.Section(html.Props{Class: "df-phone-sheet-section", Aria: map[string]string{"label": sheetCombatActions(locale)}, Style: map[string]string{"display": "grid", "gap": "7px"}},
		sheetSectionHeading(sheetCombatActions(locale), sheetActionHint(state), false), html.Div(html.Props{Style: map[string]string{"display": "grid", "gap": "7px"}}, items...),
	)
}

func sheetOtherPanel(locale string) ui.Node {
	items := []SheetAction{
		{ID: "use_item", Label: "Use Item", Icon: "□", Enabled: true},
		{ID: "help_ally", Label: "Help Ally", Icon: "+", Enabled: true},
		{ID: "ready", Label: "Ready", Icon: "⌛", Enabled: true},
		{ID: "dash", Label: "Dash", Icon: "↗", Enabled: true},
	}
	children := make([]ui.Node, 0, len(items))
	for _, item := range items {
		children = append(children, sheetOtherAction(item))
	}
	return html.Section(html.Props{Class: "df-phone-sheet-section", Aria: map[string]string{"label": sheetOther(locale)}, Style: map[string]string{"display": "grid", "gap": "7px"}},
		sheetSectionHeading(sheetOther(locale), "", false), html.Div(html.Props{Style: map[string]string{"display": "grid", "grid-template-columns": "repeat(2, minmax(0, 1fr))", "gap": "7px"}}, children...),
	)
}

func sheetActionRow(action SheetAction) ui.Node {
	theme := DefaultPhoneTheme()
	return html.Button(html.Props{Type: "button", Class: "df-phone-sheet-action", Disabled: !action.Enabled, Aria: map[string]string{"label": action.Label}, Style: map[string]string{
		"width": "100%", "min-height": "50px", "display": "flex", "align-items": "center", "gap": "11px", "padding": "8px 13px",
		"border": "1px solid rgba(168,159,140,.42)", "border-radius": "9px", "background": "rgba(23,26,35,.92)", "color": theme.Parchment, "text-align": "left",
	}}, html.Span(html.Props{Style: map[string]string{"width": "28px", "height": "28px", "flex": "0 0 28px", "display": "grid", "place-items": "center", "border": "1px solid rgba(217,164,65,.5)", "border-radius": "50%", "color": theme.GoldBright, "font-family": theme.Serif, "font-size": "17px"}}, html.Text(action.Icon)), html.Span(html.Props{Style: map[string]string{"min-width": "0", "display": "grid", "gap": "1px"}}, html.Strong(html.Props{Style: map[string]string{"font-family": theme.Serif, "font-size": "17px", "line-height": "1.05"}}, html.Text(action.Label)), html.Small(html.Props{Style: map[string]string{"overflow": "hidden", "color": theme.Muted, "font-family": theme.Sans, "font-size": "11px", "text-overflow": "ellipsis", "white-space": "nowrap"}}, html.Text(action.Detail))))
}

func sheetOtherAction(action SheetAction) ui.Node {
	theme := DefaultPhoneTheme()
	return html.Button(html.Props{Type: "button", Class: "df-phone-sheet-other", Disabled: !action.Enabled, Aria: map[string]string{"label": action.Label}, Style: map[string]string{
		"min-height": "52px", "display": "grid", "place-items": "center", "gap": "3px", "padding": "6px 3px", "border": "1px solid rgba(168,159,140,.38)", "border-radius": "8px", "background": "rgba(23,26,35,.88)", "color": theme.Parchment,
	}}, html.Span(html.Props{Style: map[string]string{"color": theme.Muted, "font-family": theme.Serif, "font-size": "20px", "line-height": "1"}}, html.Text(action.Icon)), html.Span(html.Props{Style: map[string]string{"font-family": theme.Serif, "font-size": "15px"}}, html.Text(action.Label)))
}

func sheetFooter(state SheetSnapshot) ui.Node {
	if len(state.Conditions) == 0 && strings.TrimSpace(state.Hook) == "" && strings.TrimSpace(state.StatusText) == "" {
		return nil
	}
	text := strings.TrimSpace(state.StatusText)
	if text == "" {
		text = strings.Join(state.Conditions, " · ")
	}
	if text == "" {
		text = state.Hook
	}
	return html.P(html.Props{Class: "df-phone-sheet-footer", Role: "status", Style: map[string]string{"margin": "0", "padding": "9px 11px", "border-left": "2px solid #d9a441", "background": "rgba(18,22,29,.72)", "color": "#a89f8c", "font-family": "Cormorant Garamond, Georgia, serif", "font-size": "15px", "font-style": "italic", "line-height": "1.25"}}, html.Text(text))
}

func sheetSectionHeading(title, detail string, compact bool) ui.Node {
	theme := DefaultPhoneTheme()
	size := "19px"
	if compact {
		size = "16px"
	}
	children := []ui.Node{html.Strong(html.Props{Style: map[string]string{"color": theme.Parchment, "font-family": theme.Serif, "font-size": size}}, html.Text(title))}
	if detail != "" {
		children = append(children, html.Small(html.Props{Style: map[string]string{"color": theme.Muted, "font-family": theme.Sans, "font-size": "11px"}}, html.Text(detail)))
	}
	return html.Div(html.Props{Style: map[string]string{"display": "flex", "align-items": "baseline", "justify-content": "space-between", "gap": "8px"}}, children...)
}

func sheetEmptyPanel(locale, title, message string) ui.Node {
	return html.Section(html.Props{Class: "df-phone-sheet-empty", Aria: map[string]string{"label": title}, Style: map[string]string{"display": "grid", "gap": "4px", "padding": "10px 12px", "border": "1px dashed rgba(168,159,140,.42)", "border-radius": "9px", "color": "#a89f8c"}}, sheetSectionHeading(title, "", true), html.P(html.Props{Style: map[string]string{"margin": "0", "font-family": "Cormorant Garamond, Georgia, serif", "font-size": "15px"}}, html.Text(message)))
}

func sheetPortrait(state SheetSnapshot) ui.Node {
	style := map[string]string{"width": "112px", "height": "112px", "overflow": "hidden", "display": "grid", "place-items": "center", "border": "1px solid #d9a441", "border-radius": "8px", "background": "linear-gradient(135deg, #2b2525, #121923)", "color": "#e7c27a", "font-family": "Cormorant Garamond, Georgia, serif", "font-size": "35px"}
	if src := portraitSrc(state.PortraitURL); src != "" {
		return html.Div(html.Props{Class: "df-phone-portrait", Style: style}, html.Img(html.Props{Src: src, Alt: state.Name, Style: map[string]string{"width": "100%", "height": "100%", "object-fit": "cover"}}))
	}
	if url := ArtURL(speciesArtAsset(state.Species)); url != "" {
		return html.Div(html.Props{Class: "df-phone-portrait", Style: style}, html.Img(html.Props{Src: url, Alt: state.Species, Style: map[string]string{"width": "100%", "height": "100%", "object-fit": "cover"}}))
	}
	return html.Div(html.Props{Class: "df-phone-portrait df-phone-portrait-fallback", Role: "img", Aria: map[string]string{"label": state.Name}, Style: style}, html.Text(sheetInitials(state.Name)))
}

func sheetRole(species, className string) string {
	parts := make([]string, 0, 2)
	if value := strings.TrimSpace(species); value != "" {
		parts = append(parts, titleCase(value))
	}
	if value := strings.TrimSpace(className); value != "" {
		parts = append(parts, titleCase(value))
	}
	if len(parts) == 0 {
		return "Adventurer"
	}
	return strings.Join(parts, " ")
}

func sheetActionHint(state SheetSnapshot) string {
	if state.AC > 0 {
		return "AC " + strconv.Itoa(int(state.AC))
	}
	return "—"
}

func sheetCombatActions(locale string) string {
	return localizedSheet(locale, "Combat Actions", "Acciones de combate")
}
func sheetOther(locale string) string { return localizedSheet(locale, "Other", "Otros") }

func localizedSheet(locale, english, spanish string) string {
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(locale)), "es") {
		return spanish
	}
	return english
}

func sheetPageStyle() map[string]string {
	background := "none"
	if url := ArtURL(phoneBackgroundAsset); url != "" {
		background = "linear-gradient(180deg, rgba(10,13,19,.32), rgba(10,13,19,.9)), url(\"" + url + "\")"
	}
	return map[string]string{"display": "grid", "align-content": "start", "gap": "12px", "min-height": "0", "box-sizing": "border-box", "padding": "2px 0 18px", "overflow": "auto", "background-color": "#10131b", "background-image": background, "background-size": "cover", "background-position": "center", "color": "#efe6d2"}
}

func sheetInitials(name string) string {
	for _, value := range strings.TrimSpace(name) {
		return string(value)
	}
	return "?"
}

func titleCase(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	return strings.ToUpper(value[:1]) + strings.ToLower(value[1:])
}
