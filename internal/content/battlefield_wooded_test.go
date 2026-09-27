package content

import "testing"

func TestWoodedPathFallback_UsesRegisteredCameraStill(t *testing.T) {
	story := DefaultOneShot()
	field := story.Encounter.Battlefield
	if field.Flat.ImageURL != "level_still_64bb46d5_tactical" {
		t.Fatalf("fallback location=%q", field.Flat.ImageURL)
	}
	if err := story.Validate(); err != nil {
		t.Fatal(err)
	}
	// Perspective is tied to the capture's ground plane, not the old tavern quad.
	if field.Flat.FloorQuadPX[0][0] != 525.44 || field.Flat.FloorQuadPX[2][1] != 1838.64 {
		t.Fatalf("camera registration=%v", field.Flat.FloorQuadPX)
	}
}
