package dm

import dungeonfluxv1 "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"

// PreviewFixture is a named DM screen snapshot used by the offline preview mode.
// State is a complete wire snapshot so the preview follows the same rendering
// path as a live Watch update.
type PreviewFixture struct {
	Name  string
	State *dungeonfluxv1.ScreenState
}

// Previews returns the deterministic DM snapshots used by preview mode.
// Each call returns fresh protobuf messages so callers may safely customize a
// fixture without changing another caller's registry.
func Previews() map[string]PreviewFixture {
	return map[string]PreviewFixture{
		"lobby":         preview("lobby", lobbyPreview()),
		"creation":      preview("creation", scenePreview("creation")),
		"opening":       preview("opening", openingPreview()),
		"exploration":   preview("exploration", scenePreview("exploration")),
		"conversation":  preview("conversation", conversationPreview()),
		"check-rolling": preview("check-rolling", checkPreview(dungeonfluxv1.DiceState_DICE_STATE_ROLLING, "rolling")),
		"check-result":  preview("check-result", checkPreview(dungeonfluxv1.DiceState_DICE_STATE_RESOLVED, "success")),
		"resolution":    preview("resolution", checkPreview(dungeonfluxv1.DiceState_DICE_STATE_RESOLVED, "success")),
		"hook":          preview("hook", hookPreview()),
		"combat-flat":   preview("combat-flat", combatPreview()),
		"cliffhanger":   preview("cliffhanger", cliffhangerPreview()),
		"end":           preview("end", endPreview()),
	}
}

// Preview returns one named fixture, or false when name is not registered.
func Preview(name string) (PreviewFixture, bool) {
	fixture, ok := Previews()[name]
	return fixture, ok
}

func preview(name string, state *dungeonfluxv1.ScreenState) PreviewFixture {
	return PreviewFixture{Name: name, State: state}
}

func state(phase string, view *dungeonfluxv1.DMView) *dungeonfluxv1.ScreenState {
	return &dungeonfluxv1.ScreenState{Version: 1, Phase: phase, Locale: "en", View: &dungeonfluxv1.ScreenState_Dm{Dm: view}}
}

func baseView() *dungeonfluxv1.DMView {
	return &dungeonfluxv1.DMView{
		BackgroundUrl: "establishing_tavern",
		Locale:        "en",
		Layers: []*dungeonfluxv1.Layer{
			{Id: "mother-vell", Url: "mother_vell", X: 68, Y: 49, Scale: 1},
		},
		BuildCards: []*dungeonfluxv1.BuildCard{
			{PlayerNumber: 1, Name: "Mira", ClassName: "Rogue", PortraitUrl: "ui/class_rogue"},
			{PlayerNumber: 2, Name: "Rook", ClassName: "Paladin", PortraitUrl: "ui/class_paladin"},
		},
	}
}

func lobbyPreview() *dungeonfluxv1.ScreenState {
	view := baseView()
	view.Callout = "ROOM: FLUX"
	return state("lobby", view)
}

func scenePreview(phase string) *dungeonfluxv1.ScreenState {
	return state(phase, baseView())
}

func openingPreview() *dungeonfluxv1.ScreenState {
	view := baseView()
	view.Clip = &dungeonfluxv1.Clip{Url: "establishing_tavern", Playing: true, Then: "STILL"}
	return state("opening", view)
}

func conversationPreview() *dungeonfluxv1.ScreenState {
	view := baseView()
	view.Narration = &dungeonfluxv1.Narration{Speaker: "Mother Vell", TextSoFar: "The river remembers what the drowned forget."}
	view.Subtitle = &dungeonfluxv1.Subtitle{Text: "The river remembers what the drowned forget."}
	view.Callout = "Mother Vell is speaking"
	return state("conversation", view)
}

func checkPreview(diceState dungeonfluxv1.DiceState, outcome string) *dungeonfluxv1.ScreenState {
	view := baseView()
	view.Dice = &dungeonfluxv1.Dice{State: diceState, D20: 17, Modifier: 4, Dc: 10, Kind: dungeonfluxv1.DiceKind_DICE_KIND_CHECK, Outcome: outcome, VsLabel: "Persuade"}
	return state("check", view)
}

func hookPreview() *dungeonfluxv1.ScreenState {
	view := baseView()
	view.Clip = &dungeonfluxv1.Clip{Url: "stranger", Playing: true, Then: "STILL"}
	view.Callout = "DM steering: personal hook → Rook"
	return state("hook_event", view)
}

func combatPreview() *dungeonfluxv1.ScreenState {
	view := &dungeonfluxv1.DMView{
		Locale: "en",
		Battlefield: &dungeonfluxv1.Battlefield{
			Mode: "FLAT", Visible: true,
			Grid: &dungeonfluxv1.Grid{Cols: 4, Rows: 3, Walkable: []*dungeonfluxv1.Cell{{C: 0, R: 1}, {C: 1, R: 1}, {C: 2, R: 1}, {C: 3, R: 1}, {C: 1, R: 2}, {C: 2, R: 2}}},
			Flat: &dungeonfluxv1.FlatBattlefield{ImageUrl: "battlefield_tavern_flat", FloorQuadPx: []float32{120, 180, 1800, 120, 1740, 940, 160, 900}},
		},
		Tokens: []*dungeonfluxv1.Token{
			{TokenId: "mira", Name: "Mira", PortraitUrl: "ui/class_rogue", Cell: &dungeonfluxv1.Cell{C: 1, R: 1}, Hp: 9, HpMax: 10, Active: true},
			{TokenId: "rook", Name: "Rook", PortraitUrl: "ui/class_paladin", Cell: &dungeonfluxv1.Cell{C: 2, R: 2}, Hp: 12, HpMax: 12},
			{TokenId: "thrall", Name: "Drowned Thrall", PortraitUrl: "stranger", Cell: &dungeonfluxv1.Cell{C: 3, R: 1}, Hp: 18, HpMax: 24, Statuses: []string{"bloodied"}},
		},
		Dice:         &dungeonfluxv1.Dice{State: dungeonfluxv1.DiceState_DICE_STATE_OFFERED, Kind: dungeonfluxv1.DiceKind_DICE_KIND_ATTACK},
		TurnTimer:    &dungeonfluxv1.Timer{Seat: "1", RemainingMs: 12000, TotalMs: 15000},
		CombatBanner: "Mira's turn",
	}
	return state("combat", view)
}

func cliffhangerPreview() *dungeonfluxv1.ScreenState {
	view := baseView()
	view.Clip = &dungeonfluxv1.Clip{Url: "cliff_generic_tower", Playing: true, Then: "STILL"}
	return state("cliffhanger", view)
}

func endPreview() *dungeonfluxv1.ScreenState {
	return state("end", &dungeonfluxv1.DMView{Locale: "en"})
}
