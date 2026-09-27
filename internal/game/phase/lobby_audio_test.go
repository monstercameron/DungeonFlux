package phase

import (
	"testing"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

func TestMachine_LobbyAudioCuesEmitOnceOnFirstPassiveEvent(t *testing.T) {
	machine, err := NewWithSeed(domain.OneShot{}, []byte("lobby-audio"))
	if err != nil {
		t.Fatal(err)
	}
	first, err := machine.Step(domain.Join{Seat: 1})
	if err != nil || len(first.Effects) != 2 {
		t.Fatalf("first lobby event = %#v, err=%v", first.Effects, err)
	}
	music, musicOK := first.Effects[0].(domain.PlaySound)
	stinger, stingerOK := first.Effects[1].(domain.PlaySound)
	if !musicOK || music.Channel != vocab.SoundMusic || music.Name != lobbyMusicAsset || !music.Loop || !stingerOK || stinger.Name != lobbyStingerAsset || stinger.Loop {
		t.Fatalf("lobby effects = %#v", first.Effects)
	}
	second, err := machine.Step(domain.Join{Seat: 2})
	if err != nil || len(second.Effects) != 0 {
		t.Fatalf("repeated lobby event = %#v, err=%v", second.Effects, err)
	}
}

func TestLobbyAudioEffects_ResetOnlyEmitsStinger(t *testing.T) {
	effects := lobbyAudioEffects(false)
	if len(effects) != 1 {
		t.Fatalf("effects = %#v", effects)
	}
	stinger := effects[0].(domain.PlaySound)
	if stinger.Name != lobbyStingerAsset || stinger.Channel != vocab.SoundSFX {
		t.Fatalf("stinger = %#v", stinger)
	}
}
