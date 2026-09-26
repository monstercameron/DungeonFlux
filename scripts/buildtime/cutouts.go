package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const defaultImagesURL = "https://api.openai.com/v1/images/generations"

// CutoutSpec describes one transparent full-body image to generate.
type CutoutSpec struct {
	LogicalName string
	Prompt      string
	Take        int
}

// CutoutOptions controls an Images API cut-out job.
type CutoutOptions struct {
	Endpoint string
	APIKey   string
	Model    string
	Size     string
	Specs    []CutoutSpec
	DryRun   bool
}

// ImagesClient requests transparent PNGs from the OpenAI Images API.
type ImagesClient struct {
	HTTPClient *http.Client
	Endpoint   string
	APIKey     string
	Model      string
	Size       string
}

type imagesRequest struct {
	Model          string `json:"model"`
	Prompt         string `json:"prompt"`
	Size           string `json:"size"`
	Background     string `json:"background"`
	OutputFormat   string `json:"output_format"`
	ResponseFormat string `json:"response_format,omitempty"`
}

type imagesResponse struct {
	Data []struct {
		B64JSON string `json:"b64_json"`
		URL     string `json:"url"`
	} `json:"data"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// Generate requests one image and returns a normalized PNG with an alpha channel.
func (c ImagesClient) Generate(ctx context.Context, prompt string) ([]byte, error) {
	if strings.TrimSpace(c.Endpoint) == "" {
		return nil, errors.New("buildtime: images endpoint is required")
	}
	if strings.TrimSpace(c.APIKey) == "" {
		return nil, errors.New("buildtime: images API key is required")
	}
	model := c.Model
	if model == "" {
		model = "gpt-image-2.5-flare"
	}
	size := c.Size
	if size == "" {
		size = "1024x1536"
	}
	body, err := json.Marshal(imagesRequest{Model: model, Prompt: prompt, Size: size, Background: "transparent", OutputFormat: "png"})
	if err != nil {
		return nil, fmt.Errorf("buildtime: encode image request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.Endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("buildtime: create image request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	req.Header.Set("Content-Type", "application/json")
	client := c.HTTPClient
	if client == nil {
		client = &http.Client{}
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("buildtime: images request: %w", err)
	}
	defer resp.Body.Close()
	var decoded imagesResponse
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return nil, fmt.Errorf("buildtime: decode image response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		message := resp.Status
		if decoded.Error != nil && decoded.Error.Message != "" {
			message = decoded.Error.Message
		}
		return nil, fmt.Errorf("buildtime: images API: %s", message)
	}
	if len(decoded.Data) == 0 {
		return nil, errors.New("buildtime: images API returned no image")
	}
	data := decoded.Data[0].B64JSON
	if data == "" {
		return nil, errors.New("buildtime: images response has no b64_json; URL downloads are unsupported")
	}
	raw, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return nil, fmt.Errorf("buildtime: decode image bytes: %w", err)
	}
	return normalizeTransparentPNG(raw)
}

// RunCutoutJob generates and registers all requested transparent cut-outs.
func RunCutoutJob(ctx context.Context, writer *ManifestWriter, options CutoutOptions) error {
	if writer == nil {
		return errors.New("buildtime: nil manifest writer")
	}
	if len(options.Specs) == 0 {
		return errors.New("buildtime: no cut-out specs")
	}
	for _, spec := range options.Specs {
		if err := validateCutoutSpec(spec); err != nil {
			return err
		}
		if options.DryRun {
			fmt.Printf("dry-run cutout %s (%s)\n", spec.LogicalName, effectiveModel(options.Model))
			continue
		}
		client := ImagesClient{HTTPClient: nil, Endpoint: effectiveImagesEndpoint(options.Endpoint), APIKey: options.APIKey, Model: options.Model, Size: options.Size}
		data, err := client.Generate(ctx, spec.Prompt)
		if err != nil {
			return fmt.Errorf("cutout %q: %w", spec.LogicalName, err)
		}
		tmp, err := os.CreateTemp(writer.root, "cutout-*.png")
		if err != nil {
			return fmt.Errorf("cutout %q: create temporary file: %w", spec.LogicalName, err)
		}
		name := tmp.Name()
		if _, err := tmp.Write(data); err != nil {
			tmp.Close()
			os.Remove(name)
			return fmt.Errorf("cutout %q: write temporary file: %w", spec.LogicalName, err)
		}
		if err := tmp.Close(); err != nil {
			os.Remove(name)
			return fmt.Errorf("cutout %q: close temporary file: %w", spec.LogicalName, err)
		}
		_, addErr := writer.AddFile(spec.LogicalName, "IMAGE_CUTOUT", name, spec.Take)
		os.Remove(name)
		if addErr != nil {
			return fmt.Errorf("cutout %q: %w", spec.LogicalName, addErr)
		}
	}
	return nil
}

func normalizeTransparentPNG(raw []byte) ([]byte, error) {
	src, err := png.Decode(bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("buildtime: decode PNG: %w", err)
	}
	bounds := src.Bounds()
	dst := image.NewNRGBA(bounds)
	draw.Draw(dst, bounds, src, bounds.Min, draw.Src)
	transparent := false
	for y := bounds.Min.Y; y < bounds.Max.Y && !transparent; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			if dst.NRGBAAt(x, y).A < 255 {
				transparent = true
				break
			}
		}
	}
	if !transparent {
		return nil, errors.New("buildtime: generated image has no transparent pixels")
	}
	var out bytes.Buffer
	if err := png.Encode(&out, dst); err != nil {
		return nil, fmt.Errorf("buildtime: encode normalized PNG: %w", err)
	}
	return out.Bytes(), nil
}

func validateCutoutSpec(spec CutoutSpec) error {
	if strings.TrimSpace(spec.LogicalName) == "" || filepath.Base(spec.LogicalName) != spec.LogicalName {
		return fmt.Errorf("buildtime: invalid cut-out logical name %q", spec.LogicalName)
	}
	if strings.TrimSpace(spec.Prompt) == "" {
		return fmt.Errorf("buildtime: empty prompt for cut-out %q", spec.LogicalName)
	}
	if spec.Take < 1 {
		return fmt.Errorf("buildtime: cut-out %q has invalid take", spec.LogicalName)
	}
	return nil
}

func effectiveImagesEndpoint(endpoint string) string {
	if endpoint != "" {
		return endpoint
	}
	if value := os.Getenv("DF_OPENAI_IMAGES_URL"); value != "" {
		return value
	}
	return defaultImagesURL
}

func effectiveModel(model string) string {
	if model == "" {
		return "gpt-image-2.5-flare"
	}
	return model
}
