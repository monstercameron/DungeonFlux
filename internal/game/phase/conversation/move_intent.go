package conversation

import (
	"strings"
	"unicode"

	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

// hasMoveKeyword gates interpretation; ordinary questions never wait for a
// classifier that could turn unrelated dialogue into a mechanical action.
func hasMoveKeyword(text string) bool {
	words := strings.FieldsFunc(strings.ToLower(text), func(r rune) bool { return !unicode.IsLetter(r) })
	for _, word := range words {
		switch word {
		case "persuade", "convince", "later", "persuadir", "convencer":
			return true
		}
	}
	normal := " " + strings.Join(words, " ") + " "
	for _, phrase := range []string{" step away ", " never mind ", " me aparto ", " más tarde ", " no importa "} {
		if strings.Contains(normal, phrase) {
			return true
		}
	}
	return false
}

// keywordMove is the failure fallback, not a semantic classifier. Ambiguous
// questions, negation and mentions of someone else's actions remain dialogue.
func keywordMove(text string) vocab.MoveID {
	if strings.ContainsAny(text, "?¿") {
		return ""
	}
	command := strings.ToLower(strings.Trim(text, ".! \t\n\r"))
	command = strings.Join(strings.Fields(command), " ")
	command = strings.TrimPrefix(command, "please ")
	for _, prefix := range []string{"i try to ", "i attempt to ", "i want to ", "i ", "try to "} {
		if after, ok := strings.CutPrefix(command, prefix); ok {
			command = after
			break
		}
	}
	for _, verb := range []string{"persuade", "convince", "persuadir", "convencer"} {
		if command == verb || strings.HasPrefix(command, verb+" ") {
			return vocab.MovePersuade
		}
	}
	switch command {
	case "step away", "later", "never mind", "me aparto", "más tarde", "no importa":
		return vocab.MoveStepAway
	}
	return ""
}
