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

const defaultMarbleURL = "https://api.worldlabs.ai/marble/v1/worlds:generate"

// SplatSpec identifies one Marble output and its desired density.
type SplatSpec struct {
	LogicalName string
	Quality     string
	Take        int
}

// SplatOptions controls the Marble generation and PLY export job.
type SplatOptions struct {
	Endpoint     string
	APIKey       string
	Prompt       string
	ImageURL     string
	Specs        []SplatSpec
	PollInterval time.Duration
	DryRun       bool
}

type marbleClient struct {
	httpClient   *http.Client
	endpoint     string
	apiKey       string
	pollInterval time.Duration
}
type marbleRequest struct {
	Model    string `json:"model"`
	Prompt   string `json:"prompt,omitempty"`
	ImageURL string `json:"image_url,omitempty"`
	Quality  string `json:"quality,omitempty"`
}
type marbleOperation struct {
	OperationID string `json:"operation_id"`
	Done        bool   `json:"done"`
	Error       *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
	Response marbleWorld `json:"response"`
}
type marbleWorld struct {
	ID     string `json:"id"`
	Assets struct {
		Splats struct {
			SemanticsMetadata struct {
				MetricScaleFactor float64 `json:"metric_scale_factor"`
				GroundPlaneOffset float64 `json:"ground_plane_offset"`
			} `json:"semantics_metadata"`
		} `json:"splats"`
	} `json:"assets"`
}
type exportRequest struct {
	AssetType  string `json:"asset_type"`
	Format     string `json:"format"`
	Resolution string `json:"resolution"`
}
type exportResponse struct {
	Done  bool `json:"done"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
	Response struct {
		URL string `json:"url"`
	} `json:"response"`
}

// RunSplatJob generates PLY assets and records Marble scale metadata.
func RunSplatJob(ctx context.Context, writer *ManifestWriter, options SplatOptions) error {
	if writer == nil {
		return errors.New("buildtime: nil manifest writer")
	}
	if len(options.Specs) == 0 {
		return errors.New("buildtime: no splat specs")
	}
	for _, spec := range options.Specs {
		if err := validateSplatSpec(spec); err != nil {
			return err
		}
		if options.DryRun {
			fmt.Printf("dry-run splat %s (%s)\n", spec.LogicalName, effectiveQuality(spec.Quality))
			continue
		}
		client := marbleClient{httpClient: &http.Client{}, endpoint: effectiveMarbleEndpoint(options.Endpoint), apiKey: options.APIKey, pollInterval: options.PollInterval}
		world, err := client.Generate(ctx, marbleRequest{Model: "marble-1.1", Prompt: options.Prompt, ImageURL: options.ImageURL, Quality: effectiveQuality(spec.Quality)})
		if err != nil {
			return fmt.Errorf("splat %q: %w", spec.LogicalName, err)
		}
		if world.ID == "" {
			return fmt.Errorf("splat %q: Marble response has no world ID", spec.LogicalName)
		}
		export, err := client.Export(ctx, world.ID, exportRequest{AssetType: "splats", Format: "ply", Resolution: exportResolution(spec.Quality)})
		if err != nil {
			return fmt.Errorf("splat %q: export PLY: %w", spec.LogicalName, err)
		}
		if export.Response.URL == "" {
			return fmt.Errorf("splat %q: Marble export has no PLY URL", spec.LogicalName)
		}
		ply, err := client.Download(ctx, export.Response.URL)
		if err != nil {
			return fmt.Errorf("splat %q: download PLY: %w", spec.LogicalName, err)
		}
		if err := addPLY(writer, spec, ply); err != nil {
			return fmt.Errorf("splat %q: %w", spec.LogicalName, err)
		}
		metadata := map[string]string{"metric_scale_factor": fmt.Sprintf("%g", world.Assets.Splats.SemanticsMetadata.MetricScaleFactor), "ground_plane_offset": fmt.Sprintf("%g", world.Assets.Splats.SemanticsMetadata.GroundPlaneOffset), "quality": effectiveQuality(spec.Quality)}
		if err := writer.SetMetadata(spec.LogicalName, 0, 0, metadata); err != nil {
			return err
		}
	}
	return nil
}

func (c marbleClient) Generate(ctx context.Context, request marbleRequest) (marbleWorld, error) {
	if strings.TrimSpace(c.apiKey) == "" {
		return marbleWorld{}, errors.New("buildtime: World Labs API key is required")
	}
	body, err := json.Marshal(request)
	if err != nil {
		return marbleWorld{}, fmt.Errorf("encode Marble request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(body))
	if err != nil {
		return marbleWorld{}, fmt.Errorf("create Marble request: %w", err)
	}
	req.Header.Set("WLT-Api-Key", c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return marbleWorld{}, fmt.Errorf("marble request: %w", err)
	}
	defer resp.Body.Close()
	var operation marbleOperation
	if err := json.NewDecoder(resp.Body).Decode(&operation); err != nil {
		return marbleWorld{}, fmt.Errorf("decode Marble response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return marbleWorld{}, fmt.Errorf("marble API: %s", marbleError(operation.Error, resp.Status))
	}
	if operation.Done {
		return operation.Response, nil
	}
	if operation.OperationID == "" {
		return marbleWorld{}, errors.New("marble response has no operation ID")
	}
	return c.poll(ctx, operation.OperationID)
}

func (c marbleClient) Export(ctx context.Context, worldID string, export exportRequest) (exportResponse, error) {
	body, err := json.Marshal(export)
	if err != nil {
		return exportResponse{}, fmt.Errorf("encode Marble export: %w", err)
	}
	base := strings.TrimSuffix(strings.TrimRight(c.endpoint, "/"), "worlds:generate")
	if base == c.endpoint {
		base = strings.TrimRight(c.endpoint, "/") + "/"
	}
	url := base + "worlds/" + worldID + ":export"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return exportResponse{}, fmt.Errorf("create Marble export: %w", err)
	}
	req.Header.Set("WLT-Api-Key", c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return exportResponse{}, fmt.Errorf("marble export: %w", err)
	}
	defer resp.Body.Close()
	var result exportResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return exportResponse{}, fmt.Errorf("decode Marble export: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return exportResponse{}, fmt.Errorf("marble export: %s", marbleError(result.Error, resp.Status))
	}
	if !result.Done {
		return exportResponse{}, errors.New("marble PLY export was not completed")
	}
	return result, nil
}

func (c marbleClient) poll(ctx context.Context, operationID string) (marbleWorld, error) {
	interval := c.pollInterval
	if interval <= 0 {
		interval = 5 * time.Second
	}
	for {
		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return marbleWorld{}, ctx.Err()
		case <-timer.C:
		}
		url := strings.TrimRight(c.endpoint, "/") + "/operations/" + operationID
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return marbleWorld{}, fmt.Errorf("create Marble poll: %w", err)
		}
		req.Header.Set("WLT-Api-Key", c.apiKey)
		resp, err := c.httpClient.Do(req)
		if err != nil {
			return marbleWorld{}, fmt.Errorf("marble poll: %w", err)
		}
		var operation marbleOperation
		decodeErr := json.NewDecoder(resp.Body).Decode(&operation)
		resp.Body.Close()
		if decodeErr != nil {
			return marbleWorld{}, fmt.Errorf("decode Marble poll: %w", decodeErr)
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return marbleWorld{}, fmt.Errorf("marble poll: %s", marbleError(operation.Error, resp.Status))
		}
		if operation.Error != nil {
			return marbleWorld{}, errors.New(operation.Error.Message)
		}
		if operation.Done {
			return operation.Response, nil
		}
	}
}

func (c marbleClient) Download(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("WLT-Api-Key", c.apiKey)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("asset download: %s", resp.Status)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read asset: %w", err)
	}
	if len(data) == 0 {
		return nil, errors.New("asset download was empty")
	}
	return data, nil
}

func addPLY(writer *ManifestWriter, spec SplatSpec, ply []byte) error {
	outputName := filepath.Join(writer.root, fmt.Sprintf("%s.ply", spec.LogicalName))
	if err := os.WriteFile(outputName, ply, 0o644); err != nil {
		return fmt.Errorf("write PLY: %w", err)
	}
	defer os.Remove(outputName)
	_, err := writer.AddFile(spec.LogicalName, "SPLAT", outputName, spec.Take)
	return err
}

func exportResolution(quality string) string {
	if effectiveQuality(quality) == "lite" {
		return "100k"
	}
	return "full_res"
}
func validateSplatSpec(spec SplatSpec) error {
	if strings.TrimSpace(spec.LogicalName) == "" || filepath.Base(spec.LogicalName) != spec.LogicalName {
		return fmt.Errorf("buildtime: invalid splat logical name %q", spec.LogicalName)
	}
	if spec.Take < 1 {
		return fmt.Errorf("buildtime: splat %q has invalid take", spec.LogicalName)
	}
	return nil
}
func effectiveMarbleEndpoint(endpoint string) string {
	if endpoint != "" {
		return endpoint
	}
	if value := os.Getenv("DF_WORLDLABS_URL"); value != "" {
		return value
	}
	return defaultMarbleURL
}
func effectiveQuality(quality string) string {
	if quality == "" {
		return "full"
	}
	return quality
}
func marbleError(apiError *struct {
	Message string `json:"message"`
}, fallback string) string {
	if apiError != nil && apiError.Message != "" {
		return apiError.Message
	}
	return fallback
}
