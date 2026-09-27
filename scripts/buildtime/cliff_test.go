package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestCliffSpec_IsPinnedFiveSecondFallback(t *testing.T) {
	spec := CliffSpec()
	if spec.ID != "CLIFF_GENERIC_TOWER" || spec.DurationMS != 5000 || spec.Resolution != "720p" || !spec.Pinned {
		t.Fatalf("unexpected cliff spec: %#v", spec)
	}
	if _, err := BuildVideoRequest(spec); err != nil {
		t.Fatal(err)
	}
	still := TowerStill("tower.png")
	if still.ID != "CLIFF_TOWER_STILL" || still.Prompt == "" || still.Metadata["aspect"] != "16:9" {
		t.Fatalf("unexpected still spec: %#v", still)
	}
}

func TestRegisterTowerStill_RecordsImageAndMetadata(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "tower.png")
	if err := os.WriteFile(source, []byte("png fixture"), 0o644); err != nil {
		t.Fatal(err)
	}
	writer, err := NewManifestWriter(filepath.Join(root, "buildtime"))
	if err != nil {
		t.Fatal(err)
	}
	if err := RegisterTowerStill(writer, TowerStill(source)); err != nil {
		t.Fatal(err)
	}
	asset := writer.manifest.Assets["CLIFF_TOWER_STILL"]
	if asset.Kind != "IMAGE" || asset.Metadata["shot"] != "CLIFF_GENERIC_TOWER" || len(asset.Takes) != 1 {
		t.Fatalf("unexpected tower manifest asset: %#v", asset)
	}
}

func TestCliffJob_DryRunDoesNotNeedSourceOrClient(t *testing.T) {
	writer, err := NewManifestWriter(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := CliffJob(VideoClient{}, t.TempDir(), "missing.png", true).Run(context.Background(), writer); err != nil {
		t.Fatal(err)
	}
	if len(writer.manifest.Assets) != 0 {
		t.Fatal("dry-run registered cliff assets")
	}
}

func TestRegisterTowerStill_RejectsBadInputs(t *testing.T) {
	writer, err := NewManifestWriter(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	cases := []TowerStillSpec{
		{Source: "x", Take: 1}, {ID: "x", Take: 1}, {ID: "x", Source: "x"}, {ID: "x", Source: "missing", Take: 1},
	}
	for i, spec := range cases {
		if err := RegisterTowerStill(writer, spec); err == nil {
			t.Fatalf("case %d accepted invalid still", i)
		}
	}
}
