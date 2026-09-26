package splat

import (
	"strings"
	"testing"
)

func TestEnvelope_includesVersionAndType(t *testing.T) {
	raw, err := envelope("pause", Pause{On: true})
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	for _, want := range []string{`"v":1`, `"type":"pause"`, `"on":true`} {
		if !strings.Contains(text, want) {
			t.Fatalf("envelope %s does not contain %s", text, want)
		}
	}
}

func TestDecodeEvent_readsReadyAndPickFields(t *testing.T) {
	event, err := decodeEvent(`{"type":"ready","fps":59.5,"gaussians":100000,"device":"webgl2"}`)
	if err != nil {
		t.Fatal(err)
	}
	if event.Type != "ready" || event.FPS != 59.5 || event.Gaussians != 100000 || event.Device != "webgl2" {
		t.Fatalf("event = %+v", event)
	}
	if _, err := decodeEvent("not json"); err == nil {
		t.Fatal("expected malformed event to fail")
	}
}
