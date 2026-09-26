package game

import (
	"testing"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

func TestViewForDM_ReturnsLobbyAndSeatCards(t *testing.T) {
	view := lobbyView()
	got := ViewForDM(view)
	if got.Path != vocab.StateLobby || len(got.Seats) != 2 {
		t.Fatalf("dm view = %#v", got)
	}
	if got.Seats[0].Build == nil || got.Seats[0].Build.Name != "Astra" {
		t.Fatalf("dm seat card = %#v", got.Seats[0].Build)
	}
	got.Seats[0].Build.Name = "changed"
	got.Seats[0].Build.Stats.SkillProficiencies["stealth"] = "changed"
	if view.Seats[0].Build.Name != "Astra" {
		t.Fatal("dm view shares build card with source")
	}
	if view.Seats[0].Build.Stats.SkillProficiencies["stealth"] != "expertise" {
		t.Fatal("dm view shares build stats with source")
	}
}

func TestViewForSeat_ProjectsOnlyRequestedSeat(t *testing.T) {
	view := lobbyView()
	got := ViewForSeat(view, 2)
	if len(got.Seats) != 1 || got.Seats[0].Seat != 2 {
		t.Fatalf("seat view = %#v", got.Seats)
	}
	if got.Version != view.Version || got.Path != view.Path || got.Paused != view.Paused {
		t.Fatalf("shared metadata was not retained: %#v", got)
	}
	if missing := ViewForSeat(view, 9).Seats; missing != nil {
		t.Fatalf("unknown seat view = %#v", missing)
	}
}

func TestViewForHost_ReturnsAllSeatsWithoutAliases(t *testing.T) {
	view := lobbyView()
	got := ViewForHost(view)
	if len(got.Seats) != 2 || got.Seats[1].Seat != 2 {
		t.Fatalf("host seats = %#v", got.Seats)
	}
	got.Seats[1].Character.Name = "changed"
	if view.Seats[1].Character.Name != "Bryn" {
		t.Fatal("host view shares character with source")
	}
}

func lobbyView() domain.View {
	return domain.View{
		Version: 4,
		Path:    vocab.StateLobby,
		Paused:  true,
		Seats: []domain.SeatView{
			{Seat: 1, PlayerNumber: 1, Build: &domain.BuildCard{Name: "Astra", Class: "Rogue", Stats: &domain.BuildStats{SkillProficiencies: map[string]string{"stealth": "expertise"}}}, Character: &domain.Character{Name: "Astra"}},
			{Seat: 2, PlayerNumber: 2, Build: &domain.BuildCard{Name: "Bryn", Class: "Paladin"}, Character: &domain.Character{Name: "Bryn"}},
		},
	}
}
