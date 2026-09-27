package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

const sfxCostPerMinute = 0.12

// SFXAsset describes one generated sound effect and its post-processing target.
type SFXAsset struct {
	ID              string
	Prompt          string
	DurationSeconds float64
	LUFS            int
	Loop            bool
}

// SFXAssets returns the complete demo sound-effect library.
func SFXAssets() []SFXAsset {
	return []SFXAsset{
		{ID: "sfx_join_tv", Prompt: "warm welcoming tavern chime with a soft wooden door creak, no music", DurationSeconds: 0.8, LUFS: -16},
		{ID: "sfx_phone_confirm", Prompt: "soft friendly confirmation tick with a tiny warm chime, no music", DurationSeconds: 0.5, LUFS: -16},
		{ID: "sfx_ready", Prompt: "short confident ready confirmation chime, warm and clear", DurationSeconds: 0.5, LUFS: -16},
		{ID: "sfx_host_start", Prompt: "short dramatic fantasy table-start sting, bright bell and low lift", DurationSeconds: 1.2, LUFS: -16},
		{ID: "sfx_phone_tick", Prompt: "tiny crisp wooden UI tick, quiet and tactile", DurationSeconds: 0.5, LUFS: -19},
		{ID: "sfx_phone_dice", Prompt: "very short soft dice rattle in a player's hand, three tiny taps", DurationSeconds: 0.7, LUFS: -16},
		{ID: "sfx_roll_reveal", Prompt: "short magical hero reveal shimmer with a warm bell resolve", DurationSeconds: 1.1, LUFS: -16},
		{ID: "sfx_hero_lock", Prompt: "brief warm hero locked chime, gentle fantasy bell", DurationSeconds: 0.8, LUFS: -16},
		{ID: "sfx_dice_roll", Prompt: "tight fantasy dice rolling across a wooden table, three quick impacts", DurationSeconds: 2, LUFS: -16},
		{ID: "sfx_check_success", Prompt: "brief bright magical success chime, warm and understated", DurationSeconds: 2, LUFS: -16},
		{ID: "sfx_check_failure", Prompt: "brief muted ominous failure sting, low bell and soft scrape", DurationSeconds: 2, LUFS: -16},
		{ID: "sfx_door_burst", Prompt: "a heavy tavern door bursts open with a wet wind gust and wood impact", DurationSeconds: 2, LUFS: -16},
		{ID: "sfx_tavern_ambience", Prompt: "seamless rain-soaked tavern ambience loop, distant room murmur and hearth, no music", DurationSeconds: 10, LUFS: -20, Loop: true},
		{ID: "sfx_stranger_sting", Prompt: "short ominous stranger arrival sting, bowed metal shimmer and a distant bell", DurationSeconds: 2, LUFS: -16},
		{ID: "sfx_cliffhanger_hit", Prompt: "one massive tower bell strike in D with a long ringing decay and low impact", DurationSeconds: 8, LUFS: -16},
		{ID: "sfx_sword_slash", Prompt: "fast clean sword slash through wet air, fantasy combat", DurationSeconds: 2, LUFS: -16},
		{ID: "sfx_blade_hit", Prompt: "sharp blade hit on a drowned corpse, wet impact", DurationSeconds: 2, LUFS: -16},
		{ID: "sfx_mace_thud", Prompt: "heavy mace thud against a waterlogged body, deep impact", DurationSeconds: 2, LUFS: -16},
		{ID: "sfx_dagger_stab", Prompt: "quick dagger stab with a restrained wet impact", DurationSeconds: 2, LUFS: -16},
		{ID: "sfx_miss_whoosh", Prompt: "fast weapon miss whoosh passing close to camera", DurationSeconds: 2, LUFS: -16},
		{ID: "sfx_slam_impact", Prompt: "drowned thrall slam impact on tavern floor, wood and water", DurationSeconds: 2, LUFS: -16},
		{ID: "sfx_thrall_groan", Prompt: "short low drowned corpse groan, no words", DurationSeconds: 2, LUFS: -16},
		{ID: "sfx_splash_collapse", Prompt: "waterlogged corpse collapsing into a dark splash and muddy slosh", DurationSeconds: 2, LUFS: -16},
		{ID: "sfx_wet_footsteps", Prompt: "two wet heavy footsteps on a wooden tavern floor, close and rhythmic", DurationSeconds: 2, LUFS: -16},
		{ID: "sfx_forest_steps", Prompt: "three quick footsteps through wet fallen leaves and soft mud on a forest path, close, no music", DurationSeconds: 1.5, LUFS: -18},
		{ID: "sfx_thrall_slam", Prompt: "a drowned corpse's heavy fist slams into an armoured adventurer, wet meaty impact with a dull clank of metal, no voice", DurationSeconds: 1.5, LUFS: -16},
		{ID: "sfx_crit_hit", Prompt: "devastating critical weapon strike on a drowned corpse, crunching wet impact with a bright metal ring and a deep low boom", DurationSeconds: 2, LUFS: -15},
		{ID: "sfx_down", Prompt: "a wounded adventurer collapses onto wet leaf litter, armour and gear clattering, one heavy final thud, no voice", DurationSeconds: 2, LUFS: -16},
		{ID: "sfx_your_turn", Prompt: "short warm two-note bell chime calling a player to act, soft and clear, no music", DurationSeconds: 0.8, LUFS: -17},
		{ID: "sfx_phone_hurt", Prompt: "short muffled heavy body blow felt through leather armour, low punchy thump, no voice", DurationSeconds: 0.8, LUFS: -17},
		{ID: "sfx_portrait_ready", Prompt: "soft painterly reveal, a quick brush swish into a gentle harp glissando ending on a warm chime, no music", DurationSeconds: 1.2, LUFS: -18},
		{ID: "sfx_talk_open", Prompt: "soft wooden click with a faint warm breath of air, a quiet listening cue, no music", DurationSeconds: 0.5, LUFS: -20},
	}
}

// SFXRequest is the ElevenLabs sound-generation request body.
type SFXRequest struct {
	Text            string  `json:"text"`
	ModelID         string  `json:"model_id"`
	DurationSeconds float64 `json:"duration_seconds"`
	PromptInfluence float64 `json:"prompt_influence"`
	Loop            bool    `json:"loop,omitempty"`
}

// BuildSFXRequest encodes an SFX prompt for ElevenLabs.
func BuildSFXRequest(asset SFXAsset) ([]byte, error) {
	if asset.ID == "" || asset.Prompt == "" {
		return nil, errors.New("buildtime: SFX requires id and prompt")
	}
	duration := asset.DurationSeconds
	if duration == 0 {
		duration = 2
	}
	if duration < 0.2 || duration > 30 {
		return nil, errors.New("buildtime: SFX duration must be between 0.2 and 30 seconds")
	}
	if asset.LUFS >= 0 {
		return nil, errors.New("buildtime: SFX loudness target must be negative")
	}
	return json.Marshal(SFXRequest{Text: asset.Prompt, ModelID: "eleven_text_to_sound_v2", DurationSeconds: duration, PromptInfluence: 0.3, Loop: asset.Loop})
}

// SFXMediaStats contains the checks measured after normalization.
type SFXMediaStats struct {
	DurationSeconds float64
	IntegratedLUFS  float64
}

// SFXProcessor normalizes one generated audio file and measures its output.
type SFXProcessor interface {
	Normalize(context.Context, string, string, int) (SFXMediaStats, error)
}

// FFmpegSFXProcessor uses ffmpeg and ffprobe for loudness normalization and checks.
type FFmpegSFXProcessor struct{}

// Normalize converts source to normalized MP3 and returns duration and integrated LUFS.
func (FFmpegSFXProcessor) Normalize(ctx context.Context, source, destination string, targetLUFS int) (SFXMediaStats, error) {
	target := strconv.Itoa(targetLUFS)
	args := []string{"-hide_banner", "-loglevel", "error", "-y", "-i", source, "-af", "loudnorm=I=" + target + ":TP=-1.5:LRA=11", "-ar", "44100", "-ac", "2", destination}
	if output, err := exec.CommandContext(ctx, "ffmpeg", args...).CombinedOutput(); err != nil {
		return SFXMediaStats{}, fmt.Errorf("normalize SFX with ffmpeg: %w: %s", err, strings.TrimSpace(string(output)))
	}
	duration, err := probeSFXDuration(ctx, destination)
	if err != nil {
		return SFXMediaStats{}, err
	}
	loudness, err := probeSFXLoudness(ctx, destination)
	if err != nil {
		return SFXMediaStats{}, err
	}
	for pass := 0; pass < 3 && math.Abs(loudness-float64(targetLUFS)) > 0.5; pass++ {
		adjusted := destination + ".adjusted-" + strconv.Itoa(pass) + ".mp3"
		delta := strconv.FormatFloat(float64(targetLUFS)-loudness, 'f', 2, 64)
		adjustArgs := []string{"-hide_banner", "-loglevel", "error", "-y", "-i", destination, "-af", "volume=" + delta + "dB,alimiter=limit=0.95", "-ar", "44100", "-ac", "2", adjusted}
		if output, adjustErr := exec.CommandContext(ctx, "ffmpeg", adjustArgs...).CombinedOutput(); adjustErr != nil {
			return SFXMediaStats{}, fmt.Errorf("correct SFX loudness with ffmpeg: %w: %s", adjustErr, strings.TrimSpace(string(output)))
		}
		if err := os.Rename(adjusted, destination); err != nil {
			return SFXMediaStats{}, fmt.Errorf("replace corrected SFX: %w", err)
		}
		loudness, err = probeSFXLoudness(ctx, destination)
		if err != nil {
			return SFXMediaStats{}, err
		}
	}
	if math.Abs(loudness-float64(targetLUFS)) > 2.5 {
		return SFXMediaStats{}, fmt.Errorf("SFX loudness remained %.1f LUFS after correction", loudness)
	}
	return SFXMediaStats{DurationSeconds: duration, IntegratedLUFS: loudness}, nil
}

func probeSFXDuration(ctx context.Context, path string) (float64, error) {
	args := []string{"-v", "error", "-show_entries", "format=duration", "-of", "default=nw=1:nk=1", path}
	output, err := exec.CommandContext(ctx, "ffprobe", args...).Output()
	if err != nil {
		return 0, fmt.Errorf("probe SFX duration: %w", err)
	}
	duration, err := strconv.ParseFloat(strings.TrimSpace(string(output)), 64)
	if err != nil {
		return 0, fmt.Errorf("parse SFX duration: %w", err)
	}
	return duration, nil
}

func probeSFXLoudness(ctx context.Context, path string) (float64, error) {
	args := []string{"-hide_banner", "-i", path, "-af", "ebur128=framelog=verbose", "-f", "null", "-"}
	output, _ := exec.CommandContext(ctx, "ffmpeg", args...).CombinedOutput()
	var value float64
	found := false
	for _, line := range strings.Split(string(output), "\n") {
		fields := strings.Fields(line)
		for i := 0; i+1 < len(fields); i++ {
			if fields[i] != "I:" {
				continue
			}
			parsed, err := strconv.ParseFloat(fields[i+1], 64)
			if err == nil {
				value, found = parsed, true
			}
		}
	}
	if !found {
		return 0, errors.New("probe SFX loudness: integrated LUFS not found")
	}
	return value, nil
}

// RenderSFX requests one effect, stores it, and records its loudness target.
func RenderSFX(ctx context.Context, client *http.Client, endpoint, outputDir string, writer *ManifestWriter, asset SFXAsset, take int) error {
	if client == nil || writer == nil {
		return errors.New("buildtime: SFX renderer requires client and writer")
	}
	tmpName, err := requestSFX(ctx, client, endpoint, outputDir, asset)
	if err != nil {
		return err
	}
	defer os.Remove(tmpName)
	if _, err := writer.AddFile(asset.ID, "SFX", tmpName, take); err != nil {
		return err
	}
	return writer.SetMetadata(asset.ID, 0, 0, map[string]string{"target_lufs": strconv.Itoa(asset.LUFS), "postprocess": "ffmpeg loudnorm"})
}

// RenderSFXWithProcessor requests, normalizes, checks, and registers one take.
func RenderSFXWithProcessor(ctx context.Context, client *http.Client, endpoint, outputDir string, writer *ManifestWriter, asset SFXAsset, take int, processor SFXProcessor) (SFXMediaStats, error) {
	if client == nil || writer == nil {
		return SFXMediaStats{}, errors.New("buildtime: SFX renderer requires client and writer")
	}
	if processor == nil {
		return SFXMediaStats{}, errors.New("buildtime: SFX processor is required")
	}
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return SFXMediaStats{}, fmt.Errorf("create SFX output: %w", err)
	}
	tmpName, err := requestSFX(ctx, client, endpoint, outputDir, asset)
	if err != nil {
		return SFXMediaStats{}, err
	}
	defer os.Remove(tmpName)
	finalName := filepath.Join(outputDir, asset.ID+"-take-"+strconv.Itoa(take)+".mp3")
	stats, err := processor.Normalize(ctx, tmpName, finalName, asset.LUFS)
	if err != nil {
		return SFXMediaStats{}, fmt.Errorf("normalize SFX %q take %d: %w", asset.ID, take, err)
	}
	if !acceptableSFXStats(asset, stats) {
		return SFXMediaStats{}, fmt.Errorf("SFX %q take %d failed duration/loudness checks: duration %.2fs, %.1f LUFS", asset.ID, take, stats.DurationSeconds, stats.IntegratedLUFS)
	}
	if _, err := writer.AddFile(asset.ID, "SFX", finalName, take); err != nil {
		return SFXMediaStats{}, err
	}
	return stats, writer.SetMetadata(asset.ID, int64(stats.DurationSeconds*1000), 0, map[string]string{"target_lufs": strconv.Itoa(asset.LUFS), "measured_lufs": strconv.FormatFloat(stats.IntegratedLUFS, 'f', 1, 64), "postprocess": "ffmpeg loudnorm"})
}

func requestSFX(ctx context.Context, client *http.Client, endpoint, outputDir string, asset SFXAsset) (string, error) {
	body, err := BuildSFXRequest(asset)
	if err != nil {
		return "", err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(endpoint, "/")+"/sound-generation", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("buildtime: create SFX request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")
	if key := os.Getenv("DF_ELEVENLABS_API_KEY"); key != "" {
		request.Header.Set("xi-api-key", key)
	}
	response, err := client.Do(request)
	if err != nil {
		return "", fmt.Errorf("buildtime: request SFX %q: %w", asset.ID, err)
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return "", fmt.Errorf("buildtime: SFX %q returned %s", asset.ID, response.Status)
	}
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return "", fmt.Errorf("buildtime: create SFX output: %w", err)
	}
	tmp, err := os.CreateTemp(outputDir, ".sfx-*.mp3")
	if err != nil {
		return "", fmt.Errorf("buildtime: create SFX temporary file: %w", err)
	}
	tmpName := tmp.Name()
	if _, err := io.Copy(tmp, response.Body); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return "", fmt.Errorf("buildtime: save SFX %q: %w", asset.ID, err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return "", fmt.Errorf("buildtime: close SFX %q: %w", asset.ID, err)
	}
	return tmpName, nil
}
