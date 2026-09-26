package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

const musicLibraryVersion = "v1"

// MusicTrack describes one of the demo's fixed music cues.
type MusicTrack struct {
	ID          string
	DurationMS  int64
	BPM         int
	Key         string
	LoopStartMS int64
	LoopEndMS   int64
	Prompt      string
}

// MusicTracks returns the twelve tracks in the §0.19 catalogue.
func MusicTracks() []MusicTrack {
	return []MusicTrack{
		{ID: "THEME_MAIN", DurationMS: 96000, BPM: 80, Key: "D dorian", LoopEndMS: 96000, Prompt: "dark fantasy chamber folk main theme for a rain-soaked smugglers' river town, instrumental only, hurdy-gurdy, cello, harp, low whistle lantern call, steady 80 BPM"},
		{ID: "CREATION_BED_LOOP", DurationMS: 96000, BPM: 80, Key: "D dorian", LoopEndMS: 96000, Prompt: "sparse quiet dark fantasy chamber folk bed for character creation, instrumental only, soft harp and celesta, no lead melody, steady 80 BPM"},
		{ID: "OPENING_SWELL", DurationMS: 12000, BPM: 80, Key: "D dorian", Prompt: "dark fantasy orchestral folk swell through river fog toward a lit tavern, instrumental only, steady 80 BPM"},
		{ID: "TAVERN_WARM_LOOP", DurationMS: 96000, BPM: 80, Key: "F major", LoopEndMS: 96000, Prompt: "warm sparse smugglers' tavern underscore under spoken dialogue, harp, muted hand drum, low whistle rests, instrumental only, steady 80 BPM"},
		{ID: "STING_STRANGER", DurationMS: 6000, BPM: 80, Key: "D phrygian", Prompt: "ominous hooded stranger arrival sting with bowed metal shimmer and a deep bell, instrumental only"},
		{ID: "STING_COMBAT_START", DurationMS: 4000, BPM: 160, Key: "D minor", Prompt: "sudden drowned corpse ambush hit with low strings and frame drum, one downbeat then held note, instrumental only, 160 BPM"},
		{ID: "COMBAT_SKIRMISH_LOOP", DurationMS: 96000, BPM: 160, Key: "D minor", LoopEndMS: 96000, Prompt: "driving fantasy tavern skirmish loop, large frame drums, cello and bass ostinato, hurdy-gurdy drone, bold whistle fragments, instrumental only, steady 160 BPM"},
		{ID: "STING_VICTORY", DurationMS: 6000, BPM: 120, Key: "D major", Prompt: "hard-won tavern brawl victory sting, rising whistle, bright harp, final bell in D, instrumental only"},
		{ID: "STING_BELL_TOLL", DurationMS: 4000, BPM: 60, Key: "D", Prompt: "one deep tower bell toll in D with a long decay over rain, instrumental only"},
		{ID: "CLIFF_TENSION_BED", DurationMS: 24000, BPM: 60, Key: "D phrygian", Prompt: "sparse midnight cliffhanger tension bed under narration, low cello pedal, bowed metal, distant bell, no climax, instrumental only, 60 BPM"},
		{ID: "STING_CLIFF_HIT", DurationMS: 6000, BPM: 60, Key: "D", Prompt: "massive tower bell strike as every tavern lantern goes out, low frame drum and long ringing decay, instrumental only"},
		{ID: "END_CARD_THEME", DurationMS: 24000, BPM: 80, Key: "D dorian", Prompt: "to be continued ending with the lantern-call whistle resolving to a held D chord and final bell, instrumental only, 80 BPM"},
	}
}

// MusicChunk is one section in an ElevenLabs composition plan.
type MusicChunk struct {
	Text          string   `json:"text"`
	DurationMS    int64    `json:"duration_ms"`
	PositiveStyle []string `json:"positive_styles,omitempty"`
	NegativeStyle []string `json:"negative_styles,omitempty"`
}

// MusicRequest is the detailed ElevenLabs Music API request body.
type MusicRequest struct {
	ModelID         string `json:"model_id"`
	Seed            uint32 `json:"seed,omitempty"`
	StoreForInpaint bool   `json:"store_for_inpainting"`
	CompositionPlan struct {
		Chunks []MusicChunk `json:"chunks"`
	} `json:"composition_plan"`
}

// MusicSeed returns the deterministic seed required by the build plan.
func MusicSeed(trackID string) uint32 {
	return crc32.ChecksumIEEE([]byte(trackID + musicLibraryVersion))
}

// BuildMusicRequest encodes a detailed music request with the pinned model.
func BuildMusicRequest(track MusicTrack) ([]byte, error) {
	if track.ID == "" || track.DurationMS < 3000 || track.BPM <= 0 || track.Prompt == "" {
		return nil, errors.New("buildtime: invalid music track")
	}
	if track.LoopEndMS != 0 && track.LoopEndMS <= track.LoopStartMS {
		return nil, errors.New("buildtime: music loop end must follow loop start")
	}
	request := MusicRequest{ModelID: "music_v2_5", Seed: MusicSeed(track.ID), StoreForInpaint: true}
	request.CompositionPlan.Chunks = []MusicChunk{{Text: track.Prompt, DurationMS: track.DurationMS, PositiveStyle: []string{"instrumental only", fmt.Sprintf("%d BPM", track.BPM), track.Key}, NegativeStyle: []string{"vocals", "lyrics", "fade out"}}}
	return json.Marshal(request)
}

// RenderMusic requests one track, stores it, and records loop metadata.
func RenderMusic(ctx context.Context, client *http.Client, endpoint, outputDir string, writer *ManifestWriter, track MusicTrack, take int) error {
	if client == nil || writer == nil {
		return errors.New("buildtime: music renderer requires client and writer")
	}
	body, err := BuildMusicRequest(track)
	if err != nil {
		return err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint+"/music/detailed?output_format=mp3_44100_192", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("buildtime: create music request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")
	if key := os.Getenv("DF_ELEVENLABS_API_KEY"); key != "" {
		request.Header.Set("xi-api-key", key)
	}
	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("buildtime: request music %q: %w", track.ID, err)
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("buildtime: music %q returned %s", track.ID, response.Status)
	}
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return fmt.Errorf("buildtime: create music output: %w", err)
	}
	tmp, err := os.CreateTemp(outputDir, ".music-*.mp3")
	if err != nil {
		return fmt.Errorf("buildtime: create music temporary file: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := io.Copy(tmp, response.Body); err != nil {
		tmp.Close()
		return fmt.Errorf("buildtime: save music %q: %w", track.ID, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("buildtime: close music %q: %w", track.ID, err)
	}
	if _, err := writer.AddFile(track.ID, "MUSIC", tmpName, take); err != nil {
		return err
	}
	return writer.SetMetadata(track.ID, track.DurationMS, 0, map[string]string{
		"bpm":           fmt.Sprintf("%d", track.BPM),
		"key":           track.Key,
		"loop_start_ms": fmt.Sprintf("%d", track.LoopStartMS),
		"loop_end_ms":   fmt.Sprintf("%d", track.LoopEndMS),
		"seed":          fmt.Sprintf("%d", MusicSeed(track.ID)),
		"model":         "music_v2_5",
	})
}

// MusicJob returns a job that renders the complete music catalogue.
func MusicJob(client *http.Client, endpoint, outputDir string, take int) Job {
	return Job{Name: "music-tracks", Run: func(ctx context.Context, writer *ManifestWriter) error {
		for _, track := range MusicTracks() {
			if err := RenderMusic(ctx, client, endpoint, filepath.Clean(outputDir), writer, track, take); err != nil {
				return err
			}
		}
		return nil
	}}
}
