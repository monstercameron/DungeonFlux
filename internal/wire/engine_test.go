package wire

import (
	"testing"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/fakes"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

func TestSynchronizedEngine_DelegatesAndReplaces(t *testing.T) {
	first := &fakes.FakeEngine{ViewValue: domain.View{Path: vocab.StateLobby}}
	engine, err := newSynchronizedEngine(first)
	if err != nil {
		t.Fatal(err)
	}
	if got := engine.View().Path; got != vocab.StateLobby {
		t.Fatalf("initial path = %q", got)
	}
	second := &fakes.FakeEngine{ViewValue: domain.View{Path: vocab.StateOpening}}
	if !engine.replace(second) {
		t.Fatal("replace() = false")
	}
	if got := engine.View().Path; got != vocab.StateOpening {
		t.Fatalf("replaced path = %q", got)
	}
}

func TestNewSynchronizedEngine_RejectsNil(t *testing.T) {
	if _, err := newSynchronizedEngine(nil); err == nil {
		t.Fatal("nil engine accepted")
	}
	engine, err := newSynchronizedEngine(&fakes.FakeEngine{})
	if err != nil {
		t.Fatal(err)
	}
	if engine.replace(nil) {
		t.Fatal("nil replacement accepted")
	}
}
