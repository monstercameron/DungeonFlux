package game

import (
	"bytes"
	"reflect"
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

func TestStateStep_JoinRetainsSeatAcrossPhases(t *testing.T) {
	s := New(domain.OneShot{}, []byte{1})
	out := s.Step(domain.Envelope{Event: domain.Join{Seat: 1, JoinKind: "phone", Locale: "es", Name: "Lyra"}})
	if out.Ack != nil {
		t.Fatalf("join without reply should not allocate an ack: %#v", out.Ack)
	}
	joined := s.View().Seats[0]
	if !joined.Connected || joined.Locale != "es" || joined.PlayerName != "Lyra" {
		t.Fatalf("joined seat = %#v", joined)
	}
	if got := s.Lobby().Seats[0].Name; got != "Lyra" {
		t.Fatalf("lobby player name = %q, want Lyra", got)
	}
	s.Step(domain.Envelope{Event: domain.HostCmd{Cmd: vocab.HostStart}})
	if got := s.View().Seats[0]; !got.Connected || got.Locale != "es" || got.PlayerName != "Lyra" {
		t.Fatalf("seat metadata was lost after phase transition: %#v", got)
	}
}

func TestStateStep_JoinEmptyNameUsesCharacterFallback(t *testing.T) {
	s := New(domain.OneShot{}, []byte("name"))
	s.Step(domain.Envelope{Event: domain.Join{Seat: 1}})
	startCreation(t, s, 1)
	character := s.View().Seats[0].Character
	if character == nil || character.Name != "Hero 1" {
		t.Fatalf("character = %#v, want Hero 1", character)
	}
}

func TestView_HeroNameWinsAfterCreation(t *testing.T) {
	s := New(domain.OneShot{}, []byte("name"))
	s.Step(domain.Envelope{Event: domain.Join{Seat: 1, Name: "Aria"}})
	startCreation(t, s, 1)
	seat := s.View().Seats[0]
	if seat.PlayerName != "Aria" || seat.Character == nil || seat.Character.Name != "Hero 1" || seat.Build.Name != "Hero 1" {
		t.Fatalf("seat identity = %#v, want player Aria and hero Hero 1", seat)
	}
}

func TestStateStep_RejoinRenamesPlayer(t *testing.T) {
	s := New(domain.OneShot{}, nil)
	s.Step(domain.Envelope{Event: domain.Join{Seat: 2, Name: "Lyra"}})
	s.Step(domain.Envelope{Event: domain.Join{Seat: 2, Name: "Brom"}})
	seat := s.View().Seats[1]
	if seat.PlayerName != "Brom" || s.Lobby().Seats[1].Name != "Brom" {
		t.Fatalf("renamed seat = %#v, lobby = %#v", seat, s.Lobby().Seats[1])
	}
}

func startCreation(t *testing.T, state *State, seat domain.SeatID) {
	t.Helper()
	state.Step(domain.Envelope{Event: domain.HostCmd{Cmd: vocab.HostStart}})
	for _, event := range []domain.Event{
		domain.Act{Seat: seat, Move: vocab.MoveSpecies, Arg: "human"},
		domain.Act{Seat: seat, Move: vocab.MoveGender, Arg: "female"},
		domain.Act{Seat: seat, Move: vocab.MoveClass, Arg: "paladin"},
		domain.Act{Seat: seat, Move: vocab.MoveRollHero},
	} {
		if out := state.Step(domain.Envelope{Event: event}); out.Ack != nil && !out.Ack.Accepted {
			t.Fatalf("creation event %T rejected: %#v", event, out.Ack)
		}
	}
}

func TestStateStep_JoinRejectsUnknownSeat(t *testing.T) {
	s := New(domain.OneShot{}, nil)
	out := s.Step(domain.Envelope{Event: domain.Join{Seat: 3}})
	if out.Ack == nil || out.Ack.Accepted || out.Ack.Reason != "invalid_seat" {
		t.Fatalf("unknown seat ack = %#v", out.Ack)
	}
}

func TestNew_WithLobbyRetainsDetachedMetadata(t *testing.T) {
	lobby := Lobby{RoomCode: "AB12", JoinURL: "https://dm.test/p?room=AB12", QRAsset: "qr-asset"}
	s := New(domain.OneShot{}, nil, WithLobby(lobby))
	got := s.Lobby()
	if got.RoomCode != lobby.RoomCode || got.JoinURL != lobby.JoinURL || got.QRAsset != lobby.QRAsset {
		t.Fatalf("lobby = %#v", got)
	}
	got.Seats[0].Joined = true
	if s.Lobby().Seats[0].Joined {
		t.Fatal("lobby seats share backing storage")
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
			wantEffects := 1
			if test.name == "start" {
				wantEffects = 4
			}
			if len(out.Effects) != wantEffects || out.Effects[0].Kind() != test.effectKind {
				t.Fatalf("effects = %#v", out.Effects)
			}
			if s.View().At != 4*time.Second || s.View().Version != 1 {
				t.Fatalf("logical clock/version not applied: %#v", s.View())
			}
		})
	}
}

func TestStateStep_PhaseTransitionEmitsCueOnce(t *testing.T) {
	state := New(domain.OneShot{}, []byte{7})
	commands := []vocab.HostCmd{vocab.HostStart}
	for range 9 {
		commands = append(commands, vocab.HostSkip)
	}
	for _, command := range commands {
		out := state.Step(domain.Envelope{Event: domain.HostCmd{Cmd: command}})
		want := CueForState(state.View().Path).Effects()
		if command == vocab.HostStart {
			want = append(want, UIAudioEffects(domain.HostCmd{Cmd: command})...)
		}
		if got := playSounds(out.Effects); !reflect.DeepEqual(got, want) {
			t.Fatalf("%s cue = %#v, want %#v", state.View().Path, got, want)
		}
	}

	out := state.Step(domain.Envelope{Event: domain.HostCmd{Cmd: vocab.HostPause}})
	if got := countPlaySound(out.Effects); got != 0 {
		t.Fatalf("pause self-transition cue play-sound effects = %d, want 0", got)
	}
}

func TestStateStep_ResetTransitionEmitsLobbyCue(t *testing.T) {
	state := New(domain.OneShot{}, []byte{7})
	state.Step(domain.Envelope{Event: domain.HostCmd{Cmd: vocab.HostStart}})
	out := state.Step(domain.Envelope{Event: domain.HostCmd{Cmd: vocab.HostReset}})
	if state.View().Path != vocab.StateLobby || countPlaySound(out.Effects) != 2 {
		t.Fatalf("reset transition path/effects = %q/%#v", state.View().Path, out.Effects)
	}
}

func countPlaySound(effects []domain.Effect) int {
	return len(playSounds(effects))
}

func playSounds(effects []domain.Effect) []domain.Effect {
	plays := make([]domain.Effect, 0, len(effects))
	for _, effect := range effects {
		if _, ok := effect.(domain.PlaySound); ok {
			plays = append(plays, effect)
		}
	}
	return plays
}

func TestStateStep_RejectionAndReset(t *testing.T) {
	s := New(domain.OneShot{}, []byte{1, 2})
	out := s.Step(domain.Envelope{Event: domain.Act{Seat: 1, Move: vocab.MoveReady}})
	if out.Ack == nil || out.Ack.Accepted || out.Ack.Reason != "unaccepted_event" {
		t.Fatalf("unexpected rejection: %#v", out.Ack)
	}

	newSeed := []byte{8, 9}
	s = NewWithDebug(domain.OneShot{}, []byte{1, 2}, true, "")
	out = s.Step(domain.Envelope{Event: domain.DebugReset{Seed: newSeed}})
	newSeed[0] = 0
	if len(out.Effects) != 1 || out.Effects[0].Kind() != vocab.EffectNewRun {
		t.Fatalf("reset effects = %#v", out.Effects)
	}
	if s.View().Path != vocab.StateLobby || !bytes.Equal(s.Inspect().Seed, []byte{8, 9}) {
		t.Fatalf("reset did not replace state: %#v", s.Inspect())
	}
}

func TestNewWithDebug_StartAndReset(t *testing.T) {
	s := NewWithDebug(domain.OneShot{}, []byte{1}, true, string(vocab.StateCombat))
	if got := s.View().Path; got != vocab.StateCombat {
		t.Fatalf("debug start path = %q, want %q", got, vocab.StateCombat)
	}
	out := s.Step(domain.Envelope{Event: domain.DebugReset{Seed: []byte{2}}})
	if out.Ack != nil || s.View().Path != vocab.StateCombat || !bytes.Equal(s.Inspect().Seed, []byte{2}) {
		t.Fatalf("debug reset = ack %v, path %q, seed %v", out.Ack, s.View().Path, s.Inspect().Seed)
	}
}

func TestDebugReset_RequiresDebug(t *testing.T) {
	s := New(domain.OneShot{}, []byte{1})
	out := s.Step(domain.Envelope{Event: domain.DebugReset{Seed: []byte{2}}})
	if out.Ack == nil || out.Ack.Accepted || out.Ack.Reason != "unaccepted_event" {
		t.Fatalf("debug reset ack = %#v", out.Ack)
	}
}

func TestHostForceD20_ValidatesAndStoresNextRoll(t *testing.T) {
	tests := []struct {
		name  string
		value int
		ok    bool
	}{
		{name: "low", value: 0},
		{name: "high", value: 21},
		{name: "valid", value: 17, ok: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			s := New(domain.OneShot{}, nil)
			out := s.Step(domain.Envelope{Event: domain.HostCmd{Cmd: vocab.HostForceD20, N: test.value}})
			if out.Ack == nil || out.Ack.Accepted != test.ok {
				t.Fatalf("ack = %#v, want accepted %v", out.Ack, test.ok)
			}
			if test.ok && s.View().NextD20 != test.value {
				t.Fatalf("next d20 = %d, want %d", s.View().NextD20, test.value)
			}
		})
	}
}

func TestConsumeForcedD20_IsOneShot(t *testing.T) {
	s := New(domain.OneShot{}, nil)
	s.nextD20 = 19
	if got, ok := s.consumeForcedD20(); !ok || got != 19 {
		t.Fatalf("first consume = %d, %v", got, ok)
	}
	if got, ok := s.consumeForcedD20(); ok || got != 0 {
		t.Fatalf("second consume = %d, %v", got, ok)
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

func TestStateStep_DelegatesCreationAndStoryToEnd(t *testing.T) {
	s := New(domain.OneShot{}, []byte("step-seed"))
	steps := []domain.Event{
		domain.HostCmd{Cmd: vocab.HostStart},
		domain.Act{Seat: 1, Move: vocab.MoveSpecies, Arg: "human"},
		domain.Act{Seat: 1, Move: vocab.MoveGender, Arg: "nonbinary"},
		domain.Act{Seat: 1, Move: vocab.MoveClass, Arg: "paladin"},
		domain.Act{Seat: 1, Move: vocab.MoveRollHero},
		domain.Act{Seat: 1, Move: vocab.MoveReady},
		domain.Act{Seat: 2, Move: vocab.MoveSpecies, Arg: "elf"},
		domain.Act{Seat: 2, Move: vocab.MoveGender, Arg: "female"},
		domain.Act{Seat: 2, Move: vocab.MoveClass, Arg: "rogue"},
		domain.Act{Seat: 2, Move: vocab.MoveRollHero},
		domain.Act{Seat: 2, Move: vocab.MoveReady},
		domain.LineDone{UtteranceID: "opening"},
		domain.Act{Seat: 1, Move: vocab.MoveTalkVell},
		domain.Act{Seat: 1, Move: vocab.MovePersuade},
		domain.TimerFired{Name: "roll_resolved"},
		domain.LineDone{UtteranceID: "reveal"},
		domain.Act{Seat: 1, Move: vocab.MoveLeave},
		domain.LineDone{UtteranceID: "stranger"},
		domain.HostCmd{Cmd: vocab.HostSkip},
		domain.LineDone{UtteranceID: "cliffhanger"},
	}
	want := []vocab.StateID{vocab.StateCreation, vocab.StateCreation, vocab.StateCreation, vocab.StateCreation, vocab.StateCreation, vocab.StateCreation, vocab.StateCreation, vocab.StateCreation, vocab.StateCreation, vocab.StateCreation, vocab.StateOpening, vocab.StateExploration, vocab.StateConversation, vocab.StateCheck, vocab.StateResolution, vocab.StateExploration, vocab.StateHookEvent, vocab.StateCombat, vocab.StateCliffhanger, vocab.StateEnd}
	for index, event := range steps {
		s.Step(domain.Envelope{Event: event})
		if got := s.View().Path; got != want[index] {
			t.Fatalf("step %d (%T) path = %q, want %q", index, event, got, want[index])
		}
	}
	if s.View().Seats[0].Character == nil || s.View().Seats[1].Character == nil {
		t.Fatal("creation characters were not retained by the root view")
	}
}
