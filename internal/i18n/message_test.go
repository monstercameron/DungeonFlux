package i18n

import (
	"testing"
)

func TestMessage_BuildersAndRender(t *testing.T) {
	catalog := NewCatalog(map[string]map[string]Entry{
		"en": {"reason.WAITING_FOR_PLAYER": {Text: "Waiting for {name}"}},
		"es": {"reason.WAITING_FOR_PLAYER": {Text: "Esperando a {name}"}},
	})
	msg := Text("reason.WAITING_FOR_PLAYER", "Waiting for {name}").WithArgs(map[string]string{"name": "Vell"})
	if got, want := msg.Render(catalog, "es"), "Esperando a Vell"; got != want {
		t.Fatalf("Render es = %q, want %q", got, want)
	}
	if got, want := msg.Render(catalog, "en"), "Waiting for Vell"; got != want {
		t.Fatalf("Render en = %q, want %q", got, want)
	}
	if got, want := Text("missing.key", "fallback text").Render(catalog, "es"), "fallback text"; got != want {
		t.Fatalf("Render missing = %q, want %q", got, want)
	}

	base := Text("reason.WAITING_FOR_PLAYER", "Waiting for {name}")
	merged := base.WithArgs(map[string]string{"name": "Vell"}).WithArgs(map[string]string{"extra": "x"}).WithCount(2)
	if merged.Args["name"] != "Vell" || merged.Args["extra"] != "x" || merged.Count != 2 {
		t.Fatalf("WithArgs/WithCount merge = %+v", merged)
	}
	if base.Args != nil || base.Count != 0 {
		t.Fatalf("builders mutated the original: %+v", base)
	}
	if got := (Message{Key: "k"}).WithArgs(nil); got.Args != nil {
		t.Fatalf("WithArgs(nil) set args: %+v", got)
	}
}
