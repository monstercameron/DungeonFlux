package phone

import (
	df "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
	"testing"
)

func TestSelectScreen_AllDemoPhases(t *testing.T) {
	tests := []struct {
		phase string
		want  ScreenKind
	}{
		{"creation", ScreenCreate},
		{"opening", ScreenSheet},
		{"lobby", ScreenMoves},
		{"exploration", ScreenMoves},
		{"conversation", ScreenConversation},
		{"check", ScreenDice},
		{"resolution", ScreenDice},
		{"hook_event", ScreenSheet},
		{"combat", ScreenCombat},
		{"cliffhanger", ScreenSheet},
		{"end", ScreenEnd},
		{"unknown", ScreenSheet},
	}
	for _, test := range tests {
		t.Run(test.phase, func(t *testing.T) {
			if got := SelectScreen(SeatView{Phase: test.phase}); got != test.want {
				t.Fatalf("SelectScreen(%q) = %q, want %q", test.phase, got, test.want)
			}
		})
	}
}

func TestSeatViewFromState_CarriesSnapshotVersion(t *testing.T) {
	state := &df.ScreenState{Version: 18, Phase: "creation", View: &df.ScreenState_Phone{Phone: &df.PhoneView{
		StatusText: "Your hero is ready to lock in",
	}}}
	view := seatViewFromState(state)
	if view.Version != 18 || view.Phase != "creation" || view.Phone.GetStatusText() != "Your hero is ready to lock in" {
		t.Fatalf("seat view = %+v", view)
	}
	if empty := seatViewFromState(nil); empty.Version != 0 || empty.Phone != nil {
		t.Fatalf("nil state view = %+v", empty)
	}
}

func TestSeatViewFromState_ProjectsNarration(t *testing.T) {
	state := &df.ScreenState{View: &df.ScreenState_Phone{Phone: &df.PhoneView{
		Narration: &df.Narration{Speaker: "Mother Vell", TextSoFar: "Speak or drink.", Done: true},
	}}}
	view := seatViewFromState(state)
	if view.Narration.Speaker != "Mother Vell" || view.Narration.Text != "Speak or drink." || !view.Narration.Done {
		t.Fatalf("narration = %#v", view.Narration)
	}
}

func TestSelectScreen_UsesSeatViewState(t *testing.T) {
	if got := SelectScreen(SeatView{Phone: &df.PhoneView{Moves: []*df.Move{{MoveId: "talk_vell"}}}}); got != ScreenMoves {
		t.Fatalf("moves screen = %q", got)
	}
	if got := SelectScreen(SeatView{Phone: &df.PhoneView{Combat: &df.CombatView{MyTurn: true}}}); got != ScreenCombat {
		t.Fatalf("combat screen = %q", got)
	}
	if got := SelectScreen(SeatView{Phone: &df.PhoneView{Character: &df.Character{}}}); got != ScreenSheet {
		t.Fatalf("sheet screen = %q", got)
	}
}

func TestSelectScreen_EndIgnoresStaleSeatControls(t *testing.T) {
	view := SeatView{Phase: "end", Phone: &df.PhoneView{
		Moves:  []*df.Move{{MoveId: "attack", Enabled: true}},
		Combat: &df.CombatView{MyTurn: true},
	}}
	if got := SelectScreen(view); got != ScreenEnd {
		t.Fatalf("end screen = %q, want %q", got, ScreenEnd)
	}
}

func TestEndModel_PreservesFinalCharacterState(t *testing.T) {
	model := NewEndModel()
	state := &df.ScreenState{View: &df.ScreenState_Phone{Phone: &df.PhoneView{
		Locale: "en", StatusText: "Victory", Character: &df.Character{Name: "Astra", ClassName: "Rogue", PortraitUrl: "hero.png"},
		Combat: &df.CombatView{Hp: 9, HpMax: 12, Statuses: []string{"Inspired"}},
	}}}
	got := model.ApplyScreenState(state)
	if got.Name != "Astra" || got.Class != "Rogue" || got.FinalHP != 9 || got.FinalHPMax != 12 || got.Outcome != "Victory" || len(got.FinalStates) != 1 {
		t.Fatalf("end snapshot = %+v", got)
	}
	got.FinalStates[0] = "changed"
	if model.Snapshot().FinalStates[0] != "Inspired" {
		t.Fatal("snapshot leaked final states")
	}
}

func TestFrameModel_TracksScreenAndConnection(t *testing.T) {
	model := NewFrameModel("", "")
	if model.DisplayName != "Player" || model.Locale != "en" || model.Connection != ConnectionConnecting || model.Location != "The Drowned Lantern" || model.ActiveTab != PhoneTabPlay {
		t.Fatalf("initial frame = %+v", model)
	}
	first := model.ApplyView(SeatView{Phase: "creation"})
	if !first.Changed || first.To != ScreenCreate || model.Connection != ConnectionOnline || model.Mode != PhoneModePlay {
		t.Fatalf("first transition = %+v, model = %+v", first, model)
	}
	second := model.ApplyView(SeatView{Phase: "creation"})
	if second.Changed || second.From != ScreenCreate || second.To != ScreenCreate {
		t.Fatalf("repeat transition = %+v", second)
	}
	model.ApplyView(SeatView{Phase: "combat"})
	if model.Mode != PhoneModeCombat || model.ActiveTab != PhoneTabPlay || model.Location != T("en", "combat.location", nil) {
		t.Fatalf("combat frame mode = %q, tab = %q", model.Mode, model.ActiveTab)
	}
	model.ApplyView(SeatView{Phase: "cliffhanger"})
	if model.Location != "The Drowned Lantern" {
		t.Fatalf("post-combat location = %q", model.Location)
	}
	model.SetConnection(ConnectionOffline)
	if model.Connection != ConnectionOffline {
		t.Fatal("offline state was not recorded")
	}
}

func TestConnectionLabel_LocalizesStableStates(t *testing.T) {
	cases := []struct {
		locale string
		state  ConnectionState
		want   string
	}{
		{"en", ConnectionOnline, "Connected"},
		{"en", ConnectionConnecting, "Connecting…"},
		{"en", ConnectionOffline, "Offline"},
		{"es", ConnectionOnline, "Conectado"},
		{"es", ConnectionOffline, "Sin conexión"},
	}
	for _, tc := range cases {
		t.Run(tc.locale+"-"+string(tc.state), func(t *testing.T) {
			if got := ConnectionLabel(tc.locale, tc.state); got != tc.want {
				t.Fatalf("label = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestDefaultPhoneTheme_ProtectsTouchAndMotionTokens(t *testing.T) {
	theme := DefaultPhoneTheme()
	if theme.Ink != "#0f1117" || theme.Gold != "#d9a441" || theme.TouchTarget != "48px" {
		t.Fatalf("theme palette = %+v", theme)
	}
	if theme.BorderRadius == "" || theme.Transition == "" || theme.Parchment == "" {
		t.Fatalf("theme omitted required token = %+v", theme)
	}
}
