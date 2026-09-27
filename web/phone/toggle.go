package phone

import (
	"sync"
	"time"
)

// toggleDebounce ignores a second tap this soon after the first, so one
// physical tap that fires twice (pointer and click, or a double bounce) cannot
// start and immediately stop a recording.
const toggleDebounce = 400 * time.Millisecond

// TogglePhase is where a tap-to-talk recording is in its lifecycle.
type TogglePhase string

const (
	// ToggleIdle means no recording is active.
	ToggleIdle TogglePhase = "idle"
	// ToggleStarting means the microphone and Talk stream are opening.
	ToggleStarting TogglePhase = "starting"
	// ToggleRecording means the microphone is live and chunks are uploading.
	ToggleRecording TogglePhase = "recording"
	// ToggleFinishing means the second tap arrived and the recording is ending.
	ToggleFinishing TogglePhase = "finishing"
)

// ToggleAction is the side effect the view performs after a transition.
type ToggleAction string

const (
	// ToggleNone means nothing to do.
	ToggleNone ToggleAction = ""
	// ToggleStart opens the microphone and the Talk stream.
	ToggleStart ToggleAction = "start"
	// ToggleFinish stops the recorder and sends TalkEnd.
	ToggleFinish ToggleAction = "finish"
)

// ToggleControl is the tap-to-start, tap-to-send state machine behind the talk
// button. It lives on the PTTModel rather than in the component, so a screen
// that remounts mid-recording (a server snapshot or an art refresh) keeps the
// recording alive; the component-held state it replaces was cancelled by every
// remount. It has no browser dependencies, so native tests cover it.
type ToggleControl struct {
	mu      sync.Mutex
	phase   TogglePhase
	stop    bool
	lastTap time.Time
	notice  string
}

// Phase returns the current lifecycle phase.
func (c *ToggleControl) Phase() TogglePhase {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.phase == "" {
		return ToggleIdle
	}
	return c.phase
}

// Notice returns the last message for the player, such as a microphone error.
func (c *ToggleControl) Notice() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.notice
}

// Tap handles one tap at now. The first tap starts a recording and the next
// sends it; a tap while the microphone is still opening sends as soon as it
// opens. Taps within toggleDebounce of the previous one are ignored.
func (c *ToggleControl) Tap(now time.Time) ToggleAction {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.lastTap.IsZero() && now.Sub(c.lastTap) < toggleDebounce {
		return ToggleNone
	}
	c.lastTap = now
	switch c.phase {
	case "", ToggleIdle:
		c.phase, c.stop, c.notice = ToggleStarting, false, ""
		return ToggleStart
	case ToggleStarting:
		c.stop = true
		return ToggleNone
	case ToggleRecording:
		c.phase = ToggleFinishing
		return ToggleFinish
	default:
		return ToggleNone
	}
}

// Started reports whether the microphone opened. A failure returns to idle
// with the error as the notice; a tap that arrived while opening sends now.
func (c *ToggleControl) Started(err error) ToggleAction {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.phase != ToggleStarting {
		return ToggleNone
	}
	if err != nil {
		c.phase, c.notice = ToggleIdle, err.Error()
		return ToggleNone
	}
	if c.stop {
		c.phase = ToggleFinishing
		return ToggleFinish
	}
	c.phase = ToggleRecording
	return ToggleNone
}

// Finished returns the control to idle. notice is empty after a normal send,
// or the error that ended the recording.
func (c *ToggleControl) Finished(notice string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.phase, c.stop, c.notice = ToggleIdle, false, notice
}

// Toggle returns the model's tap-to-talk control, which outlives any one
// render of the talk screen.
func (m *PTTModel) Toggle() *ToggleControl {
	if m == nil {
		return nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.toggle == nil {
		m.toggle = &ToggleControl{}
	}
	return m.toggle
}

// talkStatus returns the visible status line, the button glyph, and its label.
// A notice (a microphone or upload error) wins over the phase text, so a
// failure is never silent.
func talkStatus(locale string, phase TogglePhase, notice string) (status, label, aria string) {
	label, aria = "●", PTTStart(locale)
	switch phase {
	case ToggleStarting, ToggleRecording:
		label, aria, status = "■", PTTStop(locale), T(locale, "ui.ptt.recording", nil)
	case ToggleFinishing:
		label, aria, status = "…", PTTStop(locale), T(locale, "ptt.finishing", nil)
	default:
		status = PTTReady(locale)
	}
	if notice != "" && phase == ToggleIdle {
		status = notice
	}
	return status, label, aria
}
