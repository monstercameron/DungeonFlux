package phone

import (
	"testing"

	df "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
)

func TestPhoneText_ScreensRenderThroughCatalog(t *testing.T) {
	cases := []struct {
		name   string
		gotEn  string
		gotEs  string
		wantEn string
		wantEs string
	}{
		{"moves title", MovesTitle("en"), MovesTitle("es"), "Your moves", "Tus acciones"},
		{"create title", CreateTitle("en"), CreateTitle("es"), "Create your hero", "Crea tu héroe"},
		{"create hint", CreateHint("en"), CreateHint("es"), "Choose a species and gender to roll your hero.", "Elige especie y género para crear tu héroe."},
		{"species", SpeciesLabel("en"), SpeciesLabel("es"), "Human", "Humano"},
		{"gender", GenderLabel("en"), GenderLabel("es"), "Nonbinary", "No binario"},
		{"roll hero", RollHeroLabel("en"), RollHeroLabel("es"), "Roll my hero", "Crear mi héroe"},
		{"dice title", DiceTitle("en"), DiceTitle("es"), "Make your case", "Expón tu caso"},
		{"dice button", DiceButton("en"), DiceButton("es"), "Roll d20", "Tirar d20"},
		{"typed label", TypedLabel("en"), TypedLabel("es"), "Type your message", "Escribe tu mensaje"},
		{"typed hint", TypedHint("en"), TypedHint("es"), "Type what you say…", "Escribe lo que dices…"},
		{"typed send", TypedSend("en"), TypedSend("es"), "Send", "Enviar"},
		{"typed sent", TypedSent("en"), TypedSent("es"), "Message sent", "Mensaje enviado"},
		{"ptt start", PTTStart("en"), PTTStart("es"), "Start talking", "Empezar a hablar"},
		{"ptt stop", PTTStop("en"), PTTStop("es"), "Stop talking", "Dejar de hablar"},
		{"ptt ready", PTTReady("en"), PTTReady("es"), "Ready to talk", "Listo para hablar"},
		{"ptt norecord", PTTNoRecord("en"), PTTNoRecord("es"), "This browser cannot record audio", "Este navegador no puede grabar audio"},
		{"combat turn", CombatTurnTitle("en"), CombatTurnTitle("es"), "Your turn", "Tu turno"},
		{"error title", ErrorTitle("en"), ErrorTitle("es"), "DungeonFlux", "DungeonFlux"},
		{"client unavail", ClientUnavailable("en"), ClientUnavailable("es"), "Player client unavailable", "Cliente de jugador no disponible"},
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

func TestPhoneText_DiceResultPhases(t *testing.T) {
	if got := DiceResult("es", DiceRolling, 0, ""); got != "Tirando…" {
		t.Fatalf("rolling = %q", got)
	}
	if got := DiceResult("es", DiceResolved, 17, "Éxito"); got != "Resultado: 17 Éxito" {
		t.Fatalf("resolved = %q", got)
	}
	if got := DiceCheckLabel("es", 4, 10); got != "Persuasión +4 contra CD 10" {
		t.Fatalf("check label = %q", got)
	}
}

func TestPhoneText_SheetLines(t *testing.T) {
	if got := SheetName("es", "", ""); got != "Tu personaje" {
		t.Fatalf("empty name = %q", got)
	}
	if got := SheetHP("es", 8, 10); got != "PV 8/10" {
		t.Fatalf("hp = %q", got)
	}
	if got := SheetHP("en", 0, 0); got != "HP —" {
		t.Fatalf("hp none = %q", got)
	}
	if got := SheetConditions("es", nil); got != "Sin condiciones" {
		t.Fatalf("no conditions = %q", got)
	}
	if got := SheetConditions("es", []string{"prone"}); got != "Condiciones: prone" {
		t.Fatalf("conditions = %q", got)
	}
}

func TestPhoneText_RenderMsg(t *testing.T) {
	msg := &df.Text{Key: "move.attack", Fallback: "Attack the drowned thrall"}
	if got := RenderMsg("es", msg, "fallback"); got != "Atacar al ahogado" {
		t.Fatalf("msg = %q", got)
	}
	if got := RenderMsg("es", nil, "fallback"); got != "fallback" {
		t.Fatalf("nil msg = %q", got)
	}
	bare := &df.Text{Fallback: "server text"}
	if got := RenderMsg("es", bare, "fallback"); got != "server text" {
		t.Fatalf("bare msg = %q", got)
	}
}
