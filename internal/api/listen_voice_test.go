package api

import (
	"context"
	"testing"

	"github.com/monstercameron/DungeonFlux/internal/domain"
)

// TestListenHub_MessageReadersSurviveLongLine streams a 30 s line in 100 ms
// frames to the DM and a phone subscribed the way the registered
// AudioService subscribes them (SubscribeTarget, draining Messages() only).
// Before the fix the undrained legacy frame queue hit the 2 s lag limit and
// both streams were closed 2.1 s into the line.
func TestListenHub_MessageReadersSurviveLongLine(t *testing.T) {
	hub := NewListenHub()
	dm := hub.SubscribeTarget(context.Background(), AudioTarget{Kind: TargetDM}, false)
	phone := hub.SubscribeTarget(context.Background(), AudioTarget{Kind: TargetSeat, Seat: 1}, true)
	defer dm.Close()
	defer phone.Close()
	const frames = 300
	for seq := 0; seq < frames; seq++ {
		hub.Frame(domain.AudioFrame{UtteranceID: "u1", Seq: seq, SampleRate: 24000, PCMS16LE: make([]byte, 4800), Final: seq == frames-1})
		select {
		case message, ok := <-dm.Messages():
			if !ok {
				t.Fatalf("DM subscriber dropped after %d frames (%d ms of audio)", seq, seq*100)
			}
			if message.Frame == nil || message.Frame.Seq != seq || message.Frame.Final != (seq == frames-1) {
				t.Fatalf("message %d = %+v, want frame seq %d", seq, message, seq)
			}
		default:
			t.Fatalf("frame %d was not delivered", seq)
		}
	}
	for name, sub := range map[string]*ListenSubscription{"dm": dm, "phone": phone} {
		select {
		case <-sub.Done():
			t.Fatalf("%s subscriber closed during a fully drained line", name)
		default:
		}
	}
}

// TestListenHub_LegacyFrameQueueKeepsLagLimit keeps the Frames() contract for
// Subscribe listeners: frames arrive there, and a stalled reader is dropped.
func TestListenHub_LegacyFrameQueueKeepsLagLimit(t *testing.T) {
	hub := NewListenHub()
	legacy := hub.Subscribe(context.Background())
	hub.Frame(domain.AudioFrame{UtteranceID: "u1", SampleRate: 24000, PCMS16LE: make([]byte, 4800)})
	if got := <-legacy.Frames(); got.UtteranceID != "u1" {
		t.Fatalf("legacy frame = %+v, want u1", got)
	}
	for seq := 0; seq < 25; seq++ {
		hub.Frame(domain.AudioFrame{UtteranceID: "u1", SampleRate: 24000, PCMS16LE: make([]byte, 4800)})
	}
	select {
	case <-legacy.Done():
	default:
		t.Fatal("stalled legacy reader stayed subscribed past the lag limit")
	}
}

// TestListenHub_CancelPurgesQueueAndTellsDM drops an interrupted line's
// queued frames and sends the DM an AudioCancel so it fades what it already
// scheduled; other lines' frames and phones are untouched.
func TestListenHub_CancelPurgesQueueAndTellsDM(t *testing.T) {
	hub := NewListenHub()
	dm := hub.SubscribeTarget(context.Background(), AudioTarget{Kind: TargetDM}, false)
	phone := hub.SubscribeTarget(context.Background(), AudioTarget{Kind: TargetSeat, Seat: 1}, true)
	defer dm.Close()
	defer phone.Close()
	hub.Frame(domain.AudioFrame{UtteranceID: "keep", SampleRate: 24000, PCMS16LE: make([]byte, 4800)})
	hub.Frame(domain.AudioFrame{UtteranceID: "skip", SampleRate: 24000, PCMS16LE: make([]byte, 4800)})
	hub.Cancel("skip")
	var got []string
	for len(dm.Messages()) > 0 {
		message := <-dm.Messages()
		switch {
		case message.Frame != nil:
			got = append(got, "frame:"+string(message.Frame.UtteranceID))
		case message.CancelID != "":
			got = append(got, "cancel:"+message.CancelID)
		}
	}
	if len(got) != 2 || got[0] != "frame:keep" || got[1] != "cancel:skip" {
		t.Fatalf("DM messages = %v, want [frame:keep cancel:skip]", got)
	}
	if len(phone.Messages()) != 0 {
		t.Fatalf("phone received %d voice messages", len(phone.Messages()))
	}
}
