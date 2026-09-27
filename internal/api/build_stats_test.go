package api

import (
	"testing"

	"github.com/monstercameron/DungeonFlux/internal/domain"
)

func TestProjectPhone_ProjectsCompleteBuildStats(t *testing.T) {
	view := domain.View{Seats: []domain.SeatView{{
		Seat: 1,
		Build: &domain.BuildCard{Stats: &domain.BuildStats{
			Abilities: [6]int{17, 10, 14, 8, 12, 14}, SaveProficiencies: []string{"dex", "int"},
			SkillProficiencies: map[string]string{"stealth": "expertise", "persuasion": "proficient"}, HP: 11, MaxHP: 12, AC: 15,
		}},
		Character: &domain.Character{Name: "Mira", Class: "Rogue", HP: 11, MaxHP: 12, AC: 15},
	}}}
	build := ProjectPhone(view, 1).GetCharacter().GetBuild()
	if build == nil || len(build.GetAbilities()) != 6 || build.GetAbilities()[0] != 17 || build.GetHp() != 11 || build.GetHpMax() != 12 || build.GetAc() != 15 {
		t.Fatalf("build = %#v", build)
	}
	if len(build.GetSaveProfs()) != 2 || build.GetSkillProfs()["stealth"] != "expertise" {
		t.Fatalf("proficiencies = %#v/%#v", build.GetSaveProfs(), build.GetSkillProfs())
	}
	build.Abilities[0] = 1
	build.SkillProfs["stealth"] = "changed"
	second := ProjectPhone(view, 1).GetCharacter().GetBuild()
	if second.GetAbilities()[0] != 17 || second.GetSkillProfs()["stealth"] != "expertise" {
		t.Fatal("projection shares build stats with its input")
	}
	if got := ProjectPhone(view, 2); got.GetCharacter() != nil {
		t.Fatalf("unexpected other-seat character: %v", got.GetCharacter())
	}
}
