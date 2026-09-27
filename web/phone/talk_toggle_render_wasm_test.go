//go:build js && wasm

package phone

import (
	"strings"
	"testing"
	"time"

	"github.com/monstercameron/GoWebComponents/v6/testkit/render"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

func renderTalkButton(t *testing.T, model *PTTModel) string {
	t.Helper()
	render.New(t)
	markup, err := ui.RenderToString(ui.CreateElement(talkPTTScreen, talkPTTProps{model: model, locale: "en"}))
	if err != nil {
		t.Fatalf("render talk button: %v", err)
	}
	return markup
}

func TestTalkPTTScreen_rendersToggleStateFromTheModel(t *testing.T) {
	start := time.Unix(1_700_000_000, 0)
	cases := []struct {
		name      string
		prepare   func(c *ToggleControl)
		wantGlyph string
		wantLabel string
		wantRed   bool
	}{
		{name: "idle", prepare: func(*ToggleControl) {}, wantGlyph: "●", wantLabel: PTTStart("en")},
		{name: "starting", prepare: func(c *ToggleControl) { c.Tap(start) }, wantGlyph: "■", wantLabel: PTTStop("en"), wantRed: true},
		{name: "recording", prepare: func(c *ToggleControl) { c.Tap(start); c.Started(nil) }, wantGlyph: "■", wantLabel: PTTStop("en"), wantRed: true},
		{name: "finishing", prepare: func(c *ToggleControl) { c.Tap(start); c.Started(nil); c.Tap(start.Add(time.Second)) }, wantGlyph: "…", wantLabel: PTTStop("en")},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			model := NewPTTModel(nil, "seat", 1)
			tc.prepare(model.Toggle())
			markup := renderTalkButton(t, model)
			if !strings.Contains(markup, "df-phone-talk-ptt") {
				t.Fatalf("markup lacks the talk control container: %s", markup)
			}
			if !strings.Contains(markup, ">"+tc.wantGlyph+"<") {
				t.Fatalf("markup lacks glyph %q: %s", tc.wantGlyph, markup)
			}
			if !strings.Contains(markup, `aria-label="`+tc.wantLabel+`"`) {
				t.Fatalf("markup lacks label %q: %s", tc.wantLabel, markup)
			}
			if red := strings.Contains(markup, "rgba(179,55,47,.85)"); red != tc.wantRed {
				t.Fatalf("recording colour shown = %v, want %v: %s", red, tc.wantRed, markup)
			}
		})
	}
}

func TestTalkPTTScreen_statusStaysVisuallyHidden(t *testing.T) {
	model := NewPTTModel(nil, "seat", 1)
	model.Toggle().Tap(time.Unix(1_700_000_000, 0))
	model.Toggle().Started(errMicDenied{})
	markup := renderTalkButton(t, model)
	if !strings.Contains(markup, `role="status"`) || !strings.Contains(markup, "Permission denied") {
		t.Fatalf("status line lacks the microphone error for screen readers: %s", markup)
	}
	if !strings.Contains(markup, "clip:rect(0 0 0 0)") {
		t.Fatalf("status line is no longer visually hidden, which shifts the layout: %s", markup)
	}
}

func TestTalkPTTScreen_rerenderKeepsTheRecording(t *testing.T) {
	model := NewPTTModel(nil, "seat", 1)
	model.Toggle().Tap(time.Unix(1_700_000_000, 0))
	model.Toggle().Started(nil)
	for _, render := range []string{"first render", "remount"} {
		t.Run(render, func(t *testing.T) {
			if markup := renderTalkButton(t, model); !strings.Contains(markup, ">■<") {
				t.Fatalf("%s lost the live recording: %s", render, markup)
			}
		})
	}
	if model.Toggle().Phase() != ToggleRecording {
		t.Fatalf("phase after re-render = %q, want recording", model.Toggle().Phase())
	}
}

type errMicDenied struct{}

func (errMicDenied) Error() string { return "Permission denied" }
