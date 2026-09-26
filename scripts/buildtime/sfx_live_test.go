//go:build live

package main

import (
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

// TestLiveSFXBuild is the explicitly opt-in paid build-time SFX job.
func TestLiveSFXBuild(t *testing.T) {
	plan, err := PlanSFX(SFXAssets(), 2)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("sfx dry-run: requests=%d estimated_seconds=%.0f estimated_cost_usd=%.4f", plan.Requests, plan.EstimatedSeconds, plan.EstimatedCostUSD)
	if os.Getenv("DF_LIVE") != "1" || os.Getenv("DF_SFX_RUN") != "1" {
		t.Log("dry-run only; set DF_LIVE=1 and DF_SFX_RUN=1 to authorize paid generation")
		return
	}
	if os.Getenv("DF_ELEVENLABS_API_KEY") == "" {
		t.Fatal("DF_ELEVENLABS_API_KEY is required for the live SFX job")
	}
	root := os.Getenv("DF_SFX_ROOT")
	if root == "" {
		root = filepath.Join("artifacts", "runtime", "buildtime")
	}
	endpoint := os.Getenv("DF_ELEVENLABS_ENDPOINT")
	if endpoint == "" {
		endpoint = "https://api.elevenlabs.io/v1"
	}
	takes := 2
	if raw := os.Getenv("DF_SFX_TAKES"); raw != "" {
		parsed, parseErr := strconv.Atoi(raw)
		if parseErr != nil {
			t.Fatal(parseErr)
		}
		takes = parsed
	}
	summary, err := RunSFXBuild(t.Context(), SFXBuildOptions{Client: &http.Client{}, Endpoint: endpoint, Root: root, Takes: takes, Log: os.Stdout})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("sfx live summary: requests=%d seconds=%.2f estimated_cost_usd=%.4f selected=%d", summary.Requests, summary.Seconds, summary.CostUSD, summary.Selected)
}
