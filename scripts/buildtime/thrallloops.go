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
	"strings"
)

const defaultVideoURL = "https://api.segmind.com/v1/seedance"

// LoopSpec describes one four-second billboard loop.
type LoopSpec struct {
	LogicalName string
	Prompt      string
	Take        int
	DurationMS  int64
	ContactMS   int64
	FirstFrame  string
	Pinned      bool
}

// LoopOptions controls a Seedance loop job.
type LoopOptions struct {
	Endpoint string
	APIKey   string
	Model    string
	Specs    []LoopSpec
	DryRun   bool
}

type loopRequest struct {
	Model           string `json:"model"`
	Prompt          string `json:"prompt"`
	DurationSeconds int    `json:"duration_seconds"`
	Width           int    `json:"width"`
	Height          int    `json:"height"`
	FirstFrame      string `json:"first_frame,omitempty"`
	NoAudio         bool   `json:"no_audio"`
	Loop            bool   `json:"loop"`
}

type loopResponse struct {
	VideoURL string `json:"video_url"`
	Error    string `json:"error,omitempty"`
}

// LoopClient submits a loop request and downloads its resulting video.
type LoopClient struct {
	HTTPClient *http.Client
	Endpoint   string
	APIKey     string
	Model      string
}

// Generate submits one loop request and returns the resulting bytes.
func (c LoopClient) Generate(ctx context.Context, spec LoopSpec) ([]byte, error) {
	if strings.TrimSpace(c.Endpoint) == "" || strings.TrimSpace(c.APIKey) == "" {
		return nil, errors.New("buildtime: video endpoint and API key are required")
	}
	body, err := json.Marshal(buildLoopRequest(c.Model, spec))
	if err != nil {
		return nil, fmt.Errorf("buildtime: encode video request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.Endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("buildtime: create video request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	req.Header.Set("Content-Type", "application/json")
	client := c.HTTPClient
	if client == nil {
		client = &http.Client{}
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("buildtime: video request: %w", err)
	}
	defer resp.Body.Close()
	var decoded loopResponse
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return nil, fmt.Errorf("buildtime: decode video response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if decoded.Error == "" {
			decoded.Error = resp.Status
		}
		return nil, fmt.Errorf("buildtime: video API: %s", decoded.Error)
	}
	if decoded.VideoURL == "" {
		return nil, errors.New("buildtime: video response has no video_url")
	}
	return downloadLoop(ctx, client, decoded.VideoURL)
}

func downloadLoop(ctx context.Context, client *http.Client, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("buildtime: create video download: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("buildtime: download video: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("buildtime: video download: %s", resp.Status)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("buildtime: read video: %w", err)
	}
	if len(data) == 0 {
		return nil, errors.New("buildtime: video download was empty")
	}
	return data, nil
}

func buildLoopRequest(model string, spec LoopSpec) loopRequest {
	if model == "" {
		model = "seedance-2.0-mini"
	}
	return loopRequest{Model: model, Prompt: spec.Prompt, DurationSeconds: 4, Width: 480, Height: 854, FirstFrame: spec.FirstFrame, NoAudio: true, Loop: spec.Pinned}
}

// RunLoopJob renders and registers loop takes with measured timing metadata.
func RunLoopJob(ctx context.Context, writer *ManifestWriter, options LoopOptions, kind string) error {
	if writer == nil {
		return errors.New("buildtime: nil manifest writer")
	}
	if len(options.Specs) == 0 {
		return errors.New("buildtime: no loop specs")
	}
	for _, spec := range options.Specs {
		if err := validateLoopSpec(spec); err != nil {
			return err
		}
		if options.DryRun {
			fmt.Printf("dry-run loop %s (%s)\n", spec.LogicalName, effectiveVideoModel(options.Model))
			continue
		}
		data, err := (LoopClient{Endpoint: effectiveVideoEndpoint(options.Endpoint), APIKey: options.APIKey, Model: options.Model}).Generate(ctx, spec)
		if err != nil {
			return fmt.Errorf("loop %q: %w", spec.LogicalName, err)
		}
		tmp, err := os.CreateTemp(writer.root, "loop-*.mp4")
		if err != nil {
			return fmt.Errorf("loop %q: create temporary file: %w", spec.LogicalName, err)
		}
		name := tmp.Name()
		if _, err := tmp.Write(data); err != nil {
			tmp.Close()
			os.Remove(name)
			return fmt.Errorf("loop %q: write temporary file: %w", spec.LogicalName, err)
		}
		if err := tmp.Close(); err != nil {
			os.Remove(name)
			return fmt.Errorf("loop %q: close temporary file: %w", spec.LogicalName, err)
		}
		_, addErr := writer.AddFile(spec.LogicalName, kind, name, spec.Take)
		os.Remove(name)
		if addErr != nil {
			return fmt.Errorf("loop %q: %w", spec.LogicalName, addErr)
		}
		metadata := map[string]string{"model": effectiveVideoModel(options.Model), "resolution": "480x854", "aspect": "9:16", "no_audio": "true"}
		if spec.Pinned {
			metadata["pinned"] = "true"
		}
		if err := writer.SetMetadata(spec.LogicalName, spec.DurationMS, spec.ContactMS, metadata); err != nil {
			return err
		}
	}
	return nil
}

// RunThrallLoopsJob registers the thrall still, cut-out, and four loops.
func RunThrallLoopsJob(ctx context.Context, writer *ManifestWriter, options LoopOptions, stillPath, cutoutPath string) error {
	if !options.DryRun {
		for logical, source := range map[string]string{"thrall_still": stillPath, "thrall_cutout": cutoutPath} {
			if strings.TrimSpace(source) == "" {
				return fmt.Errorf("buildtime: %s source is required", logical)
			}
			if _, err := writer.AddFile(logical, map[string]string{"thrall_still": "IMAGE_STILL", "thrall_cutout": "IMAGE_CUTOUT"}[logical], source, 1); err != nil {
				return err
			}
		}
	} else {
		fmt.Println("dry-run thrall_still (existing still)")
		fmt.Println("dry-run thrall_cutout (existing cut-out)")
	}
	return RunLoopJob(ctx, writer, options, "VIDEO_LOOP")
}

func validateLoopSpec(spec LoopSpec) error {
	if strings.TrimSpace(spec.LogicalName) == "" || filepath.Base(spec.LogicalName) != spec.LogicalName {
		return fmt.Errorf("buildtime: invalid loop logical name %q", spec.LogicalName)
	}
	if strings.TrimSpace(spec.Prompt) == "" {
		return fmt.Errorf("buildtime: empty prompt for loop %q", spec.LogicalName)
	}
	if spec.Take < 1 || spec.DurationMS < 1 || spec.ContactMS < 0 {
		return fmt.Errorf("buildtime: invalid timing or take for loop %q", spec.LogicalName)
	}
	return nil
}

func effectiveVideoEndpoint(endpoint string) string {
	if endpoint != "" {
		return endpoint
	}
	if value := os.Getenv("DF_SEGMIND_VIDEO_URL"); value != "" {
		return value
	}
	return defaultVideoURL
}
func effectiveVideoModel(model string) string {
	if model == "" {
		return "seedance-2.0-mini"
	}
	return model
}
