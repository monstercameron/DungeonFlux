package dm

import (
	"testing"

	dungeonfluxv1 "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
	"github.com/monstercameron/DungeonFlux/web/shell/audio"
)

func TestNewLobbyModel_InitialisesTwoSeats(t *testing.T) {
	model := NewLobbyModel("ABCD", "https://table.test/join/ABCD")
	if model.RoomCode != "ABCD" || model.QRURL == "" || model.JoinURL != "/p?room=ABCD" {
		t.Fatalf("model identity = %+v", model)
	}
	if model.Seats[0].Number != 1 || model.Seats[1].Number != 2 {
		t.Fatalf("seats = %+v", model.Seats)
	}
	if model.AudioState == "" {
		t.Fatal("audio state is empty")
	}
}

func TestNewLobbyModel_DefaultsQRAssetAndNormalizesRoom(t *testing.T) {
	model := NewLobbyModel(" ab cd ", "")
	if model.RoomCode != "AB CD" || model.QRURL != "/assets/join-room.png" || model.JoinURL != "/p?room=AB+CD" {
		t.Fatalf("normalized lobby = %+v", model)
	}
}

func TestJoinURL_EmptyRoomOmitsEmptyQuery(t *testing.T) {
	if got := JoinURL(" "); got != "/p" {
		t.Fatalf("empty join URL = %q", got)
	}
}

func TestLobbyModel_SetSeat_IgnoresInvalidAndUpdatesValid(t *testing.T) {
	model := NewLobbyModel("", "")
	model.SetSeat(Seat{Number: 0, Name: "bad"})
	model.SetSeat(Seat{Number: 3, Name: "bad"})
	model.SetSeat(Seat{Number: 2, Name: "Rook", Joined: true, Ready: true})
	if model.Seats[0].Name != "" || model.Seats[1].Name != "Rook" || !model.Seats[1].Ready {
		t.Fatalf("seats = %+v", model.Seats)
	}
}

func TestSeatLabel_UsesNameOrPlayerNumber(t *testing.T) {
	if got := SeatLabel(Seat{Number: 1}); got != "Player 1" {
		t.Fatalf("empty label = %q", got)
	}
	if got := SeatLabel(Seat{Number: 2, Name: "Mara"}); got != "Mara" {
		t.Fatalf("named label = %q", got)
	}
}

func TestListenAudio_SchedulesFrameAndCancels(t *testing.T) {
	player := &fakePCMPlayer{}
	listen := NewListenAudio(player)
	frame := &dungeonfluxv1.AudioFrame{UtteranceId: "opening", SampleRate: 8000, PcmS16Le: []byte{1, 0, 2, 0}}
	if err := listen.Handle(&dungeonfluxv1.AudioMessage{Message: &dungeonfluxv1.AudioMessage_Frame{Frame: frame}}); err != nil {
		t.Fatal(err)
	}
	if len(player.chunks) != 1 || player.chunks[0].UtteranceID != "opening" {
		t.Fatalf("chunks = %+v", player.chunks)
	}
	if err := listen.Handle(&dungeonfluxv1.AudioMessage{Message: &dungeonfluxv1.AudioMessage_Cancel{Cancel: &dungeonfluxv1.AudioCancel{All: true}}}); err != nil {
		t.Fatal(err)
	}
}

func TestListenAudio_RejectsMissingPlayerAndMessage(t *testing.T) {
	if err := NewListenAudio(nil).Handle(nil); err == nil {
		t.Fatal("expected missing player error")
	}
	listen := NewListenAudio(&fakePCMPlayer{})
	if err := listen.Handle(nil); err == nil {
		t.Fatal("expected missing message error")
	}
}

type fakePCMPlayer struct{ chunks []audio.ScheduledChunk }

func (f *fakePCMPlayer) Play(chunk audio.ScheduledChunk) error {
	f.chunks = append(f.chunks, chunk)
	return nil
}
