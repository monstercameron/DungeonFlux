//go:build js && wasm

package phone

import (
	"strings"
	"testing"

	df "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
	"github.com/monstercameron/GoWebComponents/v6/testkit/render"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

func renderTestModels() phoneViewProps {
	return phoneViewProps{
		creation: NewCreationModel(nil, "seat", 1), sheet: NewSheetModel(),
		moves: NewMovesModel(nil, "seat"), typed: NewTypedInputModel(nil, "seat"),
		dice: NewDiceModel(nil, "seat"), combat: NewCombatModel(nil, "seat"),
		ptt: NewPTTModel(nil, "seat", 1), end: NewEndModel(), journal: NewJournalLog(),
	}
}

func TestPhoneRender_NavigationWithoutServerSnapshot(t *testing.T) {
	fixture := render.New(t)
	models := renderTestModels()
	root := func() ui.Node {
		tab := ui.UseState(PhoneTabPlay)
		taps := make(map[PhoneTabID]ui.Handler)
		for _, id := range []PhoneTabID{PhoneTabCharacter, PhoneTabJournal, PhoneTabPlay, PhoneTabMap, PhoneTabMenu} {
			taps[id] = ui.UseEvent(func() { tab.Set(id) })
		}
		return renderPhoneScreen(ScreenSheet, models, "en", SeatView{Version: 1}, tab.Get(), taps, taps[PhoneTabPlay])
	}
	fixture.Render(ui.CreateElement(root))
	for _, tc := range []struct{ button, want string }{
		{"Journal", "Journal"}, {"Map", "No map is available yet"},
		{"Menu", "How to play"}, {"Character", "Ability scores"},
		{"Play", "Ability scores"}, {"Menu", "How to play"},
	} {
		fixture.ByRole("button", tc.button).Click()
		if !strings.Contains(fixture.Text(), tc.want) {
			t.Fatalf("%s did not render %q: %s", tc.button, tc.want, fixture.Text())
		}
	}
}

func TestPhoneRender_SnapshotPreservesSheetTab(t *testing.T) {
	fixture := render.New(t)
	models := renderTestModels()
	var update func(SeatView)
	root := func() ui.Node {
		view := ui.UseState(SeatView{Version: 1, Phase: "opening"})
		update = view.Set
		return renderPhoneScreen(SelectScreen(view.Get()), models, "en", view.Get(), PhoneTabPlay, nil, ui.Handler{})
	}
	fixture.Render(ui.CreateElement(root))
	fixture.ByRole("button", "Spells").Click()
	update(SeatView{Version: 2, Phase: "opening"})
	if !strings.Contains(fixture.Text(), "No spells in this demo") {
		t.Fatalf("snapshot discarded local tab: %s", fixture.Text())
	}
	update(SeatView{Version: 3, Phase: "end"})
	if !strings.Contains(fixture.Text(), "THE END") || strings.Contains(fixture.Text(), "Ability scores") {
		t.Fatalf("end did not replace sheet: %s", fixture.Text())
	}
}

func TestPhoneRender_ConversationUpdatePreservesDraft(t *testing.T) {
	fixture := render.New(t)
	models := renderTestModels()
	models.moves.state.StatusText = "First line"
	models.moves.state.Moves = []MoveSnapshot{{ID: "persuade", Label: "Wait for Vell", Enabled: false}}
	var update func(SeatView)
	root := func() ui.Node {
		view := ui.UseState(SeatView{Version: 1, Phase: "conversation"})
		update = view.Set
		return renderPhoneScreen(ScreenConversation, models, "en", view.Get(), PhoneTabPlay, nil, ui.Handler{})
	}
	fixture.Render(ui.CreateElement(root))
	fixture.InputByID("talk-message", "Tell me about the bell")
	models.moves.state.Moves = []MoveSnapshot{{ID: "persuade", Label: "Persuade now", Enabled: true}}
	update(SeatView{Version: 2, Phase: "conversation"})
	if !strings.Contains(fixture.Text(), "Persuade now") || strings.Contains(fixture.Text(), "Wait for Vell") {
		t.Fatalf("moves stayed stale: %s", fixture.Text())
	}
	if got := models.typed.Snapshot().Text; got != "Tell me about the bell" {
		t.Fatalf("draft lost: %q", got)
	}
}

func TestPhoneRender_AllPhaseFactories(t *testing.T) {
	for _, name := range []string{"creation-pick", "sheet", "legal-moves", "ptt-idle", "dice-offered", "combat-my-turn", "end"} {
		t.Run(name, func(t *testing.T) {
			fixture := render.New(t)
			preview, _ := Preview(name)
			models := renderTestModels()
			state := &df.ScreenState{Phase: preview.View.Phase, View: &df.ScreenState_Phone{Phone: preview.View.Phone}}
			models.creation.ApplyScreenState(state)
			models.sheet.ApplyScreenState(state)
			models.moves.ApplyScreenState(state)
			models.dice.ApplyScreenState(state)
			models.combat.ApplyScreenState(state)
			models.end.ApplyScreenState(state)
			root := func() ui.Node {
				return renderPhoneScreen(SelectScreen(preview.View), models, "en", preview.View, PhoneTabPlay, nil, ui.Handler{})
			}
			fixture.Render(ui.CreateElement(root))
			if len(fixture.AllByTag("nav")) < 1 {
				t.Fatal("missing navigation")
			}
		})
	}
}

func TestPhoneRender_LatePortraitPreservesSheetTab(t *testing.T) {
	SetArtSource(nil)
	t.Cleanup(func() { SetArtSource(nil) })
	fixture := render.New(t)
	models := renderTestModels()
	preview, _ := Preview("sheet")
	models.sheet.ApplyScreenState(&df.ScreenState{Phase: preview.View.Phase, View: &df.ScreenState_Phone{Phone: preview.View.Phone}})
	var repaint func(int)
	root := func() ui.Node {
		version := ui.UseState(0)
		repaint = version.Set
		return renderPhoneScreen(ScreenSheet, models, "en", preview.View, PhoneTabPlay, nil, ui.Handler{})
	}
	fixture.Render(ui.CreateElement(root))
	if len(fixture.AllByTag("img")) != 0 || !strings.Contains(fixture.ByRole("img", "Astra Vale").Attr("class"), "portrait-fallback") {
		t.Fatal("missing art should show character initials")
	}
	fixture.ByRole("button", "Spells").Click()
	SetArtSource(missingGenderArt{})
	ArtChanged()
	repaint(1)
	if len(fixture.AllByTag("img")) != 1 {
		t.Fatal("loaded portrait did not replace initials")
	}
	if got := fixture.AllByTag("img")[0].Attr("src"); got != "blob:crest" {
		t.Fatalf("loaded portrait source = %q", got)
	}
	if !strings.Contains(fixture.Text(), "No spells in this demo") {
		t.Fatal("art arrival reset the character sheet tab")
	}
}
