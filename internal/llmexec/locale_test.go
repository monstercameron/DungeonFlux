package llmexec

import (
	"strings"
	"testing"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/ports"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

func localeTestRequest() ports.TextRequest {
	return ports.TextRequest{
		Messages: []ports.Message{
			{Role: vocab.MsgSystem, Text: "You narrate."},
			{Role: vocab.MsgUser, Text: "Begin."},
		},
		MaxTokens: 64,
	}
}

func TestWithLocale_AppendsLanguageInstruction(t *testing.T) {
	got := WithLocale(localeTestRequest(), "es")
	if got.Meta.Locale != "es" {
		t.Fatalf("meta locale = %q", got.Meta.Locale)
	}
	if !strings.Contains(got.Messages[0].Text, "Respond in Spanish") {
		t.Fatalf("system message = %q", got.Messages[0].Text)
	}
	if !strings.Contains(got.Messages[0].Text, "You narrate.") {
		t.Fatalf("system prompt lost: %q", got.Messages[0].Text)
	}
	if got.Messages[1].Text != "Begin." {
		t.Fatalf("user message changed: %q", got.Messages[1].Text)
	}
}

func TestWithLocale_EnglishLeavesPromptAlone(t *testing.T) {
	before := localeTestRequest()
	got := WithLocale(before, "en")
	if got.Meta.Locale != "en" {
		t.Fatalf("meta locale = %q", got.Meta.Locale)
	}
	if got.Messages[0].Text != "You narrate." {
		t.Fatalf("english system changed: %q", got.Messages[0].Text)
	}
}

func TestWithLocale_UnknownFallsBackToEnglish(t *testing.T) {
	got := WithLocale(localeTestRequest(), "fr")
	if got.Meta.Locale != "en" {
		t.Fatalf("meta locale = %q", got.Meta.Locale)
	}
	if strings.Contains(got.Messages[0].Text, "Spanish") {
		t.Fatalf("unknown locale added instruction: %q", got.Messages[0].Text)
	}
}

func TestLocaleSource_ResolvesRoomAndSeat(t *testing.T) {
	var zero LocaleSource
	if zero.ForRoom() != "en" || zero.ForSeat(1) != "en" {
		t.Fatal("zero source must resolve English")
	}
	source := LocaleSource{
		Room: func() string { return "es" },
		Seat: func(seat domain.SeatID) string {
			if seat == 2 {
				return "en"
			}
			return ""
		},
	}
	if source.ForRoom() != "es" {
		t.Fatalf("room = %q", source.ForRoom())
	}
	if source.ForSeat(1) != "es" {
		t.Fatalf("seat 1 = %q", source.ForSeat(1))
	}
	if source.ForSeat(2) != "en" {
		t.Fatalf("seat 2 = %q", source.ForSeat(2))
	}
}
