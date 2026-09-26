package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const musicLibraryVersion = "v1"

const (
	musicModel       = "music_v2_5"
	musicTakeCount   = 3
	musicConcurrency = 2
	musicPricePerMin = 0.15
)

var musicNegativeCore = []string{"vocals", "lyrics", "choir singing words", "spoken word", "electronic drums", "synthesizer bass", "EDM", "pop production", "rock drum kit", "electric guitar", "trailer braams", "sub drops", "hybrid trailer orchestra"}
var musicNegativeLoop = []string{"intro", "outro", "ending", "fade out", "final chord", "silence", "tempo change", "rubato"}

// MusicTrack describes one of the demo's fixed music cues.
type MusicTrack struct {
	ID             string
	DurationMS     int64
	BPM            int
	Key            string
	LoopStartMS    int64
	LoopEndMS      int64
	Prompt         string
	Loop           bool
	LoudnessLUFS   int
	ConditionTheme bool
}

// MusicTracks returns the twelve tracks in the §0.19 catalogue.
func MusicTracks() []MusicTrack {
	return []MusicTrack{
		{ID: "THEME_MAIN", DurationMS: 96000, BPM: 80, Key: "D dorian", LoopEndMS: 96000, Loop: true, LoudnessLUFS: -16, Prompt: "dark fantasy chamber folk main theme for a rain-soaked smugglers' river town, hurdy-gurdy, cello, harp, low whistle lantern call"},
		{ID: "CREATION_BED_LOOP", DurationMS: 96000, BPM: 80, Key: "D dorian", LoopEndMS: 96000, Loop: true, LoudnessLUFS: -20, ConditionTheme: true, Prompt: "sparse quiet dark fantasy chamber folk bed for character creation, soft harp and celesta, no lead melody"},
		{ID: "OPENING_SWELL", DurationMS: 12000, BPM: 80, Key: "D dorian", LoudnessLUFS: -16, ConditionTheme: true, Prompt: "dark fantasy orchestral folk swell through river fog toward a lit tavern"},
		{ID: "TAVERN_WARM_LOOP", DurationMS: 96000, BPM: 80, Key: "F major", LoopEndMS: 96000, Loop: true, LoudnessLUFS: -20, ConditionTheme: true, Prompt: "warm sparse smugglers' tavern underscore under spoken dialogue, harp, muted hand drum, low whistle rests"},
		{ID: "STING_STRANGER", DurationMS: 6000, BPM: 80, Key: "D phrygian", LoudnessLUFS: -16, ConditionTheme: true, Prompt: "ominous hooded stranger arrival sting with bowed metal shimmer and a deep bell"},
		{ID: "STING_COMBAT_START", DurationMS: 4000, BPM: 160, Key: "D minor", LoudnessLUFS: -16, ConditionTheme: true, Prompt: "sudden drowned corpse ambush hit with low strings and frame drum, one downbeat then held note"},
		{ID: "COMBAT_SKIRMISH_LOOP", DurationMS: 96000, BPM: 160, Key: "D minor", LoopEndMS: 96000, Loop: true, LoudnessLUFS: -20, ConditionTheme: true, Prompt: "driving fantasy tavern skirmish loop, large frame drums, cello and bass ostinato, hurdy-gurdy drone, bold whistle fragments"},
		{ID: "STING_VICTORY", DurationMS: 6000, BPM: 120, Key: "D major", LoudnessLUFS: -16, ConditionTheme: true, Prompt: "hard-won tavern brawl victory sting, rising whistle, bright harp, final bell in D"},
		{ID: "STING_BELL_TOLL", DurationMS: 4000, BPM: 60, Key: "D", LoudnessLUFS: -16, ConditionTheme: true, Prompt: "one deep tower bell toll in D with a long decay over rain"},
		{ID: "CLIFF_TENSION_BED", DurationMS: 24000, BPM: 60, Key: "D phrygian", LoudnessLUFS: -20, ConditionTheme: true, Prompt: "sparse midnight cliffhanger tension bed under narration, low cello pedal, bowed metal, distant bell, no climax"},
		{ID: "STING_CLIFF_HIT", DurationMS: 6000, BPM: 60, Key: "D", LoudnessLUFS: -16, ConditionTheme: true, Prompt: "massive tower bell strike as every tavern lantern goes out, low frame drum and long ringing decay"},
		{ID: "END_CARD_THEME", DurationMS: 24000, BPM: 80, Key: "D dorian", LoudnessLUFS: -16, ConditionTheme: true, Prompt: "to be continued ending with the lantern-call whistle resolving to a held D chord and final bell"},
	}
}

// MusicChunk is one section in an ElevenLabs composition plan.
type MusicChunk struct {
	Text             string   `json:"text"`
	DurationMS       int64    `json:"duration_ms"`
	PositiveStyle    []string `json:"positive_styles,omitempty"`
	NegativeStyle    []string `json:"negative_styles,omitempty"`
	ContextAdherence string   `json:"context_adherence,omitempty"`
}

// MusicConditioningRef identifies the theme excerpt used to condition a take.
type MusicConditioningRef struct {
	SongID string `json:"song_id"`
	Range  struct {
		StartMS int64 `json:"start_ms"`
		EndMS   int64 `json:"end_ms"`
	} `json:"range"`
}

// MusicRequest is the detailed ElevenLabs Music API request body.
type MusicRequest struct {
	ModelID           string                `json:"model_id"`
	Seed              uint32                `json:"seed,omitempty"`
	StoreForInpaint   bool                  `json:"store_for_inpainting"`
	ConditioningRef   *MusicConditioningRef `json:"conditioning_ref,omitempty"`
	ConditionStrength string                `json:"condition_strength,omitempty"`
	CompositionPlan   struct {
		Chunks []MusicChunk `json:"chunks"`
	} `json:"composition_plan"`
}

// MusicSeed returns the deterministic seed required by the build plan.
func MusicSeed(trackID string) uint32 {
	return crc32.ChecksumIEEE([]byte(trackID+musicLibraryVersion)) & 0x7fffffff
}

// BuildMusicRequest encodes a detailed music request with the pinned model.
func BuildMusicRequest(track MusicTrack) ([]byte, error) {
	return BuildMusicRequestWithTheme(track, "")
}

// BuildMusicRequestWithTheme adds the theme conditioning reference when available.
func BuildMusicRequestWithTheme(track MusicTrack, themeSongID string) ([]byte, error) {
	if err := validateMusicTrack(track); err != nil {
		return nil, err
	}
	request := MusicRequest{ModelID: musicModel, StoreForInpaint: true}
	if track.ID != "END_CARD_THEME" {
		request.Seed = MusicSeed(track.ID)
	}
	if track.ConditionTheme && strings.TrimSpace(themeSongID) != "" {
		request.ConditioningRef = &MusicConditioningRef{SongID: themeSongID}
		request.ConditioningRef.Range.EndMS = 24000
		request.ConditionStrength = "medium"
		if isMusicStinger(track.ID) || track.ID == "END_CARD_THEME" {
			request.ConditionStrength = "high"
		}
	}
	request.CompositionPlan.Chunks = musicChunks(track)
	return json.Marshal(request)
}

func validateMusicTrack(track MusicTrack) error {
	if track.ID == "" || track.DurationMS < 3000 || track.BPM <= 0 || track.Key == "" || track.Prompt == "" {
		return errors.New("buildtime: invalid music track")
	}
	if track.LoopEndMS != 0 && track.LoopEndMS <= track.LoopStartMS {
		return errors.New("buildtime: music loop end must follow loop start")
	}
	return nil
}

func isMusicStinger(id string) bool {
	return strings.HasPrefix(id, "STING_")
}

func musicChunks(track MusicTrack) []MusicChunk {
	tempo := fmt.Sprintf("strict steady tempo at exactly %d BPM, 4/4", track.BPM)
	core := []string{"dark fantasy chamber folk underscore", "hurdy-gurdy drone, cello and double bass, harp, low whistle, celesta, frame drum", "D dorian tonal center", "warm intimate close-mic'd acoustic recording", "medium stone-hall reverb", tempo, "instrumental only", fmt.Sprintf("%d BPM", track.BPM), track.Key}
	negative := append(append([]string(nil), musicNegativeCore...), musicNegativeLoop...)
	if track.ID == "THEME_MAIN" {
		return []MusicChunk{
			{Text: fmt.Sprintf("[Main Theme A]\nEXACTLY %d BPM, strict 4/4, no tempo drift\n{instrumental}\n%s", track.BPM, track.Prompt), DurationMS: 36000, PositiveStyle: append(core, "starts immediately with the full ensemble"), NegativeStyle: negative, ContextAdherence: "high"},
			{Text: fmt.Sprintf("[Main Theme B]\nEXACTLY %d BPM, strict 4/4, no tempo drift\n{instrumental}\ncello answers the whistle, frame drum joins softly", track.BPM), DurationMS: 36000, PositiveStyle: append(core, "warmer development of the same theme"), NegativeStyle: negative, ContextAdherence: "high"},
			{Text: "[Main Theme A return]\n{instrumental}\nrestate the opening phrase", DurationMS: 24000, PositiveStyle: core, NegativeStyle: negative, ContextAdherence: "high"},
			{Text: "[Return to top]\n{instrumental}\ncontinue exactly as the first bars, no cadence", DurationMS: 12000, PositiveStyle: core, NegativeStyle: negative, ContextAdherence: "high"},
		}
	}
	if track.ID == "END_CARD_THEME" {
		return []MusicChunk{
			{Text: "[Theme reference]\n{instrumental}\ncarry the lantern-call motif", DurationMS: 12000, PositiveStyle: core, NegativeStyle: negative, ContextAdherence: "high"},
			{Text: "[Ending]\n{instrumental}\nresolve to a held D chord and final bell", DurationMS: 12000, PositiveStyle: append(core, "a gentle to-be-continued ending"), NegativeStyle: negative, ContextAdherence: "high"},
		}
	}
	if track.Loop {
		return []MusicChunk{
			{Text: fmt.Sprintf("[Instrumental loop body]\n{instrumental}\n%s, exactly %d BPM with an unwavering 4/4 pulse", track.Prompt, track.BPM), DurationMS: track.DurationMS, PositiveStyle: core, NegativeStyle: negative, ContextAdherence: "high"},
			{Text: "[Return]\n{instrumental}\ncontinue exactly as the opening bars, no cadence", DurationMS: 12000, PositiveStyle: []string{"same texture as the opening", fmt.Sprintf("%d BPM", track.BPM)}, NegativeStyle: negative, ContextAdherence: "high"},
		}
	}
	return []MusicChunk{{Text: fmt.Sprintf("[Instrumental]\n{instrumental}\n%s, exactly %d BPM with an unwavering 4/4 pulse", track.Prompt, track.BPM), DurationMS: track.DurationMS, PositiveStyle: core, NegativeStyle: negative, ContextAdherence: "high"}}
}

func requestMusic(ctx context.Context, client *http.Client, endpoint string, body []byte) ([]byte, string, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(endpoint, "/")+"/music/detailed?output_format=mp3_44100_192", bytes.NewReader(body))
	if err != nil {
		return nil, "", fmt.Errorf("buildtime: create music request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")
	if key := os.Getenv("DF_ELEVENLABS_API_KEY"); key != "" {
		request.Header.Set("xi-api-key", key)
	}
	response, err := client.Do(request)
	if err != nil {
		return nil, "", err
	}
	defer response.Body.Close()
	data, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, "", fmt.Errorf("read music response: %w", err)
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		message := strings.TrimSpace(string(data))
		if len(message) > 500 {
			message = message[:500]
		}
		return nil, "", fmt.Errorf("music API returned %s: %s", response.Status, message)
	}
	if len(data) == 0 {
		return nil, "", errors.New("music API returned empty audio")
	}
	return data, firstHeader(response.Header, "song_id", "x-song-id"), nil
}

func firstHeader(header http.Header, names ...string) string {
	for _, name := range names {
		if value := strings.TrimSpace(header.Get(name)); value != "" {
			return value
		}
	}
	return ""
}

// RenderMusic requests one raw track, stores it, and records loop metadata.
func RenderMusic(ctx context.Context, client *http.Client, endpoint, outputDir string, writer *ManifestWriter, track MusicTrack, take int) error {
	if client == nil || writer == nil {
		return errors.New("buildtime: music renderer requires client and writer")
	}
	body, err := BuildMusicRequest(track)
	if err != nil {
		return err
	}
	audio, _, err := requestMusic(ctx, client, endpoint, body)
	if err != nil {
		return fmt.Errorf("buildtime: request music %q: %w", track.ID, err)
	}
	path, err := writeMusicFile(outputDir, track.ID, take, ".mp3", audio)
	if err != nil {
		return err
	}
	defer os.Remove(path)
	if _, err := writer.AddFile(track.ID, "MUSIC", path, take); err != nil {
		return err
	}
	loopEnd := int64(0)
	if track.Loop {
		loopEnd = track.DurationMS
	}
	return writer.SetMetadata(track.ID, track.DurationMS, 0, musicMetadata(track, 0, loopEnd, "", body, take))
}

func writeMusicFile(directory, id string, take int, extension string, data []byte) (string, error) {
	if take < 1 {
		return "", errors.New("buildtime: music take must be positive")
	}
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return "", fmt.Errorf("buildtime: create music output: %w", err)
	}
	path := filepath.Join(directory, fmt.Sprintf("%s_take%d%s", id, take, extension))
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", fmt.Errorf("buildtime: save music: %w", err)
	}
	return path, nil
}

func musicMetadata(track MusicTrack, downbeat, loopEnd int64, songID string, plan []byte, take int) map[string]string {
	hash := sha256.Sum256(plan)
	seed := ""
	if track.ID != "END_CARD_THEME" {
		seed = fmt.Sprintf("%d", MusicSeed(track.ID))
	}
	return map[string]string{
		"bpm":             fmt.Sprintf("%d", track.BPM),
		"bar_ms":          fmt.Sprintf("%d", 240000/track.BPM),
		"key":             track.Key,
		"loop_start_ms":   fmt.Sprintf("%d", downbeat),
		"loop_end_ms":     fmt.Sprintf("%d", loopEnd),
		"lufs":            fmt.Sprintf("%d", track.LoudnessLUFS),
		"seed":            seed,
		"plan_sha256":     hex.EncodeToString(hash[:]),
		"model":           musicModel,
		"take":            fmt.Sprintf("%d", take),
		"song_id":         songID,
		"processed_audio": "true",
	}
}
