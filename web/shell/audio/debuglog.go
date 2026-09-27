package audio

import (
	"fmt"
	"sort"

	dungeonfluxv1 "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
)

// ReceiveLog summarises what arrives on the Listen stream for the browser's
// ?debug=audio console log: one line per finished voice line (frames, bytes)
// and a note for cancels. It is pure, so native tests cover it.
type ReceiveLog struct {
	frames map[string]int
	bytes  map[string]int
}

// Observe records one Listen message and returns the console line to print,
// or "" when there is nothing to report yet.
func (r *ReceiveLog) Observe(message *dungeonfluxv1.AudioMessage) string {
	if r.frames == nil {
		r.frames, r.bytes = make(map[string]int), make(map[string]int)
	}
	if frame := message.GetFrame(); frame != nil {
		id := frame.GetUtteranceId()
		r.frames[id]++
		r.bytes[id] += len(frame.GetPcmS16Le())
		if !frame.GetFinal() {
			return ""
		}
		line := fmt.Sprintf("[audio] received voice line %s (%s): %d frames, %d bytes, %d Hz", id, frame.GetSpeaker(), r.frames[id], r.bytes[id], frame.GetSampleRate())
		delete(r.frames, id)
		delete(r.bytes, id)
		return line
	}
	if cancel := message.GetCancel(); cancel != nil {
		return fmt.Sprintf("[audio] cancel received for %s (all=%v)", cancel.GetUtteranceId(), cancel.GetAll())
	}
	return ""
}

// Pending lists voice lines that have started but not finished, for a
// debugging snapshot.
func (r *ReceiveLog) Pending() []string {
	ids := make([]string, 0, len(r.frames))
	for id := range r.frames {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}
