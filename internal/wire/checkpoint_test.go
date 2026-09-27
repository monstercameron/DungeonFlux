package wire

import (
	"context"
	"testing"
	"time"

	df "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
)

func TestE2E_CheckpointRestoresGameWithoutReplacingPlayerStream(t *testing.T) {
	room := newVoiceRoom(t)
	session := df.NewSessionServiceClient(room.phone)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	watch, err := session.Watch(ctx, &df.WatchRequest{SeatToken: room.seatOne})
	if err != nil {
		t.Fatal(err)
	}
	sendDebug(t, room.debug, room.ctx, "host_pause")
	before, err := room.debug.View(room.ctx, &df.ViewRequest{Room: room.room, Seat: "1", View: "phone"})
	if err != nil {
		t.Fatal(err)
	}
	wireCheckpoint(t, room, "save", "conversation")
	for range 2 {
		response, err := room.debug.Send(room.ctx, &df.SendRequest{Room: room.room, Event: "debug_goto", PayloadJson: `{"phase":"combat"}`})
		if err != nil || !response.GetAccepted() {
			t.Fatalf("goto: %v %v", response, err)
		}
		response, err = room.debug.Send(room.ctx, &df.SendRequest{Room: room.room, Event: "debug_patch", PayloadJson: `{"target":"token:pc1","fields":{"hp":"1"}}`})
		if err != nil || !response.GetAccepted() {
			t.Fatalf("patch: %v %v", response, err)
		}
		changed, err := room.debug.View(room.ctx, &df.ViewRequest{Room: room.room, Seat: "1", View: "phone"})
		if err != nil || changed.GetPhase() != "combat" {
			t.Fatalf("changed view: %v %v", changed, err)
		}
		wireCheckpoint(t, room, "load", "conversation")
		restored, err := room.debug.View(room.ctx, &df.ViewRequest{Room: room.room, Seat: "1", View: "phone"})
		if err != nil || restored.GetPhase() != before.GetPhase() || !restored.GetPaused() || restored.GetPhone().GetCharacter().GetBuild().GetHp() != before.GetPhone().GetCharacter().GetBuild().GetHp() || restored.GetVersion() <= changed.GetVersion() {
			t.Fatalf("restore lost state or version: before=%v, restored=%v, err=%v", before, restored, err)
		}
		for {
			message, err := watch.Recv()
			if err != nil {
				t.Fatalf("original player stream disconnected: %v", err)
			}
			if state := message.GetState(); state.GetVersion() >= restored.GetVersion() {
				if state.GetPhase() != before.GetPhase() || !state.GetPaused() {
					t.Fatalf("stream did not receive restored scene: %v", state)
				}
				break
			}
		}
	}
	sendDebug(t, room.debug, room.ctx, "host_resume")
	phoneAct(t, session, room.seatOne, "persuade", "")
	waitSimPhase(t, watch, "check")
}

func wireCheckpoint(t *testing.T, room voiceRoom, operation, name string) {
	t.Helper()
	response, err := room.debug.Snapshot(room.ctx, &df.SnapshotRequest{Room: room.room, Operation: operation, Data: []byte(name)})
	if err != nil || !response.GetOk() {
		t.Fatalf("snapshot %s: %v %v", operation, response, err)
	}
}

func TestCheckpointInbox_MissingRoomReturnsError(t *testing.T) {
	for _, in := range []*roomInbox{nil, {}} {
		if _, err := in.DebugSnapshot(context.Background(), "save", []byte("test")); err == nil {
			t.Fatal("missing room accepted checkpoint")
		}
	}
}
