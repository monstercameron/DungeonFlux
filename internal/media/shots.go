package media

import "strings"

// ShotPrompt is the media-facing projection of one curated camera shot.
// Content owns the authored catalogue; this projection keeps media independent
// of the pure content package while preserving the vendor-facing fields.
type ShotPrompt struct {
	ID                string
	Prompt            string
	NegativePrompt    string
	Resolution        string
	DurationSeconds   int
	LastFrameRequired bool
}

const (
	shotStyle       = "Painterly dark-fantasy illustration, visible brushstrokes, lamplight and river fog, muted amber and teal palette, deep soft shadows, non-photorealistic"
	shotConstraints = "One continuous shot, no cuts, no transitions. Preserve the composition, colors, faces and costumes of the first frame. Avoid jitter, warping, extra limbs, flicker and text"
	shotNegative    = "photorealism, live-action footage, camera shake, scene cut, transition, text, subtitles, watermark, distorted faces, extra limbs, morphing"
)

// DemoShotPrompts returns the curated shot prompts used by media generation.
func DemoShotPrompts() []ShotPrompt {
	return []ShotPrompt{
		{ID: "EST_WIDE_PUSH", Resolution: "720p", DurationSeconds: 5, LastFrameRequired: true, Prompt: "Fog drifts across a crowded smugglers' tavern at night; lanterns sway gently and river water laps at the windows. The camera pushes in slowly and steadily toward the bar. Warm lantern key light from the left, cool blue moonlight rim through the windows, drifting volumetric fog. " + shotStyle + ". " + shotConstraints + ".", NegativePrompt: shotNegative},
		{ID: "ARRIVAL_DOOR_STATIC", Resolution: "720p", DurationSeconds: 5, Prompt: "A hooded courier, dripping river water, steps slowly forward out of the rain through the open tavern door into the lamplight, clutching a sealed letter. Rain falls behind him. Static camera, locked-off wide shot of the doorway. Warm lantern light on his face, cold blue rain light behind. " + shotStyle + ". " + shotConstraints + ".", NegativePrompt: shotNegative},
		{ID: "CLIFF_TWO_PUSH", Resolution: "480p", DurationSeconds: 5, Prompt: "{PC1_LOOK} and {PC2_LOOK} stand in a tavern at midnight and glance slightly toward the window as the tower bell swings. The lanterns go out one by one from back to front until only moonlight remains. The camera pushes in slowly toward both of them. Cool blue moonlight rim light, fading warm lantern glow. " + shotStyle + ". " + shotConstraints + ".", NegativePrompt: shotNegative},
		{ID: "CLIFF_GENERIC_TOWER", Resolution: "720p", DurationSeconds: 5, LastFrameRequired: true, Prompt: "A flooded street leads to a tall old bell tower as fog rolls and the tower bell swings. The camera tilts up slowly from the flooded street to the bell. Window lights go out. Cool blue moonlight, drifting river fog. " + shotStyle + ". " + shotConstraints + ".", NegativePrompt: shotNegative},
		{ID: "BB_LOOP", Resolution: "480p", DurationSeconds: 4, Prompt: "{LOOK} stands on a flat pure green background and {ACTION}. Static camera, locked-off full-body shot. Flat pure green background, no floor, no shadow. " + shotStyle + ". " + shotConstraints + ".", NegativePrompt: shotNegative},
	}
}

// SelectShot returns the authored prompt for an engine shot ID.
func SelectShot(id string) (ShotPrompt, bool) {
	for _, shot := range DemoShotPrompts() {
		if shot.ID == id {
			return shot, true
		}
	}
	return ShotPrompt{}, false
}

// PromptForShot returns the rendered prompt and negative prompt for an ID.
// It is intended for video adapters that need only vendor request strings.
func PromptForShot(id, pc1Look, pc2Look, look, action string) (string, string, bool) {
	shot, ok := SelectShot(id)
	if !ok {
		return "", "", false
	}
	return RenderShotPrompt(shot, pc1Look, pc2Look, look, action), shot.NegativePrompt, true
}

// RenderShotPrompt substitutes the name-free look and action placeholders.
func RenderShotPrompt(shot ShotPrompt, pc1Look, pc2Look, look, action string) string {
	text := strings.ReplaceAll(shot.Prompt, "{PC1_LOOK}", strings.TrimSpace(pc1Look))
	text = strings.ReplaceAll(text, "{PC2_LOOK}", strings.TrimSpace(pc2Look))
	text = strings.ReplaceAll(text, "{LOOK}", strings.TrimSpace(look))
	return strings.ReplaceAll(text, "{ACTION}", strings.TrimSpace(action))
}
