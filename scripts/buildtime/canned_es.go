package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// SpanishCannedLines returns the Spanish renders of every English canned
// line. Asset IDs carry the "_es" suffix so the server's per-locale canned
// lookup finds them, falling back to the base asset when absent. Voices
// carry the same suffix; the TTS project provisions per-locale voices under
// that convention.
func SpanishCannedLines() []CannedLine {
	voices := map[string]string{}
	texts := map[string]string{}
	for _, line := range CannedLines() {
		voices[line.ID] = line.Voice + "-es"
	}
	for id, text := range spanishCannedText {
		texts[id] = text
	}
	lines := make([]CannedLine, 0, len(CannedLines()))
	for _, line := range CannedLines() {
		text, ok := texts[line.ID]
		if !ok || strings.TrimSpace(text) == "" {
			continue
		}
		lines = append(lines, CannedLine{ID: line.ID + "_es", Voice: voices[line.ID], Text: text})
	}
	return lines
}

var spanishCannedText = map[string]string{
	"canned_opening":              "La lluvia golpea la Linterna Ahogada. El farolero desapareció anoche y el río sigue subiendo. Dos viajeros se sacuden el agua en la barra, donde Madre Vell observa con su único ojo sano.",
	"canned_npc_reply":            "Mucha gente bebe aquí, cariño. No llevo un registro de caras y no respondo preguntas gratis.",
	"canned_npc_reveal":           "Está bien. Lo arrastraron hacia el viejo campanario. Y a medianoche esa campana sonó, aunque nadie la ha subido en años.",
	"canned_npc_refuse":           "Buen intento. He enterrado a mejores habladores que tú. Bebe o sigue tu camino; no tengo más que decir.",
	"canned_stranger_found":       "Una carta, para uno de vosotros. El sello está empapado del río y no la leí. Lo que fuera que había en esa agua, me siguió desde el río.",
	"canned_stranger_relocated":   "Una carta, para uno de vosotros. Dicen que al farolero lo arrastraron al viejo campanario. Algo mojado y muerto lo custodiaba, y me siguió desde el río.",
	"canned_cliffhanger_vell":     "Medianoche. La campana de la torre de la que advirtió Madre Vell tañe y cada farol de la taberna se apaga. En la oscuridad vuelve a sonar, lenta y paciente. Quien tira de esa cuerda ya sabe vuestros nombres.",
	"canned_cliffhanger_stranger": "Medianoche. La carta del mensajero se abre al tañer la campana y cada farol se apaga. Dentro, en tinta mojada, están vuestros nombres. La campana vuelve a sonar.",
	"canned_combat_slain_seat1":   "El acero encuentra el corazón de lodo del ahogado y se desploma en un charco de agua oscura.",
	"canned_combat_slain_seat2":   "Un último golpe y el ahogado se derrumba. El río recupera lo suyo.",
	"canned_combat_fled":          "A lo lejos, la campana tañe una vez. El ahogado se gira a medio golpe y se arrastra hacia la lluvia, hacia la torre.",
}

// SpanishCannedJob returns a job that renders the Spanish canned lines.
// In dry-run mode no network call is made: the Spanish texts are written as
// plan files and the client may be nil. Live mode renders through
// RenderCannedLine exactly like the English job.
func SpanishCannedJob(client *http.Client, endpoint, outputDir string, take int, dryRun bool) Job {
	return Job{Name: "canned-lines-es", Run: func(ctx context.Context, writer *ManifestWriter) error {
		lines := SpanishCannedLines()
		if len(lines) != len(CannedLines()) {
			return errors.New("buildtime: spanish canned lines do not cover every english line")
		}
		if dryRun {
			return writeSpanishCannedPlan(outputDir, lines)
		}
		for _, line := range lines {
			if err := RenderCannedLine(ctx, client, endpoint, filepath.Clean(outputDir), writer, line, take); err != nil {
				return err
			}
		}
		return nil
	}}
}

func writeSpanishCannedPlan(outputDir string, lines []CannedLine) error {
	if err := os.MkdirAll(filepath.Clean(outputDir), 0o755); err != nil {
		return err
	}
	for _, line := range lines {
		if strings.TrimSpace(line.Text) == "" {
			return errors.New("buildtime: spanish canned line " + line.ID + " is empty")
		}
		path := filepath.Join(filepath.Clean(outputDir), line.ID+".txt")
		if err := os.WriteFile(path, []byte(line.Text+"\n"), 0o600); err != nil {
			return err
		}
	}
	return nil
}
