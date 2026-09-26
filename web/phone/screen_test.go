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
		{"resolution", ScreenSheet},
		{"hook_event", ScreenSheet},
		{"combat", ScreenCombat},
		{"cliffhanger", ScreenSheet},
		{"end", ScreenSheet},
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
	if got := SelectScreen(view); got != ScreenSheet {
		t.Fatalf("end screen = %q, want %q", got, ScreenSheet)
	}
}

func TestFrameModel_TracksScreenAndConnection(t *testing.T) {
	model := NewFrameModel("", "")
	if model.DisplayName != "Player" || model.Locale != "en" || model.Connection != ConnectionConnecting {
		t.Fatalf("initial frame = %+v", model)
	}
	first := model.ApplyView(SeatView{Phase: "creation"})
	if !first.Changed || first.To != ScreenCreate || model.Connection != ConnectionOnline {
		t.Fatalf("first transition = %+v, model = %+v", first, model)
	}
	second := model.ApplyView(SeatView{Phase: "creation"})
	if second.Changed || second.From != ScreenCreate || second.To != ScreenCreate {
		t.Fatalf("repeat transition = %+v", second)
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
