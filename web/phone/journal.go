package phone

import "strings"

// JournalEntry is one story beat the player has heard, kept for the Journal
// tab so a player can review clues (the lamplighter, the bell tower, the
// courier's letter) without having reread the whole scene transcript.
type JournalEntry struct {
	Speaker string
	Text    string
}

// JournalLog accumulates finished narration beats client-side as the engine
// delivers them. The server does not keep a discrete clue list for the demo
// one-shot; every beat that reaches the phone (opening, an NPC reveal, the
// stranger's lines, the cliffhanger) is itself real engine narration, so
// recording it here is a faithful, additive projection rather than invented
// content. Entries are deduplicated by line ID so re-renders of the same
// snapshot never repeat a beat.
type JournalLog struct {
	seen    map[string]bool
	entries []JournalEntry
}

// NewJournalLog creates an empty journal.
func NewJournalLog() *JournalLog { return &JournalLog{seen: make(map[string]bool)} }

// Record appends a finished narration beat if it carries text and has not
// already been recorded. lineID is preferred as the dedupe key; when a
// server fixture carries no line ID, the speaker and text stand in for one.
func (j *JournalLog) Record(lineID, speaker, text string) {
	text = strings.TrimSpace(text)
	if j == nil || text == "" {
		return
	}
	key := strings.TrimSpace(lineID)
	if key == "" {
		key = speaker + "|" + text
	}
	if j.seen == nil {
		j.seen = make(map[string]bool)
	}
	if j.seen[key] {
		return
	}
	j.seen[key] = true
	j.entries = append(j.entries, JournalEntry{Speaker: strings.TrimSpace(speaker), Text: text})
}

// Entries returns a copy of the recorded beats, oldest first.
func (j *JournalLog) Entries() []JournalEntry {
	if j == nil {
		return nil
	}
	return append([]JournalEntry(nil), j.entries...)
}
