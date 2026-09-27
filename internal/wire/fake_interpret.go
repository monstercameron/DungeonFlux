package wire

import (
	"strings"

	"github.com/monstercameron/DungeonFlux/internal/ports"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

// fakeInterpret is deliberately conservative: fake mode understands explicit
// action commands, not implied persuasion. Everything else stays dialogue.
func fakeInterpret(messages []ports.Message) map[string]any {
	text, legal := fakeInterpretInput(messages)
	result := map[string]any{"clean_text": text, "kind": "DIALOGUE", "move_id": nil}
	move := fakeExplicitMove(text)
	for _, candidate := range strings.Split(legal, ",") {
		if move != "" && strings.TrimSpace(candidate) == string(move) {
			result["kind"], result["move_id"] = "MOVE", string(move)
			break
		}
	}
	return result
}

func fakeInterpretInput(messages []ports.Message) (string, string) {
	for _, message := range messages {
		if message.Role != vocab.MsgUser {
			continue
		}
		body, ok := strings.CutPrefix(message.Text, "Classify this utterance: ")
		if !ok {
			continue
		}
		end := strings.LastIndex(body, ". Phase: ")
		if end < 0 {
			return strings.TrimSpace(body), ""
		}
		text := strings.TrimSpace(body[:end])
		_, tail, _ := strings.Cut(body[end:], ". Legal moves: ")
		legal, _, _ := strings.Cut(tail, ". NPC's last line: ")
		return text, legal
	}
	return "", ""
}

func fakeExplicitMove(text string) vocab.MoveID {
	if strings.ContainsAny(text, "?¿") {
		return ""
	}
	command := strings.ToLower(strings.TrimSpace(text))
	command = strings.Trim(command, ".! ")
	command = strings.Join(strings.Fields(command), " ")
	command = strings.TrimPrefix(command, "please ")
	for _, prefix := range []string{"i try to ", "i attempt to ", "i want to ", "i ", "try to "} {
		if strings.HasPrefix(command, prefix) {
			command = strings.TrimPrefix(command, prefix)
			break
		}
	}
	for _, verb := range []string{"persuade", "convince", "persuadir", "convencer"} {
		if command == verb || strings.HasPrefix(command, verb+" ") {
			return vocab.MovePersuade
		}
	}
	switch command {
	case "step away", "later", "never mind", "nevermind", "me aparto", "más tarde", "no importa":
		return vocab.MoveStepAway
	}
	return ""
}
