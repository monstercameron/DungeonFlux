package content

import (
	"encoding/json"
	"fmt"
)

// WorldBible is the configurable setting book the AI DM reads before a run:
// where the story happens, what it is about, how it must look and feel, the
// secrets it must not reveal early, and the STT keyterms that protect names.
// A new story is a new WorldBible value, not new code; the Drowned Lantern
// below is only the default instance.
type WorldBible struct {
	ID        string   `json:"id"`
	Setting   string   `json:"setting"`
	Premise   string   `json:"premise"`
	StyleLock string   `json:"style_lock"`
	Mood      string   `json:"mood"`
	Keyterms  []string `json:"keyterms"`
	Secrets   []Secret `json:"secrets"`
}

// Secret is one fact the DM must withhold until its gate opens. GatedBehind
// names a funnel condition from plot_thread.go; the clue text enters the
// speaking prompt only when that condition holds.
type Secret struct {
	ID          string `json:"id"`
	Text        string `json:"text"`
	GatedBehind string `json:"gated_behind"`
}

// DefaultWorldBible returns the Drowned Lantern setting book (§0.7 canon).
func DefaultWorldBible() WorldBible {
	return WorldBible{
		ID:        "drowned_lantern",
		Setting:   "The Drowned Lantern, a smugglers' tavern in a flooded river town, on the night the town's lamplighter vanished. The river is rising.",
		Premise:   "Two travellers question the one-eyed barkeep about the missing lamplighter, receive a letter that names one of them, fight the drowned thing that followed the courier, and learn that whoever rings the old bell tower already knows their names.",
		StyleLock: "Painterly dark-fantasy illustration, lamplight and river fog, non-photorealistic.",
		Mood:      "Rain-hammered, conspiratorial, tender underneath: a barkeep who protects her regulars, a town holding its breath at midnight.",
		Keyterms:  []string{"Mother Vell", "the Drowned Lantern", "the bell tower"},
		Secrets: []Secret{
			{
				ID:          "bell_tower_clue",
				Text:        "The lamplighter was dragged toward the old bell tower, and the bell rang at midnight though nobody climbs it anymore.",
				GatedBehind: CondClueGranted,
			},
		},
	}
}

// LoadWorldBible decodes a WorldBible from JSON data, so a new setting ships
// as a data file rather than a code change.
func LoadWorldBible(data []byte) (WorldBible, error) {
	var bible WorldBible
	if err := json.Unmarshal(data, &bible); err != nil {
		return WorldBible{}, fmt.Errorf("decode world bible: %w", err)
	}
	if err := bible.Validate(); err != nil {
		return WorldBible{}, err
	}
	return bible, nil
}

// Validate checks that the setting book is complete and its secret gates name
// known funnel conditions.
func (b WorldBible) Validate() error {
	if b.ID == "" || b.Setting == "" || b.Premise == "" {
		return fmt.Errorf("world bible %q needs id, setting, and premise", b.ID)
	}
	if b.StyleLock == "" || b.Mood == "" {
		return fmt.Errorf("world bible %q needs style lock and mood", b.ID)
	}
	if len(b.Keyterms) == 0 {
		return fmt.Errorf("world bible %q needs STT keyterms", b.ID)
	}
	if len(b.Secrets) == 0 {
		return fmt.Errorf("world bible %q must gate at least one secret", b.ID)
	}
	for _, secret := range b.Secrets {
		if secret.ID == "" || secret.Text == "" {
			return fmt.Errorf("world bible %q holds an empty secret", b.ID)
		}
		if !KnownCondition(secret.GatedBehind) {
			return fmt.Errorf("secret %q gates behind unknown condition %q", secret.ID, secret.GatedBehind)
		}
	}
	return nil
}

// SecretFor returns the secret text the DM may use once its gate condition
// holds, or false when the condition has not opened it yet.
func (b WorldBible) SecretFor(id, condition string) (string, bool) {
	for _, secret := range b.Secrets {
		if secret.ID == id && secret.GatedBehind == condition {
			return secret.Text, true
		}
	}
	return "", false
}
