package audio

import (
	"testing"
	"time"
)

func TestCueGate_DeduplicatesOnlyWithinWindow(t *testing.T) {
	var gate CueGate
	if !gate.Allow("tick", 0) {
		t.Fatal("first cue was rejected")
	}
	if gate.Allow("tick", 149*time.Millisecond) {
		t.Fatal("cue repeated inside dedupe window")
	}
	if !gate.Allow("tick", 150*time.Millisecond) {
		t.Fatal("cue was not released at dedupe boundary")
	}
	if !gate.Allow("dice", 151*time.Millisecond) {
		t.Fatal("different cue was rejected")
	}
	gate.Reset()
	if !gate.Allow("tick", 1*time.Millisecond) {
		t.Fatal("reset did not clear cue history")
	}
}
