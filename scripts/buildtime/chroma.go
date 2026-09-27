package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"
)

// ChromaOptions controls the three green-screen latency samples.
type ChromaOptions struct {
	Endpoint string
	APIKey   string
	Model    string
	Samples  int
	DryRun   bool
}

// RunChromaLatencyJob measures and records green-screen video request latency.
func RunChromaLatencyJob(ctx context.Context, writer *ManifestWriter, options ChromaOptions) error {
	if writer == nil {
		return errors.New("buildtime: nil manifest writer")
	}
	if options.Samples == 0 {
		options.Samples = 3
	}
	if options.Samples < 1 {
		return errors.New("buildtime: chroma sample count must be positive")
	}
	for number := 1; number <= options.Samples; number++ {
		logical := "chroma_latency_" + strconv.Itoa(number)
		spec := LoopSpec{LogicalName: logical, Prompt: "full-body fantasy character on a solid #00B140 green screen, static camera", Take: 1, DurationMS: 4000}
		if options.DryRun {
			fmt.Printf("dry-run chroma sample %d (480x854, 4s)\n", number)
			continue
		}
		started := time.Now()
		data, err := (LoopClient{Endpoint: effectiveVideoEndpoint(options.Endpoint), APIKey: options.APIKey, Model: options.Model}).Generate(ctx, spec)
		if err != nil {
			return fmt.Errorf("chroma sample %d: %w", number, err)
		}
		elapsed := time.Since(started).Milliseconds()
		file, err := os.CreateTemp(writer.root, "chroma-*.mp4")
		if err != nil {
			return fmt.Errorf("chroma sample %d: create temporary file: %w", number, err)
		}
		name := file.Name()
		if _, err := file.Write(data); err != nil {
			file.Close()
			os.Remove(name)
			return fmt.Errorf("chroma sample %d: write temporary file: %w", number, err)
		}
		if err := file.Close(); err != nil {
			os.Remove(name)
			return fmt.Errorf("chroma sample %d: close temporary file: %w", number, err)
		}
		if _, err := writer.AddFile(logical, "CHROMA_SAMPLE", name, 1); err != nil {
			os.Remove(name)
			return err
		}
		os.Remove(name)
		if err := writer.SetMetadata(logical, 4000, 0, map[string]string{"elapsed_ms": strconv.FormatInt(elapsed, 10), "resolution": "480x854", "background": "#00B140", "model": effectiveVideoModel(options.Model)}); err != nil {
			return err
		}
	}
	return nil
}
