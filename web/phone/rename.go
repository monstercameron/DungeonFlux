package phone

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"

	df "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
)

// heroNameMaxLength mirrors internal/game/phase/creation.MaxHeroNameLength.
// web/phone may not import that package (it is not on the phone import
// allowlist, internal/archtest), so the limit and the sanitize rules are
// duplicated here for the player-facing check; the engine's rename move is
// still the authoritative validator.
const heroNameMaxLength = 24

// BeginRename opens the hero-name editor, seeded with the current name so
// the player edits rather than retypes it.
func (m *CreationModel) BeginRename() CreationSnapshot {
	if m == nil {
		return CreationSnapshot{Phase: CreationFailed, Error: "creation model is unavailable"}
	}
	current := ""
	if m.state.Build != nil {
		current = m.state.Build.GetName()
	}
	m.state.Renaming, m.state.RenameDraft, m.state.RenameError = true, current, ""
	return m.Snapshot()
}

// SetRenameDraft records the in-progress edit. It does not validate on every
// keystroke, so the player can pass through a momentarily invalid value
// (e.g. while deleting to retype) without an error flashing mid-edit.
func (m *CreationModel) SetRenameDraft(text string) CreationSnapshot {
	if m == nil {
		return CreationSnapshot{Phase: CreationFailed, Error: "creation model is unavailable"}
	}
	m.state.RenameDraft, m.state.RenameError = text, ""
	return m.Snapshot()
}

// CancelRename closes the editor without submitting a change.
func (m *CreationModel) CancelRename() CreationSnapshot {
	if m == nil {
		return CreationSnapshot{Phase: CreationFailed, Error: "creation model is unavailable"}
	}
	m.state.Renaming, m.state.RenameDraft, m.state.RenameError = false, "", ""
	return m.Snapshot()
}

// SubmitRename validates the draft locally (trim, strip control characters
// and markup punctuation, collapse whitespace, 1-24 characters) and, on
// success, sends it to the engine as a rename move. The engine re-validates
// with internal/game/phase/creation.SanitizeHeroName and is authoritative;
// a rejection there (for example the seat already locked) is surfaced as
// RenameError the same way a local validation failure is. On acceptance the
// editor closes; the caller re-renders from a fresh ScreenState afterward.
func (m *CreationModel) SubmitRename(ctx context.Context) <-chan ActResult {
	result := make(chan ActResult, 1)
	if m == nil || m.client == nil {
		m.setRenameErrorSafely("creation client is unavailable")
		result <- ActResult{Err: errors.New("creation client is unavailable")}
		close(result)
		return result
	}
	name, err := sanitizeHeroNameDraft(m.state.RenameDraft)
	if err != nil {
		m.state.RenameError = renameErrorText(err)
		result <- ActResult{}
		close(result)
		return result
	}
	request := &df.ActRequest{SeatToken: m.state.SeatToken, MoveId: "rename", Arg: name}
	go func() {
		outcome := <-m.client.Act(ctx, request)
		switch {
		case outcome.Err != nil:
			m.state.RenameError = outcome.Err.Error()
		case outcome.Value != nil && !outcome.Value.GetAccepted():
			reason := outcome.Value.GetReason()
			if reason == "" {
				reason = "server rejected the name"
			}
			m.state.RenameError = reason
		default:
			m.state.Renaming, m.state.RenameDraft, m.state.RenameError = false, "", ""
		}
		result <- outcome
	}()
	return result
}

func (m *CreationModel) setRenameErrorSafely(message string) {
	if m != nil {
		m.state.RenameError = message
	}
}

// renameErrorText maps a local validation error to player-facing text; the
// view renders the same two cases through the i18n catalog (RenameErrorKey).
func renameErrorText(err error) string {
	switch {
	case errors.Is(err, errRenameEmpty):
		return "empty"
	case errors.Is(err, errRenameTooLong):
		return "too_long"
	default:
		return "invalid"
	}
}

var (
	errRenameEmpty   = errors.New("hero name is empty")
	errRenameTooLong = errors.New("hero name is too long")
)

// sanitizeHeroNameDraft mirrors the engine's SanitizeHeroName rules.
func sanitizeHeroNameDraft(raw string) (string, error) {
	var b strings.Builder
	for _, r := range strings.TrimSpace(raw) {
		switch {
		case r < 0x20 || r == 0x7f:
			continue
		case r == '<' || r == '>' || r == '{' || r == '}' || r == '`' || r == '\\':
			continue
		default:
			b.WriteRune(r)
		}
	}
	name := strings.Join(strings.Fields(b.String()), " ")
	if name == "" {
		return "", errRenameEmpty
	}
	if utf8.RuneCountInString(name) > heroNameMaxLength {
		return "", errRenameTooLong
	}
	return name, nil
}
