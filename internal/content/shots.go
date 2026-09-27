package content

import (
	"fmt"
	"strings"
)

const (
	shotStyle       = "Painterly dark-fantasy illustration, visible brushstrokes, lamplight and river fog, muted amber and teal palette, deep soft shadows, non-photorealistic"
	shotConstraints = "One continuous shot, no cuts, no transitions. Preserve the composition, colors, faces and costumes of the first frame. Avoid jitter, warping, extra limbs, flicker and text"
	shotNegative    = "photorealism, live-action footage, camera shake, scene cut, transition, text, subtitles, watermark, distorted faces, extra limbs, morphing"
)

// ShotID identifies a curated camera shot in the demo library.
type ShotID string

const (
	ShotEstWidePush       ShotID = "EST_WIDE_PUSH"
	ShotNPCMCUStatic      ShotID = "NPC_MCU_STATIC"
	ShotCheckTension      ShotID = "CHECK_TENSION"
	ShotHeroLowPush       ShotID = "HERO_LOW_PUSH"
	ShotCheckFailPull     ShotID = "CHECK_FAIL_PULL"
	ShotArrivalDoorStatic ShotID = "ARRIVAL_DOOR_STATIC"
	ShotCliffTwoPush      ShotID = "CLIFF_TWO_PUSH"
	ShotCliffGenericTower ShotID = "CLIFF_GENERIC_TOWER"
	ShotBBLoop            ShotID = "BB_LOOP"
)

// Shot describes the prompt and browser fallback for one curated shot.
type Shot struct {
	ID                ShotID
	Beat              string
	Source            string
	Aspect            string
	Resolution        string
	DurationSeconds   int
	Move              string
	Prompt            string
	NegativePrompt    string
	Fallback          string
	BrowserOnly       bool
	Pinned            bool
	LastFrameRequired bool
}

// DemoShots returns a fresh copy of every shot used by the demo.
func DemoShots() []Shot {
	return []Shot{
		{ID: ShotEstWidePush, Beat: "Opening establishing clip", Source: "video", Aspect: "16:9", Resolution: "720p", DurationSeconds: 5, Move: "slow push-in", Prompt: "Fog drifts across a crowded smugglers' tavern at night; lanterns sway gently and river water laps at the windows. The camera pushes in slowly and steadily toward the bar. Warm lantern key light from the left, cool blue moonlight rim through the windows, drifting volumetric fog. " + shotStyle + ". " + shotConstraints + ".", NegativePrompt: shotNegative, Fallback: "browser layered tavern push", LastFrameRequired: true},
		{ID: ShotNPCMCUStatic, Beat: "Mother Vell speaks", Source: "browser", Aspect: "16:9", Resolution: "", DurationSeconds: 5, Move: "very slow push", Prompt: "Medium close-up of Mother Vell speaking at the tavern bar, very slow push toward her face, lantern flicker and drifting fog. " + shotStyle + ".", NegativePrompt: shotNegative, Fallback: "browser layered NPC close-up", BrowserOnly: true},
		{ID: ShotCheckTension, Beat: "Dice rolling", Source: "browser", Aspect: "16:9", Resolution: "", DurationSeconds: 0, Move: "push", Prompt: "Push toward the roller's hands and dice over the layered tavern scene, with a restrained vignette. " + shotStyle + ".", NegativePrompt: shotNegative, Fallback: "browser dice vignette", BrowserOnly: true},
		{ID: ShotHeroLowPush, Beat: "Check success", Source: "browser", Aspect: "16:9", Resolution: "", DurationSeconds: 0, Move: "push and slight upward drift", Prompt: "Push toward the successful hero with a slight upward drift and warm bloom over the layered tavern scene. " + shotStyle + ".", NegativePrompt: shotNegative, Fallback: "browser success bloom", BrowserOnly: true},
		{ID: ShotCheckFailPull, Beat: "Check failure", Source: "browser", Aspect: "16:9", Resolution: "", DurationSeconds: 0, Move: "slow pull-out", Prompt: "Slow pull-out from the failed check over the layered tavern scene, with twenty percent desaturation. " + shotStyle + ".", NegativePrompt: shotNegative, Fallback: "browser desaturated pull", BrowserOnly: true},
		{ID: ShotArrivalDoorStatic, Beat: "Stranger arrives", Source: "video", Aspect: "16:9", Resolution: "720p", DurationSeconds: 5, Move: "static locked-off wide shot", Prompt: "A hooded courier, dripping river water, steps slowly forward out of the rain through the open tavern door into the lamplight, clutching a sealed letter. Rain falls behind him. Static camera, locked-off wide shot of the doorway. Warm lantern light on his face, cold blue rain light behind. " + shotStyle + ". " + shotConstraints + ".", NegativePrompt: shotNegative, Fallback: "browser courier doorway reveal"},
		{ID: ShotCliffTwoPush, Beat: "Cliffhanger", Source: "video", Aspect: "16:9", Resolution: "480p", DurationSeconds: 5, Move: "slow push-in", Prompt: "{PC1_LOOK} and {PC2_LOOK} stand in a tavern at midnight and glance slightly toward the window as the tower bell swings. The lanterns go out one by one from back to front until only moonlight remains. The camera pushes in slowly toward both of them. Cool blue moonlight rim light, fading warm lantern glow. " + shotStyle + ". " + shotConstraints + ".", NegativePrompt: shotNegative, Fallback: "browser layered cliffhanger push", LastFrameRequired: false},
		{ID: ShotCliffGenericTower, Beat: "Cliffhanger fallback", Source: "video", Aspect: "16:9", Resolution: "720p", DurationSeconds: 5, Move: "slow tilt up", Prompt: "A flooded street leads to a tall old bell tower as fog rolls and the tower bell swings. The camera tilts up slowly from the flooded street to the bell. Window lights go out. Cool blue moonlight, drifting river fog. " + shotStyle + ". " + shotConstraints + ".", NegativePrompt: shotNegative, Fallback: "browser tower crop reveal", Pinned: true, LastFrameRequired: true},
		{ID: ShotBBLoop, Beat: "Combat billboard loop", Source: "video", Aspect: "9:16", Resolution: "480p", DurationSeconds: 4, Move: "static locked-off full-body shot", Prompt: "{LOOK} stands on a flat pure green background and {ACTION}. Static camera, locked-off full-body shot. Flat pure green background, no floor, no shadow. " + shotStyle + ". " + shotConstraints + ".", NegativePrompt: shotNegative, Fallback: "browser FLAT billboard tween"},
	}
}

// BillboardAction returns the authored action for a BB_LOOP variant.
func BillboardAction(variant, weapon string) string {
	switch strings.ToLower(variant) {
	case "idle":
		return "breathes slowly and shifts weight"
	case "attack":
		return "swings a " + strings.TrimSpace(weapon) + " once toward the right"
	case "hit":
		return "recoils from a blow"
	case "fall":
		return "collapses to the ground"
	default:
		return ""
	}
}

// RenderShotPrompt substitutes the name-free look and action placeholders.
func RenderShotPrompt(shot Shot, pc1Look, pc2Look, look, action string) string {
	text := strings.ReplaceAll(shot.Prompt, "{PC1_LOOK}", strings.TrimSpace(pc1Look))
	text = strings.ReplaceAll(text, "{PC2_LOOK}", strings.TrimSpace(pc2Look))
	text = strings.ReplaceAll(text, "{LOOK}", strings.TrimSpace(look))
	return strings.ReplaceAll(text, "{ACTION}", strings.TrimSpace(action))
}

// Validate checks that a shot contains the settings required by its source.
func (s Shot) Validate() error {
	if s.ID == "" || s.Beat == "" || s.Prompt == "" || s.Fallback == "" {
		return fmt.Errorf("shot %q is missing identity, prompt, or fallback", s.ID)
	}
	if s.Aspect != "16:9" && s.Aspect != "9:16" {
		return fmt.Errorf("shot %q has invalid aspect %q", s.ID, s.Aspect)
	}
	if s.Source == "video" && (s.Resolution == "" || s.DurationSeconds <= 0) {
		return fmt.Errorf("video shot %q needs resolution and duration", s.ID)
	}
	if s.Source != "browser" && s.Source != "video" {
		return fmt.Errorf("shot %q has invalid source %q", s.ID, s.Source)
	}
	return nil
}

// ValidateDemoShots validates the complete curated demo catalogue.
func ValidateDemoShots() error {
	seen := make(map[ShotID]bool)
	for _, shot := range DemoShots() {
		if seen[shot.ID] {
			return fmt.Errorf("duplicate shot %q", shot.ID)
		}
		seen[shot.ID] = true
		if err := shot.Validate(); err != nil {
			return err
		}
	}
	return nil
}

// FindShot returns the named demo shot, or false when it is not in the library.
func FindShot(id ShotID) (Shot, bool) {
	for _, shot := range DemoShots() {
		if shot.ID == id {
			return shot, true
		}
	}
	return Shot{}, false
}
