package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// TowerStillSpec describes the opaque still used by the generic cliffhanger shot.
type TowerStillSpec struct {
	ID       string
	Source   string
	Prompt   string
	Take     int
	Metadata map[string]string
}

// CliffSpec returns the generic fallback video and its tall-tower source still.
func CliffSpec() ClipSpec {
	return ClipSpec{
		ID: "CLIFF_GENERIC_TOWER", FirstFrameURL: "buildtime://tower-bottom", LastFrameURL: "buildtime://tower-top",
		DurationMS: 5000, Resolution: "720p", Take: 1, Pinned: true,
		Prompt: "Tilt up slowly from a flooded street to a tall old bell tower and its swinging bell. Window lights go out one by one as fog rolls through the rain. Stylized dark fantasy illustration, painterly, muted teal and amber palette. Wide 16:9 composition, no text.",
	}
}

// TowerStill returns the source contract for the generated tall tower image.
func TowerStill(source string) TowerStillSpec {
	return TowerStillSpec{
		ID: "CLIFF_TOWER_STILL", Source: source,
		Prompt: "Tall old bell tower above a flooded riverside street at midnight, viewed from a low angle through rain and rolling fog, swinging bell visible at the top, sparse cold window lights, stylized dark fantasy illustration, painterly, muted teal and amber palette. Wide 16:9 composition, no text.",
		Take:   1, Metadata: map[string]string{"shot": "CLIFF_GENERIC_TOWER", "aspect": "16:9"},
	}
}

// RegisterTowerStill adds a generated tower still and its composition metadata to the manifest.
func RegisterTowerStill(writer *ManifestWriter, spec TowerStillSpec) error {
	if writer == nil {
		return errors.New("buildtime: tower still requires a manifest writer")
	}
	if strings.TrimSpace(spec.ID) == "" || strings.TrimSpace(spec.Source) == "" || spec.Take < 1 {
		return errors.New("buildtime: tower still requires id, source, and take")
	}
	if _, err := os.Stat(spec.Source); err != nil {
		return fmt.Errorf("buildtime: tower still source: %w", err)
	}
	if _, err := writer.AddFile(spec.ID, "IMAGE", spec.Source, spec.Take); err != nil {
		return err
	}
	return writer.SetMetadata(spec.ID, 0, 0, spec.Metadata)
}

// CliffJob returns the generic cliffhanger clip and optional tower-still registration job.
func CliffJob(client VideoClient, outputDir, stillSource string, dryRun bool) Job {
	return Job{Name: "cliffhanger-fallback", Run: func(ctx context.Context, writer *ManifestWriter) error {
		still := TowerStill(stillSource)
		if dryRun {
			fmt.Printf("dry-run still %s (source %s)\n", still.ID, still.Source)
			fmt.Printf("dry-run video %s (%dms, %s)\n", CliffSpec().ID, CliffSpec().DurationMS, CliffSpec().Resolution)
			return nil
		}
		if err := RegisterTowerStill(writer, still); err != nil {
			return err
		}
		if err := client.RenderVideo(ctx, writer, filepath.Clean(outputDir), CliffSpec()); err != nil {
			return err
		}
		return nil
	}}
}
