package in

import (
	"context"
	"testing"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/ports"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

func localeRecording(t *testing.T, utterance domain.UtteranceID) *Assembler {
	t.Helper()
	assembler := NewAssembler()
	session := Session{Seat: 2, UtteranceID: utterance, MIME: "audio/webm"}
	if err := assembler.Start(context.Background(), session); err != nil {
		t.Fatal(err)
	}
	if err := assembler.Chunk(context.Background(), Chunk{Session: session, Data: []byte("audio")}); err != nil {
		t.Fatal(err)
	}
	if _, err := assembler.End(context.Background(), session); err != nil {
		t.Fatal(err)
	}
	return assembler
}

func TestTranscriber_SendsLanguageHintPerLocale(t *testing.T) {
	cases := []struct {
		name         string
		locales      ports.SeatLocales
		wantLanguage string
		wantLocale   string
	}{
		{"spanish seat", func(seat domain.SeatID) string { return "es" }, "es-ES", "es"},
		{"english default", nil, "en-US", "en"},
		{"unknown falls back", func(seat domain.SeatID) string { return "fr" }, "en-US", "en"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stt := &fakeSTT{result: ports.Transcript{Text: "hola"}}
			executor, err := NewTranscriber(stt, localeRecording(t, domain.UtteranceID(tc.name)))
			if err != nil {
				t.Fatal(err)
			}
			executor.Locales = tc.locales
			scope := domain.Scope{Machine: vocab.MachineSession}
			executor.Execute(context.Background(), domain.Transcribe{Seat: 2, UtteranceID: domain.UtteranceID(tc.name)}, scope, &eventInbox{})
			if stt.request.Language != tc.wantLanguage {
				t.Fatalf("language = %q, want %q", stt.request.Language, tc.wantLanguage)
			}
			if stt.request.Meta.Locale != tc.wantLocale {
				t.Fatalf("meta locale = %q, want %q", stt.request.Meta.Locale, tc.wantLocale)
			}
		})
	}
}
