//go:build js && wasm

package phone

import (
	"reflect"
	"strings"
	"testing"

	df "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
	"github.com/monstercameron/GoWebComponents/v6/router"
	"github.com/monstercameron/GoWebComponents/v6/testkit/render"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

func TestRenderPhoneScreen_AllFactories(t *testing.T) {
	props := phoneViewProps{
		creation: NewCreationModel(nil, "seat", 1), sheet: NewSheetModel(),
		moves: NewMovesModel(nil, "seat"), typed: NewTypedInputModel(nil, "seat"),
		dice: NewDiceModel(nil, "seat"), combat: NewCombatModel(nil, "seat"),
		ptt: NewPTTModel(nil, "seat", 1), end: NewEndModel(),
	}
	tests := []struct {
		kind    ScreenKind
		classes []string
	}{
		{ScreenCreate, []string{"df-phone-create"}},
		{ScreenSheet, []string{"df-phone-sheet"}},
		{ScreenMoves, []string{"df-phone-moves"}},
		{ScreenConversation, []string{"df-phone-conversation", "df-phone-ptt"}},
		{ScreenDice, []string{"df-phone-dice"}},
		{ScreenCombat, []string{"df-phone-combat"}},
		{ScreenEnd, []string{"df-phone-end"}},
	}
	for _, test := range tests {
		t.Run(string(test.kind), func(t *testing.T) {
			render.New(t)
			markup := renderPhoneMarkup(t, test.kind, props)
			for _, class := range test.classes {
				if !strings.Contains(markup, class) {
					t.Fatalf("screen %q missing %q: %s", test.kind, class, markup)
				}
			}
		})
	}
}

func TestRenderPhoneScreen_SelectsInteractivePhases(t *testing.T) {
	if got := SelectScreen(SeatView{Phase: "conversation"}); got != ScreenConversation {
		t.Fatalf("conversation screen = %q", got)
	}
	if got := SelectScreen(SeatView{Phase: "combat"}); got != ScreenCombat {
		t.Fatalf("combat screen = %q", got)
	}
}

func TestRenderPhoneScreen_RefreshesMovesSnapshot(t *testing.T) {
	render.New(t)
	model := NewMovesModel(nil, "seat")
	model.state.Moves = []MoveSnapshot{{ID: "talk", Label: "Talk", Enabled: true}}
	props := phoneViewProps{moves: model}
	first := renderPhoneMarkup(t, ScreenMoves, props)
	model.ApplyScreenState(&df.ScreenState{View: &df.ScreenState_Phone{Phone: &df.PhoneView{Moves: []*df.Move{{MoveId: "leave", Label: "Leave", Enabled: true}}}}})
	second := renderPhoneMarkup(t, ScreenMoves, props)
	if !strings.Contains(first, ">Talk</button>") || strings.Contains(second, ">Talk</button>") || !strings.Contains(second, ">Leave</button>") {
		t.Fatalf("first=%s second=%s", first, second)
	}
}

func TestRenderPhoneScreen_CombatMoveCountChanges(t *testing.T) {
	render.New(t)
	model := NewCombatModel(nil, "seat")
	props := phoneViewProps{combat: model}
	model.state.Moves = []*df.Move{{MoveId: "attack", Label: "Strike", Enabled: true}}
	first := renderPhoneMarkup(t, ScreenCombat, props)
	model.state.Moves = []*df.Move{{MoveId: "move", Label: "Advance", Enabled: true}, {MoveId: "end", Label: "Finish", Enabled: true}}
	second := renderPhoneMarkup(t, ScreenCombat, props)
	model.state.Moves = nil
	third := renderPhoneMarkup(t, ScreenCombat, props)
	if !strings.Contains(first, ">Strike</button>") || strings.Contains(second, ">Strike</button>") || !strings.Contains(second, ">Advance</button>") || !strings.Contains(second, ">Finish</button>") || strings.Contains(third, ">Advance</button>") {
		t.Fatalf("first=%s second=%s third=%s", first, second, third)
	}
}

func TestRenderPhoneMount_UnavailableClient(t *testing.T) {
	render.New(t)
	mount := reflect.ValueOf(Mount)
	client := reflect.Zero(reflect.TypeOf((*PhoneClient)(nil)).Elem())
	arguments := []reflect.Value{client, reflect.ValueOf("seat")}
	if mount.Type().NumIn() == 3 {
		arguments = append(arguments, reflect.ValueOf("en"))
	}
	component := mount.Call(arguments)[0].Interface().(router.Component)
	markup, err := ui.RenderToString(component(router.Attrs{}))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(markup, `role="alert"`) || !strings.Contains(markup, "df-phone") {
		t.Fatalf("unavailable-client markup = %s", markup)
	}
}

// renderPhoneMarkup covers both sides of the concurrent locale API migration.
// Reflection is confined to this test adapter so the committed pre-locale
// screen and the in-progress localized screen receive identical render checks.
func renderPhoneMarkup(t *testing.T, kind ScreenKind, props phoneViewProps) string {
	t.Helper()
	render := reflect.ValueOf(renderPhoneScreen)
	arguments := []reflect.Value{reflect.ValueOf(kind), reflect.ValueOf(props)}
	switch render.Type().NumIn() {
	case 2:
	case 3:
		arguments = append(arguments, reflect.ValueOf("en"))
	default:
		t.Fatal("unexpected phone render contract")
	}
	node := render.Call(arguments)[0].Interface().(ui.Node)
	markup, err := ui.RenderToString(node)
	if err != nil {
		t.Fatal(err)
	}
	return markup
}
