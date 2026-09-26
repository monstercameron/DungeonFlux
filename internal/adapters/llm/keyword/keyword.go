package keyword

import (
	"strings"
	"unicode"

	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

// Match returns the first legal move whose keyword occurs in transcript.
// Matching is case-insensitive and ignores punctuation around words. A move
// is never returned unless it appears in legalMoves.
func Match(transcript string, legalMoves []vocab.MoveID) (vocab.MoveID, bool) {
	words := tokenize(transcript)
	if len(words) == 0 {
		return "", false
	}
	for _, move := range legalMoves {
		for _, phrase := range keywordsFor(move) {
			if containsPhrase(words, tokenize(phrase)) {
				return move, true
			}
		}
	}
	return "", false
}

// Keywords returns a copy of the keywords recognized for move.
func Keywords(move vocab.MoveID) []string {
	return append([]string(nil), keywordsFor(move)...)
}

func keywordsFor(move vocab.MoveID) []string {
	switch move {
	case vocab.MovePersuade:
		return []string{"persuade", "convince"}
	case vocab.MoveStepAway:
		return []string{"step away", "later", "never mind"}
	default:
		return nil
	}
}

func tokenize(text string) []string {
	var words []string
	var current []rune
	flush := func() {
		if len(current) == 0 {
			return
		}
		words = append(words, strings.ToLower(string(current)))
		current = current[:0]
	}
	for _, r := range text {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			current = append(current, r)
			continue
		}
		flush()
	}
	flush()
	return words
}

func containsPhrase(words, phrase []string) bool {
	if len(phrase) == 0 || len(phrase) > len(words) {
		return false
	}
	for start := 0; start <= len(words)-len(phrase); start++ {
		matched := true
		for offset := range phrase {
			if words[start+offset] != phrase[offset] {
				matched = false
				break
			}
		}
		if matched {
			return true
		}
	}
	return false
}
