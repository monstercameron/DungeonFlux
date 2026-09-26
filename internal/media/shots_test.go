package media

import "testing"

func TestSelectShot_ReturnsCuratedSettings(t *testing.T) {
	tests := []struct {
		id         string
		resolution string
		seconds    int
		pinned     bool
	}{
		{id: "EST_WIDE_PUSH", resolution: "720p", seconds: 5, pinned: true},
		{id: "ARRIVAL_DOOR_STATIC", resolution: "720p", seconds: 5},
		{id: "CLIFF_GENERIC_TOWER", resolution: "720p", seconds: 5, pinned: true},
		{id: "BB_LOOP", resolution: "480p", seconds: 4},
	}
	for _, tc := range tests {
		t.Run(tc.id, func(t *testing.T) {
			shot, ok := SelectShot(tc.id)
			if !ok || shot.ID != tc.id {
				t.Fatalf("SelectShot(%q) = %#v, %v", tc.id, shot, ok)
			}
			if shot.Resolution != tc.resolution || shot.DurationSeconds != tc.seconds || shot.LastFrameRequired != tc.pinned {
				t.Fatalf("settings = %#v", shot)
			}
			if shot.Prompt == "" || shot.NegativePrompt == "" {
				t.Fatalf("unexpected prompt projection = %#v", shot)
			}
		})
	}
}

func TestSelectShot_UnknownID(t *testing.T) {
	if shot, ok := SelectShot("not-a-shot"); ok || shot != (ShotPrompt{}) {
		t.Fatalf("unknown shot = %#v, %v", shot, ok)
	}
}

func TestPromptForShot_RendersPlaceholdersAndNegativePrompt(t *testing.T) {
	prompt, negative, ok := PromptForShot("CLIFF_TWO_PUSH", "a paladin", "a rogue", "", "")
	if !ok || prompt == "" || negative == "" {
		t.Fatalf("prompt selection = %q, %q, %v", prompt, negative, ok)
	}
	for _, value := range []string{"a paladin", "a rogue"} {
		if !containsShotText(prompt, value) {
			t.Fatalf("prompt lacks %q: %s", value, prompt)
		}
	}
	for _, placeholder := range []string{"{PC1_LOOK}", "{PC2_LOOK}"} {
		if containsShotText(prompt, placeholder) {
			t.Fatalf("prompt retains %q: %s", placeholder, prompt)
		}
	}
}

func TestRenderShotPrompt_RendersBillboardAction(t *testing.T) {
	shot, ok := SelectShot("BB_LOOP")
	if !ok {
		t.Fatal("BB_LOOP is missing")
	}
	prompt := RenderShotPrompt(shot, "", "", "a thrall", "recoils from a blow")
	for _, value := range []string{"a thrall", "recoils from a blow"} {
		if !containsShotText(prompt, value) {
			t.Fatalf("prompt lacks %q: %s", value, prompt)
		}
	}
}

func containsShotText(value, part string) bool {
	for index := 0; index+len(part) <= len(value); index++ {
		if value[index:index+len(part)] == part {
			return true
		}
	}
	return false
}
