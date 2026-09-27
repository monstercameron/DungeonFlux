//go:build js && wasm

package phone

import (
	"context"
	"errors"
	"html"
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/v6/testkit/render"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

func TestTalkTypedInput_ShowsFailurePendingAndWaiting(t *testing.T) {
	for _, tc := range []struct {
		name       string
		prepare    func(*TypedInputModel)
		want, role string
		disabled   bool
	}{
		{"failure", func(m *TypedInputModel) {
			m.ApplySay(SayResult{Err: errors.New("Mother Vell is replying. Try again.")})
		}, "Mother Vell is replying. Try again.", "alert", false},
		{"sending", func(m *TypedInputModel) { m.Submit(context.Background()) }, "Sending…", "status", true},
		{"waiting", func(m *TypedInputModel) { m.ApplyScreenState(nil) }, "Chat opens when you talk to Mother Vell.", "status", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			render.New(t)
			model := NewTypedInputModel(&pendingSay{result: make(chan SayResult, 1)}, "seat")
			model.SetText("Where is he?")
			tc.prepare(model)
			markup, err := ui.RenderToString(ui.CreateElement(talkTypedInput, talkTypedProps{model: model, snapshot: model.Snapshot()}))
			if err != nil {
				t.Fatal(err)
			}
			markup = html.UnescapeString(markup)
			for _, want := range []string{"<form", `type="submit"`, `aria-describedby="talk-message-status"`, `role="` + tc.role + `"`, tc.want, "Where is he?"} {
				if !strings.Contains(markup, want) {
					t.Fatalf("missing %q: %s", want, markup)
				}
			}
			if strings.Contains(markup, "disabled") != tc.disabled {
				t.Fatalf("disabled mismatch: %s", markup)
			}
		})
	}
}
