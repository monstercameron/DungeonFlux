package api

import (
	"context"
	"testing"

	"github.com/monstercameron/DungeonFlux/internal/domain"
)

func TestListenHub_NewestSubscriptionReplacesOlder(t *testing.T) {
	hub := NewListenHub()
	first := hub.Subscribe(context.Background())
	second := hub.Subscribe(context.Background())
	frame := domain.AudioFrame{UtteranceID: "u1", SampleRate: 16000, PCMS16LE: []byte{1, 2}}
	hub.Frame(frame)
	if _, ok := <-first.Frames(); ok {
		t.Fatal("older subscription remained open")
	}
	got := <-second.Frames()
	if got.PCMS16LE[0] != 1 {
		t.Fatalf("frame bytes = %v, want copied frame", got.PCMS16LE)
	}
	second.Close()
}

func TestListenHub_DropsSubscriberPastTwoSeconds(t *testing.T) {
	hub := NewListenHub()
	sub := hub.Subscribe(context.Background())
	frame := domain.AudioFrame{UtteranceID: "u1", SampleRate: 1000, PCMS16LE: make([]byte, 2000)}
	hub.Frame(frame)
	hub.Frame(frame)
	hub.Frame(frame)
	if _, ok := <-sub.Frames(); ok {
		t.Fatal("subscriber remained active after lag limit")
	}
}

func TestListenHub_CancelRemovesQueuedUtterance(t *testing.T) {
	hub := NewListenHub()
	sub := hub.Subscribe(context.Background())
	hub.Frame(domain.AudioFrame{UtteranceID: "u1", SampleRate: 16000, PCMS16LE: []byte{1, 2}})
	hub.Cancel("u1")
	select {
	case _, ok := <-sub.Frames():
		if ok {
			t.Fatal("cancel retained an audio frame")
		}
	default:
	}
}

func TestListenSubscription_CloseIsIdempotent(t *testing.T) {
	sub := NewListenHub().Subscribe(context.Background())
	sub.Close()
	sub.Close()
}

func TestListenHub_TargetsPhoneSFXAndDMAllChannels(t *testing.T) {
	hub := NewListenHub()
	dm := hub.Subscribe(context.Background())
	phone := hub.SubscribeTarget(context.Background(), AudioTarget{Kind: TargetSeat, Seat: 2}, true)
	hub.Publish(AudioMessage{Channel: AudioMusic, Target: AudioTarget{Kind: TargetDM}, Chunk: &EncodedAudioChunk{Data: []byte("music")}})
	hub.Publish(AudioMessage{Channel: AudioSFX, Target: AudioTarget{Kind: TargetSeat, Seat: 2}, Chunk: &EncodedAudioChunk{Data: []byte("dice"), Final: true}})
	select {
	case message := <-phone.Messages():
		if message.Channel != AudioSFX || string(message.Chunk.Data) != "dice" {
			t.Fatalf("phone message=%+v", message)
		}
	default:
		t.Fatal("phone did not receive targeted sfx")
	}
	select {
	case message := <-dm.Messages():
		if message.Channel != AudioMusic {
			t.Fatalf("dm message=%+v", message)
		}
	default:
		t.Fatal("dm did not receive music")
	}
	phone.Close()
	dm.Close()
}

func TestListenHub_NonVoiceDropsOldestButVoiceDropsSubscriber(t *testing.T) {
	hub := NewListenHub()
	sub := hub.Subscribe(context.Background())
	for i := 0; i < 64; i++ {
		hub.Publish(AudioMessage{Channel: AudioSFX, Target: AudioTarget{Kind: TargetDM}, Chunk: &EncodedAudioChunk{Seq: uint64(i)}})
	}
	hub.Publish(AudioMessage{Channel: AudioSFX, Target: AudioTarget{Kind: TargetDM}, Chunk: &EncodedAudioChunk{Seq: 99}})
	first := <-sub.Messages()
	if first.Chunk.Seq != 1 {
		t.Fatalf("oldest retained seq=%d, want 1", first.Chunk.Seq)
	}
	voice := domain.AudioFrame{UtteranceID: "voice", SampleRate: 16000, PCMS16LE: []byte{1, 2}}
	for i := 0; i < 64; i++ {
		hub.Publish(AudioMessage{Channel: AudioVoice, Target: AudioTarget{Kind: TargetDM}, Frame: &voice})
	}
	select {
	case <-sub.Done():
	default:
		t.Fatal("full voice queue did not disconnect subscriber")
	}
}
