package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/monstercameron/DungeonFlux/internal/content"
)

const (
	defaultAmbienceDuration  = 30.0
	defaultAmbienceCrossfade = 3 * time.Second
	defaultAmbienceLUFS      = -24
	defaultSoundEndpoint     = "https://api.elevenlabs.io/v1"
)

// AmbienceScene describes one quiet, loopable bed for a one-shot beat.
type AmbienceScene struct {
	ID              string
	BeatID          string
	Prompt          string
	DurationSeconds float64
	Crossfade       time.Duration
	TargetLUFS      int
}

// AmbienceRequest is the ElevenLabs sound-generation request body.
type AmbienceRequest struct {
	Text            string  `json:"text"`
	DurationSeconds float64 `json:"duration_seconds"`
	PromptInfluence float64 `json:"prompt_influence,omitempty"`
}

// LoopTiming contains the slices used to make an end-to-start crossfade.
type LoopTiming struct {
	DurationSeconds  float64
	CrossfadeSeconds float64
}

// AmbienceScenes derives the ambience catalogue from the Drowned Lantern beats.
func AmbienceScenes() []AmbienceScene {
	story := content.DefaultOneShot()
	beats := make(map[string]string, len(story.Beats))
	for _, beat := range story.Beats {
		beats[beat.ID] = beat.Text
	}
	return []AmbienceScene{
		newAmbienceScene("ambience_tavern_rain", "opening", beats["opening"], "quiet tavern room murmur, hearth crackle, and rain on shutters"),
		newAmbienceScene("ambience_harbor_night", "conversation", beats["conversation"], "distant flooded harbor water, rope creak, far boat bell, and soft rain"),
		newAmbienceScene("ambience_bell_tower_wind", "stranger", beats["stranger"], "cold wind around an old bell tower, loose bell rope, river fog, and distant rain"),
		newAmbienceScene("ambience_combat_tension", "combat", beats["combat"], "low building combat tension, wet footsteps, tavern wood strain, and distant thunder"),
		newAmbienceScene("ambience_dawn", "cliffhanger", beats["cliffhanger"], "quiet blue-hour before dawn after a tower bell, river wind, fading rain, and no music"),
		// The two place beds below carry no story-beat text: the combat beat
		// still names the tavern door, and the fight now plays outdoors.
		newAmbienceScene("ambience_river_night", "lobby", "", "night in a rain-soaked river town: soft steady rain on cobbles and eaves, black water lapping at a wooden pier, a lantern creaking on its chain, faint distant thunder"),
		newAmbienceScene("ambience_wooded_path", "combat", "", "a wooded path at night beside a river: wind moving through tall wet trees, leaves rustling, a distant rushing river, sparse raindrops falling from branches, an owl far away"),
	}
}

// FilterAmbienceScenes keeps the scenes whose IDs are listed in only; an
// empty list keeps every scene. It lets a paid run regenerate one bed.
func FilterAmbienceScenes(scenes []AmbienceScene, only []string) []AmbienceScene {
	if len(only) == 0 {
		return scenes
	}
	keep := make(map[string]bool, len(only))
	for _, id := range only {
		keep[strings.TrimSpace(id)] = true
	}
	filtered := make([]AmbienceScene, 0, len(only))
	for _, scene := range scenes {
		if keep[scene.ID] {
			filtered = append(filtered, scene)
		}
	}
	return filtered
}

func newAmbienceScene(id, beatID, beatText, bed string) AmbienceScene {
	parts := []string{
		bed + ".",
		"This is a seamless loopable dark-fantasy ambience bed under spoken narration; no music, no melody, no intelligible words, no sudden loud hits.",
	}
	if beatText != "" {
		parts = append(parts, "Story beat: "+beatText)
	}
	prompt := strings.Join(parts, " ")
	return AmbienceScene{ID: id, BeatID: beatID, Prompt: prompt, DurationSeconds: defaultAmbienceDuration, Crossfade: defaultAmbienceCrossfade, TargetLUFS: defaultAmbienceLUFS}
}

// BuildAmbienceRequest validates and encodes an ElevenLabs ambience request.
func BuildAmbienceRequest(scene AmbienceScene) ([]byte, error) {
	if strings.TrimSpace(scene.ID) == "" || strings.TrimSpace(scene.BeatID) == "" || strings.TrimSpace(scene.Prompt) == "" {
		return nil, errors.New("buildtime: ambience requires id, beat, and prompt")
	}
	if _, err := LoopPlan(scene); err != nil {
		return nil, err
	}
	return json.Marshal(AmbienceRequest{Text: scene.Prompt, DurationSeconds: scene.DurationSeconds, PromptInfluence: 0.3})
}

// LoopPlan validates timing and returns the body and seam lengths in seconds.
func LoopPlan(scene AmbienceScene) (LoopTiming, error) {
	duration := scene.DurationSeconds
	crossfade := scene.Crossfade.Seconds()
	if duration < 30 || duration > 60 {
		return LoopTiming{}, errors.New("buildtime: ambience duration must be between 30 and 60 seconds")
	}
	if crossfade <= 0 || crossfade >= duration/2 {
		return LoopTiming{}, errors.New("buildtime: ambience crossfade must be positive and shorter than half the duration")
	}
	return LoopTiming{DurationSeconds: duration, CrossfadeSeconds: crossfade}, nil
}

// CrossfadeFilter returns an ffmpeg graph that keeps the requested duration.
func CrossfadeFilter(timing LoopTiming) string {
	bodyEnd := timing.DurationSeconds - timing.CrossfadeSeconds
	end := formatSeconds(timing.DurationSeconds)
	body := formatSeconds(bodyEnd)
	crossfade := formatSeconds(timing.CrossfadeSeconds)
	return "[0:a]atrim=start=0:end=" + body + ",asetpts=PTS-STARTPTS[body];" +
		"[0:a]atrim=start=" + body + ":end=" + end + ",asetpts=PTS-STARTPTS[tail];" +
		"[0:a]atrim=start=0:end=" + crossfade + ",asetpts=PTS-STARTPTS[head];" +
		"[tail][head]acrossfade=d=" + crossfade + ":c1=tri:c2=tri[seam];" +
		"[body][seam]concat=n=2:v=0:a=1,loudnorm=I=-24:TP=-2:LRA=7[out]"
}

func formatSeconds(seconds float64) string {
	return strconv.FormatFloat(seconds, 'f', 3, 64)
}

// RenderAmbience generates, crossfades, normalises, and registers one bed.
func RenderAmbience(ctx context.Context, client *http.Client, endpoint, ffmpegPath, outputDir string, writer *ManifestWriter, scene AmbienceScene, take int) error {
	if client == nil || writer == nil {
		return errors.New("buildtime: ambience renderer requires client and writer")
	}
	body, err := BuildAmbienceRequest(scene)
	if err != nil {
		return err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(endpoint, "/")+"/sound-generation", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("buildtime: create ambience request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")
	if key := os.Getenv("DF_ELEVENLABS_API_KEY"); key != "" {
		request.Header.Set("xi-api-key", key)
	}
	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("buildtime: request ambience %q: %w", scene.ID, err)
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("buildtime: ambience %q returned %s", scene.ID, response.Status)
	}
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return fmt.Errorf("buildtime: create ambience output: %w", err)
	}
	source, err := os.CreateTemp(outputDir, ".ambience-source-*.mp3")
	if err != nil {
		return fmt.Errorf("buildtime: create ambience source: %w", err)
	}
	sourceName := source.Name()
	defer os.Remove(sourceName)
	if _, err := io.Copy(source, response.Body); err != nil {
		source.Close()
		return fmt.Errorf("buildtime: save ambience %q: %w", scene.ID, err)
	}
	if err := source.Close(); err != nil {
		return fmt.Errorf("buildtime: close ambience %q: %w", scene.ID, err)
	}
	finalPath := filepath.Join(outputDir, scene.ID+".ogg")
	if err := normaliseAmbience(ctx, ffmpegPath, sourceName, finalPath, scene); err != nil {
		return fmt.Errorf("buildtime: post-process ambience %q: %w", scene.ID, err)
	}
	if _, err := writer.AddFile(scene.ID, "AMBIENCE", finalPath, take); err != nil {
		return err
	}
	return writer.SetMetadata(scene.ID, int64(scene.DurationSeconds*1000), 0, map[string]string{
		"beat":         scene.BeatID,
		"loop":         "true",
		"crossfade_ms": strconv.FormatInt(scene.Crossfade.Milliseconds(), 10),
		"target_lufs":  strconv.Itoa(scene.TargetLUFS),
		"postprocess":  "ffmpeg acrossfade+loudnorm",
		"source":       "internal/content.DefaultOneShot",
	})
}

func normaliseAmbience(ctx context.Context, ffmpegPath, source, destination string, scene AmbienceScene) error {
	if strings.TrimSpace(ffmpegPath) == "" {
		ffmpegPath = "ffmpeg"
	}
	temporary := destination + ".tmp.ogg"
	defer os.Remove(temporary)
	timing, err := LoopPlan(scene)
	if err != nil {
		return err
	}
	args := []string{"-y", "-v", "error", "-i", source, "-filter_complex", CrossfadeFilter(timing), "-map", "[out]", "-c:a", "libopus", "-b:a", "96k", "-ar", "48000", "-ac", "2", temporary}
	output, err := exec.CommandContext(ctx, ffmpegPath, args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("ffmpeg: %w: %s", err, strings.TrimSpace(string(output)))
	}
	if err := os.Rename(temporary, destination); err != nil {
		return fmt.Errorf("replace ambience output: %w", err)
	}
	return nil
}

// AmbienceJob returns the live or dry-run ambience generation job.
func AmbienceJob(client *http.Client, endpoint, ffmpegPath, outputDir string, take int, dryRun bool) Job {
	return AmbienceJobOnly(client, endpoint, ffmpegPath, outputDir, take, dryRun, nil)
}

// AmbienceJobOnly is AmbienceJob limited to the listed scene IDs.
func AmbienceJobOnly(client *http.Client, endpoint, ffmpegPath, outputDir string, take int, dryRun bool, only []string) Job {
	scenes := FilterAmbienceScenes(AmbienceScenes(), only)
	return Job{Name: "ambience-loops", Run: func(ctx context.Context, writer *ManifestWriter) error {
		if dryRun {
			return writeAmbiencePlan(os.Stdout, scenes)
		}
		return withManifestLock(ctx, writer, func(writer *ManifestWriter) error {
			for _, scene := range scenes {
				if err := RenderAmbience(ctx, client, endpoint, ffmpegPath, outputDir, writer, scene, take); err != nil {
					return err
				}
			}
			return nil
		})
	}}
}

func writeAmbiencePlan(out io.Writer, scenes []AmbienceScene) error {
	seconds := 0.0
	if _, err := io.WriteString(out, "ambience dry-run: "+strconv.Itoa(len(scenes))+" requests\n"); err != nil {
		return err
	}
	for _, scene := range scenes {
		seconds += scene.DurationSeconds
		if _, err := io.WriteString(out, "  "+scene.ID+" beat="+scene.BeatID+" seconds="+formatSeconds(scene.DurationSeconds)+" crossfade_ms="+strconv.FormatInt(scene.Crossfade.Milliseconds(), 10)+" target_lufs="+strconv.Itoa(scene.TargetLUFS)+"\n"); err != nil {
			return err
		}
	}
	_, err := io.WriteString(out, "total generated seconds: "+formatSeconds(seconds)+"\n")
	return err
}

func runAmbience(args []string) error {
	flags := flag.NewFlagSet("ambience", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	root := flags.String("root", "artifacts/runtime/buildtime", "build-time output directory")
	endpoint := flags.String("endpoint", defaultSoundEndpoint, "ElevenLabs API base URL")
	ffmpegPath := flags.String("ffmpeg", "ffmpeg", "ffmpeg executable")
	take := flags.Int("take", 1, "manifest take number")
	dryRun := flags.Bool("dry-run", false, "print the generation plan without network or file output")
	onlyList := flags.String("only", os.Getenv("DF_AMBIENCE_ONLY"), "comma-separated scene IDs to generate (default: all)")
	if err := flags.Parse(args); err != nil {
		return err
	}
	var only []string
	if strings.TrimSpace(*onlyList) != "" {
		only = strings.Split(*onlyList, ",")
	}
	writer, err := NewManifestWriter(*root)
	if err != nil {
		return err
	}
	if *dryRun {
		return writeAmbiencePlan(os.Stdout, FilterAmbienceScenes(AmbienceScenes(), only))
	}
	if os.Getenv("DF_ELEVENLABS_API_KEY") == "" {
		return errors.New("buildtime: DF_ELEVENLABS_API_KEY is required for live ambience generation")
	}
	client := &http.Client{Timeout: 2 * time.Minute}
	return AmbienceJobOnly(client, *endpoint, *ffmpegPath, filepath.Join(*root, "ambience"), *take, false, only).Run(context.Background(), writer)
}

func init() {
	if len(os.Args) < 2 || os.Args[1] != "ambience" {
		return
	}
	if err := runAmbience(os.Args[2:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	os.Exit(0)
}
