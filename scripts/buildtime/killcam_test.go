package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/monstercameron/DungeonFlux/internal/ports"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

type killcamVendor struct {
	submissions, polls int
	fail               bool
}

func (v *killcamVendor) SubmitReference(context.Context, ports.ReferenceVideoRequest) (ports.VideoJob, error) {
	v.submissions++
	return ports.VideoJob{ID: "test", Vendor: "fal"}, nil
}
func (v *killcamVendor) Poll(context.Context, ports.VideoJob) (ports.VideoStatus, error) {
	v.polls++
	if v.fail {
		return ports.VideoStatus{}, errors.New("interrupted")
	}
	return ports.VideoStatus{State: vocab.JobDone, URL: "fake"}, nil
}
func (v *killcamVendor) Download(context.Context, string) ([]byte, error) {
	return []byte("fake-mp4"), nil
}

func TestKillcam_CacheResumeAndIntegrity(t *testing.T) {
	root := t.TempDir()
	req := ports.ReferenceVideoRequest{Prompt: "hero wins", References: [][]byte{[]byte("hero")}, Seconds: 4, Resolution: "720p", Aspect: "16:9"}
	vendor := &killcamVendor{fail: true}
	budget := 1.0
	if _, err := renderKillcam(t.Context(), root, "killcam_paladin_victory", "paladin", "victory", req, vendor, &budget, 0); err == nil {
		t.Fatal("expected interrupted job")
	}
	vendor.fail = false
	if cached, err := renderKillcam(t.Context(), root, "killcam_paladin_victory", "paladin", "victory", req, vendor, &budget, 0); err != nil || cached || vendor.submissions != 1 {
		t.Fatalf("resume: cached=%v submissions=%d err=%v", cached, vendor.submissions, err)
	}
	if cached, err := renderKillcam(t.Context(), root, "killcam_paladin_victory", "paladin", "victory", req, nil, &budget, 0); err != nil || !cached {
		t.Fatalf("offline cache: %v %v", cached, err)
	}
	w, _ := NewManifestWriter(root)
	take := w.manifest.Assets["killcam_paladin_victory"].Takes[0]
	if err := os.WriteFile(filepath.Join(root, take.Path), []byte("corrupt"), 0600); err != nil {
		t.Fatal(err)
	}
	if cachedKillcam(root, "killcam_paladin_victory", killcamKey(req)) {
		t.Fatal("corrupt cache accepted")
	}
	if _, err := renderKillcam(t.Context(), root, "killcam_paladin_victory", "paladin", "victory", req, nil, &budget, 0); err == nil {
		t.Fatal("cache-only miss accepted")
	}
	req.Prompt = "villain wins"
	if _, err := renderKillcam(t.Context(), root, "killcam_paladin_defeat", "paladin", "defeat", req, vendor, &budget, 0); err == nil || vendor.submissions != 1 {
		t.Fatal("spend cap did not prevent submission")
	}
}

func TestKillcam_RequestUsesReferencesAndDistinctOutcomes(t *testing.T) {
	root := t.TempDir()
	ref := filepath.Join(root, "ref.png")
	if err := os.WriteFile(ref, []byte("reference"), 0600); err != nil {
		t.Fatal(err)
	}
	w, _ := NewManifestWriter(root)
	for _, name := range []string{"enemy", "level"} {
		if _, err := w.AddFile(name, "IMAGE", ref, 1); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := w.Write(); err != nil {
		t.Fatal(err)
	}
	cfg := killcamConfig{Level: "level", Enemy: "enemy"}
	hero := killcamSpec{Class: "paladin", Hero: ref, Weapon: "longsword"}
	victory, err := killcamRequest(root, cfg, hero, "victory")
	if err != nil || len(victory.References) != 3 || victory.Seconds != 4 || !strings.Contains(victory.Prompt, "longsword") {
		t.Fatalf("bad request: %v", err)
	}
	defeat, err := killcamRequest(root, cfg, hero, "defeat")
	if err != nil || killcamKey(victory) == killcamKey(defeat) || !strings.Contains(defeat.Prompt, "villain wins") {
		t.Fatal("outcome not in request/cache key")
	}
	victory.References[0] = []byte("different hero")
	if killcamKey(victory) == killcamKey(defeat) {
		t.Fatal("identity omitted from cache")
	}
	if _, err := killcamRequest(root, cfg, hero, "unknown"); err == nil {
		t.Fatal("invalid outcome accepted")
	}
}
