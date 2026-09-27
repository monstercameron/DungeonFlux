package api

import (
	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
	"testing"
)

func TestProjectCreation_StatsAndReadinessAgreeAcrossClients(t *testing.T) {
	view := domain.View{Path: vocab.StateCreation}
	for n := 1; n <= 2; n++ {
		view.Seats = append(view.Seats, domain.SeatView{
			Seat: domain.SeatID(n), PlayerNumber: n,
			Character: &domain.Character{Name: "Hero", Species: "elf", Gender: "female", Class: "rogue"},
			Build:     &domain.BuildCard{PlayerNumber: n, Stats: &domain.BuildStats{Abilities: [6]int{12, 16, 14, 10, 13, 15}, HP: 10, MaxHP: 10, AC: 14}},
			Moves:     []domain.MoveView{{ID: vocab.MoveReady, Enabled: n == 1, Reason: "Your hero is already ready"}},
		})
	}
	dm := ProjectDM(view)
	for i, card := range dm.GetBuildCards() {
		phone := ProjectPhone(view, domain.SeatID(i+1))
		hero := card.GetCharacter()
		if hero.GetSpecies() != "elf" || hero.GetGender() != "female" || hero.GetBuild().GetAbilities()[1] != 16 || hero.GetBuild().GetHp() != phone.GetCharacter().GetBuild().GetHp() {
			t.Fatalf("hero data mismatch: %v", card)
		}
		if card.GetReady() != (i == 1) || dm.GetSeats()[i].GetReady() != card.GetReady() || phone.GetCharacter().GetLocked() != card.GetReady() {
			t.Fatalf("ready state mismatch: %v", card)
		}
		if phone.GetTurnTimer().GetTotalMs() != 0 {
			t.Fatal("invented a timer")
		}
	}
}
