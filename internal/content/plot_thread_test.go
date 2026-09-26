package content

import (
	"encoding/json"
	"testing"
)

func TestDefaultPlotThreads_ValidateAgainstOneShot(t *testing.T) {
	story := DefaultOneShot()
	threads := DefaultPlotThreads()
	if err := threads.Validate(story.Beats); err != nil {
		t.Fatalf("default threads should validate: %v", err)
	}
	if len(threads.Threads) != 5 {
		t.Fatalf("demo funnel needs five threads, got %d", len(threads.Threads))
	}
}

func TestPlotThreads_EveryOutcomeHasARung(t *testing.T) {
	threads := DefaultPlotThreads()
	wants := []struct{ thread, condition string }{
		{"find_lamplighter", CondPersuadeSuccess},
		{"find_lamplighter", CondPersuadeFail},
		{"courier_hook", CondClueGranted},
		{"courier_hook", CondPersuadeFail},
		{"thrall_fight", CondCombatSlain},
		{"thrall_fight", CondCombatFled},
		{"midnight_bell", CondClueGranted},
		{"midnight_bell", CondPersuadeFail},
	}
	for _, want := range wants {
		t.Run(want.thread+"/"+want.condition, func(t *testing.T) {
			line, ok := threads.IntroFor(want.thread, want.condition)
			if !ok || line == "" {
				t.Fatalf("missing intro for %s on %s", want.thread, want.condition)
			}
		})
	}
}

func TestPlotThreads_RungForMisses(t *testing.T) {
	threads := DefaultPlotThreads()
	if _, ok := threads.RungFor("missing", CondEnter); ok {
		t.Fatal("unknown thread should miss")
	}
	if _, ok := threads.RungFor("opening", CondCombatSlain); ok {
		t.Fatal("unstaged condition should miss")
	}
	if _, ok := threads.IntroFor("opening", CondCombatSlain); ok {
		t.Fatal("unstaged intro should miss")
	}
}

func TestPlotThreads_LoadRoundTrip(t *testing.T) {
	raw, err := json.Marshal(DefaultPlotThreads())
	if err != nil {
		t.Fatalf("marshal default threads: %v", err)
	}
	loaded, err := LoadPlotThreads(raw)
	if err != nil {
		t.Fatalf("load threads JSON: %v", err)
	}
	if len(loaded.Threads) != 5 {
		t.Fatalf("round trip lost threads: %#v", loaded)
	}
	line, ok := loaded.IntroFor("thrall_fight", CondCombatFled)
	if !ok || line == "" {
		t.Fatal("round trip lost the flee rung")
	}
}

func TestPlotThreadsValidate_RejectsMalformedInputs(t *testing.T) {
	story := DefaultOneShot()
	tests := []struct {
		name string
		edit func(*PlotThreads)
	}{
		{"empty", func(p *PlotThreads) { p.Threads = nil }},
		{"duplicate", func(p *PlotThreads) { p.Threads = append(p.Threads, p.Threads[0]) }},
		{"synopsis", func(p *PlotThreads) { p.Threads[0].Synopsis = "" }},
		{"beat", func(p *PlotThreads) { p.Threads[0].BeatID = "nowhere" }},
		{"no rungs", func(p *PlotThreads) { p.Threads[0].Rungs = nil }},
		{"no opening first", func(p *PlotThreads) { p.Threads[0].Rungs[0].When = CondCombatSlain }},
		{"condition", func(p *PlotThreads) { p.Threads[0].Rungs[0].When = "never" }},
		{"repeat", func(p *PlotThreads) { p.Threads[1].Rungs[2].When = CondEnter }},
		{"line", func(p *PlotThreads) { p.Threads[0].Rungs[0].IntroLine = "" }},
		{"cap", func(p *PlotThreads) { p.Threads[3].Rungs[0].WordCap = 2 }},
		{"lead", func(p *PlotThreads) { p.Threads[0].Rungs[0].Leads = []string{"nowhere"} }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			threads := DefaultPlotThreads()
			test.edit(&threads)
			if err := threads.Validate(story.Beats); err == nil {
				t.Fatal("Validate accepted malformed threads")
			}
		})
	}
}

func TestLoadPlotThreads_RejectsBadJSON(t *testing.T) {
	if _, err := LoadPlotThreads([]byte("[oops")); err == nil {
		t.Fatal("malformed JSON should be rejected")
	}
	if _, err := LoadPlotThreads([]byte(`{"threads":[]}`)); err == nil {
		t.Fatal("empty thread set should be rejected on load")
	}
}

func TestKnownCondition_CoversFunnel(t *testing.T) {
	for _, condition := range []string{CondEnter, CondClueGranted, CondPersuadeSuccess, CondPersuadeFail, CondCombatSlain, CondCombatFled} {
		if !KnownCondition(condition) {
			t.Fatalf("funnel condition %q should be known", condition)
		}
	}
	if KnownCondition("never") {
		t.Fatal("unknown condition should not be known")
	}
}
