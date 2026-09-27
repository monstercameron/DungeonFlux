package media

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

// BillboardPromptVersion names the prompt template below. It is part of the
// cache key, so any wording change must bump it or stale clips are served.
const BillboardPromptVersion = "bb-green-v1"

// Billboard loop actions (plan §0.17 BB_LOOP_*).
const (
	BillboardIdle   = "idle"
	BillboardAttack = "attack"
	BillboardHit    = "hit"
	BillboardFall   = "fall"
)

// Billboard output settings (plan §0.17: 480p, 9:16, 4 s, no audio).
const (
	BillboardResolution = "480p"
	BillboardAspect     = "9:16"
	BillboardSeconds    = 4
)

// BillboardContactMS is the contact frame used for attack and hit loops that
// were not measured at ingest (plan §0.21.5: live loops take 1.2 s).
const BillboardContactMS = 1200

// BillboardSubject is the character a loop shows. References are identity
// images (front crop first); Look is a short name-free descriptor.
type BillboardSubject struct {
	Look       string
	Weapon     string
	References [][]byte
}

// BillboardSpec is one loop request. The level still conditions the light
// and camera angle only; it never appears in the clip.
type BillboardSpec struct {
	Action     string
	Subject    BillboardSubject
	LevelStill []byte
}

// BillboardActions reports whether action is a known loop action.
func BillboardActions(action string) bool {
	switch action {
	case BillboardIdle, BillboardAttack, BillboardHit, BillboardFall:
		return true
	}
	return false
}

// BillboardCacheKey returns the content address of a loop: sha256 over the
// model ID, prompt version, action, every reference image's sha256 (in
// order), the level still's sha256, and the output resolution, duration and
// aspect. The look and weapon text are not in the key: they are derived from
// the same character as the reference images.
func BillboardCacheKey(model string, spec BillboardSpec) string {
	lines := []string{"model=" + model, "prompt=" + BillboardPromptVersion, "action=" + spec.Action}
	for _, image := range spec.Subject.References {
		lines = append(lines, "ref="+sha256Hex(image))
	}
	lines = append(lines, "level="+sha256Hex(spec.LevelStill),
		"resolution="+BillboardResolution, fmt.Sprintf("seconds=%d", BillboardSeconds), "aspect="+BillboardAspect)
	return sha256Hex([]byte(strings.Join(lines, "\n")))
}

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// BillboardPrompt assembles the Seedance reference-to-video prompt in the
// §0.17 slot order: action, environment, camera, lighting, style,
// constraints. Identity images are @Image1..N; the level still is last.
func BillboardPrompt(spec BillboardSpec) string {
	identity := identityRefs(len(spec.Subject.References))
	level := fmt.Sprintf("@Image%d", len(spec.Subject.References)+1)
	look := strings.TrimSpace(spec.Subject.Look)
	if look == "" {
		look = "character"
	}
	parts := []string{
		fmt.Sprintf("The %s from %s, shown full body from head to feet, %s", look, identity, billboardAction(spec.Action, spec.Subject.Weapon)),
		"Solid flat chroma-key green background (#00B140) everywhere behind the character, no ground, no scenery, no shadows on the background",
		fmt.Sprintf("Static locked-off camera, full-body shot, the character centered with the feet near the bottom of the frame, seen from the same camera height and angle as %s", level),
		fmt.Sprintf("Light the character like %s: the same key light direction, color temperature and softness; %s is a lighting reference only and none of its scenery appears", level, level),
		"Keep the art style, face, costume and colors of @Image1",
		"One continuous shot, no cuts, no transitions. Avoid jitter, warping, extra limbs, flicker and text",
	}
	return strings.Join(parts, ". ") + "."
}

func identityRefs(count int) string {
	if count <= 1 {
		return "@Image1"
	}
	refs := make([]string, count)
	for index := range refs {
		refs[index] = fmt.Sprintf("@Image%d", index+1)
	}
	return strings.Join(refs[:count-1], ", ") + " and " + refs[count-1]
}

func billboardAction(action, weapon string) string {
	weapon = strings.ToLower(strings.TrimSpace(weapon))
	switch action {
	case BillboardAttack:
		if weapon == "" {
			weapon = "weapon"
		}
		if rangedWeapon(weapon) {
			return "draws and looses one shot from a " + weapon + " toward the right side of the frame, then returns to a ready stance"
		}
		return "swings a " + weapon + " once toward the right side of the frame, then returns to a ready stance"
	case BillboardHit:
		return "recoils from a blow that comes from the right, staggers half a step, then recovers to a guarded stance"
	case BillboardFall:
		return "is struck, collapses down to the bottom of the frame and lies still"
	default:
		return "stands in a relaxed combat-ready idle, breathing slowly and shifting weight, ending in the same pose it started in so the clip loops"
	}
}

func rangedWeapon(weapon string) bool {
	return strings.Contains(weapon, "bow")
}

// BillboardWeapon returns the demo class's level-one weapon (the rules
// templates' Attack names), or "" for an unknown class.
func BillboardWeapon(class string) string {
	switch strings.ToLower(strings.TrimSpace(class)) {
	case "barbarian":
		return "greataxe"
	case "bard", "sorcerer", "wizard":
		return "dagger"
	case "cleric":
		return "mace"
	case "druid":
		return "scimitar"
	case "fighter", "paladin":
		return "longsword"
	case "monk":
		return "quarterstaff"
	case "ranger":
		return "longbow"
	case "rogue":
		return "shortsword"
	case "warlock":
		return "light crossbow"
	default:
		return ""
	}
}
