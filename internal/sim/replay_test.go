package sim

import (
	"strings"
	"testing"
	"time"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

type replayEngine struct {
	outputs []domain.StepOut
	seen    []domain.Envelope
}

func (e *replayEngine) Step(env domain.Envelope) domain.StepOut {
	e.seen = append(e.seen, env)
	if len(e.outputs) == 0 {
		return domain.StepOut{}
	}
	out := e.outputs[0]
	e.outputs = e.outputs[1:]
	return out
}

func (e *replayEngine) Inspect() domain.Inspect { return domain.Inspect{} }

func TestCheck_ReplaysRecordedEffects(t *testing.T) {
	envelope := domain.Envelope{Seq: 4, At: 2 * time.Second, Event: domain.HostCmd{Cmd: vocab.HostStart}}
	want := domain.StartTimer{Name: "creation_timeout", After: time.Second}
	engine := &replayEngine{outputs: []domain.StepOut{{Effects: []domain.Effect{want}}}}
	log := []ReplayRecord{{Envelope: envelope, Effects: []domain.Effect{want}}}

	if err := Check(engine, log); err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if len(engine.seen) != 1 || engine.seen[0].At != envelope.At {
		t.Fatalf("replay envelope = %#v, want %#v", engine.seen, envelope)
	}
}

func TestCheck_ReportsFirstDifference(t *testing.T) {
	envelope := domain.Envelope{Seq: 9, Event: domain.HostCmd{Cmd: vocab.HostPause}}
	engine := &replayEngine{outputs: []domain.StepOut{{Effects: []domain.Effect{domain.PauseAll{}}}}}
	log := []ReplayRecord{{Envelope: envelope, Effects: []domain.Effect{domain.ResumeAll{}}}}

	err := Check(engine, log)
	if err == nil || !strings.Contains(err.Error(), "event 0 (seq 9)") {
		t.Fatalf("Check() error = %v, want event and sequence", err)
	}
}

func TestDiffs_ReturnsAllMismatchesAndCopiesEffects(t *testing.T) {
	want := domain.StartTimer{Name: "one"}
	got := domain.StartTimer{Name: "two"}
	log := []ReplayRecord{
		{Envelope: domain.Envelope{Seq: 1, Event: domain.HostCmd{}}, Effects: []domain.Effect{want}},
		{Envelope: domain.Envelope{Seq: 2, Event: domain.HostCmd{}}, Effects: nil},
	}
	engine := &replayEngine{outputs: []domain.StepOut{
		{Effects: []domain.Effect{got}},
		{Effects: []domain.Effect{domain.PauseAll{}}},
	}}

	diffs := Diffs(engine, log)
	if len(diffs) != 2 || diffs[0].Index != 0 || diffs[1].Index != 1 {
		t.Fatalf("Diffs() = %#v, want two ordered differences", diffs)
	}
	if len(diffs[0].Want) != 1 || diffs[0].Want[0] != want {
		t.Fatalf("Diffs() want effects = %#v", diffs[0].Want)
	}
	log[0].Effects[0] = domain.StartTimer{Name: "changed"}
	if diffs[0].Want[0] != want {
		t.Fatal("Diffs() did not copy expected effects")
	}
}

func TestCheck_AcceptsEmptyAndNilEffectLists(t *testing.T) {
	engine := &replayEngine{outputs: []domain.StepOut{{Effects: []domain.Effect{}}}}
	log := []ReplayRecord{{Envelope: domain.Envelope{Seq: 1, Event: domain.HostCmd{}}, Effects: nil}}
	if err := Check(engine, log); err != nil {
		t.Fatalf("Check() error = %v, want nil for empty effects", err)
	}
}
