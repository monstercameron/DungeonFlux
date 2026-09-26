package dm

import "testing"

func TestDMText_ScreensRenderThroughCatalog(t *testing.T) {
	cases := []struct {
		name   string
		gotEn  string
		gotEs  string
		wantEn string
		wantEs string
	}{
		{"lobby title", LobbyTitle("en"), LobbyTitle("es"), "Gather at the table", "Reuníos a la mesa"},
		{"lobby intro", LobbyIntro("en"), LobbyIntro("es"), "Players joined by scanning this code. No accounts, no app.", "Los jugadores se unen escaneando este código. Sin cuentas, sin app."},
		{"room code", RoomCodeLabel("en"), RoomCodeLabel("es"), "ROOM CODE", "CÓDIGO DE SALA"},
		{"seats", SeatsLabel("en"), SeatsLabel("es"), "Player seats", "Asientos de jugadores"},
		{"listen", ListenLabel("en"), ListenLabel("es"), "Listen", "Escuchar"},
		{"audio unlock", AudioUnlock("en"), AudioUnlock("es"), "Enable table audio", "Activar audio de la mesa"},
		{"error title", ErrorTitle("en"), ErrorTitle("es"), "DungeonFlux", "DungeonFlux"},
		{"end title", EndTitle("en"), EndTitle("es"), "The bell remembers.", "La campana recuerda."},
		{"end rules", EndRules("en"), EndRules("es"), "Rules and attribution", "Reglas y atribución"},
		{"crit", CritLabel("en"), CritLabel("es"), "CRITICAL", "CRÍTICO"},
		{"timer paused", TimerPaused("en"), TimerPaused("es"), "Turn timer paused", "Temporizador en pausa"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.gotEn != tc.wantEn {
				t.Errorf("en = %q, want %q", tc.gotEn, tc.wantEn)
			}
			if tc.gotEs != tc.wantEs {
				t.Errorf("es = %q, want %q", tc.gotEs, tc.wantEs)
			}
		})
	}
}

func TestDMText_SeatStatusAndNames(t *testing.T) {
	if got := SeatStatus("es", false, false); got != "Esperando para unirse" {
		t.Fatalf("waiting = %q", got)
	}
	if got := SeatStatus("es", true, true); got != "Listo" {
		t.Fatalf("ready = %q", got)
	}
	if got := SeatName("es", "", 2); got != "Jugador 2" {
		t.Fatalf("seat name = %q", got)
	}
	if got := SeatName("es", "Mara", 2); got != "Mara" {
		t.Fatalf("named seat = %q", got)
	}
	if got := DiceHeading("es", "attack"); got != "tirada de ataque" {
		t.Fatalf("dice heading = %q", got)
	}
	if got := VsDC("es", 10); got != "contra CD 10" {
		t.Fatalf("vs dc = %q", got)
	}
	if got := TimerLabel("es", 30000); got != "Temporizador: quedan 30000 milisegundos" {
		t.Fatalf("timer = %q", got)
	}
}

func TestEndCard_Localized(t *testing.T) {
	model := NewEndCardModel().Localized("es")
	if model.Title != "La campana recuerda." {
		t.Fatalf("title = %q", model.Title)
	}
	if model.Locale != "es" {
		t.Fatalf("locale = %q", model.Locale)
	}
	lobby := NewLobbyModel("AB12", "")
	lobby.SetLocale("es")
	if lobby.AudioState != "Esperando la apertura…" {
		t.Fatalf("audio state = %q", lobby.AudioState)
	}
}
