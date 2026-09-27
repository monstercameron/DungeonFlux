package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/monstercameron/DungeonFlux/internal/media"
	"github.com/monstercameron/DungeonFlux/internal/ports"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
	"github.com/monstercameron/DungeonFlux/internal/wire"
)

type instantVideo struct{}

func (instantVideo) SubmitReference(context.Context, ports.ReferenceVideoRequest) (ports.VideoJob, error) {
	return ports.VideoJob{ID: "j"}, nil
}
func (instantVideo) Poll(context.Context, ports.VideoJob) (ports.VideoStatus, error) {
	return ports.VideoStatus{State: vocab.JobDone, URL: "u"}, nil
}
func (instantVideo) Download(context.Context, string) ([]byte, error) { return []byte("mp4"), nil }

// offline makes envKey find no key anywhere, so nothing can reach a vendor.
func offline(t *testing.T) string {
	t.Helper()
	t.Setenv("DF_FAL_KEY", "")
	t.Setenv("DF_OPENAI_API_KEY", "")
	dir := t.TempDir()
	t.Chdir(dir)
	return dir
}

func writeFile(t *testing.T, path, body string) string {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestParseBillboardFlags(t *testing.T) {
	thrall, err := parseBillboardFlags([]string{"--thrall", "--actions", "idle,attack"})
	if err != nil || thrall.refs != "manifest:thrall_cutout|thrall_still" || thrall.register != "thrall_loop_" || thrall.weapon != thrallWeapon {
		t.Fatalf("thrall=%+v err=%v", thrall, err)
	}
	for _, args := range [][]string{{}, {"--refs", "a.png", "--actions", "dance"}, {"--nope"}} {
		if _, err := parseBillboardFlags(args); err == nil {
			t.Errorf("%v accepted", args)
		}
	}
}

func TestLevelStillAndLoopsRegistration(t *testing.T) {
	dir := offline(t)
	root := filepath.Join(dir, "buildtime")
	still := writeFile(t, filepath.Join(dir, "still.png"), "level pixels")
	if err := runBillboardCommand("level-still", []string{"--root", root, "--scene", "64BB46D5", "--file", still}); err != nil {
		t.Fatal(err)
	}
	if data, err := manifestBytes(root, "level_still_64bb46d5_tactical"); err != nil || string(data) != "level pixels" {
		t.Fatalf("data=%q err=%v", data, err)
	}
	if err := runLevelStill([]string{"--root", root}); err == nil {
		t.Fatal("missing scene accepted")
	}
	front := writeFile(t, filepath.Join(dir, "front.png"), "front")
	dataDir := filepath.Join(dir, "cache")
	spec := media.BillboardSpec{Subject: media.BillboardSubject{References: [][]byte{[]byte("front")}}, LevelStill: []byte("level pixels")}
	seed, err := wire.NewBillboardToolForTest(t.Context(), dataDir, instantVideo{})
	if err != nil {
		t.Fatal(err)
	}
	for _, action := range []string{"idle", "attack"} {
		spec.Action = action
		if _, err := seed.Generate(t.Context(), spec); err != nil {
			t.Fatal(err)
		}
	}
	_ = seed.Close()
	var out bytes.Buffer
	args := []string{"--root", root, "--data-dir", dataDir, "--refs", front, "--actions", "idle,attack", "--register", "hero_loop_"}
	if err := runBillboards(t.Context(), args, &out); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	var attack billboardLine
	if err := json.Unmarshal([]byte(lines[1]), &attack); err != nil || !attack.Cached || attack.Calls != 0 || attack.Action != "attack" {
		t.Fatalf("lines=%v err=%v", lines, err)
	}
	writer, err := NewManifestWriter(root)
	if err != nil {
		t.Fatal(err)
	}
	entry := writer.manifest.Assets["hero_loop_attack"]
	if entry.ContactMS != 1200 || entry.DurationMS != 4000 || entry.Kind != "VIDEO_LOOP" || entry.Metadata["cache_key"] != attack.Key {
		t.Fatalf("entry=%+v", entry)
	}
	if writer.manifest.Assets["hero_loop_idle"].ContactMS != 0 {
		t.Fatal("idle loop has a contact frame")
	}
	// A miss with no key fails without any vendor call.
	if err := runBillboards(t.Context(), []string{"--root", root, "--data-dir", dataDir, "--refs", front, "--actions", "fall"}, &out); err == nil {
		t.Fatal("uncached loop without a key succeeded")
	}
	if err := runBillboards(t.Context(), []string{"--root", root, "--data-dir", dataDir, "--refs", "missing.png"}, &out); err == nil {
		t.Fatal("missing reference accepted")
	}
}

func TestThrallReferenceNeedsKeyWhenMissing(t *testing.T) {
	dir := offline(t)
	root := filepath.Join(dir, "buildtime")
	if err := ensureThrallReference(t.Context(), root); err == nil {
		t.Fatal("missing thrall reference generated without a key")
	}
	writer, _ := NewManifestWriter(root)
	ref := writeFile(t, filepath.Join(dir, "thrall.png"), "thrall")
	if _, err := writer.AddFile(thrallRef, "IMAGE_CUTOUT", ref, 1); err != nil {
		t.Fatal(err)
	}
	if _, err := writer.Write(); err != nil {
		t.Fatal(err)
	}
	if err := ensureThrallReference(t.Context(), root); err != nil {
		t.Fatalf("existing reference: %v", err)
	}
}

func TestEnvKeyAndHelpers(t *testing.T) {
	dir := offline(t)
	writeFile(t, filepath.Join(dir, ".env"), "OTHER=1\nDF_FAL_KEY=\"from-file\"\n")
	if envKey("DF_FAL_KEY") != "from-file" || envKey("MISSING_KEY") != "" {
		t.Fatal("env file lookup")
	}
	t.Setenv("DF_FAL_KEY", "from-env")
	if envKey("DF_FAL_KEY") != "from-env" {
		t.Fatal("environment must win")
	}
	if got := splitList(" a, ,b "); len(got) != 2 || got[1] != "b" || firstNonEmpty("", "x") != "x" || firstNonEmpty() != "" {
		t.Fatal("helpers")
	}
	if loopContactMS("hit") != 1200 || loopContactMS("fall") != 0 {
		t.Fatal("contact")
	}
}
