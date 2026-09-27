package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync"
)

const musicBudgetUSD = 2

// MusicJob returns the live job for the complete three-take music catalogue.
func MusicJob(client *http.Client, endpoint, outputDir string, take int) Job {
	options := DefaultMusicOptions()
	return MusicJobWithOptions(client, endpoint, outputDir, take, options)
}

// MusicJobWithOptions returns a music job with explicit dry-run and tool settings.
func MusicJobWithOptions(client *http.Client, endpoint, outputDir string, take int, options MusicOptions) Job {
	return Job{Name: "music-tracks", Run: func(ctx context.Context, writer *ManifestWriter) error {
		if options.DryRun {
			return PrintMusicPlan(os.Stdout, options)
		}
		if client == nil || writer == nil {
			return errors.New("buildtime: music job requires client and writer")
		}
		return runMusic(ctx, client, endpoint, outputDir, take, options, writer)
	}}
}

func runMusic(ctx context.Context, client *http.Client, endpoint, outputDir string, take int, options MusicOptions, writer *ManifestWriter) error {
	options = normalizeMusicOptions(options)
	if options.Takes < 1 || options.MaxConcurrent < 1 {
		return errors.New("buildtime: music takes and concurrency must be positive")
	}
	if options.MaxConcurrent > musicConcurrency {
		return fmt.Errorf("buildtime: music concurrency %d exceeds Creator limit %d", options.MaxConcurrent, musicConcurrency)
	}
	musicDir := filepath.Join(outputDir, "music")
	if err := ensureMusicDirectory(musicDir); err != nil {
		return err
	}
	tracks, err := selectedMusicTracks(options.TrackIDs)
	if err != nil {
		return err
	}
	return withManifestLock(ctx, writer, func(writer *ManifestWriter) error {
		costs := &musicCostLog{path: resolveCostPath(musicDir, options.CostLogPath)}
		themeID := ""
		var themeErr error
		if len(tracks) > 0 && tracks[0].ID == "THEME_MAIN" {
			themeID, themeErr = renderTheme(ctx, client, endpoint, musicDir, options, tracks[0], costs, writer)
			tracks = tracks[1:]
		}
		trackErr := renderMusicTracks(ctx, client, endpoint, musicDir, options, themeID, tracks, costs, writer)
		_, writeErr := writer.Write()
		return errors.Join(themeErr, trackErr, writeErr)
	})
}

func renderTheme(ctx context.Context, client *http.Client, endpoint, outputDir string, options MusicOptions, track MusicTrack, costs *musicCostLog, writer *ManifestWriter) (string, error) {
	var themeID string
	var failures []error
	for takeNumber := 1; takeNumber <= options.Takes; takeNumber++ {
		id, err := renderMusicTakeWithRetry(ctx, client, endpoint, outputDir, options, "", track, takeNumber, costs, writer)
		if err != nil {
			failures = append(failures, err)
			continue
		}
		if id != "" {
			themeID = id
		}
	}
	return themeID, errors.Join(failures...)
}

func renderMusicTracks(ctx context.Context, client *http.Client, endpoint, outputDir string, options MusicOptions, themeID string, tracks []MusicTrack, costs *musicCostLog, writer *ManifestWriter) error {
	jobs := make(chan MusicTrack)
	errs := make(chan error, len(tracks))
	var workers sync.WaitGroup
	for i := 0; i < options.MaxConcurrent; i++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for track := range jobs {
				for takeNumber := 1; takeNumber <= options.Takes; takeNumber++ {
					if _, err := renderMusicTakeWithRetry(ctx, client, endpoint, outputDir, options, themeID, track, takeNumber, costs, writer); err != nil {
						errs <- err
						break
					}
				}
			}
		}()
	}
	for _, track := range tracks {
		jobs <- track
	}
	close(jobs)
	workers.Wait()
	close(errs)
	var failures []error
	for err := range errs {
		failures = append(failures, err)
	}
	return errors.Join(failures...)
}

func renderMusicTakeWithRetry(ctx context.Context, client *http.Client, endpoint, outputDir string, options MusicOptions, themeID string, track MusicTrack, take int, costs *musicCostLog, writer *ManifestWriter) (string, error) {
	var last error
	for attempt := 1; attempt <= 2; attempt++ {
		if !costs.reserve(track) {
			return "", fmt.Errorf("music budget cap reached before %s take %d", track.ID, take)
		}
		songID, err := renderMusicTake(ctx, client, endpoint, outputDir, options, themeID, track, take, writer)
		costs.record(track, take, attempt, err == nil)
		if err == nil {
			return songID, nil
		}
		last = err
		if ctx.Err() != nil {
			break
		}
	}
	return "", fmt.Errorf("music %s take %d failed after retry: %w", track.ID, take, last)
}

func renderMusicTake(ctx context.Context, client *http.Client, endpoint, outputDir string, options MusicOptions, themeID string, track MusicTrack, take int, writer *ManifestWriter) (string, error) {
	body, err := BuildMusicRequestWithTheme(track, themeID)
	if err != nil {
		return "", err
	}
	audio, songID, err := requestMusic(ctx, client, endpoint, body)
	if err != nil {
		return "", fmt.Errorf("buildtime: request music %q: %w", track.ID, err)
	}
	rawPath := filepath.Join(outputDir, fmt.Sprintf(".%s_take%d.mp3", track.ID, take))
	if err := os.WriteFile(rawPath, audio, 0o644); err != nil {
		return "", fmt.Errorf("buildtime: save raw music %q: %w", track.ID, err)
	}
	defer os.Remove(rawPath)
	measurement := beatMeasurement{}
	finalPath := musicOutputPath(outputDir, track.ID, take)
	if options.ProcessAudio {
		// Only loops need the measured tempo to match: their loop points are
		// bar-aligned. Stingers and one-shot beds (sparse, free-time) failed
		// the check even when the audio was fine, so they skip tempo folding.
		expectedBPM := track.BPM
		if !track.Loop {
			expectedBPM = 0
		}
		measurement, err = runBeatcheck(ctx, options.BeatcheckPath, rawPath, expectedBPM)
		if err != nil && !track.Loop {
			// A single hit and a held note has no regular onsets; a
			// non-loop track is then trimmed from its start.
			measurement, err = beatMeasurement{BPM: float64(track.BPM)}, nil
		}
		if err != nil {
			return "", err
		}
		if err := processMusicAudio(ctx, options.FFmpegPath, rawPath, finalPath, track, measurement); err != nil {
			return "", err
		}
	} else {
		finalPath = filepath.Join(outputDir, fmt.Sprintf("%s_take%d.mp3", track.ID, take))
		if err := os.WriteFile(finalPath, audio, 0o644); err != nil {
			return "", fmt.Errorf("buildtime: save music %q: %w", track.ID, err)
		}
	}
	if _, err := writer.AddFile(track.ID, "MUSIC", finalPath, take); err != nil {
		return "", err
	}
	loopStart := int64(measurement.DownbeatMS)
	loopEnd := loopStart + track.DurationMS
	if !track.Loop {
		loopEnd = 0
	}
	if err := writer.SetMetadata(track.ID, track.DurationMS, 0, musicMetadata(track, loopStart, loopEnd, songID, body, take)); err != nil {
		return "", err
	}
	return songID, nil
}

func resolveCostPath(directory, path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(directory, path)
}

type musicCostEntry struct {
	Vendor    string  `json:"vendor"`
	Model     string  `json:"model"`
	Track     string  `json:"track"`
	Take      int     `json:"take"`
	Attempt   int     `json:"attempt"`
	Seconds   float64 `json:"seconds"`
	CostUSD   float64 `json:"cost_usd"`
	Succeeded bool    `json:"succeeded"`
}

type musicCostLog struct {
	mu       sync.Mutex
	path     string
	spentUSD float64
}

func (log *musicCostLog) record(track MusicTrack, take, attempt int, succeeded bool) {
	var milliseconds int64
	for _, chunk := range musicChunks(track) {
		milliseconds += chunk.DurationMS
	}
	seconds := float64(milliseconds) / 1000
	entry := musicCostEntry{Vendor: "elevenlabs", Model: musicModel, Track: track.ID, Take: take, Attempt: attempt, Seconds: seconds, CostUSD: seconds / 60 * musicPricePerMin, Succeeded: succeeded}
	data, err := json.Marshal(entry)
	if err != nil {
		return
	}
	log.mu.Lock()
	defer log.mu.Unlock()
	file, err := os.OpenFile(log.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return
	}
	_, _ = file.Write(append(data, '\n'))
	_ = file.Close()
}

func (log *musicCostLog) reserve(track MusicTrack) bool {
	var milliseconds int64
	for _, chunk := range musicChunks(track) {
		milliseconds += chunk.DurationMS
	}
	cost := float64(milliseconds) / 60000 * musicPricePerMin
	log.mu.Lock()
	defer log.mu.Unlock()
	if log.spentUSD+cost > musicBudgetUSD {
		return false
	}
	log.spentUSD += cost
	return true
}
