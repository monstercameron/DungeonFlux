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
	"time"
)

// ClipSpec describes one build-time video shot.
type ClipSpec struct {
	ID            string
	Prompt        string
	FirstFrameURL string
	LastFrameURL  string
	DurationMS    int64
	Resolution    string
	Take          int
	Pinned        bool
}

// VideoRequest is the provider-neutral request sent to a video adapter.
type VideoRequest struct {
	Prompt        string `json:"prompt"`
	FirstFrameURL string `json:"first_frame_url"`
	LastFrameURL  string `json:"last_frame_url,omitempty"`
	Duration      int    `json:"duration"`
	Resolution    string `json:"resolution"`
	GenerateAudio bool   `json:"generate_audio"`
}

type videoSubmitResponse struct {
	RequestID string `json:"request_id"`
	Status    string `json:"status"`
	VideoURL  string `json:"video_url"`
	Error     string `json:"error,omitempty"`
}

// VideoClient submits and downloads one video without making vendor calls in dry-run mode.
type VideoClient struct {
	HTTPClient   *http.Client
	Endpoint     string
	APIKey       string
	PollInterval time.Duration
}

// BuildVideoRequest validates a shot and encodes the pinned no-audio request.
func BuildVideoRequest(spec ClipSpec) ([]byte, error) {
	if err := validateClipSpec(spec); err != nil {
		return nil, err
	}
	req := VideoRequest{
		Prompt: spec.Prompt, FirstFrameURL: spec.FirstFrameURL, LastFrameURL: spec.LastFrameURL,
		Duration: int(spec.DurationMS / 1000), Resolution: spec.Resolution, GenerateAudio: false,
	}
	return json.Marshal(req)
}

// ClipSpecs returns the opening and stranger-arrival production shots.
func ClipSpecs() []ClipSpec {
	return []ClipSpec{
		{
			ID: "EST_WIDE_PUSH", FirstFrameURL: "buildtime://tavern-wide", LastFrameURL: "buildtime://tavern-scene",
			DurationMS: 5000, Resolution: "720p", Take: 1, Pinned: true,
			Prompt: "Fog drifts across a crowded smugglers' tavern at night; lanterns sway gently and river water laps at the windows. The camera pushes in slowly and steadily toward the bar. Warm lantern key light from the left, cool blue moonlight rim through the windows, drifting volumetric fog. Stylized dark fantasy illustration, painterly, muted teal and amber palette. Wide 16:9 composition, no text.",
		},
		{
			ID: "ARRIVAL_DOOR_STATIC", FirstFrameURL: "buildtime://courier-door", DurationMS: 5000, Resolution: "720p", Take: 1,
			Prompt: "A hooded courier, dripping river water, steps slowly forward out of the rain through the open tavern door into the lamplight, clutching a sealed letter. Rain falls behind him. Static camera, locked-off wide shot of the doorway. Warm lantern light on his face, cold blue rain light behind. Stylized dark fantasy illustration, painterly, muted teal and amber palette. Wide 16:9 composition, no text.",
		},
	}
}

// RenderVideo submits a shot, polls a completed response, downloads it, and records metadata.
func (c VideoClient) RenderVideo(ctx context.Context, writer *ManifestWriter, outputDir string, spec ClipSpec) error {
	if writer == nil {
		return errors.New("buildtime: video renderer requires a manifest writer")
	}
	if strings.TrimSpace(c.Endpoint) == "" || c.HTTPClient == nil {
		return errors.New("buildtime: video renderer requires endpoint and HTTP client")
	}
	body, err := BuildVideoRequest(spec)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.Endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("buildtime: create video request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("buildtime: submit video %q: %w", spec.ID, err)
	}
	defer resp.Body.Close()
	var submitted videoSubmitResponse
	if err := json.NewDecoder(resp.Body).Decode(&submitted); err != nil {
		return fmt.Errorf("buildtime: decode video submission: %w", err)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices || submitted.Error != "" {
		return fmt.Errorf("buildtime: video %q submission failed: %s", spec.ID, videoError(resp, submitted))
	}
	videoURL, err := c.completedURL(ctx, submitted)
	if err != nil {
		return fmt.Errorf("buildtime: video %q: %w", spec.ID, err)
	}
	data, err := c.download(ctx, videoURL)
	if err != nil {
		return fmt.Errorf("buildtime: download video %q: %w", spec.ID, err)
	}
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return fmt.Errorf("buildtime: create video output: %w", err)
	}
	tmp, err := os.CreateTemp(outputDir, ".clip-*.mp4")
	if err != nil {
		return fmt.Errorf("buildtime: create video temporary file: %w", err)
	}
	name := tmp.Name()
	defer os.Remove(name)
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("buildtime: write video %q: %w", spec.ID, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("buildtime: close video %q: %w", spec.ID, err)
	}
	if _, err := writer.AddFile(spec.ID, "VIDEO", name, spec.Take); err != nil {
		return err
	}
	return writer.SetMetadata(spec.ID, spec.DurationMS, 0, map[string]string{
		"shot": spec.ID, "resolution": spec.Resolution, "pinned": fmt.Sprintf("%t", spec.Pinned), "audio": "false",
	})
}

// ClipJob returns the production clip job. Dry-run prints shots without contacting a vendor.
func ClipJob(client VideoClient, outputDir string, dryRun bool) Job {
	return Job{Name: "video-clips", Run: func(ctx context.Context, writer *ManifestWriter) error {
		for _, spec := range ClipSpecs() {
			if dryRun {
				fmt.Printf("dry-run video %s (%dms, %s)\n", spec.ID, spec.DurationMS, spec.Resolution)
				continue
			}
			if err := client.RenderVideo(ctx, writer, filepath.Clean(outputDir), spec); err != nil {
				return err
			}
		}
		return nil
	}}
}

func (c VideoClient) completedURL(ctx context.Context, submitted videoSubmitResponse) (string, error) {
	if submitted.VideoURL != "" {
		return submitted.VideoURL, nil
	}
	if submitted.RequestID == "" {
		return "", errors.New("video response has neither request_id nor video_url")
	}
	if submitted.Status != "done" {
		if c.PollInterval > 0 {
			timer := time.NewTimer(c.PollInterval)
			defer timer.Stop()
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-timer.C:
			}
		}
	}
	statusURL := strings.TrimRight(c.Endpoint, "/") + "/" + submitted.RequestID
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, statusURL, nil)
	if err != nil {
		return "", fmt.Errorf("create video status request: %w", err)
	}
	if c.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("poll video status: %w", err)
	}
	defer resp.Body.Close()
	var status videoSubmitResponse
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return "", fmt.Errorf("decode video status: %w", err)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices || status.VideoURL == "" {
		return "", fmt.Errorf("video job %q is not complete", submitted.RequestID)
	}
	return status.VideoURL, nil
}

func (c VideoClient) download(ctx context.Context, videoURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, videoURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create video download request: %w", err)
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("video download returned %s", resp.Status)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, errors.New("video download was empty")
	}
	return data, nil
}

func validateClipSpec(spec ClipSpec) error {
	if strings.TrimSpace(spec.ID) == "" || strings.TrimSpace(spec.Prompt) == "" || strings.TrimSpace(spec.FirstFrameURL) == "" {
		return errors.New("buildtime: video requires id, prompt, and first frame")
	}
	if spec.DurationMS < 1000 || spec.DurationMS%1000 != 0 || spec.Resolution == "" || spec.Take < 1 {
		return errors.New("buildtime: video has invalid duration, resolution, or take")
	}
	if spec.LastFrameURL != "" && !spec.Pinned {
		return errors.New("buildtime: last frame requires a pinned video")
	}
	return nil
}

func videoError(resp *http.Response, submitted videoSubmitResponse) string {
	if submitted.Error != "" {
		return submitted.Error
	}
	return resp.Status
}
