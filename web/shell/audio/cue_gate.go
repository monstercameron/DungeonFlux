package audio

import "time"

const cueDedupeWindow = 150 * time.Millisecond

// CueGate suppresses repeated playback of one cue inside the short tactile
// dedupe window. It is deterministic and uses the browser audio clock supplied
// by its caller, so it is straightforward to test without a browser.
type CueGate struct {
	last map[string]time.Duration
}

// Allow reports whether cue may play at at, recording accepted cues.
func (g *CueGate) Allow(cue string, at time.Duration) bool {
	if g == nil || cue == "" {
		return false
	}
	if g.last == nil {
		g.last = make(map[string]time.Duration)
	}
	if previous, ok := g.last[cue]; ok && at-previous < cueDedupeWindow {
		return false
	}
	g.last[cue] = at
	return true
}

// Reset forgets all previously played cues.
func (g *CueGate) Reset() {
	if g != nil {
		g.last = nil
	}
}
