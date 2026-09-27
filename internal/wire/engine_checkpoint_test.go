package wire

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/fakes"
	"github.com/monstercameron/DungeonFlux/internal/ports"
)

type checkpointTestEngine struct {
	fakes.FakeEngine
	step func(domain.Envelope) domain.StepOut
}

func (e *checkpointTestEngine) Step(env domain.Envelope) domain.StepOut { return e.step(env) }

func TestEngineCheckpoint_ReplaysImmutableEventsAndMonotonicVersions(t *testing.T) {
	var seen []domain.Envelope
	factory := func() ports.Engine {
		seen = nil
		return &checkpointTestEngine{step: func(env domain.Envelope) domain.StepOut {
			seen = append(seen, env)
			return domain.StepOut{Effects: []domain.Effect{domain.GenerateClip{}}}
		}}
	}
	engine, _ := newSynchronizedEngine(factory())
	engine.configureCheckpoints(factory)
	fields := map[string]string{"hp": "12"}
	scope := domain.Scope{Machine: "combat", Epoch: 3, Key: "hero"}
	engine.Step(domain.Envelope{Event: domain.DebugPatch{Target: "pc1", Fields: fields}, At: time.Second, Seq: 9, Scope: scope, Reply: make(chan domain.Ack, 1), RuntimeGeneration: 6})
	first, err := engine.SaveEngineCheckpoint()
	if err != nil {
		t.Fatal(err)
	}
	fields["hp"] = "1"
	engine.Step(domain.Envelope{Event: domain.Join{Seat: 2}})
	before := engine.View().Version
	for range 2 {
		if err := first(); err != nil {
			t.Fatal(err)
		}
		if len(seen) != 1 || seen[0].Event.(domain.DebugPatch).Fields["hp"] != "12" {
			t.Fatalf("saved history was mutated: %#v", seen)
		}
		if seen[0].At != time.Second || seen[0].Seq != 9 || seen[0].Scope != scope || seen[0].Reply != nil || seen[0].RuntimeGeneration != 0 {
			t.Fatalf("replay envelope lost logical context or retained runtime state: %#v", seen[0])
		}
		if engine.View().Version <= before {
			t.Fatal("restoring rolled back publication version")
		}
		before = engine.View().Version
		seen[0].Event.(domain.DebugPatch).Fields["hp"] = "99"
		engine.Step(domain.Envelope{Event: domain.Join{Seat: 1}})
	}
}

func TestEngineCheckpoint_RestoresOriginalRunFactory(t *testing.T) {
	firstFactory := func() ports.Engine { return &fakes.FakeEngine{InspectValue: domain.Inspect{Seed: []byte("first")}} }
	secondFactory := func() ports.Engine { return &fakes.FakeEngine{InspectValue: domain.Inspect{Seed: []byte("second")}} }
	engine, _ := newSynchronizedEngine(firstFactory())
	engine.configureCheckpoints(firstFactory)
	load, _ := engine.SaveEngineCheckpoint()
	engine.replace(secondFactory())
	engine.configureCheckpoints(secondFactory)
	if err := load(); err != nil || string(engine.Inspect().Seed) != "first" {
		t.Fatalf("restore used replacement run factory: %v", err)
	}
	loadAgain, _ := engine.SaveEngineCheckpoint()
	engine.replace(secondFactory())
	if err := loadAgain(); err != nil || string(engine.Inspect().Seed) != "first" {
		t.Fatal("new checkpoint did not inherit restored factory")
	}
}

func TestEngineCheckpoint_RejectsMissingFactoryAndOversizedHistory(t *testing.T) {
	factory := func() ports.Engine { return &fakes.FakeEngine{} }
	engine, _ := newSynchronizedEngine(factory())
	if _, err := engine.SaveEngineCheckpoint(); err == nil {
		t.Fatal("checkpoint without factory accepted")
	}
	engine.configureCheckpoints(factory)
	engine.Step(domain.Envelope{Event: domain.DebugPatch{Fields: map[string]string{"oversized": strings.Repeat("x", checkpointHistoryLimit)}}})
	if _, err := engine.SaveEngineCheckpoint(); err == nil || !strings.Contains(err.Error(), "limit") {
		t.Fatalf("oversized history not rejected: %v", err)
	}
	engine.Step(domain.Envelope{Event: domain.Join{}})
	if len(engine.journal.events) != 0 {
		t.Fatal("failed journal continued retaining events")
	}
}

func TestJournalEvent_DecodesValuePointerAndNil(t *testing.T) {
	for _, tc := range []struct {
		name  string
		event domain.Event
	}{{"value", domain.Join{Seat: 1}}, {"pointer", &domain.Join{Seat: 2}}, {"nil", nil}} {
		t.Run(tc.name, func(t *testing.T) {
			event := tc.event
			journal := engineJournal{factory: func() ports.Engine { return nil }}
			journal.append(domain.Envelope{Event: event})
			got, err := journal.events[0].decode()
			if err != nil || !reflect.DeepEqual(got.Event, event) {
				t.Fatalf("decoded event = %#v, %v", got.Event, err)
			}
		})
	}
}

func TestEngineCheckpoint_InvalidRestorePreservesCurrentEngine(t *testing.T) {
	engine, _ := newSynchronizedEngine(&fakes.FakeEngine{ViewValue: domain.View{Path: "original"}})
	for _, tc := range []struct {
		name    string
		journal engineJournal
	}{
		{"nil factory result", engineJournal{factory: func() ports.Engine { return nil }}},
		{"corrupt event", engineJournal{events: []journalEvent{{typeOf: reflect.TypeOf(domain.Join{}), data: []byte("{")}}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := engine.restoreJournal(tc.journal); err == nil || engine.View().Path != "original" {
				t.Fatal("invalid restore changed the current engine")
			}
		})
	}
}
