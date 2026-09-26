package game

import (
	"bytes"
	"testing"
	"time"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

func TestNew_InitialState(t *testing.T) {
	seed := []byte{1, 2, 3}
	s := New(domain.OneShot{ID: "demo"}, seed)
	seed[0] = 9

	v := s.View()
	if v.Path != vocab.StateLobby || v.Paused || v.Version != 0 {
		t.Fatalf("unexpected initial view: %#v", v)
	}
	if len(v.Seats) != 2 || v.Seats[0].Seat != 1 || v.Seats[1].Seat != 2 {
		t.Fatalf("unexpected initial seats: %#v", v.Seats)
	}
	if !bytes.Equal(s.Inspect().Seed, []byte{1, 2, 3}) {
		t.Fatalf("constructor retained caller seed: %v", s.Inspect().Seed)
	}
}

func TestStateStep_RootCommands(t *testing.T) {
	tests := []struct {
		name       string
		cmd        vocab.HostCmd
		path       vocab.StateID
		paused     bool
		effectKind vocab.EffectKind
	}{
		{name: "start", cmd: vocab.HostStart, path: vocab.StateCreation, effectKind: vocab.EffectStartTimer},
		{name: "pause", cmd: vocab.HostPause, path: vocab.StateLobby, paused: true, effectKind: vocab.EffectPauseAll},
		{name: "resume", cmd: vocab.HostResume, path: vocab.StateLobby, effectKind: vocab.EffectResumeAll},
		{name: "reset", cmd: vocab.HostReset, path: vocab.StateLobby, effectKind: vocab.EffectNewRun},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			s := New(domain.OneShot{}, []byte{7})
			if test.name == "resume" {
				s.paused = true
			}
			out := s.Step(domain.Envelope{At: 4 * time.Second, Event: domain.HostCmd{Cmd: test.cmd}})
			if s.View().Path != test.path || s.View().Paused != test.paused {
				t.Fatalf("state = path %q paused %v", s.View().Path, s.View().Paused)
			}
			if len(out.Effects) != 1 || out.Effects[0].Kind() != test.effectKind {
				t.Fatalf("effects = %#v", out.Effects)
			}
			if s.View().At != 4*time.Second || s.View().Version != 1 {
				t.Fatalf("logical clock/version not applied: %#v", s.View())
			}
		})
	}
}

func TestStateStep_RejectionAndReset(t *testing.T) {
	s := New(domain.OneShot{}, []byte{1, 2})
	out := s.Step(domain.Envelope{Event: domain.Act{Seat: 1, Move: vocab.MoveReady}})
	if out.Ack == nil || out.Ack.Accepted || out.Ack.Reason != "unaccepted_event" {
		t.Fatalf("unexpected rejection: %#v", out.Ack)
	}

	newSeed := []byte{8, 9}
	out = s.Step(domain.Envelope{Event: domain.DebugReset{Seed: newSeed}})
	newSeed[0] = 0
	if len(out.Effects) != 1 || out.Effects[0].Kind() != vocab.EffectNewRun {
		t.Fatalf("reset effects = %#v", out.Effects)
	}
	if s.View().Path != vocab.StateLobby || !bytes.Equal(s.Inspect().Seed, []byte{8, 9}) {
		t.Fatalf("reset did not replace state: %#v", s.Inspect())
	}
}

func TestStateLegalMoves_SeatAndPause(t *testing.T) {
	s := New(domain.OneShot{}, nil)
	if got := s.LegalMoves(0); got != nil {
		t.Fatalf("invalid seat moves = %v", got)
	}
	if got := s.LegalMoves(1); len(got) != 1 || got[0] != vocab.MoveReady {
		t.Fatalf("lobby moves = %v", got)
	}
	s.paused = true
	if got := s.LegalMoves(1); got != nil {
		t.Fatalf("paused moves = %v", got)
	}
}
