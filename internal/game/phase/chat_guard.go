package phase

import (
	"strings"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

// ChatRejection explains why the player should keep a message for retry.
type ChatRejection struct{ Reason string }

// Error returns the player-facing reason for refusing a chat message.
func (r *ChatRejection) Error() string { return r.Reason }

func (m Machine) validateSay(say domain.Say) error {
	var reason string
	switch {
	case m.paused:
		reason = "The game is paused. Keep your message and send it after the host resumes."
	case m.State() != vocab.StateConversation || m.conversation.Done:
		reason = "Chat is available while talking to Mother Vell."
	case say.Seat < 1 || say.Seat > 2 || say.Seat != m.spotlight:
		reason = "Wait for your turn to speak to Mother Vell."
	case m.conversation.UtteranceInFlight || m.conversation.VoiceBusy:
		reason = "Mother Vell is replying. Wait for her to finish, then send your message."
	case say.UtteranceID == "" || strings.TrimSpace(say.Text) == "":
		reason = "Enter a message before sending."
	}
	if reason != "" {
		return &ChatRejection{Reason: reason}
	}
	return nil
}

// finishPausedReply consumes completion without advancing the paused story.
// Speech workers can finish while timers and player actions are paused.
func (m *Machine) finishPausedReply(event domain.Event) (Result, bool, error) {
	if m.State() != vocab.StateConversation {
		return Result{}, false, nil
	}
	switch event.(type) {
	case domain.LineDone, domain.LineFailed:
		out, err := m.stepConversation(event)
		out.Paused = true
		return out, true, err
	default:
		return Result{}, false, nil
	}
}
