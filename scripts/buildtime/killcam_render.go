package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/monstercameron/DungeonFlux/internal/ports"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
	"github.com/monstercameron/DungeonFlux/internal/wire"
)

// Persist queue jobs before polling: interrupted renders resume without a new paid submission.
func renderKillcam(ctx context.Context, root, name, class, outcome string, req ports.ReferenceVideoRequest, vendor ports.ReferenceVideoGen, remaining *float64, interval time.Duration) (bool, error) {
	key := killcamKey(req)
	if cachedKillcam(root, name, key) {
		return true, nil
	}
	if vendor == nil {
		return false, errors.New("killcams: cache miss; generation requires DF_FAL_KEY")
	}
	jobDir := filepath.Join(root, "killcam-jobs")
	if err := os.MkdirAll(jobDir, 0700); err != nil {
		return false, err
	}
	jobPath := filepath.Join(jobDir, key+".json.tmp")
	job, err := killcamJob(ctx, vendor, req, jobPath, remaining)
	if err != nil {
		return false, err
	}
	data, err := pollKillcam(ctx, vendor, job, interval)
	if err != nil {
		return false, err
	}
	if len(data) == 0 {
		return false, errors.New("killcams: empty video")
	}
	file := filepath.Join(jobDir, key+".mp4")
	if err := os.WriteFile(file, data, 0600); err != nil {
		return false, err
	}
	writer, err := NewManifestWriter(root)
	if err != nil {
		return false, err
	}
	if _, err := writer.AddFile(name, "VIDEO", file, 1); err != nil {
		return false, err
	}
	if err := writer.SetMetadata(name, 4000, 2000, map[string]string{"cache_key": key, "class": class, "outcome": outcome, "model": wire.KillcamVideoModel, "resolution": "720p", "aspect": "16:9", "audio": "false", "prompt_version": "killcam-v1"}); err != nil {
		return false, err
	}
	if _, err := writer.Write(); err != nil {
		return false, err
	}
	if err := os.Remove(file); err != nil {
		return false, err
	}
	return false, nil
}

func killcamJob(ctx context.Context, vendor ports.ReferenceVideoGen, req ports.ReferenceVideoRequest, path string, remaining *float64) (ports.VideoJob, error) {
	var job ports.VideoJob
	data, err := os.ReadFile(path)
	if err == nil {
		if err := json.Unmarshal(data, &job); err != nil {
			return job, err
		}
		if job.ID == "" {
			return job, errors.New("killcams: cached job has no ID")
		}
		return job, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return job, err
	}
	const estimate = 0.968 // Four seconds at the documented 720p rate of $0.242/s.
	if remaining == nil || *remaining < estimate {
		return job, errors.New("killcams: estimated spend cap exceeded")
	}
	*remaining -= estimate
	job, err = vendor.SubmitReference(ctx, req)
	if err != nil {
		return job, err
	}
	data, err = json.Marshal(job)
	if err != nil {
		return job, err
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		return job, fmt.Errorf("persist submitted job %s: %w", job.ID, err)
	}
	return job, nil
}

func pollKillcam(ctx context.Context, vendor ports.ReferenceVideoGen, job ports.VideoJob, interval time.Duration) ([]byte, error) {
	for {
		status, err := vendor.Poll(ctx, job)
		if err != nil {
			return nil, err
		}
		if status.State == vocab.JobDone {
			return vendor.Download(ctx, status.URL)
		}
		if status.State == vocab.JobFailed {
			return nil, errors.New("killcams: vendor job failed")
		}
		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, ctx.Err()
		case <-timer.C:
		}
	}
}
