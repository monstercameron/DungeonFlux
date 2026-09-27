package content

import "testing"

func TestDemoShots_ValidateAndContainRequiredIDs(t *testing.T) {
	if err := ValidateDemoShots(); err != nil {
		t.Fatalf("demo shots should validate: %v", err)
	}
	want := []ShotID{ShotEstWidePush, ShotArrivalDoorStatic, ShotBBLoop, ShotCliffGenericTower, ShotCliffTwoPush, ShotNPCMCUStatic, ShotCheckTension, ShotHeroLowPush, ShotCheckFailPull}
	for _, id := range want {
		shot, ok := FindShot(id)
		if !ok || shot.Prompt == "" || shot.NegativePrompt == "" {
			t.Fatalf("shot %q missing prompt data: %#v", id, shot)
		}
	}
}

func TestDemoShots_SettingsAndPromptRules(t *testing.T) {
	tests := []struct {
		id         ShotID
		aspect     string
		resolution string
		seconds    int
		pinned     bool
	}{
		{ShotEstWidePush, "16:9", "720p", 5, false},
		{ShotArrivalDoorStatic, "16:9", "720p", 5, false},
		{ShotCliffGenericTower, "16:9", "720p", 5, true},
		{ShotBBLoop, "9:16", "480p", 4, false},
	}
	for _, tc := range tests {
		t.Run(string(tc.id), func(t *testing.T) {
			shot, ok := FindShot(tc.id)
			if !ok || shot.Aspect != tc.aspect || shot.Resolution != tc.resolution || shot.DurationSeconds != tc.seconds || shot.Pinned != tc.pinned {
				t.Fatalf("unexpected settings: %#v", shot)
			}
			if shot.NegativePrompt != shotNegative {
				t.Fatalf("negative prompt drifted: %q", shot.NegativePrompt)
			}
		})
	}
}

func TestRenderShotPrompt_SubstitutesNameFreeFields(t *testing.T) {
	shot, _ := FindShot(ShotCliffTwoPush)
	got := RenderShotPrompt(shot, "a paladin", "a rogue", "", "")
	for _, want := range []string{"a paladin", "a rogue"} {
		if !contains(got, want) {
			t.Fatalf("rendered prompt lacks %q: %s", want, got)
		}
	}
	for _, placeholder := range []string{"{PC1_LOOK}", "{PC2_LOOK}"} {
		if contains(got, placeholder) {
			t.Fatalf("placeholder %q remains in %s", placeholder, got)
		}
	}
	loop, _ := FindShot(ShotBBLoop)
	got = RenderShotPrompt(loop, "", "", "a thrall", "recoils")
	for _, want := range []string{"a thrall", "recoils"} {
		if !contains(got, want) {
			t.Fatalf("rendered billboard prompt lacks %q: %s", want, got)
		}
	}
}

func TestBillboardAction_Variants(t *testing.T) {
	tests := []struct{ variant, weapon, want string }{
		{"idle", "sword", "breathes slowly and shifts weight"},
		{"attack", "sword", "swings a sword once toward the right"},
		{"hit", "sword", "recoils from a blow"},
		{"fall", "sword", "collapses to the ground"},
		{"unknown", "sword", ""},
	}
	for _, tc := range tests {
		t.Run(tc.variant, func(t *testing.T) {
			if got := BillboardAction(tc.variant, tc.weapon); got != tc.want {
				t.Fatalf("BillboardAction(%q) = %q, want %q", tc.variant, got, tc.want)
			}
		})
	}
}

func TestShotValidate_RejectsMalformedSettings(t *testing.T) {
	tests := []struct {
		name string
		edit func(*Shot)
	}{
		{"identity", func(s *Shot) { s.ID = "" }},
		{"aspect", func(s *Shot) { s.Aspect = "4:3" }},
		{"source", func(s *Shot) { s.Source = "still" }},
		{"video duration", func(s *Shot) { s.DurationSeconds = 0 }},
		{"video resolution", func(s *Shot) { s.Resolution = "" }},
	}
	base, _ := FindShot(ShotEstWidePush)
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			shot := base
			tc.edit(&shot)
			if err := shot.Validate(); err == nil {
				t.Fatal("Validate accepted malformed shot")
			}
		})
	}
}

func contains(value, part string) bool {
	for i := 0; i+len(part) <= len(value); i++ {
		if value[i:i+len(part)] == part {
			return true
		}
	}
	return false
}
