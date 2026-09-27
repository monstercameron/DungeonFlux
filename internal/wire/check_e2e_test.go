package wire

import (
	"context"
	"testing"
	"time"

	df "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
)

func TestE2E_CheckAndResolutionAdvanceWithoutHostSkip(t *testing.T) {
	for _, tc := range []struct {
		name string
		d20  int32
	}{{"success", 17}, {"failure", 1}} {
		t.Run(tc.name, func(t *testing.T) { checkToEnd(t, tc.d20) })
	}
}

func checkToEnd(t *testing.T, d20 int32) {
	t.Helper()
	room := newVoiceRoom(t)
	session := df.NewSessionServiceClient(room.phone)
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()
	watch, err := session.Watch(ctx, &df.WatchRequest{SeatToken: room.seatOne})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = room.debug.DiceForce(room.ctx, &df.DiceForceRequest{Room: room.room, D20: d20}); err != nil {
		t.Fatal(err)
	}
	phoneAct(t, session, room.seatOne, "persuade", "")
	waitSimPhase(t, watch, "check")
	waitSimPhase(t, watch, "resolution")
	waitSimPhase(t, watch, "exploration")
	sendAct(t, room.debug, room.ctx, "2", "leave", "")
	waitSimPhase(t, watch, "hook_event")
	fightToEnd(t, room, watch)
}

func fightToEnd(t *testing.T, room voiceRoom, watch df.SessionService_WatchClient) {
	t.Helper()
	sawCombat, sawCliffhanger := false, false
	for {
		message, err := watch.Recv()
		if err != nil {
			t.Fatalf("waiting for natural ending: %v", err)
		}
		switch message.GetState().GetPhase() {
		case "combat":
			sawCombat = true
			for _, seat := range []string{"1", "2"} {
				if !contains(readLegal(t, room.debug, room.ctx, room.room, seat), "attack") {
					continue
				}
				if _, err := room.debug.DiceForce(room.ctx, &df.DiceForceRequest{Room: room.room, D20: 20}); err != nil {
					t.Fatal(err)
				}
				result, err := room.debug.Act(room.ctx, &df.DebugActRequest{Room: room.room, Seat: seat, MoveId: "attack", TargetId: "thrall"})
				if err != nil || !result.GetAccepted() {
					t.Fatalf("attack: %v %v", result, err)
				}
				break
			}
		case "cliffhanger":
			sawCliffhanger = true
		case "end":
			if !sawCombat || !sawCliffhanger {
				t.Fatal("ending bypassed combat or cliffhanger")
			}
			return
		}
	}
}
