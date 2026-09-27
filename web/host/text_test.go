package host

import "testing"

func TestHostText_ScreensRenderThroughCatalog(t *testing.T) {
	cases := []struct {
		name   string
		gotEn  string
		gotEs  string
		wantEn string
		wantEs string
	}{
		{"title", HostTitle("en"), HostTitle("es"), "DungeonFlux Host", "DungeonFlux (anfitrión)"},
		{"run", RunSection("en"), RunSection("es"), "Run status", "Estado de la partida"},
		{"assets", AssetsSection("en"), AssetsSection("es"), "Asset slots", "Recursos"},
		{"log", LogSection("en"), LogSection("es"), "Log tail", "Registro"},
		{"no snapshot", NoSnapshot("en"), NoSnapshot("es"), "No snapshot yet", "Aún sin estado"},
		{"no assets", NoAssets("en"), NoAssets("es"), "No assets", "Sin recursos"},
		{"no logs", NoLogs("en"), NoLogs("es"), "No logs", "Sin registros"},
		{"accepted", AcceptedLine("en"), AcceptedLine("es"), "Accepted", "Aceptado"},
		{"connecting", ConnectingLine("en"), ConnectingLine("es"), "Connecting…", "Conectando…"},
		{"ready", ReadyLine("en"), ReadyLine("es"), "Ready", "Listo"},
		{"room lang", RoomLocaleLabel("en"), RoomLocaleLabel("es"), "Room language", "Idioma de la sala"},
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

func TestHostText_ActionLabelsAndLines(t *testing.T) {
	for _, action := range hostActions {
		if got := HostActionLabel("es", action); got == "" || got == action.Label && action.Label == "" {
			t.Errorf("empty es label for %v", action.Command)
		}
	}
	if got := HostActionLabel("es", hostActions[0]); got != "Empezar" {
		t.Fatalf("start es = %q", got)
	}
	if got := ModeLine("es", "live"); got != "Modo: live" {
		t.Fatalf("mode = %q", got)
	}
	if got := NextD20Line("es", 17); got != "Próximo d20: 17" {
		t.Fatalf("next d20 = %q", got)
	}
	if got := SendingLine("es", "Empezar"); got != "Enviando Empezar…" {
		t.Fatalf("sending = %q", got)
	}
	if got := RejectedLine("es", "lleno"); got != "Rechazado: lleno" {
		t.Fatalf("rejected = %q", got)
	}
}

func TestHostText_TesterLinkLabels(t *testing.T) {
	if TesterLinksTitle("en") != "Tester links" || CopyLinkLabel("en") != "Copy" {
		t.Fatal("english tester labels changed")
	}
	if TesterLinksTitle("es") != "Enlaces de prueba" || CopyLinkLabel("es") != "Copiar" {
		t.Fatal("spanish tester labels changed")
	}
}

func TestHostText_NewCopyRoutesThroughCatalog(t *testing.T) {
	if HostKicker("en") != "CONTROL ROOM" || HostKicker("es") != "SALA DE CONTROL" {
		t.Fatalf("kicker = %q / %q", HostKicker("en"), HostKicker("es"))
	}
	if ControlsHeading("en") != "Run controls" || StageToolsHeading("en") != "Stage tools" {
		t.Fatalf("controls/stage headings changed")
	}
	if LinkLabelHost("en") != "Host controls" || LinkLabelDM("es") != "Pantalla del DM" {
		t.Fatalf("link labels changed")
	}
	if TokenRevealLabel("en") != "Hold to reveal" {
		t.Fatalf("token reveal label = %q", TokenRevealLabel("en"))
	}
	if ResetConfirmTitle("es") != "¿Reiniciar la partida?" {
		t.Fatalf("reset confirm title es = %q", ResetConfirmTitle("es"))
	}
	if got := PhaseLine("en", "Combat"); got != "Phase: Combat" {
		t.Fatalf("PhaseLine = %q", got)
	}
	if got := StatusSeatsValue("en", 1, 2); got != "1/2" {
		t.Fatalf("StatusSeatsValue = %q", got)
	}
	if got := StatusFailures("en", 3); got != "3 failing" {
		t.Fatalf("StatusFailures = %q", got)
	}
	if StatusAllClear("es") != "Todo en orden" {
		t.Fatalf("StatusAllClear es = %q", StatusAllClear("es"))
	}
}

func TestRoomLocaleSelector_SettleAndOptions(t *testing.T) {
	selector := NewRoomLocaleSelector("es")
	if selector.Selected != "es" {
		t.Fatalf("selected = %q", selector.Selected)
	}
	if len(selector.Options) != 2 || selector.Options[0] != "en" {
		t.Fatalf("options = %v", selector.Options)
	}
	if got := selector.Select("fr"); got != "es" {
		t.Fatalf("unsupported select = %q", got)
	}
	if got := selector.Label("es"); got != "Idioma de la sala" {
		t.Fatalf("label = %q", got)
	}
	if got := OptionLabel("es"); got != "Español" {
		t.Fatalf("option = %q", got)
	}
}
