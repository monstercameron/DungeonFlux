package content

import "testing"

func TestFallbackHeroName_DeterministicForSameSeed(t *testing.T) {
	first := FallbackHeroName("elf", "female", []byte{1, 2, 3})
	second := FallbackHeroName("elf", "female", []byte{1, 2, 3})
	if first != second {
		t.Fatalf("FallbackHeroName is not deterministic: %q vs %q", first, second)
	}
	if first == "" {
		t.Fatal("FallbackHeroName returned an empty name")
	}
}

func TestFallbackHeroName_DifferentSeedsCanDiffer(t *testing.T) {
	seen := make(map[string]bool)
	for seed := byte(0); seed < 20; seed++ {
		seen[FallbackHeroName("human", "male", []byte{seed})] = true
	}
	if len(seen) < 2 {
		t.Fatalf("expected multiple distinct names across seeds, got %v", seen)
	}
}

func TestFallbackHeroName_CoversEverySpeciesAndGender(t *testing.T) {
	species := []string{"human", "elf", "dwarf", "halfling", "orc", "tiefling", "dragonborn", "gnome", "goliath"}
	genders := []string{"female", "male", "nonbinary"}
	for _, s := range species {
		for _, g := range genders {
			name := FallbackHeroName(s, g, []byte(s+g))
			if name == "" {
				t.Errorf("FallbackHeroName(%q, %q, ...) is empty", s, g)
			}
		}
	}
}

func TestFallbackHeroName_UnknownSpeciesOrGenderUsesGenericFallback(t *testing.T) {
	name := FallbackHeroName("beholder", "unknown", []byte("seed"))
	found := false
	for _, candidate := range heroNameFallback {
		if candidate == name {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("FallbackHeroName(unknown) = %q, want one of the generic fallback names", name)
	}
}

func TestFallbackHeroName_NeverReturnsABareSeatLabel(t *testing.T) {
	for seed := byte(0); seed < 10; seed++ {
		name := FallbackHeroName("human", "nonbinary", []byte{seed})
		if name == "Hero 1" || name == "Hero 2" {
			t.Fatalf("FallbackHeroName returned a bare seat label: %q", name)
		}
	}
}
