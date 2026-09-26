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
	if client == nil || writer == nil {
		return errors.New("buildtime: canned renderer requires client and writer")
	}
	body, err := BuildCannedTTSRequest(line)
	if err != nil {
		return err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint+"/text-to-speech/"+line.Voice+"/stream?output_format=pcm_24000", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("buildtime: create canned request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")
	if key := os.Getenv("DF_ELEVENLABS_API_KEY"); key != "" {
		request.Header.Set("xi-api-key", key)
	}
	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("buildtime: request canned line %q: %w", line.ID, err)
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("buildtime: canned line %q returned %s", line.ID, response.Status)
	}
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return fmt.Errorf("buildtime: create canned output: %w", err)
	}
	tmp, err := os.CreateTemp(outputDir, ".canned-*.pcm")
	if err != nil {
		return fmt.Errorf("buildtime: create canned temporary file: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := io.Copy(tmp, response.Body); err != nil {
		tmp.Close()
		return fmt.Errorf("buildtime: save canned line %q: %w", line.ID, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("buildtime: close canned line %q: %w", line.ID, err)
	}
	if _, err := writer.AddFile(line.ID, "AUDIO", tmpName, take); err != nil {
		return err
	}
	return nil
}

// CannedJob returns a job that renders all fixed non-nudge lines.
func CannedJob(client *http.Client, endpoint, outputDir string, take int) Job {
	return Job{Name: "canned-lines", Run: func(ctx context.Context, writer *ManifestWriter) error {
		for _, line := range CannedLines() {
			if err := RenderCannedLine(ctx, client, endpoint, filepath.Clean(outputDir), writer, line, take); err != nil {
				return err
			}
		}
		return nil
	}}
}
