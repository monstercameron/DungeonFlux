package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

// SFXAsset describes one generated sound effect and its post-processing target.
type SFXAsset struct {
	ID     string
	Prompt string
	LUFS   int
}

// SFXAssets returns the complete demo sound-effect library.
func SFXAssets() []SFXAsset {
	return []SFXAsset{
		{ID: "sfx_dice_roll", Prompt: "tight fantasy dice rolling across a wooden table, three quick impacts", LUFS: -16},
		{ID: "sfx_check_success", Prompt: "brief bright magical success chime, warm and understated", LUFS: -16},
		{ID: "sfx_check_failure", Prompt: "brief muted ominous failure sting, low bell and soft scrape", LUFS: -16},
		{ID: "sfx_door_burst", Prompt: "a heavy tavern door bursts open with a wet wind gust and wood impact", LUFS: -16},
		{ID: "sfx_tavern_ambience", Prompt: "seamless rain-soaked tavern ambience loop, distant room murmur and hearth, no music", LUFS: -20},
		{ID: "sfx_stranger_sting", Prompt: "short ominous stranger arrival sting, bowed metal shimmer and a distant bell", LUFS: -16},
		{ID: "sfx_cliffhanger_hit", Prompt: "one massive tower bell strike in D with a long ringing decay and low impact", LUFS: -16},
		{ID: "sfx_sword_slash", Prompt: "fast clean sword slash through wet air, fantasy combat", LUFS: -16},
		{ID: "sfx_blade_hit", Prompt: "sharp blade hit on a drowned corpse, wet impact", LUFS: -16},
		{ID: "sfx_mace_thud", Prompt: "heavy mace thud against a waterlogged body, deep impact", LUFS: -16},
		{ID: "sfx_dagger_stab", Prompt: "quick dagger stab with a restrained wet impact", LUFS: -16},
		{ID: "sfx_miss_whoosh", Prompt: "fast weapon miss whoosh passing close to camera", LUFS: -16},
		{ID: "sfx_slam_impact", Prompt: "drowned thrall slam impact on tavern floor, wood and water", LUFS: -16},
		{ID: "sfx_thrall_groan", Prompt: "short low drowned corpse groan, no words", LUFS: -16},
		{ID: "sfx_splash_collapse", Prompt: "waterlogged corpse collapsing into a dark splash and muddy slosh", LUFS: -16},
		{ID: "sfx_wet_footsteps", Prompt: "two wet heavy footsteps on a wooden tavern floor, close and rhythmic", LUFS: -16},
	}
}

// SFXRequest is the ElevenLabs sound-generation request body.
type SFXRequest struct {
	Text            string  `json:"text"`
	DurationSeconds float64 `json:"duration_seconds,omitempty"`
	PromptInfluence float64 `json:"prompt_influence,omitempty"`
}

// BuildSFXRequest encodes an SFX prompt for ElevenLabs.
func BuildSFXRequest(asset SFXAsset) ([]byte, error) {
	if asset.ID == "" || asset.Prompt == "" {
		return nil, errors.New("buildtime: SFX requires id and prompt")
	}
	if asset.LUFS >= 0 {
		return nil, errors.New("buildtime: SFX loudness target must be negative")
	}
	return json.Marshal(SFXRequest{Text: asset.Prompt, PromptInfluence: 0.3})
}

// RenderSFX requests one effect, stores it, and records its loudness target.
func RenderSFX(ctx context.Context, client *http.Client, endpoint, outputDir string, writer *ManifestWriter, asset SFXAsset, take int) error {
	if client == nil || writer == nil {
		return errors.New("buildtime: SFX renderer requires client and writer")
	}
	body, err := BuildSFXRequest(asset)
	if err != nil {
		return err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint+"/sound-generation", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("buildtime: create SFX request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")
	if key := os.Getenv("DF_ELEVENLABS_API_KEY"); key != "" {
		request.Header.Set("xi-api-key", key)
	}
	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("buildtime: request SFX %q: %w", asset.ID, err)
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("buildtime: SFX %q returned %s", asset.ID, response.Status)
	}
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return fmt.Errorf("buildtime: create SFX output: %w", err)
	}
	tmp, err := os.CreateTemp(outputDir, ".sfx-*.mp3")
	if err != nil {
		return fmt.Errorf("buildtime: create SFX temporary file: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := io.Copy(tmp, response.Body); err != nil {
		tmp.Close()
		return fmt.Errorf("buildtime: save SFX %q: %w", asset.ID, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("buildtime: close SFX %q: %w", asset.ID, err)
	}
	if _, err := writer.AddFile(asset.ID, "SFX", tmpName, take); err != nil {
		return err
	}
	return writer.SetMetadata(asset.ID, 0, 0, map[string]string{
		"target_lufs": fmt.Sprintf("%d", asset.LUFS),
		"postprocess": "ffmpeg loudnorm",
	})
}

// SFXJob returns a job that renders the complete effect library.
func SFXJob(client *http.Client, endpoint, outputDir string, take int) Job {
	return Job{Name: "sound-effects", Run: func(ctx context.Context, writer *ManifestWriter) error {
		for _, asset := range SFXAssets() {
			if err := RenderSFX(ctx, client, endpoint, filepath.Clean(outputDir), writer, asset, take); err != nil {
				return err
			}
		}
		return nil
	}}
}
