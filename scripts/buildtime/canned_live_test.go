//go:build live

package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLiveCannedAndNudgeBuild(t *testing.T) {
	canned := CannedTTSPlan()
	nudges := NudgeTTSPlan()
	t.Logf("dry-run: canned requests=%d characters=%d estimated_cost_usd=%.4f; nudges requests=%d characters=%d estimated_cost_usd=%.4f", canned.Requests, canned.Characters, canned.EstimatedCostUSD, nudges.Requests, nudges.Characters, nudges.EstimatedCostUSD)
	if os.Getenv("DF_LIVE") != "1" || os.Getenv("DF_TTS_RUN") != "1" {
		t.Log("dry-run only; set DF_LIVE=1 and DF_TTS_RUN=1 to authorize paid generation")
		return
	}
	if os.Getenv("DF_ELEVENLABS_API_KEY") == "" {
		t.Fatal("DF_ELEVENLABS_API_KEY is required for live TTS generation")
	}
	root := os.Getenv("DF_TTS_ROOT")
	if root == "" {
		root = filepath.Join("..", "..", "artifacts", "runtime", "buildtime")
	}
	endpoint := os.Getenv("DF_ELEVENLABS_ENDPOINT")
	if endpoint == "" {
		endpoint = "https://api.elevenlabs.io/v1"
	}
	writer, err := NewManifestWriter(root)
	if err != nil {
		t.Fatal(err)
	}
	client := &http.Client{Timeout: 2 * time.Minute}
	ctx := context.Background()
	if err := CannedJob(client, endpoint, root, 1).Run(ctx, writer); err != nil {
		t.Fatal(fmt.Errorf("canned live build: %w", err))
	}
	if err := NudgeJob(client, endpoint, root, 1).Run(ctx, writer); err != nil {
		t.Fatal(fmt.Errorf("nudge live build: %w", err))
	}
	t.Logf("live summary: files=%d characters=%d", len(CannedLines())+len(NudgeLines()), canned.Characters+nudges.Characters)
}
