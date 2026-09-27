package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

const elevenLabsTTSCostPerCharacter = 0.00005

// TTSPlan is the no-network estimate for a fixed audio job.
type TTSPlan struct {
	Requests         int
	Characters       int
	EstimatedCostUSD float64
}

// CannedLine is a fixed spoken line rendered before a run starts.
type CannedLine struct {
	ID    string
	Voice string
	Text  string
}

// CannedLines returns the eleven non-nudge lines in the one-shot.
func CannedLines() []CannedLine {
	return []CannedLine{
		{ID: "canned_opening", Voice: "dm", Text: "Rain hammers the Drowned Lantern. The lamplighter vanished last night, and the river is rising. Two travellers shake off the wet at the bar, where Mother Vell watches with her one good eye."},
		{ID: "canned_npc_reply", Voice: "mother_vell", Text: "Lots of folk drink here, love. I don't keep a ledger of faces, and I don't answer questions for free."},
		{ID: "canned_npc_reveal", Voice: "mother_vell", Text: "Fine. They dragged him toward the old bell tower. And at midnight that bell rang, though nobody's climbed it in years."},
		{ID: "canned_npc_refuse", Voice: "mother_vell", Text: "Nice try. I've buried better talkers than you. Drink up or move along; I've nothing more to say."},
		{ID: "canned_stranger_found", Voice: "courier", Text: "A letter, for one of you. The seal's river-soaked, and I didn't read it. Whatever was in that water, it followed me from the river."},
		{ID: "canned_stranger_relocated", Voice: "courier", Text: "A letter, for one of you. They say the lamplighter was dragged to the old bell tower. Something wet and dead guarded it, and it followed me from the river."},
		{ID: "canned_cliffhanger_vell", Voice: "dm", Text: "Midnight. The tower bell Mother Vell warned of tolls, and every lantern in the tavern gutters out. In the dark it rings again, slow and patient. Whoever pulls that rope already knows your names."},
		{ID: "canned_cliffhanger_stranger", Voice: "dm", Text: "Midnight. The courier's letter falls open as the tower bell tolls, and every lantern in the tavern gutters out. Inside, in wet ink, are your names. The bell rings again."},
		{ID: "canned_combat_slain_seat1", Voice: "dm", Text: "Steel finds the thrall's heart of river mud, and it collapses into a pool of dark water."},
		{ID: "canned_combat_slain_seat2", Voice: "dm", Text: "One last blow, and the thrall sags. The river takes back its own."},
		{ID: "canned_combat_fled", Voice: "dm", Text: "Far off, the tower bell tolls once. The thrall turns mid-swing and lurches into the rain, toward the tower."},
	}
}

// CannedTTSPlan returns the request and character estimate for canned lines.
func CannedTTSPlan() TTSPlan {
	return planCannedLines(CannedLines())
}

func planCannedLines(lines []CannedLine) TTSPlan {
	plan := TTSPlan{Requests: len(lines)}
	for _, line := range lines {
		plan.Characters += len([]rune(line.Text))
	}
	plan.EstimatedCostUSD = float64(plan.Characters) * elevenLabsTTSCostPerCharacter
	return plan
}

// CannedTTSRequest is the ElevenLabs HTTP request for one fixed line.
type CannedTTSRequest struct {
	Text    string `json:"text"`
	ModelID string `json:"model_id"`
}

// BuildCannedTTSRequest encodes a fixed line for ElevenLabs HTTP /stream.
func BuildCannedTTSRequest(line CannedLine) ([]byte, error) {
	if line.ID == "" || line.Voice == "" || line.Text == "" {
		return nil, errors.New("buildtime: canned line requires id, voice, and text")
	}
	return json.Marshal(CannedTTSRequest{Text: line.Text, ModelID: "eleven_flash_v2_5"})
}

// RenderCannedLine requests one line and records its content-addressed take.
func RenderCannedLine(ctx context.Context, client *http.Client, endpoint, outputDir string, writer *ManifestWriter, line CannedLine, take int) error {
	_, err := renderCannedLine(ctx, client, endpoint, outputDir, writer, line, take, false)
	return err
}

type renderedAudio struct {
	Path       string
	DurationMS int64
	Characters int
	CostUSD    float64
}

func renderCannedLine(ctx context.Context, client *http.Client, endpoint, outputDir string, writer *ManifestWriter, line CannedLine, take int, normalize bool) (renderedAudio, error) {
	var result renderedAudio
	if client == nil || writer == nil {
		return result, errors.New("buildtime: canned renderer requires client and writer")
	}
	if filepath.Base(line.ID) != line.ID || strings.ContainsAny(line.ID, `/\\`) {
		return result, fmt.Errorf("buildtime: invalid audio id %q", line.ID)
	}
	tmpName, data, err := requestCannedPCM(ctx, client, endpoint, outputDir, line)
	if err != nil {
		return result, err
	}
	defer os.Remove(tmpName)
	source := tmpName
	if normalize {
		source = filepath.Join(outputDir, line.ID+"-take-"+strconv.Itoa(take)+".pcm")
		if err := normalizePCM(ctx, tmpName, source); err != nil {
			_ = os.Remove(source)
			return result, fmt.Errorf("buildtime: normalize canned line %q: %w", line.ID, err)
		}
		data, err = os.ReadFile(source)
		if err != nil {
			_ = os.Remove(source)
			return result, fmt.Errorf("buildtime: read normalized line %q: %w", line.ID, err)
		}
	}
	if _, err := writer.AddFile(line.ID, "AUDIO", source, take); err != nil {
		return result, err
	}
	result = renderedAudio{
		Path:       source,
		DurationMS: int64(len(data)) * 1000 / (24000 * 2),
		Characters: len([]rune(line.Text)),
		CostUSD:    float64(len([]rune(line.Text))) * elevenLabsTTSCostPerCharacter,
	}
	if normalize {
		if err := writer.SetMetadata(line.ID, result.DurationMS, 0, audioMetadata(line, result)); err != nil {
			return renderedAudio{}, err
		}
		logTTSCall(line, result)
	}
	return result, nil
}

func requestCannedPCM(ctx context.Context, client *http.Client, endpoint, outputDir string, line CannedLine) (string, []byte, error) {
	body, err := BuildCannedTTSRequest(line)
	if err != nil {
		return "", nil, err
	}
	voice := resolveVoiceID(line.Voice)
	url := strings.TrimRight(endpoint, "/") + "/text-to-speech/" + voice + "/stream?output_format=pcm_24000"
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", nil, fmt.Errorf("buildtime: create canned request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")
	if key := os.Getenv("DF_ELEVENLABS_API_KEY"); key != "" {
		request.Header.Set("xi-api-key", key)
	}
	response, err := client.Do(request)
	if err != nil {
		return "", nil, fmt.Errorf("buildtime: request canned line %q: %w", line.ID, err)
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return "", nil, fmt.Errorf("buildtime: canned line %q returned %s", line.ID, response.Status)
	}
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return "", nil, fmt.Errorf("buildtime: create canned output: %w", err)
	}
	tmp, err := os.CreateTemp(outputDir, ".canned-*.pcm")
	if err != nil {
		return "", nil, fmt.Errorf("buildtime: create canned temporary file: %w", err)
	}
	tmpName := tmp.Name()
	if _, err := io.Copy(tmp, response.Body); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return "", nil, fmt.Errorf("buildtime: save canned line %q: %w", line.ID, err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return "", nil, fmt.Errorf("buildtime: close canned line %q: %w", line.ID, err)
	}
	data, err := os.ReadFile(tmpName)
	if err != nil {
		_ = os.Remove(tmpName)
		return "", nil, fmt.Errorf("buildtime: inspect canned line %q: %w", line.ID, err)
	}
	if len(data) == 0 {
		_ = os.Remove(tmpName)
		return "", nil, fmt.Errorf("buildtime: canned line %q returned empty audio", line.ID)
	}
	return tmpName, data, nil
}

func resolveVoiceID(logical string) string {
	key := "DF_ELEVENLABS_VOICE_" + strings.ToUpper(strings.ReplaceAll(logical, "-", "_"))
	if voice := strings.TrimSpace(os.Getenv(key)); voice != "" {
		return voice
	}
	switch logical {
	case "dm":
		return "21m00Tcm4TlvDq8ikWAM"
	case "mother_vell":
		return "EXAVITQu4vr4xnSDxMaL"
	case "courier":
		return "pNInz6obpgDQGcFmaJgB"
	}
	return logical
}

// CannedJob returns a job that renders all fixed non-nudge lines.
func CannedJob(client *http.Client, endpoint, outputDir string, take int) Job {
	return Job{Name: "canned-lines", Run: func(ctx context.Context, writer *ManifestWriter) error {
		return withManifestLock(ctx, writer, func(writer *ManifestWriter) error {
			return renderCannedLinesLive(ctx, client, endpoint, audioOutputDir(outputDir), writer, take)
		})
	}}
}

func renderCannedLinesLive(ctx context.Context, client *http.Client, endpoint, outputDir string, writer *ManifestWriter, take int) error {
	files := make([]string, 0, len(CannedLines()))
	for _, line := range CannedLines() {
		result, err := renderCannedLine(ctx, client, endpoint, outputDir, writer, line, take, true)
		if err != nil {
			return err
		}
		files = append(files, summaryFile(result, outputDir))
	}
	logTTSSummary("canned-lines", files)
	return nil
}

func normalizePCM(ctx context.Context, source, destination string) error {
	command := exec.CommandContext(ctx, "ffmpeg", "-hide_banner", "-loglevel", "error", "-y", "-f", "s16le", "-ar", "24000", "-ac", "1", "-i", source, "-af", "loudnorm=I=-16:TP=-1.5:LRA=11", "-f", "s16le", "-ar", "24000", "-ac", "1", destination)
	output, err := command.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ffmpeg loudnorm: %w (%s)", err, strings.TrimSpace(string(output)))
	}
	return nil
}

func audioMetadata(line CannedLine, result renderedAudio) map[string]string {
	return map[string]string{
		"voice":         line.Voice,
		"characters":    strconv.Itoa(result.Characters),
		"cost_usd":      fmt.Sprintf("%.6f", result.CostUSD),
		"normalization": "loudnorm:I=-16:TP=-1.5:LRA=11",
		"sample_rate":   "24000",
		"channels":      "1",
		"format":        "pcm_s16le",
	}
}

func logTTSCall(line CannedLine, result renderedAudio) {
	slog.New(slog.NewTextHandler(os.Stderr, nil)).Info("tts_call", "asset", line.ID, "voice", line.Voice, "characters", result.Characters, "duration_ms", result.DurationMS, "cost_usd", result.CostUSD)
}

func logTTSSummary(job string, files []string) {
	slog.New(slog.NewTextHandler(os.Stderr, nil)).Info("tts_summary", "job", job, "requests", len(files), "files", strings.Join(files, ", "))
}

func summaryFile(result renderedAudio, outputDir string) string {
	return fmt.Sprintf("%s duration_ms=%d characters=%d", filepath.ToSlash(filepath.Join(filepath.Base(outputDir), filepath.Base(result.Path))), result.DurationMS, result.Characters)
}

func audioOutputDir(root string) string {
	root = filepath.Clean(root)
	if strings.EqualFold(filepath.Base(root), "audio") {
		return root
	}
	return filepath.Join(root, "audio")
}
