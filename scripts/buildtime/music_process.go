package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// MusicOptions controls the live music build job.
type MusicOptions struct {
	Takes         int
	MaxConcurrent int
	DryRun        bool
	ProcessAudio  bool
	BeatcheckPath string
	FFmpegPath    string
	CostLogPath   string
}

// MusicPlan is the request and cost summary printed before a live run.
type MusicPlan struct {
	Tracks           int     `json:"tracks"`
	Takes            int     `json:"takes"`
	Requests         int     `json:"requests"`
	GeneratedSeconds float64 `json:"generated_seconds"`
	EstimatedCostUSD float64 `json:"estimated_cost_usd"`
	MaxConcurrent    int     `json:"max_concurrent"`
}

// DefaultMusicOptions returns the Creator-plan-safe live defaults.
func DefaultMusicOptions() MusicOptions {
	return MusicOptions{Takes: musicTakeCount, MaxConcurrent: musicConcurrency, ProcessAudio: true}
}

// MusicDryRun returns the work and cost estimate without making network calls.
func MusicDryRun(options MusicOptions) (MusicPlan, error) {
	options = normalizeMusicOptions(options)
	if options.Takes < 1 || options.MaxConcurrent < 1 {
		return MusicPlan{}, errors.New("buildtime: music takes and concurrency must be positive")
	}
	plan := MusicPlan{Tracks: len(MusicTracks()), Takes: options.Takes, MaxConcurrent: options.MaxConcurrent}
	plan.Requests = plan.Tracks * plan.Takes
	for _, track := range MusicTracks() {
		for _, chunk := range musicChunks(track) {
			plan.GeneratedSeconds += float64(chunk.DurationMS) / 1000
		}
	}
	plan.GeneratedSeconds *= float64(options.Takes)
	plan.EstimatedCostUSD = plan.GeneratedSeconds / 60 * musicPricePerMin
	return plan, nil
}

// PrintMusicPlan writes a stable JSON dry-run summary to output.
func PrintMusicPlan(output io.Writer, options MusicOptions) error {
	if output == nil {
		return errors.New("buildtime: music plan output is required")
	}
	plan, err := MusicDryRun(options)
	if err != nil {
		return err
	}
	data, err := json.Marshal(plan)
	if err != nil {
		return fmt.Errorf("buildtime: encode music plan: %w", err)
	}
	_, err = fmt.Fprintln(output, string(data))
	return err
}

func normalizeMusicOptions(options MusicOptions) MusicOptions {
	defaults := DefaultMusicOptions()
	if options.Takes == 0 {
		options.Takes = defaults.Takes
	}
	if options.MaxConcurrent == 0 {
		options.MaxConcurrent = defaults.MaxConcurrent
	}
	if options.FFmpegPath == "" {
		options.FFmpegPath = "ffmpeg"
	}
	if options.CostLogPath == "" {
		options.CostLogPath = "costs.jsonl"
	}
	return options
}

type beatMeasurement struct {
	BPM        float64 `json:"bpm"`
	DownbeatMS float64 `json:"downbeat_ms"`
}

const musicTempoTolerance = 0.03

func runBeatcheck(ctx context.Context, path string, audioPath string, expected int) (beatMeasurement, error) {
	beatcheckPackage := "./scripts/buildtime/beatcheck"
	if _, err := os.Stat(beatcheckPackage); err != nil {
		beatcheckPackage = "../../scripts/buildtime/beatcheck"
	}
	command, args := "go", []string{"run", beatcheckPackage, "-input", audioPath, "-tolerance", "0.03"}
	if path != "" {
		command, args = path, []string{"-input", audioPath, "-tolerance", "0.03"}
	}
	if expected >= 80 {
		args = append(args, "-bpm", fmt.Sprintf("%d", expected))
	}
	result := exec.CommandContext(ctx, command, args...)
	var stderr bytes.Buffer
	result.Stderr = &stderr
	data, err := result.Output()
	if err != nil {
		message := strings.TrimSpace(stderr.String())
		if message == "" {
			message = err.Error()
		}
		return beatMeasurement{}, fmt.Errorf("beatcheck %q: %s", filepath.Base(audioPath), message)
	}
	var measurement beatMeasurement
	if err := json.Unmarshal(data, &measurement); err != nil {
		return beatMeasurement{}, fmt.Errorf("decode beatcheck: %w", err)
	}
	if measurement.BPM <= 0 || measurement.DownbeatMS < 0 {
		return beatMeasurement{}, errors.New("beatcheck returned invalid timing")
	}
	if expected > 0 {
		folded, err := foldMeasuredTempo(measurement.BPM, float64(expected))
		if err != nil {
			return beatMeasurement{}, err
		}
		measurement.BPM = folded
	}
	return measurement, nil
}

func foldMeasuredTempo(actual, expected float64) (float64, error) {
	if actual <= 0 || expected <= 0 {
		return 0, errors.New("beatcheck tempo must be positive")
	}
	for _, candidate := range []struct {
		tempo float64
		fold  float64
	}{
		{tempo: expected, fold: 1},
		{tempo: expected * 2, fold: 0.5},
		{tempo: expected * 0.5, fold: 2},
	} {
		if withinTempo(actual, candidate.tempo) {
			return actual * candidate.fold, nil
		}
	}
	return 0, fmt.Errorf("beatcheck BPM %.2f is outside 3%% of %.0f, including double and half time", actual, expected)
}

func withinTempo(actual, expected float64) bool {
	return actual >= expected*(1-musicTempoTolerance) && actual <= expected*(1+musicTempoTolerance)
}

func processMusicAudio(ctx context.Context, ffmpegPath, inputPath, outputPath string, track MusicTrack, measurement beatMeasurement) error {
	barSeconds := 60 / float64(track.BPM)
	args := []string{"-y", "-i", inputPath}
	filter := fmt.Sprintf("[0:a]atrim=start=%0.3f:end=%0.3f,asetpts=PTS-STARTPTS", measurement.DownbeatMS/1000, (measurement.DownbeatMS+float64(track.DurationMS))/1000)
	if track.Loop {
		start := measurement.DownbeatMS / 1000
		end := (measurement.DownbeatMS + float64(track.DurationMS)) / 1000
		prefixEnd := end - barSeconds
		filter = fmt.Sprintf("[0:a]atrim=start=%0.3f:end=%0.3f,asetpts=PTS-STARTPTS[prefix];[0:a]atrim=start=%0.3f:end=%0.3f,asetpts=PTS-STARTPTS[tail];[0:a]atrim=start=%0.3f:end=%0.3f,asetpts=PTS-STARTPTS[head];[tail][head]acrossfade=d=%0.3f:c1=tri:c2=tri[cross];[prefix][cross]concat=n=2:v=0:a=1[out]", start, prefixEnd, prefixEnd, end, start, start+barSeconds, barSeconds)
	} else {
		filter += ",asetpts=PTS-STARTPTS[out]"
	}
	filter += fmt.Sprintf(";[out]loudnorm=I=%d:TP=-1.5:LRA=11[normalized]", track.LoudnessLUFS)
	args = append(args, "-filter_complex", filter, "-map", "[normalized]", "-c:a", "libopus", "-b:a", "128k", outputPath)
	command := exec.CommandContext(ctx, ffmpegPath, args...)
	if output, err := command.CombinedOutput(); err != nil {
		message := strings.TrimSpace(string(output))
		if message == "" {
			message = err.Error()
		}
		return fmt.Errorf("ffmpeg music post-process: %s", message)
	}
	return nil
}

func musicOutputPath(directory, id string, take int) string {
	return filepath.Join(directory, fmt.Sprintf("%s_take%d.opus", id, take))
}

func ensureMusicDirectory(path string) error {
	if err := os.MkdirAll(path, 0o755); err != nil {
		return fmt.Errorf("buildtime: create music directory: %w", err)
	}
	return nil
}
