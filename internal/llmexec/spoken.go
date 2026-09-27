package llmexec

import (
	"regexp"
	"strings"
)

// jsonKey matches a JSON-style label such as location_of_lamplighter: that a
// format-hijacking player can coax out of the model.
var jsonKey = regexp.MustCompile(`(?i)\b[a-z]+(?:_[a-z]+)+\s*:\s*`)

// spokenText reduces a model reply to the words a voice should say: it drops
// *stage directions* and quotation marks, which TTS would otherwise read aloud
// ("asterisk, Mother Vell gives a gravelly chuckle") and which also counted
// against the line's word cap. JSON braces, brackets, and snake_case keys are
// dropped too, so a "reply in JSON" trick is still spoken as plain words.
// Apostrophes are kept.
func spokenText(text string) string {
	var out strings.Builder
	inAction := false
	for _, r := range text {
		switch r {
		case '*':
			inAction = !inAction
			continue
		case '"', '“', '”', '«', '»', '{', '}', '[', ']':
			continue
		}
		if !inAction {
			out.WriteRune(r)
		}
	}
	if inAction {
		// An unclosed asterisk: keep what was said rather than lose the line.
		return strings.Join(strings.Fields(strings.ReplaceAll(text, "*", "")), " ")
	}
	return strings.Join(strings.Fields(jsonKey.ReplaceAllString(out.String(), "")), " ")
}

// clueWords are the words of the gated clue and its neighbours. Mother Vell's
// ordinary replies never need them, and red-team runs coaxed them out through
// quotes, rhymes, and forged history, so a reply containing one is rejected
// and the canned evasive line plays instead. The reveal is not checked: it is
// the one line meant to carry the clue.
var clueWords = regexp.MustCompile(`(?i)\b(bells?|towers?|belfry|belfries|steeples?|spires?|ringers?|clappers?|toll(?:s|ed|ing)?|rang|rung|ringing|midnight|campanario|campanas?|chimes?|chiming|carillons?|knells?|peals?|climb(?:s|ed|ing)?|clock\s+struck)\b`)

// leaksClue reports whether an NPC reply names the gated clue or its words.
func leaksClue(reply string) bool {
	return clueWords.MatchString(reply)
}

// patronLine frames the player's words as a patron's speech, so text inside
// it (a forged "Vell:" line, a system note, a request to repeat a phrase) is
// read as something a patron said, not as instructions or her own history.
func patronLine(said string) string {
	said = strings.TrimSpace(said)
	if said == "" {
		return "A patron says nothing."
	}
	return "A patron says: <<" + strings.ReplaceAll(said, ">>", "> >") + ">>"
}
