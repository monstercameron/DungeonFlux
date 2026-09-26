package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// SFXPlan describes the requests and estimated cost before a live run.
type SFXPlan struct {
	Assets           int
	Takes            int
	Requests         int
	EstimatedSeconds float64
	EstimatedCostUSD float64
}

// PlanSFX calculates a dry-run plan without making network calls.
func PlanSFX(assets []SFXAsset, takes int) (SFXPlan, error) {
	if takes < 1 || takes > 3 {
		return SFXPlan{}, errors.New("buildtime: SFX takes must be 1, 2, or 3")
	}
	plan := SFXPlan{Assets: len(assets), Takes: takes}
	for _, asset := range assets {
		duration := asset.DurationSeconds
		if duration == 0 {
			duration = 2
		}
		if _, err := BuildSFXRequest(asset); err != nil {
			return SFXPlan{}, err
		}
		plan.EstimatedSeconds += duration * float64(takes)
	}
	plan.Requests = plan.Assets * takes
	plan.EstimatedCostUSD = plan.EstimatedSeconds / 60 * sfxCostPerMinute
	return plan, nil
}

// SFXBuildOptions configures a live SFX build.
type SFXBuildOptions struct {
	Client     *http.Client
	Endpoint   string
	Root       string
	Takes      int
	Processor  SFXProcessor
	Log        io.Writer
	MaxCostUSD float64
}

// SFXSummary reports the generated takes and measured usage.
type SFXSummary struct {
	Requests int
	Seconds  float64
	CostUSD  float64
	Selected int
}

// RunSFXBuild runs the live job under the exclusive manifest lock.
func RunSFXBuild(ctx context.Context, options SFXBuildOptions) (SFXSummary, error) {
	if options.Client == nil || strings.TrimSpace(options.Endpoint) == "" || strings.TrimSpace(options.Root) == "" {
		return SFXSummary{}, errors.New("buildtime: SFX build requires client, endpoint, and root")
	}
	if options.Takes == 0 {
		options.Takes = 2
	}
	if options.Processor == nil {
		options.Processor = FFmpegSFXProcessor{}
	}
	if options.Log == nil {
		options.Log = io.Discard
	}
	if options.MaxCostUSD == 0 {
		options.MaxCostUSD = 5
	}
	assets := SFXAssets()
	plan, err := PlanSFX(assets, options.Takes)
	if err != nil {
		return SFXSummary{}, err
	}
	if plan.EstimatedCostUSD > options.MaxCostUSD {
		return SFXSummary{}, fmt.Errorf("buildtime: SFX dry-run estimate $%.2f exceeds cap $%.2f", plan.EstimatedCostUSD, options.MaxCostUSD)
	}
	if err := os.MkdirAll(options.Root, 0o755); err != nil {
		return SFXSummary{}, fmt.Errorf("create SFX root: %w", err)
	}
	release, err := acquireSFXManifestLock(ctx, options.Root)
	if err != nil {
		return SFXSummary{}, err
	}
	defer release()
	writer, err := NewManifestWriter(options.Root)
	if err != nil {
		return SFXSummary{}, err
	}
	summary := SFXSummary{}
	for _, asset := range assets {
		bestTake, bestScore := 0, math.MaxFloat64
		for take := 1; take <= options.Takes; take++ {
			stats, renderErr := RenderSFXWithProcessor(ctx, options.Client, options.Endpoint, filepath.Join(options.Root, "sfx"), writer, asset, take, options.Processor)
			if renderErr != nil {
				return summary, renderErr
			}
			summary.Requests++
			summary.Seconds += stats.DurationSeconds
			summary.CostUSD = summary.Seconds / 60 * sfxCostPerMinute
			writeSFXLog(options.Log, asset, take, stats)
			if summary.CostUSD > options.MaxCostUSD {
				return summary, fmt.Errorf("buildtime: SFX cost cap exceeded after %d requests", summary.Requests)
			}
			if score := sfxScore(asset, stats); score < bestScore {
				bestTake, bestScore = take, score
			}
		}
		if err := writer.SelectTake(asset.ID, bestTake); err != nil {
			return summary, err
		}
		assetEntry := writer.manifest.Assets[asset.ID]
		if assetEntry.Metadata == nil {
			assetEntry.Metadata = make(map[string]string)
		}
		assetEntry.Metadata["selected_take"] = strconv.Itoa(bestTake)
		assetEntry.Metadata["takes_generated"] = strconv.Itoa(options.Takes)
		writer.manifest.Assets[asset.ID] = assetEntry
		summary.Selected++
	}
	if _, err := writer.Write(); err != nil {
		return summary, err
	}
	writeSFXSummary(options.Log, plan, summary)
	return summary, nil
}

func acceptableSFXStats(asset SFXAsset, stats SFXMediaStats) bool {
	return stats.DurationSeconds >= 0.2 && stats.DurationSeconds <= 30 && math.Abs(stats.IntegratedLUFS-float64(asset.LUFS)) <= 2.5
}

func sfxScore(asset SFXAsset, stats SFXMediaStats) float64 {
	target := asset.DurationSeconds
	if target == 0 {
		target = 2
	}
	return math.Abs(stats.IntegratedLUFS-float64(asset.LUFS))*10 + math.Abs(stats.DurationSeconds-target)
}

func writeSFXLog(output io.Writer, asset SFXAsset, take int, stats SFXMediaStats) {
	line := strings.Join([]string{"sfx_request", asset.ID, "take=" + strconv.Itoa(take), "duration_s=" + strconv.FormatFloat(stats.DurationSeconds, 'f', 2, 64), "lufs=" + strconv.FormatFloat(stats.IntegratedLUFS, 'f', 1, 64), "estimated_cost_usd=" + strconv.FormatFloat(stats.DurationSeconds/60*sfxCostPerMinute, 'f', 4, 64)}, " ") + "\n"
	_, _ = io.WriteString(output, line)
}

func writeSFXSummary(output io.Writer, plan SFXPlan, summary SFXSummary) {
	line := strings.Join([]string{"sfx_summary", "requests=" + strconv.Itoa(summary.Requests), "planned_requests=" + strconv.Itoa(plan.Requests), "seconds=" + strconv.FormatFloat(summary.Seconds, 'f', 2, 64), "estimated_cost_usd=" + strconv.FormatFloat(summary.CostUSD, 'f', 4, 64), "selected=" + strconv.Itoa(summary.Selected)}, " ") + "\n"
	_, _ = io.WriteString(output, line)
}

func acquireSFXManifestLock(ctx context.Context, root string) (func(), error) {
	path := filepath.Join(root, "manifest.lock")
	deadline := time.NewTimer(60 * time.Second)
	defer deadline.Stop()
	for {
		file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if err == nil {
			_ = file.Close()
			return func() { _ = os.Remove(path) }, nil
		}
		if !errors.Is(err, os.ErrExist) {
			return nil, fmt.Errorf("buildtime: create manifest lock: %w", err)
		}
		timer := time.NewTimer(250 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, ctx.Err()
		case <-deadline.C:
			timer.Stop()
			return nil, errors.New("buildtime: timed out waiting for manifest lock")
		case <-timer.C:
		}
	}
}

// SFXJob returns a job that renders one take of the complete effect library.
func SFXJob(client *http.Client, endpoint, outputDir string, take int) Job {
	return Job{Name: "sound-effects", Run: func(ctx context.Context, writer *ManifestWriter) error {
		for _, asset := range SFXAssets() {
			if err := RenderSFX(ctx, client, endpoint, filepath.Clean(outputDir), writer, asset, take); err != nil {
				return err
			}
		}
		return nil
	}}
}
