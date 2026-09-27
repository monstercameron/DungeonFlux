package phone

import (
	"testing"

	df "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
)

func TestNewWaitingModel_ProjectsJoinedSeats(t *testing.T) {
	model := NewWaitingModel(SeatView{
		PlayerName: " Astra Vale ", PlayerNumber: 2,
		LobbySeats: []*df.LobbySeat{
			{PlayerNumber: 1, Name: "Bram Stone", Joined: true},
			{PlayerNumber: 2, Name: "Astra Vale", Joined: true, Ready: true},
			{PlayerNumber: 3, Joined: false},
		},
	})
	if model.PlayerName != "Astra Vale" || model.PlayerNumber != 2 || len(model.Joined) != 2 {
		t.Fatalf("model = %+v", model)
	}
	if model.Joined[1].Name != "Astra Vale" || model.Joined[1].Number != 2 {
		t.Fatalf("joined = %+v", model.Joined)
	}
	if !model.Ready || !model.Joined[1].Ready || model.Joined[0].Ready {
		t.Fatalf("readiness must come from this player's server seat: %+v", model)
	}
}

func TestWaitingReadyLabel(t *testing.T) {
	for _, tc := range []struct {
		locale string
		ready  bool
		want   string
	}{
		{"en", false, "Getting ready"}, {"en", true, "Ready for adventure"},
		{"es", false, "Preparándose"}, {"es", true, "Listo para la aventura"},
	} {
		t.Run(tc.want, func(t *testing.T) {
			if got := WaitingReadyLabel(tc.locale, tc.ready); got != tc.want {
				t.Fatalf("label = %q", got)
			}
		})
	}
}

func TestNewWaitingModel_DefaultsMissingIdentity(t *testing.T) {
	model := NewWaitingModel(SeatView{Phone: &df.PhoneView{Character: &df.Character{Name: "Rook"}}})
	if model.PlayerName != "Rook" || model.PlayerNumber != 1 {
		t.Fatalf("model = %+v", model)
	}
	model = NewWaitingModel(SeatView{LobbySeats: []*df.LobbySeat{{PlayerNumber: 0, Joined: true}}})
	if len(model.Joined) != 1 || model.Joined[0].Name != "Player 1" || model.Joined[0].Number != 1 {
		t.Fatalf("fallback seat = %+v", model.Joined)
	}
}

func TestWaitingText_LocalizesAndPluralizes(t *testing.T) {
	cases := []struct {
		name, locale, status, seat, joined string
	}{
		{name: "english", locale: "en", status: "Waiting for the host to start", seat: "Seat 4", joined: "2 players joined"},
		{name: "spanish", locale: "es", status: "Esperando a que el anfitrión comience", seat: "Asiento 4", joined: "1 jugador unido"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := WaitingStatus(tc.locale); got != tc.status {
				t.Fatalf("status = %q", got)
			}
			if got := WaitingSeatLabel(tc.locale, 4); got != tc.seat {
				t.Fatalf("seat = %q", got)
			}
			count := 2
			if tc.name == "spanish" {
				count = 1
			}
			if got := joinedSummary(tc.locale, count); got != tc.joined {
				t.Fatalf("joined = %q", got)
			}
		})
	}
}
