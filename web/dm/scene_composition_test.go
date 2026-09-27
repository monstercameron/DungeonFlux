package dm

import "testing"

func TestSceneHeroFallback_PreservesIdentityAndModifiers(t *testing.T) {
	SetArtSource(mapArt{"ui/species_elf_female": "blob:elf-female", "ui/species_elf": "blob:elf"})
	t.Cleanup(func() { SetArtSource(nil) })
	if got := heroProxyArtGendered("elf", "female", "rogue", "hero"); got != "blob:elf-female" {
		t.Fatal(got)
	}
	if got := creationHeroStandIn(CreationSeat{Species: "Elf", Gender: "Female"}); got != "blob:elf-female" {
		t.Fatal(got)
	}
	if got := creationHeroStandIn(CreationSeat{PortraitURL: "blob:generated"}); got != "blob:generated" {
		t.Fatal(got)
	}
	for score, want := range map[int32]int32{0: 0, 7: -2, 8: -1, 10: 0, 15: 2, 20: 5} {
		if got := abilityModifier(score); got != want {
			t.Fatalf("score %d modifier %d", score, got)
		}
	}
}

func TestSceneComposition_OneOwnerPerForeground(t *testing.T) {
	for _, tc := range []struct {
		phase string
		want  sceneParts
	}{
		{"exploration", sceneParts{}}, {"conversation", sceneParts{chrome: true}},
		{"opening", sceneParts{true, true, true}}, {"resolution", sceneParts{chrome: true, caption: true}},
	} {
		t.Run(tc.phase, func(t *testing.T) {
			if got := sceneComposition(tc.phase); got != tc.want {
				t.Fatalf("%+v", got)
			}
		})
	}
}
