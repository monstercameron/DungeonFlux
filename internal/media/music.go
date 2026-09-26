package media

import (
	"github.com/monstercameron/DungeonFlux/internal/content"
)

// CueKind identifies the playback operation requested by a music cue.
type CueKind string

const (
	// CueNone means that no playback change is needed.
	CueNone CueKind = "none"
	// CueLoopTransition changes a loop at an outgoing bar line.
	CueLoopTransition CueKind = "loop_transition"
	// CueStinger starts a short urgent musical accent.
	CueStinger CueKind = "stinger"
)

// MusicCue is a deterministic instruction for the browser music mixer.
// StartAtMS is the earliest legal start time; a voice-gated stinger waits
// until the current line has ended before starting at or after that time.
type MusicCue struct {
	Kind        CueKind
	FromTrack   string
	Track       string
	StartAtMS   int
	CrossfadeMS int
	WaitForLine bool
}

// BarMSForBPM returns the duration of a four-beat bar at the supplied tempo.
// A non-positive tempo has no valid bar duration and returns zero.
func BarMSForBPM(bpm int) int {
	if bpm <= 0 {
		return 0
	}
	return 240000 / bpm
}

// NextBarMS returns the first bar boundary at or after positionMS. The
// downbeat is the absolute timestamp of bar zero and may be non-zero.
// Invalid bar durations return positionMS unchanged.
func NextBarMS(positionMS, downbeatMS, barMS int) int {
	if barMS <= 0 {
		return positionMS
	}
	if positionMS <= downbeatMS {
		return downbeatMS
	}
	delta := positionMS - downbeatMS
	if delta%barMS == 0 {
		return positionMS
	}
	return downbeatMS + (delta/barMS+1)*barMS
}

// ScheduleLoopTransition schedules a loop change on the outgoing track's
// next bar line. CrossfadeBars defaults to one bar, matching the demo mixer
// rule, and never produces a negative timestamp.
func ScheduleLoopTransition(from, to content.MusicTrack, positionMS int) MusicCue {
	if to.ID == "" || from.ID == to.ID {
		return MusicCue{Kind: CueNone, FromTrack: from.ID, Track: to.ID, StartAtMS: positionMS}
	}
	barMS := from.BarMS
	if barMS <= 0 {
		barMS = BarMSForBPM(from.BPM)
	}
	start := NextBarMS(positionMS, from.DownbeatMS, barMS)
	crossfadeBars := from.CrossfadeBars
	if crossfadeBars <= 0 {
		crossfadeBars = 1
	}
	return MusicCue{
		Kind:        CueLoopTransition,
		FromTrack:   from.ID,
		Track:       to.ID,
		StartAtMS:   start,
		CrossfadeMS: crossfadeBars * barMS,
	}
}

// ScheduleStinger schedules an urgent stinger immediately, or marks it to
// wait for line_done when voice is currently active. Stingers never crossfade.
func ScheduleStinger(stinger content.MusicTrack, positionMS int, voiceActive bool) MusicCue {
	return MusicCue{
		Kind:        CueStinger,
		Track:       stinger.ID,
		StartAtMS:   positionMS,
		WaitForLine: voiceActive,
	}
}
